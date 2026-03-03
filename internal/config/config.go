package config

import (
	"os"
)

type Config struct {
	DatabaseURL string
	AuthToken   string
	Addr        string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/test?sslmode=disable"),
		AuthToken:   getEnv("AUTH_TOKEN", "secret-token-1234567890abcdef"),
		Addr:        getEnv("ADDR", ":8080"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
