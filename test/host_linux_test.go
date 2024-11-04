package test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/bluewave-labs/bluewave-uptime-agent/internal/metric"
	"github.com/bluewave-labs/bluewave-uptime-agent/internal/sysfs"
	"github.com/stretchr/testify/assert"
)

// TestHostLinux tests the GetHostInformation function
// It interacts with the host system to get the NodeName and Kernel Version
// It then compares the values with the ones returned by the GetHostInformation function
func TestHostLinux(t *testing.T) {
	osPlatform, osPlatformErr := sysfs.ShellExec("uname -n") // Nodename
	osKernel, osKernelErr := sysfs.ShellExec("uname -r")     // Kernel version
	info, infoErr := metric.GetHostInformation()

	if len(infoErr) != 0 {
		t.Error(infoErr)
		t.FailNow()
	}

	if osKernelErr != nil {
		t.Error(osKernelErr.Error())
		t.FailNow()
	}

	if osPlatformErr != nil {
		t.Error(osPlatformErr.Error())
		t.FailNow()
	}

	assert.Equal(t, info.Os, runtime.GOOS)
	assert.Equal(t, info.Platform, strings.TrimSuffix(osPlatform, "\n"))
	assert.Equal(t, info.KernelVersion, strings.TrimSuffix(osKernel, "\n"))
}
