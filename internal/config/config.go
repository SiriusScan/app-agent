package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// ServerConfig holds configuration for the server
type ServerConfig struct {
	Address string // Address to listen on, e.g., :50051
}

// AgentConfig holds configuration for the agent
type AgentConfig struct {
	ServerAddress   string // Address of the gRPC server to connect to
	AgentID         string // Unique ID for this agent (defaults to hostname)
	HostID          string // Unique ID for the host record in the backend (defaults to AgentID)
	ApiBaseURL      string // Base URL for the backend REST API
	PowerShellPath  string // Optional override path for PowerShell executable
	EnableScripting bool   // Whether to enable PowerShell scripting
}

// LoadServerConfig loads server configuration from environment variables
// or uses default values
func LoadServerConfig() *ServerConfig {
	addr := os.Getenv("SERVER_ADDRESS")
	if addr == "" {
		addr = ":50051" // Default address
	}

	return &ServerConfig{
		Address: addr,
	}
}

// LoadAgentConfig loads agent configuration from environment variables
// or uses default values
func LoadAgentConfig() *AgentConfig {
	serverAddr := os.Getenv("SERVER_ADDRESS")
	if serverAddr == "" {
		serverAddr = "localhost:50051" // Default gRPC address
	}

	agentID := os.Getenv("AGENT_ID")
	if agentID == "" {
		// Generate a simple hostname-based ID if not specified
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown-host"
		}
		agentID = hostname
	}

	// Host ID for backend record, defaults to Agent ID if not set
	hostID := os.Getenv("HOST_ID")
	if hostID == "" {
		hostID = agentID
	}

	// API Base URL
	apiURL := os.Getenv("API_BASE_URL")
	if apiURL == "" {
		// Default API URL based on server address, using port 9001
		host, _, err := net.SplitHostPort(serverAddr)
		if err != nil {
			// Fallback if serverAddr is not host:port format (e.g., just hostname)
			host = serverAddr
			// Attempt to remove any potential leading slashes or protocol if user provided weird default
			host = strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://")
			// Handle cases like ":50051"
			if strings.HasPrefix(host, ":") {
				host = "localhost"
			}
		}
		// Ensure host is not empty after parsing
		if host == "" {
			host = "localhost"
		}
		apiURL = fmt.Sprintf("http://%s:9001", host)
	}

	// Optional override for PowerShell executable path
	psPath := os.Getenv("POWERSHELL_PATH")

	// Enable scripting by default, disable if explicitly set to "false"
	enableScriptingStr := os.Getenv("ENABLE_SCRIPTING")
	enableScripting := true // Default to true
	if enableScriptingStr != "" {
		parsedBool, err := strconv.ParseBool(enableScriptingStr)
		if err == nil {
			enableScripting = parsedBool
		}
	}

	return &AgentConfig{
		ServerAddress:   serverAddr,
		AgentID:         agentID,
		HostID:          hostID,
		ApiBaseURL:      apiURL,
		PowerShellPath:  psPath,
		EnableScripting: enableScripting,
	}
}
