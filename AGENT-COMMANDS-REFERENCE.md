# Sirius Agent - Command Reference

This document describes the command system and available commands for the Sirius Agent.

## Command System

The Sirius Agent uses a flexible command routing system that supports:

- **Canonical commands** (e.g., `internal:template-scan`)
- **Short aliases** (e.g., `scan`, `tscan`)
- **Longest prefix matching** for command resolution

### How It Works

When you send a command to the agent, the system:

1. Matches the command against all registered prefixes (longest match wins)
2. Resolves any aliases to their canonical command
3. Executes the command with the remaining arguments

## Available Commands & Aliases

Use the `help` command to see all available commands and their aliases:

```
help
```

Or for JSON format:

```
help --json
```

### Core Commands

| Canonical Command             | Short Aliases                    | Description                                         |
| ----------------------------- | -------------------------------- | --------------------------------------------------- |
| `internal:template-scan`      | `scan`, `template-scan`, `tscan` | Scan system using vulnerability detection templates |
| `internal:scan`               | `inventory`, `software`          | Collect software inventory                          |
| `internal:status`             | `status`, `info`                 | Get agent status information                        |
| `internal:repo`               | `repo`, `repository`             | Repository management commands                      |
| `internal:sync`               | `sync`                           | Sync data with server                               |
| `internal:help`               | `help`                           | Display available commands                          |
| `internal:list-templates`     | `templates`, `list-templates`    | List available detection templates                  |
| `internal:discover-templates` | `discover`, `discover-templates` | Discover templates from sources                     |

### Template Scanning Commands

**Scan with all templates:**

```
scan --all
```

**Scan with specific template:**

```
scan --template /path/to/template.yaml
```

**Scan specific directory:**

```
scan --directory /path/to/templates/
```

**Adjust worker count:**

```
scan --all --workers 10
```

**Output formats:**

```
scan --all --format json
scan --all --format text
scan --all --format jsonl
```

## Command Aliasing System

### For Users

Commands can be invoked using either their full canonical name or any registered alias:

```bash
# These are equivalent:
internal:template-scan --all
scan --all
tscan --all
template-scan --all
```

### For Developers

#### Registering Commands

Commands register with a canonical prefix in their `init()` function:

```go
func init() {
    commands.Register("internal:my-command", &MyCommand{})
}
```

#### Adding Aliases

Aliases are centrally managed in `internal/commands/aliases.go`:

```go
func RegisterBuiltinAliases() {
    safeRegisterAlias("mycommand", "internal:my-command")
    safeRegisterAlias("mc", "internal:my-command")
}
```

The `safeRegisterAlias` function ensures:

- The target command exists
- No duplicate aliases
- No conflicts with existing commands

## Command Development

### Creating a New Command

1. Create a new package under `internal/commands/mycommand/`
2. Implement the `commands.Command` interface
3. Register the command with a canonical prefix
4. Add user-friendly aliases in `aliases.go`

Example:

```go
package mycommand

import (
    "context"
    "github.com/SiriusScan/app-agent/internal/commands"
)

type MyCommand struct{}

func (c *MyCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo, commandString string, args string) (string, error) {
    // Implementation here
    return "command output", nil
}

func init() {
    commands.Register("internal:my-command", &MyCommand{})
}
```

Then in `aliases.go`:

```go
safeRegisterAlias("mycommand", "internal:my-command")
safeRegisterAlias("mc", "internal:my-command")
```

## Best Practices

### Command Naming

- **Canonical names**: Use `internal:` prefix for all internal commands
- **Short aliases**: 2-4 characters for frequently used commands
- **Descriptive aliases**: Full words for discoverability

### Alias Design

- Create at least one short alias (`scan`) for frequent use
- Create at least one descriptive alias (`template-scan`) for clarity
- Avoid ambiguous aliases that could confuse users

### Documentation

- Update this file when adding new commands
- Include examples in command help text
- Document all aliases in the help output

## Troubleshooting

### Command Not Found

If you see "unknown internal command":

1. Run `help` to see available commands
2. Check spelling and try aliases
3. Verify the command is registered in the agent

### Alias Conflicts

If an alias doesn't work:

1. It may conflict with an existing command
2. Check `help` output for actual registered aliases
3. Use the canonical command name

## Technical Details

### Longest Prefix Matching

The system uses longest prefix matching, so more specific commands take precedence:

```
# If both "test" and "test:long" are registered:
test          # Matches "test"
test:long     # Matches "test:long" (longer prefix wins)
```

### Thread Safety

The command registry uses `sync.RWMutex` for thread-safe operations.

### Registration Order

1. Commands register in `init()` functions
2. Aliases register in `RegisterBuiltinAliases()`
3. Called from `NewAgent()` after all imports load

For more details, see the source code in `internal/commands/`.
