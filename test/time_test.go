package test

import (
	"testing"

	"github.com/bluewave-labs/capture/internal/metric"
)

// TestGetUnixTimestamp tests the metric.GetUnixTimestamp function with various timestamp formats.
// It verifies that the function correctly converts RFC3339 timestamps to Unix timestamps
// and handles edge cases like invalid input and zero timestamps.
func TestGetUnixTimestamp(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"2023-01-01T00:00:00Z", 1672531200},           // Valid RFC3339 timestamp with seconds precision
		{"2025-06-13T20:00:49.097168933Z", 1749844849}, // Valid RFC3339 timestamp with nanosecond precision
		{"invalid-timestamp", 0},                       // Invalid timestamp format should return 0
		{"0001-01-01T00:00:00Z", 0},                    // Zero time value should return 0
	}

	// Run each test case as a subtest for better isolation and reporting
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := metric.GetUnixTimestamp(test.input)
			if result != test.expected {
				t.Errorf("expected %d, got %d", test.expected, result)
			}
		})
	}
}
