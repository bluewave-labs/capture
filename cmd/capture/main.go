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
//   - Server: Server configuration including port and API secret
//   - Targets: List of Checkmate instances to send data to
//   - Plugins: List of external plugins to execute
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
	configPath := flag.String("config", "", "Path to configuration file")
	generateConfig := flag.String("generate-config", "", "Generate a default configuration file at the specified path")
	validateConfig := flag.String("validate-config", "", "Validate a configuration file")
	showConfig := flag.Bool("show-config", false, "Show the current configuration and exit")
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

	// Generate default configuration file if requested
	if *generateConfig != "" {
		if err := config.GenerateDefaultConfig(*generateConfig); err != nil {
			log.Fatalf("Failed to generate config file: %v", err)
		}
		fmt.Printf("Default configuration file generated at: %s\n", *generateConfig)
		fmt.Println("Please edit the file and set your API_SECRET before starting the server")
		os.Exit(0)
	}

	// Validate configuration file if requested
	if *validateConfig != "" {
		if err := config.ValidateConfigFile(*validateConfig); err != nil {
			log.Fatalf("Configuration validation failed: %v", err)
		}
		fmt.Printf("Configuration file %s is valid\n", *validateConfig)
		os.Exit(0)
	}

	// Load configuration using Viper
	var err error
	appConfig, err = config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Show configuration summary if requested
	if *showConfig {
		appConfig.PrintConfigSummary()
		os.Exit(0)
	}

	// Log configuration information
	log.Printf("Starting Capture server on port %s", appConfig.Server.Port)
	log.Printf("Configuration version: %d", appConfig.Version)
	log.Printf("Log level: %s", appConfig.LogLevel)
	log.Printf("Number of targets configured: %d", len(appConfig.Targets))
	log.Printf("Number of plugins configured: %d", len(appConfig.Plugins))

	srv := server.NewServer(appConfig, nil, &handler.CaptureMeta{
		Version: Version,
	})
	log.Println("WARNING: Remember to add http://" + server.GetLocalIP() + ":" + appConfig.Server.Port + "/api/v1/metrics to your Checkmate Infrastructure Dashboard. Without this endpoint, system metrics will not be displayed.")

	srv.Serve()

	srv.GracefulShutdown(5 * time.Second)
}
