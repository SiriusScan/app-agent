package types

import (
	"encoding/json"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestTemplateSerialization(t *testing.T) {
	template := Template{
		ID: "test-template-001",
		Info: TemplateInfo{
			Name:        "Test Template",
			Author:      "Test Author",
			Severity:    SeverityHigh,
			Description: "Test description",
			References:  []string{"https://example.com"},
			CVE:         []string{"CVE-2024-1234"},
			Tags:        []string{"test", "vulnerability"},
			Version:     "1.0.0",
		},
		Detection: DetectionConfig{
			Logic: LogicAll,
			Steps: []DetectionStep{
				{
					Type:      "file_hash",
					Platforms: []Platform{PlatformLinux},
					Weight:    0.8,
					Config: map[string]interface{}{
						"path": "/usr/bin/test",
						"hash": "abc123",
					},
				},
			},
		},
	}

	// Test JSON marshaling
	jsonData, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal template to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var jsonTemplate Template
	err = json.Unmarshal(jsonData, &jsonTemplate)
	if err != nil {
		t.Fatalf("Failed to unmarshal template from JSON: %v", err)
	}

	if jsonTemplate.ID != template.ID {
		t.Errorf("JSON roundtrip failed: ID mismatch")
	}
	if jsonTemplate.Info.Name != template.Info.Name {
		t.Errorf("JSON roundtrip failed: Name mismatch")
	}

	t.Log("✅ JSON serialization working")

	// Test YAML marshaling
	yamlData, err := yaml.Marshal(template)
	if err != nil {
		t.Fatalf("Failed to marshal template to YAML: %v", err)
	}

	// Test YAML unmarshaling
	var yamlTemplate Template
	err = yaml.Unmarshal(yamlData, &yamlTemplate)
	if err != nil {
		t.Fatalf("Failed to unmarshal template from YAML: %v", err)
	}

	if yamlTemplate.ID != template.ID {
		t.Errorf("YAML roundtrip failed: ID mismatch")
	}
	if yamlTemplate.Info.Severity != template.Info.Severity {
		t.Errorf("YAML roundtrip failed: Severity mismatch")
	}

	t.Log("✅ YAML serialization working")
}

func TestResultSerialization(t *testing.T) {
	result := Result{
		TemplateID:   "test-template-001",
		TemplateName: "Test Template",
		Severity:     SeverityHigh,
		Matched:      true,
		Confidence:   0.95,
		Steps: []StepResult{
			{
				Step:    0,
				Type:    "file_hash",
				Matched: true,
				Evidence: map[string]interface{}{
					"file":          "/usr/bin/test",
					"expected_hash": "abc123",
					"actual_hash":   "abc123",
				},
				Duration: time.Second * 2,
			},
		},
		Errors:    []string{},
		Timestamp: time.Now(),
		Host:      "testhost",
	}

	// Test JSON marshaling
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal result to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var jsonResult Result
	err = json.Unmarshal(jsonData, &jsonResult)
	if err != nil {
		t.Fatalf("Failed to unmarshal result from JSON: %v", err)
	}

	if jsonResult.TemplateID != result.TemplateID {
		t.Errorf("JSON roundtrip failed: TemplateID mismatch")
	}
	if jsonResult.Matched != result.Matched {
		t.Errorf("JSON roundtrip failed: Matched mismatch")
	}
	if jsonResult.Confidence != result.Confidence {
		t.Errorf("JSON roundtrip failed: Confidence mismatch")
	}

	t.Log("✅ Result JSON serialization working")
}

func TestSeverityValidation(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		valid    bool
	}{
		{"critical", "critical", true},
		{"high", "high", true},
		{"medium", "medium", true},
		{"low", "low", true},
		{"info", "info", true},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := IsSeverityValid(tt.severity)
			if valid != tt.valid {
				t.Errorf("IsSeverityValid(%q) = %v, want %v", tt.severity, valid, tt.valid)
			}
		})
	}

	t.Log("✅ Severity validation working")
}

func TestPlatformValidation(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		valid    bool
	}{
		{"linux", "linux", true},
		{"darwin", "darwin", true},
		{"windows", "windows", true},
		{"invalid", "freebsd", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := IsPlatformValid(tt.platform)
			if valid != tt.valid {
				t.Errorf("IsPlatformValid(%q) = %v, want %v", tt.platform, valid, tt.valid)
			}
		})
	}

	t.Log("✅ Platform validation working")
}

func TestDetectionLogic(t *testing.T) {
	tests := []struct {
		name  string
		logic DetectionLogic
	}{
		{"all logic", LogicAll},
		{"any logic", LogicAny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DetectionConfig{
				Logic: tt.logic,
				Steps: []DetectionStep{},
			}

			// Test YAML marshaling
			data, err := yaml.Marshal(config)
			if err != nil {
				t.Fatalf("Failed to marshal DetectionConfig: %v", err)
			}

			// Test YAML unmarshaling
			var unmarshaled DetectionConfig
			err = yaml.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal DetectionConfig: %v", err)
			}

			if unmarshaled.Logic != tt.logic {
				t.Errorf("Logic mismatch: got %v, want %v", unmarshaled.Logic, tt.logic)
			}
		})
	}

	t.Log("✅ Detection logic types working")
}

func TestStepConfigSerialization(t *testing.T) {
	step := DetectionStep{
		Type:      "file_hash",
		Platforms: []Platform{PlatformLinux, PlatformDarwin},
		Weight:    0.75,
		Config: map[string]interface{}{
			"path":      "/usr/bin/test",
			"hash":      "abc123",
			"algorithm": "sha256",
		},
	}

	// Test YAML marshaling
	yamlData, err := yaml.Marshal(step)
	if err != nil {
		t.Fatalf("Failed to marshal step to YAML: %v", err)
	}

	// Test YAML unmarshaling
	var unmarshaled DetectionStep
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal step from YAML: %v", err)
	}

	if unmarshaled.Type != step.Type {
		t.Errorf("Type mismatch: got %v, want %v", unmarshaled.Type, step.Type)
	}
	if unmarshaled.Weight != step.Weight {
		t.Errorf("Weight mismatch: got %v, want %v", unmarshaled.Weight, step.Weight)
	}
	if len(unmarshaled.Platforms) != len(step.Platforms) {
		t.Errorf("Platforms length mismatch: got %v, want %v", len(unmarshaled.Platforms), len(step.Platforms))
	}

	t.Log("✅ DetectionStep serialization working")
}

func TestValidSeverities(t *testing.T) {
	severities := ValidSeverities()
	if len(severities) != 5 {
		t.Errorf("Expected 5 valid severities, got %d", len(severities))
	}

	expected := []Severity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo}
	for i, sev := range expected {
		if severities[i] != sev {
			t.Errorf("Severity at index %d: got %v, want %v", i, severities[i], sev)
		}
	}

	t.Log("✅ ValidSeverities() working")
}

func TestValidPlatforms(t *testing.T) {
	platforms := ValidPlatforms()
	if len(platforms) != 3 {
		t.Errorf("Expected 3 valid platforms, got %d", len(platforms))
	}

	expected := []Platform{PlatformLinux, PlatformDarwin, PlatformWindows}
	for i, plat := range expected {
		if platforms[i] != plat {
			t.Errorf("Platform at index %d: got %v, want %v", i, platforms[i], plat)
		}
	}

	t.Log("✅ ValidPlatforms() working")
}

