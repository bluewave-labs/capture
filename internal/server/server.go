package server

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bluewave-labs/capture/internal/config"
	"github.com/bluewave-labs/capture/internal/server/handler"
	"github.com/bluewave-labs/capture/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type Server struct {
	*http.Server
	MetaData *handler.CaptureMeta // Metadata can be used to store additional information about the server
}

// Serve binds the listen address, then serves requests in the background.
// A bind failure, such as the port already being in use, is returned rather than
// logged later, so the caller knows whether the server actually started.
// On success s.Addr holds the bound address, which matters when the port is 0.
func (s *Server) Serve() error {
	configured := s.Addr
	listener, err := net.Listen("tcp", configured)
	if err != nil {
		return err
	}
	s.Addr = listener.Addr().String()

	go func() {
		if err := s.Server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()
	log.Printf("server started on %s", listenAddrForLog(configured, s.Addr))
	return nil
}

// listenAddrForLog combines the configured host with the bound port. The bound address alone
// can mislead: Go binds 0.0.0.0 as the dual-stack wildcard [::], which reads as IPv6 only.
// The bound port still matters when the configured port is 0.
func listenAddrForLog(configured, bound string) string {
	host, _, err := net.SplitHostPort(configured)
	if err != nil {
		return bound
	}
	_, port, err := net.SplitHostPort(bound)
	if err != nil {
		return bound
	}
	return net.JoinHostPort(host, port)
}

// GracefulShutdown waits for an interrupt or terminate signal, then stops the server.
func (s *Server) GracefulShutdown(timeout time.Duration) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Printf("signal received: %v", sig)

	s.Stop(timeout)
}

// Stop gracefully shuts down the server, closing any connections still open after the timeout.
func (s *Server) Stop(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Println("shutting down server...")
	if err := s.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
		_ = s.Close()
	} else {
		log.Println("server shutdown gracefully")
	}
}

func InitializeHandler(config *config.Config, metadata *handler.CaptureMeta) http.Handler {
	// Initialize the Gin with default middlewares
	r := gin.Default()
	// Return 405 instead of 404 for unsupported methods on known routes
	r.HandleMethodNotAllowed = true
	metadata.Mode = gin.Mode()
	if gin.Mode() == gin.ReleaseMode {
		println("running in Release Mode")
	} else {
		println("running in Debug Mode")
	}
	// Health Check
	r.GET("/health", handler.Health)

	apiV1 := r.Group("/api/v1")
	apiV1.Use(middleware.AuthRequired(config.APISecret))

	// Create metrics handler
	metricsHandler := handler.NewMetricsHandler(metadata)

	// Metrics
	apiV1.GET("/metrics", metricsHandler.Metrics)
	apiV1.GET("/metrics/cpu", metricsHandler.MetricsCPU)
	apiV1.GET("/metrics/memory", metricsHandler.MetricsMemory)
	apiV1.GET("/metrics/disk", metricsHandler.MetricsDisk)
	apiV1.GET("/metrics/host", metricsHandler.MetricsHost)
	apiV1.GET("/metrics/smart", metricsHandler.SmartMetrics)
	apiV1.GET("/metrics/net", metricsHandler.MetricsNet)
	apiV1.GET("/metrics/raid", metricsHandler.MetricsRaid)
	apiV1.GET("/metrics/docker", metricsHandler.MetricsDocker)

	return r.Handler()
}

func NewServer(config *config.Config, handler http.Handler, metadata *handler.CaptureMeta) *Server {
	if handler == nil {
		handler = InitializeHandler(config, metadata)
	}
	return &Server{
		Server: &http.Server{
			Addr:              net.JoinHostPort(config.Listen, config.Port),
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
		MetaData: metadata,
	}
}
