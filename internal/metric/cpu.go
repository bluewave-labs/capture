package metric

import (
	"bluewave-uptime-agent/internal/sysfs"

	"github.com/shirou/gopsutil/v4/cpu"
)

type CpuData struct {
	PhysicalCore int      `json:"physical_core"` // Physical cores
	LogicalCore  int      `json:"logical_core"`  // Logical cores aka Threads
	Frequency    float64  `json:"frequency"`     // Frequency in mHz
	Temperature  *float32 `json:"temperauture"`  // Temperature in Celsius (nil if not available)
	FreePercent  float64  `json:"free_percent"`  // Free percentage  //* 1 - (Total - Idle / Total)
	UsagePercent float64  `json:"usage_percent"` // Usage percentage //* Total - Idle / Total
}

func CollectCpuMetrics() (*CpuData, error) {
	// Collect CPU Core Counts
	cpuPhysicalCoreCount, cpuPhysicalErr := cpu.Counts(false)
	cpuLogicalCoreCount, cpuLogicalErr := cpu.Counts(true)

	if cpuPhysicalErr != nil {
		return nil, cpuPhysicalErr
	}

	if cpuLogicalErr != nil {
		return nil, cpuLogicalErr
	}

	// Collect CPU Information (Frequency, Model, etc)
	cpuInformation, cpuInfoErr := cpu.Info()

	if cpuInfoErr != nil {
		return nil, cpuInfoErr
	}

	// Collect CPU Usage
	cpuTimes, cpuTimesErr := cpu.Times(false)

	if cpuTimesErr != nil {
		return nil, cpuTimesErr
	}

	// Calculate CPU Usage Percentage
	total := cpuTimes[0].User + cpuTimes[0].Nice + cpuTimes[0].System + cpuTimes[0].Idle + cpuTimes[0].Iowait + cpuTimes[0].Irq + cpuTimes[0].Softirq + cpuTimes[0].Steal + cpuTimes[0].Guest + cpuTimes[0].GuestNice
	cpuUsagePercent := (total - (cpuTimes[0].Idle + cpuTimes[0].Iowait)) / total

	// Collect CPU Temperature from sysfs
	cpuTemp, cpuTempErr := sysfs.CpuTemperature()

	if cpuTempErr != nil {
		return nil, cpuTempErr
	}

	return &CpuData{
		PhysicalCore: cpuPhysicalCoreCount,
		LogicalCore:  cpuLogicalCoreCount,
		Frequency:    cpuInformation[0].Mhz,
		Temperature:  cpuTemp,
		FreePercent:  1 - cpuUsagePercent,
		UsagePercent: cpuUsagePercent,
	}, nil
}
