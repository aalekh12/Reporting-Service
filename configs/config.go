// Package configs loads runtime configuration from environment variables.
package configs

import (
	"fmt"
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	Debug       bool
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		Debug:       getEnv("DEBUG", "false") == "true",
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
