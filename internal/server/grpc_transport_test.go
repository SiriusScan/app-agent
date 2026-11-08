package server

import (
	"strings"
	"testing"

	"github.com/SiriusScan/app-agent/internal/config"
	"go.uber.org/zap"
)

func TestServerRejectsMissingTLSBeforeStartingServices(t *testing.T) {
	cfg := &config.ServerConfig{Address: "127.0.0.1:0"}
	if _, err := NewServer(cfg, zap.NewNop()); err == nil || !strings.Contains(err.Error(), "TLS is required") {
		t.Fatalf("expected TLS configuration failure before initialization, got %v", err)
	}
	// A server constructed outside NewServer must also fail before starting
	// queue workers, accepting traffic, or dereferencing service dependencies.
	s := &Server{config: cfg, logger: zap.NewNop()}
	if err := s.Start(); err == nil || !strings.Contains(err.Error(), "TLS is required") {
		t.Fatalf("expected TLS configuration failure before Start, got %v", err)
	}
	if s.server != nil {
		t.Fatal("gRPC server initialized despite missing TLS credentials")
	}
}
