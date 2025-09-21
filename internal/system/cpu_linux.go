//go:build linux
// +build linux

package system

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	ErrCPUDetailsNotImplemented = errors.New("CPU details not implemented on linux")
)

// readTempFile reads a temperature file and returns the temperature in Celsius.
func readTempFile(path string) (float32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	temp, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}

	return float32(temp) / 1000, nil
}

// readCPUFreqFile reads a CPU frequency file and returns the frequency in kHz.
func readCPUFreqFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	freq, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}

	return freq, nil
}

// isValidCPUTempSensor determines if a temperature sensor should be considered for CPU temperature reading
func isValidCPUTempSensor(path string) bool {
	if !strings.Contains(path, "hwmon") {
		// For non-hwmon paths (like thermal_zone), assume valid
		return true
	}

	labelPath := strings.Replace(path, "_input", "_label", 1)
	label, err := os.ReadFile(labelPath)
	if err != nil {
		// No label file exists, assume it could be a CPU temperature sensor
		return true
	}

	labelStr := strings.ToLower(strings.TrimSpace(string(label)))
	return strings.Contains(labelStr, "core") || strings.Contains(labelStr, "tctl")
}

// addTemperatureIfValid reads temperature from path and adds it to temps slice if successful
func addTemperatureIfValid(path string, temps *[]float32) {
	if temp, err := readTempFile(path); err == nil {
		*temps = append(*temps, temp)
	}
}

// CPUTemperature returns the temperature of CPU cores in Celsius.
func CPUTemperature() ([]float32, error) {
	// Look in all these folders for core temp
	corePaths := []string{
		"/sys/devices/platform/coretemp.0/hwmon/hwmon*/temp*_input", // hwmon
		"/sys/class/hwmon/hwmon*/temp*_input",                       // hwmon
		// "/sys/class/thermal/thermal_zone0/temp",                     // thermal_zone. it's the same as /sys/class/hwmon/hwmon0/temp1_input
	}

	var temps []float32

	for _, pathPattern := range corePaths {
		// Find paths for inputs that may contain core temp
		matches, err := filepath.Glob(pathPattern)
		if err != nil { // Keep looking for matches if we get an error
			continue
		}

		// Loop over temp_input paths
		for _, path := range matches {
			if isValidCPUTempSensor(path) {
				addTemperatureIfValid(path, &temps)
			}
		}
	}

	if len(temps) == 0 {
		return nil, errors.New("unable to read CPU temperature")
	}
	return temps, nil
}

// CPUCurrentFrequency returns the current CPU frequency in MHz.
func CPUCurrentFrequency() (int, error) {
	frequency, cpuFrequencyError := readCPUFreqFile("/sys/devices/system/cpu/cpufreq/policy0/scaling_cur_freq")

	if cpuFrequencyError != nil {
		return 0, cpuFrequencyError
	}

	// Convert kHz to MHz
	return frequency / 1000, nil
}
