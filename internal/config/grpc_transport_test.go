package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGRPCSecureDefaults(t *testing.T) {
	creds, err := (&AgentConfig{}).GRPCTransportCredentials()
	if err != nil || creds.Info().SecurityProtocol != "tls" {
		t.Fatalf("agent must default to TLS: %v", err)
	}
	if _, err := (&ServerConfig{}).GRPCTransportCredentials(); err == nil {
		t.Fatal("server without certificate/key must fail, not accept plaintext")
	}
}

func TestGRPCTransportRejectsInvalidConfiguration(t *testing.T) {
	invalidPEM := filepath.Join(t.TempDir(), "invalid.pem")
	if err := os.WriteFile(invalidPEM, []byte("not a certificate"), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "missing.pem")
	servers := map[string]*ServerConfig{
		"nil":                       nil,
		"unknown mode":              {GRPCTransport: "false"},
		"typo cannot disable TLS":   {GRPCTransport: "TLS "},
		"certificate only":          {TLSCertFile: invalidPEM},
		"key only":                  {TLSKeyFile: invalidPEM},
		"missing files":             {TLSCertFile: missing, TLSKeyFile: missing},
		"invalid files":             {TLSCertFile: invalidPEM, TLSKeyFile: invalidPEM},
		"insecure with certificate": {GRPCTransport: GRPCTransportInsecureLoopback, Address: "127.0.0.1:0", TLSCertFile: invalidPEM},
		"insecure with key":         {GRPCTransport: GRPCTransportInsecureLoopback, Address: "127.0.0.1:0", TLSKeyFile: invalidPEM},
	}
	for name, cfg := range servers {
		t.Run("server/"+name, func(t *testing.T) {
			if _, err := cfg.GRPCTransportCredentials(); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
	agents := map[string]*AgentConfig{
		"nil":                nil,
		"unknown mode":       {GRPCTransport: "plaintext"},
		"missing CA":         {TLSCAFile: missing},
		"invalid CA":         {TLSCAFile: invalidPEM},
		"insecure with CA":   {GRPCTransport: GRPCTransportInsecureLoopback, ServerAddress: "127.0.0.1:0", TLSCAFile: invalidPEM},
		"insecure with name": {GRPCTransport: GRPCTransportInsecureLoopback, ServerAddress: "127.0.0.1:0", TLSServerName: "engine.test"},
	}
	for name, cfg := range agents {
		t.Run("agent/"+name, func(t *testing.T) {
			if _, err := cfg.GRPCTransportCredentials(); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestGRPCPlaintextIsExplicitAndLoopbackOnly(t *testing.T) {
	cases := []struct {
		address string
		allowed bool
	}{
		{"127.0.0.1:50051", true}, {"127.0.0.2:0", true}, {"[::1]:50051", true},
		{"localhost:50051", false}, {"engine.test:50051", false},
		{":50051", false}, {"0.0.0.0:50051", false}, {"[::]:50051", false},
		{"192.168.1.1:50051", false}, {"8.8.8.8:50051", false},
		{"dns:///127.0.0.1:50051", false}, {"passthrough:///127.0.0.1:50051", false},
		{"127.0.0.1", false}, {"127.0.0.1:", false}, {"127.0.0.1:https", false},
		{"127.0.0.1:65536", false}, {"127.0.0.1:-1", false},
	}
	for _, tc := range cases {
		t.Run(tc.address, func(t *testing.T) {
			s := &ServerConfig{Address: tc.address, GRPCTransport: GRPCTransportInsecureLoopback}
			serverCreds, err := s.GRPCTransportCredentials()
			if (err == nil) != tc.allowed {
				t.Fatalf("server accepted=%v, expected=%v: %v", err == nil, tc.allowed, err)
			}
			if tc.allowed && serverCreds.Info().SecurityProtocol != "insecure" {
				t.Fatal("explicit local mode did not produce plaintext credentials")
			}
			a := &AgentConfig{ServerAddress: tc.address, GRPCTransport: GRPCTransportInsecureLoopback}
			agentCreds, err := a.GRPCTransportCredentials()
			if (err == nil) != tc.allowed {
				t.Fatalf("agent accepted=%v, expected=%v: %v", err == nil, tc.allowed, err)
			}
			if tc.allowed && agentCreds.Info().SecurityProtocol != "insecure" {
				t.Fatal("explicit local mode did not produce plaintext credentials")
			}
		})
	}
}

func TestLoadGRPCEnvironment(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "engine.test:50051")
	t.Setenv("GRPC_TRANSPORT", "tls")
	t.Setenv("GRPC_TLS_CERT_FILE", "/test/server.pem")
	t.Setenv("GRPC_TLS_KEY_FILE", "/test/server.key")
	t.Setenv("GRPC_TLS_CA_FILE", "/test/ca.pem")
	t.Setenv("GRPC_TLS_SERVER_NAME", "expected.test")
	// Avoid reading any real persisted token during this configuration test.
	t.Setenv("AGENT_TOKEN_FILE", filepath.Join(t.TempDir(), "no-token"))
	s := LoadServerConfig()
	if s.Address != "engine.test:50051" || s.GRPCTransport != "tls" || s.TLSCertFile != "/test/server.pem" || s.TLSKeyFile != "/test/server.key" {
		t.Fatal("server TLS environment not loaded")
	}
	a := LoadAgentConfig()
	if a.ServerAddress != "engine.test:50051" || a.GRPCTransport != "tls" || a.TLSCAFile != "/test/ca.pem" || a.TLSServerName != "expected.test" {
		t.Fatal("agent TLS environment not loaded")
	}
}
