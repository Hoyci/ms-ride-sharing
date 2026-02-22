package config

import "ms-ride-sharing/shared/env"

type Config struct {
	Environment string
	Port        string
	UserSvcAddr string
}

func LoadConfig() *Config {
	return &Config{
		Environment: env.GetString("ENVIRONMENT", "development"),
		Port:        env.GetString("API_GATEWAY_PORT", "8080"),
		UserSvcAddr: env.GetString("USER_SERVICE_ADDR", "user-service:9091"),
	}
}
