package risk

import (
	"testing"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

func TestCalculateRiskScore(t *testing.T) {
	tests := []struct {
		name           string
		info           types.TemplateInfo
		expectedScore  float64
		expectedSource string
	}{
		{
			name: "Priority 1: Custom risk score",
			info: types.TemplateInfo{
				Severity:   types.SeverityHigh,
				RiskScore:  ptr(8.5),
				CVSSVector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
				CVSSScore:  ptr(9.0),
			},
			expectedScore:  8.5,
			expectedSource: "custom_score",
		},
		{
			name: "Priority 2: CVSS vector",
			info: types.TemplateInfo{
				Severity:   types.SeverityHigh,
				CVSSVector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
				CVSSScore:  ptr(9.0),
			},
			expectedScore:  10.0,
			expectedSource: "cvss_vector",
		},
		{
			name: "Priority 3: CVSS score",
			info: types.TemplateInfo{
				Severity:  types.SeverityHigh,
				CVSSScore: ptr(7.2),
			},
			expectedScore:  7.2,
			expectedSource: "cvss_score",
		},
		{
			name: "Priority 4: Severity mapping (critical)",
			info: types.TemplateInfo{
				Severity: types.SeverityCritical,
			},
			expectedScore:  9.5,
			expectedSource: "severity_mapping",
		},
		{
			name: "Priority 4: Severity mapping (high)",
			info: types.TemplateInfo{
				Severity: types.SeverityHigh,
			},
			expectedScore:  7.5,
			expectedSource: "severity_mapping",
		},
		{
			name: "Priority 4: Severity mapping (medium)",
			info: types.TemplateInfo{
				Severity: types.SeverityMedium,
			},
			expectedScore:  5.0,
			expectedSource: "severity_mapping",
		},
		{
			name: "Priority 4: Severity mapping (low)",
			info: types.TemplateInfo{
				Severity: types.SeverityLow,
			},
			expectedScore:  2.0,
			expectedSource: "severity_mapping",
		},
		{
			name: "Priority 4: Severity mapping (info)",
			info: types.TemplateInfo{
				Severity: types.SeverityInfo,
			},
			expectedScore:  0.0,
			expectedSource: "severity_mapping",
		},
		{
			name: "Invalid custom score falls back to CVSS vector",
			info: types.TemplateInfo{
				Severity:   types.SeverityHigh,
				RiskScore:  ptr(15.0), // Invalid (> 10.0)
				CVSSVector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
			},
			expectedScore:  10.0,
			expectedSource: "cvss_vector",
		},
		{
			name: "Invalid CVSS vector falls back to CVSS score",
			info: types.TemplateInfo{
				Severity:   types.SeverityHigh,
				CVSSVector: "INVALID",
				CVSSScore:  ptr(8.1),
			},
			expectedScore:  8.1,
			expectedSource: "cvss_score",
		},
		{
			name: "All invalid falls back to severity",
			info: types.TemplateInfo{
				Severity:   types.SeverityMedium,
				RiskScore:  ptr(-1.0), // Invalid
				CVSSVector: "INVALID",
				CVSSScore:  ptr(20.0), // Invalid
			},
			expectedScore:  5.0,
			expectedSource: "severity_mapping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, source := CalculateRiskScore(tt.info)

			if score != tt.expectedScore {
				t.Errorf("CalculateRiskScore() score = %v, want %v", score, tt.expectedScore)
			}

			if source != tt.expectedSource {
				t.Errorf("CalculateRiskScore() source = %v, want %v", source, tt.expectedSource)
			}
		})
	}
}

func TestSeverityToRiskScore(t *testing.T) {
	tests := []struct {
		name          string
		severity      types.Severity
		expectedScore float64
	}{
		{"Critical", types.SeverityCritical, 9.5},
		{"High", types.SeverityHigh, 7.5},
		{"Medium", types.SeverityMedium, 5.0},
		{"Low", types.SeverityLow, 2.0},
		{"Info", types.SeverityInfo, 0.0},
		{"Unknown", types.Severity("unknown"), 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := severityToRiskScore(tt.severity)

			if score != tt.expectedScore {
				t.Errorf("severityToRiskScore(%v) = %v, want %v", tt.severity, score, tt.expectedScore)
			}
		})
	}
}

// ptr is a helper function to create pointers to float64 values
func ptr(f float64) *float64 {
	return &f
}









