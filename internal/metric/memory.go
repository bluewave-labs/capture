package metric

import (
	"github.com/shirou/gopsutil/v4/mem"
)

func CollectMemoryMetrics() (*MemoryData, []string) {
	var memErrors []string
	vMem, vMemErr := mem.VirtualMemory()

	if vMemErr != nil {
		memErrors = append(memErrors, vMemErr.Error())
	}

	return &MemoryData{
		TotalBytes:     vMem.Total,
		AvailableBytes: vMem.Available,
		UsedBytes:      vMem.Used,
		UsagePercent:   RoundFloatPtr(vMem.UsedPercent/100, 4),
	}, memErrors
}
