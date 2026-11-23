//go:build windows

package main

import (
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/bluewave-labs/capture/internal/server"
	"golang.org/x/sys/windows/svc"
)

// bindErr is what server.Serve returns when the port is taken or the address does not exist yet
var bindErr = &net.OpError{Op: "listen", Net: "tcp", Err: errors.New("bind: address already in use")}

// bindErrs returns n bind errors, enough to outlast any retry window used in these tests
func bindErrs(n int) []error {
	errs := make([]error, n)
	for i := range errs {
		errs[i] = bindErr
	}
	return errs
}

type executeResult struct {
	specific bool
	code     uint32
	statuses []svc.Status
	calls    int
}

// runExecute runs Execute with a start function that fails with startErrs in turn and then
// succeeds. started is closed when start succeeds.
func runExecute(startErrs []error, window time.Duration, requests chan svc.ChangeRequest, started chan struct{}) executeResult {
	calls := 0
	service := &captureService{
		start: func() (*server.Server, error) {
			calls++
			if calls <= len(startErrs) {
				return nil, startErrs[calls-1]
			}
			close(started)
			return &server.Server{Server: &http.Server{ReadHeaderTimeout: time.Second}}, nil
		},
		shutdownTimeout:   time.Second,
		bindRetryWindow:   window,
		bindRetryInterval: 10 * time.Millisecond,
	}
	status := make(chan svc.Status, 1000)
	specific, code := service.Execute(nil, requests, status)
	close(status)
	result := executeResult{specific: specific, code: code, calls: calls}
	for s := range status {
		result.statuses = append(result.statuses, s)
	}
	return result
}

func TestExecuteRetriesBindErrorUntilItSucceeds(t *testing.T) {
	requests := make(chan svc.ChangeRequest)
	started := make(chan struct{})
	go func() {
		<-started
		requests <- svc.ChangeRequest{Cmd: svc.Stop}
	}()

	result := runExecute(bindErrs(2), time.Minute, requests, started)
	if result.specific || result.code != 0 {
		t.Fatalf("Execute() = %v, %d, expected a clean stop", result.specific, result.code)
	}
	if result.calls != 3 {
		t.Errorf("start called %d times, expected 3", result.calls)
	}
	var checkpoints []uint32
	running := false
	for _, s := range result.statuses {
		if s.State == svc.StartPending && s.CheckPoint > 0 {
			checkpoints = append(checkpoints, s.CheckPoint)
		}
		running = running || s.State == svc.Running
	}
	if len(checkpoints) != 2 || checkpoints[0] != 1 || checkpoints[1] != 2 {
		t.Errorf("start pending checkpoints = %v, expected [1 2]", checkpoints)
	}
	if !running {
		t.Error("service never reported Running")
	}
}

func TestExecuteFailsAfterBindRetryWindow(t *testing.T) {
	result := runExecute(bindErrs(1000), 50*time.Millisecond, make(chan svc.ChangeRequest), make(chan struct{}))
	if !result.specific || result.code != exitCodeStartFailed {
		t.Errorf("Execute() = %v, %d, expected service-specific exit code %d", result.specific, result.code, exitCodeStartFailed)
	}
	if result.calls < 2 {
		t.Errorf("start called %d times, expected retries", result.calls)
	}
}

func TestExecuteDoesNotRetryConfigErrors(t *testing.T) {
	configErr := errors.New("API_SECRET environment variable is required")
	result := runExecute([]error{configErr}, time.Minute, make(chan svc.ChangeRequest), make(chan struct{}))
	if !result.specific || result.code != exitCodeStartFailed {
		t.Errorf("Execute() = %v, %d, expected service-specific exit code %d", result.specific, result.code, exitCodeStartFailed)
	}
	if result.calls != 1 {
		t.Errorf("start called %d times, expected 1", result.calls)
	}
}

func TestExecuteStopsWhileRetrying(t *testing.T) {
	requests := make(chan svc.ChangeRequest, 1)
	requests <- svc.ChangeRequest{Cmd: svc.Stop}
	result := runExecute(bindErrs(1000), time.Minute, requests, make(chan struct{}))
	if result.specific || result.code != 0 {
		t.Errorf("Execute() = %v, %d, expected a clean stop", result.specific, result.code)
	}
	if result.calls != 1 {
		t.Errorf("start called %d times, expected 1 before the stop", result.calls)
	}
}

func TestIsBindError(t *testing.T) {
	if !isBindError(bindErr) {
		t.Error("isBindError(listen error) = false")
	}
	if isBindError(errors.New("SMART_DEVICES is invalid")) {
		t.Error("isBindError(config error) = true")
	}
}
