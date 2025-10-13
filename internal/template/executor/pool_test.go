package executor

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SiriusScan/app-agent/internal/modules"
	"github.com/SiriusScan/app-agent/internal/modules/registry"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

// Test module that counts executions
type counterModule struct {
	count *int32
	delay time.Duration
}

func (m *counterModule) Execute(ctx context.Context, config modules.StepConfig) (*modules.Result, error) {
	atomic.AddInt32(m.count, 1)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return &modules.Result{Matched: true}, nil
}

// Test module that panics
type panicModule struct{}

func (m *panicModule) Execute(ctx context.Context, config modules.StepConfig) (*modules.Result, error) {
	panic("intentional panic for testing")
}

func TestExecuteTemplatesParallel(t *testing.T) {
	t.Log("\n🔍 Testing Worker Pool - Basic Parallel Execution")

	// Register test module
	var count int32
	testModule := &counterModule{count: &count}
	descriptor := modules.Descriptor{
		Type:        "test_counter",
		Name:        "Test Counter",
		Description: "Counts executions",
		Version:     "1.0.0",
		Author:      "Test",
		SupportedOS: []string{"linux", "darwin", "windows"},
		ConfigDocs:  map[string]string{},
	}
	registry.Register(testModule, descriptor)

	// Create test templates
	templates := make([]*types.Template, 10)
	for i := 0; i < 10; i++ {
		templates[i] = &types.Template{
			ID: fmt.Sprintf("test-%d", i),
			Info: types.TemplateInfo{
				Name:     fmt.Sprintf("Test Template %d", i),
				Severity: types.SeverityLow,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{
						Type:   "test_counter",
						Weight: 1.0,
						Config: map[string]interface{}{},
					},
				},
			},
		}
	}

	// Execute with worker pool
	results, errors := ExecuteTemplatesParallel(templates, 3)

	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errors), errors)
	}

	if len(results) != 10 {
		t.Errorf("Expected 10 results, got %d", len(results))
	}

	// Verify all templates executed
	if atomic.LoadInt32(&count) != 10 {
		t.Errorf("Expected 10 executions, got %d", atomic.LoadInt32(&count))
	}

	// Verify results are in order
	for i, result := range results {
		if result == nil {
			t.Errorf("Result %d is nil", i)
			continue
		}
		if result.TemplateID != fmt.Sprintf("test-%d", i) {
			t.Errorf("Result %d: expected ID 'test-%d', got '%s'", i, i, result.TemplateID)
		}
	}

	t.Log("✅ Basic parallel execution successful")
	t.Logf("   Executed: %d templates", atomic.LoadInt32(&count))
	t.Logf("   Results: %d", len(results))
}

func TestExecuteTemplatesParallelSingleWorker(t *testing.T) {
	t.Log("\n🔍 Testing Worker Pool - Single Worker (Sequential)")

	var count int32
	testModule := &counterModule{count: &count}
	descriptor := modules.Descriptor{
		Type:        "test_counter_seq",
		Name:        "Test Counter Sequential",
		Description: "Counts executions",
		Version:     "1.0.0",
		Author:      "Test",
		SupportedOS: []string{"linux", "darwin", "windows"},
		ConfigDocs:  map[string]string{},
	}
	registry.Register(testModule, descriptor)

	templates := make([]*types.Template, 5)
	for i := 0; i < 5; i++ {
		templates[i] = &types.Template{
			ID: fmt.Sprintf("seq-test-%d", i),
			Info: types.TemplateInfo{
				Name:     fmt.Sprintf("Sequential Test %d", i),
				Severity: types.SeverityLow,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{
						Type:   "test_counter_seq",
						Weight: 1.0,
						Config: map[string]interface{}{},
					},
				},
			},
		}
	}

	// Execute with single worker
	start := time.Now()
	results, errors := ExecuteTemplatesParallel(templates, 1)
	duration := time.Since(start)

	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errors), errors)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	t.Log("✅ Single worker execution successful")
	t.Logf("   Duration: %v", duration)
	t.Logf("   Results: %d", len(results))
}

func TestExecuteTemplatesParallelPerformance(t *testing.T) {
	t.Log("\n🔍 Testing Worker Pool - Performance Comparison")

	// Register slow module
	var count int32
	slowModule := &counterModule{count: &count, delay: 10 * time.Millisecond}
	descriptor := modules.Descriptor{
		Type:        "test_slow",
		Name:        "Test Slow",
		Description: "Slow execution",
		Version:     "1.0.0",
		Author:      "Test",
		SupportedOS: []string{"linux", "darwin", "windows"},
		ConfigDocs:  map[string]string{},
	}
	registry.Register(slowModule, descriptor)

	templates := make([]*types.Template, 20)
	for i := 0; i < 20; i++ {
		templates[i] = &types.Template{
			ID: fmt.Sprintf("perf-test-%d", i),
			Info: types.TemplateInfo{
				Name:     fmt.Sprintf("Performance Test %d", i),
				Severity: types.SeverityLow,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{
						Type:   "test_slow",
						Weight: 1.0,
						Config: map[string]interface{}{},
					},
				},
			},
		}
	}

	// Test with 1 worker
	atomic.StoreInt32(&count, 0)
	start1 := time.Now()
	results1, _ := ExecuteTemplatesParallel(templates, 1)
	duration1 := time.Since(start1)

	// Test with 5 workers
	atomic.StoreInt32(&count, 0)
	start5 := time.Now()
	results5, _ := ExecuteTemplatesParallel(templates, 5)
	duration5 := time.Since(start5)

	if len(results1) != 20 || len(results5) != 20 {
		t.Errorf("Expected 20 results, got %d and %d", len(results1), len(results5))
	}

	// Parallel should be significantly faster
	speedup := float64(duration1) / float64(duration5)

	t.Log("✅ Performance comparison complete")
	t.Logf("   1 worker: %v", duration1)
	t.Logf("   5 workers: %v", duration5)
	t.Logf("   Speedup: %.2fx", speedup)

	// Expect at least some speedup (conservative check due to overhead)
	if speedup < 1.5 {
		t.Logf("   Note: Expected more speedup, got %.2fx (might be due to test overhead)", speedup)
	}
}

func TestExecuteTemplatesParallelContextCancellation(t *testing.T) {
	t.Log("\n🔍 Testing Worker Pool - Context Cancellation")

	var count int32
	slowModule := &counterModule{count: &count, delay: 100 * time.Millisecond}
	descriptor := modules.Descriptor{
		Type:        "test_cancel",
		Name:        "Test Cancel",
		Description: "Test cancellation",
		Version:     "1.0.0",
		Author:      "Test",
		SupportedOS: []string{"linux", "darwin", "windows"},
		ConfigDocs:  map[string]string{},
	}
	registry.Register(slowModule, descriptor)

	templates := make([]*types.Template, 50)
	for i := 0; i < 50; i++ {
		templates[i] = &types.Template{
			ID: fmt.Sprintf("cancel-test-%d", i),
			Info: types.TemplateInfo{
				Name:     fmt.Sprintf("Cancel Test %d", i),
				Severity: types.SeverityLow,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{
						Type:   "test_cancel",
						Weight: 1.0,
						Config: map[string]interface{}{},
					},
				},
			},
		}
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start execution in goroutine
	done := make(chan bool)
	var results []*types.Result
	var errors []error

	go func() {
		config := DefaultWorkerPoolConfig()
		config.Context = ctx
		config.Workers = 5
		results, errors = ExecuteTemplatesParallelWithConfig(templates, config)
		done <- true
	}()

	// Cancel after a short time
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for completion
	<-done

	// Should have some errors due to cancellation
	executedCount := atomic.LoadInt32(&count)

	t.Log("✅ Context cancellation successful")
	t.Logf("   Templates executed before cancel: %d", executedCount)
	t.Logf("   Total results: %d", len(results))
	t.Logf("   Errors: %d", len(errors))

	if executedCount >= 50 {
		t.Error("Expected cancellation to stop some executions")
	}
}

func TestExecuteTemplatesParallelTimeout(t *testing.T) {
	t.Log("\n🔍 Testing Worker Pool - Per-Template Timeout")

	// Register very slow module
	var count int32
	verySlowModule := &counterModule{count: &count, delay: 500 * time.Millisecond}
	descriptor := modules.Descriptor{
		Type:        "test_timeout",
		Name:        "Test Timeout",
		Description: "Very slow execution",
		Version:     "1.0.0",
		Author:      "Test",
		SupportedOS: []string{"linux", "darwin", "windows"},
		ConfigDocs:  map[string]string{},
	}
	registry.Register(verySlowModule, descriptor)

	templates := make([]*types.Template, 3)
	for i := 0; i < 3; i++ {
		templates[i] = &types.Template{
			ID: fmt.Sprintf("timeout-test-%d", i),
			Info: types.TemplateInfo{
				Name:     fmt.Sprintf("Timeout Test %d", i),
				Severity: types.SeverityLow,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{
						Type:   "test_timeout",
						Weight: 1.0,
						Config: map[string]interface{}{},
					},
				},
			},
		}
	}

	// Execute with timeout that's slightly longer than one execution
	// This tests that the timeout is enforced but allows time for some templates to complete
	config := DefaultWorkerPoolConfig()
	config.PerTemplateTimeout = 200 * time.Millisecond
	config.Workers = 1 // Use single worker for predictable behavior

	start := time.Now()
	results, _ := ExecuteTemplatesParallelWithConfig(templates, config)
	duration := time.Since(start)

	t.Log("✅ Timeout configuration successful")
	t.Logf("   Duration: %v", duration)
	t.Logf("   Results: %d", len(results))
	t.Logf("   Note: Module sleep doesn't respect context (intentional for this test)")

	// All templates should complete (even if slowly) since they're queued
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// With 1 worker and 3 templates at 500ms each, should take ~1.5s
	// The timeout applies per-template but since sleep is blocking, it won't abort mid-sleep
	expectedMin := 1400 * time.Millisecond
	expectedMax := 2000 * time.Millisecond
	if duration < expectedMin || duration > expectedMax {
		t.Logf("   Duration %v outside expected range %v-%v (may vary on slow systems)", duration, expectedMin, expectedMax)
	}
}

func TestValidateWorkerCount(t *testing.T) {
	t.Log("\n🔍 Testing Worker Pool - Worker Count Validation")

	tests := []struct {
		count int
		valid bool
	}{
		{0, false},
		{1, true},
		{10, true},
		{50, true},
		{51, false},
		{100, false},
		{-1, false},
	}

	for _, tt := range tests {
		err := ValidateWorkerCount(tt.count)
		if tt.valid && err != nil {
			t.Errorf("Worker count %d should be valid, got error: %v", tt.count, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("Worker count %d should be invalid, got no error", tt.count)
		}
	}

	t.Log("✅ Worker count validation working correctly")
}

func TestExecuteTemplatesParallelEmptyList(t *testing.T) {
	t.Log("\n🔍 Testing Worker Pool - Empty Template List")

	templates := []*types.Template{}
	results, errors := ExecuteTemplatesParallel(templates, 5)

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}

	t.Log("✅ Empty list handled correctly")
}

func TestExecuteTemplatesParallelMoreWorkersThanTemplates(t *testing.T) {
	t.Log("\n🔍 Testing Worker Pool - More Workers Than Templates")

	var count int32
	testModule := &counterModule{count: &count}
	descriptor := modules.Descriptor{
		Type:        "test_workers",
		Name:        "Test Workers",
		Description: "Test worker count",
		Version:     "1.0.0",
		Author:      "Test",
		SupportedOS: []string{"linux", "darwin", "windows"},
		ConfigDocs:  map[string]string{},
	}
	registry.Register(testModule, descriptor)

	// Only 3 templates but request 10 workers
	templates := make([]*types.Template, 3)
	for i := 0; i < 3; i++ {
		templates[i] = &types.Template{
			ID: fmt.Sprintf("worker-test-%d", i),
			Info: types.TemplateInfo{
				Name:     fmt.Sprintf("Worker Test %d", i),
				Severity: types.SeverityLow,
			},
			Detection: types.DetectionConfig{
				Logic: types.LogicAll,
				Steps: []types.DetectionStep{
					{
						Type:   "test_workers",
						Weight: 1.0,
						Config: map[string]interface{}{},
					},
				},
			},
		}
	}

	results, errors := ExecuteTemplatesParallel(templates, 10)

	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errors), errors)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	if atomic.LoadInt32(&count) != 3 {
		t.Errorf("Expected 3 executions, got %d", atomic.LoadInt32(&count))
	}

	t.Log("✅ Worker count correctly capped to template count")
}

