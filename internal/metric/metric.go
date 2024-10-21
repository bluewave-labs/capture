package metric

type ApiResponse struct {
	Cpu    CpuData     `json:"cpu"`
	Memory MemoryData  `json:"memory"`
	Disk   []*DiskData `json:"disk"`
	Host   HostData    `json:"host"`
}

func GetAllSystemMetrics() (*ApiResponse, error) {
	cpu, cpuErr := CollectCpuMetrics()
	memory, memErr := CollectMemoryMetrics()
	disk, diskErr := CollectDiskMetrics()
	host, hostErr := GetHostInformation()

	if cpuErr != nil {
		return nil, cpuErr
	}

	if memErr != nil {
		return nil, memErr
	}

	if diskErr != nil {
		return nil, diskErr
	}

	if hostErr != nil {
		return nil, hostErr
	}

	return &ApiResponse{
		Cpu:    *cpu,
		Memory: *memory,
		Disk:   disk,
		Host:   *host,
	}, nil
}
