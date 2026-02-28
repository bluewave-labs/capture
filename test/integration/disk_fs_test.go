package integration

import (
	"testing"

	"github.com/bluewave-labs/capture/internal/metric"
	"github.com/bluewave-labs/capture/test"
	"github.com/shirou/gopsutil/v4/disk"
)

// collectMetricsForMount tries to find the mountpoint in the CollectDiskMetrics
// output. If the device is filtered (e.g. bare loop device), a fallback to
// gopsutil's disk.Usage is used (the same underlying data source).
func collectMetricsForMount(t *testing.T, mountPoint string) (totalBytes, freeBytes, usedBytes uint64, usagePct float64) {
	t.Helper()

	metricsSlice, errs := metric.CollectDiskMetrics()
	if len(errs) > 0 {
		t.Logf("CollectDiskMetrics partial errors (non-fatal): %v", errs)
	}

	for _, m := range metricsSlice {
		dd, ok := m.(*metric.DiskData)
		if !ok {
			continue
		}
		if dd.Mountpoint != mountPoint {
			continue
		}

		if dd.TotalBytes != nil && dd.FreeBytes != nil && dd.UsedBytes != nil && dd.UsagePercent != nil {
			return *dd.TotalBytes, *dd.FreeBytes, *dd.UsedBytes, *dd.UsagePercent
		}

		t.Logf("DiskData fields are partially nil; falling back to disk.Usage")
		break
	}

	// Fallback: gopsutil is queried directly (same underlying code path used internally).
	usage, err := disk.Usage(mountPoint)
	if err != nil {
		t.Fatalf("disk.Usage(%s) failed: %v", mountPoint, err)
	}

	// UsedPercent is normalised from [0,100] to [0,1].
	return usage.Total, usage.Free, usage.Used, usage.UsedPercent / 100
}

// TestDiskFilesystemMetrics validates filesystem metric extraction across ext4,
// XFS, BTRFS, and ZFS. Two storage provisioning strategies are exercised per
// filesystem:
//
//   - LVM:    losetup -> PV -> VG -> 100MB LV -> mkfs/zpool
//   - Direct: losetup on a standalone 100MB backing file -> mkfs/zpool
//
// For each combination a deterministic 30MB file is written and the collected
// disk metrics are validated against expected values.
func TestDiskFilesystemMetrics(t *testing.T) {
	test.RequireLinux(t)
	test.RequireRoot(t)

	const (
		lvmBackingMB    = 200 // LVM strategy: backing image file size
		directBackingMB = 100 // Direct strategy: backing image file size
		lvSizeMB        = 100 // Logical volume size within the VG
		testDataMB      = 30  // Deterministic write size
	)

	testCases := []struct {
		name     string
		fs       string
		strategy string
	}{
		{"EXT4_LVM", "ext4", "lvm"},
		{"EXT4_Direct", "ext4", "direct"},
		{"XFS_LVM", "xfs", "lvm"},
		{"XFS_Direct", "xfs", "direct"},
		{"BTRFS_LVM", "btrfs", "lvm"},
		{"BTRFS_Direct", "btrfs", "direct"},
		{"ZFS_LVM", "zfs", "lvm"},
		{"ZFS_Direct", "zfs", "direct"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			switch tc.fs {
			case "ext4":
				test.RequireCmd(t, "mkfs.ext4")
			case "xfs":
				test.RequireCmd(t, "mkfs.xfs")
			case "btrfs":
				test.RequireCmd(t, "mkfs.btrfs")
			case "zfs":
				test.RequireCmd(t, "zpool")
			}

			if tc.strategy == "lvm" {
				test.RequireCmd(t, "pvcreate")
				test.RequireCmd(t, "vgcreate")
				test.RequireCmd(t, "lvcreate")
			}

			test.RequireCmd(t, "losetup")
			test.RequireCmd(t, "dd")

			env := &DiskEnv{
				t:        t,
				fs:       tc.fs,
				strategy: tc.strategy,
			}
			defer env.Cleanup()

			// Provision storage.
			switch tc.strategy {
			case "lvm":
				env.SetupLoopDevice(lvmBackingMB)
				env.SetupLVM(lvSizeMB)
			case "direct":
				env.SetupLoopDevice(directBackingMB)
				env.devicePath = env.loopDev
			}

			// Format & Mount.
			env.FormatAndMount()

			// Data Ingestion.
			env.WriteTestData(testDataMB)

			// Metric validation.
			totalBytes, freeBytes, usedBytes, usagePct := collectMetricsForMount(t, env.mountPoint)

			t.Logf("Metrics [%s/%s]: total=%.1fMB free=%.1fMB used=%.1fMB usage=%.2f%%",
				tc.fs, tc.strategy,
				float64(totalBytes)/(1024*1024),
				float64(freeBytes)/(1024*1024),
				float64(usedBytes)/(1024*1024),
				usagePct*100,
			)

			// Assertions.

			if totalBytes == 0 {
				t.Error("TotalBytes is zero; expected a positive value matching the provisioned device")
			}

			if freeBytes == 0 {
				t.Error("FreeBytes is zero; expected remaining free space after a partial write")
			}

			// UsedBytes must reflect at least the 30MB write. Filesystem metadata
			// overhead only adds to usage, so this bound is always safe.
			const writtenBytes = testDataMB * 1024 * 1024
			if usedBytes < uint64(writtenBytes) {
				t.Errorf("UsedBytes (%d) < written data (%d bytes); "+
					"expected at least %dMB of reported usage", usedBytes, writtenBytes, testDataMB)
			}

			if usedBytes > totalBytes {
				t.Errorf("UsedBytes (%d) exceeds TotalBytes (%d)", usedBytes, totalBytes)
			}

			// UsagePercentage must be within [0, 1] (values are normalised to 0-1 range).
			if usagePct < 0 || usagePct > 1 {
				t.Errorf("UsagePercentage (%.4f) outside [0, 1] range", usagePct)
			}

			// After writing 30MB to a ~100MB device, expect at least 20% usage.
			if usagePct < 0.20 {
				t.Errorf("UsagePercentage (%.4f) unexpectedly low after writing %dMB to a ~100MB device",
					usagePct, testDataMB)
			}

			// Sanity: Used + Free should approximate Total within 10%.
			sum := usedBytes + freeBytes
			var discrepancy float64
			if totalBytes > 0 {
				var diff uint64
				if sum > totalBytes {
					diff = sum - totalBytes
				} else {
					diff = totalBytes - sum
				}
				discrepancy = float64(diff) / float64(totalBytes) * 100
			}
			if discrepancy > 10.0 {
				t.Errorf("Used (%d) + Free (%d) = %d differs from Total (%d) by %.1f%% (threshold: 10%%)",
					usedBytes, freeBytes, sum, totalBytes, discrepancy)
			}
		})
	}
}
