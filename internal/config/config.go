package config

type Config struct {
	Port string
}

func NewConfig(port string) *Config {
	// Set default port if not provided
	if port == "" {
		port = "3000"
	}

	return &Config{
		Port: port,
	}
}
