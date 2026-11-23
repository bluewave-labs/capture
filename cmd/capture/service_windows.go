//go:build windows

package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net"
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

// A failure to bind the listen address is retried for bindRetryWindow before the start fails.
// An automatic service can start at boot before the network is ready, for example before DHCP
// has assigned the LISTEN address, and another process may hold the port for a moment.
const (
	bindRetryWindow   = 60 * time.Second
	bindRetryInterval = 2 * time.Second
)

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

	service := &captureService{
		start:             start,
		shutdownTimeout:   shutdownTimeout,
		bindRetryWindow:   bindRetryWindow,
		bindRetryInterval: bindRetryInterval,
		elog:              elog,
	}
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
	start             func() (*server.Server, error)
	shutdownTimeout   time.Duration
	bindRetryWindow   time.Duration
	bindRetryInterval time.Duration
	elog              eventLog
	startFailed       bool
}

func (s *captureService) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending, WaitHint: durationMillis(startWaitHint)}

	// Running is reported only once the configuration is valid and the port is bound
	srv, stopRequested, err := s.startWithRetry(requests, status)
	if stopRequested {
		return false, 0
	}
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

// startWithRetry starts the server. A configuration error fails at once, while a bind error is
// retried every bindRetryInterval until bindRetryWindow has passed. While it retries, the service
// stays Start pending, reports progress so Windows keeps waiting, and accepts stop, in which case
// stopRequested is true.
func (s *captureService) startWithRetry(requests <-chan svc.ChangeRequest, status chan<- svc.Status) (srv *server.Server, stopRequested bool, err error) {
	deadline := time.Now().Add(s.bindRetryWindow)
	pending := svc.Status{
		State:    svc.StartPending,
		Accepts:  svc.AcceptStop | svc.AcceptShutdown,
		WaitHint: durationMillis(s.bindRetryInterval + startWaitHint),
	}
	for attempt := uint32(1); ; attempt++ {
		srv, err = s.start()
		if err == nil || !isBindError(err) || !time.Now().Before(deadline) {
			return srv, false, err
		}
		if attempt == 1 {
			log.Printf("Capture could not bind its listen address and will retry for up to %s: %v", s.bindRetryWindow, err)
		}
		pending.CheckPoint = attempt
		status <- pending
		if s.waitOrStop(requests, status) {
			return nil, true, nil
		}
	}
}

// waitOrStop waits bindRetryInterval while answering the Service Control Manager.
// It returns true if a stop or shutdown arrived.
func (s *captureService) waitOrStop(requests <-chan svc.ChangeRequest, status chan<- svc.Status) bool {
	timer := time.NewTimer(s.bindRetryInterval)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return false
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				status <- request.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				return true
			}
		}
	}
}

// isBindError reports whether err came from binding the listen address, as opposed to
// invalid configuration
func isBindError(err error) bool {
	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Op == "listen"
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
