package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/common/color"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

func init() {
	Register(FormatText, &TextFormatter{BaseFormatter: *NewBaseFormatter()})
}

// TextFormatter outputs data as human-readable text with emojis.
type TextFormatter struct {
	BaseFormatter
}

// Name returns the format name.
func (f *TextFormatter) Name() Format {
	return FormatText
}

// FormatScanResults formats template scan results as human-readable text.
func (f *TextFormatter) FormatScanResults(results []*types.Result, summary *ScanSummary) (string, error) {
	var builder strings.Builder

	builder.WriteString("🔍 Template Scan Results\n")
	builder.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Summary section
	builder.WriteString("📊 Summary:\n")
	builder.WriteString(fmt.Sprintf("  Total Templates: %d\n", summary.TotalTemplates))
	builder.WriteString(fmt.Sprintf("  Matched: %d\n", summary.Matched))
	builder.WriteString(fmt.Sprintf("  Execution Time: %dms\n\n", summary.ExecutionTimeMs))

	// Results section
	builder.WriteString("📋 Template Results:\n\n")
	for i, result := range results {
		if result == nil {
			builder.WriteString(fmt.Sprintf("[%d] ❓ Unknown result\n", i+1))
			continue
		}

		status := "❌"
		if result.Matched {
			status = "✅"
		}

		builder.WriteString(fmt.Sprintf("[%d] %s %s (ID: %s)\n", i+1, status, result.TemplateName, result.TemplateID))

		// Color-code severity
		severityText := f.colorizedSeverity(string(result.Severity))
		builder.WriteString(fmt.Sprintf("    Severity: %s | Confidence: %.2f | Risk: %.1f\n", severityText, result.Confidence, result.RiskScore))

		if result.Matched && len(result.Steps) > 0 {
			matchedSteps := countMatchedSteps(result)
			builder.WriteString(fmt.Sprintf("    Matched Steps: %d/%d\n", matchedSteps, len(result.Steps)))
		}

		if len(result.Errors) > 0 {
			builder.WriteString(fmt.Sprintf("    Errors: %s\n", strings.Join(result.Errors, ", ")))
		}
		builder.WriteString("\n")
	}

	builder.WriteString(strings.Repeat("=", 50) + "\n")
	if summary.Matched > 0 {
		message := f.colorize("⚠️  Vulnerabilities detected!", color.BrightRed+color.Bold)
		builder.WriteString(message + "\n")
	} else {
		message := f.colorize("✅ No vulnerabilities detected.", color.BrightGreen+color.Bold)
		builder.WriteString(message + "\n")
	}

	return builder.String(), nil
}

// FormatTemplateList formats a list of templates as human-readable text.
func (f *TextFormatter) FormatTemplateList(templates []*types.Template) (string, error) {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("📋 Found %d template(s):\n\n", len(templates)))

	for i, t := range templates {
		builder.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, t.Info.Name, t.ID))
		builder.WriteString(fmt.Sprintf("   Severity: %s\n", f.colorizedSeverity(string(t.Info.Severity))))
		if t.Info.Author != "" {
			builder.WriteString(fmt.Sprintf("   Author: %s\n", t.Info.Author))
		}
		builder.WriteString(fmt.Sprintf("   Steps: %d\n", len(t.Detection.Steps)))
		if t.FilePath != "" {
			builder.WriteString(fmt.Sprintf("   File: %s\n", t.FilePath))
		}
		if t.Info.Description != "" {
			desc := t.Info.Description
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			builder.WriteString(fmt.Sprintf("   Description: %s\n", desc))
		}
		if i < len(templates)-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String(), nil
}

// FormatValidation formats a template validation result as human-readable text.
func (f *TextFormatter) FormatValidation(result *ValidationResult) (string, error) {
	var builder strings.Builder

	if result.Valid {
		builder.WriteString(fmt.Sprintf("✅ Template is valid: %s\n", result.TemplateID))
	} else {
		builder.WriteString(fmt.Sprintf("❌ Template is invalid: %s\n", result.TemplateID))
	}

	builder.WriteString(fmt.Sprintf("   Name: %s\n", result.TemplateName))
	builder.WriteString(fmt.Sprintf("   Severity: %s\n", f.colorizedSeverity(result.Severity)))
	builder.WriteString(fmt.Sprintf("   Steps: %d\n", result.StepCount))

	if len(result.Errors) > 0 {
		builder.WriteString("\n❌ Errors:\n")
		for _, err := range result.Errors {
			builder.WriteString(fmt.Sprintf("   • %s\n", err))
		}
	}

	if len(result.Warnings) > 0 {
		builder.WriteString("\n⚠️  Warnings:\n")
		for _, warn := range result.Warnings {
			builder.WriteString(fmt.Sprintf("   • %s\n", warn))
		}
	}

	return builder.String(), nil
}

// FormatSystemScan formats a system inventory scan result as human-readable text.
func (f *TextFormatter) FormatSystemScan(result *SystemScanResult) (string, error) {
	var builder strings.Builder

	builder.WriteString("📊 System Scan Results\n")
	builder.WriteString(strings.Repeat("=", 60) + "\n\n")

	// OS Information
	builder.WriteString("🖥️  System Information:\n")
	builder.WriteString(fmt.Sprintf("   OS: %s\n", result.OSInfo.OS))
	builder.WriteString(fmt.Sprintf("   Version: %s\n", result.OSInfo.Version))
	builder.WriteString(fmt.Sprintf("   Hostname: %s\n", result.OSInfo.Hostname))
	builder.WriteString(fmt.Sprintf("   Primary IP: %s\n", result.OSInfo.PrimaryIP))
	builder.WriteString("\n")

	// Packages
	if len(result.Packages) > 0 {
		builder.WriteString(fmt.Sprintf("📦 Installed Packages: %d\n\n", len(result.Packages)))
	}

	// Custom Scripts
	if len(result.CustomResults) > 0 {
		builder.WriteString(fmt.Sprintf("🔧 Custom Script Results: %d\n", len(result.CustomResults)))
		for scriptName, scriptResult := range result.CustomResults {
			builder.WriteString(fmt.Sprintf("   • %s\n", scriptName))
			if scriptResult.ExitCode == 0 {
				builder.WriteString("     ✅ Success\n")
			} else {
				builder.WriteString(fmt.Sprintf("     ❌ Exit code: %d\n", scriptResult.ExitCode))
			}
		}
		builder.WriteString("\n")
	}

	// Errors
	if len(result.ScanErrors) > 0 {
		builder.WriteString("⚠️  Scan Errors:\n")
		for _, err := range result.ScanErrors {
			builder.WriteString(fmt.Sprintf("   • %s\n", err))
		}
		builder.WriteString("\n")
	}

	builder.WriteString(strings.Repeat("=", 60) + "\n")
	builder.WriteString("✅ Scan complete\n")

	return builder.String(), nil
}

// FormatError formats an error message.
func (f *TextFormatter) FormatError(err error) string {
	return f.colorize(fmt.Sprintf("❌ Error: %v", err), color.Red)
}

// colorize wraps text with ANSI color codes if colors are enabled.
func (f *TextFormatter) colorize(text, colorCode string) string {
	if !f.ColorEnabled {
		return text
	}
	return colorCode + text + color.Reset
}

// colorizedSeverity returns a colorized severity string.
func (f *TextFormatter) colorizedSeverity(severity string) string {
	if !f.ColorEnabled {
		return strings.ToUpper(severity)
	}
	return color.ColorizedSeverity(severity)
}

// countMatchedSteps counts how many steps matched in a result.
func countMatchedSteps(result *types.Result) int {
	count := 0
	for _, step := range result.Steps {
		if step.Matched {
			count++
		}
	}
	return count
}

// FormatDuration formats a duration in a human-readable way.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}



