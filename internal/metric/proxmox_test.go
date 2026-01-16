package metric

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluewave-labs/capture/internal/config"
)

func TestProxmoxContainerMetricsIsMetric(t *testing.T) {
	// Ensure ProxmoxContainerMetrics implements the Metric interface
	var _ Metric = ProxmoxContainerMetrics{}
}

func TestProxmoxConfigIsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		config   config.ProxmoxConfig
		expected bool
	}{
		{
			name:     "empty config",
			config:   config.ProxmoxConfig{},
			expected: false,
		},
		{
			name: "only host set",
			config: config.ProxmoxConfig{
				Host: "https://pve.local:8006",
			},
			expected: false,
		},
		{
			name: "host and token ID set",
			config: config.ProxmoxConfig{
				Host:    "https://pve.local:8006",
				TokenID: "root@pam!capture",
			},
			expected: false,
		},
		{
			name: "all required fields set",
			config: config.ProxmoxConfig{
				Host:        "https://pve.local:8006",
				TokenID:     "root@pam!capture",
				TokenSecret: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			},
			expected: true,
		},
		{
			name: "all fields including optional",
			config: config.ProxmoxConfig{
				Host:          "https://pve.local:8006",
				TokenID:       "root@pam!capture",
				TokenSecret:   "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
				SkipTLSVerify: true,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsConfigured()
			if result != tt.expected {
				t.Errorf("IsConfigured() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetProxmoxMetricsNotConfigured(t *testing.T) {
	cfg := config.ProxmoxConfig{}

	metrics, errs := GetProxmoxMetrics(cfg, false)

	if errs != nil {
		t.Errorf("expected no errors, got %v", errs)
	}

	if len(metrics) != 0 {
		t.Errorf("expected empty metrics, got %v", metrics)
	}
}

func TestBuildMetricsFromResource(t *testing.T) {
	resource := proxmoxResource{
		VMID:    100,
		Name:    "webserver",
		Node:    "pve1",
		Status:  "running",
		Type:    "lxc",
		CPU:     0.15,
		MaxCPU:  2,
		Mem:     536870912,  // 512 MiB
		MaxMem:  2147483648, // 2 GiB
		Disk:    5368709120,
		MaxDisk: 21474836480,
		NetIn:   1073741824,
		NetOut:  268435456,
		Uptime:  86400,
	}

	metrics := buildMetricsFromResource(resource)

	if metrics.VMID != 100 {
		t.Errorf("VMID = %v, expected 100", metrics.VMID)
	}
	if metrics.Name != "webserver" {
		t.Errorf("Name = %v, expected webserver", metrics.Name)
	}
	if metrics.Node != "pve1" {
		t.Errorf("Node = %v, expected pve1", metrics.Node)
	}
	if metrics.Status != "running" {
		t.Errorf("Status = %v, expected running", metrics.Status)
	}
	if metrics.Type != "lxc" {
		t.Errorf("Type = %v, expected lxc", metrics.Type)
	}
	if metrics.CPUCores != 2 {
		t.Errorf("CPUCores = %v, expected 2", metrics.CPUCores)
	}
	if metrics.CPUUsage != 0.15 {
		t.Errorf("CPUUsage = %v, expected 0.15", metrics.CPUUsage)
	}
	if metrics.MemoryUsed != 536870912 {
		t.Errorf("MemoryUsed = %v, expected 536870912", metrics.MemoryUsed)
	}
	if metrics.MemoryTotal != 2147483648 {
		t.Errorf("MemoryTotal = %v, expected 2147483648", metrics.MemoryTotal)
	}
	// Memory usage should be 0.25 (512 MiB / 2 GiB)
	expectedMemoryUsage := 0.25
	if metrics.MemoryUsage < expectedMemoryUsage-0.01 || metrics.MemoryUsage > expectedMemoryUsage+0.01 {
		t.Errorf("MemoryUsage = %v, expected around %v", metrics.MemoryUsage, expectedMemoryUsage)
	}
	// SwapUsed and SwapTotal should be 0 when built from resource
	if metrics.SwapUsed != 0 {
		t.Errorf("SwapUsed = %v, expected 0", metrics.SwapUsed)
	}
	if metrics.SwapTotal != 0 {
		t.Errorf("SwapTotal = %v, expected 0", metrics.SwapTotal)
	}
	if metrics.DiskUsed != 5368709120 {
		t.Errorf("DiskUsed = %v, expected 5368709120", metrics.DiskUsed)
	}
	if metrics.DiskTotal != 21474836480 {
		t.Errorf("DiskTotal = %v, expected 21474836480", metrics.DiskTotal)
	}
	// DiskRead and DiskWrite should be 0 when built from resource
	if metrics.DiskRead != 0 {
		t.Errorf("DiskRead = %v, expected 0", metrics.DiskRead)
	}
	if metrics.DiskWrite != 0 {
		t.Errorf("DiskWrite = %v, expected 0", metrics.DiskWrite)
	}
	if metrics.NetworkIn != 1073741824 {
		t.Errorf("NetworkIn = %v, expected 1073741824", metrics.NetworkIn)
	}
	if metrics.NetworkOut != 268435456 {
		t.Errorf("NetworkOut = %v, expected 268435456", metrics.NetworkOut)
	}
	if metrics.Uptime != 86400 {
		t.Errorf("Uptime = %v, expected 86400", metrics.Uptime)
	}
}

func TestBuildMetricsFromResourceZeroMemory(t *testing.T) {
	resource := proxmoxResource{
		VMID:   100,
		Name:   "test",
		MaxMem: 0, // Zero memory (edge case)
	}

	metrics := buildMetricsFromResource(resource)

	// Memory usage should be 0 when MaxMem is 0 (prevent division by zero)
	if metrics.MemoryUsage != 0 {
		t.Errorf("MemoryUsage = %v, expected 0", metrics.MemoryUsage)
	}
}

func TestProxmoxClientListContainers(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authentication header
		auth := r.Header.Get("Authorization")
		expectedAuth := "PVEAPIToken=root@pam!capture=test-secret"
		if auth != expectedAuth {
			t.Errorf("Authorization header = %v, expected %v", auth, expectedAuth)
		}

		// Verify request path
		if r.URL.Path != "/api2/json/cluster/resources" {
			t.Errorf("Request path = %v, expected /api2/json/cluster/resources", r.URL.Path)
		}

		// Verify query parameter
		if r.URL.Query().Get("type") != "lxc" {
			t.Errorf("Query param type = %v, expected lxc", r.URL.Query().Get("type"))
		}

		// Return mock response
		response := proxmoxResourceResponse{
			Data: []proxmoxResource{
				{
					VMID:    100,
					Name:    "webserver",
					Node:    "pve1",
					Status:  "running",
					Type:    "lxc",
					CPU:     0.15,
					MaxCPU:  2,
					Mem:     536870912,
					MaxMem:  2147483648,
					Disk:    5368709120,
					MaxDisk: 21474836480,
					NetIn:   1073741824,
					NetOut:  268435456,
					Uptime:  86400,
				},
				{
					VMID:   101,
					Name:   "database",
					Node:   "pve1",
					Status: "stopped",
					Type:   "lxc",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := newProxmoxClient(config.ProxmoxConfig{
		Host:        server.URL,
		TokenID:     "root@pam!capture",
		TokenSecret: "test-secret",
	})

	containers, err := client.listLXCContainers(context.Background())
	if err != nil {
		t.Fatalf("listLXCContainers() error = %v", err)
	}

	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}

	if containers[0].VMID != 100 {
		t.Errorf("First container VMID = %v, expected 100", containers[0].VMID)
	}
	if containers[0].Name != "webserver" {
		t.Errorf("First container Name = %v, expected webserver", containers[0].Name)
	}
	if containers[1].VMID != 101 {
		t.Errorf("Second container VMID = %v, expected 101", containers[1].VMID)
	}
}

func TestProxmoxClientGetContainerStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes/pve1/lxc/100/status/current" {
			t.Errorf("Request path = %v, expected /api2/json/nodes/pve1/lxc/100/status/current", r.URL.Path)
		}

		response := proxmoxStatusResponse{
			Data: proxmoxStatus{
				VMID:      100,
				Name:      "webserver",
				Status:    "running",
				Uptime:    86400,
				CPU:       0.15,
				CPUs:      2,
				Mem:       536870912,
				MaxMem:    2147483648,
				Swap:      0,
				MaxSwap:   536870912,
				Disk:      5368709120,
				MaxDisk:   21474836480,
				DiskRead:  1073741824,
				DiskWrite: 536870912,
				NetIn:     1073741824,
				NetOut:    268435456,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := newProxmoxClient(config.ProxmoxConfig{
		Host:        server.URL,
		TokenID:     "root@pam!capture",
		TokenSecret: "test-secret",
	})

	status, err := client.getContainerStatus(context.Background(), "pve1", 100)
	if err != nil {
		t.Fatalf("getContainerStatus() error = %v", err)
	}

	if status.VMID != 100 {
		t.Errorf("VMID = %v, expected 100", status.VMID)
	}
	if status.CPUs != 2 {
		t.Errorf("CPUs = %v, expected 2", status.CPUs)
	}
	if status.Swap != 0 {
		t.Errorf("Swap = %v, expected 0", status.Swap)
	}
	if status.MaxSwap != 536870912 {
		t.Errorf("MaxSwap = %v, expected 536870912", status.MaxSwap)
	}
	if status.DiskRead != 1073741824 {
		t.Errorf("DiskRead = %v, expected 1073741824", status.DiskRead)
	}
	if status.DiskWrite != 536870912 {
		t.Errorf("DiskWrite = %v, expected 536870912", status.DiskWrite)
	}
}

func TestProxmoxClientErrorHandling(t *testing.T) {
	t.Run("HTTP error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":{"username":"invalid token"}}`))
		}))
		defer server.Close()

		client := newProxmoxClient(config.ProxmoxConfig{
			Host:        server.URL,
			TokenID:     "invalid",
			TokenSecret: "invalid",
		})

		_, err := client.listLXCContainers(context.Background())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`invalid json`))
		}))
		defer server.Close()

		client := newProxmoxClient(config.ProxmoxConfig{
			Host:        server.URL,
			TokenID:     "root@pam!capture",
			TokenSecret: "test-secret",
		})

		_, err := client.listLXCContainers(context.Background())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("connection error", func(t *testing.T) {
		client := newProxmoxClient(config.ProxmoxConfig{
			Host:        "http://invalid-host-that-does-not-exist:8006",
			TokenID:     "root@pam!capture",
			TokenSecret: "test-secret",
		})

		_, err := client.listLXCContainers(context.Background())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestGetProxmoxMetricsIntegration(t *testing.T) {
	// Create mock server that handles both endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api2/json/cluster/resources":
			response := proxmoxResourceResponse{
				Data: []proxmoxResource{
					{
						VMID:    100,
						Name:    "webserver",
						Node:    "pve1",
						Status:  "running",
						Type:    "lxc",
						CPU:     0.15,
						MaxCPU:  2,
						Mem:     536870912,
						MaxMem:  2147483648,
						Disk:    5368709120,
						MaxDisk: 21474836480,
						NetIn:   1073741824,
						NetOut:  268435456,
						Uptime:  86400,
					},
					{
						VMID:   101,
						Name:   "database",
						Node:   "pve1",
						Status: "stopped",
						Type:   "lxc",
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		case "/api2/json/nodes/pve1/lxc/100/status/current":
			response := proxmoxStatusResponse{
				Data: proxmoxStatus{
					VMID:      100,
					Name:      "webserver",
					Status:    "running",
					Uptime:    86400,
					CPU:       0.15,
					CPUs:      2,
					Mem:       536870912,
					MaxMem:    2147483648,
					Swap:      0,
					MaxSwap:   536870912,
					Disk:      5368709120,
					MaxDisk:   21474836480,
					DiskRead:  1073741824,
					DiskWrite: 536870912,
					NetIn:     1073741824,
					NetOut:    268435456,
				},
			}
			json.NewEncoder(w).Encode(response)

		case "/api2/json/nodes/pve1/lxc/101/status/current":
			response := proxmoxStatusResponse{
				Data: proxmoxStatus{
					VMID:    101,
					Name:    "database",
					Status:  "stopped",
					CPUs:    1,
					MaxMem:  1073741824,
					MaxSwap: 268435456,
					MaxDisk: 10737418240,
				},
			}
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := config.ProxmoxConfig{
		Host:        server.URL,
		TokenID:     "root@pam!capture",
		TokenSecret: "test-secret",
	}

	t.Run("running containers only", func(t *testing.T) {
		metrics, errs := GetProxmoxMetrics(cfg, false)

		if errs != nil {
			t.Errorf("unexpected errors: %v", errs)
		}

		if len(metrics) != 1 {
			t.Fatalf("expected 1 container, got %d", len(metrics))
		}

		m := metrics[0].(ProxmoxContainerMetrics)
		if m.VMID != 100 {
			t.Errorf("VMID = %v, expected 100", m.VMID)
		}
		if m.Name != "webserver" {
			t.Errorf("Name = %v, expected webserver", m.Name)
		}
		if m.Status != "running" {
			t.Errorf("Status = %v, expected running", m.Status)
		}
		if m.SwapTotal != 536870912 {
			t.Errorf("SwapTotal = %v, expected 536870912", m.SwapTotal)
		}
		if m.DiskRead != 1073741824 {
			t.Errorf("DiskRead = %v, expected 1073741824", m.DiskRead)
		}
	})

	t.Run("all containers", func(t *testing.T) {
		metrics, errs := GetProxmoxMetrics(cfg, true)

		if errs != nil {
			t.Errorf("unexpected errors: %v", errs)
		}

		if len(metrics) != 2 {
			t.Fatalf("expected 2 containers, got %d", len(metrics))
		}

		// Verify both containers are present
		vmids := make(map[int]bool)
		for _, m := range metrics {
			pm := m.(ProxmoxContainerMetrics)
			vmids[pm.VMID] = true
		}

		if !vmids[100] {
			t.Error("expected VMID 100 to be present")
		}
		if !vmids[101] {
			t.Error("expected VMID 101 to be present")
		}
	})
}

func TestGetProxmoxMetricsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errors":{"message":"internal error"}}`))
	}))
	defer server.Close()

	cfg := config.ProxmoxConfig{
		Host:        server.URL,
		TokenID:     "root@pam!capture",
		TokenSecret: "test-secret",
	}

	metrics, errs := GetProxmoxMetrics(cfg, false)

	if metrics != nil {
		t.Errorf("expected nil metrics, got %v", metrics)
	}

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	if errs[0].Metric[0] != "proxmox.container.list" {
		t.Errorf("error metric = %v, expected proxmox.container.list", errs[0].Metric[0])
	}
}

func TestProcessProxmoxContainerStatusError(t *testing.T) {
	// Test that when status endpoint fails, we get basic metrics with an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api2/json/cluster/resources":
			response := proxmoxResourceResponse{
				Data: []proxmoxResource{
					{
						VMID:    100,
						Name:    "webserver",
						Node:    "pve1",
						Status:  "running",
						Type:    "lxc",
						CPU:     0.15,
						MaxCPU:  2,
						Mem:     536870912,
						MaxMem:  2147483648,
						Disk:    5368709120,
						MaxDisk: 21474836480,
						NetIn:   1073741824,
						NetOut:  268435456,
						Uptime:  86400,
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		default:
			// Status endpoint returns error
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	cfg := config.ProxmoxConfig{
		Host:        server.URL,
		TokenID:     "root@pam!capture",
		TokenSecret: "test-secret",
	}

	metrics, errs := GetProxmoxMetrics(cfg, false)

	// Should still get metrics (basic data)
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}

	// Should have an error reported
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	// Verify the error mentions status
	if errs[0].Metric[0] != "proxmox.container.status" {
		t.Errorf("error metric = %v, expected proxmox.container.status", errs[0].Metric[0])
	}

	// Verify basic metrics are present
	m := metrics[0].(ProxmoxContainerMetrics)
	if m.VMID != 100 {
		t.Errorf("VMID = %v, expected 100", m.VMID)
	}
	if m.Name != "webserver" {
		t.Errorf("Name = %v, expected webserver", m.Name)
	}
	// Swap should be 0 since we fell back to basic metrics
	if m.SwapTotal != 0 {
		t.Errorf("SwapTotal = %v, expected 0 (basic metrics)", m.SwapTotal)
	}
}

func TestGetProxmoxMetricsContextCancellation(t *testing.T) {
	// Test that context cancellation is handled
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay response to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.ProxmoxConfig{
		Host:        server.URL,
		TokenID:     "root@pam!capture",
		TokenSecret: "test-secret",
	}

	// Create a client with very short timeout to trigger cancellation
	client := &proxmoxClient{
		httpClient: &http.Client{
			Timeout: 1 * time.Millisecond,
		},
		baseURL:     cfg.Host,
		tokenID:     cfg.TokenID,
		tokenSecret: cfg.TokenSecret,
	}

	ctx := context.Background()
	_, err := client.listLXCContainers(ctx)

	if err == nil {
		t.Error("expected error due to timeout, got nil")
	}
}

func TestProxmoxClientDoRequestReadsBodyOnce(t *testing.T) {
	// Verify body is read once regardless of status code
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "test error message"}`))
	}))
	defer server.Close()

	client := newProxmoxClient(config.ProxmoxConfig{
		Host:        server.URL,
		TokenID:     "root@pam!capture",
		TokenSecret: "test-secret",
	})

	_, err := client.doRequest(context.Background(), "/test")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// The error should contain the response body
	if !strings.Contains(err.Error(), "test error message") {
		t.Errorf("error should contain response body, got: %v", err)
	}
}

func TestGetProxmoxMetricsEmptyContainerList(t *testing.T) {
	// Test handling of empty container list
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := proxmoxResourceResponse{
			Data: []proxmoxResource{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := config.ProxmoxConfig{
		Host:        server.URL,
		TokenID:     "root@pam!capture",
		TokenSecret: "test-secret",
	}

	metrics, errs := GetProxmoxMetrics(cfg, false)

	if errs != nil {
		t.Errorf("unexpected errors: %v", errs)
	}

	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics, got %d", len(metrics))
	}
}
