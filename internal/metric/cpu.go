package metric

import (
	"time"

	"github.com/bluewave-labs/capture/internal/sysfs"
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
	var cpuUsagePercent float64

	// Percent calculates the percentage of cpu used either per CPU or combined.
	// If an interval of 0 is given it will compare the current cpu times against the last call
	// Returns one value per cpu, or a single value if percpu is set to false.
	cpuPercents, cpuPercentsErr := cpu.Percent(time.Second, false)
	if cpuPercentsErr != nil {
		cpuErrors = append(cpuErrors, CustomErr{
			Metric: []string{"cpu.usage_percent"},
			Error:  cpuPercentsErr.Error(),
		})
		cpuUsagePercent = 0
	} else {
		cpuUsagePercent = cpuPercents[0] / 100.0
	}

	// Collect CPU Temperature from sysfs
	cpuTemp, cpuTempErr := sysfs.CpuTemperature()

	if cpuTempErr != nil {
		cpuErrors = append(cpuErrors, CustomErr{
			Metric: []string{"cpu.temperature"},
			Error:  cpuTempErr.Error(),
		})
		cpuTemp = nil
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
		FreePercent:      *RoundFloatPtr(1-cpuUsagePercent, 4),
		UsagePercent:     *RoundFloatPtr(cpuUsagePercent, 4),
	}, cpuErrors
}
