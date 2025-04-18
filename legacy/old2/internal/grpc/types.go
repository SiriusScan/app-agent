package grpc

// This file previously contained custom type definitions for gRPC communication.
// These have been replaced with proper Protocol Buffer generated types.
//
// The generated types are now located in:
//   - proto/agent.pb.go (message definitions)
//   - proto/agent_grpc.pb.go (service definitions)
//
// Please use the generated types from the 'proto' package instead of the
// custom types that were previously defined here.
//
// Example:
//   import "github.com/SiriusScan/app-agent/proto"
//
//   heartbeat := &proto.HeartbeatMessage{...}
//   message := &proto.AgentMessage{...}
