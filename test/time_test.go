package test

import (
	"testing"

	"github.com/bluewave-labs/capture/internal/metric"
)

func TestGetUnixTimestamp(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"2023-01-01T00:00:00Z", 1672531200},           // Example with seconds
		{"2025-06-13T20:00:49.097168933Z", 1749844849}, // Example with nanoseconds
		{"invalid-timestamp", 0},                       // Invalid timestamp should return 0
		{"0001-01-01T00:00:00Z", 0},                    // Special case for zero timestamp
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := metric.GetUnixTimestamp(test.input)
			if result != test.expected {
				t.Errorf("expected %d, got %d", test.expected, result)
			}
		})
	}
}
