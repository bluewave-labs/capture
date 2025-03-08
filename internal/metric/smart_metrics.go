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

func parseSmartctlOutput(output string) *SmartMetric {
	startMarker := "=== START OF SMART DATA SECTION ==="
	startIdx := strings.Index(output, startMarker)
	if startIdx == -1 {
		return &SmartMetric{Data: make(map[string]string)}
	}

	section := output[startIdx:]
	endIdx := strings.Index(section, "===")
	if endIdx > len(startMarker) {
		section = section[:endIdx]
	}

	data := make(map[string]string)
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
		
		data[key] = value
	}

	return &SmartMetric{
		Data:   data,
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

func GetSmartMetrics() (MetricsSlice, []CustomErr) { 
	var smartCtlrErrs []CustomErr
	devices, err := scanDevices()
	if err != nil {
		smartCtlrErrs =append(smartCtlrErrs,CustomErr{
            Metric: []string{"smart"},
            Error:  err.Error(),
        })
	}

	var metrics MetricsSlice  // Use MetricsSlice instead of []*SmartMetric
	for _, device := range devices {
		metric, err := getMetrics(device)
		if err != nil {
			log.Printf("Skipping %s: %v", device, err)
			continue
		}
		metrics = append(metrics, metric)
	}

	return metrics, smartCtlrErrs
}