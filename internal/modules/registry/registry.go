package registry

import (
	"fmt"
	"sync"

	"github.com/SiriusScan/app-agent/internal/modules"
)

// Global registry instance
var (
	globalRegistry = &Registry{
		modules:     make(map[string]*Entry),
		descriptors: make(map[string]*modules.Descriptor),
	}
)

// Entry represents a registered module with its metadata
type Entry struct {
	Module     modules.Module
	Descriptor *modules.Descriptor
}

// Registry manages the collection of available detection modules.
// It provides thread-safe registration and lookup of modules.
type Registry struct {
	mu          sync.RWMutex
	modules     map[string]*Entry
	descriptors map[string]*modules.Descriptor
}

// Register adds a module to the global registry.
// This is typically called from module init() functions.
// Returns an error if the module type is already registered or validation fails.
func Register(module modules.Module, descriptor modules.Descriptor) error {
	return globalRegistry.register(module, &descriptor)
}

// Get retrieves a module by its type from the global registry.
// Returns nil if the module is not found.
func Get(moduleType string) modules.Module {
	return globalRegistry.get(moduleType)
}

// GetDescriptor retrieves a module's descriptor by type from the global registry.
// Returns nil if the module is not found.
func GetDescriptor(moduleType string) *modules.Descriptor {
	return globalRegistry.getDescriptor(moduleType)
}

// List returns all registered module types from the global registry.
func List() []string {
	return globalRegistry.list()
}

// ListAll returns all registered modules with their descriptors.
func ListAll() map[string]*modules.Descriptor {
	return globalRegistry.listAll()
}

// register adds a module to this registry instance
func (r *Registry) register(module modules.Module, descriptor *modules.Descriptor) error {
	// Validate required fields
	if err := validateDescriptor(descriptor); err != nil {
		return fmt.Errorf("invalid descriptor: %w", err)
	}

	if module == nil {
		return fmt.Errorf("module cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already registered
	if _, exists := r.modules[descriptor.Type]; exists {
		return fmt.Errorf("module type %q is already registered", descriptor.Type)
	}

	// Register the module
	r.modules[descriptor.Type] = &Entry{
		Module:     module,
		Descriptor: descriptor,
	}
	r.descriptors[descriptor.Type] = descriptor

	return nil
}

// get retrieves a module by type (thread-safe read)
func (r *Registry) get(moduleType string) modules.Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if entry, exists := r.modules[moduleType]; exists {
		return entry.Module
	}
	return nil
}

// getDescriptor retrieves a module's descriptor (thread-safe read)
func (r *Registry) getDescriptor(moduleType string) *modules.Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.descriptors[moduleType]
}

// list returns all registered module types (thread-safe read)
func (r *Registry) list() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.modules))
	for moduleType := range r.modules {
		types = append(types, moduleType)
	}
	return types
}

// listAll returns all descriptors (thread-safe read)
func (r *Registry) listAll() map[string]*modules.Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*modules.Descriptor, len(r.descriptors))
	for k, v := range r.descriptors {
		// Create a copy of the descriptor
		descCopy := *v
		result[k] = &descCopy
	}
	return result
}

// validateDescriptor checks that required descriptor fields are present
func validateDescriptor(d *modules.Descriptor) error {
	if d.Type == "" {
		return fmt.Errorf("type is required")
	}
	if d.Name == "" {
		return fmt.Errorf("name is required")
	}
	if d.Description == "" {
		return fmt.Errorf("description is required")
	}
	if len(d.SupportedOS) == 0 {
		return fmt.Errorf("at least one supported OS is required")
	}

	// Validate OS values
	validOS := map[string]bool{
		"linux":   true,
		"darwin":  true,
		"windows": true,
	}
	for _, os := range d.SupportedOS {
		if !validOS[os] {
			return fmt.Errorf("invalid OS %q (must be linux, darwin, or windows)", os)
		}
	}

	return nil
}

