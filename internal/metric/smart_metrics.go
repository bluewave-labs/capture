package metric

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Check if smartctl is installed
func checkSmartctlInstalled() error {
	_, err := exec.LookPath("smartctl")
	if err != nil {
		return fmt.Errorf("smartctl is not installed")
	}
	return nil
}

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

func isErrorKey(key string) bool {
	loweredKey := strings.ToLower(key)
	return strings.Contains(loweredKey, "failed") || strings.Contains(loweredKey, "error")
}

func setField(key, value string, data *SmartData) {
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
	case "host read commands":
		data.HostReadCommands = value
	case "host write commands":
		data.HostWriteCommands = value
	case "percentage used":
		data.PercentageUsed = value
	case "power cycles":
		data.PowerCycles = value
	case "power on hours":
		data.PowerOnHours = value
	case "smart overall-health self-assessment test result":
		data.SmartOverallHealthResult = value
	case "temperature":
		data.Temperature = value
	case "unsafe shutdowns":
		data.UnsafeShutdowns = value
	}
}

// parseSmartctlOutput parses the output from smartctl command
func parseSmartctlOutput(output string) (*SmartData, []CustomErr) {
	// Define the start marker to locate the section of interest
	startMarker := "=== START OF SMART DATA SECTION ==="
	startIdx := strings.Index(output, startMarker)
	if startIdx == -1 {
		return &SmartData{}, nil
	}

	// Extract the section starting from the marker
	section := output[startIdx:]
	endIdx := strings.Index(section, "===")
	if endIdx > len(startMarker) {
		section = section[:endIdx]
	}

	data := SmartData{}
	var errors []CustomErr

	// Split the section into lines
	lines := strings.Split(section, "\n")

	for _, line := range lines {
		// Skip empty lines or lines starting with "==="
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "===") {
			continue
		}

		// Split each line into a key-value pair
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Clean up value by removing extra spaces and brackets
		value = regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")
		value = strings.Trim(value, "[]")

		// Set the field in the SmartData struct
		setField(key, value, &data)

		// If the key contains "error" or "failed", add it to the errors
		if isErrorKey(key) {
			errors = append(errors, CustomErr{
				Metric: []string{key},
				Error:  fmt.Sprintf("Unable to retrieve the '%s'", key),
			})
		}
	}

	return &data, errors
}

func getMetrics(device string) (*SmartData, []CustomErr) {
	cmd := exec.Command("smartctl", "-d", "nvme", "--xall", "--nocheck", "standby", device)
	out, err := cmd.CombinedOutput()

	// If there's an exit error with exit code 4, we ignore the error
	if exitErr, ok := err.(*exec.ExitError); ok {
		switch exitErr.ExitCode() {
		case 4:

			err = nil
		case 2:
			// Exit code 2 indicates permission denied
			if strings.HasSuffix(string(out), "failed: Permission denied\n") {
				return &SmartData{}, []CustomErr{{
					Metric: []string{"smartctl"},
					Error:  "smartctl failed: permission denied (try running as root)",
				}}
			}
		}
	}

	// If there's an error executing the command, return empty SmartData and the error
	if err != nil {
		return &SmartData{}, []CustomErr{{
			Metric: []string{"smartctl"},
			Error:  fmt.Sprintf("smartctl failed: %v", err),
		}}
	}
	
	return parseSmartctlOutput(string(out))
}

// GetSmartMetrics retrieves the SMART metrics from all available devices.
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
	devices, devicesErr := scanDevices()
	if devicesErr != nil {
		// Return the error if the scan fails
		smartCtlrErrs = append(smartCtlrErrs, CustomErr{
			Metric: []string{"smart"},
			Error:  devicesErr.Error(),
		})
		return metrics, smartCtlrErrs
	}

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
