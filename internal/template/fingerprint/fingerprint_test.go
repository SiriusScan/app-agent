package fingerprint

import (
	"context"
	"runtime"
	"testing"

	"github.com/SiriusScan/app-agent/internal/config"
)

func TestCollectBasicFingerprint(t *testing.T) {
	cfg := &config.AgentConfig{
		AgentID: "test-agent-123",
	}

	fingerprint, err := CollectBasicFingerprint(context.Background(), cfg)

	// Should never return an error (partial data is acceptable)
	if err != nil {
		t.Errorf("CollectBasicFingerprint returned unexpected error: %v", err)
	}

	// Verify basic fields are populated
	if fingerprint == nil {
		t.Fatal("Fingerprint should not be nil")
	}

	// OS should match runtime
	if fingerprint.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", fingerprint.OS, runtime.GOOS)
	}

	// OSVersion should be populated (even if empty string on some systems)
	t.Logf("OS Version: %q", fingerprint.OSVersion)

	// Hostname should be populated (either real hostname or agent ID fallback)
	if fingerprint.Hostname == "" {
		t.Error("Hostname should not be empty")
	}
	t.Logf("Hostname: %q", fingerprint.Hostname)

	// PrimaryIP may be empty on some systems (no non-loopback interface)
	t.Logf("Primary IP: %q", fingerprint.PrimaryIP)

	// AgentID should match config
	if fingerprint.AgentID != cfg.AgentID {
		t.Errorf("AgentID = %q, want %q", fingerprint.AgentID, cfg.AgentID)
	}
}

func TestCollectBasicFingerprintFields(t *testing.T) {
	tests := []struct {
		name    string
		agentID string
	}{
		{"with_agent_id", "agent-001"},
		{"empty_agent_id", ""},
		{"long_agent_id", "very-long-agent-identifier-12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.AgentConfig{
				AgentID: tt.agentID,
			}

			fingerprint, err := CollectBasicFingerprint(context.Background(), cfg)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if fingerprint.AgentID != tt.agentID {
				t.Errorf("AgentID = %q, want %q", fingerprint.AgentID, tt.agentID)
			}

			// Hostname should always be set (either real or fallback to AgentID)
			if fingerprint.Hostname == "" && tt.agentID != "" {
				t.Error("Hostname should not be empty when AgentID is set")
			}
		})
	}
}

func TestCollectBasicFingerprintCancellation(t *testing.T) {
	cfg := &config.AgentConfig{
		AgentID: "test-agent",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should still complete (fingerprinting is fast and doesn't respect context)
	fingerprint, err := CollectBasicFingerprint(ctx, cfg)

	if err != nil {
		t.Errorf("Should not error on cancelled context: %v", err)
	}

	if fingerprint == nil {
		t.Error("Should still return fingerprint even with cancelled context")
	}
}

