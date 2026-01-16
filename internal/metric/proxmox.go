package metric

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bluewave-labs/capture/internal/config"
)

// ProxmoxContainerMetrics represents metrics for a single LXC container.
type ProxmoxContainerMetrics struct {
	VMID        int     `json:"vmid"`
	Name        string  `json:"name"`
	Node        string  `json:"node"`
	Status      string  `json:"status"`
	Type        string  `json:"type"`
	Uptime      int64   `json:"uptime"`
	CPUCores    int     `json:"cpu_cores"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsed  uint64  `json:"memory_used"`
	MemoryTotal uint64  `json:"memory_total"`
	MemoryUsage float64 `json:"memory_usage"`
	SwapUsed    uint64  `json:"swap_used"`
	SwapTotal   uint64  `json:"swap_total"`
	DiskUsed    uint64  `json:"disk_used"`
	DiskTotal   uint64  `json:"disk_total"`
	DiskRead    uint64  `json:"disk_read"`
	DiskWrite   uint64  `json:"disk_write"`
	NetworkIn   uint64  `json:"network_in"`
	NetworkOut  uint64  `json:"network_out"`
}

func (p ProxmoxContainerMetrics) isMetric() {}

// proxmoxClient handles communication with the Proxmox API.
type proxmoxClient struct {
	httpClient  *http.Client
	baseURL     string
	tokenID     string
	tokenSecret string
}

// proxmoxResourceResponse represents the response from /api2/json/cluster/resources.
type proxmoxResourceResponse struct {
	Data []proxmoxResource `json:"data"`
}

// proxmoxResource represents a single resource from the cluster resources endpoint.
type proxmoxResource struct {
	VMID    int     `json:"vmid"`
	Name    string  `json:"name"`
	Node    string  `json:"node"`
	Status  string  `json:"status"`
	Type    string  `json:"type"`
	CPU     float64 `json:"cpu"`
	MaxCPU  int     `json:"maxcpu"`
	Mem     uint64  `json:"mem"`
	MaxMem  uint64  `json:"maxmem"`
	Disk    uint64  `json:"disk"`
	MaxDisk uint64  `json:"maxdisk"`
	NetIn   uint64  `json:"netin"`
	NetOut  uint64  `json:"netout"`
	Uptime  int64   `json:"uptime"`
}

// proxmoxStatusResponse represents the response from /api2/json/nodes/{node}/lxc/{vmid}/status/current.
type proxmoxStatusResponse struct {
	Data proxmoxStatus `json:"data"`
}

// proxmoxStatus represents detailed status of a container.
type proxmoxStatus struct {
	VMID      int     `json:"vmid"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Uptime    int64   `json:"uptime"`
	CPU       float64 `json:"cpu"`
	CPUs      int     `json:"cpus"`
	Mem       uint64  `json:"mem"`
	MaxMem    uint64  `json:"maxmem"`
	Swap      uint64  `json:"swap"`
	MaxSwap   uint64  `json:"maxswap"`
	Disk      uint64  `json:"disk"`
	MaxDisk   uint64  `json:"maxdisk"`
	DiskRead  uint64  `json:"diskread"`
	DiskWrite uint64  `json:"diskwrite"`
	NetIn     uint64  `json:"netin"`
	NetOut    uint64  `json:"netout"`
}

// newProxmoxClient creates a new Proxmox API client.
func newProxmoxClient(cfg config.ProxmoxConfig) *proxmoxClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.SkipTLSVerify, //nolint:gosec // User-configurable option for self-signed certs
		},
	}

	return &proxmoxClient{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
		baseURL:     cfg.Host,
		tokenID:     cfg.TokenID,
		tokenSecret: cfg.TokenSecret,
	}
}

// doRequest performs an authenticated request to the Proxmox API.
func (c *proxmoxClient) doRequest(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set the authentication header using the Proxmox API token format
	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", c.tokenID, c.tokenSecret))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// listLXCContainers retrieves all LXC containers from the cluster.
func (c *proxmoxClient) listLXCContainers(ctx context.Context) ([]proxmoxResource, error) {
	body, err := c.doRequest(ctx, "/api2/json/cluster/resources?type=lxc")
	if err != nil {
		return nil, err
	}

	var response proxmoxResourceResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return response.Data, nil
}

// getContainerStatus retrieves detailed status for a specific container.
func (c *proxmoxClient) getContainerStatus(ctx context.Context, node string, vmid int) (*proxmoxStatus, error) {
	path := fmt.Sprintf("/api2/json/nodes/%s/lxc/%d/status/current", node, vmid)
	body, err := c.doRequest(ctx, path)
	if err != nil {
		return nil, err
	}

	var response proxmoxStatusResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &response.Data, nil
}

// GetProxmoxMetrics retrieves metrics for all LXC containers from a Proxmox server.
// If all is true, it includes stopped containers. Otherwise, only running containers are returned.
// Returns empty data (not an error) if Proxmox is not configured.
func GetProxmoxMetrics(cfg config.ProxmoxConfig, all bool) (MetricsSlice, []CustomErr) {
	metrics := make(MetricsSlice, 0)
	var containerErrors []CustomErr

	// Return empty data if Proxmox is not configured
	if !cfg.IsConfigured() {
		return metrics, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := newProxmoxClient(cfg)

	// List all LXC containers
	containers, err := client.listLXCContainers(ctx)
	if err != nil {
		containerErrors = append(containerErrors, CustomErr{
			Metric: []string{"proxmox.container.list"},
			Error:  err.Error(),
		})
		return nil, containerErrors
	}

	// Filter containers first
	filteredContainers := make([]proxmoxResource, 0, len(containers))
	for _, container := range containers {
		if !all && container.Status != "running" {
			continue
		}
		filteredContainers = append(filteredContainers, container)
	}

	// Process containers and collect detailed metrics
	for _, container := range filteredContainers {
		metric, customErr := processProxmoxContainer(ctx, client, container)
		if customErr.Error != "" {
			containerErrors = append(containerErrors, customErr)
			// Still add the metric with basic data if we got a partial result
			if metric.VMID != 0 {
				metrics = append(metrics, metric)
			}
			continue
		}
		metrics = append(metrics, metric)
	}

	if len(containerErrors) > 0 {
		return metrics, containerErrors
	}

	return metrics, nil
}

// processProxmoxContainer processes a single LXC container and returns its metrics.
func processProxmoxContainer(ctx context.Context, client *proxmoxClient, resource proxmoxResource) (ProxmoxContainerMetrics, CustomErr) {
	// Get detailed status for the container (includes swap and disk I/O)
	status, err := client.getContainerStatus(ctx, resource.Node, resource.VMID)
	if err != nil {
		// Fall back to basic metrics from the resource list if status fails
		// Report the error but still return usable data
		return buildMetricsFromResource(resource), CustomErr{
			Metric: []string{"proxmox.container.status", fmt.Sprintf("vmid:%d", resource.VMID)},
			Error:  fmt.Sprintf("failed to get detailed status, using basic metrics: %v", err),
		}
	}

	// Calculate memory usage percentage
	var memoryUsage float64
	if status.MaxMem > 0 {
		memoryUsage = float64(status.Mem) / float64(status.MaxMem)
	}

	return ProxmoxContainerMetrics{
		VMID:        resource.VMID,
		Name:        resource.Name,
		Node:        resource.Node,
		Status:      resource.Status,
		Type:        resource.Type,
		Uptime:      status.Uptime,
		CPUCores:    status.CPUs,
		CPUUsage:    status.CPU,
		MemoryUsed:  status.Mem,
		MemoryTotal: status.MaxMem,
		MemoryUsage: memoryUsage,
		SwapUsed:    status.Swap,
		SwapTotal:   status.MaxSwap,
		DiskUsed:    status.Disk,
		DiskTotal:   status.MaxDisk,
		DiskRead:    status.DiskRead,
		DiskWrite:   status.DiskWrite,
		NetworkIn:   status.NetIn,
		NetworkOut:  status.NetOut,
	}, CustomErr{}
}

// buildMetricsFromResource creates metrics from the basic resource data
// when detailed status is not available.
func buildMetricsFromResource(resource proxmoxResource) ProxmoxContainerMetrics {
	var memoryUsage float64
	if resource.MaxMem > 0 {
		memoryUsage = float64(resource.Mem) / float64(resource.MaxMem)
	}

	return ProxmoxContainerMetrics{
		VMID:        resource.VMID,
		Name:        resource.Name,
		Node:        resource.Node,
		Status:      resource.Status,
		Type:        resource.Type,
		Uptime:      resource.Uptime,
		CPUCores:    resource.MaxCPU,
		CPUUsage:    resource.CPU,
		MemoryUsed:  resource.Mem,
		MemoryTotal: resource.MaxMem,
		MemoryUsage: memoryUsage,
		SwapUsed:    0,
		SwapTotal:   0,
		DiskUsed:    resource.Disk,
		DiskTotal:   resource.MaxDisk,
		DiskRead:    0,
		DiskWrite:   0,
		NetworkIn:   resource.NetIn,
		NetworkOut:  resource.NetOut,
	}
}
