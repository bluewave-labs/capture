package test

import (
	"os"
	"testing"
)

func SkipIfCI(t *testing.T, condition *bool, message string) {
	if os.Getenv("CI") == "true" && condition != nil && *condition {
		t.Skip(message)
	}
}
