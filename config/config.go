package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/tracoco/DifyGate/gate"
)

// Config holds all application configuration
type Config struct {
	DIFYGATE gate.DIFYGateConfig
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		DIFYGATE: gate.DIFYGateConfig{
			Host:     getEnv("DIFYGATE_SMTP_HOST", "smtp.gmail.com"),
			Port:     getEnvAsInt("DIFYGATE_SMTP_PORT", 587),
			Username: os.Getenv("DIFYGATE_SMTP_USERNAME"),
			Password: os.Getenv("DIFYGATE_SMTP_PASSWORD"),
			FromName: getEnv("DIFYGATE_SMTP_FROM_NAME", "DifyGate Email Service"),
		},
	}

	return config, nil
}

// Helper functions to extract environment variables
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}
