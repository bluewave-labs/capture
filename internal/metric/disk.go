package metric

import (
	"os"
	"path/filepath"
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

	// Combine all metrics into a DiskData structure
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

// collectIOStats gathers IO-related metrics for a device.
// Supports LVM/device-mapper by resolving /dev/mapper/* -> /dev/dm-*,
// searching /sys/block for the matching dm-* device, and trying multiple
// key candidates against the map returned by disk.IOCounters().
func collectIOStats(device string) (*disk.IOCountersStat, *CustomErr) {
	// Get all counters once and look up by key
	all, err := disk.IOCounters()
	if err != nil {
		return nil, &CustomErr{
			Metric: []string{"disk.read_bytes", "disk.write_bytes", "disk.read_time", "disk.write_time"},
			Error:  err.Error() + " " + device,
		}
	}

	candidates := buildDeviceKeyCandidates(device)

	// 1) Direct map key match
	for _, k := range candidates {
		if stat, ok := all[k]; ok {
			return &stat, nil
		}
	}

	// 2) Fallback: match by stat.Name field
	for _, stat := range all {
		for _, k := range candidates {
			if stat.Name == k {
				s := stat
				return &s, nil
			}
		}
	}

	return nil, &CustomErr{
		Metric: []string{"disk.read_bytes", "disk.write_bytes", "disk.read_time", "disk.write_time"},
		Error:  "device stats not found: " + device + " (tried: " + strings.Join(candidates, ", ") + ")",
	}
}

// buildDeviceKeyCandidates returns possible keys for the disk.IOCounters() map.
// Handles paths like /dev/sda, /dev/nvme0n1, and /dev/mapper/vg-lv by:
//  - stripping /dev/
//  - taking the basename
//  - resolving symlinks when applicable
//  - scanning /sys/block/dm-*/dm/name to find the matching dm-* device
func buildDeviceKeyCandidates(device string) []string {
	if runtime.GOOS == "windows" {
		// On Windows, gopsutil uses names like "C:", so keep as-is.
		d := strings.TrimSpace(device)
		return []string{d}
	}

	var out []string
	d := strings.TrimSpace(device)

	// Strip /dev/
	out = append(out, strings.TrimPrefix(d, "/dev/"))
	// Basename (e.g., /dev/mapper/vg-lv -> vg-lv)
	base := filepath.Base(d)
	out = append(out, base)

	// Resolve symlinks (works for typical udev/by-id/by-uuid, not for mapper pseudo-devices)
	if resolved, err := filepath.EvalSymlinks(d); err == nil && resolved != "" {
		out = append(out, strings.TrimPrefix(resolved, "/dev/"))
		out = append(out, filepath.Base(resolved))
	}

	// If it's an LVM/device-mapper path, try to discover dm-* via /sys/block
	if strings.HasPrefix(d, "/dev/mapper/") || strings.HasPrefix(base, "dm-") {
		if dm := findDMForMapperBase(base); dm != "" {
			// Add dm-* key (this is what gopsutil uses in IOCounters)
			out = append(out, dm)
		}
	}

	// Deduplicate and drop empties
	seen := map[string]struct{}{}
	uniq := make([]string, 0, len(out))
	for _, k := range out {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			uniq = append(uniq, k)
		}
	}
	return uniq
}

// findDMForMapperBase tries to map a /dev/mapper/<name> basename to its dm-*
// by scanning /sys/block/dm-*/dm/name and comparing values.
//
// For LVM, the mapper basename typically matches the content of /sys/block/dm-*/dm/name.
// Example:
//   /dev/mapper/ubuntu--vg-ubuntu--lv  ->  /sys/block/dm-0/dm/name == "ubuntu--vg-ubuntu--lv"  => dm-0
func findDMForMapperBase(mapperBase string) string {
	const sysBlock = "/sys/block"
	entries, err := os.ReadDir(sysBlock)
	if err != nil {
		return ""
	}

	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "dm-") {
			continue
		}
		// Read /sys/block/dm-*/dm/name to get the logical name
		dmNamePath := filepath.Join(sysBlock, name, "dm", "name")
		b, err := os.ReadFile(dmNamePath)
		if err != nil {
			continue
		}
		dmLogical := strings.TrimSpace(string(b))

		// Compare mapper basename to dm logical name.
		// LVM encodes '-' as '--' in names; mapperBase already carries that encoding,
		// and /sys/block/.../dm/name typically matches the same encoding.
		if dmLogical == mapperBase {
			return name
		}

		// Extra tolerance: also try a relaxed comparison removing all slashes
		// and comparing lowercased (helps in edge cases with udev rules).
		if strings.EqualFold(strings.ReplaceAll(dmLogical, "/", ""), strings.ReplaceAll(mapperBase, "/", "")) {
			return name
		}
	}

	return ""
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
// Lists all partitions on the system using disk.Partitions(all=true).
// Checks each partition for filtering conditions.
// For each valid partition, gathers the specified metrics.
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
	var checkedDevices = make(map[string]struct{}) // Track already processed devices

	// List all partitions on the system
	partitions, partErr := disk.Partitions(true)
	if partErr != nil {
		diskErrors = append(diskErrors, CustomErr{
			Metric: []string{"disk.partitions"},
			Error:  partErr.Error(),
		})
		return MetricsSlice{defaultDiskData[0]}, diskErrors
	}

	// Iterate through partitions and apply filters
	for _, partition := range partitions {
		// Skip duplicates
		if _, ok := checkedDevices[partition.Device]; ok {
			continue
		}

		// Apply filtering logic
		if !shouldIncludePartition(partition) {
			continue
		}

		// Gather metrics for valid partitions
		diskMetrics, err := collectPartitionMetrics(partition)
		if err.Error != "" {
			diskErrors = append(diskErrors, err)
			continue
		}

		checkedDevices[partition.Device] = struct{}{} // Mark as checked
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
