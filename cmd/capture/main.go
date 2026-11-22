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
)

// Application configuration variable that holds the settings.
//
//   - Listen: Server bind address, default is 0.0.0.0
//
//   - Port: Server port, default is 59232
//
//   - APISecret: Secret key for API access, default is a blank string.
var appConfig *config.Config

// shutdownTimeout is how long the server waits for in-flight requests when stopping.
const shutdownTimeout = 5 * time.Second

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

	// Under the Windows Service Control Manager, startServer runs only after the
	// service has connected, so a startup error is reported to Windows as a stop
	// with an error code instead of a process that never answered.
	if isWindowsService() {
		os.Exit(runService(startServer, shutdownTimeout))
	}

	srv, err := startServer()
	if err != nil {
		log.Fatalln(err)
	}
	srv.GracefulShutdown(shutdownTimeout)
}

// startServer reads the configuration from the environment and starts serving.
// It returns an error for invalid configuration or when the listen address cannot be bound.
func startServer() (*server.Server, error) {
	cfg, err := config.Load(
		os.Getenv("PORT"),
		os.Getenv("API_SECRET"),
		os.Getenv("LISTEN"),
	)
	if err != nil {
		return nil, err
	}
	appConfig = cfg

	if err := metric.ConfigureSmartDevices(os.Getenv("SMART_DEVICES")); err != nil {
		return nil, err
	}

	srv := server.NewServer(appConfig, nil, &handler.CaptureMeta{
		Version: Version,
	})
	if err := srv.Serve(); err != nil {
		return nil, err
	}

	log.Println("WARNING: Remember to add http://" + server.GetLocalIP() + ":" + appConfig.Port + "/api/v1/metrics to your Checkmate Infrastructure Dashboard. Without this endpoint, system metrics will not be displayed.")
	return srv, nil
}
