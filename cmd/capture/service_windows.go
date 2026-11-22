//go:build windows

package main

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/bluewave-labs/capture/internal/server"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

// eventSource is the Windows Event Log source Capture writes to when running as a service
const eventSource = "capture"

// Event IDs written to the Application log
const (
	eventIDInfo  = 1
	eventIDError = 2
)

// exitCodeStartFailed is the service-specific exit code reported when Capture cannot start.
// Windows treats a stop with an exit code as a failed start rather than a crash, so the
// restart actions set by install.ps1 do not retry a configuration error forever.
const exitCodeStartFailed = 1

// startWaitHint tells the Service Control Manager how long starting may take
const startWaitHint = 10 * time.Second

// isWindowsService reports whether Capture was started by the Windows Service Control Manager.
func isWindowsService() bool {
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Printf("failed to detect Windows service: %v", err)
		return false
	}
	return isService
}

// runService runs Capture under the Service Control Manager until the service is stopped,
// and returns the process exit code. A service has no console, so log output goes to the
// Windows Event Log.
func runService(start func() (*server.Server, error), shutdownTimeout time.Duration) int {
	elog := openEventLog()
	defer elog.close()
	log.SetFlags(0)
	log.SetOutput(elog)

	service := &captureService{start: start, shutdownTimeout: shutdownTimeout, elog: elog}
	// The name is ignored for services that run in their own process
	if err := svc.Run(eventSource, service); err != nil {
		elog.logError(fmt.Sprintf("Capture could not run as a Windows service: %v", err))
		return 1
	}
	if service.startFailed {
		return 1
	}
	return 0
}

// captureService implements svc.Handler
type captureService struct {
	start           func() (*server.Server, error)
	shutdownTimeout time.Duration
	elog            eventLog
	startFailed     bool
}

func (s *captureService) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending, WaitHint: durationMillis(startWaitHint)}

	// Running is reported only once the configuration is valid and the port is bound
	srv, err := s.start()
	if err != nil {
		s.startFailed = true
		s.elog.logError(fmt.Sprintf("Capture could not start: %v", err))
		return true, exitCodeStartFailed
	}
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	// Only stop, shutdown and interrogate are accepted, so other commands are never sent
	for request := range requests {
		if request.Cmd == svc.Interrogate {
			status <- request.CurrentStatus
			continue
		}
		if request.Cmd == svc.Stop || request.Cmd == svc.Shutdown {
			status <- svc.Status{State: svc.StopPending, WaitHint: durationMillis(s.shutdownTimeout + 5*time.Second)}
			srv.Stop(s.shutdownTimeout)
			return false, 0
		}
	}
	return false, 0
}

// durationMillis converts a duration to the milliseconds the Service Control Manager expects,
// clamped to the uint32 range
func durationMillis(d time.Duration) uint32 {
	millis := d.Milliseconds()
	if millis <= 0 {
		return 0
	}
	if millis > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(millis)
}

// eventLog writes to the Application log under the capture source.
// If the log cannot be opened, messages are dropped, since a service has nowhere else to write.
type eventLog struct {
	log *eventlog.Log
}

func openEventLog() eventLog {
	// Registering fails if the source already exists, which is expected after the first start
	_ = eventlog.InstallAsEventCreate(eventSource, eventlog.Error|eventlog.Warning|eventlog.Info)

	elog, err := eventlog.Open(eventSource)
	if err != nil {
		return eventLog{}
	}
	return eventLog{log: elog}
}

// Write logs each line from the standard logger as an information event
func (e eventLog) Write(p []byte) (int, error) {
	if e.log != nil {
		_ = e.log.Info(eventIDInfo, strings.TrimSpace(string(p)))
	}
	return len(p), nil
}

// logError logs an error event
func (e eventLog) logError(message string) {
	if e.log != nil {
		_ = e.log.Error(eventIDError, message)
	}
}

func (e eventLog) close() {
	if e.log != nil {
		_ = e.log.Close()
	}
}
