package test

import (
	"fmt"
	"slices"
	"testing"
)

// generateDevices simulates N unique device names.
func generateDevices(n int) []string {
	devices := make([]string, n)
	for i := range n {
		devices[i] = fmt.Sprintf("/dev/sd%d", i)
	}
	return devices
}

// Benchmark for slices.Contains approach
func BenchmarkCheckedDevicesWithSlices(b *testing.B) {
	b.ReportAllocs()

	devices := generateDevices(10000) // 10k devices
	var checked []string

	b.ResetTimer()
	for i := range b.N {
		target := devices[i%len(devices)]
		if !slices.Contains(checked, target) {
			checked = append(checked, target)
		}
	}
}

// Benchmark for map[string]struct{} approach
func BenchmarkCheckedDevicesWithMap(b *testing.B) {
	b.ReportAllocs()

	devices := generateDevices(10000)
	checked := make(map[string]struct{})

	b.ResetTimer()
	for i := range b.N {
		target := devices[i%len(devices)]
		if _, ok := checked[target]; !ok {
			checked[target] = struct{}{}
		}
	}
}
