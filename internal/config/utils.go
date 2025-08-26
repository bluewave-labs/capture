package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// GenerateDefaultConfig creates a default configuration file
func GenerateDefaultConfig(path string) error {
	defaultConfig := `version: 1

# Server configuration for the capture HTTP server
server:
  port: "59232"  # Port to listen on (default: 59232)
  api_secret: ""  # Required: API secret for authentication - SET THIS!

# Targets are the Checkmate instances to which the capture will send data
# Each target can have its own timeout, retry delay, and retry count
targets: []
  # Example target configuration:
  # - name: "My Checkmate Instance"
  #   endpoint: "https://checkmate.example.com/api/v1/metrics"
  #   api_secret: "checkmate-api-secret-here"
  #   timeout: "30s"
  #   retry_delay: "5s"
  #   retry_count: 3

# Global configuration for all targets (applied when target-specific settings are not provided)
global_interval: "1m"      # How often to collect and send metrics
global_timeout: "30s"      # Request timeout
global_retry_delay: "5s"   # Delay between retries
global_retry_count: 3      # Number of retries for failed requests

# Logging configuration
log_level: "info"  # Options: error, warn, info, debug

# External plugins - scripts or programs that return status codes
plugins: []
  # Example plugin configuration:
  # - name: "disk-health-check"
  #   command: "/usr/local/bin/check-disk-health.sh"
`

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write the default configuration
	if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}

	log.Printf("Default configuration file created at: %s", path)
	log.Printf("Please edit the file and set your API_SECRET before starting the server")

	return nil
}

// ValidateConfigFile validates if a configuration file exists and is readable
func ValidateConfigFile(path string) error {
	if path == "" {
		return fmt.Errorf("config file path is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", path)
	}

	// Try to read and parse the config
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	return nil
}

// PrintConfigSummary prints a summary of the current configuration
func (c *Config) PrintConfigSummary() {
	fmt.Println("=== Capture Configuration Summary ===")
	fmt.Printf("Version: %d\n", c.Version)
	fmt.Printf("Server Port: %s\n", c.Server.Port)
	fmt.Printf("API Secret: %s\n", maskSecret(c.Server.APISecret))
	fmt.Printf("Log Level: %s\n", c.LogLevel)
	fmt.Printf("Global Interval: %s\n", c.GlobalInterval)
	fmt.Printf("Global Timeout: %s\n", c.GlobalTimeout)
	fmt.Printf("Global Retry Count: %d\n", c.GlobalRetryCount)
	fmt.Printf("Global Retry Delay: %s\n", c.GlobalRetryDelay)
	fmt.Printf("Number of Targets: %d\n", len(c.Targets))
	fmt.Printf("Number of Plugins: %d\n", len(c.Plugins))

	if len(c.Targets) > 0 {
		fmt.Println("\nTargets:")
		for i, target := range c.Targets {
			fmt.Printf("  [%d] %s -> %s\n", i+1, target.Name, target.Endpoint)
		}
	}

	if len(c.Plugins) > 0 {
		fmt.Println("\nPlugins:")
		for i, plugin := range c.Plugins {
			fmt.Printf("  [%d] %s: %s\n", i+1, plugin.Name, plugin.Command)
		}
	}
	fmt.Println("======================================")
}

// maskSecret masks a secret string for safe logging
func maskSecret(secret string) string {
	if secret == "" {
		return "<not set>"
	}
	if len(secret) <= 4 {
		return "****"
	}
	return secret[:2] + strings.Repeat("*", len(secret)-4) + secret[len(secret)-2:]
}
