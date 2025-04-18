package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/SiriusScan/go-api/sirius/queue"
)

type TaskMessage struct {
	ID        string            `json:"command_id"`
	Type      string            `json:"type"`
	AgentID   string            `json:"agent_id"`
	Args      map[string]string `json:"args"`
	Timeout   int64             `json:"timeout"`
	Timestamp time.Time         `json:"timestamp"`
}

func main() {
	// Create a test task message
	task := TaskMessage{
		ID:      "test-cmd-001",
		Type:    "STATUS",
		AgentID: "test-agent-1",
		Args: map[string]string{
			"scan_id": "test-scan-001",
			"target":  "localhost",
		},
		Timeout:   60,
		Timestamp: time.Now(),
	}

	// Convert task to JSON
	taskJSON, err := json.Marshal(task)
	if err != nil {
		log.Fatalf("Failed to marshal task: %v", err)
	}

	// Send the message to the tasks queue
	if err := queue.Send("tasks", string(taskJSON)); err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	log.Printf("Successfully sent task: %s", string(taskJSON))
}
