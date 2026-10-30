package metric

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
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

func TestMergeSmartData(t *testing.T) {
	sata := &SmartData{
		PowerCycles:              "87",
		PowerOnHours:             "35210",
		SmartOverallHealthResult: "PASSED",
		Temperature:              "34 Celsius",
		UnsafeShutdowns:          "41",
	}
	nvme := &SmartData{
		AvailableSpare:           "100%",
		PercentageUsed:           "3%",
		PowerOnHours:             "6120",
		SmartOverallHealthResult: "PASSED",
		Temperature:              "41 Celsius",
	}

	t.Run("NVMe values are kept on a mixed host", func(t *testing.T) {
		got := mergeSmartData([]*SmartData{sata, nvme})
		expected := SmartData{
			AvailableSpare:           "100%",
			PercentageUsed:           "3%",
			PowerCycles:              "87",
			PowerOnHours:             "6120",
			SmartOverallHealthResult: "PASSED",
			Temperature:              "41 Celsius",
			UnsafeShutdowns:          "41",
		}
		if got != expected {
			t.Errorf("mergeSmartData() = %+v, expected %+v", got, expected)
		}
	})

	t.Run("a failed drive is reported", func(t *testing.T) {
		failed := *sata
		failed.SmartOverallHealthResult = "FAILED"
		if got := mergeSmartData([]*SmartData{nvme, &failed}); got.SmartOverallHealthResult != "FAILED" {
			t.Errorf("mergeSmartData() health = %q, expected FAILED", got.SmartOverallHealthResult)
		}
	})

	t.Run("no results", func(t *testing.T) {
		if got := mergeSmartData(nil); got != (SmartData{}) {
			t.Errorf("mergeSmartData(nil) = %+v, expected empty", got)
		}
	})
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
			if got == nil || *got != tt.expected {
				t.Errorf("parseSmartctlJSON() = %+v, expected %+v", got, tt.expected)
			}
		})
	}

	t.Run("failed health status", func(t *testing.T) {
		got, err := parseSmartctlJSON([]byte(`{"smart_status": {"passed": false}}`))
		if err != nil || got == nil || got.SmartOverallHealthResult != "FAILED" {
			t.Errorf("parseSmartctlJSON() = %+v, %v, expected FAILED health result", got, err)
		}
	})

	t.Run("device in standby has no data", func(t *testing.T) {
		got, err := parseSmartctlJSON([]byte(`{"smartctl": {"exit_status": 0, "messages": [{"string": "Device is in STANDBY mode, exit(0)", "severity": "information"}]}}`))
		if err != nil || got != nil {
			t.Errorf("parseSmartctlJSON() = %+v, %v, expected nil data and no error", got, err)
		}
	})

	t.Run("device without SMART support is an error", func(t *testing.T) {
		got, err := parseSmartctlJSON([]byte(`{"smartctl": {"exit_status": 0, "messages": [{"string": "SMART support is: Unavailable - device lacks SMART capability.", "severity": "information"}]}}`))
		if err == nil || !strings.Contains(err.Error(), "SMART support is: Unavailable") {
			t.Errorf("parseSmartctlJSON() = %+v, %v, expected an error naming the reason", got, err)
		}
	})

	t.Run("no data and no messages is an error", func(t *testing.T) {
		if got, err := parseSmartctlJSON([]byte(`{}`)); err == nil {
			t.Errorf("parseSmartctlJSON() = %+v, expected an error", got)
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
