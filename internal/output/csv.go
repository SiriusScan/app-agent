package output

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

func init() {
	Register(FormatCSV, &CSVFormatter{})
}

// CSVFormatter outputs data as comma-separated values.
type CSVFormatter struct {
	BaseFormatter
}

// Name returns the format name.
func (f *CSVFormatter) Name() Format {
	return FormatCSV
}

// FormatScanResults formats template scan results as CSV.
func (f *CSVFormatter) FormatScanResults(results []*types.Result, summary *ScanSummary) (string, error) {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)

	// Header
	header := []string{
		"template_id",
		"template_name",
		"severity",
		"matched",
		"risk_score",
		"confidence",
		"vulnerability_id",
		"cve",
		"host",
		"timestamp",
		"errors",
	}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Data rows
	for _, result := range results {
		if result == nil {
			continue
		}

		matched := "false"
		if result.Matched {
			matched = "true"
		}

		cve := strings.Join(result.CVE, ";")
		errors := strings.Join(result.Errors, ";")

		row := []string{
			result.TemplateID,
			result.TemplateName,
			string(result.Severity),
			matched,
			fmt.Sprintf("%.2f", result.RiskScore),
			fmt.Sprintf("%.2f", result.Confidence),
			result.VulnerabilityID,
			cve,
			result.Host,
			result.Timestamp.Format("2006-01-02T15:04:05Z"),
			errors,
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV write error: %w", err)
	}

	return builder.String(), nil
}

// FormatTemplateList formats a list of templates as CSV.
func (f *CSVFormatter) FormatTemplateList(templates []*types.Template) (string, error) {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)

	// Header
	header := []string{
		"id",
		"name",
		"severity",
		"author",
		"version",
		"steps",
		"tags",
		"file_path",
	}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Data rows
	for _, t := range templates {
		tags := strings.Join(t.Info.Tags, ";")

		row := []string{
			t.ID,
			t.Info.Name,
			string(t.Info.Severity),
			t.Info.Author,
			t.Info.Version,
			fmt.Sprintf("%d", len(t.Detection.Steps)),
			tags,
			t.FilePath,
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV write error: %w", err)
	}

	return builder.String(), nil
}

// FormatValidation formats a template validation result as CSV.
func (f *CSVFormatter) FormatValidation(result *ValidationResult) (string, error) {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)

	// Header
	header := []string{
		"id",
		"name",
		"valid",
		"severity",
		"steps",
		"errors",
		"warnings",
	}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Data row
	valid := "false"
	if result.Valid {
		valid = "true"
	}

	errors := strings.Join(result.Errors, ";")
	warnings := strings.Join(result.Warnings, ";")

	row := []string{
		result.TemplateID,
		result.TemplateName,
		valid,
		result.Severity,
		fmt.Sprintf("%d", result.StepCount),
		errors,
		warnings,
	}
	if err := writer.Write(row); err != nil {
		return "", fmt.Errorf("failed to write CSV row: %w", err)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV write error: %w", err)
	}

	return builder.String(), nil
}

// FormatSystemScan formats a system inventory scan result as CSV.
func (f *CSVFormatter) FormatSystemScan(result *SystemScanResult) (string, error) {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)

	// Header for packages
	header := []string{
		"type",
		"name",
		"version",
		"source",
		"os",
		"hostname",
	}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Package rows
	for _, pkg := range result.Packages {
		row := []string{
			"package",
			pkg.Name,
			pkg.Version,
			pkg.Source,
			result.OSInfo.OS,
			result.OSInfo.Hostname,
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV write error: %w", err)
	}

	return builder.String(), nil
}

// FormatError formats an error message.
func (f *CSVFormatter) FormatError(err error) string {
	return fmt.Sprintf("error,%q", err.Error())
}

