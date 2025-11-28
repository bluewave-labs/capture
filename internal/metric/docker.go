package metric

import (
	"context"
	"encoding/json"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type ContainerMetrics struct {
	ContainerID   string                 `json:"container_id"`
	ContainerName string                 `json:"container_name"`
	Status        string                 `json:"status"` // "created", "running", "paused", "restarting", "removing", "exited", "dead"
	Health        *ContainerHealthStatus `json:"health"`
	Running       bool                   `json:"running"`
	BaseImage     string                 `json:"base_image"`
	ExposedPorts  []Port                 `json:"exposed_ports"`
	StartedAt     int64                  `json:"started_at"`  // Unix timestamp
	FinishedAt    int64                  `json:"finished_at"` // Unix timestamp
	Stats         *ContainerStats        `json:"stats"`
}

type ContainerStats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   uint64  `json:"memory_usage"`
	MemoryLimit   uint64  `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
	NetworkRx     uint64  `json:"network_rx_bytes"`
	NetworkTx     uint64  `json:"network_tx_bytes"`
	BlockRead     uint64  `json:"block_read_bytes"`
	BlockWrite    uint64  `json:"block_write_bytes"`
	PIDs          uint64  `json:"pids"`
}

func (c ContainerMetrics) isMetric() {}

type Port struct {
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
}

type ContainerHealthStatus struct {
	Healthy bool                  `json:"healthy"`
	Source  ContainerHealthSource `json:"source"`  // "container_health_check", "state_based_health_check"
	Message string                `json:"message"` // Additional message if needed
}

type ContainerHealthSource string

const (
	SourceContainerHealthCheck  ContainerHealthSource = "container_health_check"
	SourceStateBasedHealthCheck ContainerHealthSource = "state_based_health_check"
)

func GetDockerMetrics(all bool) (MetricsSlice, []CustomErr) {
	var metrics = make(MetricsSlice, 0)
	var containerErrors []CustomErr

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli, err := initializeDockerClient()
	if err != nil {
		containerErrors = append(containerErrors, CustomErr{
			Metric: []string{"docker.client"},
			Error:  err.Error(),
		})
		return nil, containerErrors
	}
	defer cli.Close()

	containers, err := listContainers(ctx, cli, all)
	if err != nil {
		return nil, append(containerErrors, CustomErr{
			Metric: []string{"docker.container.list"},
			Error:  err.Error(),
		})
	}

	for _, container := range containers {
		metric, customErr := processContainer(ctx, cli, container)
		if customErr.Error != "" {
			containerErrors = append(containerErrors, customErr)
			continue
		}
		metrics = append(metrics, metric)
	}

	if len(containerErrors) > 0 {
		return metrics, containerErrors
	}

	return metrics, nil
}

// initializeDockerClient creates a new Docker client with environment configuration.
func initializeDockerClient() (*client.Client, error) {
	// Initialize the Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return cli, nil
}

// listContainers retrieves the list of containers from Docker.
func listContainers(ctx context.Context, cli *client.Client, all bool) ([]container.Summary, error) {
	// List all containers
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All: all,
	})
	if err != nil {
		return nil, err
	}
	return containers, nil
}

// processContainer processes a single container and returns its metrics.
func processContainer(ctx context.Context, cli *client.Client, container container.Summary) (ContainerMetrics, CustomErr) {
	containerInspectResponse, err := inspectContainer(ctx, cli, container.ID)
	if err != nil {
		return ContainerMetrics{}, CustomErr{
			Metric: []string{"docker.container.inspect"},
			Error:  err.Error(),
		}
	}

	portList := extractExposedPorts(containerInspectResponse)

	containerStats, customErr := getContainerStats(ctx, cli, container.ID)
	if customErr.Error != "" {
		return ContainerMetrics{}, customErr
	}

	cpuPercent := calculateCPUPercent(containerStats)
	memUsage, memLimit, memPercent := calculateMemoryMetrics(containerStats)
	rx, tx := calculateNetworkMetrics(containerStats)
	blockRead, blockWrite := calculateBlockIOMetrics(containerStats)
	pids := containerStats.PidsStats.Current

	return ContainerMetrics{
		ContainerID:   container.ID,
		ContainerName: getContainerName(container.Names),
		Status:        containerInspectResponse.State.Status, // Can be one of "created", "running", "paused", "restarting", "removing", "exited", or "dead"
		Running:       containerInspectResponse.State.Running,
		BaseImage:     container.Image,
		ExposedPorts:  portList,
		StartedAt:     GetUnixTimestamp(containerInspectResponse.State.StartedAt),
		FinishedAt:    GetUnixTimestamp(containerInspectResponse.State.FinishedAt),
		Health:        healthCheck(containerInspectResponse),
		Stats: &ContainerStats{
			CPUPercent:    cpuPercent,
			MemoryUsage:   memUsage,
			MemoryLimit:   memLimit,
			MemoryPercent: memPercent,
			NetworkRx:     rx,
			NetworkTx:     tx,
			BlockRead:     blockRead,
			BlockWrite:    blockWrite,
			PIDs:          pids,
		},
	}, CustomErr{}
}

// inspectContainer inspects a container and returns its detailed information.
func inspectContainer(ctx context.Context, cli *client.Client, containerID string) (container.InspectResponse, error) {
	// Inspect each container
	containerInspectResponse, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return container.InspectResponse{}, err
	}
	return containerInspectResponse, nil
}

// extractExposedPorts extracts the exposed ports from a container inspection response.
func extractExposedPorts(containerInspectResponse container.InspectResponse) []Port {
	portList := make([]Port, 0)
	if containerInspectResponse.Config == nil {
		return portList
	}
	for port := range containerInspectResponse.Config.ExposedPorts {
		portList = append(portList, Port{
			Port:     port.Port(),
			Protocol: port.Proto(),
		})
	}
	return portList
}

// dockerStatsResponse represents the raw statistics response from Docker API.
type dockerStatsResponse struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage  uint64   `json:"total_usage"`
			PercpuUsage []uint64 `json:"percpu_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs  uint32 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
		Stats struct {
			InactiveFile uint64 `json:"inactive_file"`
		} `json:"stats"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	BlkioStats struct {
		IoServiceBytesRecursive []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
	} `json:"blkio_stats"`
	PidsStats struct {
		Current uint64 `json:"current"`
	} `json:"pids_stats"`
}

// getContainerStats retrieves and decodes container statistics.
func getContainerStats(ctx context.Context, cli *client.Client, containerID string) (dockerStatsResponse, CustomErr) {
	// Get container stats
	stats, err := cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return dockerStatsResponse{}, CustomErr{
			Metric: []string{"docker.container.stats"},
			Error:  err.Error(),
		}
	}

	defer stats.Body.Close()

	var s dockerStatsResponse
	dec := json.NewDecoder(stats.Body)
	if err := dec.Decode(&s); err != nil {
		return dockerStatsResponse{}, CustomErr{
			Metric: []string{"docker.container.stats.decode"},
			Error:  err.Error(),
		}
	}

	return s, CustomErr{}
}

// calculateCPUPercent calculates the CPU usage percentage from Docker stats.
func calculateCPUPercent(s dockerStatsResponse) float64 {
	// Calculate CPU percent (use Docker's common calculation)
	cpuPercent := 0.0
	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(s.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(s.CPUStats.SystemUsage) - float64(s.PreCPUStats.SystemUsage)
	onlineCPUs := float64(s.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 && len(s.CPUStats.CPUUsage.PercpuUsage) > 0 {
		onlineCPUs = float64(len(s.CPUStats.CPUUsage.PercpuUsage))
	}
	if systemDelta > 0.0 && cpuDelta > 0.0 && onlineCPUs > 0 {
		cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}
	return cpuPercent
}

// calculateMemoryMetrics calculates memory usage, limit, and percentage from Docker stats.
// Uses the same calculation as Docker CLI: usage - inactive_file (cache that can be reclaimed)
func calculateMemoryMetrics(s dockerStatsResponse) (uint64, uint64, float64) {
	// Memory usage and percent
	// Docker CLI subtracts inactive_file (page cache) from usage to get "real" memory usage
	memUsage := s.MemoryStats.Usage
	if s.MemoryStats.Stats.InactiveFile > 0 && s.MemoryStats.Stats.InactiveFile < memUsage {
		memUsage -= s.MemoryStats.Stats.InactiveFile
	}
	memLimit := s.MemoryStats.Limit
	memPercent := 0.0
	if memLimit > 0 {
		memPercent = float64(memUsage) / float64(memLimit) * 100.0
	}
	return memUsage, memLimit, memPercent
}

// calculateNetworkMetrics calculates total network bytes received and transmitted.
func calculateNetworkMetrics(s dockerStatsResponse) (uint64, uint64) {
	// Network bytes (sum across interfaces)
	var rx uint64
	var tx uint64
	if s.Networks != nil {
		for _, v := range s.Networks {
			rx += v.RxBytes
			tx += v.TxBytes
		}
	}
	return rx, tx
}

// calculateBlockIOMetrics calculates total block I/O read and write bytes.
func calculateBlockIOMetrics(s dockerStatsResponse) (uint64, uint64) {
	// Block I/O bytes
	var blockRead uint64
	var blockWrite uint64
	for _, stat := range s.BlkioStats.IoServiceBytesRecursive {
		// Docker uses lowercase operation names: "read" and "write"
		switch stat.Op {
		case "read", "Read":
			blockRead += stat.Value
		case "write", "Write":
			blockWrite += stat.Value
		}
	}
	return blockRead, blockWrite
}

func stateBasedHealthCheck(inspectResponse container.InspectResponse) bool {
	if inspectResponse.State == nil {
		// If the state is nil, we cannot determine health
		return false
	}
	// Check for explicit failure conditions first
	if inspectResponse.State.OOMKilled || inspectResponse.State.Dead || inspectResponse.State.ExitCode != 0 {
		return false
	}

	// Only consider healthy if running and status is "running"
	return inspectResponse.State != nil && inspectResponse.State.Running && inspectResponse.State.Status == "running"
}

// healthCheck returns the health status of a container based on its inspection response.
// If there is health check information available, it returns the health status.
// If not, it runs a state based health check based on the container's state.
// If the container is running and healthy, it returns a healthy status.
// If the container is not running or has failed, it returns an unhealthy status.
// If the container is starting, it returns 'healthy' status.
func healthCheck(inspectResponse container.InspectResponse) *ContainerHealthStatus {
	if inspectResponse.State.Health != nil {
		// If the container has a health check, return its status
		return &ContainerHealthStatus{
			// If the health check is healthy or starting, consider it healthy
			Healthy: inspectResponse.State.Health.Status == "healthy" || inspectResponse.State.Health.Status == "starting",
			Source:  SourceContainerHealthCheck,
			Message: "Based on container health check",
		}
	}

	// If no health check is defined, run state based health check
	return &ContainerHealthStatus{
		Healthy: stateBasedHealthCheck(inspectResponse),
		Source:  SourceStateBasedHealthCheck,
		Message: "Based on container state",
	}
}

// getContainerName extracts the container name from the list of names.
func getContainerName(names []string) string {
	if len(names) == 0 || len(names[0]) == 0 {
		// If there are no names or the first name is empty, return an empty string
		return ""
	}

	if names[0][0] == '/' {
		return names[0][1:] // Remove the leading '/' from the container name
	}

	return names[0]
}

// GetUnixTimestamp converts a timestamp string in RFC3339 format to a Unix timestamp.
// If the timestamp is invalid or represents the zero value, it returns 0.
// The function handles both seconds and nanoseconds precision.
func GetUnixTimestamp(timestamp string) int64 {
	// Convert the timestamp string to a Unix timestamp
	t, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil || timestamp == "0001-01-01T00:00:00Z" {
		return 0 // Return 0 if parsing fails or if the timestamp is the zero value
	}
	return t.Unix()
}
