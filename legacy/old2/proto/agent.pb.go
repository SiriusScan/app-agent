// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: proto/agent.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// AgentStatus represents the current state of an agent
type AgentStatus int32

const (
	AgentStatus_AGENT_STATUS_UNSPECIFIED AgentStatus = 0
	AgentStatus_AGENT_STATUS_READY       AgentStatus = 1
	AgentStatus_AGENT_STATUS_BUSY        AgentStatus = 2
	AgentStatus_AGENT_STATUS_ERROR       AgentStatus = 3
)

// Enum value maps for AgentStatus.
var (
	AgentStatus_name = map[int32]string{
		0: "AGENT_STATUS_UNSPECIFIED",
		1: "AGENT_STATUS_READY",
		2: "AGENT_STATUS_BUSY",
		3: "AGENT_STATUS_ERROR",
	}
	AgentStatus_value = map[string]int32{
		"AGENT_STATUS_UNSPECIFIED": 0,
		"AGENT_STATUS_READY":       1,
		"AGENT_STATUS_BUSY":        2,
		"AGENT_STATUS_ERROR":       3,
	}
)

func (x AgentStatus) Enum() *AgentStatus {
	p := new(AgentStatus)
	*p = x
	return p
}

func (x AgentStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (AgentStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_agent_proto_enumTypes[0].Descriptor()
}

func (AgentStatus) Type() protoreflect.EnumType {
	return &file_proto_agent_proto_enumTypes[0]
}

func (x AgentStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use AgentStatus.Descriptor instead.
func (AgentStatus) EnumDescriptor() ([]byte, []int) {
	return file_proto_agent_proto_rawDescGZIP(), []int{0}
}

// ResultStatus represents the status of a command execution
type ResultStatus int32

const (
	ResultStatus_RESULT_STATUS_UNSPECIFIED ResultStatus = 0
	ResultStatus_RESULT_STATUS_SUCCESS     ResultStatus = 1
	ResultStatus_RESULT_STATUS_ERROR       ResultStatus = 2
	ResultStatus_RESULT_STATUS_TIMEOUT     ResultStatus = 3
	ResultStatus_RESULT_STATUS_PENDING     ResultStatus = 4
	ResultStatus_RESULT_STATUS_UNKNOWN     ResultStatus = 5
)

// Enum value maps for ResultStatus.
var (
	ResultStatus_name = map[int32]string{
		0: "RESULT_STATUS_UNSPECIFIED",
		1: "RESULT_STATUS_SUCCESS",
		2: "RESULT_STATUS_ERROR",
		3: "RESULT_STATUS_TIMEOUT",
		4: "RESULT_STATUS_PENDING",
		5: "RESULT_STATUS_UNKNOWN",
	}
	ResultStatus_value = map[string]int32{
		"RESULT_STATUS_UNSPECIFIED": 0,
		"RESULT_STATUS_SUCCESS":     1,
		"RESULT_STATUS_ERROR":       2,
		"RESULT_STATUS_TIMEOUT":     3,
		"RESULT_STATUS_PENDING":     4,
		"RESULT_STATUS_UNKNOWN":     5,
	}
)

func (x ResultStatus) Enum() *ResultStatus {
	p := new(ResultStatus)
	*p = x
	return p
}

func (x ResultStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ResultStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_agent_proto_enumTypes[1].Descriptor()
}

func (ResultStatus) Type() protoreflect.EnumType {
	return &file_proto_agent_proto_enumTypes[1]
}

func (x ResultStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ResultStatus.Descriptor instead.
func (ResultStatus) EnumDescriptor() ([]byte, []int) {
	return file_proto_agent_proto_rawDescGZIP(), []int{1}
}

// AgentMessage represents messages sent from agents to the server
type AgentMessage struct {
	state    protoimpl.MessageState `protogen:"open.v1"`
	AgentId  string                 `protobuf:"bytes,1,opt,name=agent_id,json=agentId,proto3" json:"agent_id,omitempty"`
	Version  string                 `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
	Status   AgentStatus            `protobuf:"varint,3,opt,name=status,proto3,enum=sirius_app_agent.AgentStatus" json:"status,omitempty"`
	Metadata map[string]string      `protobuf:"bytes,4,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// Types that are valid to be assigned to Payload:
	//
	//	*AgentMessage_Heartbeat
	//	*AgentMessage_Auth
	//	*AgentMessage_Result
	Payload       isAgentMessage_Payload `protobuf_oneof:"payload"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AgentMessage) Reset() {
	*x = AgentMessage{}
	mi := &file_proto_agent_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AgentMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AgentMessage) ProtoMessage() {}

func (x *AgentMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_agent_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AgentMessage.ProtoReflect.Descriptor instead.
func (*AgentMessage) Descriptor() ([]byte, []int) {
	return file_proto_agent_proto_rawDescGZIP(), []int{0}
}

func (x *AgentMessage) GetAgentId() string {
	if x != nil {
		return x.AgentId
	}
	return ""
}

func (x *AgentMessage) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *AgentMessage) GetStatus() AgentStatus {
	if x != nil {
		return x.Status
	}
	return AgentStatus_AGENT_STATUS_UNSPECIFIED
}

func (x *AgentMessage) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *AgentMessage) GetPayload() isAgentMessage_Payload {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *AgentMessage) GetHeartbeat() *HeartbeatMessage {
	if x != nil {
		if x, ok := x.Payload.(*AgentMessage_Heartbeat); ok {
			return x.Heartbeat
		}
	}
	return nil
}

func (x *AgentMessage) GetAuth() *AuthRequest {
	if x != nil {
		if x, ok := x.Payload.(*AgentMessage_Auth); ok {
			return x.Auth
		}
	}
	return nil
}

func (x *AgentMessage) GetResult() *ResultMessage {
	if x != nil {
		if x, ok := x.Payload.(*AgentMessage_Result); ok {
			return x.Result
		}
	}
	return nil
}

type isAgentMessage_Payload interface {
	isAgentMessage_Payload()
}

type AgentMessage_Heartbeat struct {
	Heartbeat *HeartbeatMessage `protobuf:"bytes,5,opt,name=heartbeat,proto3,oneof"`
}

type AgentMessage_Auth struct {
	Auth *AuthRequest `protobuf:"bytes,6,opt,name=auth,proto3,oneof"`
}

type AgentMessage_Result struct {
	Result *ResultMessage `protobuf:"bytes,7,opt,name=result,proto3,oneof"`
}

func (*AgentMessage_Heartbeat) isAgentMessage_Payload() {}

func (*AgentMessage_Auth) isAgentMessage_Payload() {}

func (*AgentMessage_Result) isAgentMessage_Payload() {}

// CommandMessage represents commands sent from the server to agents
type CommandMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	CommandId     string                 `protobuf:"bytes,1,opt,name=command_id,json=commandId,proto3" json:"command_id,omitempty"`
	AgentId       string                 `protobuf:"bytes,2,opt,name=agent_id,json=agentId,proto3" json:"agent_id,omitempty"`
	Type          string                 `protobuf:"bytes,3,opt,name=type,proto3" json:"type,omitempty"`
	Parameters    map[string]string      `protobuf:"bytes,4,rep,name=parameters,proto3" json:"parameters,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	TimeoutMs     int64                  `protobuf:"varint,5,opt,name=timeout_ms,json=timeoutMs,proto3" json:"timeout_ms,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CommandMessage) Reset() {
	*x = CommandMessage{}
	mi := &file_proto_agent_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CommandMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandMessage) ProtoMessage() {}

func (x *CommandMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_agent_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CommandMessage.ProtoReflect.Descriptor instead.
func (*CommandMessage) Descriptor() ([]byte, []int) {
	return file_proto_agent_proto_rawDescGZIP(), []int{1}
}

func (x *CommandMessage) GetCommandId() string {
	if x != nil {
		return x.CommandId
	}
	return ""
}

func (x *CommandMessage) GetAgentId() string {
	if x != nil {
		return x.AgentId
	}
	return ""
}

func (x *CommandMessage) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *CommandMessage) GetParameters() map[string]string {
	if x != nil {
		return x.Parameters
	}
	return nil
}

func (x *CommandMessage) GetTimeoutMs() int64 {
	if x != nil {
		return x.TimeoutMs
	}
	return 0
}

// ResultMessage represents command execution results
type ResultMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	CommandId     string                 `protobuf:"bytes,1,opt,name=command_id,json=commandId,proto3" json:"command_id,omitempty"`
	AgentId       string                 `protobuf:"bytes,2,opt,name=agent_id,json=agentId,proto3" json:"agent_id,omitempty"`
	Status        ResultStatus           `protobuf:"varint,3,opt,name=status,proto3,enum=sirius_app_agent.ResultStatus" json:"status,omitempty"`
	Data          map[string]string      `protobuf:"bytes,4,rep,name=data,proto3" json:"data,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	ErrorMessage  string                 `protobuf:"bytes,5,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
	Output        string                 `protobuf:"bytes,6,opt,name=output,proto3" json:"output,omitempty"`                                                                               // Command output text
	ExitCode      int32                  `protobuf:"varint,7,opt,name=exit_code,json=exitCode,proto3" json:"exit_code,omitempty"`                                                          // Command exit code
	Type          string                 `protobuf:"bytes,8,opt,name=type,proto3" json:"type,omitempty"`                                                                                   // Result type
	Metadata      map[string]string      `protobuf:"bytes,9,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"` // Additional metadata
	CompletedAt   int64                  `protobuf:"varint,10,opt,name=completed_at,json=completedAt,proto3" json:"completed_at,omitempty"`                                                // Completion timestamp
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ResultMessage) Reset() {
	*x = ResultMessage{}
	mi := &file_proto_agent_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ResultMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResultMessage) ProtoMessage() {}

func (x *ResultMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_agent_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResultMessage.ProtoReflect.Descriptor instead.
func (*ResultMessage) Descriptor() ([]byte, []int) {
	return file_proto_agent_proto_rawDescGZIP(), []int{2}
}

func (x *ResultMessage) GetCommandId() string {
	if x != nil {
		return x.CommandId
	}
	return ""
}

func (x *ResultMessage) GetAgentId() string {
	if x != nil {
		return x.AgentId
	}
	return ""
}

func (x *ResultMessage) GetStatus() ResultStatus {
	if x != nil {
		return x.Status
	}
	return ResultStatus_RESULT_STATUS_UNSPECIFIED
}

func (x *ResultMessage) GetData() map[string]string {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *ResultMessage) GetErrorMessage() string {
	if x != nil {
		return x.ErrorMessage
	}
	return ""
}

func (x *ResultMessage) GetOutput() string {
	if x != nil {
		return x.Output
	}
	return ""
}

func (x *ResultMessage) GetExitCode() int32 {
	if x != nil {
		return x.ExitCode
	}
	return 0
}

func (x *ResultMessage) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *ResultMessage) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *ResultMessage) GetCompletedAt() int64 {
	if x != nil {
		return x.CompletedAt
	}
	return 0
}

// ResultResponse represents the server's acknowledgment of results
type ResultResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	CommandId     string                 `protobuf:"bytes,1,opt,name=command_id,json=commandId,proto3" json:"command_id,omitempty"`
	Accepted      bool                   `protobuf:"varint,2,opt,name=accepted,proto3" json:"accepted,omitempty"`
	Message       string                 `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ResultResponse) Reset() {
	*x = ResultResponse{}
	mi := &file_proto_agent_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ResultResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResultResponse) ProtoMessage() {}

func (x *ResultResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_agent_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResultResponse.ProtoReflect.Descriptor instead.
func (*ResultResponse) Descriptor() ([]byte, []int) {
	return file_proto_agent_proto_rawDescGZIP(), []int{3}
}

func (x *ResultResponse) GetCommandId() string {
	if x != nil {
		return x.CommandId
	}
	return ""
}

func (x *ResultResponse) GetAccepted() bool {
	if x != nil {
		return x.Accepted
	}
	return false
}

func (x *ResultResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

// HeartbeatMessage represents agent heartbeat data
type HeartbeatMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Timestamp     int64                  `protobuf:"varint,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Metrics       map[string]string      `protobuf:"bytes,2,rep,name=metrics,proto3" json:"metrics,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HeartbeatMessage) Reset() {
	*x = HeartbeatMessage{}
	mi := &file_proto_agent_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HeartbeatMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HeartbeatMessage) ProtoMessage() {}

func (x *HeartbeatMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_agent_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HeartbeatMessage.ProtoReflect.Descriptor instead.
func (*HeartbeatMessage) Descriptor() ([]byte, []int) {
	return file_proto_agent_proto_rawDescGZIP(), []int{4}
}

func (x *HeartbeatMessage) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *HeartbeatMessage) GetMetrics() map[string]string {
	if x != nil {
		return x.Metrics
	}
	return nil
}

// AuthRequest represents agent authentication data
type AuthRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	AgentId       string                 `protobuf:"bytes,1,opt,name=agent_id,json=agentId,proto3" json:"agent_id,omitempty"`
	AuthToken     string                 `protobuf:"bytes,2,opt,name=auth_token,json=authToken,proto3" json:"auth_token,omitempty"`
	Capabilities  map[string]string      `protobuf:"bytes,3,rep,name=capabilities,proto3" json:"capabilities,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AuthRequest) Reset() {
	*x = AuthRequest{}
	mi := &file_proto_agent_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AuthRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AuthRequest) ProtoMessage() {}

func (x *AuthRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_agent_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AuthRequest.ProtoReflect.Descriptor instead.
func (*AuthRequest) Descriptor() ([]byte, []int) {
	return file_proto_agent_proto_rawDescGZIP(), []int{5}
}

func (x *AuthRequest) GetAgentId() string {
	if x != nil {
		return x.AgentId
	}
	return ""
}

func (x *AuthRequest) GetAuthToken() string {
	if x != nil {
		return x.AuthToken
	}
	return ""
}

func (x *AuthRequest) GetCapabilities() map[string]string {
	if x != nil {
		return x.Capabilities
	}
	return nil
}

var File_proto_agent_proto protoreflect.FileDescriptor

const file_proto_agent_proto_rawDesc = "" +
	"\n" +
	"\x11proto/agent.proto\x12\x10sirius_app_agent\"\xc0\x03\n" +
	"\fAgentMessage\x12\x19\n" +
	"\bagent_id\x18\x01 \x01(\tR\aagentId\x12\x18\n" +
	"\aversion\x18\x02 \x01(\tR\aversion\x125\n" +
	"\x06status\x18\x03 \x01(\x0e2\x1d.sirius_app_agent.AgentStatusR\x06status\x12H\n" +
	"\bmetadata\x18\x04 \x03(\v2,.sirius_app_agent.AgentMessage.MetadataEntryR\bmetadata\x12B\n" +
	"\theartbeat\x18\x05 \x01(\v2\".sirius_app_agent.HeartbeatMessageH\x00R\theartbeat\x123\n" +
	"\x04auth\x18\x06 \x01(\v2\x1d.sirius_app_agent.AuthRequestH\x00R\x04auth\x129\n" +
	"\x06result\x18\a \x01(\v2\x1f.sirius_app_agent.ResultMessageH\x00R\x06result\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01B\t\n" +
	"\apayload\"\x8e\x02\n" +
	"\x0eCommandMessage\x12\x1d\n" +
	"\n" +
	"command_id\x18\x01 \x01(\tR\tcommandId\x12\x19\n" +
	"\bagent_id\x18\x02 \x01(\tR\aagentId\x12\x12\n" +
	"\x04type\x18\x03 \x01(\tR\x04type\x12P\n" +
	"\n" +
	"parameters\x18\x04 \x03(\v20.sirius_app_agent.CommandMessage.ParametersEntryR\n" +
	"parameters\x12\x1d\n" +
	"\n" +
	"timeout_ms\x18\x05 \x01(\x03R\ttimeoutMs\x1a=\n" +
	"\x0fParametersEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"\x92\x04\n" +
	"\rResultMessage\x12\x1d\n" +
	"\n" +
	"command_id\x18\x01 \x01(\tR\tcommandId\x12\x19\n" +
	"\bagent_id\x18\x02 \x01(\tR\aagentId\x126\n" +
	"\x06status\x18\x03 \x01(\x0e2\x1e.sirius_app_agent.ResultStatusR\x06status\x12=\n" +
	"\x04data\x18\x04 \x03(\v2).sirius_app_agent.ResultMessage.DataEntryR\x04data\x12#\n" +
	"\rerror_message\x18\x05 \x01(\tR\ferrorMessage\x12\x16\n" +
	"\x06output\x18\x06 \x01(\tR\x06output\x12\x1b\n" +
	"\texit_code\x18\a \x01(\x05R\bexitCode\x12\x12\n" +
	"\x04type\x18\b \x01(\tR\x04type\x12I\n" +
	"\bmetadata\x18\t \x03(\v2-.sirius_app_agent.ResultMessage.MetadataEntryR\bmetadata\x12!\n" +
	"\fcompleted_at\x18\n" +
	" \x01(\x03R\vcompletedAt\x1a7\n" +
	"\tDataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"e\n" +
	"\x0eResultResponse\x12\x1d\n" +
	"\n" +
	"command_id\x18\x01 \x01(\tR\tcommandId\x12\x1a\n" +
	"\baccepted\x18\x02 \x01(\bR\baccepted\x12\x18\n" +
	"\amessage\x18\x03 \x01(\tR\amessage\"\xb7\x01\n" +
	"\x10HeartbeatMessage\x12\x1c\n" +
	"\ttimestamp\x18\x01 \x01(\x03R\ttimestamp\x12I\n" +
	"\ametrics\x18\x02 \x03(\v2/.sirius_app_agent.HeartbeatMessage.MetricsEntryR\ametrics\x1a:\n" +
	"\fMetricsEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"\xdd\x01\n" +
	"\vAuthRequest\x12\x19\n" +
	"\bagent_id\x18\x01 \x01(\tR\aagentId\x12\x1d\n" +
	"\n" +
	"auth_token\x18\x02 \x01(\tR\tauthToken\x12S\n" +
	"\fcapabilities\x18\x03 \x03(\v2/.sirius_app_agent.AuthRequest.CapabilitiesEntryR\fcapabilities\x1a?\n" +
	"\x11CapabilitiesEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01*r\n" +
	"\vAgentStatus\x12\x1c\n" +
	"\x18AGENT_STATUS_UNSPECIFIED\x10\x00\x12\x16\n" +
	"\x12AGENT_STATUS_READY\x10\x01\x12\x15\n" +
	"\x11AGENT_STATUS_BUSY\x10\x02\x12\x16\n" +
	"\x12AGENT_STATUS_ERROR\x10\x03*\xb2\x01\n" +
	"\fResultStatus\x12\x1d\n" +
	"\x19RESULT_STATUS_UNSPECIFIED\x10\x00\x12\x19\n" +
	"\x15RESULT_STATUS_SUCCESS\x10\x01\x12\x17\n" +
	"\x13RESULT_STATUS_ERROR\x10\x02\x12\x19\n" +
	"\x15RESULT_STATUS_TIMEOUT\x10\x03\x12\x19\n" +
	"\x15RESULT_STATUS_PENDING\x10\x04\x12\x19\n" +
	"\x15RESULT_STATUS_UNKNOWN\x10\x052\xb6\x01\n" +
	"\fAgentService\x12Q\n" +
	"\aConnect\x12\x1e.sirius_app_agent.AgentMessage\x1a .sirius_app_agent.CommandMessage\"\x00(\x010\x01\x12S\n" +
	"\fReportResult\x12\x1f.sirius_app_agent.ResultMessage\x1a .sirius_app_agent.ResultResponse\"\x00B-Z+github.com/SiriusScan/app-agent/proto;protob\x06proto3"

var (
	file_proto_agent_proto_rawDescOnce sync.Once
	file_proto_agent_proto_rawDescData []byte
)

func file_proto_agent_proto_rawDescGZIP() []byte {
	file_proto_agent_proto_rawDescOnce.Do(func() {
		file_proto_agent_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proto_agent_proto_rawDesc), len(file_proto_agent_proto_rawDesc)))
	})
	return file_proto_agent_proto_rawDescData
}

var file_proto_agent_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_proto_agent_proto_msgTypes = make([]protoimpl.MessageInfo, 12)
var file_proto_agent_proto_goTypes = []any{
	(AgentStatus)(0),         // 0: sirius_app_agent.AgentStatus
	(ResultStatus)(0),        // 1: sirius_app_agent.ResultStatus
	(*AgentMessage)(nil),     // 2: sirius_app_agent.AgentMessage
	(*CommandMessage)(nil),   // 3: sirius_app_agent.CommandMessage
	(*ResultMessage)(nil),    // 4: sirius_app_agent.ResultMessage
	(*ResultResponse)(nil),   // 5: sirius_app_agent.ResultResponse
	(*HeartbeatMessage)(nil), // 6: sirius_app_agent.HeartbeatMessage
	(*AuthRequest)(nil),      // 7: sirius_app_agent.AuthRequest
	nil,                      // 8: sirius_app_agent.AgentMessage.MetadataEntry
	nil,                      // 9: sirius_app_agent.CommandMessage.ParametersEntry
	nil,                      // 10: sirius_app_agent.ResultMessage.DataEntry
	nil,                      // 11: sirius_app_agent.ResultMessage.MetadataEntry
	nil,                      // 12: sirius_app_agent.HeartbeatMessage.MetricsEntry
	nil,                      // 13: sirius_app_agent.AuthRequest.CapabilitiesEntry
}
var file_proto_agent_proto_depIdxs = []int32{
	0,  // 0: sirius_app_agent.AgentMessage.status:type_name -> sirius_app_agent.AgentStatus
	8,  // 1: sirius_app_agent.AgentMessage.metadata:type_name -> sirius_app_agent.AgentMessage.MetadataEntry
	6,  // 2: sirius_app_agent.AgentMessage.heartbeat:type_name -> sirius_app_agent.HeartbeatMessage
	7,  // 3: sirius_app_agent.AgentMessage.auth:type_name -> sirius_app_agent.AuthRequest
	4,  // 4: sirius_app_agent.AgentMessage.result:type_name -> sirius_app_agent.ResultMessage
	9,  // 5: sirius_app_agent.CommandMessage.parameters:type_name -> sirius_app_agent.CommandMessage.ParametersEntry
	1,  // 6: sirius_app_agent.ResultMessage.status:type_name -> sirius_app_agent.ResultStatus
	10, // 7: sirius_app_agent.ResultMessage.data:type_name -> sirius_app_agent.ResultMessage.DataEntry
	11, // 8: sirius_app_agent.ResultMessage.metadata:type_name -> sirius_app_agent.ResultMessage.MetadataEntry
	12, // 9: sirius_app_agent.HeartbeatMessage.metrics:type_name -> sirius_app_agent.HeartbeatMessage.MetricsEntry
	13, // 10: sirius_app_agent.AuthRequest.capabilities:type_name -> sirius_app_agent.AuthRequest.CapabilitiesEntry
	2,  // 11: sirius_app_agent.AgentService.Connect:input_type -> sirius_app_agent.AgentMessage
	4,  // 12: sirius_app_agent.AgentService.ReportResult:input_type -> sirius_app_agent.ResultMessage
	3,  // 13: sirius_app_agent.AgentService.Connect:output_type -> sirius_app_agent.CommandMessage
	5,  // 14: sirius_app_agent.AgentService.ReportResult:output_type -> sirius_app_agent.ResultResponse
	13, // [13:15] is the sub-list for method output_type
	11, // [11:13] is the sub-list for method input_type
	11, // [11:11] is the sub-list for extension type_name
	11, // [11:11] is the sub-list for extension extendee
	0,  // [0:11] is the sub-list for field type_name
}

func init() { file_proto_agent_proto_init() }
func file_proto_agent_proto_init() {
	if File_proto_agent_proto != nil {
		return
	}
	file_proto_agent_proto_msgTypes[0].OneofWrappers = []any{
		(*AgentMessage_Heartbeat)(nil),
		(*AgentMessage_Auth)(nil),
		(*AgentMessage_Result)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_agent_proto_rawDesc), len(file_proto_agent_proto_rawDesc)),
			NumEnums:      2,
			NumMessages:   12,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_agent_proto_goTypes,
		DependencyIndexes: file_proto_agent_proto_depIdxs,
		EnumInfos:         file_proto_agent_proto_enumTypes,
		MessageInfos:      file_proto_agent_proto_msgTypes,
	}.Build()
	File_proto_agent_proto = out.File
	file_proto_agent_proto_goTypes = nil
	file_proto_agent_proto_depIdxs = nil
}
