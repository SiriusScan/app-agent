# Sirius Agent

[![Release](https://img.shields.io/github/v/release/SiriusScan/app-agent)](https://github.com/SiriusScan/app-agent/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/SiriusScan/app-agent)](go.mod)
[![License](https://img.shields.io/github/license/SiriusScan/app-agent)](LICENSE)

A template-based vulnerability detection agent for the Sirius scanning platform. The agent executes YAML-defined templates to detect software vulnerabilities, misconfigurations, and security issues on host systems.

## Features

- **Template-Based Detection**: YAML templates define vulnerability checks using a Nuclei-inspired DSL
- **Modular Architecture**: Pluggable detection modules (file hash, file content, version commands, scripts)
- **Cross-Platform**: Supports Linux, macOS, and Windows (amd64 and arm64)
- **Standalone Mode**: Run locally without server connectivity for rapid development and testing
- **Server Integration**: Connects to Sirius server via gRPC for centralized management
- **Template Repository**: Automatic template synchronization with versioning and integrity checks

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [Releases](https://github.com/SiriusScan/app-agent/releases) page.

### Build from Source

```bash
# Clone the repository
git clone https://github.com/SiriusScan/app-agent.git
cd app-agent

# Build
go build -o sirius-agent ./cmd/sirius-agent

# Or use GoReleaser for all platforms
goreleaser build --snapshot --clean
```

## Quick Start

### CLI Mode (Local Execution)

Run a single template against the local system:

```bash
# Run a specific template
./sirius-agent template run ./templates/builtin/01-file-hash.yaml

# Run all templates discovered by the template manager
./sirius-agent template run-all

# Run all templates in a directory
./sirius-agent template run-all ./templates/builtin/

# List available templates
./sirius-agent template list
```

### Server Mode (Default)

Connect to a Sirius server for centralized management:

```bash
# 1) Verify binary
./sirius-agent version

# 2) Start with defaults (SERVER_ADDRESS defaults to localhost:50051)
./sirius-agent

# 3) Optional: override via environment
SERVER_ADDRESS=localhost:50051 AGENT_ID=agent-001 ./sirius-agent

# 4) Optional: explicit server command
./sirius-agent server --address localhost:50051 --agent-id agent-001
```

## Project Structure

```
.
├── cmd/
│   ├── sirius-agent/      # Main agent binary
│   ├── agent/             # Legacy standalone entrypoint (deprecated)
│   ├── server/            # Sirius Engine server binary
│   ├── template-cli/      # Template management CLI tools
│   ├── test-discovery/    # Template discovery test harness
│   └── test-integration/  # Integration test harness
├── internal/
│   ├── agent/             # Agent core logic
│   ├── cmd/               # CLI command implementations
│   ├── commands/          # Agent commands (scan, sync, etc.)
│   ├── common/            # Shared utilities
│   ├── config/            # Configuration management
│   ├── modules/           # Detection modules
│   │   ├── filecontent/   # File content/regex matching
│   │   ├── filehash/      # File hash verification
│   │   └── versioncmd/    # Version command execution
│   ├── repository/        # Template repository management
│   ├── server/            # Server-side components
│   └── template/          # Template parsing and execution
├── proto/                 # Protocol Buffer definitions
├── templates/
│   ├── builtin/           # Built-in detection templates
│   └── examples/          # Example templates
├── testing/               # Integration test infrastructure
└── documentation/         # Project documentation
```

## Legacy Notice

`cmd/agent` is kept for backward compatibility. The canonical runtime and release target is `cmd/sirius-agent`.

## Template Format

Templates use a YAML-based DSL inspired by Nuclei:

```yaml
id: example-weak-password
info:
  name: Weak Password Detection
  author: security-team
  severity: high
  description: Detects common weak password patterns
  tags: [password, security, config]

detection:
  - type: file_content
    path: /etc/shadow
    regex: "root::\\$"

  - type: file_content
    path: /etc/passwd
    regex: "root::0:0"

test_strategy: |
  Check /etc/shadow and /etc/passwd for accounts without passwords.

remediation: |
  Set strong passwords for all system accounts.
```

See [documentation/README.template-architect-guide.md](documentation/README.template-architect-guide.md) for the complete template authoring guide.

## Configuration

| Environment Variable | Description                                                                    | Default                       |
| -------------------- | ------------------------------------------------------------------------------ | ----------------------------- |
| `SERVER_ADDRESS`     | Agent gRPC server address (agent mode) / server bind address (server process) | `localhost:50051` or `:50051` |
| `AGENT_ID`           | Unique agent identifier                                                        | hostname                      |
| `HOST_ID`            | Host record identifier                                                         | `AGENT_ID`                    |
| `API_BASE_URL`       | Backend REST API URL                                                           | `http://<host>:9001`          |
| `POWERSHELL_PATH`    | PowerShell path override                                                       | auto-detect                   |
| `ENABLE_SCRIPTING`   | Enable script execution                                                        | `true`                        |
| `AGENT_AUTH_TOKEN`   | Persisted auth token value                                                     | empty                         |
| `AGENT_TOKEN_FILE`   | Token file location                                                            | `/data/.sirius-agent-token`*  |
| `LOG_LEVEL`          | Logging verbosity                                                              | `info`                        |

\* Falls back to `~/.sirius-agent-token` when `/data` is unavailable.

## Development

### Prerequisites

- Go 1.23+
- Protocol Buffers compiler (protoc)
- GoReleaser (for releases)

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests (requires Docker)
cd testing && make test
```

### Building

```bash
# Development build
go build -o sirius-agent ./cmd/sirius-agent

# Release build with version info
go build -ldflags "-X main.version=v1.0.0" -o sirius-agent ./cmd/sirius-agent

# Cross-platform builds
goreleaser build --snapshot --clean
```

## Documentation

- [Template Architect Guide](documentation/README.template-architect-guide.md) - How to write detection templates
- [Agent Commands Reference](documentation/AGENT-COMMANDS-REFERENCE.md) - CLI command documentation
- [Risk Scoring](documentation/RISK-SCORING.md) - Vulnerability severity and scoring
- [PRD](documentation/agent_template_system_PRD.md) - Product requirements document

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is part of the Sirius Security Platform. See [LICENSE](LICENSE) for details.
