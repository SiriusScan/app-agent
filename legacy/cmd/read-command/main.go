package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/SiriusScan/go-api/sirius/store"
)

func main() {
	// Define command-line flags
	agentID := flag.String("agent", "", "Agent ID to read commands for")
	cmdID := flag.String("id", "", "Specific command ID to read")
	readLatest := flag.Bool("latest", false, "Read the latest command for the agent")
	waitForResult := flag.Bool("wait", false, "Wait for command completion (up to 30 seconds)")
	onlyResult := flag.Bool("result", false, "Only show command result")
	listKeys := flag.Bool("list-keys", false, "List all keys in Valkey related to the command or agent")
	flag.Parse()

	// Validate inputs
	if *agentID == "" {
		log.Fatal("Error: --agent flag is required")
	}

	// Create a context
	ctx := context.Background()

	// Initialize the Valkey store
	kvStore, err := store.NewValkeyStore()
	if err != nil {
		log.Fatalf("Failed to connect to Valkey: %v", err)
	}
	defer kvStore.Close()

	// If we're listing keys, do that and exit
	if *listKeys {
		listRelatedKeys(ctx, kvStore, *agentID, *cmdID)
		return
	}

	// Determine the key to read
	var key string
	if *readLatest {
		key = fmt.Sprintf("cmd:%s:latest", *agentID)
	} else if *cmdID != "" {
		key = fmt.Sprintf("cmd:%s:%s", *agentID, *cmdID)
	} else {
		log.Fatal("Either --id or --latest flag must be provided")
	}

	// If we're only looking for results, get the command ID first if needed
	var commandID string
	if *onlyResult {
		if *cmdID != "" {
			commandID = *cmdID
		} else if *readLatest {
			// Get the latest command ID
			resp, err := kvStore.GetValue(ctx, key)
			if err != nil {
				log.Fatalf("Failed to read command: %v", err)
			}

			if resp.Type == "nil" || resp.Message.Value == "" {
				log.Fatalf("No command found for key: %s", key)
			}

			var cmd map[string]interface{}
			if err := json.Unmarshal([]byte(resp.Message.Value), &cmd); err != nil {
				log.Fatalf("Failed to parse command data: %v", err)
			}

			commandID = cmd["id"].(string)
		}

		// Skip to showing results
		showCommandResult(ctx, kvStore, commandID, *waitForResult)
		return
	}

	// Read the command data
	resp, err := kvStore.GetValue(ctx, key)
	if err != nil {
		log.Fatalf("Failed to read command: %v", err)
	}

	// Check if we got a valid response
	if resp.Type == "nil" || resp.Message.Value == "" {
		log.Fatalf("No command found for key: %s", key)
	}

	// Parse the command data
	var cmd map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Message.Value), &cmd); err != nil {
		log.Fatalf("Failed to parse command data: %v", err)
	}

	// Print command details in a nicely formatted way
	fmt.Println("\nCommand Details:")
	fmt.Println("---------------")
	fmt.Printf("Command ID:    %s\n", cmd["id"])
	fmt.Printf("Type:          %s\n", cmd["type"])
	fmt.Printf("Agent ID:      %s\n", cmd["agent_id"])

	// Format timestamp
	if ts, ok := cmd["timestamp"].(float64); ok {
		timestamp := time.Unix(int64(ts), 0)
		fmt.Printf("Timestamp:     %s\n", timestamp.Format(time.RFC3339))
	}

	// Print parameters
	if params, ok := cmd["params"].(map[string]interface{}); ok {
		fmt.Println("\nParameters:")
		for k, v := range params {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	// Check if there's a status for this command
	if commandID, ok := cmd["id"].(string); ok {
		statusKey := fmt.Sprintf("status:%s:%s", *agentID, commandID)
		statusResp, err := kvStore.GetValue(ctx, statusKey)
		if err == nil && statusResp.Type != "nil" && statusResp.Message.Value != "" {
			fmt.Printf("\nStatus: %s\n", statusResp.Message.Value)
		}

		// Check for command result
		if *waitForResult {
			showCommandResult(ctx, kvStore, commandID, true)
		} else {
			// Just check once without waiting
			showCommandResult(ctx, kvStore, commandID, false)
		}
	}
}

// listRelatedKeys lists all keys in Valkey related to the agent or command
func listRelatedKeys(ctx context.Context, kvStore store.KVStore, agentID, cmdID string) {
	// Try method from distributor.go first - use the SCAN operation
	// These patterns will help us find keys
	patterns := []string{}

	// Add patterns based on agent ID
	if agentID != "" {
		patterns = append(patterns,
			fmt.Sprintf("cmd:%s:*", agentID),
			fmt.Sprintf("status:%s:*", agentID),
			fmt.Sprintf("agent:%s:*", agentID),
			fmt.Sprintf("result:%s:*", agentID),
		)
	}

	// Add patterns based on command ID
	if cmdID != "" {
		patterns = append(patterns,
			fmt.Sprintf("*:%s", cmdID),
			fmt.Sprintf("*:result:%s", cmdID),
			fmt.Sprintf("*:%s:*", cmdID),
		)
	}

	fmt.Println("\nSearching for keys with the following patterns:")
	for _, pattern := range patterns {
		fmt.Printf("  - %s\n", pattern)
	}

	foundKeys := map[string]struct{}{}

	// Search for each pattern
	for _, pattern := range patterns {
		// Try to get all keys matching the pattern
		// Note: If this fails, we'll fall back to reading specific known key patterns
		tryGetMatchingKeys(ctx, kvStore, pattern, foundKeys)
	}

	// If we didn't find any keys and have a command ID, try some specific known key formats
	if len(foundKeys) == 0 && cmdID != "" {
		specificKeys := []string{
			fmt.Sprintf("cmd:%s:%s", agentID, cmdID),
			fmt.Sprintf("cmd:result:%s", cmdID),
			fmt.Sprintf("result:%s", cmdID),
			fmt.Sprintf("status:%s:%s", agentID, cmdID),
			fmt.Sprintf("command:result:%s", cmdID),
		}

		fmt.Println("\nTrying specific key formats:")
		for _, key := range specificKeys {
			fmt.Printf("  - %s\n", key)
			tryReadKey(ctx, kvStore, key, foundKeys)
		}
	}

	// Output the keys we found
	fmt.Printf("\nFound %d keys:\n", len(foundKeys))
	for key := range foundKeys {
		// Try to get value to see if it exists
		resp, err := kvStore.GetValue(ctx, key)
		if err == nil && resp.Type != "nil" && resp.Message.Value != "" {
			fmt.Printf("  - %s (exists)\n", key)

			// For small values, show the contents
			if len(resp.Message.Value) < 100 {
				fmt.Printf("    Value: %s\n", resp.Message.Value)
			} else {
				// Just show the beginning for large values
				fmt.Printf("    Value: %s... (%d bytes)\n", resp.Message.Value[:80], len(resp.Message.Value))
			}
		} else {
			fmt.Printf("  - %s (empty or error)\n", key)
		}
	}

	if len(foundKeys) == 0 {
		fmt.Println("No keys found matching the search patterns.")
	}
}

// tryGetMatchingKeys attempts to get all keys matching a pattern
func tryGetMatchingKeys(ctx context.Context, kvStore store.KVStore, pattern string, foundKeys map[string]struct{}) {
	// This is a placeholder - the actual KVStore implementation would need to support a SCAN-like operation
	// For now, we'll just try a direct read
	resp, err := kvStore.GetValue(ctx, pattern)
	if err == nil && resp.Type != "nil" && resp.Message.Value != "" {
		foundKeys[pattern] = struct{}{}
	}
}

// tryReadKey attempts to read a specific key
func tryReadKey(ctx context.Context, kvStore store.KVStore, key string, foundKeys map[string]struct{}) {
	resp, err := kvStore.GetValue(ctx, key)
	if err == nil && resp.Type != "nil" && resp.Message.Value != "" {
		foundKeys[key] = struct{}{}
	}
}

// showCommandResult attempts to retrieve and display command result data
func showCommandResult(ctx context.Context, kvStore store.KVStore, commandID string, wait bool) {
	if commandID == "" {
		return
	}

	// Try to find result
	var result map[string]interface{}
	var found bool

	// Define key patterns to check
	resultKeyPatterns := []string{
		fmt.Sprintf("cmd:result:%s", commandID),     // First pattern
		fmt.Sprintf("result:%s", commandID),         // Second pattern
		fmt.Sprintf("command:result:%s", commandID), // Third pattern
	}

	// If wait flag is true, poll for results
	if wait {
		fmt.Println("\nWaiting for command result...")

		// Try for up to 30 seconds
		for i := 0; i < 30; i++ {
			found, result = tryGetResult(ctx, kvStore, resultKeyPatterns)
			if found {
				break
			}
			time.Sleep(1 * time.Second)
			fmt.Print(".")
		}
		fmt.Println()
	} else {
		// Just try once
		found, result = tryGetResult(ctx, kvStore, resultKeyPatterns)
	}

	if !found {
		if wait {
			fmt.Println("\nNo result found within timeout period.")
		}
		return
	}

	// Print result details
	fmt.Println("\nCommand Result:")
	fmt.Println("---------------")

	// Print common fields
	if status, ok := result["status"].(string); ok {
		fmt.Printf("Status:      %s\n", status)
	}

	if completedAt, ok := result["completed_at"].(float64); ok {
		timestamp := time.Unix(int64(completedAt), 0)
		fmt.Printf("Completed:   %s\n", timestamp.Format(time.RFC3339))
	}

	// Print data/output fields
	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Println("\nOutput Data:")
		for k, v := range data {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	// Print error field if present
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		fmt.Printf("\nError: %s\n", errMsg)
	}

	// Print raw output if present
	if output, ok := result["output"].(string); ok && output != "" {
		fmt.Printf("\nRaw Output:\n%s\n", output)
	}
}

// tryGetResult attempts to get a result from multiple possible key patterns
func tryGetResult(ctx context.Context, kvStore store.KVStore, keyPatterns []string) (bool, map[string]interface{}) {
	for _, key := range keyPatterns {
		resp, err := kvStore.GetValue(ctx, key)
		if err != nil || resp.Type == "nil" || resp.Message.Value == "" {
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(resp.Message.Value), &result); err != nil {
			continue
		}

		return true, result
	}

	return false, nil
}
