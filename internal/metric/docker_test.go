package metric

import (
	"testing"
)

// Mock data based on docker stats output:
// CONTAINER ID                                                       NAME              CPU %     MEM USAGE / LIMIT   MEM %     NET I/O         BLOCK I/O         PIDS
// 09e88f793869982fdb86e0ac183a4487a8bcf763179c9bbf2f8d6e25492f23bc   optimistic_jang   0.00%     4.488MiB / 15GiB    0.03%     8.03kB / 126B   3.58MB / 3.14MB   7

func TestCalculateCPUPercent(t *testing.T) {
	t.Run("zero CPU usage", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.CPUStats.CPUUsage.TotalUsage = 1000000
		stats.CPUStats.CPUUsage.PercpuUsage = []uint64{250000, 250000, 250000, 250000}
		stats.CPUStats.SystemUsage = 10000000000
		stats.CPUStats.OnlineCPUs = 4
		stats.PreCPUStats.CPUUsage.TotalUsage = 1000000
		stats.PreCPUStats.SystemUsage = 10000000000

		result := calculateCPUPercent(stats)
		if result != 0.0 {
			t.Errorf("calculateCPUPercent() = %v, expected 0.0", result)
		}
	})

	t.Run("normal CPU usage", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.CPUStats.CPUUsage.TotalUsage = 5000000
		stats.CPUStats.CPUUsage.PercpuUsage = []uint64{1250000, 1250000, 1250000, 1250000}
		stats.CPUStats.SystemUsage = 10100000000
		stats.CPUStats.OnlineCPUs = 4
		stats.PreCPUStats.CPUUsage.TotalUsage = 1000000
		stats.PreCPUStats.SystemUsage = 10000000000

		result := calculateCPUPercent(stats)
		expected := 16.0 // (4000000 / 100000000) * 4 * 100 = 16%
		// Allow small tolerance for floating point precision
		if result < expected-0.1 || result > expected+0.1 {
			t.Errorf("calculateCPUPercent() = %v, expected around %v", result, expected)
		}
	})

	t.Run("infer CPU count from PercpuUsage", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.CPUStats.CPUUsage.TotalUsage = 5000000
		stats.CPUStats.CPUUsage.PercpuUsage = []uint64{1250000, 1250000, 1250000, 1250000}
		stats.CPUStats.SystemUsage = 10100000000
		stats.CPUStats.OnlineCPUs = 0 // Not set
		stats.PreCPUStats.CPUUsage.TotalUsage = 1000000
		stats.PreCPUStats.SystemUsage = 10000000000

		result := calculateCPUPercent(stats)
		expected := 16.0 // (4000000 / 100000000) * 4 * 100 = 16%
		// Allow small tolerance for floating point precision
		if result < expected-0.1 || result > expected+0.1 {
			t.Errorf("calculateCPUPercent() = %v, expected around %v", result, expected)
		}
	})
}

func TestCalculateMemoryMetrics(t *testing.T) {
	t.Run("optimistic_jang container - 4.488MiB / 15GiB (0.03%)", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.MemoryStats.Usage = 5238784             // Raw usage: 4.996 MiB
		stats.MemoryStats.Stats.InactiveFile = 532480 // Inactive file cache: 0.508 MiB
		stats.MemoryStats.Limit = 16101339136         // 15 GiB in bytes

		usage, limit, percentage := calculateMemoryMetrics(stats)
		// Docker CLI calculation: usage - inactive_file = 5238784 - 532480 = 4706304 bytes (4.488 MiB)
		expectedUsage := uint64(4706304)
		if usage != expectedUsage {
			t.Errorf("usage = %v, expected %v", usage, expectedUsage)
		}
		if limit != 16101339136 {
			t.Errorf("limit = %v, expected 16101339136", limit)
		}
		expectedPercentage := 0.0292
		// Allow small tolerance for floating point precision
		if percentage < expectedPercentage-0.0001 || percentage > expectedPercentage+0.0001 {
			t.Errorf("percentage = %v, expected around %v", percentage, expectedPercentage)
		}
	})

	t.Run("memory without inactive_file (no cache)", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.MemoryStats.Usage = 1048576
		stats.MemoryStats.Stats.InactiveFile = 0 // No cache
		stats.MemoryStats.Limit = 2097152

		usage, limit, percentage := calculateMemoryMetrics(stats)
		// When inactive_file is 0, usage should remain unchanged
		if usage != 1048576 {
			t.Errorf("usage = %v, expected 1048576", usage)
		}
		if limit != 2097152 {
			t.Errorf("limit = %v, expected 2097152", limit)
		}
		expectedPercentage := 50.0
		if percentage != expectedPercentage {
			t.Errorf("percentage = %v, expected %v", percentage, expectedPercentage)
		}
	})

	t.Run("zero memory limit", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.MemoryStats.Usage = 1048576
		stats.MemoryStats.Limit = 0

		usage, limit, percentage := calculateMemoryMetrics(stats)
		if usage != 1048576 {
			t.Errorf("usage = %v, expected 1048576", usage)
		}
		if limit != 0 {
			t.Errorf("limit = %v, expected 0", limit)
		}
		if percentage != 0.0 {
			t.Errorf("percentage = %v, expected 0.0", percentage)
		}
	})
}

func TestCalculateNetworkMetrics(t *testing.T) {
	t.Run("optimistic_jang container - 8.03kB / 126B", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.Networks = make(map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		})
		stats.Networks["eth0"] = struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		}{
			RxBytes: 8030, // 8.03 kB
			TxBytes: 126,
		}

		rx, tx := calculateNetworkMetrics(stats)
		if rx != 8030 {
			t.Errorf("rx = %v, expected 8030", rx)
		}
		if tx != 126 {
			t.Errorf("tx = %v, expected 126", tx)
		}
	})

	t.Run("multiple network interfaces", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.Networks = make(map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		})
		stats.Networks["eth0"] = struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		}{
			RxBytes: 5000,
			TxBytes: 100,
		}
		stats.Networks["eth1"] = struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		}{
			RxBytes: 3030,
			TxBytes: 26,
		}

		rx, tx := calculateNetworkMetrics(stats)
		if rx != 8030 {
			t.Errorf("rx = %v, expected 8030", rx)
		}
		if tx != 126 {
			t.Errorf("tx = %v, expected 126", tx)
		}
	})

	t.Run("nil networks", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.Networks = nil

		rx, tx := calculateNetworkMetrics(stats)
		if rx != 0 {
			t.Errorf("rx = %v, expected 0", rx)
		}
		if tx != 0 {
			t.Errorf("tx = %v, expected 0", tx)
		}
	})
}

func TestCalculateBlockIOMetrics(t *testing.T) {
	t.Run("optimistic_jang container - 3.58MB / 3.14MB", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.BlkioStats.IoServiceBytesRecursive = []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		}{
			{Op: "read", Value: 3580000},  // 3.58 MB
			{Op: "write", Value: 3140000}, // 3.14 MB
		}

		read, write := calculateBlockIOMetrics(stats)
		if read != 3580000 {
			t.Errorf("read = %v, expected 3580000", read)
		}
		if write != 3140000 {
			t.Errorf("write = %v, expected 3140000", write)
		}
	})

	t.Run("uppercase operation names", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.BlkioStats.IoServiceBytesRecursive = []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		}{
			{Op: "Read", Value: 1000000},
			{Op: "Write", Value: 500000},
		}

		read, write := calculateBlockIOMetrics(stats)
		if read != 1000000 {
			t.Errorf("read = %v, expected 1000000", read)
		}
		if write != 500000 {
			t.Errorf("write = %v, expected 500000", write)
		}
	})

	t.Run("multiple read and write operations", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.BlkioStats.IoServiceBytesRecursive = []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		}{
			{Op: "read", Value: 2000000},
			{Op: "read", Value: 1580000},
			{Op: "write", Value: 2000000},
			{Op: "write", Value: 1140000},
		}

		read, write := calculateBlockIOMetrics(stats)
		if read != 3580000 {
			t.Errorf("read = %v, expected 3580000", read)
		}
		if write != 3140000 {
			t.Errorf("write = %v, expected 3140000", write)
		}
	})

	t.Run("empty blkio stats", func(t *testing.T) {
		stats := dockerStatsResponse{}
		stats.BlkioStats.IoServiceBytesRecursive = []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		}{}

		read, write := calculateBlockIOMetrics(stats)
		if read != 0 {
			t.Errorf("read = %v, expected 0", read)
		}
		if write != 0 {
			t.Errorf("write = %v, expected 0", write)
		}
	})
}

func TestGetContainerName(t *testing.T) {
	t.Run("name with leading slash", func(t *testing.T) {
		result := getContainerName([]string{"/optimistic_jang"})
		if result != "optimistic_jang" {
			t.Errorf("getContainerName() = %v, expected optimistic_jang", result)
		}
	})

	t.Run("name without leading slash", func(t *testing.T) {
		result := getContainerName([]string{"optimistic_jang"})
		if result != "optimistic_jang" {
			t.Errorf("getContainerName() = %v, expected optimistic_jang", result)
		}
	})

	t.Run("empty names array", func(t *testing.T) {
		result := getContainerName([]string{})
		if result != "" {
			t.Errorf("getContainerName() = %v, expected empty string", result)
		}
	})

	t.Run("empty first name", func(t *testing.T) {
		result := getContainerName([]string{""})
		if result != "" {
			t.Errorf("getContainerName() = %v, expected empty string", result)
		}
	})

	t.Run("multiple names", func(t *testing.T) {
		result := getContainerName([]string{"/optimistic_jang", "/another_name"})
		if result != "optimistic_jang" {
			t.Errorf("getContainerName() = %v, expected optimistic_jang", result)
		}
	})
}

func TestGetUnixTimestamp(t *testing.T) {
	t.Run("valid RFC3339Nano timestamp", func(t *testing.T) {
		result := GetUnixTimestamp("2023-11-28T10:30:45.123456789Z")
		// Check that it's a valid positive timestamp (not zero)
		if result <= 0 {
			t.Errorf("GetUnixTimestamp() = %v, expected positive timestamp", result)
		}
		// Verify it's in the expected range (Nov 2023)
		if result < 1700000000 || result > 1702000000 {
			t.Errorf("GetUnixTimestamp() = %v, expected timestamp around Nov 2023", result)
		}
	})

	t.Run("valid RFC3339 timestamp without nano", func(t *testing.T) {
		result := GetUnixTimestamp("2023-11-28T10:30:45Z")
		// Check that it's a valid positive timestamp (not zero)
		if result <= 0 {
			t.Errorf("GetUnixTimestamp() = %v, expected positive timestamp", result)
		}
		// Verify it's in the expected range (Nov 2023)
		if result < 1700000000 || result > 1702000000 {
			t.Errorf("GetUnixTimestamp() = %v, expected timestamp around Nov 2023", result)
		}
	})

	t.Run("zero value timestamp", func(t *testing.T) {
		result := GetUnixTimestamp("0001-01-01T00:00:00Z")
		if result != 0 {
			t.Errorf("GetUnixTimestamp() = %v, expected 0", result)
		}
	})

	t.Run("invalid timestamp", func(t *testing.T) {
		result := GetUnixTimestamp("invalid")
		if result != 0 {
			t.Errorf("GetUnixTimestamp() = %v, expected 0", result)
		}
	})

	t.Run("empty timestamp", func(t *testing.T) {
		result := GetUnixTimestamp("")
		if result != 0 {
			t.Errorf("GetUnixTimestamp() = %v, expected 0", result)
		}
	})
}

// TestMockContainerStats tests the complete container stats structure
// based on the mock docker stats output
func TestMockContainerStats(t *testing.T) {
	// Mock data for: 09e88f793869982fdb86e0ac183a4487a8bcf763179c9bbf2f8d6e25492f23bc optimistic_jang
	mockStats := dockerStatsResponse{}

	// CPU Stats
	mockStats.CPUStats.CPUUsage.TotalUsage = 1000000 // Minimal usage for 0.00%
	mockStats.CPUStats.CPUUsage.PercpuUsage = []uint64{250000, 250000, 250000, 250000}
	mockStats.CPUStats.SystemUsage = 10000000000
	mockStats.CPUStats.OnlineCPUs = 4
	mockStats.PreCPUStats.CPUUsage.TotalUsage = 1000000
	mockStats.PreCPUStats.SystemUsage = 10000000000

	// Memory Stats
	mockStats.MemoryStats.Usage = 5238784             // Raw usage: 4.996 MiB
	mockStats.MemoryStats.Stats.InactiveFile = 532480 // Inactive file cache: 0.508 MiB
	mockStats.MemoryStats.Limit = 16101339136         // 15 GiB

	// Network Stats
	mockStats.Networks = make(map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	})
	mockStats.Networks["eth0"] = struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	}{
		RxBytes: 8030, // 8.03 kB
		TxBytes: 126,
	}

	// Block I/O Stats
	mockStats.BlkioStats.IoServiceBytesRecursive = []struct {
		Op    string `json:"op"`
		Value uint64 `json:"value"`
	}{
		{Op: "read", Value: 3580000},  // 3.58 MB
		{Op: "write", Value: 3140000}, // 3.14 MB
	}

	// PIDs Stats
	mockStats.PidsStats.Current = 7

	// Test CPU calculation
	cpuPercent := calculateCPUPercent(mockStats)
	if cpuPercent > 0.01 { // Should be close to 0%
		t.Errorf("CPU percent should be near 0%%, got %v", cpuPercent)
	}

	// Test memory calculation
	memUsage, memLimit, memPercent := calculateMemoryMetrics(mockStats)
	// Expected: 5238784 - 532480 = 4706304 bytes (4.488 MiB)
	expectedMemUsage := uint64(4706304)
	if memUsage != expectedMemUsage {
		t.Errorf("Memory usage = %v, expected %v (usage - inactive_file)", memUsage, expectedMemUsage)
	}
	if memLimit != 16101339136 {
		t.Errorf("Memory limit = %v, expected 16101339136", memLimit)
	}
	if memPercent < 0.029 || memPercent > 0.03 {
		t.Errorf("Memory percent = %v, expected around 0.03%%", memPercent)
	}

	// Test network calculation
	rx, tx := calculateNetworkMetrics(mockStats)
	if rx != 8030 {
		t.Errorf("Network RX = %v, expected 8030", rx)
	}
	if tx != 126 {
		t.Errorf("Network TX = %v, expected 126", tx)
	}

	// Test block I/O calculation
	blockRead, blockWrite := calculateBlockIOMetrics(mockStats)
	if blockRead != 3580000 {
		t.Errorf("Block read = %v, expected 3580000", blockRead)
	}
	if blockWrite != 3140000 {
		t.Errorf("Block write = %v, expected 3140000", blockWrite)
	}

	// Test PIDs
	if mockStats.PidsStats.Current != 7 {
		t.Errorf("PIDs = %v, expected 7", mockStats.PidsStats.Current)
	}

	t.Logf("Mock container stats validated successfully")
	t.Logf("Container ID: 09e88f793869982fdb86e0ac183a4487a8bcf763179c9bbf2f8d6e25492f23bc")
	t.Logf("Container Name: optimistic_jang")
	t.Logf("CPU: %.2f%%", cpuPercent)
	t.Logf("Memory: %.2fMiB / %.2fGiB (%.2f%%)", float64(memUsage)/1048576, float64(memLimit)/1073741824, memPercent)
	t.Logf("Network: %d / %d bytes", rx, tx)
	t.Logf("Block I/O: %d / %d bytes", blockRead, blockWrite)
	t.Logf("PIDs: %d", mockStats.PidsStats.Current)
}
