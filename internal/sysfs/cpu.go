package sysfs

import (
	"strconv"
	"strings"
)

func CpuTemperature() (float32, error) {
	temperature, cpuTemperatureError := ShellExec("cat /sys/class/hwmon/hwmon3/temp1_input")

	if cpuTemperatureError != nil {
		return 0, cpuTemperatureError
	}

	temperature = strings.TrimSuffix(temperature, "\n")
	temp, strConvErr := strconv.Atoi(temperature)

	if strConvErr != nil {
		return 0, strConvErr
	}

	var temp_float = float32(temp) / 1000
	return temp_float, nil
}

func CpuCurrentFrequency() (int, error) {
	frequency, cpuFrequencyError := ShellExec("cat /sys/devices/system/cpu/cpufreq/policy0/scaling_cur_freq")

	if cpuFrequencyError != nil {
		return 0, cpuFrequencyError
	}

	frequency = strings.TrimSuffix(frequency, "\n")
	freq, strConvErr := strconv.Atoi(frequency)

	if strConvErr != nil {
		return 0, strConvErr
	}

	// Convert frequency to mHz
	return freq / 1000, nil
}
