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

// Application configuration variable that holds the settings.
//
//   - Port: Server port, default is 59232
//
//   - APISecret: Secret key for API access, default is a blank string.
var appConfig *config.Config

// Build information variables populated at build time.
// These variables are typically set using ldflags during the build process
// to provide runtime access to build metadata.
var (
	// Version represents the Capture version (default: "develop").
	Version = "develop"
	// Commit contains the Git commit hash of the build.
	Commit = "unknown"
	// CommitDate holds the date of the commit.
	CommitDate = "unknown"
	// CompiledAt stores the build compilation date.
	CompiledAt = "unknown"
	// GitTag contains the Git tag associated with the build, if any.
	GitTag = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "Display the version of the capture")
	flag.Parse()

	// Check if the version flag is provided and show build information
	if *showVersion {
		fmt.Println("Capture Build Information")
		fmt.Println("-------------------------")
		fmt.Printf("Version          : %s\n", Version)
		fmt.Printf("Commit Hash      : %s\n", Commit)
		fmt.Printf("Commit Date      : %s\n", CommitDate)
		fmt.Printf("Compiled At      : %s\n", CompiledAt)
		fmt.Printf("Git Tag          : %s\n", GitTag)
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
