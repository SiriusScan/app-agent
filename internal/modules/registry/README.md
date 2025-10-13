# Module Registry

Thread-safe registry for detection modules.

## Purpose

- Register detection modules at startup
- Look up modules by type during template execution
- List all available modules
- Provide module metadata (descriptor information)

## Key Files

- `registry.go` - Registry implementation with sync.RWMutex
- `types.go` - Module interface and Descriptor struct (TODO: will be in parent modules/ directory)

