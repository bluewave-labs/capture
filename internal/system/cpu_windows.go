//go:build windows
// +build windows

package system

import (
	"fmt"

	"github.com/yusufpapurcu/wmi"
)

// CPUTemperature returns CPU temperatures in Celsius for all thermal zones.
// Note: May not work on all systems due to WMI access restrictions.
func CPUTemperature() ([]float32, error) {
	var temps []struct {
		CurrentTemperature uint32
	}

	err := wmi.Query("SELECT CurrentTemperature FROM MSAcpi_ThermalZoneTemperature", &temps)
	if err != nil {
		return nil, fmt.Errorf("failed to query thermal zone temperature: %w", err)
	}

	if len(temps) == 0 {
		return nil, fmt.Errorf("no thermal zone data available")
	}

	temperatures := make([]float32, 0, len(temps))
	for _, temp := range temps {
		// Convert from tenths of Kelvin to Celsius
		tempC := float32(temp.CurrentTemperature-2732) / 10.0
		temperatures = append(temperatures, tempC)
	}

	return temperatures, nil
}

// CPUCurrentFrequency returns the current CPU frequency in MHz.
func CPUCurrentFrequency() (int, error) {
	var processors []struct {
		CurrentClockSpeed uint32
	}

	err := wmi.Query("SELECT CurrentClockSpeed FROM Win32_Processor", &processors)
	if err != nil {
		return 0, fmt.Errorf("failed to query processor frequency: %w", err)
	}

	if len(processors) == 0 {
		return 0, fmt.Errorf("no processor data available")
	}

	// Return the first processor's current clock speed in MHz
	return int(processors[0].CurrentClockSpeed), nil
}
