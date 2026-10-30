package metric

import (
	"os"
	"reflect"
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
		{"entry that is not a path", "-a,sda:sat", nil},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseSmartDevices(tt.value); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseSmartDevices(%q) = %+v, expected %+v", tt.value, got, tt.expected)
			}
		})
	}
}

func TestMergeDevices(t *testing.T) {
	tests := []struct {
		name      string
		scanned   []smartDevice
		overrides []smartDevice
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeDevices(tt.scanned, tt.overrides); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("mergeDevices() = %+v, expected %+v", got, tt.expected)
			}
		})
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
		got, err := parseSmartctlJSON([]byte(`{"smartctl": {"exit_status": 0, "messages": [{"string": "Device is in STANDBY mode", "severity": "information"}]}}`))
		if err != nil || got != nil {
			t.Errorf("parseSmartctlJSON() = %+v, %v, expected nil data and no error", got, err)
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
