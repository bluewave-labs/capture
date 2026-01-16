// Mock Proxmox API server for testing the Capture Proxmox integration.
// This simulates the Proxmox VE REST API with realistic LXC container data.
//
// Usage:
//   go run ./cmd/mock-proxmox/
//
// Then run Capture with:
//   API_SECRET=test \
//   PROXMOX_HOST=http://localhost:8006 \
//   PROXMOX_TOKEN_ID=root@pam!capture \
//   PROXMOX_TOKEN_SECRET=test-secret-uuid \
//   go run ./cmd/capture/
//
// Test the endpoint:
//   curl -H "Authorization: Bearer test" http://localhost:59232/api/v1/metrics/proxmox
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	expectedTokenID     = "root@pam!capture"
	expectedTokenSecret = "test-secret-uuid"
)

// Mock data representing LXC containers
var mockContainers = []lxcResource{
	{
		VMID:    100,
		Name:    "webserver",
		Node:    "pve1",
		Status:  "running",
		Type:    "lxc",
		CPU:     0.15,
		MaxCPU:  2,
		Mem:     536870912,  // 512 MiB
		MaxMem:  2147483648, // 2 GiB
		Disk:    5368709120, // 5 GiB
		MaxDisk: 21474836480,
		NetIn:   1073741824,
		NetOut:  268435456,
		Uptime:  86400,
	},
	{
		VMID:    101,
		Name:    "database",
		Node:    "pve1",
		Status:  "running",
		Type:    "lxc",
		CPU:     0.45,
		MaxCPU:  4,
		Mem:     3221225472,  // 3 GiB
		MaxMem:  8589934592,  // 8 GiB
		Disk:    32212254720, // 30 GiB
		MaxDisk: 107374182400,
		NetIn:   2147483648,
		NetOut:  1073741824,
		Uptime:  172800,
	},
	{
		VMID:    102,
		Name:    "cache-server",
		Node:    "pve1",
		Status:  "running",
		Type:    "lxc",
		CPU:     0.08,
		MaxCPU:  1,
		Mem:     268435456,  // 256 MiB
		MaxMem:  1073741824, // 1 GiB
		Disk:    1073741824, // 1 GiB
		MaxDisk: 10737418240,
		NetIn:   536870912,
		NetOut:  134217728,
		Uptime:  259200,
	},
	{
		VMID:    103,
		Name:    "backup-container",
		Node:    "pve1",
		Status:  "stopped",
		Type:    "lxc",
		CPU:     0,
		MaxCPU:  2,
		Mem:     0,
		MaxMem:  4294967296,
		Disk:    0,
		MaxDisk: 53687091200,
		NetIn:   0,
		NetOut:  0,
		Uptime:  0,
	},
}

type lxcResource struct {
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

type lxcStatus struct {
	VMID      int     `json:"vmid"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Type      string  `json:"type"`
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

type resourceResponse struct {
	Data []lxcResource `json:"data"`
}

type statusResponse struct {
	Data lxcStatus `json:"data"`
}

func main() {
	mux := http.NewServeMux()

	// Cluster resources endpoint
	mux.HandleFunc("/api2/json/cluster/resources", handleClusterResources)

	// Container status endpoint (matches pattern /api2/json/nodes/{node}/lxc/{vmid}/status/current)
	mux.HandleFunc("/api2/json/nodes/", handleNodeRequests)

	server := &http.Server{
		Addr:              ":8006",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("Mock Proxmox API server starting on :8006")
	log.Println("")
	log.Println("Expected authentication:")
	log.Printf("  Token ID:     %s", expectedTokenID)
	log.Printf("  Token Secret: %s", expectedTokenSecret)
	log.Println("")
	log.Println("Test with:")
	log.Println("  API_SECRET=test \\")
	log.Println("  PROXMOX_HOST=http://localhost:8006 \\")
	log.Println("  PROXMOX_TOKEN_ID=root@pam!capture \\")
	log.Println("  PROXMOX_TOKEN_SECRET=test-secret-uuid \\")
	log.Println("  go run ./cmd/capture/")
	log.Println("")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func validateAuth(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	expected := fmt.Sprintf("PVEAPIToken=%s=%s", expectedTokenID, expectedTokenSecret)
	return auth == expected
}

func handleClusterResources(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] %s %s", r.Method, r.URL.Path, r.URL.RawQuery)

	if !validateAuth(r) {
		log.Println("  -> 401 Unauthorized (invalid token)")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": map[string]string{"username": "invalid token"},
		})
		return
	}

	// Check for type=lxc filter
	resourceType := r.URL.Query().Get("type")
	if resourceType != "" && resourceType != "lxc" {
		log.Printf("  -> 200 OK (empty, filtered by type=%s)", resourceType)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resourceResponse{Data: []lxcResource{}})
		return
	}

	// Add some randomness to simulate real metrics
	containers := make([]lxcResource, len(mockContainers))
	copy(containers, mockContainers)
	for i := range containers {
		if containers[i].Status == "running" {
			// Slightly vary CPU usage
			containers[i].CPU = containers[i].CPU * (0.8 + rand.Float64()*0.4)
			// Slightly vary memory
			variation := uint64(float64(containers[i].Mem) * (0.95 + rand.Float64()*0.1))
			if variation < containers[i].MaxMem {
				containers[i].Mem = variation
			}
			// Increment network counters
			containers[i].NetIn += uint64(rand.Intn(10000))
			containers[i].NetOut += uint64(rand.Intn(5000))
			// Increment uptime
			containers[i].Uptime++
		}
	}

	log.Printf("  -> 200 OK (returning %d containers)", len(containers))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resourceResponse{Data: containers})
}

func handleNodeRequests(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] %s", r.Method, r.URL.Path)

	if !validateAuth(r) {
		log.Println("  -> 401 Unauthorized (invalid token)")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": map[string]string{"username": "invalid token"},
		})
		return
	}

	// Parse path: /api2/json/nodes/{node}/lxc/{vmid}/status/current
	path := strings.TrimPrefix(r.URL.Path, "/api2/json/nodes/")
	parts := strings.Split(path, "/")

	if len(parts) < 4 || parts[1] != "lxc" || parts[3] != "status" {
		log.Println("  -> 404 Not Found (invalid path)")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	vmidStr := parts[2]
	vmid, err := strconv.Atoi(vmidStr)
	if err != nil {
		log.Printf("  -> 400 Bad Request (invalid vmid: %s)", vmidStr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Find the container
	var container *lxcResource
	for i := range mockContainers {
		if mockContainers[i].VMID == vmid {
			container = &mockContainers[i]
			break
		}
	}

	if container == nil {
		log.Printf("  -> 404 Not Found (vmid %d not found)", vmid)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Build detailed status response
	status := lxcStatus{
		VMID:    container.VMID,
		Name:    container.Name,
		Status:  container.Status,
		Type:    container.Type,
		Uptime:  container.Uptime,
		CPU:     container.CPU,
		CPUs:    container.MaxCPU,
		Mem:     container.Mem,
		MaxMem:  container.MaxMem,
		Swap:    0,
		MaxSwap: container.MaxMem / 4, // 25% of memory as swap
		Disk:    container.Disk,
		MaxDisk: container.MaxDisk,
		NetIn:   container.NetIn,
		NetOut:  container.NetOut,
	}

	// Add disk I/O for running containers
	if container.Status == "running" {
		status.DiskRead = uint64(rand.Int63n(10737418240))   // Up to 10 GiB
		status.DiskWrite = uint64(rand.Int63n(5368709120))  // Up to 5 GiB
		status.Swap = uint64(rand.Int63n(int64(status.MaxSwap / 10))) // Up to 10% of max swap
	}

	log.Printf("  -> 200 OK (container %d: %s)", vmid, container.Name)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statusResponse{Data: status})
}
