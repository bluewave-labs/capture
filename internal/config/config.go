package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Target represents a Checkmate instance configuration
type Target struct {
	Name       string        `mapstructure:"name" yaml:"name"`
	Endpoint   string        `mapstructure:"endpoint" yaml:"endpoint"`
	APISecret  string        `mapstructure:"api_secret" yaml:"api_secret"`
	Timeout    time.Duration `mapstructure:"timeout" yaml:"timeout"`
	RetryDelay time.Duration `mapstructure:"retry_delay" yaml:"retry_delay"`
	RetryCount int           `mapstructure:"retry_count" yaml:"retry_count"`
}

// Plugin represents a capture plugin configuration
type Plugin struct {
	Name    string `mapstructure:"name" yaml:"name"`
	Command string `mapstructure:"command" yaml:"command"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Port      string `mapstructure:"port" yaml:"port"`
	APISecret string `mapstructure:"api_secret" yaml:"api_secret"`
}

// Config represents the complete application configuration
type Config struct {
	Version          int           `mapstructure:"version" yaml:"version"`
	Server           ServerConfig  `mapstructure:"server" yaml:"server"`
	Targets          []Target      `mapstructure:"targets" yaml:"targets"`
	GlobalInterval   time.Duration `mapstructure:"global_interval" yaml:"global_interval"`
	GlobalTimeout    time.Duration `mapstructure:"global_timeout" yaml:"global_timeout"`
	GlobalRetryDelay time.Duration `mapstructure:"global_retry_delay" yaml:"global_retry_delay"`
	GlobalRetryCount int           `mapstructure:"global_retry_count" yaml:"global_retry_count"`
	LogLevel         string        `mapstructure:"log_level" yaml:"log_level"`
	Plugins          []Plugin      `mapstructure:"plugins" yaml:"plugins"`
}

// LoadConfig loads configuration from file, environment variables, and sets defaults
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config file path and name
	if configPath != "" {
		// Use specific config file path
		dir := filepath.Dir(configPath)
		filename := filepath.Base(configPath)
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)

		v.AddConfigPath(dir)
		v.SetConfigName(name)
		v.SetConfigType(strings.TrimPrefix(ext, "."))
	} else {
		// Look for config in multiple locations
		v.SetConfigName("capture")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("$HOME/.capture")
		v.AddConfigPath("/etc/capture")
	}

	// Environment variable configuration
	v.SetEnvPrefix("CAPTURE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("No config file found, using defaults and environment variables")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		log.Printf("Using config file: %s", v.ConfigFileUsed())
	}

	// Override with environment variables for backward compatibility
	if port := os.Getenv("PORT"); port != "" {
		v.Set("server.port", port)
	}
	if apiSecret := os.Getenv("API_SECRET"); apiSecret != "" {
		v.Set("server.api_secret", apiSecret)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate required fields
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	v.SetDefault("version", 1)
	v.SetDefault("server.port", "59232")
	v.SetDefault("server.api_secret", "")
	v.SetDefault("global_interval", "1m")
	v.SetDefault("global_timeout", "30s")
	v.SetDefault("global_retry_delay", "5s")
	v.SetDefault("global_retry_count", 3)
	v.SetDefault("log_level", "info")
	v.SetDefault("targets", []Target{})
	v.SetDefault("plugins", []Plugin{})
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.Server.APISecret == "" {
		return fmt.Errorf("API_SECRET is required for security purposes. Please set it in config file or environment variable")
	}

	if config.Server.Port == "" {
		config.Server.Port = "59232"
	}

	// Apply global settings to targets that don't have them set
	for i := range config.Targets {
		target := &config.Targets[i]
		if target.Timeout == 0 {
			target.Timeout = config.GlobalTimeout
		}
		if target.RetryDelay == 0 {
			target.RetryDelay = config.GlobalRetryDelay
		}
		if target.RetryCount == 0 {
			target.RetryCount = config.GlobalRetryCount
		}
	}

	return nil
}

// NewConfig creates a new config with provided values (for backward compatibility)
func NewConfig(port string, apiSecret string) *Config {
	config := &Config{
		Version: 1,
		Server: ServerConfig{
			Port:      port,
			APISecret: apiSecret,
		},
		GlobalInterval:   time.Minute,
		GlobalTimeout:    30 * time.Second,
		GlobalRetryDelay: 5 * time.Second,
		GlobalRetryCount: 3,
		LogLevel:         "info",
		Targets:          []Target{},
		Plugins:          []Plugin{},
	}

	// Set default port if not provided
	if config.Server.Port == "" {
		config.Server.Port = "59232"
	}

	// Print error message if API_SECRET is not provided
	if config.Server.APISecret == "" {
		log.Fatalln("API_SECRET environment variable is required for security purposes. Please set it before starting the server.")
	}

	return config
}

// Default returns a default configuration
func Default() *Config {
	return &Config{
		Version: 1,
		Server: ServerConfig{
			Port:      "59232",
			APISecret: "",
		},
		GlobalInterval:   time.Minute,
		GlobalTimeout:    30 * time.Second,
		GlobalRetryDelay: 5 * time.Second,
		GlobalRetryCount: 3,
		LogLevel:         "info",
		Targets:          []Target{},
		Plugins:          []Plugin{},
	}
}

// Port returns the server port (for backward compatibility)
func (c *Config) Port() string {
	return c.Server.Port
}

// APISecret returns the API secret (for backward compatibility)
func (c *Config) APISecret() string {
	return c.Server.APISecret
}
