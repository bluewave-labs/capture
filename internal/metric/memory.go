package metric

import (
	"github.com/shirou/gopsutil/v4/mem"
)

func CollectMemoryMetrics() (*MemoryData, []string) {
	var memErrors []string
	defaultMemoryData := &MemoryData{
		TotalBytes:     0,
		AvailableBytes: 0,
		UsedBytes:      0,
		UsagePercent:   RoundFloatPtr(0, 4),
	}
	vMem, vMemErr := mem.VirtualMemory()

	if vMemErr != nil {
		memErrors = append(memErrors, vMemErr.Error())
		return defaultMemoryData, memErrors
	}

	return &MemoryData{
		TotalBytes:     vMem.Total,
		AvailableBytes: vMem.Available,
		UsedBytes:      vMem.Used,
		UsagePercent:   RoundFloatPtr(vMem.UsedPercent/100, 4),
	}, memErrors
}
