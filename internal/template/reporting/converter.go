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
			VID:         result.TemplateID,       // Use template ID as vulnerability ID
			Title:       result.TemplateName,     // Template name as title
			Description: result.TemplateName,     // Use name as description (fallback)
			RiskScore:   result.RiskScore,        // Use calculated risk score from result
			Severity:    string(result.Severity), // Include severity string
			CVSSScore:   result.RiskScore,        // Also populate CVSSScore field
			CVSSVector:  result.CVSSVector,       // Include CVSS vector if available
		}

		vulnerabilities = append(vulnerabilities, vuln)
	}

	return vulnerabilities
}

// BuildHostData constructs a sirius.Host from fingerprint and vulnerabilities.
func BuildHostData(fp *fingerprint.HostFingerprint, vulns []sirius.Vulnerability) sirius.Host {
	return sirius.Host{
		HID:             fp.AgentID,         // Use agent ID as host ID
		OS:              fp.OS,              // Operating system
		OSVersion:       fp.OSVersion,       // OS version
		IP:              fp.PrimaryIP,       // Primary IP address
		Hostname:        fp.Hostname,        // System hostname
		Vulnerabilities: vulns,              // Detected vulnerabilities
		Ports:           []sirius.Port{},    // No port data from template scans
		Services:        []sirius.Service{}, // No service data from template scans
		CPE:             []string{},         // No CPE data from template scans
		Users:           []string{},         // No user data from template scans
		Notes:           []string{},         // No notes
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

		tr := map[string]interface{}{
			"template_id":      result.TemplateID,
			"vulnerability_id": result.TemplateID, // Same as VID in vulnerabilities
			"vulnerable":       result.Matched,
			"confidence":       result.Confidence,
			"severity":         string(result.Severity),
			"risk_score":       result.RiskScore, // Include calculated risk score
		}

		// Include CVSS vector if available
		if result.CVSSVector != "" {
			tr["cvss_vector"] = result.CVSSVector
		}

		templateResults = append(templateResults, tr)
	}

	return map[string]interface{}{
		"agent_version":    "1.0.0-template-mvp",
		"scan_duration":    int64(executionTime.Milliseconds()),
		"template_results": templateResults,
		"template_count":   len(results),
		"matched_count":    matchedCount,
	}
}

// Note: severityToRiskScore function removed - risk scoring now handled by
// internal/template/risk package with priority system:
// 1. Custom risk_score
// 2. CVSS vector calculation
// 3. CVSS score
// 4. Severity mapping (fallback)
