package config

import (
	"os"
	"strconv"

	"github.com/SiriusScan/app-agent/internal/store"
)

// LoadStoreConfig loads store configuration from environment variables
func LoadStoreConfig() *store.Config {
	cfg := &store.Config{
		Address:  getEnv("STORE_ADDRESS", "localhost:6379"),
		Username: getEnv("STORE_USERNAME", ""),
		Password: getEnv("STORE_PASSWORD", ""),
		Database: getEnv("STORE_DATABASE", "0"),
	}

	// Parse timeouts with defaults
	cfg.ConnectionTimeout = getEnvInt("STORE_CONN_TIMEOUT", 5)
	cfg.OperationTimeout = getEnvInt("STORE_OP_TIMEOUT", 3)

	return cfg
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as an integer with a default value
func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
