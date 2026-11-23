package metric

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
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

// smartctlTimeout bounds a single smartctl run so one hung drive, typically behind
// a USB bridge, cannot block the whole request. Variables so tests can shorten them.
var (
	smartctlTimeout   = 20 * time.Second
	smartctlWaitDelay = 2 * time.Second
	smartctlCommand   = "smartctl"
)

// errSmartctlTimeout is returned when smartctl does not finish within smartctlTimeout
var errSmartctlTimeout = errors.New("smartctl timed out")

// runSmartctl runs smartctl with a timeout. When the timeout expires the process is
// killed; WaitDelay stops a process that does not exit from holding the request.
func runSmartctl(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), smartctlTimeout)
	defer cancel()

	// #nosec G204 -- no shell is involved; paths come from smartctl --scan or from SMART_DEVICES, whose entries must start with "/"
	cmd := exec.CommandContext(ctx, smartctlCommand, args...)
	cmd.WaitDelay = smartctlWaitDelay
	out, err := cmd.Output()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return out, fmt.Errorf("%w after %s", errSmartctlTimeout, smartctlTimeout)
	}
	return out, err
}

// ATA attribute IDs used to fill SmartData for SATA drives
const (
	ataAttrReallocatedSectors   = 5
	ataAttrUnexpectedPowerLoss  = 174
	ataAttrPowerOffRetract      = 192
	ataAttrPendingSectors       = 197
	ataAttrOfflineUncorrectable = 198
)

// Values of SmartData.Status
const (
	SmartStatusOK          = "ok"
	SmartStatusStandby     = "standby"
	SmartStatusUnsupported = "unsupported"
	SmartStatusError       = "error"
)

// smartctlWarningBits maps smartctl exit status bits to warnings about the drive's health.
// Bit 2 (a SMART command failed) is left out: some drives, such as Apple SSDs, set it on
// every run because they do not support every log, which says nothing about their health.
// See the RETURN VALUES section of smartctl(8).
var smartctlWarningBits = []struct {
	bit     int
	warning string
}{
	{3, "disk_failing"},
	{4, "prefail_attribute_at_threshold"},
	{5, "attribute_was_at_threshold"},
	{6, "error_log_has_errors"},
	{7, "self_test_log_has_errors"},
}

// smartctlConcurrency caps how many drives are read at once
const smartctlConcurrency = 4

// smartctlOutput holds the parts of `smartctl --json` output that Capture reports
type smartctlOutput struct {
	Smartctl struct {
		Messages []struct {
			String string `json:"string"`
		} `json:"messages"`
	} `json:"smartctl"`
	Device struct {
		Type     string `json:"type"`
		Protocol string `json:"protocol"`
	} `json:"device"`
	ModelName       string `json:"model_name"`
	SerialNumber    string `json:"serial_number"`
	FirmwareVersion string `json:"firmware_version"`
	UserCapacity    *struct {
		Bytes uint64 `json:"bytes"`
	} `json:"user_capacity"`
	ScsiGrownDefectList *int64 `json:"scsi_grown_defect_list"`
	SmartStatus         *struct {
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
		MediaErrors             int64 `json:"media_errors"`
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
	_, err := exec.LookPath(smartctlCommand)
	if err != nil {
		return fmt.Errorf("smartctl is not installed")
	}
	return nil
}

// scanDevices returns a list of disks using `smartctl --scan`
func scanDevices() ([]smartDevice, error) {
	out, err := runSmartctl("--scan")
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

// parseSmartctlJSON converts `smartctl --json` output to SmartData. Status is standby when
// smartctl skipped a sleeping disk, unsupported when it reported no SMART data for another
// reason (e.g. a virtual disk or a USB bridge without passthrough), and ok otherwise.
func parseSmartctlJSON(output []byte) (*SmartData, error) {
	var parsed smartctlOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse smartctl JSON output (smartctl 7.0 or newer is required): %w", err)
	}

	data := SmartData{
		Type:            parsed.Device.Type,
		Protocol:        parsed.Device.Protocol,
		Status:          SmartStatusOK,
		ModelName:       parsed.ModelName,
		SerialNumber:    parsed.SerialNumber,
		FirmwareVersion: parsed.FirmwareVersion,
		Warnings:        []string{},
	}
	if parsed.UserCapacity != nil {
		capacity := parsed.UserCapacity.Bytes
		data.CapacityBytes = &capacity
	}

	if parsed.SmartStatus == nil && parsed.Temperature == nil && parsed.PowerOnTime == nil && parsed.NvmeLog == nil && parsed.AtaAttributes == nil {
		data.Status = SmartStatusUnsupported
		if standbyMessage.MatchString(smartctlMessages(output)) {
			data.Status = SmartStatusStandby
		}
		return &data, nil
	}

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
	if parsed.ScsiGrownDefectList != nil {
		data.ReallocatedSectors = parsed.ScsiGrownDefectList
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
		mediaErrors := nvme.MediaErrors
		data.MediaErrors = &mediaErrors
	}

	if ata := parsed.AtaAttributes; ata != nil {
		rawValues := make(map[int]int64, len(ata.Table))
		for _, attr := range ata.Table {
			rawValues[attr.ID] = attr.Raw.Value
		}
		// Attributes a drive does not report stay nil, so a missing reading is not shown as 0
		rawValue := func(id int) *int64 {
			if value, ok := rawValues[id]; ok {
				return &value
			}
			return nil
		}
		data.ReallocatedSectors = rawValue(ataAttrReallocatedSectors)
		data.PendingSectors = rawValue(ataAttrPendingSectors)
		data.OfflineUncorrectable = rawValue(ataAttrOfflineUncorrectable)

		if value, ok := rawValues[ataAttrUnexpectedPowerLoss]; ok {
			data.UnsafeShutdowns = strconv.FormatInt(value, 10)
		} else if value, ok := rawValues[ataAttrPowerOffRetract]; ok {
			data.UnsafeShutdowns = strconv.FormatInt(value, 10)
		}
	}

	return &data, nil
}

// smartctlWarnings converts smartctl's exit status to health warnings
func smartctlWarnings(exitCode int) []string {
	warnings := []string{}
	if exitCode < 0 {
		return warnings
	}
	for _, warningBit := range smartctlWarningBits {
		if exitCode&(1<<warningBit.bit) != 0 {
			warnings = append(warnings, warningBit.warning)
		}
	}
	return warnings
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

// getMetrics reads one device. It always returns the device, with status error when it could not be read.
func getMetrics(device smartDevice) (*SmartData, []CustomErr) {
	failed := func(reason string) (*SmartData, []CustomErr) {
		return &SmartData{Device: device.Path, Type: device.Type, Status: SmartStatusError, Warnings: []string{}},
			[]CustomErr{{
				Metric: []string{"smartctl"},
				Error:  fmt.Sprintf("smartctl failed for %s: %s", device.Path, reason),
			}}
	}

	out, err := runSmartctl(smartctlArgs(device)...)

	// smartctl returns a bitmask; only the low bits mean the device could not be read
	exitCode := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
		if exitCode&smartctlFatalExitBits != 0 {
			return failed(smartctlFailureReason(out, exitErr.Stderr, err))
		}
		err = nil
	}
	if err != nil {
		return failed(err.Error())
	}

	data, parseErr := parseSmartctlJSON(out)
	if parseErr != nil {
		return failed(parseErr.Error())
	}
	data.Device = device.Path
	if device.Type != "" {
		data.Type = device.Type
	}
	data.Warnings = smartctlWarnings(exitCode)
	return data, nil
}

// dedupeDrives drops later entries for a drive already listed under another path, such as
// a multipath disk or /dev/nvme0 and /dev/nvme0n1. Drives are matched by model and serial;
// entries without a serial are always kept.
func dedupeDrives(drives []*SmartData) []*SmartData {
	seen := make(map[string]bool, len(drives))
	unique := make([]*SmartData, 0, len(drives))
	for _, drive := range drives {
		if drive.SerialNumber != "" {
			key := drive.ModelName + "\x00" + drive.SerialNumber
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		unique = append(unique, drive)
	}
	return unique
}

// readDevices reads every device, at most smartctlConcurrency at a time, keeping the device order
func readDevices(devices []smartDevice) ([]*SmartData, []CustomErr) {
	drives := make([]*SmartData, len(devices))
	deviceErrs := make([][]CustomErr, len(devices))

	var waitGroup sync.WaitGroup
	slots := make(chan struct{}, smartctlConcurrency)
	for i, device := range devices {
		waitGroup.Add(1)
		go func(i int, device smartDevice) {
			defer waitGroup.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			drives[i], deviceErrs[i] = getMetrics(device)
		}(i, device)
	}
	waitGroup.Wait()

	var errs []CustomErr
	for _, deviceErr := range deviceErrs {
		errs = append(errs, deviceErr...)
	}
	return drives, errs
}

// GetSmartMetrics returns the SMART state of every drive, one entry per drive.
// Devices from `smartctl --scan` can be extended or given an explicit type with SMART_DEVICES.
func GetSmartMetrics() (MetricsSlice, []CustomErr) {
	metrics := make(MetricsSlice, 0)

	// Check if smartctl is installed
	if err := checkSmartctlInstalled(); err != nil {
		return metrics, []CustomErr{{
			Metric: []string{"smartctl"},
			Error:  err.Error(),
		}}
	}

	// Scan for devices
	var smartCtlrErrs []CustomErr
	devices, devicesErr := scanDevices()
	if devicesErr != nil {
		smartCtlrErrs = append(smartCtlrErrs, CustomErr{
			Metric: []string{"smart"},
			Error:  devicesErr.Error(),
		})
		// Without configured devices there is nothing left to query
		if len(smartDeviceOverrides) == 0 {
			return metrics, smartCtlrErrs
		}
	}
	devices = mergeDevices(devices, smartDeviceOverrides, resolveDevicePath)

	drives, deviceErrs := readDevices(devices)
	smartCtlrErrs = append(smartCtlrErrs, deviceErrs...)
	for _, drive := range dedupeDrives(drives) {
		metrics = append(metrics, drive)
	}
	return metrics, smartCtlrErrs
}
