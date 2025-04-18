package main

import (
	"encoding/json"
	"flag"
	"log"

	"github.com/SiriusScan/app-agent/internal/command"
	"github.com/SiriusScan/go-api/sirius/queue"
)

func main() {
	agentID := flag.String("agent", "", "Agent ID to send command to")
	cmd := flag.String("command", "", "Command to execute")
	flag.Parse()

	logger := log.Default()

	if *agentID == "" || *cmd == "" {
		logger.Fatal("agent and command are required")
	}

	// Create command message
	msg := command.NewMessage(*agentID, *cmd)

	// Convert to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		logger.Fatal("failed to marshal command message:", err)
	}

	// Send command
	if err := queue.Send("agent.commands", string(jsonData)); err != nil {
		logger.Fatal("failed to send command:", err)
	}

	logger.Printf("Command sent successfully to agent %s", *agentID)
}
