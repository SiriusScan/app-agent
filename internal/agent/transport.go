package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/SiriusScan/app-agent/internal/debugtrace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/SiriusScan/app-agent/proto/hello"
)

type grpcTransport struct {
	logger   *zap.Logger
	conn     *grpc.ClientConn
	client   pb.HelloServiceClient
	stream   pb.HelloService_ConnectStreamClient
	streamMu sync.Mutex
}

func newGRPCTransport(logger *zap.Logger) *grpcTransport {
	return &grpcTransport{logger: logger}
}

func (t *grpcTransport) Connect(address string) error {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	t.conn = conn
	t.client = pb.NewHelloServiceClient(conn)
	return nil
}

func (t *grpcTransport) OpenStream(ctx context.Context, values map[string]string) (pb.HelloService_ConnectStreamClient, error) {
	md := metadata.New(values)
	streamCtx := metadata.NewOutgoingContext(ctx, md)

	stream, err := t.client.ConnectStream(streamCtx)
	if err != nil {
		// #region agent log
		debugtrace.Log("pre-fix", "H1,H3,H4", "internal/agent/transport.go:48", "open_stream_failed", map[string]interface{}{
			"agentIdMeta": values["agent_id"],
			"error":       err.Error(),
		})
		// #endregion
		if t.conn != nil {
			_ = t.conn.Close()
		}
		return nil, fmt.Errorf("failed to establish stream: %w", err)
	}

	t.stream = stream
	// #region agent log
	debugtrace.Log("pre-fix", "H1,H3,H4", "internal/agent/transport.go:59", "open_stream_succeeded", map[string]interface{}{
		"agentIdMeta":          values["agent_id"],
		"scriptingEnabledMeta": values["scripting_enabled"],
	})
	// #endregion
	return stream, nil
}

func (t *grpcTransport) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return t.client.Ping(ctx, req)
}

func (t *grpcTransport) Stream() pb.HelloService_ConnectStreamClient {
	return t.stream
}

func (t *grpcTransport) Send(msg *pb.AgentMessage) error {
	t.streamMu.Lock()
	defer t.streamMu.Unlock()
	return t.stream.Send(msg)
}

func (t *grpcTransport) Close() error {
	if t.conn == nil {
		return nil
	}
	return t.conn.Close()
}
