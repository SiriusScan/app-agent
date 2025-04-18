package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name       string
		envVars    map[string]string
		wantLevel  string
		wantFormat string
		wantErr    bool
	}{
		{
			name:       "default values",
			envVars:    map[string]string{},
			wantLevel:  "info",
			wantFormat: "json",
			wantErr:    false,
		},
		{
			name: "custom values",
			envVars: map[string]string{
				"SIRIUS_LOG_LEVEL":  "debug",
				"SIRIUS_LOG_FORMAT": "text",
			},
			wantLevel:  "debug",
			wantFormat: "text",
			wantErr:    false,
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"SIRIUS_LOG_LEVEL": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			envVars: map[string]string{
				"SIRIUS_LOG_FORMAT": "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment before each test
			os.Clearenv()

			// Set environment variables for test
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			got, err := LoadFromEnv("")
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if got.Log.Level != tt.wantLevel {
				t.Errorf("LoadFromEnv() Log.Level = %v, want %v", got.Log.Level, tt.wantLevel)
			}

			if got.Log.Format != tt.wantFormat {
				t.Errorf("LoadFromEnv() Log.Format = %v, want %v", got.Log.Format, tt.wantFormat)
			}
		})
	}
}

func TestConfig_validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Log: struct {
					Level  string
					Format string
					File   string
				}{
					Level:  "info",
					Format: "json",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			cfg: &Config{
				Log: struct {
					Level  string
					Format string
					File   string
				}{
					Level:  "invalid",
					Format: "json",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			cfg: &Config{
				Log: struct {
					Level  string
					Format string
					File   string
				}{
					Level:  "info",
					Format: "invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.validate(); (err != nil) != tt.wantErr {
				t.Errorf("Config.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  bool
	}{
		{"debug level", "debug", true},
		{"info level", "info", true},
		{"warn level", "warn", true},
		{"error level", "error", true},
		{"invalid level", "invalid", false},
		{"empty level", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidLogLevel(tt.level); got != tt.want {
				t.Errorf("isValidLogLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "use default value",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
		{
			name:         "use environment value",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "custom",
			want:         "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment before each test
			os.Clearenv()

			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			}

			if got := getEnvWithDefault(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("getEnvWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}
