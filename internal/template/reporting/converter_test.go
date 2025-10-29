package reporting

import (
	"testing"
	"time"

	"github.com/SiriusScan/app-agent/internal/template/fingerprint"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

func TestConvertTemplateResultsToVulnerabilities(t *testing.T) {
	tests := []struct {
		name          string
		results       []*types.Result
		expectedCount int
	}{
		{
			name: "single_matched_template",
			results: []*types.Result{
				{
					TemplateID:   "CVE-2024-TEST-001",
					TemplateName: "Test Vulnerability",
					Matched:      true,
					Confidence:   1.0,
					Severity:     types.SeverityCritical,
				},
			},
			expectedCount: 1,
		},
		{
			name: "multiple_matched_templates",
			results: []*types.Result{
				{
					TemplateID:   "CVE-2024-TEST-001",
					TemplateName: "Critical Vulnerability",
					Matched:      true,
					Severity:     types.SeverityCritical,
				},
				{
					TemplateID:   "CVE-2024-TEST-002",
					TemplateName: "High Vulnerability",
					Matched:      true,
					Severity:     types.SeverityHigh,
				},
			},
			expectedCount: 2,
		},
		{
			name: "mixed_matched_and_unmatched",
			results: []*types.Result{
				{
					TemplateID:   "CVE-2024-TEST-001",
					TemplateName: "First Vulnerability",
					Matched:      true,
					Severity:     types.SeverityCritical,
				},
				{
					TemplateID:   "CVE-2024-TEST-002",
					TemplateName: "Second Vulnerability",
					Matched:      false, // Not matched, should be skipped
					Severity:     types.SeverityInfo,
				},
				{
					TemplateID:   "CVE-2024-TEST-003",
					TemplateName: "Third Vulnerability",
					Matched:      true,
					Severity:     types.SeverityMedium,
				},
			},
			expectedCount: 2, // Only matched templates
		},
		{
			name:          "empty_results",
			results:       []*types.Result{},
			expectedCount: 0,
		},
		{
			name: "nil_results",
			results: []*types.Result{
				nil, // Should be skipped gracefully
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vulns := ConvertTemplateResultsToVulnerabilities(tt.results)

			if len(vulns) != tt.expectedCount {
				t.Errorf("got %d vulnerabilities, want %d", len(vulns), tt.expectedCount)
			}

			// Verify all returned vulnerabilities have required fields
			for _, vuln := range vulns {
				if vuln.VID == "" {
					t.Error("Vulnerability VID should not be empty")
				}
				if vuln.Title == "" {
					t.Error("Vulnerability Title should not be empty")
				}
				if vuln.RiskScore < 0 || vuln.RiskScore > 10 {
					t.Errorf("RiskScore %f out of valid range 0.0-10.0", vuln.RiskScore)
				}
			}
		})
	}
}

// Note: severityToRiskScore() function has been moved to internal/template/risk package
// and is tested in risk/calculator_test.go. This test is no longer needed here.

func TestBuildHostData(t *testing.T) {
	fp := &fingerprint.HostFingerprint{
		OS:        "linux",
		OSVersion: "Ubuntu 22.04",
		Hostname:  "test-host",
		PrimaryIP: "192.168.1.100",
		AgentID:   "agent-123",
	}

	vulns := ConvertTemplateResultsToVulnerabilities([]*types.Result{
		{
			TemplateID:   "CVE-2024-TEST-001",
			TemplateName: "Test Vuln",
			Matched:      true,
			Severity:     types.SeverityCritical,
		},
	})

	host := BuildHostData(fp, vulns)

	// Verify fingerprint data is mapped correctly
	if host.HID != fp.AgentID {
		t.Errorf("HID = %q, want %q", host.HID, fp.AgentID)
	}
	if host.OS != fp.OS {
		t.Errorf("OS = %q, want %q", host.OS, fp.OS)
	}
	if host.OSVersion != fp.OSVersion {
		t.Errorf("OSVersion = %q, want %q", host.OSVersion, fp.OSVersion)
	}
	if host.Hostname != fp.Hostname {
		t.Errorf("Hostname = %q, want %q", host.Hostname, fp.Hostname)
	}
	if host.IP != fp.PrimaryIP {
		t.Errorf("IP = %q, want %q", host.IP, fp.PrimaryIP)
	}

	// Verify vulnerabilities are included
	if len(host.Vulnerabilities) != len(vulns) {
		t.Errorf("got %d vulnerabilities, want %d", len(host.Vulnerabilities), len(vulns))
	}

	// Verify empty slices are initialized (not nil)
	if host.Ports == nil {
		t.Error("Ports should be initialized as empty slice, not nil")
	}
	if host.Services == nil {
		t.Error("Services should be initialized as empty slice, not nil")
	}
	if host.CPE == nil {
		t.Error("CPE should be initialized as empty slice, not nil")
	}
	if host.Users == nil {
		t.Error("Users should be initialized as empty slice, not nil")
	}
	if host.Notes == nil {
		t.Error("Notes should be initialized as empty slice, not nil")
	}
}

func TestBuildAgentMetadata(t *testing.T) {
	results := []*types.Result{
		{
			TemplateID: "CVE-2024-TEST-001",
			Matched:    true,
			Confidence: 1.0,
			Severity:   types.SeverityCritical,
		},
		{
			TemplateID: "CVE-2024-TEST-002",
			Matched:    false,
			Confidence: 0.0,
			Severity:   types.SeverityInfo,
		},
		{
			TemplateID: "CVE-2024-TEST-003",
			Matched:    true,
			Confidence: 0.8,
			Severity:   types.SeverityHigh,
		},
	}

	executionTime := 1234 * time.Millisecond

	metadata := BuildAgentMetadata(results, executionTime)

	// Verify required fields
	if metadata["agent_version"] == "" {
		t.Error("agent_version should not be empty")
	}

	scanDuration, ok := metadata["scan_duration"].(int64)
	if !ok {
		t.Fatal("scan_duration should be int64")
	}
	if scanDuration != 1234 {
		t.Errorf("scan_duration = %d, want 1234", scanDuration)
	}

	templateResults, ok := metadata["template_results"].([]map[string]interface{})
	if !ok {
		t.Fatal("template_results should be []map[string]interface{}")
	}
	if len(templateResults) != 3 {
		t.Errorf("got %d template_results, want 3", len(templateResults))
	}

	templateCount, ok := metadata["template_count"].(int)
	if !ok {
		t.Fatal("template_count should be int")
	}
	if templateCount != 3 {
		t.Errorf("template_count = %d, want 3", templateCount)
	}

	matchedCount, ok := metadata["matched_count"].(int)
	if !ok {
		t.Fatal("matched_count should be int")
	}
	if matchedCount != 2 {
		t.Errorf("matched_count = %d, want 2 (only matched templates)", matchedCount)
	}

	// Verify template_results structure
	for _, tr := range templateResults {
		if tr["template_id"] == "" {
			t.Error("template_id should not be empty")
		}
		if tr["vulnerability_id"] == "" {
			t.Error("vulnerability_id should not be empty")
		}
		if _, ok := tr["vulnerable"].(bool); !ok {
			t.Error("vulnerable should be bool")
		}
		if _, ok := tr["confidence"].(float64); !ok {
			t.Error("confidence should be float64")
		}
		if tr["severity"] == "" {
			t.Error("severity should not be empty")
		}
	}
}

func TestBuildAgentMetadataWithNilResults(t *testing.T) {
	results := []*types.Result{
		nil, // Should be handled gracefully
		{
			TemplateID: "CVE-2024-TEST-001",
			Matched:    true,
		},
	}

	metadata := BuildAgentMetadata(results, 100*time.Millisecond)

	templateResults, ok := metadata["template_results"].([]map[string]interface{})
	if !ok {
		t.Fatal("template_results should exist")
	}

	// Should only include non-nil result
	if len(templateResults) != 1 {
		t.Errorf("got %d template_results, want 1 (nil results skipped)", len(templateResults))
	}

	matchedCount := metadata["matched_count"].(int)
	if matchedCount != 1 {
		t.Errorf("matched_count = %d, want 1", matchedCount)
	}
}
