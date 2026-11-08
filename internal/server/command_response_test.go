package server

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	pb "github.com/SiriusScan/app-agent/proto/hello"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type commandReplyStream struct {
	pb.HelloService_ConnectStreamServer
	send func(*pb.ServerMessage) error
}

func (s commandReplyStream) Send(message *pb.ServerMessage) error { return s.send(message) }

type capturedCommandReply struct {
	queue, body string
	private     bool
}
type commandReplyCapture struct {
	mu      sync.Mutex
	replies []capturedCommandReply
}

func (c *commandReplyCapture) publish(queueName, body string, private bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.replies = append(c.replies, capturedCommandReply{queueName, body, private})
	return nil
}
func (c *commandReplyCapture) snapshot() []capturedCommandReply {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]capturedCommandReply(nil), c.replies...)
}
func commandReplyTestServer(streams map[string]pb.HelloService_ConnectStreamServer) (*Server, *commandReplyCapture) {
	capture := &commandReplyCapture{}
	return &Server{
		logger: zap.NewNop(), agents: streams, pendingCommands: map[string]string{},
		commandReplyPublisher: capture.publish,
	}, capture
}
func commandReplyJSON(t *testing.T, command map[string]interface{}) string {
	t.Helper()
	data, err := json.Marshal(command)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func exactTemplateCommand(agent string) map[string]interface{} {
	return map[string]interface{}{
		"command": "internal:template-scan --template-id=fixture", "timestamp": "fixture-time",
		"target": map[string]string{"type": "agent", "id": agent},
	}
}
func correlatedCommand(command map[string]interface{}, id string) map[string]interface{} {
	command["correlationId"] = id
	command["responseQueue"] = privateAgentReplyPrefix + id
	return command
}

func TestTemplateNoReplyCannotContaminateConcurrentScanACK(t *testing.T) {
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	stream := commandReplyStream{send: func(*pb.ServerMessage) error {
		close(entered)
		<-release
		return nil
	}}
	server, capture := commandReplyTestServer(map[string]pb.HelloService_ConnectStreamServer{"online": stream})
	template := exactTemplateCommand("online")
	template["noReply"] = true
	raw := commandReplyJSON(t, template)
	go func() { defer close(finished); server.processQueueCommand(raw) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("template command did not reach fake stream")
	}
	// The template send is still pending when another caller requests a scan
	// on an offline agent. It must receive its own failure, never an orphan ACK.
	server.processQueueCommand(commandReplyJSON(t, correlatedCommand(exactTemplateCommand("offline"), "scan-request")))
	server.processQueueCommand(commandReplyJSON(t, correlatedCommand(map[string]interface{}{"action": "list_agents"}, "list-request")))
	close(release)
	<-finished
	replies := capture.snapshot()
	if len(replies) != 2 {
		t.Fatalf("unexpected replies: %+v", replies)
	}
	for _, reply := range replies {
		if !reply.private || reply.queue == AGENT_RESPONSE_QUEUE {
			t.Fatalf("orphan shared ACK: %+v", reply)
		}
		var envelope struct {
			CorrelationID string          `json:"correlationId"`
			Response      json.RawMessage `json:"response"`
		}
		if err := json.Unmarshal([]byte(reply.body), &envelope); err != nil {
			t.Fatal(err)
		}
		if reply.queue != privateAgentReplyPrefix+envelope.CorrelationID {
			t.Fatalf("crossed correlation: %+v", reply)
		}
		if envelope.CorrelationID == "scan-request" {
			var response CommandResponse
			if err := json.Unmarshal(envelope.Response, &response); err != nil {
				t.Fatal(err)
			}
			if response.Success || response.Error != "Agent offline not connected" {
				t.Fatalf("wrong scan ACK: %+v", response)
			}
		} else {
			var agents []string
			if err := json.Unmarshal(envelope.Response, &agents); err != nil {
				t.Fatal(err)
			}
			if len(agents) != 1 || agents[0] != "online" {
				t.Fatalf("wrong agent list: %v", agents)
			}
		}
	}
}

func TestNoReplySuppressesAllCommandOutcomes(t *testing.T) {
	for _, outcome := range []string{"success", "invalid-target", "offline", "stream-error", "list", "session", "unknown"} {
		t.Run(outcome, func(t *testing.T) {
			stream := commandReplyStream{send: func(*pb.ServerMessage) error {
				if outcome == "stream-error" {
					return errors.New("fake stream failure")
				}
				return nil
			}}
			server, capture := commandReplyTestServer(map[string]pb.HelloService_ConnectStreamServer{"online": stream})
			command := exactTemplateCommand("online")
			switch outcome {
			case "invalid-target":
				command["target"] = map[string]string{"type": "not-agent", "id": "online"}
			case "offline":
				command["target"] = map[string]string{"type": "agent", "id": "offline"}
			case "list":
				command = map[string]interface{}{"action": "list_agents"}
			case "session":
				command = map[string]interface{}{"action": "initialize_session", "agentId": "offline"}
			case "unknown":
				command = map[string]interface{}{}
			}
			command["noReply"] = true
			server.processQueueCommand(commandReplyJSON(t, command))
			if replies := capture.snapshot(); len(replies) != 0 {
				t.Fatalf("noReply leaked ACK: %+v", replies)
			}
		})
	}
}

func TestCommandACKFollowsStreamOutcome(t *testing.T) {
	for _, fails := range []bool{false, true} {
		var capture *commandReplyCapture
		stream := commandReplyStream{send: func(*pb.ServerMessage) error {
			if len(capture.snapshot()) != 0 {
				t.Error("ACK sent before stream outcome")
			}
			if fails {
				return errors.New("fake failure")
			}
			return nil
		}}
		server, output := commandReplyTestServer(map[string]pb.HelloService_ConnectStreamServer{"online": stream})
		capture = output
		server.processQueueCommand(commandReplyJSON(t, correlatedCommand(exactTemplateCommand("online"), "only-request")))
		replies := capture.snapshot()
		if len(replies) != 1 {
			t.Fatalf("expected one ACK: %+v", replies)
		}
		var envelope struct {
			CorrelationID string          `json:"correlationId"`
			Response      CommandResponse `json:"response"`
		}
		if err := json.Unmarshal([]byte(replies[0].body), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.CorrelationID != "only-request" || envelope.Response.Success == fails {
			t.Fatalf("incorrect ACK: %+v", envelope)
		}
		if fails && len(server.pendingCommands) != 0 {
			t.Fatal("failed send left pending mapping")
		}
	}
}

func TestInvalidReplyRoutesNeverExecuteOrFallBack(t *testing.T) {
	for _, route := range []map[string]interface{}{
		{"responseQueue": "agent_response.rpc.one"}, {"correlationId": "one"},
		{"responseQueue": AGENT_RESPONSE_QUEUE, "correlationId": "one"},
		{"responseQueue": "agent_response.rpc.one", "correlationId": "two"},
		{"responseQueue": "agent_response.rpc.one", "correlationId": "one", "noReply": true},
		{"responseQueue": "agent_response.rpc.bad space", "correlationId": "bad space"},
	} {
		stream := commandReplyStream{send: func(*pb.ServerMessage) error { t.Fatal("invalid routing executed command"); return nil }}
		server, capture := commandReplyTestServer(map[string]pb.HelloService_ConnectStreamServer{"online": stream})
		command := exactTemplateCommand("online")
		for key, value := range route {
			command[key] = value
		}
		server.processQueueCommand(commandReplyJSON(t, command))
		server.processQueueCommand(`{"command":`)
		if len(capture.snapshot()) != 0 {
			t.Fatal("invalid route contaminated shared replies")
		}
	}
}

func TestLegacyListAndSessionResponsesKeepOriginalShape(t *testing.T) {
	server, capture := commandReplyTestServer(map[string]pb.HelloService_ConnectStreamServer{"online": commandReplyStream{}})
	server.processQueueCommand(`{"action":"list_agents"}`)
	server.processQueueCommand(`{"action":"initialize_session","agentId":"online"}`)
	replies := capture.snapshot()
	if len(replies) != 2 {
		t.Fatal("missing legacy replies")
	}
	for _, reply := range replies {
		if reply.private || reply.queue != AGENT_RESPONSE_QUEUE {
			t.Fatalf("legacy route changed: %+v", reply)
		}
	}
	var agents []string
	if err := json.Unmarshal([]byte(replies[0].body), &agents); err != nil || len(agents) != 1 {
		t.Fatalf("legacy list changed: %s", replies[0].body)
	}
	var response CommandResponse
	if err := json.Unmarshal([]byte(replies[1].body), &response); err != nil || !response.Success {
		t.Fatalf("legacy ACK changed: %s", replies[1].body)
	}
}

type publishOnlyFake struct {
	called bool
	err    error
}

func (f *publishOnlyFake) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	if exchange != "" || key != "agent_response.rpc.fixture" || mandatory || immediate || msg.ContentType != "application/json" || string(msg.Body) != "fixture" {
		return errors.New("unexpected publish arguments")
	}
	f.called = true
	return f.err
}
func TestPrivateRepliesPublishWithoutDeclaringOrPurgingQueues(t *testing.T) {
	failure := errors.New("broker unavailable")
	for _, want := range []error{nil, failure} {
		publisher := &publishOnlyFake{err: want}
		if err := publishExistingReply(publisher, "agent_response.rpc.fixture", "fixture"); !errors.Is(err, want) || !publisher.called {
			t.Fatalf("publish called=%v error=%v", publisher.called, err)
		}
	}
}
