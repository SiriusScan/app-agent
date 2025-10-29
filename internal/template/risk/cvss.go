package risk

import (
	"fmt"
	"math"
	"strings"
)

// ParseCVSSVector extracts the base score from a CVSS v3.x vector string
// Supports CVSS v3.0 and v3.1 vectors
// Returns the base score or error if invalid
func ParseCVSSVector(vector string) (float64, error) {
	// Validate format: CVSS:3.1/AV:X/AC:X/...
	if !strings.HasPrefix(vector, "CVSS:3.") {
		return 0, fmt.Errorf("only CVSS v3.x vectors supported, got: %s", vector)
	}

	// Extract components
	components, err := parseVectorComponents(vector)
	if err != nil {
		return 0, fmt.Errorf("failed to parse vector components: %w", err)
	}

	// Calculate base score using CVSS v3 formula
	score, err := calculateBaseScore(components)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate base score: %w", err)
	}

	return score, nil
}

// parseVectorComponents extracts metric values from CVSS vector string
func parseVectorComponents(vector string) (map[string]string, error) {
	components := make(map[string]string)

	// Split on forward slash
	parts := strings.Split(vector, "/")
	if len(parts) < 9 { // CVSS:3.x + 8 base metrics
		return nil, fmt.Errorf("invalid CVSS vector format: insufficient metrics")
	}

	// Skip first part (CVSS:3.x version)
	for _, part := range parts[1:] {
		// Parse metric:value format
		kv := strings.Split(part, ":")
		if len(kv) != 2 {
			continue // Skip invalid parts
		}
		components[kv[0]] = kv[1]
	}

	// Validate required base metrics
	requiredMetrics := []string{"AV", "AC", "PR", "UI", "S", "C", "I", "A"}
	for _, metric := range requiredMetrics {
		if _, ok := components[metric]; !ok {
			return nil, fmt.Errorf("missing required base metric: %s", metric)
		}
	}

	return components, nil
}

// calculateBaseScore computes CVSS v3 base score using official formula
// Reference: https://www.first.org/cvss/specification-document
func calculateBaseScore(components map[string]string) (float64, error) {
	// Get Impact Sub Score (ISS)
	iss, err := calculateImpactSubScore(components)
	if err != nil {
		return 0, err
	}

	// Get Exploitability Sub Score
	exploitability, err := calculateExploitability(components)
	if err != nil {
		return 0, err
	}

	// Calculate Impact based on scope
	var impact float64
	scope := components["S"]
	if scope == "U" { // Unchanged
		impact = 6.42 * iss
	} else if scope == "C" { // Changed
		impact = 7.52*(iss-0.029) - 3.25*math.Pow(iss-0.02, 15)
	} else {
		return 0, fmt.Errorf("invalid scope value: %s", scope)
	}

	// Calculate Base Score
	var baseScore float64
	if impact <= 0 {
		baseScore = 0.0
	} else {
		if scope == "U" {
			baseScore = roundUp(math.Min(impact+exploitability, 10.0))
		} else { // Changed
			baseScore = roundUp(math.Min(1.08*(impact+exploitability), 10.0))
		}
	}

	return baseScore, nil
}

// calculateImpactSubScore calculates the Impact Sub Score (ISS)
func calculateImpactSubScore(components map[string]string) (float64, error) {
	// Get impact metric values
	c, err := getImpactValue(components["C"])
	if err != nil {
		return 0, fmt.Errorf("invalid confidentiality impact: %w", err)
	}

	i, err := getImpactValue(components["I"])
	if err != nil {
		return 0, fmt.Errorf("invalid integrity impact: %w", err)
	}

	a, err := getImpactValue(components["A"])
	if err != nil {
		return 0, fmt.Errorf("invalid availability impact: %w", err)
	}

	// ISS = 1 - [(1-C) × (1-I) × (1-A)]
	iss := 1 - ((1 - c) * (1 - i) * (1 - a))
	return iss, nil
}

// calculateExploitability calculates the Exploitability Sub Score
func calculateExploitability(components map[string]string) (float64, error) {
	// Get attack vector value
	av, err := getAttackVectorValue(components["AV"])
	if err != nil {
		return 0, fmt.Errorf("invalid attack vector: %w", err)
	}

	// Get attack complexity value
	ac, err := getAttackComplexityValue(components["AC"])
	if err != nil {
		return 0, fmt.Errorf("invalid attack complexity: %w", err)
	}

	// Get privileges required value (depends on scope)
	pr, err := getPrivilegesRequiredValue(components["PR"], components["S"])
	if err != nil {
		return 0, fmt.Errorf("invalid privileges required: %w", err)
	}

	// Get user interaction value
	ui, err := getUserInteractionValue(components["UI"])
	if err != nil {
		return 0, fmt.Errorf("invalid user interaction: %w", err)
	}

	// Exploitability = 8.22 × AV × AC × PR × UI
	exploitability := 8.22 * av * ac * pr * ui
	return exploitability, nil
}

// getAttackVectorValue returns the numerical value for Attack Vector
func getAttackVectorValue(av string) (float64, error) {
	switch av {
	case "N": // Network
		return 0.85, nil
	case "A": // Adjacent
		return 0.62, nil
	case "L": // Local
		return 0.55, nil
	case "P": // Physical
		return 0.2, nil
	default:
		return 0, fmt.Errorf("unknown attack vector: %s", av)
	}
}

// getAttackComplexityValue returns the numerical value for Attack Complexity
func getAttackComplexityValue(ac string) (float64, error) {
	switch ac {
	case "L": // Low
		return 0.77, nil
	case "H": // High
		return 0.44, nil
	default:
		return 0, fmt.Errorf("unknown attack complexity: %s", ac)
	}
}

// getPrivilegesRequiredValue returns the numerical value for Privileges Required
// Value depends on whether scope is changed or unchanged
func getPrivilegesRequiredValue(pr string, scope string) (float64, error) {
	switch pr {
	case "N": // None
		return 0.85, nil
	case "L": // Low
		if scope == "C" { // Changed
			return 0.68, nil
		}
		return 0.62, nil // Unchanged
	case "H": // High
		if scope == "C" { // Changed
			return 0.50, nil
		}
		return 0.27, nil // Unchanged
	default:
		return 0, fmt.Errorf("unknown privileges required: %s", pr)
	}
}

// getUserInteractionValue returns the numerical value for User Interaction
func getUserInteractionValue(ui string) (float64, error) {
	switch ui {
	case "N": // None
		return 0.85, nil
	case "R": // Required
		return 0.62, nil
	default:
		return 0, fmt.Errorf("unknown user interaction: %s", ui)
	}
}

// getImpactValue returns the numerical value for impact metrics (C/I/A)
func getImpactValue(impact string) (float64, error) {
	switch impact {
	case "H": // High
		return 0.56, nil
	case "L": // Low
		return 0.22, nil
	case "N": // None
		return 0.0, nil
	default:
		return 0, fmt.Errorf("unknown impact value: %s", impact)
	}
}

// roundUp rounds a score to one decimal place, always rounding up
// This matches the CVSS v3 specification
func roundUp(score float64) float64 {
	return math.Ceil(score*10) / 10
}









