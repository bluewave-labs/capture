package metric

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// smartDevice is a device to query with smartctl. An empty Type lets smartctl autodetect it.
type smartDevice struct {
	Path string
	Type string
}

// smartctl exit status bits that mean no usable data was produced.
// Higher bits describe disk health and do not prevent reading the output.
// See the RETURN VALUES section of smartctl(8).
const smartctlFatalExitBits = 0b11

// ATA attribute IDs used to fill SmartData for SATA drives
const (
	ataAttrUnexpectedPowerLoss = 174
	ataAttrPowerOffRetract     = 192
)

// smartctlOutput holds the parts of `smartctl --json` output that Capture reports
type smartctlOutput struct {
	Smartctl struct {
		Messages []struct {
			String string `json:"string"`
		} `json:"messages"`
	} `json:"smartctl"`
	SmartStatus *struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	Temperature *struct {
		Current int `json:"current"`
	} `json:"temperature"`
	PowerOnTime *struct {
		Hours int64 `json:"hours"`
	} `json:"power_on_time"`
	PowerCycleCount *int64 `json:"power_cycle_count"`
	NvmeLog         *struct {
		CriticalWarning         int   `json:"critical_warning"`
		AvailableSpare          int   `json:"available_spare"`
		AvailableSpareThreshold int   `json:"available_spare_threshold"`
		PercentageUsed          int   `json:"percentage_used"`
		DataUnitsRead           int64 `json:"data_units_read"`
		DataUnitsWritten        int64 `json:"data_units_written"`
		HostReads               int64 `json:"host_reads"`
		HostWrites              int64 `json:"host_writes"`
		ControllerBusyTime      int64 `json:"controller_busy_time"`
		PowerCycles             int64 `json:"power_cycles"`
		PowerOnHours            int64 `json:"power_on_hours"`
		UnsafeShutdowns         int64 `json:"unsafe_shutdowns"`
	} `json:"nvme_smart_health_information_log"`
	AtaAttributes *struct {
		Table []struct {
			ID  int `json:"id"`
			Raw struct {
				Value int64 `json:"value"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
}

// Check if smartctl is installed
func checkSmartctlInstalled() error {
	_, err := exec.LookPath("smartctl")
	if err != nil {
		return fmt.Errorf("smartctl is not installed")
	}
	return nil
}

// scanDevices returns a list of disks using `smartctl --scan`
func scanDevices() ([]smartDevice, error) {
	out, err := exec.Command("smartctl", "--scan").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to scan devices: %w", err)
	}
	return parseScanOutput(string(out)), nil
}

// parseScanOutput parses `smartctl --scan` lines such as "/dev/sda -d sat # /dev/sda [SAT], ATA device".
// The generic "scsi" type is dropped because SATA drives are often reported as scsi
// but only answer to sat, while smartctl's autodetection picks the right type for both.
func parseScanOutput(output string) []smartDevice {
	var devices []smartDevice
	for _, line := range strings.Split(output, "\n") {
		if idx := strings.Index(line, "#"); idx != -1 {
			line = line[:idx]
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		device := smartDevice{Path: fields[0]}
		if len(fields) >= 3 && fields[1] == "-d" && fields[2] != "scsi" {
			device.Type = fields[2]
		}
		devices = append(devices, device)
	}
	return devices
}

// parseSmartDevices parses the SMART_DEVICES value, a comma-separated list of
// "path" or "path:type" entries, e.g. "/dev/sda:sat,/dev/nvme0".
// Types that contain commas, such as "megaraid,0", are kept together: a piece
// that does not start with "/" continues the previous entry's type.
func parseSmartDevices(value string) []smartDevice {
	var devices []smartDevice
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.HasPrefix(entry, "/") {
			// Paths always start with "/", which also keeps entries from being read as smartctl options
			if len(devices) > 0 {
				devices[len(devices)-1].Type += "," + entry
			}
			continue
		}
		path, deviceType, _ := strings.Cut(entry, ":")
		devices = append(devices, smartDevice{Path: strings.TrimSpace(path), Type: strings.TrimSpace(deviceType)})
	}
	return devices
}

// mergeDevices applies overrides to the scanned devices. Every scanned entry whose
// path appears in overrides is replaced by the override entries for that path, so
// a path behind a RAID controller can be listed once per disk.
func mergeDevices(scanned, overrides []smartDevice) []smartDevice {
	overridden := make(map[string]bool, len(overrides))
	for _, override := range overrides {
		overridden[override.Path] = true
	}

	merged := make([]smartDevice, 0, len(scanned)+len(overrides))
	for _, device := range scanned {
		if !overridden[device.Path] {
			merged = append(merged, device)
		}
	}
	return append(merged, overrides...)
}

// smartctlArgs builds the smartctl arguments for a device.
// Devices in standby are skipped with exit status 0 so disks are not spun up.
func smartctlArgs(device smartDevice) []string {
	args := []string{"--json", "--xall", "--nocheck=standby,0"}
	if device.Type != "" {
		args = append(args, "-d", device.Type)
	}
	return append(args, device.Path)
}

// parseSmartctlJSON converts `smartctl --json` output to SmartData.
// It returns nil when the output holds no SMART data, e.g. for a device in standby.
func parseSmartctlJSON(output []byte) (*SmartData, error) {
	var parsed smartctlOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse smartctl JSON output (smartctl 7.0 or newer is required): %w", err)
	}

	if parsed.SmartStatus == nil && parsed.Temperature == nil && parsed.PowerOnTime == nil && parsed.NvmeLog == nil && parsed.AtaAttributes == nil {
		return nil, nil
	}

	data := SmartData{}
	if parsed.SmartStatus != nil {
		data.SmartOverallHealthResult = "FAILED"
		if parsed.SmartStatus.Passed {
			data.SmartOverallHealthResult = "PASSED"
		}
	}
	if parsed.Temperature != nil {
		data.Temperature = fmt.Sprintf("%d Celsius", parsed.Temperature.Current)
	}
	if parsed.PowerOnTime != nil {
		data.PowerOnHours = strconv.FormatInt(parsed.PowerOnTime.Hours, 10)
	}
	if parsed.PowerCycleCount != nil {
		data.PowerCycles = strconv.FormatInt(*parsed.PowerCycleCount, 10)
	}

	if nvme := parsed.NvmeLog; nvme != nil {
		data.CriticalWarning = fmt.Sprintf("0x%02x", nvme.CriticalWarning)
		data.AvailableSpare = fmt.Sprintf("%d%%", nvme.AvailableSpare)
		data.AvailableSpareThreshold = fmt.Sprintf("%d%%", nvme.AvailableSpareThreshold)
		data.PercentageUsed = fmt.Sprintf("%d%%", nvme.PercentageUsed)
		data.DataUnitsRead = strconv.FormatInt(nvme.DataUnitsRead, 10)
		data.DataUnitsWritten = strconv.FormatInt(nvme.DataUnitsWritten, 10)
		data.HostReadCommands = strconv.FormatInt(nvme.HostReads, 10)
		data.HostWriteCommands = strconv.FormatInt(nvme.HostWrites, 10)
		data.ControllerBusyTime = strconv.FormatInt(nvme.ControllerBusyTime, 10)
		data.PowerCycles = strconv.FormatInt(nvme.PowerCycles, 10)
		data.PowerOnHours = strconv.FormatInt(nvme.PowerOnHours, 10)
		data.UnsafeShutdowns = strconv.FormatInt(nvme.UnsafeShutdowns, 10)
	}

	if ata := parsed.AtaAttributes; ata != nil {
		rawValues := make(map[int]int64, len(ata.Table))
		for _, attr := range ata.Table {
			rawValues[attr.ID] = attr.Raw.Value
		}
		if value, ok := rawValues[ataAttrUnexpectedPowerLoss]; ok {
			data.UnsafeShutdowns = strconv.FormatInt(value, 10)
		} else if value, ok := rawValues[ataAttrPowerOffRetract]; ok {
			data.UnsafeShutdowns = strconv.FormatInt(value, 10)
		}
	}

	return &data, nil
}

// smartctlMessages joins the messages smartctl reported in its JSON output
func smartctlMessages(output []byte) string {
	var parsed smartctlOutput
	if json.Unmarshal(output, &parsed) != nil {
		return strings.TrimSpace(string(output))
	}
	messages := make([]string, 0, len(parsed.Smartctl.Messages))
	for _, message := range parsed.Smartctl.Messages {
		messages = append(messages, message.String)
	}
	return strings.Join(messages, "; ")
}

func getMetrics(device smartDevice) (*SmartData, []CustomErr) {
	// #nosec G204 -- no shell is involved; paths come from smartctl --scan or SMART_DEVICES and always start with "/"
	out, err := exec.Command("smartctl", smartctlArgs(device)...).Output()

	// smartctl returns a bitmask; only the low bits mean the device could not be read
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode()&smartctlFatalExitBits == 0 {
			err = nil
		} else {
			reason := smartctlMessages(out)
			if strings.Contains(reason, "Permission denied") {
				reason = "permission denied (try running as root)"
			}
			if reason == "" {
				reason = err.Error()
			}
			return nil, []CustomErr{{
				Metric: []string{"smartctl"},
				Error:  fmt.Sprintf("smartctl failed for %s: %s", device.Path, reason),
			}}
		}
	}

	// If there's an error executing the command, return the error
	if err != nil {
		return nil, []CustomErr{{
			Metric: []string{"smartctl"},
			Error:  fmt.Sprintf("smartctl failed for %s: %v", device.Path, err),
		}}
	}

	data, parseErr := parseSmartctlJSON(out)
	if parseErr != nil {
		return nil, []CustomErr{{
			Metric: []string{"smartctl"},
			Error:  fmt.Sprintf("smartctl failed for %s: %v", device.Path, parseErr),
		}}
	}
	return data, nil
}

// GetSmartMetrics retrieves the SMART metrics from all available devices.
// Devices from `smartctl --scan` can be extended or given an explicit type with SMART_DEVICES.
func GetSmartMetrics() (SmartData, []CustomErr) {
	var metrics SmartData
	var smartCtlrErrs []CustomErr

	// Check if smartctl is installed
	if err := checkSmartctlInstalled(); err != nil {
		smartCtlrErrs = append(smartCtlrErrs, CustomErr{
			Metric: []string{"smartctl"},
			Error:  err.Error(),
		})
		return metrics, smartCtlrErrs
	}
	// Scan for devices
	overrides := parseSmartDevices(os.Getenv("SMART_DEVICES"))
	devices, devicesErr := scanDevices()
	if devicesErr != nil {
		smartCtlrErrs = append(smartCtlrErrs, CustomErr{
			Metric: []string{"smart"},
			Error:  devicesErr.Error(),
		})
		// Without configured devices there is nothing left to query
		if len(overrides) == 0 {
			return metrics, smartCtlrErrs
		}
	}
	devices = mergeDevices(devices, overrides)

	// Iterate over devices and collect metrics
	for _, device := range devices {
		metric, metricErr := getMetrics(device)

		if metric != nil {
			metrics = *metric
		}

		if len(metricErr) > 0 {
			smartCtlrErrs = append(smartCtlrErrs, metricErr...)
		}
	}

	return metrics, smartCtlrErrs
}
