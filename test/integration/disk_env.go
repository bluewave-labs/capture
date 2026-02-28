package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bluewave-labs/capture/test"
)

// DiskEnv holds all provisioned resources for a single filesystem test case
// and provides methods to set up storage, format/mount, write test data, and
// tear everything down.
type DiskEnv struct {
	t           *testing.T
	fs          string // "ext4", "xfs", "btrfs", or "zfs"
	strategy    string // "lvm" or "direct"
	backingFile string // backing image file for the loop device
	loopDev     string // /dev/loopN
	vgName      string // LVM volume group name (lvm strategy only)
	lvName      string // LVM logical volume name (lvm strategy only)
	lvPath      string // /dev/<vgName>/<lvName> (lvm strategy only)
	zpoolName   string // ZFS pool name (zfs only)
	mountPoint  string // directory where the filesystem is mounted
	devicePath  string // device passed to mkfs or zpool create
}

// SetupLoopDevice creates a backing image file of sizeMB and attaches it as a loop device.
func (e *DiskEnv) SetupLoopDevice(sizeMB int) {
	e.t.Helper()

	dir := e.t.TempDir()
	e.backingFile = filepath.Join(dir, "disk.img")
	test.ShellRun(e.t, "truncate", "-s", fmt.Sprintf("%dM", sizeMB), e.backingFile)

	e.loopDev = test.ShellRun(e.t, "losetup", "--find", "--show", e.backingFile)
	e.t.Logf("Loop device: %s (%dMB)", e.loopDev, sizeMB)
}

// SetupLVM initialises an LVM Physical Volume, Volume Group, and Logical Volume
// on the previously created loop device.
func (e *DiskEnv) SetupLVM(lvSizeMB int) {
	e.t.Helper()

	test.ShellRun(e.t, "pvcreate", "-f", e.loopDev)

	e.vgName = fmt.Sprintf("captvg%s%d", e.fs, os.Getpid())
	test.ShellRun(e.t, "vgcreate", e.vgName, e.loopDev)

	e.lvName = fmt.Sprintf("captlv%s", e.fs)
	test.ShellRun(e.t, "lvcreate", "-L", fmt.Sprintf("%dM", lvSizeMB), "-n", e.lvName, e.vgName, "-y")

	e.lvPath = fmt.Sprintf("/dev/%s/%s", e.vgName, e.lvName)
	e.devicePath = e.lvPath
	e.t.Logf("LVM device: %s", e.lvPath)
}

// FormatAndMount formats devicePath with the target filesystem and mounts it.
func (e *DiskEnv) FormatAndMount() {
	e.t.Helper()

	e.mountPoint = e.t.TempDir()

	switch e.fs {
	case "ext4":
		test.ShellRun(e.t, "mkfs.ext4", "-F", e.devicePath)
		test.ShellRun(e.t, "mount", e.devicePath, e.mountPoint)
	case "xfs":
		test.ShellRun(e.t, "mkfs.xfs", "-f", e.devicePath)
		test.ShellRun(e.t, "mount", e.devicePath, e.mountPoint)
	case "btrfs":
		test.ShellRun(e.t, "mkfs.btrfs", "-f", e.devicePath)
		test.ShellRun(e.t, "mount", e.devicePath, e.mountPoint)
	case "zfs":
		e.zpoolName = fmt.Sprintf("captpool%s%d", e.fs, os.Getpid())
		test.ShellRun(e.t, "zpool", "create", "-f", "-m", e.mountPoint, e.zpoolName, e.devicePath)
	}

	e.t.Logf("Mounted %s on %s at %s", e.fs, e.devicePath, e.mountPoint)
}

// WriteTestData writes a deterministic file of sizeMB to the mount point using dd.
func (e *DiskEnv) WriteTestData(sizeMB int) {
	e.t.Helper()

	target := filepath.Join(e.mountPoint, "testdata.bin")
	test.ShellRun(e.t, "dd",
		"if=/dev/zero",
		fmt.Sprintf("of=%s", target),
		"bs=1M",
		fmt.Sprintf("count=%d", sizeMB),
		"conv=fdatasync",
	)
}

// Cleanup releases all provisioned resources in reverse order. It is safe to
// call even if provisioning was only partially completed.
func (e *DiskEnv) Cleanup() {
	// 1. Unmount / destroy ZFS pool
	if e.fs == "zfs" && e.zpoolName != "" {
		test.ShellRun(nil, "zpool", "destroy", "-f", e.zpoolName)
	} else if e.mountPoint != "" {
		test.ShellRun(nil, "umount", "-f", e.mountPoint)
	}

	// 2. Remove LVM stack
	if e.lvPath != "" {
		test.ShellRun(nil, "lvremove", "-f", e.lvPath)
	}
	if e.vgName != "" {
		test.ShellRun(nil, "vgremove", "-f", e.vgName)
	}
	if e.loopDev != "" && e.strategy == "lvm" {
		test.ShellRun(nil, "pvremove", "-f", e.loopDev)
	}

	// 3. Detach loop device
	if e.loopDev != "" {
		test.ShellRun(nil, "losetup", "-d", e.loopDev)
	}

	// 4. Remove backing image file (also removed by t.TempDir cleanup)
	if e.backingFile != "" {
		os.Remove(e.backingFile)
	}

	e.t.Logf("Cleanup completed for %s/%s", e.fs, e.strategy)
}
