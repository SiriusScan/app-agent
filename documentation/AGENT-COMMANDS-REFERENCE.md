# Sirius Agent - Command Reference

This document describes the command system and available commands for the Sirius Agent.

## Operating Modes

The Sirius Agent supports two operational modes:

### 1. Direct CLI Mode (Recommended for Testing)

Run commands directly from the terminal without a server connection:

```bash
sirius-agent scan --format text
sirius-agent template run my-template.yaml
sirius-agent module list
```

**Use Cases:**

- Local testing and development
- Quick scans without server infrastructure
- Troubleshooting and debugging
- Template validation

### 2. Server Mode (Production)

Connect to the Sirius gRPC server for remote command execution:

```bash
sirius-agent server
```

The server will send commands using the internal command registry.

## Command System

The Sirius Agent uses a flexible command routing system that supports:

- **CLI commands** (e.g., `sirius-agent scan`)
- **Internal commands** (e.g., `internal:scan` via server)
- **Short aliases** (e.g., `scan`, `tscan`)
- **Longest prefix matching** for command resolution

### How It Works

#### CLI Mode

Direct command execution via Cobra CLI framework:

```bash
sirius-agent <command> [flags]
```

#### Server Mode

Commands sent via gRPC are matched against registered prefixes:

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

## CLI Commands

These commands are available in direct CLI mode. Run any command with `--help` to see detailed usage.

### Scan Command

Perform system scanning and inventory collection.

```bash
# Basic package scan (default: JSON output)
sirius-agent scan

# Human-readable output
sirius-agent scan --format text

# Run custom detection scripts
sirius-agent scan --scripts=check-suid.sh
sirius-agent scan --scripts=script1.sh,script2.sh

# List templates (shortcut)
sirius-agent scan --list-templates

# Save to file
sirius-agent scan --output scan-results.json

# Suppress logging
sirius-agent scan --log-level error
```

### Template Commands

Work with vulnerability detection templates.

```bash
# Run single template
sirius-agent template run ./templates/my-template.yaml

# Run all templates in directory
sirius-agent template run-all ./templates/

# Run all templates with custom workers
sirius-agent template run-all --workers 10 --timeout 300

# Validate template syntax
sirius-agent template validate ./my-template.yaml

# List available templates
sirius-agent template list
sirius-agent template list --directory ./custom-templates/
```

### Module Commands

Inspect available detection modules.

```bash
# List all registered modules
sirius-agent module list

# Show detailed module information
sirius-agent module info file_hash
sirius-agent module info file_content
sirius-agent module info version_cmd
```

### Server Command

Start agent in server mode for gRPC communication.

```bash
# Connect to server (default: localhost:50051)
sirius-agent server

# Connect to custom server
SERVER_ADDRESS=remote-server:50051 sirius-agent server

# With custom agent ID
AGENT_ID=my-agent sirius-agent server
```

### Version Command

Display agent version.

```bash
sirius-agent version
```

## Internal Commands (Server Mode)

When running in server mode, these commands are sent via gRPC from the Sirius server.

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
