package metric

import (
	"slices"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

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
	var checkedSlice = make([]string, 0, 10) // To keep track of checked partitions

	// Set all flag "true" to get all partitions instead of just physical ones.
	partitions, partErr := disk.Partitions(true)

	if partErr != nil {
		diskErrors = append(diskErrors, CustomErr{
			Metric: []string{"disk.partitions"},
			Error:  partErr.Error(),
		})
	}

	for _, p := range partitions {
		// Filter out partitions that are already checked or loop devices
		// Include both /dev devices (except loops) and ZFS filesystems
		// * ZFS filesystems are not prefixed with /dev, so we check for that separately
		if slices.Contains(checkedSlice, p.Device) ||
			(strings.Contains(p.Device, "/dev/loop") || (!strings.HasPrefix(p.Device, "/dev") && p.Fstype != "zfs")) {
			continue
		}

		diskIOCounts, diskIOErr := disk.IOCounters(p.Device)
		if diskIOErr != nil {
			diskErrors = append(diskErrors, CustomErr{
				Metric: []string{"disk.read_bytes", "disk.write_bytes", "disk.read_time", "disk.write_time"},
				Error:  diskIOErr.Error() + " " + p.Device,
			})
			continue
		}

		deviceName := strings.TrimPrefix(p.Device, "/dev/")
		stats := diskIOCounts[deviceName]

		diskUsage, diskUsageErr := disk.Usage(p.Mountpoint)
		if diskUsageErr != nil {
			diskErrors = append(diskErrors, CustomErr{
				Metric: []string{"disk.usage_percent", "disk.total_bytes", "disk.free_bytes"},
				Error:  diskUsageErr.Error() + " " + p.Mountpoint,
			})
			continue
		}

		checkedSlice = append(checkedSlice, p.Device)
		metricsSlice = append(metricsSlice, &DiskData{
			Device:       p.Device,
			TotalBytes:   &diskUsage.Total,
			UsedBytes:    &diskUsage.Used,
			FreeBytes:    &diskUsage.Free,
			ReadBytes:    &stats.ReadBytes,
			WriteBytes:   &stats.WriteBytes,
			ReadTime:     &stats.ReadTime,
			WriteTime:    &stats.WriteTime,
			UsagePercent: RoundFloatPtr(diskUsage.UsedPercent/100, 4),

			TotalInodes:        &diskUsage.InodesTotal,
			FreeInodes:         &diskUsage.InodesFree,
			UsedInodes:         &diskUsage.InodesUsed,
			InodesUsagePercent: RoundFloatPtr(diskUsage.InodesUsedPercent/100, 4),
		})
	}

	if len(diskErrors) == 0 {
		return metricsSlice, nil
	}

	if len(metricsSlice) == 0 {
		return MetricsSlice{defaultDiskData[0]}, diskErrors
	}

	return metricsSlice, diskErrors
}
