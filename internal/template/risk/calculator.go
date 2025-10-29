package risk

import (
	"github.com/SiriusScan/app-agent/internal/template/types"
)

// CalculateRiskScore determines the final risk score using priority system:
// 1. Custom risk_score (if provided and valid)
// 2. CVSS vector calculation (if provided and parseable)
// 3. CVSS score (if provided and valid)
// 4. Severity mapping (always available as fallback)
//
// Returns the calculated score and the source method used.
func CalculateRiskScore(info types.TemplateInfo) (score float64, source string) {
	// Priority 1: Custom numerical score
	if info.RiskScore != nil && types.ValidateRiskScore(*info.RiskScore) {
		return *info.RiskScore, "custom_score"
	}

	// Priority 2: CVSS vector parsing
	if info.CVSSVector != "" {
		if score, err := ParseCVSSVector(info.CVSSVector); err == nil && types.ValidateRiskScore(score) {
			return score, "cvss_vector"
		}
		// If CVSS vector parsing fails, continue to next priority
	}

	// Priority 3: Pre-calculated CVSS score
	if info.CVSSScore != nil && types.ValidateRiskScore(*info.CVSSScore) {
		return *info.CVSSScore, "cvss_score"
	}

	// Priority 4: Severity mapping (always available)
	return severityToRiskScore(info.Severity), "severity_mapping"
}

// severityToRiskScore maps severity levels to numerical scores
// This provides a consistent fallback when no other risk scoring method is available
func severityToRiskScore(severity types.Severity) float64 {
	switch severity {
	case types.SeverityCritical:
		return 9.5 // Critical range: 9.0-10.0
	case types.SeverityHigh:
		return 7.5 // High range: 7.0-8.9
	case types.SeverityMedium:
		return 5.0 // Medium range: 4.0-6.9
	case types.SeverityLow:
		return 2.0 // Low range: 0.1-3.9
	case types.SeverityInfo:
		return 0.0 // Info: 0.0
	default:
		return 0.0 // Unknown severity treated as info
	}
}









