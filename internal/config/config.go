package config

import "log"

type Config struct {
	Port      string
	APISecret string
	Proxmox   ProxmoxConfig
}

type ProxmoxConfig struct {
	Host          string
	TokenID       string
	TokenSecret   string
	SkipTLSVerify bool
}

func (p ProxmoxConfig) IsConfigured() bool {
	return p.Host != "" && p.TokenID != "" && p.TokenSecret != ""
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

func NewConfigWithProxmox(port string, apiSecret string, proxmox ProxmoxConfig) *Config {
	cfg := NewConfig(port, apiSecret)
	cfg.Proxmox = proxmox
	return cfg
}

func Default() *Config {
	return &Config{
		Port:      defaultPort,
		APISecret: "",
	}
}
