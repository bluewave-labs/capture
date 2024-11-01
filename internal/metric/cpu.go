package metric

import (
	"bluewave-uptime-agent/internal/sysfs"
	"github.com/shirou/gopsutil/v4/cpu"
)

func CollectCpuMetrics() (*CpuData, []string) {
	// Collect CPU Core Counts
	cpuPhysicalCoreCount, cpuPhysicalErr := cpu.Counts(false)
	cpuLogicalCoreCount, cpuLogicalErr := cpu.Counts(true)

	var cpuErrors []string
	if cpuPhysicalErr != nil {
		cpuErrors = append(cpuErrors, cpuPhysicalErr.Error())
		cpuPhysicalCoreCount = 0
	}

	if cpuLogicalErr != nil {
		cpuErrors = append(cpuErrors, cpuLogicalErr.Error())
		cpuLogicalCoreCount = 0
	}

	// Collect CPU Information (Frequency, Model, etc)
	cpuInformation, cpuInfoErr := cpu.Info()
	var cpuFrequency float64
	if cpuInfoErr != nil {
		cpuErrors = append(cpuErrors, cpuInfoErr.Error())
		cpuFrequency = 0
	} else {
		cpuFrequency = cpuInformation[0].Mhz
	}

	// Collect CPU Usage
	cpuTimes, cpuTimesErr := cpu.Times(false)
	var cpuUsagePercent float64

	if cpuTimesErr != nil {
		cpuErrors = append(cpuErrors, cpuTimesErr.Error())
		cpuUsagePercent = 0
	} else {
		// Calculate CPU Usage Percentage
		total := cpuTimes[0].User + cpuTimes[0].Nice + cpuTimes[0].System + cpuTimes[0].Idle + cpuTimes[0].Iowait + cpuTimes[0].Irq + cpuTimes[0].Softirq + cpuTimes[0].Steal + cpuTimes[0].Guest + cpuTimes[0].GuestNice
		cpuUsagePercent = (total - (cpuTimes[0].Idle + cpuTimes[0].Iowait)) / total
	}

	// Collect CPU Temperature from sysfs
	// cpuTemp, cpuTempErr := sysfs.CpuTemperature()

	// if cpuTempErr != nil {
	// 	return nil, cpuTempErr
	// }

	cpuCurrentFrequency, cpuCurFreqErr := sysfs.CpuCurrentFrequency()
	if cpuCurFreqErr != nil {
		cpuErrors = append(cpuErrors, cpuCurFreqErr.Error())
		cpuCurrentFrequency = 0
	}

	return &CpuData{
		PhysicalCore:     cpuPhysicalCoreCount,
		LogicalCore:      cpuLogicalCoreCount,
		Frequency:        cpuFrequency,
		CurrentFrequency: cpuCurrentFrequency,
		Temperature:      nil,
		FreePercent:      1 - cpuUsagePercent,
		UsagePercent:     cpuUsagePercent,
	}, cpuErrors
}
