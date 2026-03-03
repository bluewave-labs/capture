package test

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// SkipIfCI skips the current test if running in a CI environment and the given condition is true.
// It checks if the CI environment variable is set to "true" and if the provided condition pointer
// is not nil and points to true. If both conditions are met, the test is skipped with the provided message.
func SkipIfCI(t *testing.T, condition *bool, message string) {
	t.Helper()
	if os.Getenv("CI") == "true" && condition != nil && *condition {
		t.Skip(message)
	}
}

// RequireRoot skips the test if not running as root.
func RequireRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("test requires root privileges (sudo)")
	}
}

// RequireLinux skips the test if not running on Linux.
func RequireLinux(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("test requires linux")
	}
}

// RequireCmd skips the test if the named executable is not in PATH.
func RequireCmd(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("required command %q not found in PATH", name)
	}
}

// ShellRun executes a command and returns the trimmed combined stdout+stderr
// output. When t is non-nil, the test is failed fatally on error. When t is
// nil, errors are silently ignored (useful for cleanup paths where partial
// failure is acceptable).
func ShellRun(t *testing.T, name string, args ...string) string {
	if t != nil {
		t.Helper()
	}
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if t != nil {
			t.Fatalf("command %q failed: %v\noutput: %s",
				name+" "+strings.Join(args, " "), err, out)
		}
		return ""
	}
	return strings.TrimSpace(string(out))
}
