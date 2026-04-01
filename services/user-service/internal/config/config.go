package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort  string
	DBHost   string
	DBPort   string
	DBUser   string
	DBPass   string
	DBName   string
	DBSSLMode string
	NATSURL  string
}

func Load() *Config {
	return &Config{
		AppPort:   getEnv("APP_PORT", "8002"),
		DBHost:    getEnv("DB_HOST", "localhost"),
		DBPort:    getEnv("DB_PORT", "5432"),
		DBUser:    getEnv("DB_USER", "coachlink"),
		DBPass:    getEnv("DB_PASSWORD", "secret"),
		DBName:    getEnv("DB_NAME", "user_db"),
		DBSSLMode: getEnv("DB_SSLMODE", "disable"),
		NATSURL:   getEnv("NATS_URL", "nats://localhost:4222"),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPass, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
