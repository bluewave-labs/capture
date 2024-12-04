package metric

import (
	"github.com/shirou/gopsutil/v4/disk"
)

func CollectDiskMetrics() (MetricsSlice, []CustomErr) {
	defaultDiskData := []*DiskData{
		{
			MountPoint:      "",
			ReadSpeedBytes:  nil,
			WriteSpeedBytes: nil,
			TotalBytes:      nil,
			FreeBytes:       nil,
			UsagePercent:    nil,
		},
	}
	var diskErrors []CustomErr
	var metricsSlice MetricsSlice

	// Set all flag to false to get only necessary partitions
	// Avoiding unnecessary partitions like /run/user/1000, /run/credentials
	partitions, partErr := disk.Partitions(false)

	if partErr != nil {
		diskErrors = append(diskErrors, CustomErr{
			Metric: []string{"disk.partitions"},
			Error:  partErr.Error(),
		})
	}

	for _, p := range partitions {
		diskUsage, diskUsageErr := disk.Usage(p.Mountpoint)

		if diskUsageErr != nil {
			diskErrors = append(diskErrors, CustomErr{
				Metric: []string{"disk.usage_percent", "disk.total_bytes", "disk.free_bytes"},
				Error:  diskUsageErr.Error() + p.Mountpoint,
			})
			return MetricsSlice{defaultDiskData[0]}, diskErrors
		}

		metricsSlice = append(metricsSlice, &DiskData{
			MountPoint:      diskUsage.Path,
			ReadSpeedBytes:  nil, // TODO: Implement
			WriteSpeedBytes: nil, // TODO: Implement
			TotalBytes:      &diskUsage.Total,
			FreeBytes:       &diskUsage.Free,
			UsagePercent:    RoundFloatPtr(diskUsage.UsedPercent/100, 4),
		})
	}

	return metricsSlice, diskErrors
}

// func CollectDiskMetricsTrial() (map[string]disk2.IOCountersStat, error) {
// 	diskIOCounts, diskIOCountErr := disk2.IOCounters()

// 	if diskIOCountErr != nil {
// 		return nil, diskIOCountErr
// 	}

// 	for a, i := range diskIOCounts {
// 		fmt.Println(a)
// 		fmt.Println(i.Name)
// 	}

// 	return diskIOCounts, nil
// }
