package metric

type MetricsSlice []Metric

func (m MetricsSlice) isMetric() {}

type Metric interface {
	isMetric()
}

type SmartData struct {
	AvailableSpare           string `json:"available_spare"`
	AvailableSpareThreshold  string `json:"available_spare_threshold"`
	ControllerBusyTime       string `json:"controller_busy_time"`
	CriticalWarning          string `json:"critical_warning"`
	DataUnitsRead            string `json:"data_units_read"`
	DataUnitsWritten         string `json:"data_units_written"`
	HostReadCommands         string `json:"host_read_commands"`
	HostWriteCommands        string `json:"host_write_commands"`
	PercentageUsed           string `json:"percentage_used"`
	PowerCycles              string `json:"power_cycles"`
	PowerOnHours             string `json:"power_on_hours"`
	SmartOverallHealthResult string `json:"smart_overall_health_self_assessment_test_result"`
	Temperature              string `json:"temperature"`
	UnsafeShutdowns          string `json:"unsafe_shutdowns"`
}

func (s SmartData) isMetric() {}

type AllMetrics struct {
	CPU    CPUData      `json:"cpu"`
	Memory MemoryData   `json:"memory"`
	Disk   MetricsSlice `json:"disk"`
	Host   HostData     `json:"host"`
	Net    MetricsSlice `json:"net"`
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
	Device             string   `json:"device"`               // Device
	TotalBytes         *uint64  `json:"total_bytes"`          // Total space of device in bytes
	FreeBytes          *uint64  `json:"free_bytes"`           // Free space of device in bytes
	UsedBytes          *uint64  `json:"used_bytes"`           // Used space of device in bytes
	UsagePercent       *float64 `json:"usage_percent"`        // Usage percent of device
	TotalInodes        *uint64  `json:"total_inodes"`         // Total space of device in inodes
	FreeInodes         *uint64  `json:"free_inodes"`          // Free space of device in inodes
	UsedInodes         *uint64  `json:"used_inodes"`          // Used space of device in inodes
	InodesUsagePercent *float64 `json:"inodes_usage_percent"` // Usage percent of device in inodes
	ReadBytes          *uint64  `json:"read_bytes"`           // Amount of data read from the disk in bytes
	WriteBytes         *uint64  `json:"write_bytes"`          // Amount of data written to the disk in bytes
	ReadTime           *uint64  `json:"read_time"`            // Cumulative time spent performing read operations
	WriteTime          *uint64  `json:"write_time"`           // Cumulative time spent performing write operations
}

func (d DiskData) isMetric() {}

type HostData struct {
	Os            string `json:"os"`             // Operating System
	Platform      string `json:"platform"`       // Platform Name
	KernelVersion string `json:"kernel_version"` // Kernel Version
}

func (h HostData) isMetric() {}

type NetData struct {
	Name        string `json:"name"`         // Network Interface Name
	BytesSent   uint64 `json:"bytes_sent"`   // Bytes sent
	BytesRecv   uint64 `json:"bytes_recv"`   // Bytes received
	PacketsSent uint64 `json:"packets_sent"` // Packets sent
	PacketsRecv uint64 `json:"packets_recv"` // Packets received
	ErrIn       uint64 `json:"err_in"`       // Incoming packets with errors
	ErrOut      uint64 `json:"err_out"`      // Outgoing packets with errors
	DropIn      uint64 `json:"drop_in"`      // Incoming packets that were dropped
	DropOut     uint64 `json:"drop_out"`     // Outgoing packets that were dropped
	FIFOIn      uint64 `json:"fifo_in"`      // Incoming packets dropped due to full buffer
	FIFOOut     uint64 `json:"fifo_out"`     // Outgoing packets dropped due to full buffer
}

func (n NetData) isMetric() {}

func GetAllSystemMetrics() (AllMetrics, []CustomErr) {
	cpu, cpuErr := CollectCPUMetrics()
	memory, memErr := CollectMemoryMetrics()
	disk, diskErr := CollectDiskMetrics()
	host, hostErr := GetHostInformation()
	net, netErr := GetNetInformation()

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

	if netErr != nil {
		errors = append(errors, netErr...)
	}

	return AllMetrics{
		CPU:    *cpu,
		Memory: *memory,
		Disk:   disk,
		Host:   *host,
		Net:    net,
	}, errors
}
