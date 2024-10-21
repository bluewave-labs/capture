package sysfs

import (
	"strconv"
	"strings"
)

func CpuTemperature() (*float32, error) {
	temperature, cpuTemperatureError := ShellExec("cat /sys/class/hwmon/hwmon3/temp1_input")

	if cpuTemperatureError != nil {
		return nil, cpuTemperatureError
	}

	temperature = strings.TrimSuffix(temperature, "\n")
	temp, strConvErr := strconv.Atoi(temperature)

	if strConvErr != nil {
		return nil, strConvErr
	}

	var temp_float = float32(temp) / 1000
	return &temp_float, nil
}
