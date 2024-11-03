package metric

import (
	disk2 "github.com/shirou/gopsutil/v4/disk"
)

func CollectDiskMetrics() (MetricsSlice, []CustomErr) {
	defaultDiskData := []*DiskData{
		{
			ReadSpeedBytes:  nil,
			WriteSpeedBytes: nil,
			TotalBytes:      nil,
			FreeBytes:       nil,
			UsagePercent:    nil,
		},
	}
	var diskErrors []CustomErr
	diskUsage, diskUsageErr := disk2.Usage("/")

	if diskUsageErr != nil {
		diskErrors = append(diskErrors, CustomErr{
			Metric: []string{"disk.usage_percent", "disk.total_bytes", "disk.free_bytes"},
			Error:  diskUsageErr.Error(),
		})
		return MetricsSlice{defaultDiskData[0]}, diskErrors
	}

	// diskMetrics, diskErr := disk1.Get()
	// if diskErr != nil {
	// 	log.Fatalf("Unable to get disk metrics")
	// }

	// for _, p := range diskMetrics {
	// 	fmt.Println(p.Name, p.ReadsCompleted)
	// }

	// var a uint64 = 2e+12
	var metricsSlice MetricsSlice

	metricsSlice = append(metricsSlice, &DiskData{
		ReadSpeedBytes:  nil, // TODO: Implement
		WriteSpeedBytes: nil, // TODO: Implement
		TotalBytes:      &diskUsage.Total,
		FreeBytes:       &diskUsage.Free,
		UsagePercent:    RoundFloatPtr(diskUsage.UsedPercent/100, 4),
	})
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
