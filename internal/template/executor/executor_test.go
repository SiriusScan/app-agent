package executor

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/SiriusScan/app-agent/internal/modules"
	"github.com/SiriusScan/app-agent/internal/modules/registry"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

// Mock module for testing
type mockModule struct {
	shouldMatch bool
	shouldError bool
	evidence    map[string]interface{}
}

func (m *mockModule) Execute(ctx context.Context, config modules.StepConfig) (*modules.Result, error) {
	result := &modules.Result{
		Matched:  m.shouldMatch,
		Evidence: m.evidence,
	}

	if m.shouldError {
		result.Error = "mock module error"
	}

	return result, nil
}

// Register mock modules for testing
func init() {
	// Always matching module
	registry.Register(&mockModule{shouldMatch: true}, modules.Descriptor{
		Type:        "test_match",
		Name:        "Test Match Module",
		Description: "Always matches",
		SupportedOS: []string{"linux", "darwin", "windows"},
	})

	// Never matching module
	registry.Register(&mockModule{shouldMatch: false}, modules.Descriptor{
		Type:        "test_no_match",
		Name:        "Test No Match Module",
		Description: "Never matches",
		SupportedOS: []string{"linux", "darwin", "windows"},
	})

	// Error module
	registry.Register(&mockModule{shouldError: true}, modules.Descriptor{
		Type:        "test_error",
		Name:        "Test Error Module",
		Description: "Always errors",
		SupportedOS: []string{"linux", "darwin", "windows"},
	})
}

func TestExecutor_ExecuteTemplate(t *testing.T) {
	t.Log("\n🔍 Testing Executor.ExecuteTemplate()...")

	executor := New()
	ctx := context.Background()

	t.Run("single step match", func(t *testing.T) {
		t.Log("\n  Testing single step that matches...")

		template := &types.Template{
			ID: "test-001",
			Info: types.TemplateInfo{
				Name:     "Test Template",
				Severity: types.SeverityHigh,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{
						Type:   "test_match",
						Weight: 1.0,
						Config: map[string]interface{}{},
					},
				},
			},
		}

		result, err := executor.ExecuteTemplate(ctx, template)
		if err != nil {
			t.Fatalf("❌ ExecuteTemplate() returned error: %v", err)
		}

		if !result.Matched {
			t.Error("Expected matched=true")
		}

		if result.Confidence != 1.0 {
			t.Errorf("Expected confidence=1.0, got %f", result.Confidence)
		}

		if len(result.Steps) != 1 {
			t.Errorf("Expected 1 step result, got %d", len(result.Steps))
		}

		t.Log("  ✅ Single step match successful")
	})

	t.Run("single step no match", func(t *testing.T) {
		t.Log("\n  Testing single step that doesn't match...")

		template := &types.Template{
			ID: "test-002",
			Info: types.TemplateInfo{
				Name:     "Test Template",
				Severity: types.SeverityMedium,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{
						Type:   "test_no_match",
						Weight: 1.0,
						Config: map[string]interface{}{},
					},
				},
			},
		}

		result, err := executor.ExecuteTemplate(ctx, template)
		if err != nil {
			t.Fatalf("❌ ExecuteTemplate() returned error: %v", err)
		}

		if result.Matched {
			t.Error("Expected matched=false")
		}

		if result.Confidence != 0.0 {
			t.Errorf("Expected confidence=0.0, got %f", result.Confidence)
		}

		t.Log("  ✅ Single step no match successful")
	})

	t.Run("AND logic all match", func(t *testing.T) {
		t.Log("\n  Testing AND logic with all steps matching...")

		template := &types.Template{
			ID: "test-003",
			Info: types.TemplateInfo{
				Name:     "Test Template",
				Severity: types.SeverityCritical,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{Type: "test_match", Weight: 0.8, Config: map[string]interface{}{}},
					{Type: "test_match", Weight: 0.6, Config: map[string]interface{}{}},
				},
			},
		}

		result, err := executor.ExecuteTemplate(ctx, template)
		if err != nil {
			t.Fatalf("❌ ExecuteTemplate() returned error: %v", err)
		}

		if !result.Matched {
			t.Error("Expected matched=true for AND logic with all matching")
		}

		// AND logic: confidence should be minimum weight
		if result.Confidence != 0.6 {
			t.Errorf("Expected confidence=0.6 (min weight), got %f", result.Confidence)
		}

		if len(result.Steps) != 2 {
			t.Errorf("Expected 2 step results, got %d", len(result.Steps))
		}

		t.Log("  ✅ AND logic all match successful")
	})

	t.Run("AND logic one no match", func(t *testing.T) {
		t.Log("\n  Testing AND logic with one step not matching...")

		template := &types.Template{
			ID: "test-004",
			Info: types.TemplateInfo{
				Name:     "Test Template",
				Severity: types.SeverityLow,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{Type: "test_match", Weight: 1.0, Config: map[string]interface{}{}},
					{Type: "test_no_match", Weight: 1.0, Config: map[string]interface{}{}},
				},
			},
		}

		result, err := executor.ExecuteTemplate(ctx, template)
		if err != nil {
			t.Fatalf("❌ ExecuteTemplate() returned error: %v", err)
		}

		if result.Matched {
			t.Error("Expected matched=false for AND logic with one not matching")
		}

		t.Log("  ✅ AND logic one no match successful")
	})

	t.Run("OR logic one match", func(t *testing.T) {
		t.Log("\n  Testing OR logic with one step matching...")

		template := &types.Template{
			ID: "test-005",
			Info: types.TemplateInfo{
				Name:     "Test Template",
				Severity: types.SeverityHigh,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAny,
				Steps: []types.DetectionStep{
					{Type: "test_match", Weight: 0.7, Config: map[string]interface{}{}},
					{Type: "test_no_match", Weight: 0.9, Config: map[string]interface{}{}},
				},
			},
		}

		result, err := executor.ExecuteTemplate(ctx, template)
		if err != nil {
			t.Fatalf("❌ ExecuteTemplate() returned error: %v", err)
		}

		if !result.Matched {
			t.Error("Expected matched=true for OR logic with one matching")
		}

		// OR logic: confidence should be maximum matched weight
		if result.Confidence != 0.7 {
			t.Errorf("Expected confidence=0.7 (matched weight), got %f", result.Confidence)
		}

		t.Log("  ✅ OR logic one match successful")
	})

	t.Run("OR logic all match", func(t *testing.T) {
		t.Log("\n  Testing OR logic with all steps matching...")

		template := &types.Template{
			ID: "test-006",
			Info: types.TemplateInfo{
				Name:     "Test Template",
				Severity: types.SeverityHigh,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAny,
				Steps: []types.DetectionStep{
					{Type: "test_match", Weight: 0.7, Config: map[string]interface{}{}},
					{Type: "test_match", Weight: 0.9, Config: map[string]interface{}{}},
				},
			},
		}

		result, err := executor.ExecuteTemplate(ctx, template)
		if err != nil {
			t.Fatalf("❌ ExecuteTemplate() returned error: %v", err)
		}

		if !result.Matched {
			t.Error("Expected matched=true for OR logic with all matching")
		}

		// OR logic: confidence should be maximum weight
		if result.Confidence != 0.9 {
			t.Errorf("Expected confidence=0.9 (max weight), got %f", result.Confidence)
		}

		t.Log("  ✅ OR logic all match successful")
	})

	t.Run("step with error continues", func(t *testing.T) {
		t.Log("\n  Testing that errors don't stop execution...")

		template := &types.Template{
			ID: "test-007",
			Info: types.TemplateInfo{
				Name:     "Test Template",
				Severity: types.SeverityInfo,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{Type: "test_error", Weight: 1.0, Config: map[string]interface{}{}},
					{Type: "test_match", Weight: 1.0, Config: map[string]interface{}{}},
				},
			},
		}

		result, err := executor.ExecuteTemplate(ctx, template)
		if err != nil {
			t.Fatalf("❌ ExecuteTemplate() returned error: %v", err)
		}

		// Both steps should have run
		if len(result.Steps) != 2 {
			t.Errorf("Expected 2 step results, got %d", len(result.Steps))
		}

		// Should have errors collected
		if len(result.Errors) == 0 {
			t.Error("Expected errors to be collected")
		}

		t.Log("  ✅ Error handling successful")
	})

	t.Run("platform filtering", func(t *testing.T) {
		t.Log("\n  Testing platform filtering...")

		currentPlatform := types.Platform(runtime.GOOS)
		otherPlatform := types.PlatformLinux
		if currentPlatform == types.PlatformLinux {
			otherPlatform = types.PlatformDarwin
		}

		template := &types.Template{
			ID: "test-008",
			Info: types.TemplateInfo{
				Name:     "Test Template",
				Severity: types.SeverityMedium,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{
						Type:      "test_match",
						Platforms: []types.Platform{currentPlatform},
						Weight:    1.0,
						Config:    map[string]interface{}{},
					},
					{
						Type:      "test_match",
						Platforms: []types.Platform{otherPlatform},
						Weight:    1.0,
						Config:    map[string]interface{}{},
					},
				},
			},
		}

		result, err := executor.ExecuteTemplate(ctx, template)
		if err != nil {
			t.Fatalf("❌ ExecuteTemplate() returned error: %v", err)
		}

		// Only one step should have run (for current platform)
		if len(result.Steps) != 1 {
			t.Errorf("Expected 1 step result (platform filtered), got %d", len(result.Steps))
		}

		t.Log("  ✅ Platform filtering successful")
	})

	t.Run("nil template error", func(t *testing.T) {
		t.Log("\n  Testing nil template error...")

		_, err := executor.ExecuteTemplate(ctx, nil)
		if err == nil {
			t.Fatal("❌ Expected error for nil template, got nil")
		}

		if !strings.Contains(err.Error(), "cannot be nil") {
			t.Errorf("Expected error about nil template, got: %v", err)
		}

		t.Logf("  ✅ Nil template error: %v", err)
	})

	t.Run("module not found", func(t *testing.T) {
		t.Log("\n  Testing module not found...")

		template := &types.Template{
			ID: "test-009",
			Info: types.TemplateInfo{
				Name:     "Test Template",
				Severity: types.SeverityLow,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{
						Type:   "nonexistent_module",
						Weight: 1.0,
						Config: map[string]interface{}{},
					},
				},
			},
		}

		result, err := executor.ExecuteTemplate(ctx, template)
		if err != nil {
			t.Fatalf("❌ ExecuteTemplate() returned error: %v", err)
		}

		// Should not match
		if result.Matched {
			t.Error("Expected matched=false for missing module")
		}

		// Should have step error
		if len(result.Steps) != 1 {
			t.Errorf("Expected 1 step result, got %d", len(result.Steps))
		}

		if result.Steps[0].Error == "" {
			t.Error("Expected error in step result")
		}

		t.Logf("  ✅ Module not found error: %s", result.Steps[0].Error)
	})

	t.Run("context timeout", func(t *testing.T) {
		t.Log("\n  Testing context timeout...")

		// Create a very short timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // Ensure context is expired

		template := &types.Template{
			ID: "test-010",
			Info: types.TemplateInfo{
				Name:     "Test Template",
				Severity: types.SeverityHigh,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{Type: "test_match", Weight: 1.0, Config: map[string]interface{}{}},
				},
			},
		}

		_, err := executor.ExecuteTemplate(timeoutCtx, template)
		if err != nil {
			t.Fatalf("❌ ExecuteTemplate() returned error: %v", err)
		}

		// Execution should complete but might have timeout in evidence
		t.Log("  ✅ Context timeout handled gracefully")
	})

	t.Log("\n✅ Executor.ExecuteTemplate() tests completed")
}

func TestExecutor_CustomTimeout(t *testing.T) {
	t.Log("\n🔍 Testing Executor with custom timeout...")

	customTimeout := 1 * time.Second
	executor := NewWithTimeout(customTimeout)

	if executor.stepTimeout != customTimeout {
		t.Errorf("Expected step timeout %v, got %v", customTimeout, executor.stepTimeout)
	}

	t.Log("  ✅ Custom timeout set correctly")
}

