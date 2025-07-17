//go:build darwin
// +build darwin

package system

import (
	"errors"
)

var (
	ErrCPUDetailsNotImplemented = errors.New("CPU details not implemented on darwin")
)

func CPUTemperature() ([]float32, error) {
	return nil, ErrCPUDetailsNotImplemented
}
func CPUCurrentFrequency() (int, error) {
	return 0, ErrCPUDetailsNotImplemented
}
