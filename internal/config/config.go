package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
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
	AuthToken       string // Persisted authentication token for gRPC auth
	TokenFilePath   string // File path where the auth token is stored
	EnrollAPIKey    string // API key used only for first-time enrollment (not persisted)
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

	// Token file path — defaults to /data/.sirius-agent-token  (container-friendly)
	// or ~/.sirius-agent-token if /data is not writable.
	tokenFilePath := os.Getenv("AGENT_TOKEN_FILE")
	if tokenFilePath == "" {
		dataDir := "/data"
		if info, err := os.Stat(dataDir); err == nil && info.IsDir() {
			tokenFilePath = filepath.Join(dataDir, ".sirius-agent-token")
		} else {
			home, _ := os.UserHomeDir()
			if home == "" {
				home = "/tmp"
			}
			tokenFilePath = filepath.Join(home, ".sirius-agent-token")
		}
	}

	// Load persisted auth token from file (if it exists).
	authToken := os.Getenv("AGENT_AUTH_TOKEN")
	if authToken == "" {
		authToken = loadTokenFromFile(tokenFilePath)
	}

	// Enrollment API key — used only on first connect when no auth token exists.
	enrollKey := os.Getenv("SIRIUS_AGENT_ENROLL_KEY")
	if enrollKey == "" {
		enrollKey = os.Getenv("AGENT_ENROLL_KEY")
	}

	return &AgentConfig{
		ServerAddress:   serverAddr,
		AgentID:         agentID,
		HostID:          hostID,
		ApiBaseURL:      apiURL,
		PowerShellPath:  psPath,
		EnableScripting: enableScripting,
		AuthToken:       authToken,
		TokenFilePath:   tokenFilePath,
		EnrollAPIKey:    enrollKey,
	}
}

// loadTokenFromFile reads a token from disk, returning "" on any error.
func loadTokenFromFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// SaveAuthToken persists the auth token to the configured file.
func (c *AgentConfig) SaveAuthToken(token string) error {
	c.AuthToken = token
	dir := filepath.Dir(c.TokenFilePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create token dir: %w", err)
	}
	return os.WriteFile(c.TokenFilePath, []byte(token+"\n"), 0600)
}
