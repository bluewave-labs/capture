package metric

import (
	"github.com/shirou/gopsutil/v4/host"
)

type HostData struct {
	Os            *string `json:"os"`             // Operating System
	Platform      *string `json:"platform"`       // Platform Name
	KernelVersion *string `json:"kernel_version"` // Kernel Version
}

func GetHostInformation() (*HostData, error) {
	info, infoErr := host.Info()

	if infoErr != nil {
		return nil, infoErr
	}

	return &HostData{
		Os:            &info.OS,
		Platform:      &info.Platform,
		KernelVersion: &info.KernelVersion,
	}, nil
}
