package metric

type MetricsSlice []Metric

func (m MetricsSlice) isMetric() {}

type Metric interface {
	isMetric()
}

type ApiResponse struct {
	Data   Metric      `json:"data"`
	Errors []CustomErr `json:"errors"`
}

type AllMetrics struct {
	Cpu    CpuData      `json:"cpu"`
	Memory MemoryData   `json:"memory"`
	Disk   MetricsSlice `json:"disk"`
	Host   HostData     `json:"host"`
}

func (a AllMetrics) isMetric() {}

type CustomErr struct {
	Metric []string `json:"metric"`
	Error  string   `json:"err"`
}

type CpuData struct {
	PhysicalCore     int       `json:"physical_core"`     // Physical cores
	LogicalCore      int       `json:"logical_core"`      // Logical cores aka Threads
	Frequency        float64   `json:"frequency"`         // Frequency in mHz
	CurrentFrequency int       `json:"current_frequency"` // Current Frequency in mHz
	Temperature      []float32 `json:"temperature"`       // Temperature in Celsius (nil if not available)
	FreePercent      float64   `json:"free_percent"`      // Free percentage                               //* 1 - (Total - Idle / Total)
	UsagePercent     float64   `json:"usage_percent"`     // Usage percentage                              //* Total - Idle / Total
}

func (c CpuData) isMetric() {}

type MemoryData struct {
	TotalBytes     uint64   `json:"total_bytes"`     // Total space in bytes
	AvailableBytes uint64   `json:"available_bytes"` // Available space in bytes
	UsedBytes      uint64   `json:"used_bytes"`      // Used space in bytes      //* Total - Free - Buffers - Cached
	UsagePercent   *float64 `json:"usage_percent"`   // Usage Percent            //* (Used / Total) * 100.0
}

func (m MemoryData) isMetric() {}

type DiskData struct {
	ReadSpeedBytes  *uint64  `json:"read_speed_bytes"`  // TODO: Implement
	WriteSpeedBytes *uint64  `json:"write_speed_bytes"` // TODO: Implement
	TotalBytes      *uint64  `json:"total_bytes"`       // Total space of "/" in bytes
	FreeBytes       *uint64  `json:"free_bytes"`        // Free space of "/" in bytes
	UsagePercent    *float64 `json:"usage_percent"`     // Usage Percent of "/"
}

func (d DiskData) isMetric() {}

type HostData struct {
	Os            string `json:"os"`             // Operating System
	Platform      string `json:"platform"`       // Platform Name
	KernelVersion string `json:"kernel_version"` // Kernel Version
}

func (h HostData) isMetric() {}

func GetAllSystemMetrics() (AllMetrics, []CustomErr) {
	cpu, cpuErr := CollectCpuMetrics()
	memory, memErr := CollectMemoryMetrics()
	disk, diskErr := CollectDiskMetrics()
	host, hostErr := GetHostInformation()

	var errors []CustomErr

	if cpuErr != nil {
		errors = append(errors, cpuErr...)
	}

	if memErr != nil {
		errors = append(errors, memErr...)
	}

	if diskErr != nil {
		errors = append(errors, diskErr...)
	}

	if hostErr != nil {
		errors = append(errors, hostErr...)
	}

	return AllMetrics{
		Cpu:    *cpu,
		Memory: *memory,
		Disk:   disk,
		Host:   *host,
	}, errors
}
