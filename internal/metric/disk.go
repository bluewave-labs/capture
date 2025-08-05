package metric

import (
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

// isLoopbackDevice checks if the partition is a loopback device.
func isLoopbackDevice(p disk.PartitionStat) bool {
	return strings.Contains(p.Device, "/dev/loop")
}

// isZFSFilesystem checks if the partition type is ZFS.
func isZFSFilesystem(p disk.PartitionStat) bool {
	return p.Fstype == "zfs"
}

// isDevPrefixed checks if the device path starts with /dev.
func isDevPrefixed(p disk.PartitionStat) bool {
	return strings.HasPrefix(p.Device, "/dev")
}

// isWindowsDrive checks if the device is a Windows drive (C:, D:, etc.).
func isWindowsDrive(p disk.PartitionStat) bool {
	// Windows drives typically look like "C:", "D:", etc.
	device := strings.TrimSpace(p.Device)
	if len(device) >= 2 {
		return device[1] == ':' && ((device[0] >= 'A' && device[0] <= 'Z') || (device[0] >= 'a' && device[0] <= 'z'))
	}
	return false
}

// isSpecialPartition checks if the partition is a special system partition
// (Recovery, EFI, System Reserved, etc.).
func isSpecialPartition(p disk.PartitionStat) bool {
	deviceUpper := strings.ToUpper(p.Device)

	specialPatterns := []string{
		"RECOVERY",
		"SYSTEM RESERVED",
		"EFI",
	}

	for _, pattern := range specialPatterns {
		if strings.Contains(deviceUpper, pattern) {
			return true
		}
	}

	return false
}

// shouldIncludePartition determines if a partition should be included in metrics
// collection based on the disk metric flow rules.
func shouldIncludePartition(partition disk.PartitionStat) bool {
	// Always include ZFS filesystems
	if isZFSFilesystem(partition) {
		return true
	}

	// Skip loopback devices
	if isLoopbackDevice(partition) {
		return false
	}

	// Skip special system partitions
	if isSpecialPartition(partition) {
		return false
	}

	// For Unix systems, require /dev prefix
	if runtime.GOOS != "windows" {
		if !isDevPrefixed(partition) {
			return false
		}
	} else {
		// For Windows, include drives that look like C:, D:, etc.
		if !isWindowsDrive(partition) {
			return false
		}
	}

	return true
}

// collectPartitionMetrics gathers all required metrics for a single partition.
func collectPartitionMetrics(partition disk.PartitionStat) (*DiskData, CustomErr) {
	// Collect IO statistics
	ioStats, ioErr := collectIOStats(partition.Device)
	if ioErr != nil {
		return nil, *ioErr
	}

	// Collect usage statistics
	usageStats, usageErr := collectUsageStats(partition.Mountpoint)
	if usageErr != nil {
		return nil, *usageErr
	}

	// Combine all metrics into DiskData structure
	return &DiskData{
		Device:       partition.Device,
		TotalBytes:   &usageStats.Total,
		UsedBytes:    &usageStats.Used,
		FreeBytes:    &usageStats.Free,
		UsagePercent: RoundFloatPtr(usageStats.UsedPercent/100, 4),

		TotalInodes:        &usageStats.InodesTotal,
		FreeInodes:         &usageStats.InodesFree,
		UsedInodes:         &usageStats.InodesUsed,
		InodesUsagePercent: RoundFloatPtr(usageStats.InodesUsedPercent/100, 4),

		ReadBytes:  &ioStats.ReadBytes,
		WriteBytes: &ioStats.WriteBytes,
		ReadTime:   &ioStats.ReadTime,
		WriteTime:  &ioStats.WriteTime,
	}, CustomErr{}
}

// collectIOStats collects IO-related metrics for a device.
func collectIOStats(device string) (*disk.IOCountersStat, *CustomErr) {
	diskIOCounts, diskIOErr := disk.IOCounters(device)
	if diskIOErr != nil {
		return nil, &CustomErr{
			Metric: []string{"disk.read_bytes", "disk.write_bytes", "disk.read_time", "disk.write_time"},
			Error:  diskIOErr.Error() + " " + device,
		}
	}

	// Extract device name for lookup (handle cross-platform differences)
	deviceName := device
	if runtime.GOOS != "windows" {
		deviceName = strings.TrimPrefix(device, "/dev/")
	}

	stats, exists := diskIOCounts[deviceName]
	if !exists {
		return nil, &CustomErr{
			Metric: []string{"disk.read_bytes", "disk.write_bytes", "disk.read_time", "disk.write_time"},
			Error:  "device stats not found: " + device,
		}
	}

	return &stats, nil
}

// collectUsageStats collects usage-related metrics for a mountpoint.
func collectUsageStats(mountpoint string) (*disk.UsageStat, *CustomErr) {
	diskUsage, diskUsageErr := disk.Usage(mountpoint)
	if diskUsageErr != nil {
		return nil, &CustomErr{
			Metric: []string{"disk.usage_percent", "disk.total_bytes", "disk.free_bytes", "disk.used_bytes",
				"disk.total_inodes", "disk.free_inodes", "disk.used_inodes", "disk.inodes_usage_percent"},
			Error: diskUsageErr.Error() + " " + mountpoint,
		}
	}

	return diskUsage, nil
}

// CollectDiskMetrics collects disk metrics following the disk metric flow specification.
// List all partitions on the system using disk.Partitions(all=true)
// Check each partition for filtering conditions
// For each valid partition, gather the specified metrics
func CollectDiskMetrics() (MetricsSlice, []CustomErr) {
	defaultDiskData := []*DiskData{
		{
			Device:             "unknown",
			TotalBytes:         nil,
			FreeBytes:          nil,
			UsedBytes:          nil,
			ReadBytes:          nil,
			WriteBytes:         nil,
			ReadTime:           nil,
			WriteTime:          nil,
			UsagePercent:       nil,
			TotalInodes:        nil,
			FreeInodes:         nil,
			UsedInodes:         nil,
			InodesUsagePercent: nil,
		},
	}

	var diskErrors []CustomErr
	var metricsSlice MetricsSlice
	var checkedDevices = make(map[string]struct{}) // To keep track of checked partitions

	// List all partitions on the system. Using disk.Partitions(all=true)
	partitions, partErr := disk.Partitions(true)
	if partErr != nil {
		diskErrors = append(diskErrors, CustomErr{
			Metric: []string{"disk.partitions"},
			Error:  partErr.Error(),
		})
		return MetricsSlice{defaultDiskData[0]}, diskErrors
	}

	// Check each partition for the filtering
	for _, partition := range partitions {
		// Check if the partition is already checked to avoid duplicates
		if _, ok := checkedDevices[partition.Device]; ok {
			continue
		}

		// Apply filtering logic based on the disk metric flow
		if !shouldIncludePartition(partition) {
			continue
		}

		// For each valid partition, gather the specified metrics
		diskMetrics, err := collectPartitionMetrics(partition)
		if err.Error != "" {
			diskErrors = append(diskErrors, err)
			continue
		}

		checkedDevices[partition.Device] = struct{}{} // Mark this partition as checked
		metricsSlice = append(metricsSlice, diskMetrics)
	}

	if len(diskErrors) == 0 {
		return metricsSlice, nil
	}

	if len(metricsSlice) == 0 {
		return MetricsSlice{defaultDiskData[0]}, diskErrors
	}

	return metricsSlice, diskErrors
}
