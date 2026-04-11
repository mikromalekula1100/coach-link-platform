package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort            string
	TrainingServiceURL string
}

func Load() *Config {
	cfg := &Config{
		AppPort:            getEnv("APP_PORT", "8005"),
		TrainingServiceURL: os.Getenv("TRAINING_SERVICE_URL"),
	}

	if cfg.TrainingServiceURL == "" {
		fmt.Fprintln(os.Stderr, "FATAL: TRAINING_SERVICE_URL is required")
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
