package metric

import (
	"github.com/bluewave-labs/capture/internal/system"
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

	prettyName, prettyErr := system.GetPrettyName()
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
