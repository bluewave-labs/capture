package metric

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

// sysfsRoot is overridable for tests.
var sysfsRoot = "/sys"

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
	var errs []CustomErr

	// Collect IO statistics (optional).
	// ZFS pools are identified by pool name rather than a /dev/ block device path,
	// so they require a dedicated collector that reads from the ZFS kstat interface
	// or falls back to the zpool command.
	var ioStats *disk.IOCountersStat
	var ioErr *CustomErr
	if isZFSFilesystem(partition) {
		ioStats, ioErr = collectZFSIOStats(partition.Device)
	} else {
		ioStats, ioErr = collectIOStats(partition.Device)
	}
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

	data := &DiskData{Device: partition.Device, Mountpoint: partition.Mountpoint}
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
// Handles paths like /dev/sda, /dev/nvme0n1, /dev/mapper/vg-lv -> dm-0, etc.
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

// collectZFSIOStats gathers IO-related metrics for a ZFS pool.
//
// ZFS pools expose cumulative I/O counters through the Linux kernel statistics
// interface at /proc/spl/kstat/zfs/<pool>/io. When that interface is unavailable
// (non-Linux or kernel module not loaded) the function falls back to parsing the
// output of `zpool iostat`. If neither source is reachable it returns nil without
// an error – the absence of ZFS I/O counters is not a failure condition, since
// ZFS does not register itself as a standard block device in /proc/diskstats.
func collectZFSIOStats(poolName string) (*disk.IOCountersStat, *CustomErr) {
	// Preferred path: Linux OpenZFS kstat interface (cumulative counters).
	if runtime.GOOS == "linux" {
		if stat, err := readZFSKStat(poolName); err == nil && stat != nil {
			return stat, nil
		}
	}

	// Fallback: zpool iostat command.
	if stat, err := zpoolIOStat(poolName); err == nil && stat != nil {
		return stat, nil
	}

	// ZFS IO stats are unavailable – return nil without an error.
	return nil, nil
}

// readZFSKStat reads cumulative IO counters for poolName from the Linux kernel
// statistics interface at /proc/spl/kstat/zfs/<poolName>/io.
func readZFSKStat(poolName string) (*disk.IOCountersStat, error) {
	path := filepath.Join("/proc/spl/kstat/zfs", poolName, "io")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	stat := &disk.IOCountersStat{Name: poolName}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		val, parseErr := strconv.ParseUint(fields[2], 10, 64)
		if parseErr != nil {
			continue
		}
		switch fields[0] {
		case "nread":
			stat.ReadBytes = val
		case "nwritten":
			stat.WriteBytes = val
		case "rtime":
			// rtime is in nanoseconds; convert to milliseconds to match gopsutil convention.
			stat.ReadTime = val / 1_000_000
		case "wtime":
			stat.WriteTime = val / 1_000_000
		case "reads":
			stat.ReadCount = val
		case "writes":
			stat.WriteCount = val
		}
	}
	return stat, nil
}

// zpoolIOStat collects IO statistics for a ZFS pool by running
// `zpool iostat -H -p <poolName>`, which reports average throughput since the
// pool was last imported or statistics were cleared. The function returns nil,
// nil when the zpool binary is unavailable or the command fails.
func zpoolIOStat(poolName string) (*disk.IOCountersStat, error) {
	// -H: scripted/parseable output (tab-separated, no headers).
	// -p: exact numeric values.
	out, err := exec.Command("zpool", "iostat", "-H", "-p", poolName).Output()
	if err != nil {
		return nil, fmt.Errorf("zpool iostat: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty zpool iostat output for pool %s", poolName)
	}

	// Expected columns: pool  alloc  free  read_iops  write_iops  read_bw  write_bw
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 7 {
		return nil, fmt.Errorf("unexpected zpool iostat output: %q", lines[len(lines)-1])
	}

	parseField := func(s string) (uint64, error) {
		if s == "-" {
			return 0, nil
		}
		return strconv.ParseUint(s, 10, 64)
	}

	readOps, err := parseField(fields[3])
	if err != nil {
		return nil, fmt.Errorf("parsing read_iops: %w", err)
	}
	writeOps, err := parseField(fields[4])
	if err != nil {
		return nil, fmt.Errorf("parsing write_iops: %w", err)
	}
	readBW, err := parseField(fields[5])
	if err != nil {
		return nil, fmt.Errorf("parsing read_bw: %w", err)
	}
	writeBW, err := parseField(fields[6])
	if err != nil {
		return nil, fmt.Errorf("parsing write_bw: %w", err)
	}

	return &disk.IOCountersStat{
		Name:       poolName,
		ReadBytes:  readBW,
		WriteBytes: writeBW,
		ReadCount:  readOps,
		WriteCount: writeOps,
	}, nil
}

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
			Mountpoint:         "unknown",
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
