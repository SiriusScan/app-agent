package command

import (
	"fmt"
	"time"
)

// Message represents a command message sent through RabbitMQ
type Message struct {
	MessageID string    `json:"message_id"`
	AgentID   string    `json:"agent_id"`
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
}

// NewMessage creates a new command message
func NewMessage(agentID, command string) *Message {
	return &Message{
		MessageID: fmt.Sprintf("cmd-%d", time.Now().UnixNano()),
		AgentID:   agentID,
		Command:   command,
		Timestamp: time.Now(),
	}
}
