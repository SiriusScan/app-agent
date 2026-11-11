package agent

import (
	"context"
	"testing"

	"github.com/SiriusScan/app-agent/internal/config"
	pb "github.com/SiriusScan/app-agent/proto/hello"
	"go.uber.org/zap"
)

type enrollmentStream struct {
	pb.HelloService_ConnectStreamClient
	sent []*pb.AgentMessage
}

func (s *enrollmentStream) Send(msg *pb.AgentMessage) error { s.sent = append(s.sent, msg); return nil }

func TestEnrollmentCredentialOnlySentWithoutToken(t *testing.T) {
	cfg := &config.AgentConfig{AgentID: "agent", EnrollmentCredential: "synthetic-credential"}
	stream := &enrollmentStream{}
	agent := &Agent{config: cfg, logger: zap.NewNop(), transport: &grpcTransport{stream: stream}}
	if err := agent.sendAuthenticatedHeartbeat(context.Background()); err != nil {
		t.Fatal(err)
	}
	if stream.sent[0].GetEnrollmentCredential() != "synthetic-credential" || stream.sent[0].GetAuthToken() != "" {
		t.Fatal("missing enrollment credential")
	}
	agent.authToken = "synthetic-token"
	if err := agent.sendAuthenticatedHeartbeat(context.Background()); err != nil {
		t.Fatal(err)
	}
	if stream.sent[1].GetEnrollmentCredential() != "" || stream.sent[1].GetAuthToken() != "synthetic-token" {
		t.Fatal("credential sent on reconnect")
	}
}
