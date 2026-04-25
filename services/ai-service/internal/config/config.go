package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort             string
	AnalyticsServiceURL string
	UserServiceURL      string
	OllamaURL           string
	OllamaModel         string
}

func Load() *Config {
	cfg := &Config{
		AppPort:             getEnv("APP_PORT", "8006"),
		AnalyticsServiceURL: os.Getenv("ANALYTICS_SERVICE_URL"),
		UserServiceURL:      os.Getenv("USER_SERVICE_URL"),
		OllamaURL:           getEnv("OLLAMA_URL", "http://ollama:11434"),
		OllamaModel:         getEnv("OLLAMA_MODEL", "gemma3:4b"),
	}

	if cfg.AnalyticsServiceURL == "" {
		fmt.Fprintln(os.Stderr, "FATAL: ANALYTICS_SERVICE_URL is required")
		os.Exit(1)
	}

	if cfg.UserServiceURL == "" {
		fmt.Fprintln(os.Stderr, "FATAL: USER_SERVICE_URL is required")
		os.Exit(1)
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
