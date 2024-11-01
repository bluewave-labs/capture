package metric

import (
	"github.com/shirou/gopsutil/v4/host"
)

func GetHostInformation() (*HostData, []string) {
	var hostErrors []string
	defaultHostData := &HostData{
		Os:            "unknown",
		Platform:      "unknown",
		KernelVersion: "unknown",
	}
	info, infoErr := host.Info()

	if infoErr != nil {
		hostErrors = append(hostErrors, infoErr.Error())
		return defaultHostData, hostErrors
	}

	return &HostData{
		Os:            info.OS,
		Platform:      info.Platform,
		KernelVersion: info.KernelVersion,
	}, hostErrors
}
