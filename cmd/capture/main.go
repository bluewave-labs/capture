package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
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

// getLocalIP retrieves the local IP address of the machine.
// It returns the first non-loopback IPv4 address found.
// If no valid address is found, it returns "<ip-address>" as a placeholder.
// This function is used to display the local IP address in the log message.
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "<ip-address>"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "<ip-address>"
}

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
	apiV1.GET("/metrics/smart", handler.SmartMetrics)

	log.Println("WARNING: Remember to add http://" + getLocalIP() + ":" + appConfig.Port + "/api/v1/metrics to your Checkmate Infrastructure Dashboard. Without this endpoint, system metrics will not be displayed.")

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
	srvErr := srv.ListenAndServe()
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
