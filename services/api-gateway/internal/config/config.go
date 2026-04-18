package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the API Gateway.
type Config struct {
	Port                   string
	AuthServiceURL         string
	UserServiceURL         string
	TrainingServiceURL     string
	NotificationServiceURL string
	AnalyticsServiceURL    string
	AIServiceURL           string
	BDUIServiceURL         string
	JWTSecret              string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Port:                   getEnv("APP_PORT", "8080"),
		AuthServiceURL:         getEnv("AUTH_SERVICE_URL", "http://localhost:8001"),
		UserServiceURL:         getEnv("USER_SERVICE_URL", "http://localhost:8002"),
		TrainingServiceURL:     getEnv("TRAINING_SERVICE_URL", "http://localhost:8003"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8004"),
		AnalyticsServiceURL:    getEnv("ANALYTICS_SERVICE_URL", "http://localhost:8005"),
		AIServiceURL:           getEnv("AI_SERVICE_URL", "http://localhost:8006"),
		BDUIServiceURL:         getEnv("BDUI_SERVICE_URL", "http://localhost:8007"),
		JWTSecret:              os.Getenv("JWT_SECRET"),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
