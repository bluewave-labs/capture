package integration

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bluewave-labs/capture/internal/metric"
	"github.com/bluewave-labs/capture/test"
)

// findRaidArray returns the array with the given device from GetRaidMetrics
func findRaidArray(t *testing.T, device string) metric.RaidData {
	t.Helper()

	arrays, errs := metric.GetRaidMetrics()
	if len(errs) > 0 {
		t.Fatalf("GetRaidMetrics() errors = %+v", errs)
	}
	for _, item := range arrays {
		if array, ok := item.(*metric.RaidData); ok && array.Device == device {
			return *array
		}
	}
	t.Fatalf("array %s not found in %+v", device, arrays)
	return metric.RaidData{}
}

// memberState returns the state of the member with the given device
func memberState(array metric.RaidData, device string) string {
	for _, member := range array.Members {
		if member.Device == device {
			return member.State
		}
	}
	return ""
}

// TestRaidMetrics builds a real two-disk mdadm mirror on loop devices, checks that it
// is reported as healthy, then fails one member and checks that it is reported as degraded.
func TestRaidMetrics(t *testing.T) {
	test.RequireLinux(t)
	test.RequireRoot(t)
	test.RequireCmd(t, "mdadm")
	test.RequireCmd(t, "losetup")

	const arrayDevice = "/dev/md200"
	dir := t.TempDir()
	loopDevices := make([]string, 0, 2)
	t.Cleanup(func() {
		test.ShellRun(nil, "mdadm", "--stop", arrayDevice)
		for _, loopDevice := range loopDevices {
			test.ShellRun(nil, "mdadm", "--zero-superblock", loopDevice)
			test.ShellRun(nil, "losetup", "-d", loopDevice)
		}
	})

	for i := range 2 {
		backingFile := filepath.Join(dir, fmt.Sprintf("member%d.img", i))
		test.ShellRun(t, "truncate", "-s", "64M", backingFile)
		loopDevices = append(loopDevices, test.ShellRun(t, "losetup", "--find", "--show", backingFile))
	}

	// --assume-clean skips the initial resync so the array starts healthy and idle
	test.ShellRun(t, "mdadm", "--create", arrayDevice, "--run", "--assume-clean",
		"--level=1", "--raid-devices=2", "--metadata=1.2", loopDevices[0], loopDevices[1])

	t.Run("healthy", func(t *testing.T) {
		array := findRaidArray(t, arrayDevice)
		t.Logf("Healthy array: %+v", array)

		if array.Level != "raid1" {
			t.Errorf("Level = %q, expected raid1", array.Level)
		}
		if array.RaidDisks == nil || *array.RaidDisks != 2 {
			t.Errorf("RaidDisks = %v, expected 2", array.RaidDisks)
		}
		if array.DegradedDisks == nil || *array.DegradedDisks != 0 {
			t.Errorf("DegradedDisks = %v, expected 0", array.DegradedDisks)
		}
		if array.SyncAction == nil || *array.SyncAction != "idle" {
			t.Errorf("SyncAction = %v, expected idle", array.SyncAction)
		}
		for _, loopDevice := range loopDevices {
			if state := memberState(array, loopDevice); !strings.Contains(state, "in_sync") {
				t.Errorf("member %s state = %q, expected in_sync", loopDevice, state)
			}
		}
	})

	t.Run("degraded", func(t *testing.T) {
		test.ShellRun(t, "mdadm", arrayDevice, "--fail", loopDevices[1])

		array := findRaidArray(t, arrayDevice)
		t.Logf("Degraded array: %+v", array)

		if array.DegradedDisks == nil || *array.DegradedDisks != 1 {
			t.Errorf("DegradedDisks = %v, expected 1", array.DegradedDisks)
		}
		if state := memberState(array, loopDevices[1]); !strings.Contains(state, "faulty") {
			t.Errorf("failed member %s state = %q, expected faulty", loopDevices[1], state)
		}
		if state := memberState(array, loopDevices[0]); !strings.Contains(state, "in_sync") {
			t.Errorf("remaining member %s state = %q, expected in_sync", loopDevices[0], state)
		}
	})
}
