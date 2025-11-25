package output

import (
	"os"

	"golang.org/x/term"
)

// DefaultTerminalWidth is the default width when terminal size cannot be detected.
const DefaultTerminalWidth = 80

// IsColorEnabled checks if color output should be enabled.
// Colors are disabled if:
// - NO_COLOR environment variable is set
// - TERM is "dumb"
// - stdout is not a terminal
func IsColorEnabled() bool {
	// Check NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check for dumb terminal
	if os.Getenv("TERM") == "dumb" {
		return false
	}

	// Check if stdout is a terminal
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// IsTerminal returns true if stdout is connected to a terminal.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// GetTerminalWidth returns the current terminal width.
// Returns DefaultTerminalWidth if the width cannot be determined.
func GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return DefaultTerminalWidth
	}
	return width
}

// GetTerminalSize returns the current terminal width and height.
// Returns default values if the size cannot be determined.
func GetTerminalSize() (width, height int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		return DefaultTerminalWidth, 24
	}
	return w, h
}

