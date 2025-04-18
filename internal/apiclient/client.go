package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/SiriusScan/go-api/sirius"
	// Assuming this is the correct import path for the Host struct
)

const defaultTimeout = 15 * time.Second

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
