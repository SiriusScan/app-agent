package output

import (
	"fmt"
	"strings"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

func init() {
	Register(FormatQuiet, &QuietFormatter{})
}

// QuietFormatter outputs minimal information, suitable for scripting and CI/CD.
// It only outputs essential status information on a single line.
type QuietFormatter struct {
	BaseFormatter
}

// Name returns the format name.
func (f *QuietFormatter) Name() Format {
	return FormatQuiet
}

// FormatScanResults formats template scan results in minimal format.
// Output: "VULNERABLE: 3" or "SAFE: 0"
func (f *QuietFormatter) FormatScanResults(results []*types.Result, summary *ScanSummary) (string, error) {
	if summary.Matched > 0 {
		return fmt.Sprintf("VULNERABLE: %d\n", summary.Matched), nil
	}
	return "SAFE: 0\n", nil
}

// FormatTemplateList formats a list of templates in minimal format.
// Output: count and IDs
func (f *QuietFormatter) FormatTemplateList(templates []*types.Template) (string, error) {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("TEMPLATES: %d\n", len(templates)))

	// List just the IDs
	for _, t := range templates {
		builder.WriteString(t.ID + "\n")
	}

	return builder.String(), nil
}

// FormatValidation formats a template validation result in minimal format.
// Output: "VALID" or "INVALID"
func (f *QuietFormatter) FormatValidation(result *ValidationResult) (string, error) {
	if result.Valid {
		return "VALID\n", nil
	}
	return fmt.Sprintf("INVALID: %d errors\n", len(result.Errors)), nil
}

// FormatSystemScan formats a system inventory scan result in minimal format.
// Output: "PACKAGES: 150"
func (f *QuietFormatter) FormatSystemScan(result *SystemScanResult) (string, error) {
	return fmt.Sprintf("PACKAGES: %d\n", len(result.Packages)), nil
}

// FormatError formats an error message minimally.
func (f *QuietFormatter) FormatError(err error) string {
	return fmt.Sprintf("ERROR: %v\n", err)
}

