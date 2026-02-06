package output

import (
	"fmt"
	"strings"

	"github.com/SiriusScan/app-agent/internal/common/color"
	"github.com/SiriusScan/app-agent/internal/template/types"
	"github.com/olekukonko/tablewriter"
)

func init() {
	Register(FormatTable, &TableFormatter{BaseFormatter: *NewBaseFormatter()})
}

// TableFormatter outputs data as formatted ASCII tables.
type TableFormatter struct {
	BaseFormatter
}

// Name returns the format name.
func (f *TableFormatter) Name() Format {
	return FormatTable
}

// FormatScanResults formats template scan results as a table.
func (f *TableFormatter) FormatScanResults(results []*types.Result, summary *ScanSummary) (string, error) {
	var builder strings.Builder

	// Header
	builder.WriteString("\n")
	builder.WriteString(f.colorize("🔍 SCAN RESULTS", color.Bold+color.BrightCyan))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("─", 70))
	builder.WriteString("\n\n")

	// Results table
	table := tablewriter.NewWriter(&builder)
	table.SetHeader([]string{"#", "Template", "Severity", "Status", "Risk", "Confidence"})
	table.SetBorder(true)
	table.SetRowLine(false)
	table.SetHeaderLine(true)
	table.SetCenterSeparator("│")
	table.SetColumnSeparator("│")
	table.SetRowSeparator("─")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)

	// Set column widths
	table.SetColWidth(30)

	for i, result := range results {
		if result == nil {
			table.Append([]string{
				fmt.Sprintf("%d", i+1),
				"Unknown",
				"-",
				f.colorize("ERROR", color.Red),
				"-",
				"-",
			})
			continue
		}

		status := f.colorize("SAFE", color.Green)
		if result.Matched {
			status = f.colorize("VULNERABLE", color.BrightRed+color.Bold)
		}

		severity := f.colorizedSeverity(string(result.Severity))
		riskScore := fmt.Sprintf("%.1f", result.RiskScore)
		confidence := fmt.Sprintf("%.0f%%", result.Confidence*100)

		// Truncate long template names
		name := result.TemplateName
		if len(name) > 28 {
			name = name[:25] + "..."
		}

		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			name,
			severity,
			status,
			riskScore,
			confidence,
		})
	}

	table.Render()

	// Summary section
	builder.WriteString("\n")
	builder.WriteString(f.colorize("📊 SUMMARY", color.Bold+color.BrightCyan))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("─", 40))
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("  Total Templates:  %d\n", summary.TotalTemplates))
	builder.WriteString(fmt.Sprintf("  Vulnerabilities:  %s\n", f.colorizedCount(summary.Matched)))
	builder.WriteString(fmt.Sprintf("  Safe:             %d\n", summary.NotMatched))
	if summary.Errors > 0 {
		builder.WriteString(fmt.Sprintf("  Errors:           %s\n", f.colorize(fmt.Sprintf("%d", summary.Errors), color.Yellow)))
	}
	builder.WriteString(fmt.Sprintf("  Execution Time:   %dms\n", summary.ExecutionTimeMs))
	builder.WriteString(fmt.Sprintf("  Workers:          %d\n", summary.Workers))
	builder.WriteString(strings.Repeat("─", 40))
	builder.WriteString("\n")

	// Final status
	if summary.Matched > 0 {
		builder.WriteString(f.colorize("⚠️  VULNERABILITIES DETECTED", color.BrightRed+color.Bold))
	} else {
		builder.WriteString(f.colorize("✅ NO VULNERABILITIES DETECTED", color.BrightGreen+color.Bold))
	}
	builder.WriteString("\n")

	return builder.String(), nil
}

// FormatTemplateList formats a list of templates as a table.
func (f *TableFormatter) FormatTemplateList(templates []*types.Template) (string, error) {
	var builder strings.Builder

	// Header
	builder.WriteString("\n")
	builder.WriteString(f.colorize("📋 AVAILABLE TEMPLATES", color.Bold+color.BrightCyan))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("─", 70))
	builder.WriteString("\n\n")

	// Templates table
	table := tablewriter.NewWriter(&builder)
	table.SetHeader([]string{"#", "ID", "Name", "Severity", "Steps", "Author"})
	table.SetBorder(true)
	table.SetRowLine(false)
	table.SetHeaderLine(true)
	table.SetCenterSeparator("│")
	table.SetColumnSeparator("│")
	table.SetRowSeparator("─")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)

	for i, t := range templates {
		// Truncate long names
		name := t.Info.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		id := t.ID
		if len(id) > 20 {
			id = id[:17] + "..."
		}

		author := t.Info.Author
		if len(author) > 15 {
			author = author[:12] + "..."
		}
		if author == "" {
			author = "-"
		}

		severity := f.colorizedSeverity(string(t.Info.Severity))

		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			id,
			name,
			severity,
			fmt.Sprintf("%d", len(t.Detection.Steps)),
			author,
		})
	}

	table.Render()

	// Summary
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("Total: %d template(s)\n", len(templates)))

	return builder.String(), nil
}

// FormatValidation formats a template validation result as a table.
func (f *TableFormatter) FormatValidation(result *ValidationResult) (string, error) {
	var builder strings.Builder

	builder.WriteString("\n")

	if result.Valid {
		builder.WriteString(f.colorize("✅ TEMPLATE VALID", color.BrightGreen+color.Bold))
	} else {
		builder.WriteString(f.colorize("❌ TEMPLATE INVALID", color.BrightRed+color.Bold))
	}
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("─", 40))
	builder.WriteString("\n")

	builder.WriteString(fmt.Sprintf("  ID:       %s\n", result.TemplateID))
	builder.WriteString(fmt.Sprintf("  Name:     %s\n", result.TemplateName))
	builder.WriteString(fmt.Sprintf("  Severity: %s\n", f.colorizedSeverity(result.Severity)))
	builder.WriteString(fmt.Sprintf("  Steps:    %d\n", result.StepCount))

	if len(result.Errors) > 0 {
		builder.WriteString("\n")
		builder.WriteString(f.colorize("Errors:", color.Red))
		builder.WriteString("\n")
		for _, err := range result.Errors {
			builder.WriteString(fmt.Sprintf("  • %s\n", err))
		}
	}

	if len(result.Warnings) > 0 {
		builder.WriteString("\n")
		builder.WriteString(f.colorize("Warnings:", color.Yellow))
		builder.WriteString("\n")
		for _, warn := range result.Warnings {
			builder.WriteString(fmt.Sprintf("  • %s\n", warn))
		}
	}

	builder.WriteString(strings.Repeat("─", 40))
	builder.WriteString("\n")

	return builder.String(), nil
}

// FormatSystemScan formats a system inventory scan result as a table.
func (f *TableFormatter) FormatSystemScan(result *SystemScanResult) (string, error) {
	var builder strings.Builder

	// Header
	builder.WriteString("\n")
	builder.WriteString(f.colorize("🖥️  SYSTEM SCAN RESULTS", color.Bold+color.BrightCyan))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("─", 60))
	builder.WriteString("\n\n")

	// System Information
	builder.WriteString(f.colorize("System Information", color.Bold))
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("  OS:         %s\n", result.OSInfo.OS))
	builder.WriteString(fmt.Sprintf("  Version:    %s\n", result.OSInfo.Version))
	builder.WriteString(fmt.Sprintf("  Hostname:   %s\n", result.OSInfo.Hostname))
	builder.WriteString(fmt.Sprintf("  Primary IP: %s\n", result.OSInfo.PrimaryIP))
	builder.WriteString("\n")

	// Packages summary
	builder.WriteString(f.colorize("📦 Packages", color.Bold))
	builder.WriteString(fmt.Sprintf(" (%d installed)\n", len(result.Packages)))

	if len(result.Packages) > 0 && len(result.Packages) <= 20 {
		// Show packages in a table if not too many
		table := tablewriter.NewWriter(&builder)
		table.SetHeader([]string{"Name", "Version", "Source"})
		table.SetBorder(true)
		table.SetRowLine(false)
		table.SetHeaderLine(true)
		table.SetCenterSeparator("│")
		table.SetColumnSeparator("│")
		table.SetRowSeparator("─")

		for _, pkg := range result.Packages {
			source := pkg.Source
			if source == "" {
				source = "-"
			}
			table.Append([]string{pkg.Name, pkg.Version, source})
		}
		table.Render()
	} else if len(result.Packages) > 20 {
		builder.WriteString(fmt.Sprintf("  (Use --format json to see all %d packages)\n", len(result.Packages)))
	}

	// Custom results
	if len(result.CustomResults) > 0 {
		builder.WriteString("\n")
		builder.WriteString(f.colorize("🔧 Custom Script Results", color.Bold))
		builder.WriteString(fmt.Sprintf(" (%d scripts)\n", len(result.CustomResults)))

		for name, res := range result.CustomResults {
			status := f.colorize("✅", color.Green)
			if res.ExitCode != 0 {
				status = f.colorize("❌", color.Red)
			}
			builder.WriteString(fmt.Sprintf("  %s %s (exit: %d)\n", status, name, res.ExitCode))
		}
	}

	// Errors
	if len(result.ScanErrors) > 0 {
		builder.WriteString("\n")
		builder.WriteString(f.colorize("⚠️  Scan Errors", color.Yellow))
		builder.WriteString("\n")
		for _, err := range result.ScanErrors {
			builder.WriteString(fmt.Sprintf("  • %s\n", err))
		}
	}

	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("─", 60))
	builder.WriteString("\n")
	builder.WriteString(f.colorize("✅ Scan complete", color.Green))
	builder.WriteString("\n")

	return builder.String(), nil
}

// FormatError formats an error message.
func (f *TableFormatter) FormatError(err error) string {
	return f.colorize(fmt.Sprintf("❌ Error: %v", err), color.Red)
}

// colorize wraps text with ANSI color codes if colors are enabled.
func (f *TableFormatter) colorize(text, colorCode string) string {
	if !f.ColorEnabled {
		return text
	}
	return colorCode + text + color.Reset
}

// colorizedSeverity returns a colorized severity string.
func (f *TableFormatter) colorizedSeverity(severity string) string {
	if !f.ColorEnabled {
		return strings.ToUpper(severity)
	}
	return color.ColorizedSeverity(severity)
}

// colorizedCount returns a colorized count (red if > 0, green if 0).
func (f *TableFormatter) colorizedCount(count int) string {
	if count > 0 {
		return f.colorize(fmt.Sprintf("%d", count), color.BrightRed+color.Bold)
	}
	return f.colorize(fmt.Sprintf("%d", count), color.Green)
}



