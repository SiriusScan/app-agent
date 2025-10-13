package registry

import (
	"context"
	"sync"
	"testing"

	"github.com/SiriusScan/app-agent/internal/modules"
)

// DummyModule is a test module implementation
type DummyModule struct {
	name string
}

func (d *DummyModule) Execute(ctx context.Context, config modules.StepConfig) (*modules.Result, error) {
	return &modules.Result{
		Matched: true,
		Evidence: map[string]interface{}{
			"module": d.name,
		},
	}, nil
}

func TestRegister(t *testing.T) {
	// Create a fresh registry for testing
	reg := &Registry{
		modules:     make(map[string]*Entry),
		descriptors: make(map[string]*modules.Descriptor),
	}

	module := &DummyModule{name: "test"}
	descriptor := modules.Descriptor{
		Type:        "test_module",
		Name:        "Test Module",
		Description: "A test module",
		Version:     "1.0.0",
		Author:      "Test Author",
		SupportedOS: []string{"linux", "darwin"},
		ConfigDocs: map[string]string{
			"field1": "Test field 1",
		},
	}

	// Test successful registration
	err := reg.register(module, &descriptor)
	if err != nil {
		t.Fatalf("Failed to register module: %v", err)
	}

	// Test duplicate registration
	err = reg.register(module, &descriptor)
	if err == nil {
		t.Fatal("Expected error when registering duplicate module, got nil")
	}

	t.Log("✅ Module registration and duplicate detection working")
}

func TestGet(t *testing.T) {
	// Create a fresh registry
	reg := &Registry{
		modules:     make(map[string]*Entry),
		descriptors: make(map[string]*modules.Descriptor),
	}

	module := &DummyModule{name: "test"}
	descriptor := modules.Descriptor{
		Type:        "test_module",
		Name:        "Test Module",
		Description: "A test module",
		SupportedOS: []string{"linux"},
	}

	reg.register(module, &descriptor)

	// Test retrieval
	retrieved := reg.get("test_module")
	if retrieved == nil {
		t.Fatal("Failed to retrieve registered module")
	}

	// Test non-existent module
	notFound := reg.get("non_existent")
	if notFound != nil {
		t.Fatal("Expected nil for non-existent module")
	}

	t.Log("✅ Module retrieval working")
}

func TestList(t *testing.T) {
	// Create a fresh registry
	reg := &Registry{
		modules:     make(map[string]*Entry),
		descriptors: make(map[string]*modules.Descriptor),
	}

	// Register multiple modules
	moduleNames := []string{"module1", "module2", "module3"}
	for _, name := range moduleNames {
		m := &DummyModule{name: name}
		d := modules.Descriptor{
			Type:        name,
			Name:        name,
			Description: "Test module " + name,
			SupportedOS: []string{"linux"},
		}
		reg.register(m, &d)
	}

	// Test list
	list := reg.list()
	if len(list) != 3 {
		t.Fatalf("Expected 3 modules, got %d", len(list))
	}

	// Verify all modules are in the list
	found := make(map[string]bool)
	for _, name := range list {
		found[name] = true
	}

	for _, expected := range moduleNames {
		if !found[expected] {
			t.Fatalf("Expected module %q not found in list", expected)
		}
	}

	t.Log("✅ Module listing working")
}

func TestValidation(t *testing.T) {
	// Create a fresh registry
	reg := &Registry{
		modules:     make(map[string]*Entry),
		descriptors: make(map[string]*modules.Descriptor),
	}

	module := &DummyModule{name: "test"}

	tests := []struct {
		name       string
		descriptor modules.Descriptor
		expectErr  bool
	}{
		{
			name: "missing type",
			descriptor: modules.Descriptor{
				Name:        "Test",
				Description: "Test",
				SupportedOS: []string{"linux"},
			},
			expectErr: true,
		},
		{
			name: "missing name",
			descriptor: modules.Descriptor{
				Type:        "test",
				Description: "Test",
				SupportedOS: []string{"linux"},
			},
			expectErr: true,
		},
		{
			name: "missing description",
			descriptor: modules.Descriptor{
				Type:        "test",
				Name:        "Test",
				SupportedOS: []string{"linux"},
			},
			expectErr: true,
		},
		{
			name: "missing supported OS",
			descriptor: modules.Descriptor{
				Type:        "test",
				Name:        "Test",
				Description: "Test",
				SupportedOS: []string{},
			},
			expectErr: true,
		},
		{
			name: "invalid OS",
			descriptor: modules.Descriptor{
				Type:        "test",
				Name:        "Test",
				Description: "Test",
				SupportedOS: []string{"invalid_os"},
			},
			expectErr: true,
		},
		{
			name: "valid descriptor",
			descriptor: modules.Descriptor{
				Type:        "test_valid",
				Name:        "Test Valid",
				Description: "Test valid descriptor",
				SupportedOS: []string{"linux", "darwin", "windows"},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reg.register(module, &tt.descriptor)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}

	t.Log("✅ Descriptor validation working")
}

func TestConcurrency(t *testing.T) {
	// Create a fresh registry
	reg := &Registry{
		modules:     make(map[string]*Entry),
		descriptors: make(map[string]*modules.Descriptor),
	}

	// Register some initial modules
	for i := 0; i < 5; i++ {
		m := &DummyModule{name: string(rune('A' + i))}
		d := modules.Descriptor{
			Type:        string(rune('A' + i)),
			Name:        string(rune('A' + i)),
			Description: "Test module",
			SupportedOS: []string{"linux"},
		}
		reg.register(m, &d)
	}

	// Concurrent reads
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = reg.list()
			_ = reg.get("A")
			_ = reg.getDescriptor("B")
		}()
	}

	wg.Wait()

	t.Log("✅ Concurrent access working (run with -race to verify)")
}

func TestStepConfig(t *testing.T) {
	config := modules.StepConfig{
		"string_field": "test_value",
		"bool_field":   true,
		"int_field":    42,
		"float_field":  3.14,
		"slice_field":  []interface{}{"a", "b", "c"},
	}

	// Test GetString
	if val := config.GetString("string_field"); val != "test_value" {
		t.Errorf("Expected 'test_value', got %q", val)
	}
	if val := config.GetString("missing"); val != "" {
		t.Errorf("Expected empty string for missing field, got %q", val)
	}

	// Test GetBool
	if val := config.GetBool("bool_field"); val != true {
		t.Errorf("Expected true, got %v", val)
	}
	if val := config.GetBool("missing"); val != false {
		t.Errorf("Expected false for missing field, got %v", val)
	}

	// Test GetInt
	if val := config.GetInt("int_field"); val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}

	// Test GetFloat
	if val := config.GetFloat("float_field"); val != 3.14 {
		t.Errorf("Expected 3.14, got %f", val)
	}

	// Test GetStringSlice
	slice := config.GetStringSlice("slice_field")
	if len(slice) != 3 {
		t.Errorf("Expected slice length 3, got %d", len(slice))
	}

	t.Log("✅ StepConfig helper methods working")
}

