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

### Standalone Mode

Run a single template against the local system:

```bash
# Run a specific template
./sirius-agent scan --template templates/builtin/01-file-hash.yaml

# Run all templates in a directory
./sirius-agent scan --template-dir templates/builtin/

# List available templates
./sirius-agent template list
```

### Server Mode

Connect to a Sirius server for centralized management:

```bash
# Set server configuration
export SIRIUS_SERVER_ADDRESS=localhost:50051
export SIRIUS_AGENT_ID=agent-001

# Start the agent
./sirius-agent
```

## Project Structure

```
.
├── cmd/
│   ├── sirius-agent/      # Main agent binary
│   ├── server/            # Development server
│   └── template-cli/      # Template management CLI
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

| Environment Variable    | Description              | Default               |
| ----------------------- | ------------------------ | --------------------- |
| `SIRIUS_SERVER_ADDRESS` | Server gRPC address      | `localhost:50051`     |
| `SIRIUS_AGENT_ID`       | Unique agent identifier  | hostname              |
| `SIRIUS_TEMPLATE_DIR`   | Local template directory | `~/.sirius/templates` |
| `SIRIUS_LOG_LEVEL`      | Logging verbosity        | `info`                |

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
