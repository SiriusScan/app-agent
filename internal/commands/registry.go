package commands

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

// ErrUnknownCommand is returned when Dispatch cannot find a matching command prefix.
var ErrUnknownCommand = errors.New("unknown internal command")

type Registry struct {
	mu              sync.RWMutex
	commands        map[string]Command
	aliases         map[string]string
	orderedPrefixes []string
}

func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command),
		aliases:  make(map[string]string),
	}
}

var defaultRegistry = NewRegistry()

func DefaultRegistry() *Registry {
	return defaultRegistry
}

func ResetDefaultRegistryForTests() {
	defaultRegistry = NewRegistry()
}

// Register adds a command to the default registry with its associated prefix.
func Register(prefix string, cmd Command) {
	defaultRegistry.Register(prefix, cmd)
}

// RegisterAlias adds an alias to the default registry.
func RegisterAlias(alias, canonicalPrefix string) {
	defaultRegistry.RegisterAlias(alias, canonicalPrefix)
}

// Get retrieves a command by its exact prefix from the default registry.
func Get(prefix string) (Command, bool) {
	return defaultRegistry.Get(prefix)
}

// Dispatch executes a command using the default registry.
func Dispatch(ctx context.Context, agentInfo AgentInfo, commandString string) (string, error) {
	return defaultRegistry.Dispatch(ctx, agentInfo, commandString)
}

// ListCommands lists all commands registered in the default registry.
func ListCommands() map[string][]string {
	return defaultRegistry.ListCommands()
}

func (r *Registry) Register(prefix string, cmd Command) {
	if prefix == "" {
		panic("commands: Register called with empty prefix")
	}
	if cmd == nil {
		panic("commands: Register command cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, dup := r.commands[prefix]; dup {
		panic("commands: Register called twice for prefix " + prefix)
	}
	r.commands[prefix] = cmd
	r.orderedPrefixes = append(r.orderedPrefixes, prefix)
	r.sortPrefixesLocked()
}

func (r *Registry) RegisterAlias(alias, canonicalPrefix string) {
	if alias == "" {
		panic("commands: RegisterAlias called with empty alias")
	}
	if canonicalPrefix == "" {
		panic("commands: RegisterAlias called with empty canonical prefix")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.commands[canonicalPrefix]; !exists {
		panic("commands: RegisterAlias called for non-existent prefix " + canonicalPrefix)
	}
	if _, dup := r.aliases[alias]; dup {
		panic("commands: RegisterAlias called twice for alias " + alias)
	}
	if _, dup := r.commands[alias]; dup {
		panic("commands: RegisterAlias conflicts with existing prefix " + alias)
	}

	r.aliases[alias] = canonicalPrefix
	r.orderedPrefixes = append(r.orderedPrefixes, alias)
	r.sortPrefixesLocked()
}

func (r *Registry) Get(prefix string) (Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, found := r.commands[prefix]
	return cmd, found
}

func (r *Registry) Dispatch(ctx context.Context, agentInfo AgentInfo, commandString string) (string, error) {
	r.mu.RLock()
	prefixes := append([]string(nil), r.orderedPrefixes...)
	aliases := make(map[string]string, len(r.aliases))
	for alias, canonical := range r.aliases {
		aliases[alias] = canonical
	}
	r.mu.RUnlock()

	for _, prefix := range prefixes {
		if strings.HasPrefix(commandString, prefix) {
			canonicalPrefix := prefix
			if aliasTarget, isAlias := aliases[prefix]; isAlias {
				canonicalPrefix = aliasTarget
			}

			cmd, found := r.Get(canonicalPrefix)
			if !found {
				continue
			}

			args := strings.TrimSpace(commandString[len(prefix):])
			return cmd.Execute(ctx, agentInfo, commandString, args)
		}
	}

	return "", ErrUnknownCommand
}

func (r *Registry) ListCommands() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]string)
	for prefix := range r.commands {
		result[prefix] = []string{}
	}
	for alias, canonical := range r.aliases {
		if aliasesSlice, exists := result[canonical]; exists {
			result[canonical] = append(aliasesSlice, alias)
		}
	}
	return result
}

func (r *Registry) HasAlias(alias string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.aliases[alias]
	return exists
}

func (r *Registry) HasCommand(prefix string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.commands[prefix]
	return exists
}

func (r *Registry) sortPrefixesLocked() {
	sort.Slice(r.orderedPrefixes, func(i, j int) bool {
		return len(r.orderedPrefixes[i]) > len(r.orderedPrefixes[j])
	})
}
