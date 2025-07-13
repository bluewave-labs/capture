package test

import (
	"testing"

	"github.com/bluewave-labs/capture/internal/system"
)

// TestPrettyName tests the functionality of retrieving the pretty name of the system.
// It checks if the pretty name can be fetched without errors and logs the result.
func TestPrettyName(t *testing.T) {
	prettyName, err := system.GetPrettyName()
	if err != nil {
		t.Fatalf("Failed to get pretty name: %v", err)
	}
	t.Logf("Pretty Name: %s", prettyName)
}
