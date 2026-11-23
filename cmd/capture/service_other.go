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

// runService is never called outside Windows, since isWindowsService is always false.
func runService(_ func() (*server.Server, error), _ time.Duration) int {
	return 1
}
