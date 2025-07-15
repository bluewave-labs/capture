package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bluewave-labs/capture/internal/config"
	"github.com/bluewave-labs/capture/internal/metric"
	"github.com/bluewave-labs/capture/internal/server"
	"github.com/bluewave-labs/capture/internal/server/handler"
	"github.com/joho/godotenv"
)

var appConfig *config.Config

var Version = "develop" // This will be set during compile time using go build ldflags

func init() {
	godotenv.Load()
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

	// Initialize InfluxDB storage
	influxStorage := metric.NewInfluxDBStorage()
	defer influxStorage.Close()

	// Start periodic disk metric collection and storage
	go func() {
		for {
			diskMetrics, _ := metric.CollectDiskMetrics()
			for _, m := range diskMetrics {
				d, ok := m.(*metric.DiskData)
				if !ok {
					continue
				}
				tags := map[string]string{"device": d.Device}
				fields := map[string]interface{}{
					"total_bytes":   derefUint64(d.TotalBytes),
					"used_bytes":    derefUint64(d.UsedBytes),
					"free_bytes":    derefUint64(d.FreeBytes),
					"usage_percent": derefFloat64(d.UsagePercent),
					// add more fields as needed
				}
				err := influxStorage.WriteMetric("disk", tags, fields, time.Now())
				if err != nil {
					log.Println("Failed to write disk metric:", err)
				}
			}
			time.Sleep(10 * time.Second) // or your preferred interval
		}
	}()

	srv := server.NewServer(appConfig, nil, &handler.CaptureMeta{
		Version: Version,
	}, influxStorage)

	log.Println("WARNING: Remember to add http://" + server.GetLocalIP() + ":" + appConfig.Port + "/api/v1/metrics to your Checkmate Infrastructure Dashboard. Without this endpoint, system metrics will not be displayed.")

	srv.Serve()

	srv.GracefulShutdown(5 * time.Second)
}

func derefUint64(p *uint64) uint64 {
	if p != nil {
		return *p
	}
	return 0
}

func derefFloat64(p *float64) float64 {
	if p != nil {
		return *p
	}
	return 0
}
