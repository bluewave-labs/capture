package test

import (
	"bluewave-uptime-agent/internal/metric"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func shellExec(c string) (string, error) {
	if strings.Contains(c, "&&") || strings.Contains(c, "||") || strings.Contains(c, ";") {
		return "", errors.New("It's forbidden to execute consecutive commands")
	}
	cmd := exec.Command("bash", "-c", c)

	// Run the command and capture the output
	output, err := cmd.Output()

	if err != nil {
		return "", err
	}

	return string(output), nil
}

// TestHostLinux tests the GetHostInformation function
// It interacts with the host system to get the NodeName and Kernel Version
// It then compares the values with the ones returned by the GetHostInformation function
func TestHostLinux(t *testing.T) {
	osPlatform, osPlatformErr := shellExec("uname -n") // Nodename
	osKernel, osKernelErr := shellExec("uname -r")     // Kernel version
	info, infoErr := metric.GetHostInformation()

	if infoErr != nil {
		t.Error(infoErr.Error())
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

	assert.Equal(t, *info.Os, runtime.GOOS)
	assert.Equal(t, *info.Platform, strings.TrimSuffix(osPlatform, "\n"))
	assert.Equal(t, *info.KernelVersion, strings.TrimSuffix(osKernel, "\n"))
}
