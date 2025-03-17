package metric

type MetricsSlice []Metric

func (m MetricsSlice) isMetric() {}

type Metric interface {
	isMetric()
}

type APIResponse struct {
	Data   Metric      `json:"data"`
	Errors []CustomErr `json:"errors"`
}

type AllMetrics struct {
	CPU    CPUData      `json:"cpu"`
	Memory MemoryData   `json:"memory"`
	Disk   MetricsSlice `json:"disk"`
	Host   HostData     `json:"host"`
}

func (a AllMetrics) isMetric() {}

type CustomErr struct {
	Metric []string `json:"metric"`
	Error  string   `json:"err"`
}

type CPUData struct {
	PhysicalCore     int       `json:"physical_core"`     // Physical cores
	LogicalCore      int       `json:"logical_core"`      // Logical cores aka Threads
	Frequency        float64   `json:"frequency"`         // Frequency in mHz
	CurrentFrequency int       `json:"current_frequency"` // Current Frequency in mHz
	Temperature      []float32 `json:"temperature"`       // Temperature in Celsius (nil if not available)
	FreePercent      float64   `json:"free_percent"`      // Free percentage                               //* 1 - (Total - Idle / Total)
	UsagePercent     float64   `json:"usage_percent"`     // Usage percentage                              //* Total - Idle / Total
}

func (c CPUData) isMetric() {}

type MemoryData struct {
	TotalBytes     uint64   `json:"total_bytes"`     // Total space in bytes
	AvailableBytes uint64   `json:"available_bytes"` // Available space in bytes
	UsedBytes      uint64   `json:"used_bytes"`      // Used space in bytes      //* Total - Free - Buffers - Cached
	UsagePercent   *float64 `json:"usage_percent"`   // Usage Percent            //* (Used / Total) * 100.0
}

func (m MemoryData) isMetric() {}

type DiskData struct {
	Device       string   `json:"device"`        // Device
	TotalBytes   *uint64  `json:"total_bytes"`   // Total space of device in bytes
	FreeBytes    *uint64  `json:"free_bytes"`    // Free space of device in bytes
	ReadBytes    *uint64  `json:"read_bytes"`    // Amount of data read from the disk in bytes
	WriteBytes   *uint64  `json:"write_bytes"`   // Amount of data written to the disk in bytes
	ReadTime     *uint64  `json:"read_time"`     // Cumulative time spent performing read operations
	WriteTime    *uint64  `json:"write_time"`    // Cumulative time spent performing write operations
	UsagePercent *float64 `json:"usage_percent"` // Usage Percent of device
}

func (d DiskData) isMetric() {}

type HostData struct {
	Os            string `json:"os"`             // Operating System
	Platform      string `json:"platform"`       // Platform Name
	KernelVersion string `json:"kernel_version"` // Kernel Version
}

func (h HostData) isMetric() {}

func GetAllSystemMetrics() (AllMetrics, []CustomErr) {
	cpu, cpuErr := CollectCPUMetrics()
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
		CPU:    *cpu,
		Memory: *memory,
		Disk:   disk,
		Host:   *host,
	}, errors
}
