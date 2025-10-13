package reporting

import (
	"time"

	"github.com/SiriusScan/app-agent/internal/template/fingerprint"
	"github.com/SiriusScan/app-agent/internal/template/types"
	"github.com/SiriusScan/go-api/sirius"
)

// ConvertTemplateResultsToVulnerabilities converts template scan results to sirius.Vulnerability format.
// Only matched templates are converted to vulnerabilities.
func ConvertTemplateResultsToVulnerabilities(results []*types.Result) []sirius.Vulnerability {
	vulnerabilities := make([]sirius.Vulnerability, 0)

	for _, result := range results {
		if result == nil || !result.Matched {
			continue // Skip unmatched templates
		}

		vuln := sirius.Vulnerability{
			VID:         result.TemplateID,                     // Use template ID as vulnerability ID
			Title:       result.TemplateName,                   // Template name as title
			Description: result.TemplateName,                   // Use name as description (fallback)
			RiskScore:   severityToRiskScore(result.Severity),  // Map severity to risk score
			Severity:    string(result.Severity),               // Include severity string
		}

		vulnerabilities = append(vulnerabilities, vuln)
	}

	return vulnerabilities
}

// BuildHostData constructs a sirius.Host from fingerprint and vulnerabilities.
func BuildHostData(fp *fingerprint.HostFingerprint, vulns []sirius.Vulnerability) sirius.Host {
	return sirius.Host{
		HID:             fp.AgentID,   // Use agent ID as host ID
		OS:              fp.OS,         // Operating system
		OSVersion:       fp.OSVersion,  // OS version
		IP:              fp.PrimaryIP,  // Primary IP address
		Hostname:        fp.Hostname,   // System hostname
		Vulnerabilities: vulns,         // Detected vulnerabilities
		Ports:           []sirius.Port{}, // No port data from template scans
		Services:        []sirius.Service{}, // No service data from template scans
		CPE:             []string{},    // No CPE data from template scans
		Users:           []string{},    // No user data from template scans
		Notes:           []string{},    // No notes
	}
}

// BuildAgentMetadata constructs the agent_metadata JSONB field content.
// This includes template-specific information for enhanced reporting.
func BuildAgentMetadata(results []*types.Result, executionTime time.Duration) map[string]interface{} {
	templateResults := make([]map[string]interface{}, 0)
	matchedCount := 0

	for _, result := range results {
		if result == nil {
			continue
		}

		if result.Matched {
			matchedCount++
		}

		templateResults = append(templateResults, map[string]interface{}{
			"template_id":      result.TemplateID,
			"vulnerability_id": result.TemplateID, // Same as VID in vulnerabilities
			"vulnerable":       result.Matched,
			"confidence":       result.Confidence,
			"severity":         string(result.Severity),
		})
	}

	return map[string]interface{}{
		"agent_version":   "1.0.0-template-mvp",
		"scan_duration":   int64(executionTime.Milliseconds()),
		"template_results": templateResults,
		"template_count":  len(results),
		"matched_count":   matchedCount,
	}
}

// severityToRiskScore converts template severity to a numeric risk score (0.0-10.0).
// This mapping aligns with CVSS-style scoring where:
//   - Critical: 9.0-10.0
//   - High:     7.0-8.9
//   - Medium:   4.0-6.9
//   - Low:      0.1-3.9
//   - Info:     0.0
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

