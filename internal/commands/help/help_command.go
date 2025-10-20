package help

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/SiriusScan/app-agent/internal/commands"
)

func init() {
	commands.Register("help", &HelpCommand{})
	commands.Register("internal:help", &HelpCommand{})
}

// HelpCommand displays available commands and their aliases.
type HelpCommand struct{}

// Execute lists all available commands and their aliases.
func (h *HelpCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo, commandString string, args string) (string, error) {
	agentInfo.Logger.Info("Executing help command")

	cmdList := commands.ListCommands()

	// Check if user wants JSON format
	format := "text"
	if strings.Contains(args, "--json") {
		format = "json"
	}

	if format == "json" {
		return h.generateJSONOutput(cmdList)
	}

	return h.generateTextOutput(cmdList), nil
}

func (h *HelpCommand) generateTextOutput(cmdList map[string][]string) string {
	var output strings.Builder

	output.WriteString("📋 Available Agent Commands\n")
	output.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Sort commands for consistent output
	commands := make([]string, 0, len(cmdList))
	for cmd := range cmdList {
		commands = append(commands, cmd)
	}
	sort.Strings(commands)

	for _, cmd := range commands {
		aliases := cmdList[cmd]

		// Format command
		output.WriteString(fmt.Sprintf("• %s\n", cmd))

		// Add aliases if they exist
		if len(aliases) > 0 {
			sort.Strings(aliases) // Sort aliases for consistency
			output.WriteString(fmt.Sprintf("  Aliases: %s\n", strings.Join(aliases, ", ")))
		}
		output.WriteString("\n")
	}

	output.WriteString(strings.Repeat("=", 50) + "\n")
	output.WriteString("💡 Usage: Send any command or its alias to the agent\n")
	output.WriteString("   Example: 'scan --all' or 'template-scan --all'\n")

	return output.String()
}

func (h *HelpCommand) generateJSONOutput(cmdList map[string][]string) (string, error) {
	type CommandInfo struct {
		Command string   `json:"command"`
		Aliases []string `json:"aliases"`
	}

	commands := make([]CommandInfo, 0, len(cmdList))
	for cmd, aliases := range cmdList {
		sort.Strings(aliases)
		commands = append(commands, CommandInfo{
			Command: cmd,
			Aliases: aliases,
		})
	}

	// Sort by command name
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Command < commands[j].Command
	})

	jsonData, err := json.MarshalIndent(commands, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonData), nil
}


