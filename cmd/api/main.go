package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bluewave-labs/bluewave-uptime-agent/internal/config"
	"github.com/bluewave-labs/bluewave-uptime-agent/internal/handler"
	"github.com/bluewave-labs/bluewave-uptime-agent/internal/middleware"
	"github.com/gin-gonic/gin"
)

var appConfig = config.NewConfig(
	os.Getenv("PORT"),
	os.Getenv("API_SECRET"),
	os.Getenv("ALLOW_PUBLIC_API"),
)

func main() {
	r := gin.Default()
	apiV1 := r.Group("/api/v1")
	apiV1.Use(middleware.AuthRequired(appConfig.ApiSecret))

	// Health Check
	apiV1.GET("/health", handler.Health)

	// Metrics
	apiV1.GET("/metrics", handler.Metrics)
	apiV1.GET("/metrics/cpu", handler.MetricsCPU)
	apiV1.GET("/metrics/memory", handler.MetricsMemory)
	apiV1.GET("/metrics/disk", handler.MetricsDisk)
	apiV1.GET("/metrics/host", handler.MetricsHost)

	// WebSocket Connection
	apiV1.GET("/ws/metrics", handler.WebSocket)

	server := &http.Server{
		Addr:    ":" + appConfig.Port,
		Handler: r.Handler(),
	}

	// TODO: Add graceful shutdown
	serve(server)

}

func serve(srv *http.Server) {
	srvErr := srv.ListenAndServe()
	if srvErr != nil && srvErr != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", srvErr)
	}
}
