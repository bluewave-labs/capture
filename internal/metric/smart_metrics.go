package metric

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
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

// smartctlDeviceTypes are the -d types from smartctl(8). A ":" suffix in a SMART_DEVICES
// entry is only read as a type when it starts with one of these, because device paths
// such as /dev/disk/by-path/pci-0000:00:17.0-ata-1 contain colons themselves.
var smartctlDeviceTypes = map[string]bool{
	"3ware": true, "aacraid": true, "areca": true, "ata": true, "auto": true, "cciss": true,
	"hpt": true, "intelliprop": true, "jmb39x": true, "jmb39x-q": true, "jmb39x-q2": true,
	"jms56x": true, "marvell": true, "megaraid": true, "nvme": true, "sat": true, "scsi": true,
	"sntasmedia": true, "sntjmicron": true, "sntrealtek": true, "test": true, "usbasm1352r": true,
	"usbcypress": true, "usbjmicron": true, "usbprolific": true, "usbsunplus": true,
}

// splitDeviceEntry splits a "path" or "path:type" entry at its last colon when the
// part after it names a smartctl device type
func splitDeviceEntry(entry string) smartDevice {
	if idx := strings.LastIndex(entry, ":"); idx != -1 {
		deviceType := strings.TrimSpace(entry[idx+1:])
		base, _, _ := strings.Cut(deviceType, "+")
		if smartctlDeviceTypes[base] {
			return smartDevice{Path: strings.TrimSpace(entry[:idx]), Type: deviceType}
		}
	}
	return smartDevice{Path: entry}
}

// parseSmartDevices parses the SMART_DEVICES value, a comma-separated list of
// "path" or "path:type" entries, e.g. "/dev/sda:sat,/dev/nvme0".
// Types that contain commas, such as "megaraid,0", are kept together: a piece
// that does not start with "/" continues the previous entry's type.
func parseSmartDevices(value string) ([]smartDevice, error) {
	var devices []smartDevice
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.HasPrefix(entry, "/") {
			devices = append(devices, splitDeviceEntry(entry))
			continue
		}

		// Paths always start with "/", which also keeps entries from being read as smartctl options.
		// A continuation with ":" is most likely a path missing its leading "/", e.g. "/dev/sdb:sat,sdc:sat".
		if len(devices) == 0 || devices[len(devices)-1].Type == "" || strings.Contains(entry, ":") {
			return nil, fmt.Errorf("invalid SMART_DEVICES entry %q: device paths must start with \"/\"", entry)
		}
		devices[len(devices)-1].Type += "," + entry
	}
	return devices, nil
}

// smartDeviceOverrides holds the devices from SMART_DEVICES, set once at startup by ConfigureSmartDevices
var smartDeviceOverrides []smartDevice

// ConfigureSmartDevices parses the SMART_DEVICES value so configuration errors are reported at startup
func ConfigureSmartDevices(value string) error {
	devices, err := parseSmartDevices(value)
	if err != nil {
		return err
	}
	for _, device := range devices {
		if device.Type == "" {
			log.Printf("SMART_DEVICES: querying %s with autodetected type", device.Path)
		} else {
			log.Printf("SMART_DEVICES: querying %s with type %s", device.Path, device.Type)
		}
	}
	smartDeviceOverrides = devices
	return nil
}

// resolveDevicePath follows symlinks such as /dev/disk/by-id/... so the same disk is not
// queried twice under two names. Paths that cannot be resolved are returned unchanged.
func resolveDevicePath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return path
}

// mergeDevices applies overrides to the scanned devices. Every scanned entry whose
// resolved path matches an override is replaced by the override entries, so a path
// behind a RAID controller can be listed once per disk.
func mergeDevices(scanned, overrides []smartDevice, resolve func(string) string) []smartDevice {
	overridden := make(map[string]bool, len(overrides))
	for _, override := range overrides {
		overridden[resolve(override.Path)] = true
	}

	merged := make([]smartDevice, 0, len(scanned)+len(overrides))
	for _, device := range scanned {
		if !overridden[resolve(device.Path)] {
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

// standbyMessage matches the message smartctl prints when --nocheck skips a sleeping disk,
// e.g. "Device is in STANDBY mode, exit(0)"
var standbyMessage = regexp.MustCompile(`Device is in \S+ mode`)

// parseSmartctlJSON converts `smartctl --json` output to SmartData.
// It returns nil when the device is in standby, and an error when smartctl reported
// no SMART data for another reason, e.g. SMART unsupported behind a USB bridge.
func parseSmartctlJSON(output []byte) (*SmartData, error) {
	var parsed smartctlOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse smartctl JSON output (smartctl 7.0 or newer is required): %w", err)
	}

	if parsed.SmartStatus == nil && parsed.Temperature == nil && parsed.PowerOnTime == nil && parsed.NvmeLog == nil && parsed.AtaAttributes == nil {
		messages := smartctlMessages(output)
		if standbyMessage.MatchString(messages) {
			return nil, nil
		}
		if messages == "" {
			messages = "SMART may be unsupported or disabled on this device"
		}
		return nil, fmt.Errorf("no SMART data reported: %s", messages)
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

// smartctlFailureReason describes why smartctl could not read a device
func smartctlFailureReason(stdout, stderr []byte, err error) string {
	combined := string(stdout) + string(stderr)
	switch {
	case strings.Contains(strings.ToLower(combined), "unrecognized option"):
		// smartctl before 7.0 has no --json option
		return "smartctl 7.0 or newer is required"
	case strings.Contains(combined, "Permission denied"):
		return "permission denied (try running as root)"
	}

	var reasons []string
	if messages := smartctlMessages(stdout); messages != "" {
		reasons = append(reasons, messages)
	}
	if text := strings.TrimSpace(string(stderr)); text != "" {
		reasons = append(reasons, text)
	}
	if len(reasons) == 0 {
		return err.Error()
	}
	return strings.Join(reasons, "; ")
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
			return nil, []CustomErr{{
				Metric: []string{"smartctl"},
				Error:  fmt.Sprintf("smartctl failed for %s: %s", device.Path, smartctlFailureReason(out, exitErr.Stderr, err)),
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

// smartDataFields lists the fields of SmartData so results from several drives can be combined
func smartDataFields(data *SmartData) []*string {
	return []*string{
		&data.AvailableSpare, &data.AvailableSpareThreshold, &data.ControllerBusyTime, &data.CriticalWarning,
		&data.DataUnitsRead, &data.DataUnitsWritten, &data.HostReadCommands, &data.HostWriteCommands,
		&data.PercentageUsed, &data.PowerCycles, &data.PowerOnHours, &data.SmartOverallHealthResult,
		&data.Temperature, &data.UnsafeShutdowns,
	}
}

// mergeSmartData combines per-device results into the single SmartData the API returns
// until per-drive output exists. NVMe results come first so their values are kept for
// fields both drive types report, other drives only fill fields that are still empty,
// and a FAILED health result from any drive is reported.
func mergeSmartData(results []*SmartData) SmartData {
	ordered := make([]*SmartData, 0, len(results))
	for _, result := range results {
		// PercentageUsed is only filled from the NVMe health log
		if result.PercentageUsed != "" {
			ordered = append(ordered, result)
		}
	}
	for _, result := range results {
		if result.PercentageUsed == "" {
			ordered = append(ordered, result)
		}
	}

	var merged SmartData
	mergedFields := smartDataFields(&merged)
	for _, result := range ordered {
		for i, field := range smartDataFields(result) {
			if *mergedFields[i] == "" {
				*mergedFields[i] = *field
			}
		}
		if result.SmartOverallHealthResult == "FAILED" {
			merged.SmartOverallHealthResult = "FAILED"
		}
	}
	return merged
}

// GetSmartMetrics retrieves the SMART metrics from all available devices.
// Devices from `smartctl --scan` can be extended or given an explicit type with SMART_DEVICES.
func GetSmartMetrics() (SmartData, []CustomErr) {
	var smartCtlrErrs []CustomErr

	// Check if smartctl is installed
	if err := checkSmartctlInstalled(); err != nil {
		smartCtlrErrs = append(smartCtlrErrs, CustomErr{
			Metric: []string{"smartctl"},
			Error:  err.Error(),
		})
		return SmartData{}, smartCtlrErrs
	}
	// Scan for devices
	devices, devicesErr := scanDevices()
	if devicesErr != nil {
		smartCtlrErrs = append(smartCtlrErrs, CustomErr{
			Metric: []string{"smart"},
			Error:  devicesErr.Error(),
		})
		// Without configured devices there is nothing left to query
		if len(smartDeviceOverrides) == 0 {
			return SmartData{}, smartCtlrErrs
		}
	}
	devices = mergeDevices(devices, smartDeviceOverrides, resolveDevicePath)

	// Iterate over devices and collect metrics
	var results []*SmartData
	for _, device := range devices {
		metric, metricErr := getMetrics(device)

		if metric != nil {
			results = append(results, metric)
		}

		if len(metricErr) > 0 {
			smartCtlrErrs = append(smartCtlrErrs, metricErr...)
		}
	}

	return mergeSmartData(results), smartCtlrErrs
}
