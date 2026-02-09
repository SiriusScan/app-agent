package config

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a zap.Logger whose level is driven by the LOG_LEVEL
// environment variable. Valid values: "debug", "info", "warn", "error".
// Default: "info".
//
// In debug mode a full DevelopmentConfig is used (human-friendly, with
// caller info). At info and above a production-style config is used
// (JSON, no caller) to keep output concise.
func NewLogger() *zap.Logger {
	level := parseZapLevel(os.Getenv("LOG_LEVEL"))

	var cfg zap.Config
	if level == zapcore.DebugLevel {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	cfg.Level = zap.NewAtomicLevelAt(level)

	logger, err := cfg.Build()
	if err != nil {
		// Fallback to a no-op logger; should never happen.
		logger = zap.NewNop()
	}
	return logger
}

// parseZapLevel converts a string log level to a zapcore.Level.
func parseZapLevel(s string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return zapcore.DebugLevel
	case "info", "":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
