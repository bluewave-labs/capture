package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
