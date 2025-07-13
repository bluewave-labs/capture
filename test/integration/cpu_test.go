package test

import (
	"testing"

	"github.com/bluewave-labs/capture/internal/system"
)

// TestCPUTemperature tests the functionality of retrieving the CPU temperature.
// It checks if the temperature can be fetched without errors and logs the result.
func TestCPUTemperature(t *testing.T) {
	temperature, err := system.CPUTemperature()
	if err != nil {
		t.Fatalf("Failed to get CPU temperature: %v", err)
	}
	t.Logf("CPU Temperature: %v", temperature)
}

// TestCPUCurrentFrequency tests the functionality of retrieving the CPU's current frequency.
// It checks if the frequency can be fetched without errors and logs the result.
func TestCPUCurrentFrequency(t *testing.T) {
	frequency, err := system.CPUCurrentFrequency()
	if err != nil {
		t.Fatalf("Failed to get CPU current frequency: %v", err)
	}
	t.Logf("CPU Current Frequency: %v MHz", frequency)
}
