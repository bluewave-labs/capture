package metric

import (
	"github.com/shirou/gopsutil/v4/host"
)

func GetHostInformation() (*HostData, []string) {
	var hostErrors []string
	info, infoErr := host.Info()

	if infoErr != nil {
		hostErrors = append(hostErrors, infoErr.Error())
	}

	return &HostData{
		Os:            info.OS,
		Platform:      info.Platform,
		KernelVersion: info.KernelVersion,
	}, hostErrors
}
