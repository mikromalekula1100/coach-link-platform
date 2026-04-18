package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort            string
	UserServiceURL     string
	TrainingServiceURL string
}

func Load() *Config {
	cfg := &Config{
		AppPort:            getEnv("APP_PORT", "8007"),
		UserServiceURL:     os.Getenv("USER_SERVICE_URL"),
		TrainingServiceURL: os.Getenv("TRAINING_SERVICE_URL"),
	}

	if cfg.UserServiceURL == "" {
		fmt.Fprintln(os.Stderr, "FATAL: USER_SERVICE_URL is required")
		os.Exit(1)
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
