// Package color provides ANSI color code utilities for terminal output.
package color

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color codes
const (
	Reset = "\033[0m"
	Bold  = "\033[1m"
	Dim   = "\033[2m"

	// Foreground colors
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	// Bright colors
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
)

// IsColorEnabled checks if color output should be enabled based on environment.
// Colors are disabled if NO_COLOR is set or if not outputting to a terminal.
func IsColorEnabled() bool {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if stdout is a terminal
	// Note: This is a simple check; for production use, consider golang.org/x/term
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Colorize wraps text with ANSI color codes if colors are enabled.
func Colorize(text, color string) string {
	if !IsColorEnabled() {
		return text
	}
	return color + text + Reset
}

// Sprint formats and colorizes text (alias for Colorize with fmt.Sprintf).
func Sprint(color string, format string, args ...interface{}) string {
	text := fmt.Sprintf(format, args...)
	return Colorize(text, color)
}

// SeverityColor returns the appropriate color for a severity level.
func SeverityColor(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return BrightRed
	case "high":
		return Red
	case "medium":
		return Yellow
	case "low":
		return Blue
	case "info":
		return Gray
	default:
		return White
	}
}

// ColorizedSeverity returns a colorized severity string.
func ColorizedSeverity(severity string) string {
	color := SeverityColor(severity)
	return Colorize(strings.ToUpper(severity), color+Bold)
}


