package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluewave-labs/capture/internal/config"
	"github.com/bluewave-labs/capture/internal/server/handler"
	"github.com/gin-gonic/gin"
)

func TestInitializeHandlerMethodHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := InitializeHandler(config.NewConfig("", "test-secret", ""), &handler.CaptureMeta{Version: "test"})

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"unsupported method on known route", http.MethodTrace, "/api/v1/metrics/cpu", http.StatusMethodNotAllowed},
		{"unsupported method on health", http.MethodPost, "/health", http.StatusMethodNotAllowed},
		{"unknown route", http.MethodGet, "/api/v1/unknown", http.StatusNotFound},
		{"known route without token", http.MethodGet, "/api/v1/metrics/cpu", http.StatusUnauthorized},
		{"raid route without token", http.MethodGet, "/api/v1/metrics/raid", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.want {
				t.Errorf("%s %s = %d, expected %d", tt.method, tt.path, rec.Code, tt.want)
			}
		})
	}
}

func TestNewServerListenAddress(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		want   string
	}{
		{"default configuration", config.Default(), "0.0.0.0:59232"},
		{"empty environment", config.NewConfig("", "test-secret", ""), "0.0.0.0:59232"},
		{"IPv4 loopback", config.NewConfig("8080", "test-secret", "127.0.0.1"), "127.0.0.1:8080"},
		{"specific interface", config.NewConfig("", "test-secret", "192.168.1.10"), "192.168.1.10:59232"},
		{"IPv6 loopback", config.NewConfig("8080", "test-secret", "::1"), "[::1]:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.config, http.NewServeMux(), nil)
			if srv.Addr != tt.want {
				t.Errorf("server address = %q, want %q", srv.Addr, tt.want)
			}
		})
	}
}

func TestServeReturnsBindError(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	_, port, _ := net.SplitHostPort(occupied.Addr().String())

	srv := NewServer(config.NewConfig(port, "test-secret", "127.0.0.1"), http.NewServeMux(), nil)
	if err := srv.Serve(); err == nil {
		srv.Stop(time.Second)
		t.Fatalf("Serve on occupied port %s returned nil, want a bind error", port)
	}
}

func TestServeAndStop(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := NewServer(config.NewConfig("0", "test-secret", "127.0.0.1"), mux, nil)

	if err := srv.Serve(); err != nil {
		t.Fatalf("Serve returned %v", err)
	}
	if strings.HasSuffix(srv.Addr, ":0") {
		t.Fatalf("Addr = %q, want the bound port", srv.Addr)
	}

	resp, err := http.Get("http://" + srv.Addr + "/health")
	if err != nil {
		t.Fatalf("request while serving failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status while serving = %d, want 200", resp.StatusCode)
	}

	// A normal stop must not end the process (the serve goroutine used to log.Fatalf here)
	srv.Stop(time.Second)
	time.Sleep(100 * time.Millisecond)

	if resp, err := http.Get("http://" + srv.Addr + "/health"); err == nil {
		resp.Body.Close()
		t.Fatal("request after Stop succeeded, want connection refused")
	}
}

func TestListenAddrForLog(t *testing.T) {
	tests := []struct{ configured, bound, want string }{
		{"0.0.0.0:59232", "[::]:59232", "0.0.0.0:59232"},
		{"127.0.0.1:0", "127.0.0.1:51234", "127.0.0.1:51234"},
		{"[::1]:8080", "[::1]:8080", "[::1]:8080"},
		{"not-an-address", "[::]:59232", "[::]:59232"},
	}
	for _, tt := range tests {
		if got := listenAddrForLog(tt.configured, tt.bound); got != tt.want {
			t.Errorf("listenAddrForLog(%q, %q) = %q, want %q", tt.configured, tt.bound, got, tt.want)
		}
	}
}
