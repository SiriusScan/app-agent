package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/proto/scan"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	agentID    = flag.String("agent", "", "Agent ID to send command to")
	cmdType    = flag.String("type", "STOP", "Command type (SCAN, STOP, UPDATE)")
	params     = flag.String("params", "", "Command parameters in key=value format, comma separated")
	serverAddr = flag.String("server", "localhost:9000", "gRPC server address")
	timeout    = flag.Int64("timeout", 60, "Command timeout in seconds")
)

func main() {
	flag.Parse()

	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	if *agentID == "" {
		logger.Fatal("Agent ID is required")
	}

	// Parse command type
	var commandType scan.CommandType
	switch strings.ToUpper(*cmdType) {
	case "SCAN":
		commandType = scan.CommandType_COMMAND_TYPE_SCAN
	case "STOP":
		commandType = scan.CommandType_COMMAND_TYPE_STOP
	case "UPDATE":
		commandType = scan.CommandType_COMMAND_TYPE_UPDATE
	default:
		logger.Fatal("Invalid command type",
			zap.String("type", *cmdType),
			zap.String("valid_types", "SCAN,STOP,UPDATE"))
	}

	// Parse parameters
	parameters := make(map[string]string)
	if *params != "" {
		paramPairs := strings.Split(*params, ",")
		for _, pair := range paramPairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				parameters[parts[0]] = parts[1]
			}
		}
	}

	// Add default parameters based on command type
	switch commandType {
	case scan.CommandType_COMMAND_TYPE_SCAN:
		if _, ok := parameters["target"]; !ok {
			parameters["target"] = "localhost"
		}
		if _, ok := parameters["scan_type"]; !ok {
			parameters["scan_type"] = "vulnerability"
		}
	case scan.CommandType_COMMAND_TYPE_STOP:
		if _, ok := parameters["reason"]; !ok {
			parameters["reason"] = "user_requested"
		}
	case scan.CommandType_COMMAND_TYPE_UPDATE:
		if _, ok := parameters["update_type"]; !ok {
			parameters["update_type"] = "definitions"
		}
		if _, ok := parameters["version"]; !ok {
			parameters["version"] = "1.0.0"
		}
	}

	// Connect to gRPC server
	logger.Info("Connecting to gRPC server", zap.String("server", *serverAddr))
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("Failed to connect to server", zap.Error(err))
	}
	defer conn.Close()

	// Create client
	client := scan.NewAgentServiceClient(conn)

	// Create command message
	commandID := uuid.New().String()
	command := &scan.CommandMessage{
		CommandId:  commandID,
		AgentId:    *agentID,
		Type:       commandType,
		Parameters: parameters,
		TimeoutMs:  *timeout * 1000, // Convert to milliseconds
	}

	// Send command
	req := &scan.SendCommandRequest{
		AgentId: *agentID,
		Command: command,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger.Info("Sending command",
		zap.String("command_id", commandID),
		zap.String("agent_id", *agentID),
		zap.String("type", commandType.String()),
		zap.Any("parameters", parameters),
	)

	resp, err := client.SendCommand(ctx, req)
	if err != nil {
		logger.Fatal("Failed to send command", zap.Error(err))
	}

	// Check response
	logger.Info("Command sent successfully",
		zap.String("command_id", resp.CommandId),
		zap.Bool("success", resp.Success),
		zap.String("message", resp.Message),
	)

	// Wait for a moment to see if command is processed
	time.Sleep(500 * time.Millisecond)

	// Try to get command result
	resultReq := &scan.GetCommandResultRequest{
		CommandId: commandID,
	}

	resultCtx, resultCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer resultCancel()

	resultResp, err := client.GetCommandResult(resultCtx, resultReq)
	if err != nil {
		logger.Info("Command result not available yet",
			zap.String("command_id", commandID),
			zap.Error(err),
		)
	} else {
		logger.Info("Command result received",
			zap.String("command_id", resultResp.Result.CommandId),
			zap.String("status", resultResp.Result.Status.String()),
			zap.String("output", resultResp.Result.Output),
			zap.String("error", resultResp.Result.ErrorMessage),
		)
	}

	logger.Info("Done")
}
