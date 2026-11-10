package metric

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// sysBlockRoot is where Linux exposes block devices; md arrays appear as /sys/block/md*/md.
// Other platforms have no such directory, so no arrays are reported there.
const sysBlockRoot = "/sys/block"

type RaidData struct {
	Device        string       `json:"device"`         // Array device, e.g. /dev/md0
	Level         string       `json:"level"`          // RAID level, e.g. raid1 (empty while the array is inactive)
	State         string       `json:"state"`          // md array_state, e.g. clean, active, read-auto, inactive
	RaidDisks     *int         `json:"raid_disks"`     // Number of member slots in the array
	DegradedDisks *int         `json:"degraded_disks"` // Missing members (0 means healthy); nil for levels without redundancy
	SyncAction    *string      `json:"sync_action"`    // idle, resync, recover, check, repair, reshape or frozen; nil without redundancy
	SyncProgress  *float64     `json:"sync_progress"`  // Progress of the running sync from 0 to 1; nil when no sync is running
	Members       []RaidMember `json:"members"`        // Member devices and their state
}

func (r RaidData) isMetric() {}

type RaidMember struct {
	Device string `json:"device"` // Member device, e.g. /dev/sda1
	State  string `json:"state"`  // md member state, e.g. in_sync, faulty, spare, write_mostly
}

// GetRaidMetrics reports the state of Linux software RAID (mdadm) arrays
func GetRaidMetrics() (MetricsSlice, []CustomErr) {
	return collectRaidMetrics(sysBlockRoot)
}

// collectRaidMetrics reads every md array below root, which is /sys/block outside tests
func collectRaidMetrics(root string) (MetricsSlice, []CustomErr) {
	arrays := make(MetricsSlice, 0)
	var raidErrs []CustomErr

	mdDirs, err := filepath.Glob(filepath.Join(root, "md*", "md"))
	if err != nil {
		return arrays, []CustomErr{{Metric: []string{"raid"}, Error: err.Error()}}
	}
	sort.Strings(mdDirs)

	for _, mdDir := range mdDirs {
		device := "/dev/" + filepath.Base(filepath.Dir(mdDir))
		array, err := readRaidArray(device, mdDir)
		if err != nil {
			raidErrs = append(raidErrs, CustomErr{
				Metric: []string{"raid"},
				Error:  fmt.Sprintf("failed to read %s: %v", device, err),
			})
			continue
		}
		arrays = append(arrays, array)
	}
	return arrays, raidErrs
}

// readRaidArray reads one array from its sysfs md directory.
// Only array_state is required; the other files do not exist for every level or state.
func readRaidArray(device, mdDir string) (*RaidData, error) {
	state, err := readSysfsValue(filepath.Join(mdDir, "array_state"))
	if err != nil {
		return nil, err
	}

	array := &RaidData{Device: device, State: state, Members: make([]RaidMember, 0)}
	array.Level, _ = readSysfsValue(filepath.Join(mdDir, "level"))
	array.RaidDisks = readSysfsInt(filepath.Join(mdDir, "raid_disks"))
	array.DegradedDisks = readSysfsInt(filepath.Join(mdDir, "degraded"))
	if action, err := readSysfsValue(filepath.Join(mdDir, "sync_action")); err == nil {
		array.SyncAction = &action
	}
	if completed, err := readSysfsValue(filepath.Join(mdDir, "sync_completed")); err == nil {
		array.SyncProgress = parseSyncCompleted(completed)
	}

	memberDirs, err := filepath.Glob(filepath.Join(mdDir, "dev-*"))
	if err != nil {
		return nil, err
	}
	sort.Strings(memberDirs)
	for _, memberDir := range memberDirs {
		memberState, err := readSysfsValue(filepath.Join(memberDir, "state"))
		if err != nil {
			continue
		}
		array.Members = append(array.Members, RaidMember{
			Device: "/dev/" + strings.TrimPrefix(filepath.Base(memberDir), "dev-"),
			State:  memberState,
		})
	}
	return array, nil
}

// parseSyncCompleted converts sync_completed ("done / total" in sectors, or "none") to a 0-1 fraction
func parseSyncCompleted(value string) *float64 {
	doneText, totalText, found := strings.Cut(value, "/")
	if !found {
		return nil
	}
	done, doneErr := strconv.ParseUint(strings.TrimSpace(doneText), 10, 64)
	total, totalErr := strconv.ParseUint(strings.TrimSpace(totalText), 10, 64)
	if doneErr != nil || totalErr != nil || total == 0 {
		return nil
	}
	progress := float64(done) / float64(total)
	return &progress
}

func readSysfsValue(path string) (string, error) {
	content, err := os.ReadFile(path) // #nosec G304 -- paths are built from sysfs globs, not user input
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%s does not exist", filepath.Base(path))
		}
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// readSysfsInt returns the first number in a sysfs file, or nil when the file is missing
// or does not start with a number. During a reshape raid_disks reads "new (old)", e.g. "4 (3)".
func readSysfsInt(path string) *int {
	value, err := readSysfsValue(path)
	if err != nil {
		return nil
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return nil
	}
	number, err := strconv.Atoi(fields[0])
	if err != nil {
		return nil
	}
	return &number
}
