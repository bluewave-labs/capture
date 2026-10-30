//go:build windows

package main

import (
	"log"
	"strings"
	"time"

	"github.com/bluewave-labs/capture/internal/server"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

// eventSource is the Windows Event Log source Capture writes to when running as a service
const eventSource = "capture"

// isWindowsService reports whether Capture was started by the Windows Service Control Manager.
func isWindowsService() bool {
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Printf("failed to detect Windows service: %v", err)
		return false
	}
	return isService
}

// redirectLogToEventLog sends log output to the Windows Event Log, since a service has no console.
// The returned function closes the event log handle.
func redirectLogToEventLog() func() {
	// Registering fails if the source already exists, which is expected after the first start
	_ = eventlog.InstallAsEventCreate(eventSource, eventlog.Error|eventlog.Warning|eventlog.Info)

	elog, err := eventlog.Open(eventSource)
	if err != nil {
		return func() {}
	}
	log.SetFlags(0)
	log.SetOutput(eventLogWriter{elog: elog})
	return func() { _ = elog.Close() }
}

// eventLogWriter writes each log line to the Windows Event Log
type eventLogWriter struct {
	elog *eventlog.Log
}

func (w eventLogWriter) Write(p []byte) (int, error) {
	message := strings.TrimSpace(string(p))
	lowered := strings.ToLower(message)

	var err error
	switch {
	case strings.Contains(lowered, "error") || strings.Contains(lowered, "required"):
		err = w.elog.Error(1, message)
	case strings.Contains(lowered, "warning"):
		err = w.elog.Warning(1, message)
	default:
		err = w.elog.Info(1, message)
	}
	return len(p), err
}

// runService runs the server under the Service Control Manager until the service is stopped.
func runService(srv *server.Server, shutdownTimeout time.Duration) error {
	// The name is ignored for services that run in their own process
	return svc.Run(eventSource, &captureService{srv: srv, shutdownTimeout: shutdownTimeout})
}

// captureService implements svc.Handler
type captureService struct {
	srv             *server.Server
	shutdownTimeout time.Duration
}

func (s *captureService) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	s.srv.Serve()
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	// Only stop, shutdown and interrogate are accepted, so other commands are never sent
	for request := range requests {
		if request.Cmd == svc.Interrogate {
			status <- request.CurrentStatus
			continue
		}
		if request.Cmd == svc.Stop || request.Cmd == svc.Shutdown {
			status <- svc.Status{State: svc.StopPending}
			s.srv.Stop(s.shutdownTimeout)
			return false, 0
		}
	}
	return false, 0
}
