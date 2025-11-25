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

		// Use VulnerabilityID from result (already resolved by executor)
		// Falls back to TemplateID if VulnerabilityID is empty
		vid := result.VulnerabilityID
		if vid == "" {
			vid = result.TemplateID
		}

		// Use template description if available, otherwise fall back to name
		description := result.Description
		if description == "" {
			description = result.TemplateName
		}

		// Determine CVE ID (first CVE if any)
		cveID := ""
		if len(result.CVE) > 0 {
			cveID = result.CVE[0]
		}

		// Determine category from CWE or tags
		category := ""
		if len(result.CWE) > 0 {
			category = result.CWE[0]
		} else if len(result.Tags) > 0 {
			category = result.Tags[0]
		}

		// Build metadata with template tracking info
		metadata := map[string]interface{}{
			"template_id": result.TemplateID,
			"confidence":  result.Confidence,
		}
		if len(result.CVE) > 1 {
			metadata["additional_cves"] = result.CVE[1:]
		}
		if len(result.CWE) > 0 {
			metadata["cwe"] = result.CWE
		}
		if len(result.References) > 0 {
			metadata["references"] = result.References
		}

		vuln := sirius.Vulnerability{
			VID:         vid,                     // Use resolved vulnerability ID
			Title:       result.TemplateName,     // Template name as title
			Description: description,             // Use template description
			RiskScore:   result.RiskScore,        // Use calculated risk score from result
			CVEID:       cveID,                   // Primary CVE ID if available
			CVSSScore:   result.RiskScore,        // Also populate CVSSScore field
			CVSSVector:  result.CVSSVector,       // Include CVSS vector if available
			Severity:    string(result.Severity), // Include severity string
			Remediation: result.Remediation,      // Include remediation guidance
			Category:    category,                // Category from CWE or tags
			Tags:        result.Tags,             // Include all tags
			Confidence:  result.Confidence,       // Detection confidence
			Metadata:    metadata,                // Additional metadata
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

		// Use VulnerabilityID from result, fallback to TemplateID
		vid := result.VulnerabilityID
		if vid == "" {
			vid = result.TemplateID
		}

		tr := map[string]interface{}{
			"template_id":      result.TemplateID, // Detection rule identifier
			"vulnerability_id": vid,               // VID used in Sirius (CVE/CWE/SIRIUS-XXX)
			"vulnerable":       result.Matched,
			"confidence":       result.Confidence,
			"severity":         string(result.Severity),
			"risk_score":       result.RiskScore,
		}

		// Include CVSS vector if available
		if result.CVSSVector != "" {
			tr["cvss_vector"] = result.CVSSVector
		}

		// Include CVE/CWE references if available
		if len(result.CVE) > 0 {
			tr["cve"] = result.CVE
		}
		if len(result.CWE) > 0 {
			tr["cwe"] = result.CWE
		}

		templateResults = append(templateResults, tr)
	}

	return map[string]interface{}{
		"agent_version":    "1.0.0",
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
