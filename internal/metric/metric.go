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

type SmartMetric struct {
	Data   SmartData    `json:"data"`   // The SMART data
	Errors []CustomErr  `json:"errors"` // Any errors encountered
}

func (s SmartMetric) isMetric() {}

type SmartData struct {
	AvailableSpare                string `json:"available_spare"`
	AvailableSpareThreshold       string `json:"available_spare_threshold"`
	ControllerBusyTime            string `json:"controller_busy_time"`
	CriticalWarning               string `json:"critical_warning"`
	DataUnitsRead                 string `json:"data_units_read"`
	DataUnitsWritten              string `json:"data_units_written"`
	ErrorInformationLogEntries    string `json:"error_information_log_entries"`
	HostReadCommands              string `json:"host_read_commands"`
	HostWriteCommands             string `json:"host_write_commands"`
	MediaAndDataIntegrityErrors  string `json:"media_and_data_integrity_errors"`
	PercentageUsed                string `json:"percentage_used"`
	PowerCycles                   string `json:"power_cycles"`
	PowerOnHours                  string `json:"power_on_hours"`
	Read1EntriesFromErrorLogFailed string `json:"read_1_entries_from_error_information_log_failed"`
	SmartOverallHealthResult      string `json:"smart_overall-health_self-assessment_test_result"`
	Temperature                   string `json:"temperature"`
	UnsafeShutdowns               string `json:"unsafe_shutdowns"`
}

func (s SmartData) isMetric() {}

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