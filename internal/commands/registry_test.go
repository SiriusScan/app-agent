package commands

import (
	"context"
	"testing"
)

// MockCommand is a test implementation of the Command interface
type MockCommand struct {
	ExecuteCalled bool
	ReturnOutput  string
	ReturnError   error
}

func (m *MockCommand) Execute(ctx context.Context, agentInfo AgentInfo, commandString string, args string) (string, error) {
	m.ExecuteCalled = true
	return m.ReturnOutput, m.ReturnError
}

func TestRegisterAlias(t *testing.T) {
	// Clear existing registrations for testing
	registryMu.Lock()
	registry = make(map[string]Command)
	aliases = make(map[string]string)
	orderedPrefixes = []string{}
	registryMu.Unlock()

	// Register a test command
	mockCmd := &MockCommand{ReturnOutput: "test output"}
	Register("test:command", mockCmd)

	// Register an alias
	RegisterAlias("tc", "test:command")

	// Verify alias was registered
	if canonical, exists := aliases["tc"]; !exists {
		t.Error("Alias 'tc' was not registered")
	} else if canonical != "test:command" {
		t.Errorf("Alias 'tc' points to %q, expected 'test:command'", canonical)
	}

	// Verify we can dispatch using the alias
	ctx := context.Background()
	output, err := Dispatch(ctx, AgentInfo{}, "tc arg1 arg2")

	if err != nil {
		t.Errorf("Dispatch with alias failed: %v", err)
	}

	if !mockCmd.ExecuteCalled {
		t.Error("Command was not executed when dispatched via alias")
	}

	if output != "test output" {
		t.Errorf("Got output %q, expected 'test output'", output)
	}
}

func TestRegisterAlias_Panics(t *testing.T) {
	// Clear existing registrations for testing
	registryMu.Lock()
	registry = make(map[string]Command)
	aliases = make(map[string]string)
	orderedPrefixes = []string{}
	registryMu.Unlock()

	// Test: Panic when canonical prefix doesn't exist
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when registering alias for non-existent prefix")
		}
	}()

	RegisterAlias("alias", "nonexistent:command")
}

func TestRegisterAlias_DuplicatePanic(t *testing.T) {
	// Clear existing registrations for testing
	registryMu.Lock()
	registry = make(map[string]Command)
	aliases = make(map[string]string)
	orderedPrefixes = []string{}
	registryMu.Unlock()

	mockCmd := &MockCommand{}
	Register("test:cmd", mockCmd)
	RegisterAlias("alias1", "test:cmd")

	// Test: Panic when registering duplicate alias
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when registering duplicate alias")
		}
	}()

	RegisterAlias("alias1", "test:cmd")
}

func TestListCommands(t *testing.T) {
	// Clear existing registrations for testing
	registryMu.Lock()
	registry = make(map[string]Command)
	aliases = make(map[string]string)
	orderedPrefixes = []string{}
	registryMu.Unlock()

	// Register test commands and aliases
	mockCmd1 := &MockCommand{}
	mockCmd2 := &MockCommand{}

	Register("cmd:one", mockCmd1)
	Register("cmd:two", mockCmd2)

	RegisterAlias("c1", "cmd:one")
	RegisterAlias("one", "cmd:one")
	RegisterAlias("c2", "cmd:two")

	// Get command list
	list := ListCommands()

	// Verify structure
	if len(list) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(list))
	}

	// Check cmd:one aliases
	if aliases, exists := list["cmd:one"]; !exists {
		t.Error("Command 'cmd:one' not in list")
	} else if len(aliases) != 2 {
		t.Errorf("Expected 2 aliases for 'cmd:one', got %d", len(aliases))
	}

	// Check cmd:two aliases
	if aliases, exists := list["cmd:two"]; !exists {
		t.Error("Command 'cmd:two' not in list")
	} else if len(aliases) != 1 {
		t.Errorf("Expected 1 alias for 'cmd:two', got %d", len(aliases))
	}
}

func TestDispatch_LongestPrefixMatching(t *testing.T) {
	// Clear existing registrations for testing
	registryMu.Lock()
	registry = make(map[string]Command)
	aliases = make(map[string]string)
	orderedPrefixes = []string{}
	registryMu.Unlock()

	// Register commands with overlapping prefixes
	shortCmd := &MockCommand{ReturnOutput: "short"}
	longCmd := &MockCommand{ReturnOutput: "long"}

	Register("test", shortCmd)
	Register("test:long", longCmd)

	ctx := context.Background()

	// Test that longer prefix wins
	output, err := Dispatch(ctx, AgentInfo{}, "test:long command")
	if err != nil {
		t.Errorf("Dispatch failed: %v", err)
	}
	if output != "long" {
		t.Errorf("Expected 'long', got %q", output)
	}
	if !longCmd.ExecuteCalled {
		t.Error("Long command should have been executed")
	}
	if shortCmd.ExecuteCalled {
		t.Error("Short command should not have been executed")
	}
}


