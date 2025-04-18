package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Logging settings
	Log struct {
		Level  string
		Format string // json or text
		File   string // optional file path
	}

	// Store settings
	Store struct {
		ValkeyURL string
	}
}

// LoadFromEnv creates a new Config instance from environment variables
// and optionally from a .env file
func LoadFromEnv(envFile string) (*Config, error) {
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return nil, fmt.Errorf("error loading env file: %w", err)
		}
	}

	cfg := &Config{}

	// Logging configuration
	cfg.Log.Level = getEnvWithDefault("SIRIUS_LOG_LEVEL", "info")
	cfg.Log.Format = getEnvWithDefault("SIRIUS_LOG_FORMAT", "json")
	cfg.Log.File = os.Getenv("SIRIUS_LOG_FILE")

	// Store configuration
	valkeyURL := os.Getenv("SIRIUS_VALKEY")
	if valkeyURL == "" {
		return nil, fmt.Errorf("SIRIUS_VALKEY environment variable is required")
	}

	// Format Valkey URL if needed
	if !strings.HasPrefix(valkeyURL, "redis://") {
		// If it's just host:port, add redis:// prefix
		if strings.Contains(valkeyURL, ":") {
			valkeyURL = "redis://" + valkeyURL
		} else {
			// If just host, add default Redis port
			valkeyURL = fmt.Sprintf("redis://%s:6379", valkeyURL)
		}
	}
	cfg.Store.ValkeyURL = valkeyURL

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// validate checks if all required configuration values are present and valid
func (c *Config) validate() error {
	// Validate log level
	if !isValidLogLevel(c.Log.Level) {
		return fmt.Errorf("invalid log level: %s", c.Log.Level)
	}

	// Validate log format
	if c.Log.Format != "json" && c.Log.Format != "text" {
		return fmt.Errorf("invalid log format: %s", c.Log.Format)
	}

	// Validate Valkey URL format
	if !strings.HasPrefix(c.Store.ValkeyURL, "redis://") {
		return fmt.Errorf("invalid Valkey URL format: must start with redis://")
	}

	return nil
}

// Helper functions

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func isValidLogLevel(level string) bool {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	return validLevels[level]
}
