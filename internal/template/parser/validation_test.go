package parser

import (
	"strings"
	"testing"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

// validTemplate returns a fully valid template for testing
func validTemplate() *types.Template {
	return &types.Template{
		ID: "test-001",
		Info: types.TemplateInfo{
			Name:        "Test Template",
			Severity:    types.SeverityHigh,
			Description: "A test template",
		},
		Detection: types.DetectionConfig{
			Logic: types.LogicAll,
			Steps: []types.DetectionStep{
				{
					Type:      "file_hash",
					Platforms: []types.Platform{types.PlatformLinux},
					Weight:    0.8,
					Config: map[string]interface{}{
						"path": "/test/file",
					},
				},
			},
		},
	}
}

func TestValidateTemplate(t *testing.T) {
	t.Log("\n🔍 Testing ValidateTemplate()...")

	t.Run("valid template", func(t *testing.T) {
		t.Log("\n  Testing valid template...")

		template := validTemplate()
		err := ValidateTemplate(template)
		if err != nil {
			t.Fatalf("❌ Expected no error for valid template, got: %v", err)
		}

		t.Log("  ✅ Valid template passed validation")
	})

	t.Run("nil template", func(t *testing.T) {
		t.Log("\n  Testing nil template...")

		err := ValidateTemplate(nil)
		if err == nil {
			t.Fatal("❌ Expected error for nil template, got nil")
		}

		if !strings.Contains(err.Error(), "cannot be nil") {
			t.Errorf("Expected error about nil, got: %v", err)
		}

		t.Logf("  ✅ Nil template rejected with error: %v", err)
	})

	t.Run("missing ID", func(t *testing.T) {
		t.Log("\n  Testing missing ID...")

		template := validTemplate()
		template.ID = ""

		err := ValidateTemplate(template)
		if err == nil {
			t.Fatal("❌ Expected error for missing ID, got nil")
		}

		if !strings.Contains(err.Error(), "ID is required") {
			t.Errorf("Expected error about missing ID, got: %v", err)
		}

		t.Logf("  ✅ Missing ID rejected with error: %v", err)
	})

	t.Run("missing name", func(t *testing.T) {
		t.Log("\n  Testing missing info.name...")

		template := validTemplate()
		template.Info.Name = ""

		err := ValidateTemplate(template)
		if err == nil {
			t.Fatal("❌ Expected error for missing name, got nil")
		}

		if !strings.Contains(err.Error(), "info.name is required") {
			t.Errorf("Expected error about missing name, got: %v", err)
		}

		t.Logf("  ✅ Missing name rejected with error: %v", err)
	})

	t.Run("missing severity", func(t *testing.T) {
		t.Log("\n  Testing missing info.severity...")

		template := validTemplate()
		template.Info.Severity = ""

		err := ValidateTemplate(template)
		if err == nil {
			t.Fatal("❌ Expected error for missing severity, got nil")
		}

		if !strings.Contains(err.Error(), "info.severity is required") {
			t.Errorf("Expected error about missing severity, got: %v", err)
		}

		t.Logf("  ✅ Missing severity rejected with error: %v", err)
	})

	t.Run("invalid severity", func(t *testing.T) {
		t.Log("\n  Testing invalid severity...")

		template := validTemplate()
		template.Info.Severity = "super-critical"

		err := ValidateTemplate(template)
		if err == nil {
			t.Fatal("❌ Expected error for invalid severity, got nil")
		}

		if !strings.Contains(err.Error(), "info.severity") || !strings.Contains(err.Error(), "invalid") {
			t.Errorf("Expected error about invalid severity, got: %v", err)
		}

		t.Logf("  ✅ Invalid severity rejected with error: %v", err)
	})

	t.Run("no detection steps", func(t *testing.T) {
		t.Log("\n  Testing template with no detection steps...")

		template := validTemplate()
		template.Detection.Steps = []types.DetectionStep{}

		err := ValidateTemplate(template)
		if err == nil {
			t.Fatal("❌ Expected error for no detection steps, got nil")
		}

		if !strings.Contains(err.Error(), "detection.steps") {
			t.Errorf("Expected error about detection steps, got: %v", err)
		}

		t.Logf("  ✅ No detection steps rejected with error: %v", err)
	})

	t.Run("invalid detection logic", func(t *testing.T) {
		t.Log("\n  Testing invalid detection logic...")

		template := validTemplate()
		template.Detection.Logic = "some"

		err := ValidateTemplate(template)
		if err == nil {
			t.Fatal("❌ Expected error for invalid logic, got nil")
		}

		if !strings.Contains(err.Error(), "detection.logic") {
			t.Errorf("Expected error about invalid logic, got: %v", err)
		}

		t.Logf("  ✅ Invalid logic rejected with error: %v", err)
	})

	t.Run("multiple validation errors", func(t *testing.T) {
		t.Log("\n  Testing multiple validation errors...")

		template := validTemplate()
		template.ID = ""
		template.Info.Name = ""
		template.Info.Severity = "invalid"
		template.Detection.Steps = []types.DetectionStep{}

		err := ValidateTemplate(template)
		if err == nil {
			t.Fatal("❌ Expected error for multiple issues, got nil")
		}

		// Should contain all error messages
		errStr := err.Error()
		if !strings.Contains(errStr, "ID is required") {
			t.Error("Expected error to mention missing ID")
		}
		if !strings.Contains(errStr, "info.name is required") {
			t.Error("Expected error to mention missing name")
		}
		if !strings.Contains(errStr, "severity") && !strings.Contains(errStr, "invalid") {
			t.Error("Expected error to mention invalid severity")
		}
		if !strings.Contains(errStr, "detection.steps") {
			t.Error("Expected error to mention detection steps")
		}

		t.Logf("  ✅ Multiple errors reported:\n%v", err)
	})

	t.Log("\n✅ ValidateTemplate() tests completed")
}

func TestValidateDetectionSteps(t *testing.T) {
	t.Log("\n🔍 Testing detection step validation...")

	t.Run("valid steps", func(t *testing.T) {
		t.Log("\n  Testing valid detection steps...")

		template := validTemplate()
		err := ValidateTemplate(template)
		if err != nil {
			t.Fatalf("❌ Expected no error for valid steps, got: %v", err)
		}

		t.Log("  ✅ Valid steps passed validation")
	})

	t.Run("missing step type", func(t *testing.T) {
		t.Log("\n  Testing missing step type...")

		template := validTemplate()
		template.Detection.Steps[0].Type = ""

		err := ValidateTemplate(template)
		if err == nil {
			t.Fatal("❌ Expected error for missing step type, got nil")
		}

		if !strings.Contains(err.Error(), "type is required") {
			t.Errorf("Expected error about missing type, got: %v", err)
		}

		t.Logf("  ✅ Missing step type rejected with error: %v", err)
	})

	t.Run("invalid platform", func(t *testing.T) {
		t.Log("\n  Testing invalid platform...")

		template := validTemplate()
		template.Detection.Steps[0].Platforms = []types.Platform{"solaris"}

		err := ValidateTemplate(template)
		if err == nil {
			t.Fatal("❌ Expected error for invalid platform, got nil")
		}

		if !strings.Contains(err.Error(), "platforms") || !strings.Contains(err.Error(), "invalid") {
			t.Errorf("Expected error about invalid platform, got: %v", err)
		}

		t.Logf("  ✅ Invalid platform rejected with error: %v", err)
	})

	t.Run("invalid weight - negative", func(t *testing.T) {
		t.Log("\n  Testing negative weight...")

		template := validTemplate()
		template.Detection.Steps[0].Weight = -0.5

		err := ValidateTemplate(template)
		if err == nil {
			t.Fatal("❌ Expected error for negative weight, got nil")
		}

		if !strings.Contains(err.Error(), "weight") || !strings.Contains(err.Error(), "invalid") {
			t.Errorf("Expected error about invalid weight, got: %v", err)
		}

		t.Logf("  ✅ Negative weight rejected with error: %v", err)
	})

	t.Run("invalid weight - too high", func(t *testing.T) {
		t.Log("\n  Testing weight > 1.0...")

		template := validTemplate()
		template.Detection.Steps[0].Weight = 1.5

		err := ValidateTemplate(template)
		if err == nil {
			t.Fatal("❌ Expected error for weight > 1.0, got nil")
		}

		if !strings.Contains(err.Error(), "weight") || !strings.Contains(err.Error(), "invalid") {
			t.Errorf("Expected error about invalid weight, got: %v", err)
		}

		t.Logf("  ✅ Weight > 1.0 rejected with error: %v", err)
	})

	t.Run("valid boundary weights", func(t *testing.T) {
		t.Log("\n  Testing valid boundary weight values...")

		// Test 0.0
		template := validTemplate()
		template.Detection.Steps[0].Weight = 0.0
		if err := ValidateTemplate(template); err != nil {
			t.Errorf("❌ Weight 0.0 should be valid, got error: %v", err)
		}

		// Test 1.0
		template.Detection.Steps[0].Weight = 1.0
		if err := ValidateTemplate(template); err != nil {
			t.Errorf("❌ Weight 1.0 should be valid, got error: %v", err)
		}

		t.Log("  ✅ Boundary weights (0.0, 1.0) are valid")
	})

	t.Run("multiple steps with mixed validity", func(t *testing.T) {
		t.Log("\n  Testing multiple steps with various errors...")

		template := validTemplate()
		template.Detection.Steps = []types.DetectionStep{
			{
				Type:      "file_hash",
				Platforms: []types.Platform{types.PlatformLinux},
				Weight:    0.8,
				Config:    map[string]interface{}{"path": "/test"},
			},
			{
				Type:      "", // Missing type
				Platforms: []types.Platform{"invalid"}, // Invalid platform
				Weight:    2.0, // Invalid weight
				Config:    map[string]interface{}{},
			},
		}

		err := ValidateTemplate(template)
		if err == nil {
			t.Fatal("❌ Expected errors for invalid second step, got nil")
		}

		errStr := err.Error()
		if !strings.Contains(errStr, "steps[1]") {
			t.Error("Expected error to reference steps[1]")
		}
		if !strings.Contains(errStr, "type is required") {
			t.Error("Expected error about missing type")
		}
		if !strings.Contains(errStr, "platform") {
			t.Error("Expected error about invalid platform")
		}
		if !strings.Contains(errStr, "weight") {
			t.Error("Expected error about invalid weight")
		}

		t.Logf("  ✅ Multiple step errors reported:\n%v", err)
	})

	t.Log("\n✅ Detection step validation tests completed")
}

func TestValidSeverityLevels(t *testing.T) {
	t.Log("\n🔍 Testing all valid severity levels...")

	severities := []types.Severity{
		types.SeverityCritical,
		types.SeverityHigh,
		types.SeverityMedium,
		types.SeverityLow,
		types.SeverityInfo,
	}

	for _, severity := range severities {
		t.Run(string(severity), func(t *testing.T) {
			template := validTemplate()
			template.Info.Severity = severity

			err := ValidateTemplate(template)
			if err != nil {
				t.Errorf("❌ Severity '%s' should be valid, got error: %v", severity, err)
			}
		})
	}

	t.Log("  ✅ All severity levels validated correctly")
}

func TestValidPlatforms(t *testing.T) {
	t.Log("\n🔍 Testing all valid platforms...")

	platforms := []types.Platform{
		types.PlatformLinux,
		types.PlatformDarwin,
		types.PlatformWindows,
	}

	for _, platform := range platforms {
		t.Run(string(platform), func(t *testing.T) {
			template := validTemplate()
			template.Detection.Steps[0].Platforms = []types.Platform{platform}

			err := ValidateTemplate(template)
			if err != nil {
				t.Errorf("❌ Platform '%s' should be valid, got error: %v", platform, err)
			}
		})
	}

	t.Log("  ✅ All platforms validated correctly")
}

func TestValidDetectionLogic(t *testing.T) {
	t.Log("\n🔍 Testing valid detection logic values...")

	logics := []types.DetectionLogic{
		types.LogicAll,
		types.LogicAny,
	}

	for _, logic := range logics {
		t.Run(string(logic), func(t *testing.T) {
			template := validTemplate()
			template.Detection.Logic = logic

			err := ValidateTemplate(template)
			if err != nil {
				t.Errorf("❌ Logic '%s' should be valid, got error: %v", logic, err)
			}
		})
	}

	t.Log("  ✅ All detection logic values validated correctly")
}

