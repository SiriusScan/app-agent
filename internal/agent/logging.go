package agent

import (
	"fmt"
	"time"

	"github.com/SiriusScan/go-api/sirius/logging"
)

// LoggingClient provides a centralized way to send structured logs to the API
// This is now a wrapper around the SDK's global logging client.
type LoggingClient struct{}

// NewLoggingClient creates a new LoggingClient instance.
// It ensures the SDK's global client is initialized.
func NewLoggingClient() *LoggingClient {
	// The SDK's Init() function is idempotent, safe to call multiple times.
	// It will only initialize if not already initialized.
	logging.Init()
	return &LoggingClient{}
}

// LogAgentEvent logs a general event related to agent operations
func (lc *LoggingClient) LogAgentEvent(eventType, message string, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["event_type"] = eventType
	logging.Log("sirius-agent", "agent-manager", logging.LogLevelInfo, message, metadata, map[string]interface{}{"type": "business_event"})
}

// LogCommandExecution logs the execution of a command
func (lc *LoggingClient) LogCommandExecution(commandType, commandID string, duration time.Duration, success bool, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["command_type"] = commandType
	metadata["command_id"] = commandID
	metadata["duration_ms"] = duration.Milliseconds()
	metadata["success"] = success
	
	level := logging.LogLevelInfo
	if !success {
		level = logging.LogLevelError
	}
	
	message := fmt.Sprintf("Command execution %s: %s", messageFromSuccess(success), commandType)
	logging.Log("sirius-agent", "command-execution", level, message, metadata, map[string]interface{}{"type": "performance_metric"})
}

// LogAgentError logs an error related to agent operations
func (lc *LoggingClient) LogAgentError(commandType, commandID, errorCode, message string, err error) {
	metadata := map[string]interface{}{
		"command_type": commandType,
		"command_id":   commandID,
		"error_code":   errorCode,
		"details":      err.Error(),
	}
	logging.Log("sirius-agent", "agent-manager", logging.LogLevelError, message, metadata, map[string]interface{}{"type": "error"})
}

// LogGRPCOperation logs gRPC-related operations
func (lc *LoggingClient) LogGRPCOperation(operation string, success bool, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["operation"] = operation
	metadata["success"] = success
	
	level := logging.LogLevelInfo
	if !success {
		level = logging.LogLevelError
	}
	
	message := fmt.Sprintf("gRPC operation %s: %s", messageFromSuccess(success), operation)
	logging.Log("sirius-agent", "grpc-client", level, message, metadata, map[string]interface{}{"type": "system_event"})
}

// LogConnectionEvent logs connection-related events
func (lc *LoggingClient) LogConnectionEvent(event string, success bool, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["connection_event"] = event
	metadata["success"] = success
	
	level := logging.LogLevelInfo
	if !success {
		level = logging.LogLevelError
	}
	
	message := fmt.Sprintf("Connection event %s: %s", messageFromSuccess(success), event)
	logging.Log("sirius-agent", "connection-manager", level, message, metadata, map[string]interface{}{"type": "system_event"})
}

// LogHeartbeatEvent logs heartbeat-related events
func (lc *LoggingClient) LogHeartbeatEvent(event string, success bool, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["heartbeat_event"] = event
	metadata["success"] = success
	
	level := logging.LogLevelInfo
	if !success {
		level = logging.LogLevelError
	}
	
	message := fmt.Sprintf("Heartbeat event %s: %s", messageFromSuccess(success), event)
	logging.Log("sirius-agent", "heartbeat-manager", level, message, metadata, map[string]interface{}{"type": "system_event"})
}

// LogSyncOperation logs synchronization operations
func (lc *LoggingClient) LogSyncOperation(operation, resourceType string, success bool, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["operation"] = operation
	metadata["resource_type"] = resourceType
	metadata["success"] = success
	
	level := logging.LogLevelInfo
	if !success {
		level = logging.LogLevelError
	}
	
	message := fmt.Sprintf("Sync operation %s: %s %s", messageFromSuccess(success), operation, resourceType)
	logging.Log("sirius-agent", "sync-manager", level, message, metadata, map[string]interface{}{"type": "business_event"})
}

// LogServiceLifecycle logs service lifecycle events
func (lc *LoggingClient) LogServiceLifecycle(event string, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["lifecycle_event"] = event
	
	logging.Log("sirius-agent", "service-lifecycle", logging.LogLevelInfo, fmt.Sprintf("Service %s", event), metadata, map[string]interface{}{"type": "system_event"})
}

// LogPowerShellOperation logs PowerShell-related operations
func (lc *LoggingClient) LogPowerShellOperation(operation string, success bool, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["operation"] = operation
	metadata["success"] = success
	
	level := logging.LogLevelInfo
	if !success {
		level = logging.LogLevelError
	}
	
	message := fmt.Sprintf("PowerShell operation %s: %s", messageFromSuccess(success), operation)
	logging.Log("sirius-agent", "powershell-manager", level, message, metadata, map[string]interface{}{"type": "system_event"})
}

// LogScanOperation logs scan-related operations
func (lc *LoggingClient) LogScanOperation(operation, target string, success bool, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["operation"] = operation
	metadata["target"] = target
	metadata["success"] = success
	
	level := logging.LogLevelInfo
	if !success {
		level = logging.LogLevelError
	}
	
	message := fmt.Sprintf("Scan operation %s: %s on %s", messageFromSuccess(success), operation, target)
	logging.Log("sirius-agent", "scan-manager", level, message, metadata, map[string]interface{}{"type": "business_event"})
}

func messageFromSuccess(success bool) string {
	if success {
		return "completed successfully"
	}
	return "failed"
}
