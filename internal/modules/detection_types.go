package modules

import "sort"

// KnownDetectionTypes enumerates every detection type understood by an
// executor module. It is the single source of truth for template
// validation across the project.
//
// Two consumers depend on this set:
//
//  1. The agent runtime, which executes templates by looking up a Module
//     in the registry by Type. Modules self-register in their package
//     init() with the same string used here, so the agent's behavior is
//     determined by which module packages are blank-imported.
//
//  2. The engine-side template ingestion path
//     (internal/template/valkey/storage.go), which validates incoming
//     templates before persisting them to ValKey. The engine binary does
//     not import the module packages (it never executes templates), so
//     it cannot consult the registry directly and instead checks against
//     this declarative set.
//
// Adding a new detection module therefore requires two edits in this
// package: the module's own init() with its Descriptor, and an entry
// here. The drift between the two is asserted by
// detection_types_registry_test.go, which blank-imports every module
// and fails the build if either side is missing an entry.
var KnownDetectionTypes = map[string]struct{}{
	"file_hash":    {},
	"file_content": {},
	"file_search":  {},
	"version_cmd":  {},
}

// IsKnownDetectionType reports whether t names a detection type that
// has a corresponding executor module.
func IsKnownDetectionType(t string) bool {
	_, ok := KnownDetectionTypes[t]
	return ok
}

// KnownDetectionTypeNames returns the known detection type names in
// deterministic (sorted) order. Useful for error messages and docs.
func KnownDetectionTypeNames() []string {
	out := make([]string, 0, len(KnownDetectionTypes))
	for t := range KnownDetectionTypes {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
