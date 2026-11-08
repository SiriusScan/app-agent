package server

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/SiriusScan/go-api/sirius/queue"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

const privateAgentReplyPrefix = "agent_response.rpc."

// Reject partial or arbitrary private destinations without a shared fallback.
func validCommandReplyRoute(cmd CommandMessage) bool {
	if cmd.NoReply {
		return cmd.ResponseQueue == "" && cmd.CorrelationID == ""
	}
	if cmd.ResponseQueue == "" && cmd.CorrelationID == "" {
		return true // Legacy callers retain their original raw response shape.
	}
	id := cmd.CorrelationID
	if id == "" || len(id) > 128 || cmd.ResponseQueue != privateAgentReplyPrefix+id {
		return false
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// Broker-independent routing allows tests to use fake streams and publishers.
func (s *Server) processQueueCommand(message string) {
	var cmd CommandMessage
	if err := json.Unmarshal([]byte(message), &cmd); err != nil {
		// No trustworthy routing envelope: a shared error could poison another
		// caller's next read, so malformed messages are logged without replying.
		s.logger.Error("Invalid agent command JSON", zap.Error(err))
		return
	}
	if !validCommandReplyRoute(cmd) {
		s.logger.Warn("Invalid agent command reply route")
		return
	}
	switch cmd.Action {
	case "list_agents":
		s.handleListAgents(cmd)
	case "initialize_session":
		s.handleInitializeSession(cmd)
	default:
		if cmd.Command != "" {
			s.handleExecuteCommand(cmd)
		} else {
			s.replyToCommand(cmd, CommandResponse{Success: false, Error: "Unknown or empty command"})
		}
	}
}

func (s *Server) replyToCommand(cmd CommandMessage, response interface{}) {
	if cmd.NoReply {
		return
	}
	if !validCommandReplyRoute(cmd) {
		s.logger.Warn("Refusing invalid agent command reply route")
		return
	}
	queueName := AGENT_RESPONSE_QUEUE
	private := cmd.ResponseQueue != ""
	if private {
		queueName = cmd.ResponseQueue
		response = struct {
			CorrelationID string      `json:"correlationId"`
			Response      interface{} `json:"response"`
		}{cmd.CorrelationID, response}
	}
	body, err := json.Marshal(response)
	if err != nil {
		s.logger.Error("Failed to encode agent command reply", zap.Error(err))
		return
	}
	publish := s.commandReplyPublisher
	if publish == nil {
		publish = publishCommandReply
	}
	if err := publish(queueName, string(body), private); err != nil {
		s.logger.Error("Failed to publish agent command reply", zap.String("queue", queueName), zap.Error(err))
	}
}

// Never declare private queues. Their requester owns an exclusive auto-delete
// queue; redeclaration would fail or resurrect an abandoned queue after timeout.
func publishCommandReply(queueName, message string, private bool) error {
	if !private {
		return queue.Send(queueName, message)
	}
	address := strings.TrimSpace(os.Getenv("RABBITMQ_URL"))
	if address == "" {
		address = queue.SIRIUS_RABBITMQ
	}
	connection, err := amqp.Dial(address)
	if err != nil {
		return fmt.Errorf("connect for private command reply: %w", err)
	}
	defer connection.Close()
	channel, err := connection.Channel()
	if err != nil {
		return fmt.Errorf("open private reply channel: %w", err)
	}
	defer channel.Close()
	return publishExistingReply(channel, queueName, message)
}

// This interface cannot declare or purge queues. Late replies to a deleted
// queue are dropped by the broker and never rerouted to the shared queue.
type existingReplyPublisher interface {
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

func publishExistingReply(channel existingReplyPublisher, queueName, message string) error {
	return channel.Publish("", queueName, false, false, amqp.Publishing{
		ContentType: "application/json", Body: []byte(message),
	})
}
