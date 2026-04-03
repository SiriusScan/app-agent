package server

import (
	"testing"

	"go.uber.org/zap"
	goapistore "github.com/SiriusScan/go-api/sirius/store"
)

func TestIsConcreteHostIP(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"192.168.1.1", true},
		{"::1", true},
		{"10.0.0.0/24", false},
		{"192.168.1.1-10", false},
		{"", false},
		{"example.com", false},
	}
	for _, tc := range tests {
		if got := isConcreteHostIP(tc.in); got != tc.want {
			t.Errorf("isConcreteHostIP(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestPrimaryIPMatchesTarget_CIDR(t *testing.T) {
	if !primaryIPMatchesTarget("10.0.0.5", "10.0.0.0/24") {
		t.Fatal("expected primary to match CIDR")
	}
	if primaryIPMatchesTarget("10.0.1.1", "10.0.0.0/24") {
		t.Fatal("expected out-of-range IP not to match CIDR")
	}
}

func TestResolveAgentHostIP_CIDRTargetMatchesPrimary(t *testing.T) {
	s := &Server{logger: zap.NewNop()}
	sr := &goapistore.ScanResult{
		Targets: []string{"10.0.0.0/24"},
		Hosts:   []goapistore.HostEntry{},
	}
	ip := s.resolveAgentHostIP("agent-1", "", "10.0.0.5", sr)
	if ip != "10.0.0.5" {
		t.Fatalf("got %q want 10.0.0.5", ip)
	}
}

func TestResolveAgentHostIP_DoesNotReturnCIDRString(t *testing.T) {
	s := &Server{logger: zap.NewNop()}
	sr := &goapistore.ScanResult{
		Targets: []string{"10.0.0.0/24"},
		Hosts:   []goapistore.HostEntry{},
	}
	ip := s.resolveAgentHostIP("not-a-hostname", "", "", sr)
	if ip != "" {
		t.Fatalf("expected empty host key when nothing can be resolved, got %q (must not be CIDR)", ip)
	}
	if ip == "10.0.0.0/24" {
		t.Fatal("must never return raw CIDR as host IP")
	}
}

func TestResolveAgentHostIP_MergesOntoDiscoveredHost(t *testing.T) {
	s := &Server{logger: zap.NewNop()}
	sr := &goapistore.ScanResult{
		Targets: []string{"10.0.0.0/24"},
		Hosts: []goapistore.HostEntry{
			{ID: "10.0.0.7", IP: "10.0.0.7", Hostname: "h1", Sources: []string{"network"}},
		},
	}
	ip := s.resolveAgentHostIP("agent-1", "", "", sr)
	if ip != "10.0.0.7" {
		t.Fatalf("got %q want first unclaimed discovered host 10.0.0.7", ip)
	}
}

func TestAppendAgentVulnerabilitiesDeduped(t *testing.T) {
	existing := []goapistore.VulnerabilitySummary{
		{ID: "v1", HostID: "10.0.0.1", ScanSource: "agent", AgentID: "a1"},
	}
	newOnes := []goapistore.VulnerabilitySummary{
		{ID: "v1", ScanSource: "agent", AgentID: "a1"},
		{ID: "v2", ScanSource: "agent", AgentID: "a1"},
	}
	out, added := appendAgentVulnerabilitiesDeduped(existing, newOnes, "10.0.0.1", "a1")
	if added != 1 {
		t.Fatalf("added %d want 1 (v1 duplicate skipped)", added)
	}
	if len(out) != 2 {
		t.Fatalf("len %d want 2", len(out))
	}
}
