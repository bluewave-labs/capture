package metric

import (
	"slices"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

func CollectDiskMetrics() (MetricsSlice, []CustomErr) {
	defaultDiskData := []*DiskData{
		{
			Device:          "unknown",
			ReadSpeedBytes:  nil,
			WriteSpeedBytes: nil,
			TotalBytes:      nil,
			FreeBytes:       nil,
			UsagePercent:    nil,
		},
	}
	var diskErrors []CustomErr
	var metricsSlice MetricsSlice
	var checkedSlice []string // To keep track of checked partitions

	// Set all flag to "false" to get only necessary partitions
	// Avoiding unnecessary partitions like /run/user/1000, /run/credentials
	partitions, partErr := disk.Partitions(false)

	if partErr != nil {
		diskErrors = append(diskErrors, CustomErr{
			Metric: []string{"disk.partitions"},
			Error:  partErr.Error(),
		})
	}

	for _, p := range partitions {
		// Filter out partitions that are already checked or not a device
		// Also, exclude '/dev/loop' devices to avoid unnecessary partitions
		// * /dev/loop devices are used for mounting snap packages
		if slices.Contains(checkedSlice, p.Device) || !strings.HasPrefix(p.Device, "/dev") || strings.HasPrefix(p.Device, "/dev/loop") {
			continue
		}

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
			Device:          p.Device,
			ReadSpeedBytes:  nil, // TODO: Implement
			WriteSpeedBytes: nil, // TODO: Implement
			TotalBytes:      &diskUsage.Total,
			FreeBytes:       &diskUsage.Free,
			UsagePercent:    RoundFloatPtr(diskUsage.UsedPercent/100, 4),
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
