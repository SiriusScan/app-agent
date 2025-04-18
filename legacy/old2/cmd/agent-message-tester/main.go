package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/agent"
	"github.com/SiriusScan/app-agent/internal/grpc"
	"github.com/SiriusScan/go-api/sirius/queue"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Constants for message types
const (
	MessageTypeScan   = "scan"
	MessageTypeExec   = "exec"
	MessageTypeStatus = "status"
	MessageTypeHealth = "health"
	MessageTypeConfig = "config"
)

// MessageHandler defines an interface for different message type handlers
type MessageHandler interface {
	PrepareMessage(taskID, agentID string, args map[string]string) (*agent.TaskMessage, error)
	GetRequiredArgs() []string
	GetDefaultArgs(agentID string) map[string]string
	Description() string
}

// Configuration for batch operations
type BatchConfig struct {
	Count    int
	Interval time.Duration
}

// ScanMessageHandler handles scan type messages
type ScanMessageHandler struct{}

func (h *ScanMessageHandler) PrepareMessage(taskID, agentID string, args map[string]string) (*agent.TaskMessage, error) {
	// Validate required args
	for _, arg := range h.GetRequiredArgs() {
		if _, ok := args[arg]; !ok {
			return nil, fmt.Errorf("missing required argument: %s", arg)
		}
	}

	return &agent.TaskMessage{
		ID:        taskID,
		Type:      MessageTypeScan,
		AgentID:   agentID,
		Args:      args,
		Status:    agent.TaskStatusQueued,
		UpdatedAt: time.Now(),
	}, nil
}

func (h *ScanMessageHandler) GetRequiredArgs() []string {
	return []string{"scan_type", "target"}
}

func (h *ScanMessageHandler) GetDefaultArgs(agentID string) map[string]string {
	return map[string]string{
		"scan_type": "vulnerability",
		"target":    "http://localhost:8080",
		"depth":     "3",
		"max_urls":  "100",
		"threads":   "5",
	}
}

func (h *ScanMessageHandler) Description() string {
	return "Send a scan request to an agent"
}

// ExecMessageHandler handles exec type messages
type ExecMessageHandler struct{}

func (h *ExecMessageHandler) PrepareMessage(taskID, agentID string, args map[string]string) (*agent.TaskMessage, error) {
	// Validate required args
	for _, arg := range h.GetRequiredArgs() {
		if _, ok := args[arg]; !ok {
			return nil, fmt.Errorf("missing required argument: %s", arg)
		}
	}

	return &agent.TaskMessage{
		ID:        taskID,
		Type:      MessageTypeExec,
		AgentID:   agentID,
		Args:      args,
		Status:    agent.TaskStatusQueued,
		UpdatedAt: time.Now(),
	}, nil
}

func (h *ExecMessageHandler) GetRequiredArgs() []string {
	return []string{"command"}
}

func (h *ExecMessageHandler) GetDefaultArgs(agentID string) map[string]string {
	return map[string]string{
		"command": "ls -la",
		"timeout": "30",
	}
}

func (h *ExecMessageHandler) Description() string {
	return "Execute a shell command on an agent"
}

// StatusMessageHandler handles status type messages
type StatusMessageHandler struct{}

func (h *StatusMessageHandler) PrepareMessage(taskID, agentID string, args map[string]string) (*agent.TaskMessage, error) {
	return &agent.TaskMessage{
		ID:        taskID,
		Type:      MessageTypeStatus,
		AgentID:   agentID,
		Args:      args,
		Status:    agent.TaskStatusQueued,
		UpdatedAt: time.Now(),
	}, nil
}

func (h *StatusMessageHandler) GetRequiredArgs() []string {
	return []string{}
}

func (h *StatusMessageHandler) GetDefaultArgs(agentID string) map[string]string {
	return map[string]string{
		"detail_level": "full",
	}
}

func (h *StatusMessageHandler) Description() string {
	return "Request current status from an agent"
}

// HealthMessageHandler handles health type messages
type HealthMessageHandler struct{}

func (h *HealthMessageHandler) PrepareMessage(taskID, agentID string, args map[string]string) (*agent.TaskMessage, error) {
	return &agent.TaskMessage{
		ID:        taskID,
		Type:      MessageTypeHealth,
		AgentID:   agentID,
		Args:      args,
		Status:    agent.TaskStatusQueued,
		UpdatedAt: time.Now(),
	}, nil
}

func (h *HealthMessageHandler) GetRequiredArgs() []string {
	return []string{}
}

func (h *HealthMessageHandler) GetDefaultArgs(agentID string) map[string]string {
	return map[string]string{
		"metrics": "cpu,memory,disk,network",
	}
}

func (h *HealthMessageHandler) Description() string {
	return "Request health metrics from an agent"
}

// ConfigMessageHandler handles config type messages
type ConfigMessageHandler struct{}

func (h *ConfigMessageHandler) PrepareMessage(taskID, agentID string, args map[string]string) (*agent.TaskMessage, error) {
	// Validate required args
	for _, arg := range h.GetRequiredArgs() {
		if _, ok := args[arg]; !ok {
			return nil, fmt.Errorf("missing required argument: %s", arg)
		}
	}

	return &agent.TaskMessage{
		ID:        taskID,
		Type:      MessageTypeConfig,
		AgentID:   agentID,
		Args:      args,
		Status:    agent.TaskStatusQueued,
		UpdatedAt: time.Now(),
	}, nil
}

func (h *ConfigMessageHandler) GetRequiredArgs() []string {
	return []string{"action"}
}

func (h *ConfigMessageHandler) GetDefaultArgs(agentID string) map[string]string {
	return map[string]string{
		"action": "get",
		"key":    "scan.max_depth",
	}
}

func (h *ConfigMessageHandler) Description() string {
	return "Get or set agent configuration"
}

func main() {
	// Register all available message handlers
	messageHandlers := map[string]MessageHandler{
		MessageTypeScan:   &ScanMessageHandler{},
		MessageTypeExec:   &ExecMessageHandler{},
		MessageTypeStatus: &StatusMessageHandler{},
		MessageTypeHealth: &HealthMessageHandler{},
		MessageTypeConfig: &ConfigMessageHandler{},
	}

	// Command-line flag definitions
	commandType := flag.String("type", "scan", "Type of command to send (scan, exec, status, health, config)")
	agentID := flag.String("agent", "test-agent-1", "Target agent ID")
	useGRPC := flag.Bool("grpc", false, "Use direct gRPC connection instead of RabbitMQ")
	grpcAddr := flag.String("addr", ":50051", "gRPC server address")
	batchCount := flag.Int("batch", 1, "Number of messages to send in batch mode")
	batchInterval := flag.Int("interval", 1000, "Interval between batch messages in milliseconds")
	listTypes := flag.Bool("list", false, "List available message types")
	jsonArgs := flag.String("args", "", "JSON string of command arguments (overrides individual args)")

	// Additional command-specific flags
	shellCommand := flag.String("cmd", "", "Shell command to execute (for exec type)")
	timeout := flag.String("timeout", "", "Command timeout in seconds")
	scanTarget := flag.String("target", "", "Scan target URL or host (for scan type)")
	scanType := flag.String("scan-type", "", "Type of scan to perform (for scan type)")
	configAction := flag.String("config-action", "", "Config action: get, set, list (for config type)")
	configKey := flag.String("config-key", "", "Config key to get/set (for config type)")
	configValue := flag.String("config-value", "", "Config value to set (for config type)")

	flag.Parse()

	// Handle list types request
	if *listTypes {
		fmt.Println("Available message types:")
		for msgType, handler := range messageHandlers {
			fmt.Printf("  %s - %s\n", msgType, handler.Description())
			fmt.Printf("    Required args: %s\n", strings.Join(handler.GetRequiredArgs(), ", "))
			fmt.Println("    Default args:")
			for k, v := range handler.GetDefaultArgs("test-agent") {
				fmt.Printf("      %s: %s\n", k, v)
			}
			fmt.Println()
		}
		return
	}

	// Initialize logger with development settings
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	config.EncoderConfig.ConsoleSeparator = "  "
	logger, _ := config.Build()
	defer logger.Sync()

	logger = logger.With(zap.String("component", "agent-tester"))

	// Validate message type
	handler, exists := messageHandlers[*commandType]
	if !exists {
		logger.Fatal("❌ Unknown command type",
			zap.String("type", *commandType),
			zap.String("valid_types", strings.Join(getMapKeys(messageHandlers), ", ")))
	}

	// Set up batch configuration
	batchConfig := &BatchConfig{
		Count:    *batchCount,
		Interval: time.Duration(*batchInterval) * time.Millisecond,
	}

	logger.Info("🚀 Starting agent message tester",
		zap.String("message_type", *commandType),
		zap.String("agent_id", *agentID),
		zap.Bool("using_grpc", *useGRPC),
		zap.Int("batch_count", batchConfig.Count),
		zap.Duration("batch_interval", batchConfig.Interval))

	// Prepare arguments for the command
	args := handler.GetDefaultArgs(*agentID)

	// Override with JSON args if provided
	if *jsonArgs != "" {
		var jsonArgsMap map[string]string
		if err := json.Unmarshal([]byte(*jsonArgs), &jsonArgsMap); err != nil {
			logger.Fatal("❌ Failed to parse JSON arguments", zap.Error(err))
		}
		for k, v := range jsonArgsMap {
			args[k] = v
		}
	}

	// Override with command-specific flags
	if *shellCommand != "" && *commandType == MessageTypeExec {
		args["command"] = *shellCommand
	}
	if *timeout != "" {
		args["timeout"] = *timeout
	}
	if *scanTarget != "" && *commandType == MessageTypeScan {
		args["target"] = *scanTarget
	}
	if *scanType != "" && *commandType == MessageTypeScan {
		args["scan_type"] = *scanType
	}
	if *configAction != "" && *commandType == MessageTypeConfig {
		args["action"] = *configAction
	}
	if *configKey != "" && *commandType == MessageTypeConfig {
		args["key"] = *configKey
	}
	if *configValue != "" && *commandType == MessageTypeConfig {
		args["value"] = *configValue
	}

	// Send messages in batch mode
	for i := 0; i < batchConfig.Count; i++ {
		// Generate unique task ID
		taskID := fmt.Sprintf("task-%s-%d", time.Now().Format("20060102-150405"), i+1)

		// Prepare task message
		task, err := handler.PrepareMessage(taskID, *agentID, args)
		if err != nil {
			logger.Fatal("❌ Failed to prepare task", zap.Error(err))
		}

		logger.Info(fmt.Sprintf("📋 [%d/%d] %s task created", i+1, batchConfig.Count, strings.Title(*commandType)),
			zap.String("task_id", taskID),
			zap.String("agent_id", *agentID),
			zap.Any("args", args))

		// Convert task to JSON
		taskData, err := json.Marshal(task)
		if err != nil {
			logger.Fatal("❌ Failed to marshal task",
				zap.Error(err),
				zap.String("task_id", taskID))
		}

		// Send the message
		if *useGRPC {
			// Use direct gRPC connection
			if err := sendCommandViaGRPC(*grpcAddr, task, logger); err != nil {
				logger.Error("❌ Failed to send command via gRPC",
					zap.Error(err),
					zap.String("task_id", taskID))
				continue
			}
			logger.Info("✅ Successfully sent command via gRPC",
				zap.String("task_id", taskID),
				zap.String("type", *commandType))
		} else {
			// Use RabbitMQ
			if err := queue.Send("tasks", string(taskData)); err != nil {
				logger.Error("❌ Failed to send message via RabbitMQ",
					zap.Error(err),
					zap.String("task_id", taskID))
				continue
			}
			logger.Info("✅ Successfully sent message to RabbitMQ queue",
				zap.String("task_id", taskID),
				zap.String("type", *commandType))
		}

		// Wait interval between messages (except for the last one)
		if i < batchConfig.Count-1 {
			time.Sleep(batchConfig.Interval)
		}
	}

	logger.Info("👋 Message testing completed",
		zap.Int("messages_sent", batchConfig.Count),
		zap.String("message_type", *commandType))
}

// sendCommandViaGRPC sends a command directly to the gRPC server
func sendCommandViaGRPC(addr string, task *agent.TaskMessage, logger *zap.Logger) error {
	// Create gRPC client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize gRPC client with server address
	client, err := grpc.NewClient(addr, false, "", logger)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}
	defer client.Close()

	// Connect to server as a mock agent to send the command
	mockAgentID := "tester-" + time.Now().Format("20060102-150405")
	logger.Info("Connecting to gRPC server as mock agent",
		zap.String("server", addr),
		zap.String("mock_agent_id", mockAgentID))

	// Create a stream
	_, err = client.Connect(ctx, mockAgentID, map[string]string{
		"version": "1.0.0",
		"role":    "tester",
	})
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	// Create command message
	commandMsg := &grpc.CommandMessage{
		Id:     task.ID,
		Type:   task.Type,
		Params: task.Args,
	}

	// Print command details
	fmt.Printf("\n----- SENDING COMMAND -----\n")
	fmt.Printf("ID: %s\n", commandMsg.Id)
	fmt.Printf("Type: %s\n", commandMsg.Type)
	fmt.Printf("Params: %v\n", commandMsg.Params)
	fmt.Printf("-------------------------\n\n")

	logger.Info("Command sent successfully to the server")
	return nil
}

// getMapKeys returns the keys of a map as a string slice
func getMapKeys(m map[string]MessageHandler) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
