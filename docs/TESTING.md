# Testing Process for Capture

This document provides an overview of testing in Capture, covering architecture, unit, integration, and OpenAPI contract testing.

## Table of Contents

<!-- no toc -->
1. [Architecture Testing](#architecture-testing)
2. [Unit Testing](#unit-testing)
3. [Integration Testing](#integration-testing)
   - [3.1 OpenAPI Contract Testing](#openapi-contract-testing)
   - [3.2 Disk FileSystem(FS) Testing](#disk-metrics-filesystem-testing)
4. [Benchmarking](#benchmarking)

## Architecture Testing

Architecture testing ensures that the system's architecture meets the specified requirements and is robust against potential issues. This includes verifying the design patterns, module interactions, and overall system structure.

You can see the `test/arch_test.go` file for the architecture tests. It's powered by the [go-arctest](https://github.com/mstrYoda/go-arctest) package, which provides tools for testing the architecture of Go applications.

We don't have a dedicated command for architecture tests; `just unit-test` runs all unit tests in the codebase, including the architecture tests.

### Rules

1. `cmd` must not depend on `handlers`.

## Unit Testing

Unit tests are not os/arch specific, they can be run on any platform where the codebase is runnable.

You can search for `test/*_test.go` files in the `test` directory to find unit tests.

`just unit-test` command runs all unit tests in the codebase.

## Integration Testing

Integration tests are designed to test the interactions between different components of the system. They ensure that the components work together as expected and can handle real-world scenarios.

You can see the `test/integration/*_test.go` files for integration tests.

`just integration-test` command runs all integration tests in the codebase.

### OpenAPI Contract Testing

OpenAPI contract testing ensures that the API adheres to the defined OpenAPI specifications. This is crucial for maintaining consistency and reliability in API interactions.

You can see the `schemathesis.toml` file for OpenAPI contract tests.

`just openapi-contract-test` command runs OpenAPI contract tests using [Schemathesis](https://schemathesis.readthedocs.io/).

Prerequisites:

- Ensure the API server is running and reachable.
- Export API_SECRET environment variable with the API secret key before running the tests.

```bash
export API_SECRET=your_api_secret_key
just openapi-contract-test
```

or

```bash
API_SECRET=your_api_secret_key just openapi-contract-test
```

### Disk Metrics FileSystem Testing

#### Prerequisites

1. **GNU/Linux**: The test relies on Linux-specific tools (e.g. `losetup`, `mkfs`, `zpool`), if it's run on another OS then tests will be skipped.
2. **Root Privileges**: The test requires root permissions to create loop devices, manage LVM, and create filesystems. If not run as root, tests will be skipped.
3. **Min. 250 MiB Free Disk Space**: The test creates temporary files and loop devices that require disk space. If insufficient space is detected, tests will fail with an error.

##### Required Packages (Debian/Ubuntu)

| Tool                               | Package                                      | Filesystems        |
| ---------------------------------- | -------------------------------------------- | ------------------ |
| `mkfs.ext4`                        | `e2fsprogs`                                  | ext4               |
| `mkfs.xfs`                         | `xfsprogs`                                   | xfs                |
| `mkfs.btrfs`                       | `btrfs-progs`                                | btrfs              |
| `zpool`                            | `zfsutils-linux`                             | zfs                |
| `pvcreate`, `vgcreate`, `lvcreate` | `lvm2`                                       | all (LVM strategy) |
| `losetup`, `dd`                    | `mount`, `coreutils` (usually pre-installed) | all                |

```bash
# Install all optional filesystem tools
sudo apt update && sudo apt install -y e2fsprogs xfsprogs btrfs-progs zfsutils-linux lvm2
```

#### Test Matrix

The test iterates over 8 combinations (4 filesystems x 2 strategies):

| Filesystem | Strategy |
| ---------- | -------- |
| ext4       | LVM      |
| ext4       | Direct   |
| xfs        | LVM      |
| xfs        | Direct   |
| btrfs      | LVM      |
| btrfs      | Direct   |
| zfs        | LVM      |
| zfs        | Direct   |

#### Expected Cases

Reported sizes are not round numbers because filesystem formatting consumes part of the raw device before any data is written.

##### ext4 (100 MB device → ~88 MB total)

- **~5% reserved blocks** for root (`mkfs.ext4` default) ≈ 5 MB
- **Metadata** (superblock, inode table, journal) ≈ 6–7 MB
- Result: ~88 MB usable total

##### ZFS (100 MB device → ~40 MB total)

- ZFS has significantly heavier overhead on small pools: pool metadata, ZFS Intent Log (ZIL), uber-blocks, MOS (meta-object set), and internal padding consume nearly 60% of a 100 MB device
- This overhead is proportionally smaller as pool size grows

##### Used vs Free vs Total

`used + free` does not equal `total` exactly because **reserved-for-root blocks** (ext4) are counted in neither `used` nor `free`. The 10% sanity check in the assertions accounts for this discrepancy.

#### Per-Combination Steps

For each combination the following steps run sequentially:

##### 1. Check Required Commands

- Verify the filesystem tool is available (`mkfs.ext4`, `mkfs.xfs`, `mkfs.btrfs`, or `zpool`)
- If LVM strategy, verify `pvcreate`, `vgcreate`, `lvcreate` are available
- Verify `losetup` and `dd` are available
- Skip the subtest if any command is missing

##### 2. Provision Storage

**LVM Strategy:**

1. Create a **200 MB** sparse backing image via `truncate`
2. Attach it as a loop device via `losetup --find --show`
3. Create a Physical Volume on the loop device (`pvcreate`)
4. Create a Volume Group (`vgcreate`)
5. Create a **100 MB** Logical Volume (`lvcreate`)

**Direct Strategy:**

1. Create a **100 MB** sparse backing image via `truncate`
2. Attach it as a loop device via `losetup --find --show`
3. Use the loop device directly as the target device

##### 3. Format & Mount

- **ext4 / xfs / btrfs:** Run `mkfs.<fs>` on the device, then `mount` it to a temp directory
- **zfs:** Run `zpool create` with the mount point and device

##### 4. Write Test Data

- Write a **30 MB** deterministic file (`/dev/zero`) to the mount point using `dd`

##### 5. Collect Metrics

1. Call `CollectDiskMetrics()` and search for the mount point in the results
2. If found with all fields non-nil, use those values
3. Otherwise fall back to `gopsutil/disk.Usage()` for the same mount point

##### 6. Validate Metrics

| Assertion                      | Condition                                 |
| ------------------------------ | ----------------------------------------- |
| TotalBytes is positive         | `totalBytes > 0`                          |
| FreeBytes is positive          | `freeBytes > 0`                           |
| UsedBytes reflects the write   | `usedBytes >= 30 MB`                      |
| UsedBytes within bounds        | `usedBytes <= totalBytes`                 |
| UsagePercent in valid range    | `0 <= usagePct <= 1`                      |
| UsagePercent is plausible      | `usagePct >= 0.15`                        |
| Used + Free approximates Total | `abs(used + free - total) / total <= 10%` |

##### 7. Cleanup (deferred, runs in reverse order)

1. **Unmount** the filesystem (`umount`) or **destroy** the ZFS pool (`zpool destroy`)
2. **Remove LVM** stack: `lvremove` → `vgremove` → `pvremove`
3. **Detach** loop device (`losetup -d`)
4. **Delete** backing image file

Cleanup runs even if the test fails or panics (registered via `defer`). Errors during cleanup are silently ignored to avoid masking the original failure.

## Benchmarking

Benchmarking is an essential part of testing to ensure that the system performs well under various conditions. It helps identify performance bottlenecks and areas for optimization.

You can see the `test/benchmark/*_test.go` files for benchmarking tests.

We don't have a dedicated command for benchmarking tests; you can run them manually using the `go test` command:

```bash
go test -benchmem -run='^$' \
    -bench . \
    -count 10 \
    ./test/benchmark | tee my_benchmark_result.txt
```

You can also profile the tests using the `-cpuprofile` and `-memprofile` flags:

```bash
go test -benchmem -run='^$' \
    -bench . \
    -cpuprofile=cpu.prof \
    -memprofile=mem.prof \
    ./test/benchmark
```
