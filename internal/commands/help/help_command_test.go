package help

import (
	"context"
	"strings"
	"testing"

	"github.com/SiriusScan/app-agent/internal/commands"
	"go.uber.org/zap"
)

func TestHelpCommand_Execute(t *testing.T) {
	t.Log("\n🔍 Testing Help Command")

	cmd := &HelpCommand{}

	tests := []struct {
		name           string
		args           string
		expectJSON     bool
		expectContains []string
	}{
		{
			name:           "Text format (default)",
			args:           "",
			expectJSON:     false,
			expectContains: []string{"Available Agent Commands", "Usage:"},
		},
		{
			name:           "JSON format",
			args:           "--json",
			expectJSON:     true,
			expectContains: []string{`"command"`, `"aliases"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			agentInfo := commands.AgentInfo{
				Logger: zap.NewNop(),
			}

			output, err := cmd.Execute(ctx, agentInfo, "help "+tt.args, tt.args)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if output == "" {
				t.Error("Expected non-empty output")
			}

			// Check for expected content
			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Output missing expected content %q", expected)
				}
			}

			t.Logf("  ✅ Output format: %s", map[bool]string{true: "JSON", false: "Text"}[tt.expectJSON])
			t.Logf("  ✅ Output length: %d characters", len(output))
		})
	}
}

func TestHelpCommand_ListsCommands(t *testing.T) {
	t.Log("\n🔍 Testing Help Command - Lists Commands")

	cmd := &HelpCommand{}
	ctx := context.Background()
	agentInfo := commands.AgentInfo{
		Logger: zap.NewNop(),
	}

	output, err := cmd.Execute(ctx, agentInfo, "help", "")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should contain the help command itself
	if !strings.Contains(output, "help") {
		t.Error("Output should list the help command")
	}

	t.Logf("  ✅ Help command lists available commands")
}


