// Package config handles application configuration from environment variables.
// All sensitive values (API keys, DB credentials) must come from environment.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	// Server settings
	Port           int
	Environment    string
	AllowedOrigins string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// Razorpay credentials
	Razorpay RazorpayConfig

	// JWT settings
	JWTSecret     string
	JWTExpiration int // hours
}

// RazorpayConfig holds Razorpay API credentials
type RazorpayConfig struct {
	KeyID        string
	KeySecret    string
	WebhookSecret string
}

// Load reads configuration from environment variables.
// Returns error if required variables are missing.
func Load() (*Config, error) {
	cfg := &Config{}

	// Server settings with defaults
	cfg.Port = getEnvInt("PORT", 8080)
	cfg.Environment = getEnv("ENVIRONMENT", "development")
	cfg.AllowedOrigins = getEnv("ALLOWED_ORIGINS", "*")

	// Database - required
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Redis - required
	cfg.RedisURL = os.Getenv("REDIS_URL")
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("REDIS_URL environment variable is required")
	}

	// Razorpay - required for payment processing
	cfg.Razorpay.KeyID = os.Getenv("RAZORPAY_KEY_ID")
	cfg.Razorpay.KeySecret = os.Getenv("RAZORPAY_KEY_SECRET")
	cfg.Razorpay.WebhookSecret = os.Getenv("RAZORPAY_WEBHOOK_SECRET")

	if cfg.Razorpay.KeyID == "" || cfg.Razorpay.KeySecret == "" {
		return nil, fmt.Errorf("RAZORPAY_KEY_ID and RAZORPAY_KEY_SECRET are required")
	}

	// JWT settings
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	cfg.JWTExpiration = getEnvInt("JWT_EXPIRATION_HOURS", 24)

	return cfg, nil
}

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns environment variable as int or default
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
