package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Graceful shutdown
	go serve(server)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("server shutdown:", err)
	}
	select {
	case <-ctx.Done():
		log.Println("timeout of 5 seconds.")
	}
	log.Println("server exiting")
}

func serve(srv *http.Server) {
	srvErr := srv.ListenAndServe()
	if srvErr != nil && srvErr != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", srvErr)
	}
}
