package config

import "log"

type Config struct {
	Listen    string
	Port      string
	APISecret string
}

var defaultPort = "59232"
var defaultListen = "0.0.0.0"

func NewConfig(port string, apiSecret string, listen string) *Config {
	if listen == "" {
		listen = defaultListen
	}

	// Set default port if not provided
	if port == "" {
		port = defaultPort
	}

	// Print error message if API_SECRET is not provided
	if apiSecret == "" {
		log.Fatalln("API_SECRET environment variable is required for security purposes. Please set it before starting the server.")
	}

	return &Config{
		Listen:    listen,
		Port:      port,
		APISecret: apiSecret,
	}
}

func Default() *Config {
	return &Config{
		Listen:    defaultListen,
		Port:      defaultPort,
		APISecret: "",
	}
}
