package metric

import (
	"github.com/bluewave-labs/bluewave-uptime-agent/internal/sysfs"
	"github.com/shirou/gopsutil/v4/cpu"
)

func CollectCpuMetrics() (*CpuData, []CustomErr) {
	// Collect CPU Core Counts
	cpuPhysicalCoreCount, cpuPhysicalErr := cpu.Counts(false)
	cpuLogicalCoreCount, cpuLogicalErr := cpu.Counts(true)

	var cpuErrors []CustomErr
	if cpuPhysicalErr != nil {
		cpuErrors = append(cpuErrors, CustomErr{
			Metric: []string{"cpu.physical_core"},
			Error:  cpuPhysicalErr.Error(),
		})
		cpuPhysicalCoreCount = 0
	}

	if cpuLogicalErr != nil {
		cpuErrors = append(cpuErrors, CustomErr{
			Metric: []string{"cpu.logical_core"},
			Error:  cpuLogicalErr.Error(),
		})
		cpuLogicalCoreCount = 0
	}

	// Collect CPU Information (Frequency, Model, etc)
	cpuInformation, cpuInfoErr := cpu.Info()
	var cpuFrequency float64
	if cpuInfoErr != nil {
		cpuErrors = append(cpuErrors, CustomErr{
			Metric: []string{"cpu.frequency"},
			Error:  cpuInfoErr.Error(),
		})
		cpuFrequency = 0
	} else {
		cpuFrequency = cpuInformation[0].Mhz
	}

	// Collect CPU Usage
	cpuTimes, cpuTimesErr := cpu.Times(false)
	var cpuUsagePercent float64

	if cpuTimesErr != nil {
		cpuErrors = append(cpuErrors, CustomErr{
			Metric: []string{"cpu.usage_percent"},
			Error:  cpuTimesErr.Error(),
		})
		cpuUsagePercent = 0
	} else {
		// Calculate CPU Usage Percentage
		total := cpuTimes[0].User + cpuTimes[0].Nice + cpuTimes[0].System + cpuTimes[0].Idle + cpuTimes[0].Iowait + cpuTimes[0].Irq + cpuTimes[0].Softirq + cpuTimes[0].Steal + cpuTimes[0].Guest + cpuTimes[0].GuestNice
		cpuUsagePercent = (total - (cpuTimes[0].Idle + cpuTimes[0].Iowait)) / total
	}

	// Collect CPU Temperature from sysfs
	cpuTemp, cpuTempErr := sysfs.CpuTemperature()

	if cpuTempErr != nil {
		cpuErrors = append(cpuErrors, CustomErr{
			Metric: []string{"cpu.temperature"},
			Error:  cpuTempErr.Error(),
		})
	}

	cpuCurrentFrequency, cpuCurFreqErr := sysfs.CpuCurrentFrequency()
	if cpuCurFreqErr != nil {
		cpuErrors = append(cpuErrors, CustomErr{
			Metric: []string{"cpu.current_frequency"},
			Error:  cpuCurFreqErr.Error(),
		})
		cpuCurrentFrequency = 0
	}

	return &CpuData{
		PhysicalCore:     cpuPhysicalCoreCount,
		LogicalCore:      cpuLogicalCoreCount,
		Frequency:        cpuFrequency,
		CurrentFrequency: cpuCurrentFrequency,
		Temperature:      cpuTemp,
		FreePercent:      1 - cpuUsagePercent,
		UsagePercent:     cpuUsagePercent,
	}, cpuErrors
}
