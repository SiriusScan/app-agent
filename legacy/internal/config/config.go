package config

import (
	"os"
	"strconv"
)

// ServerConfig holds configuration for the Agent Manager (server)
type ServerConfig struct {
	Version        string // Application version
	Environment    string // development, staging, production
	GRPCServerAddr string // gRPC server address
	LogLevel       string // Logging level
	LogFormat      string // Logging format
}

// AgentConfig holds configuration for the Test Agent (client)
type AgentConfig struct {
	AgentID      string            // Unique agent identifier
	ServerAddr   string            // gRPC server address
	LogLevel     string            // Logging level
	LogFormat    string            // Logging format
	Capabilities map[string]string // Agent capabilities
}

// LoadServerConfig loads server configuration from environment variables
func LoadServerConfig() *ServerConfig {
	return &ServerConfig{
		Version:        getEnv("APP_VERSION", "0.1.0"),
		Environment:    getEnv("APP_ENV", "development"),
		GRPCServerAddr: getEnv("GRPC_SERVER_ADDR", ":50051"),
		LogLevel:       getEnv("SIRIUS_LOG_LEVEL", "info"),
		LogFormat:      getEnv("SIRIUS_LOG_FORMAT", "console"),
	}
}

// LoadAgentConfig loads agent configuration from environment variables
func LoadAgentConfig() *AgentConfig {
	return &AgentConfig{
		AgentID:    getEnv("AGENT_ID", "agent-001"),
		ServerAddr: getEnv("SERVER_ADDR", "localhost:50051"),
		LogLevel:   getEnv("SIRIUS_LOG_LEVEL", "info"),
		LogFormat:  getEnv("SIRIUS_LOG_FORMAT", "console"),
		Capabilities: map[string]string{
			"scan_type": "full",
		},
	}
}

// Helper function to get environment variable with a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Helper function to get boolean environment variable
func getBoolEnv(key string, defaultValue bool) bool {
	strValue := os.Getenv(key)
	if strValue == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(strValue)
	if err != nil {
		return defaultValue
	}

	return boolValue
}
