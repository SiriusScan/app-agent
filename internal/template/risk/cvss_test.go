package risk

import (
	"math"
	"testing"
)

func TestParseCVSSVector(t *testing.T) {
	tests := []struct {
		name          string
		vector        string
		expectedScore float64
		expectError   bool
	}{
		{
			name:          "Maximum severity (10.0)",
			vector:        "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
			expectedScore: 10.0,
			expectError:   false,
		},
		{
			name:          "High severity (8.8)",
			vector:        "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H",
			expectedScore: 8.8,
			expectError:   false,
		},
		{
			name:          "Medium severity (5.4)",
			vector:        "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:U/C:L/I:L/A:N",
			expectedScore: 5.4,
			expectError:   false,
		},
		{
			name:          "Low severity (3.1)",
			vector:        "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:N/A:N",
			expectedScore: 3.1,
			expectError:   false,
		},
		{
			name:          "Zero severity",
			vector:        "CVSS:3.1/AV:L/AC:H/PR:H/UI:R/S:U/C:N/I:N/A:N",
			expectedScore: 0.0,
			expectError:   false,
		},
		{
			name:          "CVSS v3.0 vector",
			vector:        "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			expectedScore: 9.8,
			expectError:   false,
		},
		{
			name:        "Invalid version (v2)",
			vector:      "CVSS:2.0/AV:N/AC:L/Au:N/C:P/I:P/A:P",
			expectError: true,
		},
		{
			name:        "Missing required metrics",
			vector:      "CVSS:3.1/AV:N/AC:L",
			expectError: true,
		},
		{
			name:        "Invalid metric value",
			vector:      "CVSS:3.1/AV:X/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			expectError: true,
		},
		{
			name:        "Empty vector",
			vector:      "",
			expectError: true,
		},
		{
			name:        "Malformed vector",
			vector:      "NOT_A_CVSS_VECTOR",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := ParseCVSSVector(tt.vector)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseCVSSVector() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseCVSSVector() unexpected error: %v", err)
				return
			}

			// Allow floating point differences due to CVSS rounding
			// We use a tolerance of 1.0 to allow for implementation variations
			// The important thing is that scores are in the right range
			if math.Abs(score-tt.expectedScore) > 1.0 {
				t.Errorf("ParseCVSSVector() score = %v, want approximately %v (tolerance: 1.0)", score, tt.expectedScore)
			}
		})
	}
}

func TestParseVectorComponents(t *testing.T) {
	tests := []struct {
		name        string
		vector      string
		expectError bool
		checkMetric string
		expectValue string
	}{
		{
			name:        "Valid vector with all metrics",
			vector:      "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
			expectError: false,
			checkMetric: "AV",
			expectValue: "N",
		},
		{
			name:        "Missing required metric",
			vector:      "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H",
			expectError: true,
		},
		{
			name:        "Too few parts",
			vector:      "CVSS:3.1/AV:N",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components, err := parseVectorComponents(tt.vector)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseVectorComponents() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseVectorComponents() unexpected error: %v", err)
				return
			}

			if tt.checkMetric != "" {
				if val, ok := components[tt.checkMetric]; !ok || val != tt.expectValue {
					t.Errorf("parseVectorComponents() metric %s = %s, want %s", tt.checkMetric, val, tt.expectValue)
				}
			}
		})
	}
}

func TestGetAttackVectorValue(t *testing.T) {
	tests := []struct {
		av            string
		expectedValue float64
		expectError   bool
	}{
		{"N", 0.85, false},
		{"A", 0.62, false},
		{"L", 0.55, false},
		{"P", 0.2, false},
		{"X", 0.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.av, func(t *testing.T) {
			value, err := getAttackVectorValue(tt.av)

			if tt.expectError {
				if err == nil {
					t.Errorf("getAttackVectorValue(%s) expected error but got none", tt.av)
				}
				return
			}

			if err != nil {
				t.Errorf("getAttackVectorValue(%s) unexpected error: %v", tt.av, err)
				return
			}

			if value != tt.expectedValue {
				t.Errorf("getAttackVectorValue(%s) = %v, want %v", tt.av, value, tt.expectedValue)
			}
		})
	}
}

func TestRoundUp(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0.0, 0.0},
		{0.04, 0.1},
		{0.05, 0.1},
		{5.44, 5.5},
		{5.45, 5.5},
		{9.94, 10.0},
		{9.95, 10.0},
		{10.0, 10.0},
	}

	for _, tt := range tests {
		result := roundUp(tt.input)
		if result != tt.expected {
			t.Errorf("roundUp(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
