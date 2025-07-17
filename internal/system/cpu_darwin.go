//go:build darwin
// +build darwin

package system

import (
	"errors"
)

var (
	ErrNotImplemented = errors.New("not implemented on this platform")
)

func CPUTemperature() ([]float32, error) {
	return nil, ErrNotImplemented
}
func CPUCurrentFrequency() (int, error) {
	return 0, ErrNotImplemented
}
