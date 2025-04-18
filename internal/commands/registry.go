package commands

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	registry        = make(map[string]Command)
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
		panic(fmt.Sprintf("commands: Register called twice for prefix %q", prefix))
	}
	registry[prefix] = cmd

	// Keep prefixes sorted by length descending for longest match logic
	orderedPrefixes = append(orderedPrefixes, prefix)
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
// If found, it extracts arguments and calls the command's Execute method.
// If no command matches, it returns ErrUnknownCommand.
func Dispatch(ctx context.Context, agentInfo AgentInfo, commandString string) (output string, err error) {
	registryMu.RLock()
	prefixes := orderedPrefixes // Use the sorted list
	registryMu.RUnlock()

	for _, prefix := range prefixes {
		if strings.HasPrefix(commandString, prefix) {
			cmd, found := Get(prefix) // Get the command associated with this prefix
			if !found {
				// Should not happen if registration logic is correct, but handle defensively
				continue
			}

			// Extract arguments (part after prefix, trimmed)
			args := strings.TrimSpace(commandString[len(prefix):])

			// Execute the command
			return cmd.Execute(ctx, agentInfo, commandString, args)
		}
	}

	// No matching prefix found
	return "", ErrUnknownCommand
}
