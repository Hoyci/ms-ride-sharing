package config

import "ms-ride-sharing/shared/env"

type Config struct {
	Environment      string
	Port             string
	JWTSecret        string
	PostgresUser     string
	PostgresPassword string
	PostgresHost     string
	PostgresPort     string
	PostgresDB       string
	RedisHost        string
	RedisPort        string
}

func LoadConfig() *Config {
	return &Config{
		Environment:      env.GetString("ENVIRONMENT", "development"),
		Port:             env.GetString("USER_SERVICE_PORT", "9091"),
		JWTSecret:        env.GetString("JWT_SECRET", "um-secret-mto-dificil"),
		PostgresUser:     env.GetString("POSTGRES_USER", "user"),
		PostgresPassword: env.GetString("POSTGRES_PASSWORD", "password"),
		PostgresHost:     env.GetString("POSTGRES_HOST", "user-service-db"),
		PostgresPort:     env.GetString("POSTGRES_PORT", "5432"),
		PostgresDB:       env.GetString("POSTGRES_DB", "postgres"),
		RedisHost:        env.GetString("REDIS_HOST", "localhost"),
		RedisPort:        env.GetString("REDIS_PORT", "6379"),
	}
}
