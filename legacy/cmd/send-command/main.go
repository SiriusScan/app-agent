package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/SiriusScan/go-api/sirius/queue"
)

// Command represents a command to be sent to agents
type Command struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	AgentID   string            `json:"agent_id"`
	Params    map[string]string `json:"params"`
	Timestamp int64             `json:"timestamp"`
}

func main() {
	// Define command-line flags
	queueName := flag.String("queue", "agent-commands", "Name of the queue to send the command to")
	agentID := flag.String("agent", "", "Agent ID to send command to (required)")
	cmdType := flag.String("type", "ping", "Command type (default: ping)")
	paramKey := flag.String("param-key", "", "Optional parameter key")
	paramValue := flag.String("param-value", "", "Optional parameter value")
	flag.Parse()

	// Validate required flags
	if *agentID == "" {
		log.Fatal("Error: --agent flag is required")
	}

	// Create a command with a unique ID based on timestamp
	cmd := Command{
		ID:        fmt.Sprintf("cmd-%d", time.Now().UnixNano()),
		Type:      *cmdType,
		AgentID:   *agentID,
		Params:    make(map[string]string),
		Timestamp: time.Now().Unix(),
	}

	// Add parameter if provided
	if *paramKey != "" {
		cmd.Params[*paramKey] = *paramValue
	}

	// Add some default parameters based on command type
	if *cmdType == "scan" {
		if _, ok := cmd.Params["target"]; !ok {
			cmd.Params["target"] = "localhost"
		}
	}

	// Convert command to JSON
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		log.Fatalf("Error encoding command to JSON: %v", err)
	}

	// Send the command to the queue using the Sirius API
	err = queue.Send(*queueName, string(cmdJSON))
	if err != nil {
		log.Fatalf("Error sending command to queue: %v", err)
	}

	// Print success message with command details
	fmt.Printf("Command sent successfully:\n")
	fmt.Printf("  Queue:      %s\n", *queueName)
	fmt.Printf("  Command ID: %s\n", cmd.ID)
	fmt.Printf("  Type:       %s\n", cmd.Type)
	fmt.Printf("  Agent ID:   %s\n", cmd.AgentID)
	fmt.Printf("  Parameters: %v\n", cmd.Params)
}
