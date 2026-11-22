package metric

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseScanOutput(t *testing.T) {
	output := `/dev/sdb -d scsi # /dev/sdb, SCSI device
/dev/sdc -d sat # /dev/sdc [SAT], ATA device
/dev/nvme0 -d nvme # /dev/nvme0, NVMe device
/dev/bus/0 -d megaraid,0 # /dev/bus/0 [megaraid_disk_00], SCSI device
# a comment line

`
	expected := []smartDevice{
		{Path: "/dev/sdb"},
		{Path: "/dev/sdc", Type: "sat"},
		{Path: "/dev/nvme0", Type: "nvme"},
		{Path: "/dev/bus/0", Type: "megaraid,0"},
	}

	if got := parseScanOutput(output); !reflect.DeepEqual(got, expected) {
		t.Errorf("parseScanOutput() = %+v, expected %+v", got, expected)
	}
}

func TestParseSmartDevices(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []smartDevice
	}{
		{"empty", "", nil},
		{"path with type", "/dev/sdb:sat", []smartDevice{{Path: "/dev/sdb", Type: "sat"}}},
		{"path without type", "/dev/nvme0", []smartDevice{{Path: "/dev/nvme0"}}},
		{
			"multiple with spaces and empty entries",
			" /dev/sdb:sat , ,/dev/nvme0",
			[]smartDevice{{Path: "/dev/sdb", Type: "sat"}, {Path: "/dev/nvme0"}},
		},
		{
			"type containing a comma",
			"/dev/bus/0:megaraid,0,/dev/bus/0:megaraid,1,/dev/sdb",
			[]smartDevice{{Path: "/dev/bus/0", Type: "megaraid,0"}, {Path: "/dev/bus/0", Type: "megaraid,1"}, {Path: "/dev/sdb"}},
		},
		{
			"type with subtype",
			"/dev/bus/0:sat+megaraid,0",
			[]smartDevice{{Path: "/dev/bus/0", Type: "sat+megaraid,0"}},
		},
		{
			"path containing colons with type",
			"/dev/disk/by-path/pci-0000:00:17.0-ata-1:sat",
			[]smartDevice{{Path: "/dev/disk/by-path/pci-0000:00:17.0-ata-1", Type: "sat"}},
		},
		{
			"path containing colons without type",
			"/dev/disk/by-path/pci-0000:00:17.0-ata-1",
			[]smartDevice{{Path: "/dev/disk/by-path/pci-0000:00:17.0-ata-1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSmartDevices(tt.value)
			if err != nil {
				t.Fatalf("parseSmartDevices(%q) error = %v", tt.value, err)
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseSmartDevices(%q) = %+v, expected %+v", tt.value, got, tt.expected)
			}
		})
	}
}

func TestParseSmartDevicesInvalid(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"entry that is not a path", "-a,/dev/sda:sat"},
		{"path missing its leading slash", "/dev/sdb:sat,sdc:sat"},
		{"continuation after a path without type", "/dev/sdb,0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := parseSmartDevices(tt.value); err == nil {
				t.Errorf("parseSmartDevices(%q) = %+v, expected an error", tt.value, got)
			}
		})
	}
}

func TestMergeDevices(t *testing.T) {
	identity := func(path string) string { return path }
	tests := []struct {
		name      string
		scanned   []smartDevice
		overrides []smartDevice
		resolve   func(string) string
		expected  []smartDevice
	}{
		{
			name:     "no overrides",
			scanned:  []smartDevice{{Path: "/dev/sdb"}},
			expected: []smartDevice{{Path: "/dev/sdb"}},
		},
		{
			name:      "override type and add unscanned device",
			scanned:   []smartDevice{{Path: "/dev/sdb"}, {Path: "/dev/nvme0", Type: "nvme"}},
			overrides: []smartDevice{{Path: "/dev/sdb", Type: "sat"}, {Path: "/dev/sdz", Type: "sat"}},
			expected: []smartDevice{
				{Path: "/dev/nvme0", Type: "nvme"},
				{Path: "/dev/sdb", Type: "sat"},
				{Path: "/dev/sdz", Type: "sat"},
			},
		},
		{
			name:      "shared RAID path is replaced as a whole",
			scanned:   []smartDevice{{Path: "/dev/bus/0", Type: "megaraid,0"}, {Path: "/dev/bus/0", Type: "megaraid,1"}},
			overrides: []smartDevice{{Path: "/dev/bus/0", Type: "sat+megaraid,0"}, {Path: "/dev/bus/0", Type: "sat+megaraid,1"}},
			expected:  []smartDevice{{Path: "/dev/bus/0", Type: "sat+megaraid,0"}, {Path: "/dev/bus/0", Type: "sat+megaraid,1"}},
		},
		{
			name:      "symlinked override replaces the scanned device",
			scanned:   []smartDevice{{Path: "/dev/sda"}},
			overrides: []smartDevice{{Path: "/dev/disk/by-id/ata-disk1", Type: "sat"}},
			resolve: func(path string) string {
				if path == "/dev/disk/by-id/ata-disk1" {
					return "/dev/sda"
				}
				return path
			},
			expected: []smartDevice{{Path: "/dev/disk/by-id/ata-disk1", Type: "sat"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolve := tt.resolve
			if resolve == nil {
				resolve = identity
			}
			if got := mergeDevices(tt.scanned, tt.overrides, resolve); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("mergeDevices() = %+v, expected %+v", got, tt.expected)
			}
		})
	}
}

func int64Ptr(value int64) *int64    { return &value }
func uint64Ptr(value uint64) *uint64 { return &value }

func TestDedupeDrives(t *testing.T) {
	drives := []*SmartData{
		{Device: "/dev/sda", ModelName: "Disk A", SerialNumber: "A1"},
		{Device: "/dev/sdc", ModelName: "Disk A", SerialNumber: "A1"}, // same disk on a second path
		{Device: "/dev/sdb", ModelName: "Disk B", SerialNumber: "A1"}, // same serial, different model
		{Device: "/dev/sdd", Status: SmartStatusStandby},              // no serial
		{Device: "/dev/sde", Status: SmartStatusStandby},              // no serial
	}

	devices := make([]string, 0, len(drives))
	for _, drive := range dedupeDrives(drives) {
		devices = append(devices, drive.Device)
	}
	expected := []string{"/dev/sda", "/dev/sdb", "/dev/sdd", "/dev/sde"}
	if !reflect.DeepEqual(devices, expected) {
		t.Errorf("dedupeDrives() devices = %v, expected %v", devices, expected)
	}
}

func TestSmartctlWarnings(t *testing.T) {
	tests := []struct {
		exitCode int
		expected []string
	}{
		{0, []string{}},
		// Bit 2 alone: a SMART command failed, e.g. an unsupported log on Apple SSDs
		{4, []string{}},
		{8, []string{"disk_failing"}},
		{4 | 16 | 64, []string{"prefail_attribute_at_threshold", "error_log_has_errors"}},
		{32 | 128, []string{"attribute_was_at_threshold", "self_test_log_has_errors"}},
		{-1, []string{}},
	}
	for _, tt := range tests {
		if got := smartctlWarnings(tt.exitCode); !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("smartctlWarnings(%d) = %v, expected %v", tt.exitCode, got, tt.expected)
		}
	}
}

func TestSmartctlArgs(t *testing.T) {
	tests := []struct {
		name     string
		device   smartDevice
		expected []string
	}{
		{"autodetect", smartDevice{Path: "/dev/sdb"}, []string{"--json", "--xall", "--nocheck=standby,0", "/dev/sdb"}},
		{"explicit type", smartDevice{Path: "/dev/sdb", Type: "sat"}, []string{"--json", "--xall", "--nocheck=standby,0", "-d", "sat", "/dev/sdb"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := smartctlArgs(tt.device); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("smartctlArgs() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestParseSmartctlJSON(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		expected SmartData
	}{
		{
			name:    "SATA drive",
			fixture: "testdata/smartctl_sata.json",
			expected: SmartData{
				Type:                     "sat",
				Protocol:                 "ATA",
				Status:                   SmartStatusOK,
				ModelName:                "WDC WD40EFRX-68N32N0",
				SerialNumber:             "WD-TESTSERIAL0001",
				FirmwareVersion:          "82.00A82",
				CapacityBytes:            uint64Ptr(4000787030016),
				Warnings:                 []string{},
				ReallocatedSectors:       int64Ptr(0),
				PendingSectors:           int64Ptr(2),
				OfflineUncorrectable:     int64Ptr(0),
				PowerCycles:              "87",
				PowerOnHours:             "35210",
				SmartOverallHealthResult: "PASSED",
				Temperature:              "34 Celsius",
				UnsafeShutdowns:          "41",
			},
		},
		{
			name:    "NVMe drive",
			fixture: "testdata/smartctl_nvme.json",
			expected: SmartData{
				Type:                     "nvme",
				Protocol:                 "NVMe",
				Status:                   SmartStatusOK,
				ModelName:                "Samsung SSD 980 PRO 1TB",
				SerialNumber:             "S5GXNX0TEST001",
				FirmwareVersion:          "5B2QGXA7",
				CapacityBytes:            uint64Ptr(1000204886016),
				Warnings:                 []string{},
				MediaErrors:              int64Ptr(3),
				AvailableSpare:           "100%",
				AvailableSpareThreshold:  "10%",
				ControllerBusyTime:       "1874",
				CriticalWarning:          "0x00",
				DataUnitsRead:            "21384756",
				DataUnitsWritten:         "30128467",
				HostReadCommands:         "245119874",
				HostWriteCommands:        "401227310",
				PercentageUsed:           "3%",
				PowerCycles:              "412",
				PowerOnHours:             "6120",
				SmartOverallHealthResult: "PASSED",
				Temperature:              "41 Celsius",
				UnsafeShutdowns:          "29",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := os.ReadFile(tt.fixture)
			if err != nil {
				t.Fatalf("failed to read fixture: %v", err)
			}

			got, err := parseSmartctlJSON(output)
			if err != nil {
				t.Fatalf("parseSmartctlJSON() error = %v", err)
			}
			if got == nil || !reflect.DeepEqual(*got, tt.expected) {
				t.Errorf("parseSmartctlJSON() = %+v, expected %+v", got, tt.expected)
			}
		})
	}
}

// TestParseSmartctlJSONAppleNVMe uses real smartctl 7.5 output from an Apple Silicon Mac, serial number replaced
func TestParseSmartctlJSONAppleNVMe(t *testing.T) {
	output, err := os.ReadFile("testdata/smartctl_apple_nvme.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	got, err := parseSmartctlJSON(output)
	if err != nil {
		t.Fatalf("parseSmartctlJSON() error = %v", err)
	}
	if got.Status != SmartStatusOK || got.ModelName != "APPLE SSD AP0512Z" || got.SerialNumber != "APPLETESTSERIAL1" ||
		got.Protocol != "NVMe" || got.SmartOverallHealthResult != "PASSED" || got.Temperature == "" ||
		got.PercentageUsed == "" || got.MediaErrors == nil {
		t.Errorf("parseSmartctlJSON() = %+v", got)
	}
	if got.CapacityBytes != nil {
		t.Errorf("CapacityBytes = %d, expected nil because the drive does not report it", *got.CapacityBytes)
	}
}

func TestParseSmartctlJSONEdgeCases(t *testing.T) {
	t.Run("missing ATA attributes stay nil", func(t *testing.T) {
		got, err := parseSmartctlJSON([]byte(`{"ata_smart_attributes": {"table": [{"id": 9, "raw": {"value": 100}}]}}`))
		if err != nil {
			t.Fatalf("parseSmartctlJSON() error = %v", err)
		}
		if got.ReallocatedSectors != nil || got.PendingSectors != nil || got.OfflineUncorrectable != nil {
			t.Errorf("parseSmartctlJSON() = %+v, expected nil sector counts", got)
		}
	})

	t.Run("SAS grown defect list", func(t *testing.T) {
		got, err := parseSmartctlJSON([]byte(`{"device": {"protocol": "SCSI"}, "smart_status": {"passed": true}, "scsi_grown_defect_list": 12}`))
		if err != nil || got.ReallocatedSectors == nil || *got.ReallocatedSectors != 12 || got.Protocol != "SCSI" {
			t.Errorf("parseSmartctlJSON() = %+v, %v, expected 12 reallocated sectors", got, err)
		}
	})

	t.Run("failed health status", func(t *testing.T) {
		got, err := parseSmartctlJSON([]byte(`{"smart_status": {"passed": false}}`))
		if err != nil || got == nil || got.SmartOverallHealthResult != "FAILED" {
			t.Errorf("parseSmartctlJSON() = %+v, %v, expected FAILED health result", got, err)
		}
	})

	t.Run("device in standby", func(t *testing.T) {
		got, err := parseSmartctlJSON([]byte(`{"smartctl": {"exit_status": 0, "messages": [{"string": "Device is in STANDBY mode, exit(0)", "severity": "information"}]}}`))
		if err != nil || got == nil || got.Status != SmartStatusStandby || got.SmartOverallHealthResult != "" {
			t.Errorf("parseSmartctlJSON() = %+v, %v, expected status standby and no values", got, err)
		}
	})

	t.Run("device without SMART support", func(t *testing.T) {
		got, err := parseSmartctlJSON([]byte(`{"model_name": "QEMU HARDDISK", "smartctl": {"exit_status": 0, "messages": [{"string": "SMART support is: Unavailable - device lacks SMART capability.", "severity": "information"}]}}`))
		if err != nil || got == nil || got.Status != SmartStatusUnsupported || got.ModelName != "QEMU HARDDISK" {
			t.Errorf("parseSmartctlJSON() = %+v, %v, expected status unsupported with the model", got, err)
		}
	})

	t.Run("output that is not JSON", func(t *testing.T) {
		if _, err := parseSmartctlJSON([]byte("smartctl: unrecognized option '--json'")); err == nil {
			t.Errorf("parseSmartctlJSON() expected an error for non-JSON output")
		}
	})
}

func TestSmartctlMessages(t *testing.T) {
	output := []byte(`{"smartctl": {"messages": [{"string": "Smartctl open device: /dev/sdb failed: Permission denied", "severity": "error"}]}}`)
	if got := smartctlMessages(output); got != "Smartctl open device: /dev/sdb failed: Permission denied" {
		t.Errorf("smartctlMessages() = %q", got)
	}
	if got := smartctlMessages([]byte("plain text error\n")); got != "plain text error" {
		t.Errorf("smartctlMessages() = %q, expected plain text fallback", got)
	}
}

func TestSmartctlFailureReason(t *testing.T) {
	exitErr := errors.New("exit status 2")
	tests := []struct {
		name     string
		stdout   string
		stderr   string
		expected string
	}{
		{
			"smartctl older than 7.0",
			"=======> UNRECOGNIZED OPTION: json\nUse smartctl -h to get a usage summary",
			"smartctl: unrecognized option '--json'",
			"smartctl 7.0 or newer is required",
		},
		{
			"permission denied",
			`{"smartctl": {"messages": [{"string": "Smartctl open device: /dev/sdb failed: Permission denied", "severity": "error"}]}}`,
			"",
			"permission denied (try running as root)",
		},
		{
			"messages and stderr are both kept",
			`{"smartctl": {"messages": [{"string": "Smartctl open device: /dev/sdz failed: No such device", "severity": "error"}]}}`,
			"some stderr detail\n",
			"Smartctl open device: /dev/sdz failed: No such device; some stderr detail",
		},
		{"no output falls back to the exit status", "", "", "exit status 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := smartctlFailureReason([]byte(tt.stdout), []byte(tt.stderr), exitErr); got != tt.expected {
				t.Errorf("smartctlFailureReason() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

// fakeSmartctl points smartctlCommand at a shell script for the duration of a test
func fakeSmartctl(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake smartctl is a shell script")
	}
	path := filepath.Join(t.TempDir(), "smartctl")
	// #nosec G306 -- the fake smartctl must be executable; it lives in a per-test temp dir
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o700); err != nil {
		t.Fatalf("failed to write fake smartctl: %v", err)
	}

	command, timeout, waitDelay := smartctlCommand, smartctlTimeout, smartctlWaitDelay
	t.Cleanup(func() { smartctlCommand, smartctlTimeout, smartctlWaitDelay = command, timeout, waitDelay })
	smartctlCommand = path
	smartctlTimeout = 200 * time.Millisecond
	smartctlWaitDelay = 100 * time.Millisecond
}

func TestRunSmartctlTimeout(t *testing.T) {
	fakeSmartctl(t, "sleep 30")

	start := time.Now()
	_, err := runSmartctl("--scan")
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("runSmartctl() returned after %s, expected it to stop at the timeout", elapsed)
	}
	if !errors.Is(err, errSmartctlTimeout) {
		t.Errorf("runSmartctl() error = %v, expected errSmartctlTimeout", err)
	}
}

func TestGetMetricsTimeout(t *testing.T) {
	fakeSmartctl(t, "sleep 30")

	data, errs := getMetrics(smartDevice{Path: "/dev/sdz", Type: "sat"})
	if len(errs) != 1 || !strings.Contains(errs[0].Error, "/dev/sdz") || !strings.Contains(errs[0].Error, "timed out") {
		t.Errorf("getMetrics() errors = %+v, expected a timeout error naming the device", errs)
	}
	// The drive is still listed so it does not look like it disappeared
	if data == nil || data.Device != "/dev/sdz" || data.Type != "sat" || data.Status != SmartStatusError {
		t.Errorf("getMetrics() = %+v, expected the device with status error", data)
	}
}

func TestGetMetricsWarningsAndType(t *testing.T) {
	// Exit status 68: bit 2 (ignored) and bit 6 (error log has errors)
	fakeSmartctl(t, `cat testdata/smartctl_sata.json; exit 68`)
	smartctlTimeout = 10 * time.Second

	data, errs := getMetrics(smartDevice{Path: "/dev/sdf"})
	if len(errs) != 0 {
		t.Fatalf("getMetrics() errors = %+v", errs)
	}
	if data.Device != "/dev/sdf" || data.Type != "sat" || data.Status != SmartStatusOK {
		t.Errorf("getMetrics() = %+v, expected /dev/sdf with the type smartctl detected", data)
	}
	if !reflect.DeepEqual(data.Warnings, []string{"error_log_has_errors"}) {
		t.Errorf("Warnings = %v, expected [error_log_has_errors]", data.Warnings)
	}
}

func TestReadDevicesKeepsOrder(t *testing.T) {
	// The last argument is the device path; answer with a minimal drive named after it
	fakeSmartctl(t, `for last; do :; done; printf '{"serial_number": "%s", "smart_status": {"passed": true}}' "$last"`)
	smartctlTimeout = 10 * time.Second

	devices := make([]smartDevice, 0, smartctlConcurrency*3)
	for i := range smartctlConcurrency * 3 {
		devices = append(devices, smartDevice{Path: fmt.Sprintf("/dev/sd%c", 'a'+i)})
	}
	drives, errs := readDevices(devices)
	if len(errs) != 0 {
		t.Fatalf("readDevices() errors = %+v", errs)
	}
	for i, drive := range drives {
		if drive.Device != devices[i].Path || drive.SerialNumber != devices[i].Path {
			t.Errorf("drive %d = %+v, expected %s", i, drive, devices[i].Path)
		}
	}
}

func TestRunSmartctlOutput(t *testing.T) {
	fakeSmartctl(t, `echo "/dev/sda -d sat # /dev/sda [SAT], ATA device"`)
	// Only the timeout tests need a short limit; starting a shell can take longer on a busy machine
	smartctlTimeout = 10 * time.Second

	out, err := runSmartctl("--scan")
	if err != nil {
		t.Fatalf("runSmartctl() error = %v", err)
	}
	if got := parseScanOutput(string(out)); len(got) != 1 || got[0].Path != "/dev/sda" {
		t.Errorf("parseScanOutput(runSmartctl()) = %+v", got)
	}
}
