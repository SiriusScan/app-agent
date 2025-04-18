package queue

import (
	"context"
)

// Message represents a message in the queue
type Message struct {
	ID      string
	Type    string
	Payload []byte
}

// Client defines the interface for queue operations
type Client interface {
	// Connect establishes a connection to the queue server
	Connect(ctx context.Context) error

	// Close closes the connection to the queue server
	Close() error

	// Publish sends a message to the queue
	Publish(ctx context.Context, msg *Message) error

	// Subscribe starts consuming messages from the queue
	Subscribe(ctx context.Context, handler func(*Message) error) error
}
