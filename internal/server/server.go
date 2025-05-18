package server

import (
	"context"
	"log"
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
}

// Serve function starts the HTTP server and listens for incoming requests concurrently.
// It uses a goroutine to handle the server's ListenAndServe method, allowing the main thread to continue executing.
func (s *Server) Serve() {
	go func() {
		if err := s.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()
	log.Printf("server started on %s", s.Server.Addr)
}

// Shutdown gracefully shuts down the server with a timeout.
func (s *Server) GracefulShutdown(timeout time.Duration) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Printf("signal received: %v", sig)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Println("shutting down server...")
	if err := s.Server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	} else {
		log.Println("server shutdown gracefully")
	}
}

func InitializeHandler(config *config.Config) http.Handler {
	// Initialize the Gin with default middlewares
	r := gin.Default()
	if gin.Mode() == gin.ReleaseMode {
		println("running in Release Mode")
	} else {
		println("running in Debug Mode")
	}
	apiV1 := r.Group("/api/v1")
	apiV1.Use(middleware.AuthRequired(config.APISecret))

	// Health Check
	apiV1.GET("/health", handler.Health)

	// Metrics
	apiV1.GET("/metrics", handler.Metrics)
	apiV1.GET("/metrics/cpu", handler.MetricsCPU)
	apiV1.GET("/metrics/memory", handler.MetricsMemory)
	apiV1.GET("/metrics/disk", handler.MetricsDisk)
	apiV1.GET("/metrics/host", handler.MetricsHost)
	apiV1.GET("/metrics/smart", handler.SmartMetrics)
	apiV1.GET("/metrics/net", handler.MetricsNet)

	return r.Handler()
}

func NewServer(config *config.Config, handler http.Handler) *Server {
	if handler == nil {
		handler = InitializeHandler(config)
	}
	return &Server{
		Server: &http.Server{
			Addr:              ":" + config.Port,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}
