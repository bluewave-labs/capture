package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
version: 1
server:
  port: "8080"
  api_secret: "test-secret"
log_level: "debug"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load the config
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if config.Version != 1 {
		t.Errorf("Expected version 1, got %d", config.Version)
	}

	if config.Server.Port != "8080" {
		t.Errorf("Expected port 8080, got %s", config.Server.Port)
	}

	if config.Server.APISecret != "test-secret" {
		t.Errorf("Expected api_secret 'test-secret', got '%s'", config.Server.APISecret)
	}

	if config.LogLevel != "debug" {
		t.Errorf("Expected log_level 'debug', got '%s'", config.LogLevel)
	}

	// Check defaults
	if config.GlobalTimeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", config.GlobalTimeout)
	}

	if config.GlobalRetryCount != 3 {
		t.Errorf("Expected default retry count 3, got %d", config.GlobalRetryCount)
	}
}

func TestLoadConfigWithTargetsAndPlugins(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
version: 1
server:
  port: "9090"
  api_secret: "test-secret-2"

targets:
  - name: "Test Target"
    endpoint: "https://test.example.com/metrics"
    api_secret: "target-secret"
    timeout: "45s"
    retry_count: 5

global_timeout: "20s"
global_retry_count: 2

plugins:
  - name: "test-plugin"
    command: "/bin/echo test"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load the config
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify targets
	if len(config.Targets) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(config.Targets))
	}

	target := config.Targets[0]
	if target.Name != "Test Target" {
		t.Errorf("Expected target name 'Test Target', got '%s'", target.Name)
	}

	if target.Endpoint != "https://test.example.com/metrics" {
		t.Errorf("Expected target endpoint 'https://test.example.com/metrics', got '%s'", target.Endpoint)
	}

	if target.Timeout != 45*time.Second {
		t.Errorf("Expected target timeout 45s, got %v", target.Timeout)
	}

	if target.RetryCount != 5 {
		t.Errorf("Expected target retry count 5, got %d", target.RetryCount)
	}

	// Verify plugins
	if len(config.Plugins) != 1 {
		t.Fatalf("Expected 1 plugin, got %d", len(config.Plugins))
	}

	plugin := config.Plugins[0]
	if plugin.Name != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got '%s'", plugin.Name)
	}

	if plugin.Command != "/bin/echo test" {
		t.Errorf("Expected plugin command '/bin/echo test', got '%s'", plugin.Command)
	}
}

func TestEnvironmentVariableOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("CAPTURE_SERVER_PORT", "7777")
	os.Setenv("CAPTURE_LOG_LEVEL", "error")
	os.Setenv("API_SECRET", "env-secret") // Legacy env var
	defer func() {
		os.Unsetenv("CAPTURE_SERVER_PORT")
		os.Unsetenv("CAPTURE_LOG_LEVEL")
		os.Unsetenv("API_SECRET")
	}()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
version: 1
server:
  port: "8080"
  api_secret: "file-secret"
log_level: "info"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load the config
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment overrides
	if config.Server.Port != "7777" {
		t.Errorf("Expected port 7777 from env override, got %s", config.Server.Port)
	}

	if config.LogLevel != "error" {
		t.Errorf("Expected log_level 'error' from env override, got '%s'", config.LogLevel)
	}

	if config.Server.APISecret != "env-secret" {
		t.Errorf("Expected api_secret 'env-secret' from legacy env override, got '%s'", config.Server.APISecret)
	}
}

func TestConfigValidation(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	// Test config without API secret
	configContent := `
version: 1
server:
  port: "8080"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load the config - should fail validation
	_, err = LoadConfig(configFile)
	if err == nil {
		t.Fatal("Expected validation error for missing API secret, but got none")
	}

	if !contains(err.Error(), "API_SECRET is required") {
		t.Errorf("Expected validation error about API_SECRET, got: %v", err)
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test the old NewConfig function
	config := NewConfig("9999", "old-secret")

	if config.Server.Port != "9999" {
		t.Errorf("Expected port 9999, got %s", config.Server.Port)
	}

	if config.Server.APISecret != "old-secret" {
		t.Errorf("Expected api_secret 'old-secret', got '%s'", config.Server.APISecret)
	}

	// Test backwards compatibility methods
	if config.Port() != "9999" {
		t.Errorf("Expected Port() to return 9999, got %s", config.Port())
	}

	if config.APISecret() != "old-secret" {
		t.Errorf("Expected APISecret() to return 'old-secret', got '%s'", config.APISecret())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}
