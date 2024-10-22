package metric

import (
	"github.com/shirou/gopsutil/v4/mem"
)

func CollectMemoryMetrics() (*MemoryData, error) {
	vMem, vMemErr := mem.VirtualMemory()

	if vMemErr != nil {
		return nil, vMemErr
	}

	return &MemoryData{
		TotalBytes:     vMem.Total,
		AvailableBytes: vMem.Available,
		UsedBytes:      vMem.Used,
		UsagePercent:   RoundFloatPtr(vMem.UsedPercent/100, 4),
	}, nil
}
