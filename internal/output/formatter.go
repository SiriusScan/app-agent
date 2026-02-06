package output

import (
	"github.com/SiriusScan/app-agent/internal/template/types"
)

// Formatter defines the interface for output formatters.
// Each output format (JSON, table, CSV, etc.) implements this interface.
type Formatter interface {
	// Name returns the format name (e.g., "json", "table").
	Name() Format

	// FormatScanResults formats template scan results.
	FormatScanResults(results []*types.Result, summary *ScanSummary) (string, error)

	// FormatTemplateList formats a list of templates.
	FormatTemplateList(templates []*types.Template) (string, error)

	// FormatValidation formats a template validation result.
	FormatValidation(result *ValidationResult) (string, error)

	// FormatSystemScan formats a system inventory scan result.
	FormatSystemScan(result *SystemScanResult) (string, error)

	// FormatError formats an error message.
	FormatError(err error) string
}

// BaseFormatter provides common functionality for formatters.
type BaseFormatter struct {
	// ColorEnabled indicates whether ANSI colors should be used.
	ColorEnabled bool

	// TerminalWidth is the detected terminal width (0 = auto).
	TerminalWidth int
}

// NewBaseFormatter creates a new BaseFormatter with default settings.
func NewBaseFormatter() *BaseFormatter {
	return &BaseFormatter{
		ColorEnabled:  IsColorEnabled(),
		TerminalWidth: GetTerminalWidth(),
	}
}

// SetColorEnabled sets whether colors are enabled.
func (f *BaseFormatter) SetColorEnabled(enabled bool) {
	f.ColorEnabled = enabled
}

// SetTerminalWidth sets the terminal width.
func (f *BaseFormatter) SetTerminalWidth(width int) {
	f.TerminalWidth = width
}



