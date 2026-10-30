//go:build !windows

package main

import (
	"time"

	"github.com/bluewave-labs/capture/internal/server"
)

// isWindowsService always reports false outside Windows.
func isWindowsService() bool {
	return false
}

// redirectLogToEventLog is a no-op outside Windows.
func redirectLogToEventLog() func() {
	return func() {}
}

// runService is a no-op outside Windows.
func runService(_ *server.Server, _ time.Duration) error {
	return nil
}
