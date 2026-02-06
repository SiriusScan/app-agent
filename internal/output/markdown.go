package output

import (
	"fmt"
	"strings"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

func init() {
	Register(FormatMarkdown, &MarkdownFormatter{})
}

// MarkdownFormatter outputs data as Markdown tables, suitable for documentation and reports.
type MarkdownFormatter struct {
	BaseFormatter
}

// Name returns the format name.
func (f *MarkdownFormatter) Name() Format {
	return FormatMarkdown
}

// FormatScanResults formats template scan results as Markdown.
func (f *MarkdownFormatter) FormatScanResults(results []*types.Result, summary *ScanSummary) (string, error) {
	var builder strings.Builder

	// Title
	builder.WriteString("# Scan Results\n\n")

	// Summary
	builder.WriteString("## Summary\n\n")
	builder.WriteString(fmt.Sprintf("- **Total Templates:** %d\n", summary.TotalTemplates))
	builder.WriteString(fmt.Sprintf("- **Vulnerabilities Found:** %d\n", summary.Matched))
	builder.WriteString(fmt.Sprintf("- **Safe:** %d\n", summary.NotMatched))
	if summary.Errors > 0 {
		builder.WriteString(fmt.Sprintf("- **Errors:** %d\n", summary.Errors))
	}
	builder.WriteString(fmt.Sprintf("- **Execution Time:** %dms\n", summary.ExecutionTimeMs))
	builder.WriteString(fmt.Sprintf("- **Workers:** %d\n", summary.Workers))
	builder.WriteString("\n")

	// Status badge
	if summary.Matched > 0 {
		builder.WriteString("**Status:** ⚠️ VULNERABILITIES DETECTED\n\n")
	} else {
		builder.WriteString("**Status:** ✅ NO VULNERABILITIES DETECTED\n\n")
	}

	// Results table
	builder.WriteString("## Results\n\n")
	builder.WriteString("| # | Template | Severity | Status | Risk | Confidence |\n")
	builder.WriteString("|---|----------|----------|--------|------|------------|\n")

	for i, result := range results {
		if result == nil {
			builder.WriteString(fmt.Sprintf("| %d | Unknown | - | ❓ Error | - | - |\n", i+1))
			continue
		}

		status := "✅ Safe"
		if result.Matched {
			status = "⚠️ Vulnerable"
		}

		severity := strings.ToUpper(string(result.Severity))
		name := result.TemplateName
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		builder.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %.1f | %.0f%% |\n",
			i+1, name, severity, status, result.RiskScore, result.Confidence*100))
	}
	builder.WriteString("\n")

	// Detailed findings for vulnerabilities
	matchedResults := make([]*types.Result, 0)
	for _, r := range results {
		if r != nil && r.Matched {
			matchedResults = append(matchedResults, r)
		}
	}

	if len(matchedResults) > 0 {
		builder.WriteString("## Vulnerability Details\n\n")

		for i, result := range matchedResults {
			builder.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, result.TemplateName))
			builder.WriteString(fmt.Sprintf("- **ID:** `%s`\n", result.TemplateID))
			builder.WriteString(fmt.Sprintf("- **Severity:** %s\n", strings.ToUpper(string(result.Severity))))
			builder.WriteString(fmt.Sprintf("- **Risk Score:** %.1f\n", result.RiskScore))

			if result.Description != "" {
				builder.WriteString(fmt.Sprintf("- **Description:** %s\n", result.Description))
			}

			if len(result.CVE) > 0 {
				builder.WriteString(fmt.Sprintf("- **CVE:** %s\n", strings.Join(result.CVE, ", ")))
			}

			if result.Remediation != "" {
				builder.WriteString(fmt.Sprintf("\n**Remediation:** %s\n", result.Remediation))
			}

			if len(result.References) > 0 {
				builder.WriteString("\n**References:**\n")
				for _, ref := range result.References {
					builder.WriteString(fmt.Sprintf("- %s\n", ref))
				}
			}

			builder.WriteString("\n")
		}
	}

	return builder.String(), nil
}

// FormatTemplateList formats a list of templates as Markdown.
func (f *MarkdownFormatter) FormatTemplateList(templates []*types.Template) (string, error) {
	var builder strings.Builder

	// Title
	builder.WriteString("# Available Templates\n\n")
	builder.WriteString(fmt.Sprintf("**Total:** %d templates\n\n", len(templates)))

	// Table
	builder.WriteString("| # | ID | Name | Severity | Steps | Author |\n")
	builder.WriteString("|---|-----|------|----------|-------|--------|\n")

	for i, t := range templates {
		name := t.Info.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		id := t.ID
		if len(id) > 20 {
			id = id[:17] + "..."
		}

		author := t.Info.Author
		if author == "" {
			author = "-"
		} else if len(author) > 15 {
			author = author[:12] + "..."
		}

		severity := strings.ToUpper(string(t.Info.Severity))

		builder.WriteString(fmt.Sprintf("| %d | `%s` | %s | %s | %d | %s |\n",
			i+1, id, name, severity, len(t.Detection.Steps), author))
	}
	builder.WriteString("\n")

	return builder.String(), nil
}

// FormatValidation formats a template validation result as Markdown.
func (f *MarkdownFormatter) FormatValidation(result *ValidationResult) (string, error) {
	var builder strings.Builder

	// Title with status
	if result.Valid {
		builder.WriteString("# ✅ Template Valid\n\n")
	} else {
		builder.WriteString("# ❌ Template Invalid\n\n")
	}

	// Details
	builder.WriteString("## Details\n\n")
	builder.WriteString(fmt.Sprintf("- **ID:** `%s`\n", result.TemplateID))
	builder.WriteString(fmt.Sprintf("- **Name:** %s\n", result.TemplateName))
	builder.WriteString(fmt.Sprintf("- **Severity:** %s\n", strings.ToUpper(result.Severity)))
	builder.WriteString(fmt.Sprintf("- **Steps:** %d\n", result.StepCount))
	builder.WriteString("\n")

	// Errors
	if len(result.Errors) > 0 {
		builder.WriteString("## Errors\n\n")
		for _, err := range result.Errors {
			builder.WriteString(fmt.Sprintf("- ❌ %s\n", err))
		}
		builder.WriteString("\n")
	}

	// Warnings
	if len(result.Warnings) > 0 {
		builder.WriteString("## Warnings\n\n")
		for _, warn := range result.Warnings {
			builder.WriteString(fmt.Sprintf("- ⚠️ %s\n", warn))
		}
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// FormatSystemScan formats a system inventory scan result as Markdown.
func (f *MarkdownFormatter) FormatSystemScan(result *SystemScanResult) (string, error) {
	var builder strings.Builder

	// Title
	builder.WriteString("# System Scan Report\n\n")

	// System Information
	builder.WriteString("## System Information\n\n")
	builder.WriteString(fmt.Sprintf("- **OS:** %s\n", result.OSInfo.OS))
	builder.WriteString(fmt.Sprintf("- **Version:** %s\n", result.OSInfo.Version))
	builder.WriteString(fmt.Sprintf("- **Hostname:** %s\n", result.OSInfo.Hostname))
	builder.WriteString(fmt.Sprintf("- **Primary IP:** %s\n", result.OSInfo.PrimaryIP))
	builder.WriteString("\n")

	// Packages
	builder.WriteString("## Installed Packages\n\n")
	builder.WriteString(fmt.Sprintf("**Total:** %d packages\n\n", len(result.Packages)))

	if len(result.Packages) > 0 && len(result.Packages) <= 50 {
		builder.WriteString("| Name | Version | Source |\n")
		builder.WriteString("|------|---------|--------|\n")

		for _, pkg := range result.Packages {
			source := pkg.Source
			if source == "" {
				source = "-"
			}
			builder.WriteString(fmt.Sprintf("| %s | %s | %s |\n", pkg.Name, pkg.Version, source))
		}
		builder.WriteString("\n")
	} else if len(result.Packages) > 50 {
		builder.WriteString("*(Too many packages to display. Use `--format json` for complete list.)*\n\n")
	}

	// Custom Results
	if len(result.CustomResults) > 0 {
		builder.WriteString("## Custom Script Results\n\n")

		for name, res := range result.CustomResults {
			status := "✅ Success"
			if res.ExitCode != 0 {
				status = fmt.Sprintf("❌ Failed (exit: %d)", res.ExitCode)
			}
			builder.WriteString(fmt.Sprintf("- **%s:** %s\n", name, status))
		}
		builder.WriteString("\n")
	}

	// Errors
	if len(result.ScanErrors) > 0 {
		builder.WriteString("## Scan Errors\n\n")
		for _, err := range result.ScanErrors {
			builder.WriteString(fmt.Sprintf("- ⚠️ %s\n", err))
		}
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// FormatError formats an error message in Markdown.
func (f *MarkdownFormatter) FormatError(err error) string {
	return fmt.Sprintf("**Error:** %v\n", err)
}



