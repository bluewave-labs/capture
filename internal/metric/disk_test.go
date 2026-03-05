package metric

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

// TestDiskMetricsContent verifies that disk metrics can be collected and contain valid data.
// It specifically checks that the Mountpoint field is correctly populated for detected disks.
func TestDiskMetricsContent(t *testing.T) {
	// Attempt to collect disk metrics using the internal metric package
	metricsSlice, errs := CollectDiskMetrics()

	// Log any partial errors encountered during collection (e.g. permission denied on specific partitions)
	// We do not fail the test here as partial failures are expected in some environments.
	if len(errs) > 0 {
		t.Logf("Encountered partial errors collecting disk metrics: %v", errs)
	}

	// If there are truly no metrics, or only the default "unknown" entry, skip in this environment.
	if len(metricsSlice) == 0 {
		t.Skip("No disk metrics found. This may indicate restricted disk access or a collection failure.")
	}

	if len(metricsSlice) == 1 {
		if dd, ok := metricsSlice[0].(*DiskData); ok &&
			dd.Device == "unknown" && dd.Mountpoint == "unknown" {
			t.Skip("Only default 'unknown' disk metric returned; disk metrics not available in this environment.")
		}
	}

	foundValidDisk := false

	for _, m := range metricsSlice {
		// Assert that the metric is of type *DiskData
		diskData, ok := m.(*DiskData)
		if !ok {
			t.Errorf("Expected *DiskData, got %T", m)
			continue
		}

		// Log the discovered device and mountpoint for debugging context
		t.Logf("Found Disk: Device=%s, Mountpoint=%s", diskData.Device, diskData.Mountpoint)

		// Validation: The Mountpoint field must not be empty
		switch diskData.Mountpoint {
		case "":
			t.Errorf("Disk device %s has an empty Mountpoint field", diskData.Device)
		case "unknown":
			t.Logf("Disk device %s has default 'unknown' mountpoint (metric collection may have failed)", diskData.Device)
		default:
			foundValidDisk = true
		}
	}

	// If we iterated through metrics but didn't find any valid disk with a mountpoint, log a warning.
	if !foundValidDisk {
		t.Error("No valid disks with mountpoints were found. The mountpoint feature may not be working correctly.")
	}
}

// TestResolveDMNameFromMapperWithRoot verifies that the function correctly resolves the device mapper name.
func TestResolveDMNameFromMapperWithRoot(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sysfs dm tests are linux-only")
	}

	root := t.TempDir()
	name := "ubuntu--vg-ubuntu--lv"

	p := filepath.Join(root, "block", "dm-0", "dm")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(p, "name"), []byte(name+"\n"), 0o600); err != nil {
		t.Fatalf("write name: %v", err)
	}

	dm, ok := resolveDMNameFromMapperWithRoot(name, root)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if dm != "dm-0" {
		t.Fatalf("expected dm-0, got %q", dm)
	}
}

func TestBuildDeviceKeyCandidates_AddsDMForMapper(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sysfs dm tests are linux-only")
	}

	old := sysfsRoot
	t.Cleanup(func() { sysfsRoot = old })

	root := t.TempDir()
	sysfsRoot = root

	name := "ubuntu--vg-ubuntu--lv"
	p := filepath.Join(root, "block", "dm-0", "dm")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(p, "name"), []byte(name), 0o600); err != nil {
		t.Fatalf("write name: %v", err)
	}

	candidates := buildDeviceKeyCandidates("/dev/mapper/" + name)
	if !slices.Contains(candidates, "dm-0") {
		t.Fatalf("expected candidates to include dm-0, got %v", candidates)
	}
}
