// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: proto/scan/agent.proto

package scan

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

// Agent status values
type AgentStatus int32

const (
	AgentStatus_UNKNOWN     AgentStatus = 0
	AgentStatus_IDLE        AgentStatus = 1
	AgentStatus_BUSY        AgentStatus = 2
	AgentStatus_ERROR       AgentStatus = 3
	AgentStatus_MAINTENANCE AgentStatus = 4
)

// Enum value maps for AgentStatus.
var (
	AgentStatus_name = map[int32]string{
		0: "UNKNOWN",
		1: "IDLE",
		2: "BUSY",
		3: "ERROR",
		4: "MAINTENANCE",
	}
	AgentStatus_value = map[string]int32{
		"UNKNOWN":     0,
		"IDLE":        1,
		"BUSY":        2,
		"ERROR":       3,
		"MAINTENANCE": 4,
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
	return file_proto_scan_agent_proto_enumTypes[0].Descriptor()
}

func (AgentStatus) Type() protoreflect.EnumType {
	return &file_proto_scan_agent_proto_enumTypes[0]
}

func (x AgentStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use AgentStatus.Descriptor instead.
func (AgentStatus) EnumDescriptor() ([]byte, []int) {
	return file_proto_scan_agent_proto_rawDescGZIP(), []int{0}
}

// Simple ping request
type PingRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	AgentId       string                 `protobuf:"bytes,1,opt,name=agent_id,json=agentId,proto3" json:"agent_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PingRequest) Reset() {
	*x = PingRequest{}
	mi := &file_proto_scan_agent_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PingRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PingRequest) ProtoMessage() {}

func (x *PingRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_scan_agent_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PingRequest.ProtoReflect.Descriptor instead.
func (*PingRequest) Descriptor() ([]byte, []int) {
	return file_proto_scan_agent_proto_rawDescGZIP(), []int{0}
}

func (x *PingRequest) GetAgentId() string {
	if x != nil {
		return x.AgentId
	}
	return ""
}

// Simple ping response
type PingResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Message       string                 `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	Timestamp     int64                  `protobuf:"varint,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PingResponse) Reset() {
	*x = PingResponse{}
	mi := &file_proto_scan_agent_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PingResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PingResponse) ProtoMessage() {}

func (x *PingResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_scan_agent_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PingResponse.ProtoReflect.Descriptor instead.
func (*PingResponse) Descriptor() ([]byte, []int) {
	return file_proto_scan_agent_proto_rawDescGZIP(), []int{1}
}

func (x *PingResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *PingResponse) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

// Message sent from agent to server
type AgentMessage struct {
	state   protoimpl.MessageState `protogen:"open.v1"`
	AgentId string                 `protobuf:"bytes,1,opt,name=agent_id,json=agentId,proto3" json:"agent_id,omitempty"`
	Status  AgentStatus            `protobuf:"varint,2,opt,name=status,proto3,enum=scan.AgentStatus" json:"status,omitempty"`
	// Types that are valid to be assigned to Payload:
	//
	//	*AgentMessage_Heartbeat
	//	*AgentMessage_CommandResult
	Payload       isAgentMessage_Payload `protobuf_oneof:"payload"`
	Metadata      map[string]string      `protobuf:"bytes,5,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AgentMessage) Reset() {
	*x = AgentMessage{}
	mi := &file_proto_scan_agent_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AgentMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AgentMessage) ProtoMessage() {}

func (x *AgentMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_scan_agent_proto_msgTypes[2]
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
	return file_proto_scan_agent_proto_rawDescGZIP(), []int{2}
}

func (x *AgentMessage) GetAgentId() string {
	if x != nil {
		return x.AgentId
	}
	return ""
}

func (x *AgentMessage) GetStatus() AgentStatus {
	if x != nil {
		return x.Status
	}
	return AgentStatus_UNKNOWN
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

func (x *AgentMessage) GetCommandResult() *CommandResult {
	if x != nil {
		if x, ok := x.Payload.(*AgentMessage_CommandResult); ok {
			return x.CommandResult
		}
	}
	return nil
}

func (x *AgentMessage) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type isAgentMessage_Payload interface {
	isAgentMessage_Payload()
}

type AgentMessage_Heartbeat struct {
	Heartbeat *HeartbeatMessage `protobuf:"bytes,3,opt,name=heartbeat,proto3,oneof"`
}

type AgentMessage_CommandResult struct {
	CommandResult *CommandResult `protobuf:"bytes,4,opt,name=command_result,json=commandResult,proto3,oneof"`
}

func (*AgentMessage_Heartbeat) isAgentMessage_Payload() {}

func (*AgentMessage_CommandResult) isAgentMessage_Payload() {}

// Message sent from server to agent
type CommandMessage struct {
	state          protoimpl.MessageState `protogen:"open.v1"`
	Id             string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Type           string                 `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	Params         map[string]string      `protobuf:"bytes,3,rep,name=params,proto3" json:"params,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	TimeoutSeconds int64                  `protobuf:"varint,4,opt,name=timeout_seconds,json=timeoutSeconds,proto3" json:"timeout_seconds,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *CommandMessage) Reset() {
	*x = CommandMessage{}
	mi := &file_proto_scan_agent_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CommandMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandMessage) ProtoMessage() {}

func (x *CommandMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_scan_agent_proto_msgTypes[3]
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
	return file_proto_scan_agent_proto_rawDescGZIP(), []int{3}
}

func (x *CommandMessage) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *CommandMessage) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *CommandMessage) GetParams() map[string]string {
	if x != nil {
		return x.Params
	}
	return nil
}

func (x *CommandMessage) GetTimeoutSeconds() int64 {
	if x != nil {
		return x.TimeoutSeconds
	}
	return 0
}

// Heartbeat message
type HeartbeatMessage struct {
	state          protoimpl.MessageState `protogen:"open.v1"`
	Timestamp      int64                  `protobuf:"varint,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	CpuUsage       float64                `protobuf:"fixed64,2,opt,name=cpu_usage,json=cpuUsage,proto3" json:"cpu_usage,omitempty"`
	MemoryUsage    float64                `protobuf:"fixed64,3,opt,name=memory_usage,json=memoryUsage,proto3" json:"memory_usage,omitempty"`
	ActiveCommands int32                  `protobuf:"varint,4,opt,name=active_commands,json=activeCommands,proto3" json:"active_commands,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *HeartbeatMessage) Reset() {
	*x = HeartbeatMessage{}
	mi := &file_proto_scan_agent_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HeartbeatMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HeartbeatMessage) ProtoMessage() {}

func (x *HeartbeatMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_scan_agent_proto_msgTypes[4]
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
	return file_proto_scan_agent_proto_rawDescGZIP(), []int{4}
}

func (x *HeartbeatMessage) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *HeartbeatMessage) GetCpuUsage() float64 {
	if x != nil {
		return x.CpuUsage
	}
	return 0
}

func (x *HeartbeatMessage) GetMemoryUsage() float64 {
	if x != nil {
		return x.MemoryUsage
	}
	return 0
}

func (x *HeartbeatMessage) GetActiveCommands() int32 {
	if x != nil {
		return x.ActiveCommands
	}
	return 0
}

// Command result message
type CommandResult struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	CommandId     string                 `protobuf:"bytes,1,opt,name=command_id,json=commandId,proto3" json:"command_id,omitempty"`
	Status        string                 `protobuf:"bytes,2,opt,name=status,proto3" json:"status,omitempty"`
	Data          map[string]string      `protobuf:"bytes,3,rep,name=data,proto3" json:"data,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Error         string                 `protobuf:"bytes,4,opt,name=error,proto3" json:"error,omitempty"`
	CompletedAt   int64                  `protobuf:"varint,5,opt,name=completed_at,json=completedAt,proto3" json:"completed_at,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CommandResult) Reset() {
	*x = CommandResult{}
	mi := &file_proto_scan_agent_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CommandResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandResult) ProtoMessage() {}

func (x *CommandResult) ProtoReflect() protoreflect.Message {
	mi := &file_proto_scan_agent_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CommandResult.ProtoReflect.Descriptor instead.
func (*CommandResult) Descriptor() ([]byte, []int) {
	return file_proto_scan_agent_proto_rawDescGZIP(), []int{5}
}

func (x *CommandResult) GetCommandId() string {
	if x != nil {
		return x.CommandId
	}
	return ""
}

func (x *CommandResult) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *CommandResult) GetData() map[string]string {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *CommandResult) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

func (x *CommandResult) GetCompletedAt() int64 {
	if x != nil {
		return x.CompletedAt
	}
	return 0
}

var File_proto_scan_agent_proto protoreflect.FileDescriptor

const file_proto_scan_agent_proto_rawDesc = "" +
	"\n" +
	"\x16proto/scan/agent.proto\x12\x04scan\"(\n" +
	"\vPingRequest\x12\x19\n" +
	"\bagent_id\x18\x01 \x01(\tR\aagentId\"F\n" +
	"\fPingResponse\x12\x18\n" +
	"\amessage\x18\x01 \x01(\tR\amessage\x12\x1c\n" +
	"\ttimestamp\x18\x02 \x01(\x03R\ttimestamp\"\xd0\x02\n" +
	"\fAgentMessage\x12\x19\n" +
	"\bagent_id\x18\x01 \x01(\tR\aagentId\x12)\n" +
	"\x06status\x18\x02 \x01(\x0e2\x11.scan.AgentStatusR\x06status\x126\n" +
	"\theartbeat\x18\x03 \x01(\v2\x16.scan.HeartbeatMessageH\x00R\theartbeat\x12<\n" +
	"\x0ecommand_result\x18\x04 \x01(\v2\x13.scan.CommandResultH\x00R\rcommandResult\x12<\n" +
	"\bmetadata\x18\x05 \x03(\v2 .scan.AgentMessage.MetadataEntryR\bmetadata\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01B\t\n" +
	"\apayload\"\xd2\x01\n" +
	"\x0eCommandMessage\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12\x12\n" +
	"\x04type\x18\x02 \x01(\tR\x04type\x128\n" +
	"\x06params\x18\x03 \x03(\v2 .scan.CommandMessage.ParamsEntryR\x06params\x12'\n" +
	"\x0ftimeout_seconds\x18\x04 \x01(\x03R\x0etimeoutSeconds\x1a9\n" +
	"\vParamsEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"\x99\x01\n" +
	"\x10HeartbeatMessage\x12\x1c\n" +
	"\ttimestamp\x18\x01 \x01(\x03R\ttimestamp\x12\x1b\n" +
	"\tcpu_usage\x18\x02 \x01(\x01R\bcpuUsage\x12!\n" +
	"\fmemory_usage\x18\x03 \x01(\x01R\vmemoryUsage\x12'\n" +
	"\x0factive_commands\x18\x04 \x01(\x05R\x0eactiveCommands\"\xeb\x01\n" +
	"\rCommandResult\x12\x1d\n" +
	"\n" +
	"command_id\x18\x01 \x01(\tR\tcommandId\x12\x16\n" +
	"\x06status\x18\x02 \x01(\tR\x06status\x121\n" +
	"\x04data\x18\x03 \x03(\v2\x1d.scan.CommandResult.DataEntryR\x04data\x12\x14\n" +
	"\x05error\x18\x04 \x01(\tR\x05error\x12!\n" +
	"\fcompleted_at\x18\x05 \x01(\x03R\vcompletedAt\x1a7\n" +
	"\tDataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01*J\n" +
	"\vAgentStatus\x12\v\n" +
	"\aUNKNOWN\x10\x00\x12\b\n" +
	"\x04IDLE\x10\x01\x12\b\n" +
	"\x04BUSY\x10\x02\x12\t\n" +
	"\x05ERROR\x10\x03\x12\x0f\n" +
	"\vMAINTENANCE\x10\x042v\n" +
	"\fAgentService\x12-\n" +
	"\x04Ping\x12\x11.scan.PingRequest\x1a\x12.scan.PingResponse\x127\n" +
	"\aConnect\x12\x12.scan.AgentMessage\x1a\x14.scan.CommandMessage(\x010\x01B,Z*github.com/SiriusScan/app-agent/proto/scanb\x06proto3"

var (
	file_proto_scan_agent_proto_rawDescOnce sync.Once
	file_proto_scan_agent_proto_rawDescData []byte
)

func file_proto_scan_agent_proto_rawDescGZIP() []byte {
	file_proto_scan_agent_proto_rawDescOnce.Do(func() {
		file_proto_scan_agent_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proto_scan_agent_proto_rawDesc), len(file_proto_scan_agent_proto_rawDesc)))
	})
	return file_proto_scan_agent_proto_rawDescData
}

var file_proto_scan_agent_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proto_scan_agent_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_proto_scan_agent_proto_goTypes = []any{
	(AgentStatus)(0),         // 0: scan.AgentStatus
	(*PingRequest)(nil),      // 1: scan.PingRequest
	(*PingResponse)(nil),     // 2: scan.PingResponse
	(*AgentMessage)(nil),     // 3: scan.AgentMessage
	(*CommandMessage)(nil),   // 4: scan.CommandMessage
	(*HeartbeatMessage)(nil), // 5: scan.HeartbeatMessage
	(*CommandResult)(nil),    // 6: scan.CommandResult
	nil,                      // 7: scan.AgentMessage.MetadataEntry
	nil,                      // 8: scan.CommandMessage.ParamsEntry
	nil,                      // 9: scan.CommandResult.DataEntry
}
var file_proto_scan_agent_proto_depIdxs = []int32{
	0, // 0: scan.AgentMessage.status:type_name -> scan.AgentStatus
	5, // 1: scan.AgentMessage.heartbeat:type_name -> scan.HeartbeatMessage
	6, // 2: scan.AgentMessage.command_result:type_name -> scan.CommandResult
	7, // 3: scan.AgentMessage.metadata:type_name -> scan.AgentMessage.MetadataEntry
	8, // 4: scan.CommandMessage.params:type_name -> scan.CommandMessage.ParamsEntry
	9, // 5: scan.CommandResult.data:type_name -> scan.CommandResult.DataEntry
	1, // 6: scan.AgentService.Ping:input_type -> scan.PingRequest
	3, // 7: scan.AgentService.Connect:input_type -> scan.AgentMessage
	2, // 8: scan.AgentService.Ping:output_type -> scan.PingResponse
	4, // 9: scan.AgentService.Connect:output_type -> scan.CommandMessage
	8, // [8:10] is the sub-list for method output_type
	6, // [6:8] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_proto_scan_agent_proto_init() }
func file_proto_scan_agent_proto_init() {
	if File_proto_scan_agent_proto != nil {
		return
	}
	file_proto_scan_agent_proto_msgTypes[2].OneofWrappers = []any{
		(*AgentMessage_Heartbeat)(nil),
		(*AgentMessage_CommandResult)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_scan_agent_proto_rawDesc), len(file_proto_scan_agent_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_scan_agent_proto_goTypes,
		DependencyIndexes: file_proto_scan_agent_proto_depIdxs,
		EnumInfos:         file_proto_scan_agent_proto_enumTypes,
		MessageInfos:      file_proto_scan_agent_proto_msgTypes,
	}.Build()
	File_proto_scan_agent_proto = out.File
	file_proto_scan_agent_proto_goTypes = nil
	file_proto_scan_agent_proto_depIdxs = nil
}
