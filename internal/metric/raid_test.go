package metric

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeSysfs creates files below root from a map of relative path to content
func writeSysfs(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for path, content := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatalf("failed to create %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content+"\n"), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", full, err)
		}
	}
}

func intPtr(value int) *int           { return &value }
func stringPtr(value string) *string  { return &value }
func floatPtr(value float64) *float64 { return &value }
func raidArrays(slice MetricsSlice) []RaidData {
	arrays := make([]RaidData, 0, len(slice))
	for _, item := range slice {
		arrays = append(arrays, *item.(*RaidData))
	}
	return arrays
}

func TestCollectRaidMetrics(t *testing.T) {
	root := t.TempDir()
	writeSysfs(t, root, map[string]string{
		// Healthy mirror
		"md0/md/array_state":    "clean",
		"md0/md/level":          "raid1",
		"md0/md/raid_disks":     "2",
		"md0/md/degraded":       "0",
		"md0/md/sync_action":    "idle",
		"md0/md/sync_completed": "none",
		"md0/md/dev-sda1/state": "in_sync",
		"md0/md/dev-sdb1/state": "in_sync",
		// Degraded mirror rebuilding onto a new disk
		"md1/md/array_state":    "active",
		"md1/md/level":          "raid1",
		"md1/md/raid_disks":     "2",
		"md1/md/degraded":       "1",
		"md1/md/sync_action":    "recover",
		"md1/md/sync_completed": "250 / 1000",
		"md1/md/dev-sda2/state": "in_sync",
		"md1/md/dev-sdc2/state": "spare",
		// Striped array: no redundancy files
		"md2/md/array_state":   "clean",
		"md2/md/level":         "raid0",
		"md2/md/raid_disks":    "2",
		"md2/md/dev-sdd/state": "in_sync",
		"md2/md/dev-sde/state": "in_sync",
		// Not an md device
		"sda/size": "1000",
	})

	got, errs := collectRaidMetrics(root)
	if len(errs) != 0 {
		t.Fatalf("collectRaidMetrics() errors = %+v", errs)
	}

	expected := []RaidData{
		{
			Device: "/dev/md0", Level: "raid1", State: "clean",
			RaidDisks: intPtr(2), DegradedDisks: intPtr(0), SyncAction: stringPtr("idle"),
			Members: []RaidMember{{Device: "/dev/sda1", State: "in_sync"}, {Device: "/dev/sdb1", State: "in_sync"}},
		},
		{
			Device: "/dev/md1", Level: "raid1", State: "active",
			RaidDisks: intPtr(2), DegradedDisks: intPtr(1), SyncAction: stringPtr("recover"), SyncProgress: floatPtr(0.25),
			Members: []RaidMember{{Device: "/dev/sda2", State: "in_sync"}, {Device: "/dev/sdc2", State: "spare"}},
		},
		{
			Device: "/dev/md2", Level: "raid0", State: "clean",
			RaidDisks: intPtr(2),
			Members:   []RaidMember{{Device: "/dev/sdd", State: "in_sync"}, {Device: "/dev/sde", State: "in_sync"}},
		},
	}
	if arrays := raidArrays(got); !reflect.DeepEqual(arrays, expected) {
		t.Errorf("collectRaidMetrics() = %+v, expected %+v", arrays, expected)
	}
}

func TestCollectRaidMetricsInactiveArray(t *testing.T) {
	root := t.TempDir()
	writeSysfs(t, root, map[string]string{
		"md127/md/array_state":   "inactive",
		"md127/md/level":         "",
		"md127/md/raid_disks":    "0",
		"md127/md/dev-sdf/state": "faulty",
	})

	got, errs := collectRaidMetrics(root)
	if len(errs) != 0 {
		t.Fatalf("collectRaidMetrics() errors = %+v", errs)
	}
	expected := []RaidData{{
		Device: "/dev/md127", State: "inactive", RaidDisks: intPtr(0),
		Members: []RaidMember{{Device: "/dev/sdf", State: "faulty"}},
	}}
	if arrays := raidArrays(got); !reflect.DeepEqual(arrays, expected) {
		t.Errorf("collectRaidMetrics() = %+v, expected %+v", arrays, expected)
	}
}

func TestCollectRaidMetricsNoArrays(t *testing.T) {
	got, errs := collectRaidMetrics(t.TempDir())
	if len(errs) != 0 || got == nil || len(got) != 0 {
		t.Errorf("collectRaidMetrics() = %+v, %+v, expected an empty, non-nil list", got, errs)
	}
}

func TestCollectRaidMetricsUnreadableArray(t *testing.T) {
	root := t.TempDir()
	writeSysfs(t, root, map[string]string{
		"md0/md/level":       "raid1",
		"md1/md/array_state": "clean",
		"md1/md/level":       "raid1",
	})

	got, errs := collectRaidMetrics(root)
	if len(got) != 1 || len(errs) != 1 {
		t.Fatalf("collectRaidMetrics() = %+v, %+v, expected one array and one error", got, errs)
	}
	if errs[0].Metric[0] != "raid" {
		t.Errorf("error metric = %v, expected raid", errs[0].Metric)
	}
}

func TestParseSyncCompleted(t *testing.T) {
	tests := []struct {
		value    string
		expected *float64
	}{
		{"none", nil},
		{"delayed", nil},
		{"0 / 0", nil},
		{"500 / 1000", floatPtr(0.5)},
		{"1000 / 1000", floatPtr(1)},
	}
	for _, tt := range tests {
		got := parseSyncCompleted(tt.value)
		if (got == nil) != (tt.expected == nil) || (got != nil && *got != *tt.expected) {
			t.Errorf("parseSyncCompleted(%q) = %v, expected %v", tt.value, got, tt.expected)
		}
	}
}

func TestReadSysfsIntReshape(t *testing.T) {
	root := t.TempDir()
	writeSysfs(t, root, map[string]string{"raid_disks": "4 (3)"})
	if got := readSysfsInt(filepath.Join(root, "raid_disks")); got == nil || *got != 4 {
		t.Errorf("readSysfsInt() = %v, expected 4", got)
	}
}
