package config

import (
	"errors"
	"log"
)

type Config struct {
	Port           string
	ApiSecret      string
	AllowPublicApi bool
}

var isPublicApiAllowed bool

func NewConfig(port string, apiSecret string, allowPublicApi string) *Config {
	// Set default port if not provided
	if port == "" {
		port = "3000"
	}

	if allowPublicApi == "true" {
		isPublicApiAllowed = true
	} else if allowPublicApi == "false" || allowPublicApi == "" {
		isPublicApiAllowed = false
	} else {
		log.Panic(errors.New("Invalid bool value on AllowPublicApi"))
	}

	return &Config{
		Port:           port,
		ApiSecret:      apiSecret,
		AllowPublicApi: isPublicApiAllowed,
	}
}
