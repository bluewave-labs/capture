package metric

import (
	"github.com/shirou/gopsutil/v4/host"
)

func GetHostInformation() (*HostData, error) {
	info, infoErr := host.Info()

	if infoErr != nil {
		return nil, infoErr
	}

	return &HostData{
		Os:            info.OS,
		Platform:      info.Platform,
		KernelVersion: info.KernelVersion,
	}, nil
}
