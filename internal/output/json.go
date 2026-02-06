package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

func init() {
	Register(FormatJSON, &JSONFormatter{})
	Register(FormatJSONL, &JSONLFormatter{})
}

// JSONFormatter outputs data as pretty-printed JSON.
type JSONFormatter struct {
	BaseFormatter
}

// Name returns the format name.
func (f *JSONFormatter) Name() Format {
	return FormatJSON
}

// FormatScanResults formats template scan results as JSON.
func (f *JSONFormatter) FormatScanResults(results []*types.Result, summary *ScanSummary) (string, error) {
	output := &ScanOutput{
		Summary: summary,
		Results: results,
	}
	return f.marshal(output)
}

// FormatTemplateList formats a list of templates as JSON.
func (f *JSONFormatter) FormatTemplateList(templates []*types.Template) (string, error) {
	list := make([]*TemplateInfo, len(templates))
	for i, t := range templates {
		list[i] = NewTemplateInfo(t)
	}
	return f.marshal(list)
}

// FormatValidation formats a template validation result as JSON.
func (f *JSONFormatter) FormatValidation(result *ValidationResult) (string, error) {
	return f.marshal(result)
}

// FormatSystemScan formats a system inventory scan result as JSON.
func (f *JSONFormatter) FormatSystemScan(result *SystemScanResult) (string, error) {
	return f.marshal(result)
}

// FormatError formats an error message as JSON.
func (f *JSONFormatter) FormatError(err error) string {
	output := map[string]interface{}{
		"error": err.Error(),
	}
	result, _ := f.marshal(output)
	return result
}

func (f *JSONFormatter) marshal(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// JSONLFormatter outputs data as JSON Lines (one JSON object per line).
type JSONLFormatter struct {
	BaseFormatter
}

// Name returns the format name.
func (f *JSONLFormatter) Name() Format {
	return FormatJSONL
}

// FormatScanResults formats template scan results as JSON Lines.
func (f *JSONLFormatter) FormatScanResults(results []*types.Result, summary *ScanSummary) (string, error) {
	var builder strings.Builder

	for _, result := range results {
		if result == nil {
			continue
		}
		data, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("failed to marshal result: %w", err)
		}
		builder.Write(data)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// FormatTemplateList formats a list of templates as JSON Lines.
func (f *JSONLFormatter) FormatTemplateList(templates []*types.Template) (string, error) {
	var builder strings.Builder

	for _, t := range templates {
		info := NewTemplateInfo(t)
		data, err := json.Marshal(info)
		if err != nil {
			return "", fmt.Errorf("failed to marshal template: %w", err)
		}
		builder.Write(data)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// FormatValidation formats a template validation result as JSON Lines.
func (f *JSONLFormatter) FormatValidation(result *ValidationResult) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal validation result: %w", err)
	}
	return string(data) + "\n", nil
}

// FormatSystemScan formats a system inventory scan result as JSON Lines.
func (f *JSONLFormatter) FormatSystemScan(result *SystemScanResult) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal system scan result: %w", err)
	}
	return string(data) + "\n", nil
}

// FormatError formats an error message as JSON.
func (f *JSONLFormatter) FormatError(err error) string {
	output := map[string]string{
		"error": err.Error(),
	}
	data, _ := json.Marshal(output)
	return string(data)
}



