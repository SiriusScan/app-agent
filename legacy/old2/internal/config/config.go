package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config represents the application configuration
type Config struct {
	// Version of the application
	Version string

	// Environment (development, staging, production)
	Environment string

	// gRPC server address
	GRPCServerAddr string

	// Log level and format
	LogLevel  string
	LogFormat string

	// Metrics configuration
	MetricsAddr string

	// TLS configuration
	TLSEnabled  bool
	TLSCertFile string
	TLSKeyFile  string
}

// AgentConfig represents the agent-specific configuration
type AgentConfig struct {
	// Version of the application
	Version string

	// Environment (development, staging, production)
	Environment string

	// gRPC server address
	GRPCServerAddr string

	// Log level and format
	LogLevel  string
	LogFormat string

	// Agent ID
	AgentID string

	// TLS configuration
	TLSEnabled  bool
	TLSCertFile string
}

// Load loads the application configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file if it exists
	_ = loadEnvFile()

	// Create and return config from environment variables
	cfg := &Config{
		Version:        getEnv("APP_VERSION", "1.0.0"),
		Environment:    getEnv("APP_ENV", "development"),
		GRPCServerAddr: getEnv("GRPC_SERVER_ADDR", ":50051"),
		LogLevel:       getEnv("SIRIUS_LOG_LEVEL", "info"),
		LogFormat:      getEnv("SIRIUS_LOG_FORMAT", "json"),
		MetricsAddr:    getEnv("METRICS_ADDR", ":2112"),
		TLSEnabled:     getEnv("TLS_ENABLED", "false") == "true",
		TLSCertFile:    getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:     getEnv("TLS_KEY_FILE", ""),
	}

	return cfg, nil
}

// LoadAgentConfig loads agent-specific configuration from environment variables
func LoadAgentConfig() (*AgentConfig, error) {
	// Try to load .env file if it exists
	_ = loadEnvFile()

	// Create and return config from environment variables
	cfg := &AgentConfig{
		Version:        getEnv("APP_VERSION", "1.0.0"),
		Environment:    getEnv("APP_ENV", "development"),
		GRPCServerAddr: getEnv("SERVER_ADDR", ":50051"),
		LogLevel:       getEnv("SIRIUS_LOG_LEVEL", "info"),
		LogFormat:      getEnv("SIRIUS_LOG_FORMAT", "json"),
		AgentID:        getEnv("AGENT_ID", ""),
		TLSEnabled:     getEnv("TLS_ENABLED", "false") == "true",
		TLSCertFile:    getEnv("TLS_CERT_FILE", ""),
	}

	return cfg, nil
}

// loadEnvFile attempts to load a .env file from the current directory or parent directories
func loadEnvFile() error {
	// Current directory
	if err := godotenv.Load(); err == nil {
		return nil
	}

	// Try parent directory
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	parentDir := filepath.Dir(dir)
	return godotenv.Load(filepath.Join(parentDir, ".env"))
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
