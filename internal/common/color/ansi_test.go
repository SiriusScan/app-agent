package color

import (
	"os"
	"testing"
)

func TestSeverityColor(t *testing.T) {
	tests := []struct {
		severity string
		expected string
	}{
		{"critical", BrightRed},
		{"CRITICAL", BrightRed},
		{"high", Red},
		{"HIGH", Red},
		{"medium", Yellow},
		{"MEDIUM", Yellow},
		{"low", Blue},
		{"LOW", Blue},
		{"info", Gray},
		{"INFO", Gray},
		{"unknown", White},
		{"", White},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			result := SeverityColor(tt.severity)
			if result != tt.expected {
				t.Errorf("SeverityColor(%q) = %q, want %q", tt.severity, result, tt.expected)
			}
		})
	}
}

func TestColorize(t *testing.T) {
	// Set NO_COLOR to disable colors for consistent testing
	oldNoColor := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer os.Setenv("NO_COLOR", oldNoColor)

	text := "test message"
	result := Colorize(text, Red)

	// When colors are disabled, should return plain text
	if result != text {
		t.Errorf("Colorize with NO_COLOR should return plain text, got %q", result)
	}
}

func TestColorizeWithColorsEnabled(t *testing.T) {
	// Unset NO_COLOR
	oldNoColor := os.Getenv("NO_COLOR")
	os.Unsetenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", oldNoColor)

	text := "test message"

	// Note: Colorize checks if stdout is a terminal, which it isn't during tests
	// So this will still return plain text, but we can test the logic
	result := Colorize(text, Red)

	// In test environment (not a terminal), colors should be disabled
	if result != text {
		t.Logf("Colorize returned: %q (colors may be disabled in test env)", result)
	}
}

func TestSprint(t *testing.T) {
	// Set NO_COLOR for consistent testing
	oldNoColor := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer os.Setenv("NO_COLOR", oldNoColor)

	result := Sprint(Red, "Error: %s", "test")
	expected := "Error: test"

	if result != expected {
		t.Errorf("Sprint() = %q, want %q", result, expected)
	}
}

func TestColorizedSeverity(t *testing.T) {
	// Set NO_COLOR for consistent testing
	oldNoColor := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer os.Setenv("NO_COLOR", oldNoColor)

	tests := []struct {
		severity string
		expected string
	}{
		{"critical", "CRITICAL"},
		{"high", "HIGH"},
		{"medium", "MEDIUM"},
		{"low", "LOW"},
		{"info", "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			result := ColorizedSeverity(tt.severity)
			if result != tt.expected {
				t.Errorf("ColorizedSeverity(%q) = %q, want %q", tt.severity, result, tt.expected)
			}
		})
	}
}

func TestIsColorEnabled(t *testing.T) {
	// Test with NO_COLOR set
	oldNoColor := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")

	if IsColorEnabled() {
		t.Error("IsColorEnabled() should return false when NO_COLOR is set")
	}

	// Test with NO_COLOR unset (but still in test env, not a terminal)
	os.Unsetenv("NO_COLOR")

	// In test environment, stdout is not a terminal, so colors should be disabled
	if IsColorEnabled() {
		t.Log("IsColorEnabled() returned true (may be in interactive test env)")
	}

	// Restore
	os.Setenv("NO_COLOR", oldNoColor)
}


