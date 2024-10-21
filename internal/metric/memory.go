package metric

import (
	"github.com/shirou/gopsutil/v4/mem"
)

type MemoryData struct {
	TotalBytes     uint64   `json:"total_bytes"`     // Total space in bytes
	AvailableBytes uint64   `json:"available_bytes"` // Available space in bytes
	UsedBytes      uint64   `json:"used_bytes"`      // Used space in bytes      //* Total - Free - Buffers - Cached
	UsagePercent   *float64 `json:"usage_percent"`   // Usage Percent            //* (Used / Total) * 100.0
}

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
