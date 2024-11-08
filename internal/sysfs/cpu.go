package sysfs

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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

func readCpuFreqFile(path string) (int, error) {
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

func CpuTemperature() ([]float32, error) {
	// Look in all these folders for core temp
	corePaths := []string{
		"/sys/devices/platform/coretemp.0/hwmon/hwmon*/temp*_input",
		"/sys/class/hwmon/hwmon*/temp*_input",
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
			// Look in the corresponding label to see if this is a core temp
			labelPath := strings.Replace(path, "_input", "_label", 1)
			if label, err := os.ReadFile(labelPath); err == nil {
				labelStr := strings.ToLower(strings.TrimSpace(string(label)))
				// Only process if it's a core
				// * tctl is the temperature control value for AMD processors. We should also consider it as a core temperature.
				if strings.Contains(labelStr, "core") || strings.Contains(labelStr, "tctl") {
					if temp, err := readTempFile(path); err == nil {
						temps = append(temps, temp)
					}
				}
			}
		}
	}

	if len(temps) == 0 {
		return nil, errors.New("unable to read CPU temperature")
	}
	return temps, nil
}

func CpuCurrentFrequency() (int, error) {
	frequency, cpuFrequencyError := readCpuFreqFile("/sys/devices/system/cpu/cpufreq/policy0/scaling_cur_freq")

	if cpuFrequencyError != nil {
		return 0, cpuFrequencyError
	}

	// Convert frequency to mHz
	return frequency / 1000, nil
}
