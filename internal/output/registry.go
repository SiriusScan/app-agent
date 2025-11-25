package output

import (
	"fmt"
	"sync"
)

var (
	// formatters holds all registered formatters.
	formatters = make(map[Format]Formatter)
	mu         sync.RWMutex

	// defaultFormat is the format used when none is specified.
	defaultFormat = FormatJSON
)

// Register registers a formatter for the given format.
// This is typically called from init() functions in formatter implementations.
func Register(format Format, formatter Formatter) {
	mu.Lock()
	defer mu.Unlock()
	formatters[format] = formatter
}

// Get returns the formatter for the given format.
// Returns an error if the format is not registered.
func Get(format Format) (Formatter, error) {
	mu.RLock()
	defer mu.RUnlock()

	f, ok := formatters[format]
	if !ok {
		return nil, fmt.Errorf("unknown output format: %s", format)
	}
	return f, nil
}

// GetByString returns the formatter for the given format string.
// Returns an error if the format is not registered.
func GetByString(format string) (Formatter, error) {
	return Get(Format(format))
}

// GetDefault returns the default formatter (JSON).
func GetDefault() Formatter {
	f, _ := Get(defaultFormat)
	return f
}

// SetDefault sets the default format.
func SetDefault(format Format) error {
	if _, err := Get(format); err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	defaultFormat = format
	return nil
}

// Available returns a list of all registered format names.
func Available() []Format {
	mu.RLock()
	defer mu.RUnlock()

	formats := make([]Format, 0, len(formatters))
	for format := range formatters {
		formats = append(formats, format)
	}
	return formats
}

// AvailableStrings returns a list of all registered format names as strings.
func AvailableStrings() []string {
	formats := Available()
	result := make([]string, len(formats))
	for i, f := range formats {
		result[i] = string(f)
	}
	return result
}

// MustGet returns the formatter for the given format, panicking if not found.
// Use this only when you know the format exists (e.g., during initialization).
func MustGet(format Format) Formatter {
	f, err := Get(format)
	if err != nil {
		panic(err)
	}
	return f
}

// AutoDetect returns the best format based on the current environment.
// - If stdout is a terminal, returns "table" for interactive use.
// - Otherwise, returns "json" for piping and scripting.
func AutoDetect() Format {
	if IsTerminal() {
		return FormatTable
	}
	return FormatJSON
}

// GetOrAuto returns the formatter for the given format, or auto-detects if empty.
func GetOrAuto(format string) (Formatter, error) {
	if format == "" || format == "auto" {
		return Get(AutoDetect())
	}
	return GetByString(format)
}

