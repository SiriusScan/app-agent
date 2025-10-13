package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// CustomContentMessage defines the schema for custom content update messages
// operation: create, update, delete
type CustomContentMessage struct {
	Operation   string                 `json:"operation"`
	Type        string                 `json:"type"` // "template" or "script"
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Content     map[string]interface{} `json:"content"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// StartCustomContentConsumer starts a background goroutine to consume custom content updates from RabbitMQ
func (s *Server) StartCustomContentConsumer(ctx context.Context, amqpURL, queueName string) {
	go func() {
		s.logger.Info("Starting RabbitMQ consumer for custom content updates", zap.String("queue", queueName))
		for {
			if err := s.consumeCustomContentQueue(ctx, amqpURL, queueName); err != nil {
				s.logger.Error("Custom content consumer error, retrying in 5s", zap.Error(err))
				time.Sleep(5 * time.Second)
			}
		}
	}()
}

func (s *Server) consumeCustomContentQueue(ctx context.Context, amqpURL, queueName string) error {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		queueName,
		false, // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	msgs, err := ch.Consume(
		queueName,
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Custom content consumer shutting down")
			return nil
		case d := <-msgs:
			if len(d.Body) == 0 {
				// Silently ignore empty messages to reduce log spam
				// This can happen during queue purging or connection issues
				continue
			}
			var msg CustomContentMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				s.logger.Error("Failed to parse custom content message", zap.ByteString("body", d.Body), zap.Error(err))
				continue
			}
			s.handleCustomContentMessage(&msg)
		}
	}
}

func (s *Server) handleCustomContentMessage(msg *CustomContentMessage) {
	s.logger.Info("Processing custom content message", zap.String("operation", msg.Operation), zap.String("type", msg.Type), zap.String("id", msg.ID))

	// Validate required fields
	if msg.Operation == "" || msg.Type == "" || msg.ID == "" {
		s.logger.Error("Missing required fields in custom content message", zap.Any("msg", msg))
		return
	}

	switch strings.ToLower(msg.Operation) {
	case "create", "update":
		s.saveCustomContent(msg)
	case "delete":
		s.deleteCustomContent(msg)
	default:
		s.logger.Error("Unknown operation in custom content message", zap.String("operation", msg.Operation))
	}
}

func (s *Server) saveCustomContent(msg *CustomContentMessage) {
	// Convert content map to YAML string for templates, JSON string for scripts
	var contentString string
	var marshalErr error

	if msg.Type == "template" {
		// For templates, convert to YAML
		contentString, marshalErr = s.convertToYAML(msg.Content)
	} else {
		// For scripts, convert to JSON
		contentJSON, err := json.Marshal(msg.Content)
		if err != nil {
			s.logger.Error("Failed to marshal content to JSON", zap.String("id", msg.ID), zap.Error(err))
			return
		}
		contentString = string(contentJSON)
	}

	if marshalErr != nil {
		s.logger.Error("Failed to convert content", zap.String("id", msg.ID), zap.Error(marshalErr))
		return
	}

	content := &CustomContent{
		ID:          msg.ID,
		Name:        msg.Name,
		Type:        msg.Type,
		Content:     contentString,
		Description: msg.Description,
		Tags:        msg.Tags,
		Metadata:    msg.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	var saveErr error
	if msg.Type == "template" {
		saveErr = s.customStorage.SaveTemplate(content)
	} else if msg.Type == "script" {
		saveErr = s.customStorage.SaveScript(content)
	} else {
		s.logger.Error("Unknown custom content type", zap.String("type", msg.Type))
		return
	}
	if saveErr != nil {
		s.logger.Error("Failed to save custom content", zap.String("type", msg.Type), zap.String("id", msg.ID), zap.Error(saveErr))
	} else {
		s.logger.Info("Custom content saved", zap.String("type", msg.Type), zap.String("id", msg.ID))
	}
}

func (s *Server) convertToYAML(content map[string]interface{}) (string, error) {
	// Convert map to YAML using gopkg.in/yaml.v3
	yamlBytes, err := yaml.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to YAML: %w", err)
	}
	return string(yamlBytes), nil
}

func (s *Server) deleteCustomContent(msg *CustomContentMessage) {
	var err error
	if msg.Type == "template" {
		err = s.customStorage.DeleteTemplate(msg.ID)
	} else if msg.Type == "script" {
		err = s.customStorage.DeleteScript(msg.ID)
	} else {
		s.logger.Error("Unknown custom content type for delete", zap.String("type", msg.Type))
		return
	}
	if err != nil {
		s.logger.Error("Failed to delete custom content", zap.String("type", msg.Type), zap.String("id", msg.ID), zap.Error(err))
	} else {
		s.logger.Info("Custom content deleted", zap.String("type", msg.Type), zap.String("id", msg.ID))
	}
}
