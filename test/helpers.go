package test

import (
	"os"
	"testing"
)

// SkipIfCI skips the current test if running in a CI environment and the given condition is true.
// It checks if the CI environment variable is set to "true" and if the provided condition pointer
// is not nil and points to true. If both conditions are met, the test is skipped with the provided message.
func SkipIfCI(t *testing.T, condition *bool, message string) {
	if os.Getenv("CI") == "true" && condition != nil && *condition {
		t.Skip(message)
	}
}
