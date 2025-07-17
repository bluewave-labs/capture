package integration

import (
	"errors"
	"runtime"
	"testing"

	"github.com/bluewave-labs/capture/internal/system"
	"github.com/bluewave-labs/capture/test"
)

// TestCPUTemperature tests the functionality of retrieving the CPU temperature.
// It checks if the temperature can be fetched without errors and logs the result.
func TestCPUTemperature(t *testing.T) {
	platformToSkipOnCI := runtime.GOOS == "linux" || runtime.GOOS == "windows" // GitHub ubuntu and windows runners may not have permission to access CPU frequency
	test.SkipIfCI(t, &platformToSkipOnCI, "Skipping CPU temperature test in CI environment due to potential permission and virtualization issues")

	temperature, err := system.CPUTemperature()
	if err != nil {
		if errors.Is(err, system.ErrCPUDetailsNotImplemented) {
			t.Skip("CPU temperature retrieval is not implemented on this platform")
		}

		t.Fatalf("Failed to get CPU temperature: %v", err)
	}
	t.Logf("CPU Temperature: %v", temperature)
}

// TestCPUCurrentFrequency tests the functionality of retrieving the CPU's current frequency.
// It checks if the frequency can be fetched without errors and logs the result.
func TestCPUCurrentFrequency(t *testing.T) {
	platformToSkipOnCI := runtime.GOOS == "linux" // GitHub ubuntu runners may not have permission to access CPU frequency
	test.SkipIfCI(t, &platformToSkipOnCI, "Skipping CPU current frequency test in CI environment due to potential permission and virtualization issues")

	frequency, err := system.CPUCurrentFrequency()
	if err != nil {
		if errors.Is(err, system.ErrCPUDetailsNotImplemented) {
			t.Skip("CPU current frequency retrieval is not implemented on this platform")
		}

		t.Fatalf("Failed to get CPU current frequency: %v", err)
	}
	t.Logf("CPU Current Frequency: %v MHz", frequency)
}
