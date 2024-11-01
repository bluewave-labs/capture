package metric

import (
	disk2 "github.com/shirou/gopsutil/v4/disk"
)

func CollectDiskMetrics() (*[]*DiskData, []string) {
	defaultDiskData := []*DiskData{
		{
			ReadSpeedBytes:  nil,
			WriteSpeedBytes: nil,
			TotalBytes:      nil,
			FreeBytes:       nil,
			UsagePercent:    nil,
		},
	}
	var diskData []*DiskData
	var diskErrors []string
	diskUsage, diskUsageErr := disk2.Usage("/")

	if diskUsageErr != nil {
		diskErrors = append(diskErrors, diskUsageErr.Error())
		return &defaultDiskData, diskErrors
	}

	// diskMetrics, diskErr := disk1.Get()
	// if diskErr != nil {
	// 	log.Fatalf("Unable to get disk metrics")
	// }

	// for _, p := range diskMetrics {
	// 	fmt.Println(p.Name, p.ReadsCompleted)
	// }

	// var a uint64 = 2e+12
	diskSlice := append(diskData, &DiskData{
		ReadSpeedBytes:  nil, // TODO: Implement
		WriteSpeedBytes: nil, // TODO: Implement
		TotalBytes:      &diskUsage.Total,
		FreeBytes:       &diskUsage.Free,
		UsagePercent:    RoundFloatPtr(diskUsage.UsedPercent/100, 4),
	})

	return &diskSlice, diskErrors
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
