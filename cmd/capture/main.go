package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bluewave-labs/capture/internal/config"
	"github.com/bluewave-labs/capture/internal/handler"
	"github.com/bluewave-labs/capture/internal/middleware"
	"github.com/gin-gonic/gin"
)

var appConfig *config.Config

var Version = "develop" // This will be set during compile time using go build ldflags

func main() {
	showVersion := flag.Bool("version", false, "Display the version of the capture")
	flag.Parse()

	// Check if the version flag is provided
	if *showVersion {
		fmt.Printf("Capture version: %s\n", Version)
		os.Exit(0)
	}

	appConfig = config.NewConfig(
		os.Getenv("PORT"),
		os.Getenv("API_SECRET"),
	)

	// Initialize the Gin with default middlewares
	r := gin.Default()
	apiV1 := r.Group("/api/v1")
	apiV1.Use(middleware.AuthRequired(appConfig.APISecret))

	// Health Check
	apiV1.GET("/health", handler.Health)

	// Metrics
	apiV1.GET("/metrics", handler.Metrics)
	apiV1.GET("/metrics/cpu", handler.MetricsCPU)
	apiV1.GET("/metrics/memory", handler.MetricsMemory)
	apiV1.GET("/metrics/disk", handler.MetricsDisk)
	apiV1.GET("/metrics/host", handler.MetricsHost)

	server := &http.Server{
		Addr:              ":" + appConfig.Port,
		Handler:           r.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go serve(server)

	if err := gracefulShutdown(server, 5*time.Second); err != nil {
		log.Fatalln("graceful shutdown error", err)
	}
}

func serve(srv *http.Server) {
	srvErr := srv.ListenAndServeTLS("././tls/cert.pem", "././tls/key.pem")
	if srvErr != nil && srvErr != http.ErrServerClosed {
		log.Fatalf("listen error: %s\n", srvErr)
	}
}

func gracefulShutdown(srv *http.Server, timeout time.Duration) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Printf("signal received: %v", sig)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return srv.Shutdown(ctx)
}
