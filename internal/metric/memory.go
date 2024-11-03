package metric

import (
	"github.com/shirou/gopsutil/v4/mem"
)

func CollectMemoryMetrics() (*MemoryData, []CustomErr) {
	var memErrors []CustomErr
	defaultMemoryData := &MemoryData{
		TotalBytes:     0,
		AvailableBytes: 0,
		UsedBytes:      0,
		UsagePercent:   RoundFloatPtr(0, 4),
	}
	vMem, vMemErr := mem.VirtualMemory()

	if vMemErr != nil {
		memErrors = append(memErrors, CustomErr{
			Metric: []string{"memory.total_bytes", "memory.available_bytes", "memory.used_bytes", "memory.usage_percent"},
			Error:  vMemErr.Error(),
		})
		return defaultMemoryData, memErrors
	}

	return &MemoryData{
		TotalBytes:     vMem.Total,
		AvailableBytes: vMem.Available,
		UsedBytes:      vMem.Used,
		UsagePercent:   RoundFloatPtr(vMem.UsedPercent/100, 4),
	}, memErrors
}
