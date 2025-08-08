package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bluewave-labs/capture/internal/config"
	"github.com/bluewave-labs/capture/internal/server"
	"github.com/bluewave-labs/capture/internal/server/handler"
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

	srv := server.NewServer(appConfig, nil, &handler.CaptureMeta{
		Version: Version,
	})
	log.Println("WARNING: Remember to add http://" + server.GetLocalIP() + ":" + appConfig.Port + "/api/v1/metrics to your Checkmate Infrastructure Dashboard. Without this endpoint, system metrics will not be displayed.")

	srv.Serve()

	srv.GracefulShutdown(5 * time.Second)
}
