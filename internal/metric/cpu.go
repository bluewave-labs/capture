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
	FreePercent  *float64 `json:"free_percent"`  // TODO: Implement
	UsagePercent *float64 `json:"usage_percent"` // TODO: Implement
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
		FreePercent:  nil, //TODO: Implement
		UsagePercent: nil, //TODO: Implement
	}, nil
}
