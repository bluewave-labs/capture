package config

import "log"

type Config struct {
	Port      string
	APISecret string
}

var defaultPort = "59232"

func NewConfig(port string, apiSecret string) *Config {
	// Set default port if not provided
	if port == "" {
		port = defaultPort
	}

	// Print error message if API_SECRET is not provided
	if apiSecret == "" {
		log.Fatalln("API_SECRET environment variable is required for security purposes. Please set it before starting the server.")
	}

	return &Config{
		Port:      port,
		APISecret: apiSecret,
	}
}

func Default() *Config {
	return &Config{
		Port:      defaultPort,
		APISecret: "",
	}
}
