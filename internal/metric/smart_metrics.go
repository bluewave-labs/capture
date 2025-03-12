package metric

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

// scanDevices returns a list of disks using `smartctl --scan`
func scanDevices() ([]string, error) {
	out, err := exec.Command("smartctl", "--scan").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to scan devices: %v", err)
	}

	var devices []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			devices = append(devices, fields[0])
		}
	}
	return devices, nil
}

// parseSmartctlOutput parses the output from smartctl command
func parseSmartctlOutput(output string) *SmartMetric {
	startMarker := "=== START OF SMART DATA SECTION ==="
	startIdx := strings.Index(output, startMarker)
	if startIdx == -1 {
		return &SmartMetric{Data: SmartData{}, Errors: []CustomErr{}}
	}

	section := output[startIdx:]
	endIdx := strings.Index(section, "===")
	if endIdx > len(startMarker) {
		section = section[:endIdx]
	}

	data := SmartData{}
	var errors []CustomErr
	lines := strings.Split(section, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "===") {
			continue
		}
		
		// Split into key/value pairs
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Clean up value: remove extra spaces and brackets
		value = regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")
		value = strings.Trim(value, "[]")

		switch strings.ToLower(key) {
		case "available spare":
			data.AvailableSpare = value
		case "available spare threshold":
			data.AvailableSpareThreshold = value
		case "controller busy time":
			data.ControllerBusyTime = value
		case "critical warning":
			data.CriticalWarning = value
		case "data units read":
			data.DataUnitsRead = value
		case "data units written":
			data.DataUnitsWritten = value
		case "error information log entries":
			data.ErrorInformationLogEntries = value
		case "host read commands":
			data.HostReadCommands = value
		case "host write commands":
			data.HostWriteCommands = value
		case "media and data integrity errors":
			data.MediaAndDataIntegrityErrors = value
		case "percentage used":
			data.PercentageUsed = value
		case "power cycles":
			data.PowerCycles = value
		case "power on hours":
			data.PowerOnHours = value
		case "read 1 entries from error information log failed":
			data.Read1EntriesFromErrorLogFailed = value
		case "smart overall-health self-assessment test result":
			data.SmartOverallHealthResult = value
		case "temperature":
			data.Temperature = value
		case "unsafe shutdowns":
			data.UnsafeShutdowns = value
		}

		// If the key contains "error" or "failed", add it to the errors
		if strings.Contains(key, "failed") || strings.Contains(key, "error") {
			errors = append(errors, CustomErr{
				Metric: []string{key},
				Error:  fmt.Sprintf("Unable to retrieve the '%s'", key),
			})
		}
	}

	return &SmartMetric{
		Data:   data,
		Errors: errors,
	}
}

func getMetrics(device string) (*SmartMetric, error) {
	cmd := exec.Command("smartctl", "-d", "nvme", "--xall", "--nocheck", "standby", device)
	out, err := cmd.CombinedOutput()

	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 4 {
		err = nil
	}

	if err != nil {
		return nil, fmt.Errorf("smartctl failed: %v", err)
	}

	return parseSmartctlOutput(string(out)), nil
}

// GetSmartMetrics retrieves the SMART metrics from all available devices.
func GetSmartMetrics() (SmartMetric, []CustomErr) {
	var smartCtlrErrs []CustomErr
	devices, err := scanDevices()
	if err != nil {
		smartCtlrErrs = append(smartCtlrErrs, CustomErr{
			Metric: []string{"smart"},
			Error:  err.Error(),
		})
	}

	var metrics SmartMetric
	for _, device := range devices {
		metric, err := getMetrics(device)
		if err != nil {
			log.Printf("Skipping %s: %v", device, err)
			continue
		}
	
		metrics = *metric
	}
	return metrics, smartCtlrErrs
}
