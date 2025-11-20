package integration

import (
	"testing"

	"github.com/bluewave-labs/capture/internal/metric"
)

// TestDiskMetricsContent verifies that disk metrics can be collected and contain valid data.
// It specifically checks that the Mountpoint field is correctly populated for detected disks.
func TestDiskMetricsContent(t *testing.T) {
	// Attempt to collect disk metrics using the internal metric package
	metricsSlice, errs := metric.CollectDiskMetrics()

	// Log any partial errors encountered during collection (e.g. permission denied on specific partitions)
	// We do not fail the test here as partial failures are expected in some environments.
	if len(errs) > 0 {
		t.Logf("Encountered partial errors collecting disk metrics: %v", errs)
	}

	// Verify that we retrieved at least some metrics
	if len(metricsSlice) == 0 {
		t.Skip("No disk metrics found. This may indicate restricted disk access or a collection failure.")
	}

	foundValidDisk := false

	for _, m := range metricsSlice {
		// Assert that the metric is of type *DiskData
		diskData, ok := m.(*metric.DiskData)
		if !ok {
			t.Errorf("Expected *metric.DiskData, got %T", m)
			continue
		}

		// Log the discovered device and mountpoint for debugging context
		t.Logf("Found Disk: Device=%s, Mountpoint=%s", diskData.Device, diskData.Mountpoint)

		// Validation: The Mountpoint field must not be empty
		if diskData.Mountpoint == "" {
			t.Errorf("Disk device %s has an empty Mountpoint field", diskData.Device)
		} else if diskData.Mountpoint == "unknown" {
			t.Logf("Disk device %s has default 'unknown' mountpoint (metric collection may have failed)", diskData.Device)
 		} else {
			foundValidDisk = true
		}
	}

	// If we iterated through metrics but didn't find any valid disk with a mountpoint, log a warning.
	if !foundValidDisk {
		t.Error("No valid disks with mountpoints were found. The mountpoint feature may not be working correctly.")
	}
}