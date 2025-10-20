package commands

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

var (
	registry        = make(map[string]Command)
	aliases         = make(map[string]string) // Maps alias -> canonical prefix
	registryMu      sync.RWMutex
	orderedPrefixes []string // Keep prefixes sorted for longest match
)

// ErrUnknownCommand is returned when Dispatch cannot find a matching command prefix.
var ErrUnknownCommand = errors.New("unknown internal command")

// Register adds a command to the registry with its associated prefix.
// It panics if the prefix is empty or already registered.
func Register(prefix string, cmd Command) {
	if prefix == "" {
		panic("commands: Register called with empty prefix")
	}
	if cmd == nil {
		panic("commands: Register command cannot be nil")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, dup := registry[prefix]; dup {
		panic("commands: Register called twice for prefix " + prefix)
	}
	registry[prefix] = cmd

	// Keep prefixes sorted by length descending for longest match logic
	orderedPrefixes = append(orderedPrefixes, prefix)
	sort.Slice(orderedPrefixes, func(i, j int) bool {
		return len(orderedPrefixes[i]) > len(orderedPrefixes[j])
	})
}

// RegisterAlias adds an alias for an existing command prefix.
// The alias will be resolved to the canonical prefix during dispatch.
// It panics if the alias is empty or already registered, or if the target prefix doesn't exist.
func RegisterAlias(alias, canonicalPrefix string) {
	if alias == "" {
		panic("commands: RegisterAlias called with empty alias")
	}
	if canonicalPrefix == "" {
		panic("commands: RegisterAlias called with empty canonical prefix")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	// Verify the canonical prefix exists
	if _, exists := registry[canonicalPrefix]; !exists {
		panic("commands: RegisterAlias called for non-existent prefix " + canonicalPrefix)
	}

	// Check for duplicate alias
	if _, dup := aliases[alias]; dup {
		panic("commands: RegisterAlias called twice for alias " + alias)
	}

	// Also check that the alias doesn't conflict with existing prefixes
	if _, dup := registry[alias]; dup {
		panic("commands: RegisterAlias conflicts with existing prefix " + alias)
	}

	aliases[alias] = canonicalPrefix

	// Add alias to ordered prefixes for matching
	orderedPrefixes = append(orderedPrefixes, alias)
	sort.Slice(orderedPrefixes, func(i, j int) bool {
		return len(orderedPrefixes[i]) > len(orderedPrefixes[j])
	})
}

// Get retrieves a command by its exact prefix.
func Get(prefix string) (Command, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	cmd, found := registry[prefix]
	return cmd, found
}

// Dispatch finds and executes the command corresponding to the commandString.
// It uses longest prefix matching to find the command in the registry.
// Aliases are automatically resolved to their canonical command prefixes.
// If found, it extracts arguments and calls the command's Execute method.
// If no command matches, it returns ErrUnknownCommand.
func Dispatch(ctx context.Context, agentInfo AgentInfo, commandString string) (output string, err error) {
	registryMu.RLock()
	prefixes := orderedPrefixes // Use the sorted list
	registryMu.RUnlock()

	for _, prefix := range prefixes {
		if strings.HasPrefix(commandString, prefix) {
			// Check if this is an alias and resolve it
			canonicalPrefix := prefix
			if aliasTarget, isAlias := aliases[prefix]; isAlias {
				canonicalPrefix = aliasTarget
			}

			cmd, found := Get(canonicalPrefix) // Get the command associated with the canonical prefix
			if !found {
				// Should not happen if registration logic is correct, but handle defensively
				continue
			}

			// Extract arguments (part after the matched prefix, trimmed)
			args := strings.TrimSpace(commandString[len(prefix):])

			// Execute the command
			return cmd.Execute(ctx, agentInfo, commandString, args)
		}
	}

	// No matching prefix found
	return "", ErrUnknownCommand
}

// ListCommands returns a list of all registered commands and their aliases.
func ListCommands() map[string][]string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make(map[string][]string)

	// Add all canonical commands
	for prefix := range registry {
		result[prefix] = []string{}
	}

	// Add aliases to their canonical commands
	for alias, canonical := range aliases {
		if aliasesSlice, exists := result[canonical]; exists {
			result[canonical] = append(aliasesSlice, alias)
		}
	}

	return result
}
