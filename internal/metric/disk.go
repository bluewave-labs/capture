package metric

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

// sysfsRoot is overridable for tests.
var sysfsRoot = "/sys"

// isLoopbackDevice reports whether the partition's device path indicates a loopback device.
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

// collectPartitionMetrics collects available IO and usage metrics for the given partition.
// If at least one metric source (IO or usage) is available, it returns a DiskData populated with the corresponding fields.
// If neither source yields data, it returns nil and either a merged CustomErr of collection failures or a CustomErr with Metric ["disk"] and message "no disk stats available".
// If one or more collection operations failed while some metrics were gathered, it returns the partial DiskData and a merged CustomErr describing the failures.
func collectPartitionMetrics(partition disk.PartitionStat) (*DiskData, CustomErr) {
	var errs []CustomErr

	// Collect IO statistics (optional)
	ioStats, ioErr := collectIOStats(partition.Device)
	if ioErr != nil {
		errs = append(errs, *ioErr)
	}

	// Collect usage statistics (optional)
	usageStats, usageErr := collectUsageStats(partition.Mountpoint)
	if usageErr != nil {
		errs = append(errs, *usageErr)
	}

	if ioStats == nil && usageStats == nil {
		if len(errs) > 0 {
			return nil, mergeDiskErrors(errs)
		}
		return nil, CustomErr{Metric: []string{"disk"}, Error: "no disk stats available"}
	}

	data := &DiskData{Device: partition.Device}
	if usageStats != nil {
		data.TotalBytes = &usageStats.Total
		data.UsedBytes = &usageStats.Used
		data.FreeBytes = &usageStats.Free
		data.UsagePercent = RoundFloatPtr(usageStats.UsedPercent/100, 4)

		data.TotalInodes = &usageStats.InodesTotal
		data.FreeInodes = &usageStats.InodesFree
		data.UsedInodes = &usageStats.InodesUsed
		data.InodesUsagePercent = RoundFloatPtr(usageStats.InodesUsedPercent/100, 4)
	}

	if ioStats != nil {
		data.ReadBytes = &ioStats.ReadBytes
		data.WriteBytes = &ioStats.WriteBytes
		data.ReadTime = &ioStats.ReadTime
		data.WriteTime = &ioStats.WriteTime
	}

	if len(errs) == 0 {
		return data, CustomErr{}
	}

	return data, mergeDiskErrors(errs)
}

// mergeDiskErrors merges multiple CustomErr values into a single CustomErr.
// It aggregates unique metric names from all inputs (sorted) and concatenates non-empty error messages with "; " as the separator.
func mergeDiskErrors(errs []CustomErr) CustomErr {
	metricSet := map[string]struct{}{}
	metrics := make([]string, 0, 8)
	msgs := make([]string, 0, len(errs))

	for _, e := range errs {
		for _, m := range e.Metric {
			if _, ok := metricSet[m]; ok {
				continue
			}
			metricSet[m] = struct{}{}
			metrics = append(metrics, m)
		}
		if strings.TrimSpace(e.Error) != "" {
			msgs = append(msgs, e.Error)
		}
	}

	sort.Strings(metrics)
	return CustomErr{
		Metric: metrics,
		Error:  strings.Join(msgs, "; "),
	}
}

// collectIOStats gathers IO-related metrics for a device.
// Supports LVM/device-mapper by resolving /dev/mapper/* -> /dev/dm-*
// and trying multiple key candidates against the map returned by disk.IOCounters().
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
// buildDeviceKeyCandidates generates possible IO counter key names for a device path.
// It returns a deduplicated list of candidate keys that may appear in gopsutil/disk.IOCounters maps.
// For Windows it returns the trimmed device string (e.g., "C:").
// For non-Windows it includes the device with a "/dev/" prefix stripped, the path basename, any symlink-resolved equivalents, and, for "/dev/mapper/*" paths, the resolved "dm-<n>" name discovered via sysfs.
//
// device is the device path or name (for example "/dev/sda", "/dev/nvme0n1", or "/dev/mapper/vg-lv").
// The returned slice contains unique, non-empty candidate key strings in unspecified order.
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
	out = append(out, filepath.Base(d))

	// Resolve symlinks: /dev/mapper/vg-lv -> /dev/dm-0 -> dm-0
	if resolved, err := filepath.EvalSymlinks(d); err == nil && resolved != "" {
		out = append(out, strings.TrimPrefix(resolved, "/dev/"))
		out = append(out, filepath.Base(resolved))
	}

	// Resolve device-mapper name via sysfs: /dev/mapper/<name> -> dm-<n>
	if strings.HasPrefix(d, "/dev/mapper/") {
		if dm, ok := resolveDMNameFromMapperWithRoot(filepath.Base(d), sysfsRoot); ok {
			out = append(out, dm)
		}
	}

	// Deduplicate
	seen := map[string]struct{}{}
	uniq := make([]string, 0, len(out))
	for _, k := range out {
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

// resolveDMNameFromMapperWithRoot attempts to resolve a device-mapper name to its corresponding
// "dm-N" identifier by scanning sysfs under the provided root (e.g., "<root>/block/dm-*/dm/name").
// If mapperName is empty, not found, or an IO/error prevents resolution, it returns "" and false.
// On success it returns the matching "dm-N" name and true.
func resolveDMNameFromMapperWithRoot(mapperName, root string) (string, bool) {
	if strings.TrimSpace(mapperName) == "" {
		return "", false
	}

	pattern := filepath.Join(root, "block", "dm-*", "dm", "name")
	paths, err := filepath.Glob(pattern)
	if err != nil || len(paths) == 0 {
		return "", false
	}

	for _, p := range paths {
		b, readErr := os.ReadFile(p)
		if readErr != nil {
			continue
		}
		name := strings.TrimSpace(string(b))
		if name != mapperName {
			continue
		}
		// .../block/dm-0/dm/name -> dm-0
		dmDir := filepath.Dir(filepath.Dir(p))
		return filepath.Base(dmDir), true
	}

	return "", false
}

// collectUsageStats collects disk usage statistics for the given mountpoint.
// On success it returns the `disk.UsageStat`. On failure it returns a `CustomErr`
// whose `Metric` slice lists the usage-related metric names and whose `Error`
// message contains the original error text followed by the mountpoint.
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
// CollectDiskMetrics gathers disk metrics for all system partitions that meet the package's inclusion criteria.
// It returns a MetricsSlice containing per-partition DiskData and a slice of CustomErr for any collection errors; if no errors occurred the error slice is nil.
// The function filters and deduplicates partitions, accumulates available IO and usage metrics even when some sources fail, and—if errors exist but no metrics were collected—returns a single default "unknown" DiskData alongside the errors.
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
		// Apply filtering logic
		if !shouldIncludePartition(partition) {
			continue
		}

		// Skip duplicates (even on errors) to avoid repeated error spam
		if _, ok := checkedDevices[partition.Device]; ok {
			continue
		}
		checkedDevices[partition.Device] = struct{}{}

		// Gather metrics for valid partitions
		diskMetrics, err := collectPartitionMetrics(partition)
		if err.Error != "" {
			diskErrors = append(diskErrors, err)
		}
		if diskMetrics != nil {
			metricsSlice = append(metricsSlice, diskMetrics)
		}
	}

	if len(diskErrors) == 0 {
		return metricsSlice, nil
	}

	if len(metricsSlice) == 0 {
		return MetricsSlice{defaultDiskData[0]}, diskErrors
	}

	return metricsSlice, diskErrors
}