package config

import (
	"errors"
	"log"
)

type Config struct {
	Listen    string
	Port      string
	APISecret string
}

var defaultPort = "59232"
var defaultListen = "0.0.0.0"

// ErrMissingAPISecret is returned by Load when no API secret is configured.
var ErrMissingAPISecret = errors.New("API_SECRET environment variable is required for security purposes; set it before starting the server")

// Load builds the configuration, filling in defaults, and reports invalid values as an error.
func Load(port string, apiSecret string, listen string) (*Config, error) {
	if apiSecret == "" {
		return nil, ErrMissingAPISecret
	}
	if listen == "" {
		listen = defaultListen
	}
	if port == "" {
		port = defaultPort
	}
	return &Config{
		Listen:    listen,
		Port:      port,
		APISecret: apiSecret,
	}, nil
}

// NewConfig is Load for callers that cannot handle an error: it exits the process instead.
func NewConfig(port string, apiSecret string, listen string) *Config {
	config, err := Load(port, apiSecret, listen)
	if err != nil {
		log.Fatalln(err)
	}
	return config
}

func Default() *Config {
	return &Config{
		Listen:    defaultListen,
		Port:      defaultPort,
		APISecret: "",
	}
}
