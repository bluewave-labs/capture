package metric

import (
	"bufio"
	"errors"
	"os"
	"strings"

	"github.com/shirou/gopsutil/v4/host"
)

func GetHostInformation() (*HostData, []CustomErr) {
	var hostErrors []CustomErr
	defaultHostData := HostData{
		Os:            "unknown",
		Platform:      "unknown",
		KernelVersion: "unknown",
		PrettyName:    "unknown",
	}
	info, infoErr := host.Info()

	if infoErr != nil {
		hostErrors = append(hostErrors, CustomErr{
			Metric: []string{"host.os", "host.platform", "host.kernel_version"},
			Error:  infoErr.Error(),
		})
		return &defaultHostData, hostErrors
	}

	prettyName, prettyErr := GetPrettyName()
	if prettyErr != nil {
		hostErrors = append(hostErrors, CustomErr{
			Metric: []string{"host.pretty_name"},
			Error:  prettyErr.Error(),
		})
		prettyName = "unknown"
	}

	return &HostData{
		Os:            info.OS,
		Platform:      info.Platform,
		KernelVersion: info.KernelVersion,
		PrettyName:    prettyName,
	}, hostErrors
}

func GetPrettyName() (string, error) {
	const osReleasePath = "/etc/os-release"

	file, err := os.Open(osReleasePath)
	if err != nil {
		return "", errors.New("unable to open /etc/os-release: " + err.Error())
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			pretty := strings.TrimPrefix(line, "PRETTY_NAME=")
			return strings.Trim(pretty, `"`), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", errors.New("error reading /etc/os-release: " + err.Error())
	}

	return "", errors.New("PRETTY_NAME field not found in /etc/os-release")
}
