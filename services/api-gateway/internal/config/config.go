package config

import "ms-ride-sharing/shared/env"

type Config struct {
	Environment string
	Port        string
	JWTSecret   string
	UserSvcAddr string
	RedisHost   string
	RedisPort   string
}

func LoadConfig() *Config {
	return &Config{
		Environment: env.GetString("ENVIRONMENT", "development"),
		Port:        env.GetString("API_GATEWAY_PORT", "8080"),
		JWTSecret:   env.GetString("JWT_SECRET", ""),
		UserSvcAddr: env.GetString("USER_SERVICE_ADDR", "user-service:9091"),
		RedisHost:   env.GetString("REDIS_HOST", "localhost"),
		RedisPort:   env.GetString("REDIS_PORT", "6379"),
	}
}
