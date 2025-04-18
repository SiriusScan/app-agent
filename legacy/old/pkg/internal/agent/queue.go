package agent

import (
	"fmt"

	"github.com/SiriusScan/go-api/sirius/queue"
)

// QueueClient wraps the queue.Listen function to provide a more testable interface
type QueueClient interface {
	Listen(queueName string, handler func(string)) error
}

// DefaultQueueClient is the default implementation of QueueClient
type DefaultQueueClient struct{}

// Listen implements the QueueClient interface using the queue package
func (c *DefaultQueueClient) Listen(queueName string, handler func(string)) error {
	if queueName == "" {
		return fmt.Errorf("queue name cannot be empty")
	}
	queue.Listen(queueName, handler)
	return nil
}

// NewQueueClient creates a new DefaultQueueClient
func NewQueueClient() QueueClient {
	return &DefaultQueueClient{}
}
