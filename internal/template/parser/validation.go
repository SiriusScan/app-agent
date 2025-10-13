package parser

import (
	"fmt"
	"strings"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

// ValidateTemplate validates a template structure and returns an error if invalid.
// It checks required fields, severity levels, platform names, weights, and detection logic.
func ValidateTemplate(t *types.Template) error {
	if t == nil {
		return fmt.Errorf("template cannot be nil")
	}

	var errors []string

	// Validate required ID field
	if strings.TrimSpace(t.ID) == "" {
		errors = append(errors, "template ID is required")
	}

	// Validate required Info fields
	if strings.TrimSpace(t.Info.Name) == "" {
		errors = append(errors, "info.name is required")
	}

	if strings.TrimSpace(string(t.Info.Severity)) == "" {
		errors = append(errors, "info.severity is required")
	} else if !types.IsSeverityValid(string(t.Info.Severity)) {
		errors = append(errors, fmt.Sprintf(
			"info.severity '%s' is invalid (must be one of: %s)",
			t.Info.Severity,
			joinSeverities(),
		))
	}

	// Validate detection configuration
	if len(t.Detection.Steps) == 0 {
		errors = append(errors, "detection.steps must contain at least one step")
	}

	// Validate detection logic
	if t.Detection.Logic != "" {
		if t.Detection.Logic != types.LogicAll && t.Detection.Logic != types.LogicAny {
			errors = append(errors, fmt.Sprintf(
				"detection.logic '%s' is invalid (must be 'all' or 'any')",
				t.Detection.Logic,
			))
		}
	}

	// Validate each detection step
	for i, step := range t.Detection.Steps {
		stepErrors := validateDetectionStep(step, i)
		errors = append(errors, stepErrors...)
	}

	// If there are validation errors, return them as a single error
	if len(errors) > 0 {
		return fmt.Errorf("template validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// validateDetectionStep validates a single detection step
func validateDetectionStep(step types.DetectionStep, index int) []string {
	var errors []string
	prefix := fmt.Sprintf("detection.steps[%d]", index)

	// Validate step type is not empty
	if strings.TrimSpace(step.Type) == "" {
		errors = append(errors, fmt.Sprintf("%s.type is required", prefix))
	}

	// Validate platforms if specified
	for i, platform := range step.Platforms {
		if !types.IsPlatformValid(string(platform)) {
			errors = append(errors, fmt.Sprintf(
				"%s.platforms[%d] '%s' is invalid (must be one of: %s)",
				prefix, i, platform, joinPlatforms(),
			))
		}
	}

	// Validate weight is in valid range
	if step.Weight < 0.0 || step.Weight > 1.0 {
		errors = append(errors, fmt.Sprintf(
			"%s.weight %f is invalid (must be between 0.0 and 1.0)",
			prefix, step.Weight,
		))
	}

	return errors
}

// joinSeverities returns a comma-separated string of valid severity levels
func joinSeverities() string {
	severities := types.ValidSeverities()
	strs := make([]string, len(severities))
	for i, s := range severities {
		strs[i] = string(s)
	}
	return strings.Join(strs, ", ")
}

// joinPlatforms returns a comma-separated string of valid platform names
func joinPlatforms() string {
	platforms := types.ValidPlatforms()
	strs := make([]string, len(platforms))
	for i, p := range platforms {
		strs[i] = string(p)
	}
	return strings.Join(strs, ", ")
}

