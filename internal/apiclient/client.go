package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/debugtrace"
	"github.com/SiriusScan/go-api/sirius"
	"github.com/SiriusScan/go-api/sirius/postgres/models"
)

const defaultTimeout = 15 * time.Second

var serviceAPIKeyEnvNames = []string{
	"SIRIUS_API_KEY",
	"SIRIUS_AGENT_API_KEY",
	"API_KEY",
}

var serviceAPIKeyFileEnvNames = []string{
	"SIRIUS_API_KEY_FILE",
	"SIRIUS_AGENT_API_KEY_FILE",
	"API_KEY_FILE",
}

func serviceAPIKey() (string, error) {
	for _, envName := range serviceAPIKeyEnvNames {
		key := strings.TrimSpace(os.Getenv(envName))
		if key != "" {
			// #region agent log
			debugtrace.Log("pre-fix", "H3,H5", "internal/apiclient/client.go:34", "service_api_key_from_env", map[string]interface{}{
				"envName": envName,
			})
			// #endregion
			return key, nil
		}
	}
	for _, envName := range serviceAPIKeyFileEnvNames {
		path := strings.TrimSpace(os.Getenv(envName))
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			// #region agent log
			debugtrace.Log("pre-fix", "H3,H5", "internal/apiclient/client.go:47", "service_api_key_file_read_failed", map[string]interface{}{
				"envName": envName,
				"path":    path,
				"error":   err.Error(),
			})
			// #endregion
			continue
		}
		key := strings.TrimSpace(string(data))
		if key != "" {
			// #region agent log
			debugtrace.Log("pre-fix", "H3,H5", "internal/apiclient/client.go:57", "service_api_key_from_file", map[string]interface{}{
				"envName": envName,
				"path":    path,
			})
			// #endregion
			return key, nil
		}
	}
	// #region agent log
	debugtrace.Log("pre-fix", "H3,H5", "internal/apiclient/client.go:65", "service_api_key_missing", map[string]interface{}{
		"acceptedEnvVars":     serviceAPIKeyEnvNames,
		"acceptedFileEnvVars": serviceAPIKeyFileEnvNames,
	})
	// #endregion
	return "", fmt.Errorf(
		"service API key is required for agent API calls (set one of: %s or file variants: %s)",
		strings.Join(serviceAPIKeyEnvNames, ", "),
		strings.Join(serviceAPIKeyFileEnvNames, ", "),
	)
}

// ServiceAPIKeyConfigured reports whether a supported API key env var is set.
func ServiceAPIKeyConfigured() bool {
	_, err := serviceAPIKey()
	return err == nil
}

// ServiceAPIKeyEnvNames returns a copy of accepted API key env var names.
func ServiceAPIKeyEnvNames() []string {
	names := make([]string, 0, len(serviceAPIKeyEnvNames)+len(serviceAPIKeyFileEnvNames))
	names = append(names, serviceAPIKeyEnvNames...)
	names = append(names, serviceAPIKeyFileEnvNames...)
	return names
}

// UpdateHostRecord sends the host data to the backend API to create or update the host record.
// It performs an HTTP POST request to the endpoint {apiBaseURL}/host.
func UpdateHostRecord(ctx context.Context, apiBaseURL string, hostData sirius.Host) error {
	// Marshal the host data to JSON
	jsonData, err := json.Marshal(hostData)
	if err != nil {
		return fmt.Errorf("failed to marshal host data to JSON: %w", err)
	}

	// Construct the target URL
	targetURL := fmt.Sprintf("%s/host", apiBaseURL)

	// Create a context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Create the HTTP POST request
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, targetURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SiriusScanAgent/1.0") // Optional: identify the agent
	key, err := serviceAPIKey()
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", key)

	// Execute the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP request to %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	// Check the response status code (Expect 200 OK, 201 Created, or 204 No Content for success)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		// Attempt to read body for more error info, but don't fail if reading fails
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received unexpected status code %d from %s: %s", resp.StatusCode, targetURL, string(bodyBytes))
	}

	// Success
	return nil
}

// UpdateHostRecordWithEnhancedData sends host data with enhanced JSONB fields (software inventory, system fingerprint, agent metadata).
// It performs an HTTP POST request to the endpoint {apiBaseURL}/host/with-source.
func UpdateHostRecordWithEnhancedData(
	ctx context.Context,
	apiBaseURL string,
	hostData sirius.Host,
	softwareInventory map[string]interface{},
	systemFingerprint map[string]interface{},
	agentMetadata map[string]interface{},
) error {
	// Build enhanced request payload
	payload := map[string]interface{}{
		"host": hostData,
		"source": models.ScanSource{
			Name:    "sirius-agent",
			Version: "1.0.0-template-mvp",
			Config:  "template-scan",
		},
	}

	// Add JSONB fields if they have data
	if len(softwareInventory) > 0 {
		payload["software_inventory"] = softwareInventory
	}
	if len(systemFingerprint) > 0 {
		payload["system_fingerprint"] = systemFingerprint
	}
	if len(agentMetadata) > 0 {
		payload["agent_metadata"] = agentMetadata
	}

	// Marshal the payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal enhanced host data to JSON: %w", err)
	}

	// Construct the target URL
	targetURL := fmt.Sprintf("%s/host/with-source", apiBaseURL)

	// Create a context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Create the HTTP POST request
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, targetURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SiriusScanAgent/1.0-MVP")
	key, err := serviceAPIKey()
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", key)

	// Execute the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP request to %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received unexpected status code %d from %s: %s", resp.StatusCode, targetURL, string(bodyBytes))
	}

	// Success
	return nil
}
