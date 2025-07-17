package test

import (
	"os"
	"testing"
)

func SkipIfCI(t *testing.T, message string) {
	if os.Getenv("CI") == "true" {
		t.Skip(message)
	}
}
