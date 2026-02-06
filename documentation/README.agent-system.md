# Sirius Agent System - Comprehensive Documentation

## Executive Summary

The Sirius Agent System is a distributed vulnerability detection platform consisting of lightweight agents deployed on target hosts and a centralized server (engine) that orchestrates scanning operations. The system uses a template-based detection approach inspired by Nuclei, enabling flexible and extensible vulnerability scanning without requiring agent updates for new detection rules.

---

## Table of Contents

1. [System Architecture](#system-architecture)
2. [Core Components](#core-components)
3. [Communication Protocol](#communication-protocol)
4. [Agent Deep Dive](#agent-deep-dive)
5. [Server (Engine) Deep Dive](#server-engine-deep-dive)
6. [Template System](#template-system)
7. [Detection Modules](#detection-modules)
8. [Command System](#command-system)
9. [Storage & State Management](#storage--state-management)
10. [Template Repository Management](#template-repository-management)
11. [Configuration Reference](#configuration-reference)
12. [Development Guide](#development-guide)
13. [Deployment](#deployment)

---

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           SIRIUS PLATFORM                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────────┐          ┌──────────────────────────────────┐    │
│  │   Sirius UI      │          │        Sirius Engine             │    │
│  │  (Frontend)      │◄─────────┤        (app-agent/server)        │    │
│  │                  │  REST    │                                  │    │
│  └──────────────────┘  API     │  ┌────────────────────────────┐  │    │
│                                │  │  Template Manager          │  │    │
│                                │  │  - GitHub Sync             │  │    │
│                                │  │  - Repository Management   │  │    │
│                                │  │  - Priority Resolution     │  │    │
│                                │  └────────────────────────────┘  │    │
│                                │                                  │    │
│                                │  ┌────────────────────────────┐  │    │
│                                │  │  Agent Manager             │  │    │
│                                │  │  - gRPC Stream Handler     │  │    │
│                                │  │  - Command Dispatch        │  │    │
│                                │  │  - Response Tracking       │  │    │
│                                │  └────────────────────────────┘  │    │
│                                └──────────────────────────────────┘    │
│                                              │                          │
│                              gRPC (Bidirectional Stream)                │
│                                              │                          │
│      ┌───────────────────────────────────────┴────────────────────┐    │
│      │                                                            │    │
│  ┌───▼───────────┐    ┌───────────────┐    ┌───────────────┐     │    │
│  │   Agent 1     │    │   Agent 2     │    │   Agent N     │     │    │
│  │  (Linux)      │    │  (Windows)    │    │  (macOS)      │     │    │
│  │               │    │               │    │               │     │    │
│  │ ┌───────────┐ │    │ ┌───────────┐ │    │ ┌───────────┐ │     │    │
│  │ │ Executor  │ │    │ │ Executor  │ │    │ │ Executor  │ │     │    │
│  │ └───────────┘ │    │ └───────────┘ │    │ └───────────┘ │     │    │
│  │ ┌───────────┐ │    │ ┌───────────┐ │    │ ┌───────────┐ │     │    │
│  │ │ Modules   │ │    │ │ Modules   │ │    │ │ Modules   │ │     │    │
│  │ └───────────┘ │    │ └───────────┘ │    │ └───────────┘ │     │    │
│  └───────────────┘    └───────────────┘    └───────────────┘     │    │
│                                                                  │    │
└─────────────────────────────────────────────────────────────────────────┘

                    EXTERNAL SERVICES
┌──────────────────────────────────────────────────────────────────────────┐
│                                                                          │
│  ┌─────────────┐    ┌─────────────┐    ┌────────────────────────────┐   │
│  │   Valkey    │    │  RabbitMQ   │    │  GitHub Repositories       │   │
│  │  (Storage)  │    │   (Queue)   │    │  - sirius-agent-modules    │   │
│  │             │    │             │    │  - Custom repos            │   │
│  └─────────────┘    └─────────────┘    └────────────────────────────┘   │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
1. Template Sync Flow:
   GitHub Repository → Server → Valkey Storage → Agent Local Cache

2. Command Flow:
   UI → RabbitMQ → Server → gRPC Stream → Agent → Execute → Results → Valkey → UI

3. Heartbeat Flow:
   Agent → gRPC Stream → Server (every 30 seconds)
```

---

## Core Components

### 1. Sirius Agent (`cmd/sirius-agent`)

A lightweight, cross-platform binary that runs on target hosts. Responsible for:

- Executing vulnerability detection templates
- Collecting system inventory (packages, software)
- Running custom detection scripts
- Maintaining connection with the server
- Local template caching and execution

**Supported Platforms:**

- Linux (amd64, arm64)
- Windows (amd64, arm64)
- macOS (amd64, arm64)

### 2. Sirius Engine (`cmd/server`)

The centralized server component that:

- Manages agent connections via gRPC
- Processes commands from RabbitMQ queues
- Synchronizes templates from GitHub repositories
- Stores data in Valkey (Redis-compatible)
- Coordinates template distribution to agents

### 3. Template System

A YAML-based domain-specific language (DSL) for defining vulnerability checks:

- Inspired by Project Nuclei
- Supports multiple detection module types
- Platform-specific detection steps
- Risk scoring via CVSS or custom scores

### 4. Detection Modules

Pluggable detection engines:

- **file_hash**: Verify file integrity via cryptographic hashes
- **file_content**: Search files for patterns using regex
- **version_cmd**: Execute commands and extract version information

---

## Communication Protocol

### gRPC Service Definition

The communication between agents and server uses Protocol Buffers:

```protobuf
service HelloService {
  // Simple connectivity check
  rpc Ping(PingRequest) returns (PingResponse) {}

  // Execute a single command
  rpc ExecuteCommand(CommandRequest) returns (CommandResponse) {}

  // Bidirectional streaming for continuous communication
  rpc ConnectStream(stream AgentMessage) returns (stream ServerMessage) {}
}
```

### Message Types

| Type                    | Direction      | Purpose                            |
| ----------------------- | -------------- | ---------------------------------- |
| `HEARTBEAT`             | Agent → Server | Periodic health check with metrics |
| `COMMAND`               | Server → Agent | Execute a command on the agent     |
| `RESULT`                | Agent → Server | Command execution results          |
| `ACKNOWLEDGMENT`        | Server → Agent | Confirm result receipt             |
| `TEMPLATE_UPDATE`       | Server → Agent | Push template updates              |
| `TEMPLATE_SYNC_REQUEST` | Agent → Server | Request template synchronization   |

### Connection Lifecycle

```
Agent                                Server
  │                                    │
  ├─────── Connect (gRPC) ────────────►│
  │                                    │
  │◄──────── Stream Open ──────────────┤
  │                                    │
  │    AgentMessage (Heartbeat)        │
  ├───────────────────────────────────►│
  │                                    │
  │    ServerMessage (Command)         │
  │◄───────────────────────────────────┤
  │                                    │
  │    AgentMessage (Result)           │
  ├───────────────────────────────────►│
  │                                    │
  │    ServerMessage (Acknowledgment)  │
  │◄───────────────────────────────────┤
  │                                    │
  └────── (repeats) ───────────────────┘
```

---

## Agent Deep Dive

### Agent Initialization

```go
// Agent structure
type Agent struct {
    logger           *zap.Logger
    config           *config.AgentConfig
    conn             *grpc.ClientConn
    client           pb.HelloServiceClient
    stream           pb.HelloService_ConnectStreamClient
    startTime        time.Time
    agentInfo        commands.AgentInfo
    powerShellPath   string
    scriptingEnabled bool
    syncManager      *templateagent.AgentSyncManager
}
```

### Agent Operating Modes

#### 1. Server Mode (Default)

Connects to the Sirius server and waits for commands:

```bash
./sirius-agent
# or explicitly
./sirius-agent server
```

#### 2. CLI Mode (Standalone)

Execute commands locally without server connectivity:

```bash
# Run vulnerability scan with all templates
./sirius-agent scan --all

# Run specific template
./sirius-agent template run ./templates/my-template.yaml

# List available modules
./sirius-agent module list
```

### Command Processing Flow

```
                    ┌─────────────────────┐
                    │  Receive Command    │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │  Match Prefix       │
                    │  (Longest Match)    │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
    ┌─────────▼─────────┐     │      ┌─────────▼─────────┐
    │  Internal Command │     │      │  Unknown Command  │
    │  (commands pkg)   │     │      │  (Script/Shell)   │
    └─────────┬─────────┘     │      └─────────┬─────────┘
              │               │                │
              │               │                │
    ┌─────────▼─────────┐     │      ┌─────────▼─────────┐
    │  Execute Handler  │     │      │  PowerShell/Bash  │
    └─────────┬─────────┘     │      │  Execution        │
              │               │      └─────────┬─────────┘
              │               │                │
              └───────────────┴────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │  Send Result      │
                    │  to Server        │
                    └───────────────────┘
```

### Heartbeat Mechanism

The agent sends periodic heartbeats every 30 seconds containing:

- Timestamp
- CPU usage (planned)
- Memory usage (current heap allocation in MB)

---

## Server (Engine) Deep Dive

### Server Architecture

```go
type Server struct {
    pb.UnimplementedHelloServiceServer
    logger            *zap.Logger
    config            *config.ServerConfig
    server            *grpc.Server

    // Agent management
    agentsMutex       sync.RWMutex
    agents            map[string]pb.HelloService_ConnectStreamServer

    // Command tracking
    commandsMutex     sync.RWMutex
    commands          map[string]*CommandStatus
    pendingCommands   map[string]string

    // Storage
    responseStore     store.ResponseStore
    valkeyClient      valkey.Client

    // Template management
    templateManager   *ServerTemplateManager
    repositoryManager *RepositoryManager
    syncQueueProcessor *TemplateSyncQueueProcessor
}
```

### Queue Processing

The server listens to RabbitMQ queues for commands:

| Queue Name                 | Purpose                   |
| -------------------------- | ------------------------- |
| `agent_commands`           | Incoming commands from UI |
| `agent_response`           | Responses back to UI      |
| `agent.template.sync.jobs` | Template sync requests    |

### Command Message Format

```json
{
  "action": "list_agents|initialize_session|<empty>",
  "command": "scan --all",
  "agentId": "agent-001",
  "userId": "user-123",
  "timestamp": "2024-01-15T10:30:00Z",
  "sessionId": "session-456",
  "target": {
    "type": "agent",
    "id": "agent-001"
  },
  "responseQueue": "terminal_response"
}
```

---

## Template System

### Template Structure

```yaml
id: CVE-2024-EXAMPLE
info:
  name: Example Vulnerability Detection
  author: security-team
  severity: high
  vulnerability_id: CVE-2024-EXAMPLE # Optional: VID for Sirius DB
  description: Detects example vulnerability

  # Risk Scoring (Priority: risk_score > cvss_vector > cvss_score > severity)
  risk_score: 8.5 # Custom score (0.0-10.0)
  cvss_vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
  cvss_score: 9.8

  references:
    - https://example.com/advisory
  cve:
    - CVE-2024-EXAMPLE
  cwe:
    - CWE-79
  tags:
    - web
    - injection
  version: "1.0"
  remediation: |
    Update to version 2.0 or later.

detection:
  logic: all # 'all' (AND) or 'any' (OR)
  steps:
    - type: file_hash
      platforms:
        - linux
        - darwin
      weight: 1.0
      config:
        path: /usr/bin/vulnerable-app
        hash: abc123...
        algorithm: sha256

    - type: file_content
      platforms:
        - linux
      weight: 0.8
      config:
        path: /etc/app/config.yaml
        regex: "version:\\s*1\\.0"
        multiline: false

    - type: version_cmd
      weight: 0.9
      config:
        command: ["vulnerable-app", "--version"]
        regex: "v(1\\.[0-5]\\.[0-9]+)"
        exit_code: 0
```

### Template Execution Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Template Execution                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────────┐ │
│  │   Parse     │───►│   Filter    │───►│   Execute Steps         │ │
│  │   YAML      │    │   by OS     │    │   Sequentially          │ │
│  └─────────────┘    └─────────────┘    └───────────┬─────────────┘ │
│                                                    │               │
│                              ┌─────────────────────┼───────────────┤
│                              │                     │               │
│                    ┌─────────▼───────┐   ┌────────▼────────┐      │
│                    │  Step 1:        │   │  Step 2:        │      │
│                    │  file_hash      │   │  file_content   │ ...  │
│                    │  ┌───────────┐  │   │  ┌───────────┐  │      │
│                    │  │ Module    │  │   │  │ Module    │  │      │
│                    │  │ Execute   │  │   │  │ Execute   │  │      │
│                    │  └───────────┘  │   │  └───────────┘  │      │
│                    │       │         │   │       │         │      │
│                    │  matched/error  │   │  matched/error  │      │
│                    └────────┬────────┘   └────────┬────────┘      │
│                             │                     │               │
│                             └──────────┬──────────┘               │
│                                        │                          │
│                          ┌─────────────▼─────────────┐            │
│                          │   Evaluate Logic          │            │
│                          │   (AND/OR)                │            │
│                          └─────────────┬─────────────┘            │
│                                        │                          │
│                          ┌─────────────▼─────────────┐            │
│                          │   Calculate Confidence    │            │
│                          │   & Risk Score            │            │
│                          └─────────────┬─────────────┘            │
│                                        │                          │
│                          ┌─────────────▼─────────────┐            │
│                          │   Build Result            │            │
│                          └───────────────────────────┘            │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
```

### Risk Scoring Priority

1. **Custom Risk Score** (`risk_score: 8.5`) - Direct numerical value
2. **CVSS Vector** (`cvss_vector: "CVSS:3.1/..."`) - Parsed and calculated
3. **CVSS Score** (`cvss_score: 9.8`) - Pre-calculated CVSS
4. **Severity Mapping** - Fallback based on severity level

| Severity | Default Score |
| -------- | ------------- |
| Critical | 9.5           |
| High     | 7.5           |
| Medium   | 5.0           |
| Low      | 2.0           |
| Info     | 0.0           |

---

## Detection Modules

### Module Registry

Modules self-register during initialization via Go's `init()` function:

```go
func init() {
    descriptor := modules.Descriptor{
        Type:        "file_hash",
        Name:        "File Hash Validator",
        Description: "Compares file cryptographic hashes",
        Version:     "1.0.0",
        Author:      "Sirius Scan",
        SupportedOS: []string{"linux", "darwin", "windows"},
        ConfigDocs: map[string]string{
            "path":      "Path to the file to check",
            "hash":      "Expected hash value",
            "algorithm": "Hash algorithm (sha256, sha1, md5, sha512)",
        },
    }
    registry.Register(&FileHashModule{}, descriptor)
}
```

### Available Modules

#### 1. File Hash (`file_hash`)

Validates file integrity by comparing cryptographic hashes.

**Config:**

```yaml
config:
  path: /usr/bin/application
  hash: ace82037990b313aa3bfb845f7c609ee1c3f811287fc6fa6896cde2ec47327cd
  algorithm: sha256 # sha256, sha1, md5, sha512
```

**Evidence:**

```json
{
  "path": "/usr/bin/application",
  "algorithm": "sha256",
  "expected_hash": "ace82037...",
  "actual_hash": "ace82037...",
  "matched": true
}
```

#### 2. File Content (`file_content`)

Searches file contents for regex patterns.

**Config:**

```yaml
config:
  path: /etc/config.yaml
  regex: "password:\\s*admin"
  multiline: false
```

**Evidence:**

```json
{
  "file_path": "/etc/config.yaml",
  "pattern": "password:\\s*admin",
  "matched": true,
  "matched_text": "password: admin",
  "matched_line": 15
}
```

**Features:**

- 10MB file size limit
- 5-second regex timeout (ReDoS protection)
- Line-by-line or multiline matching

#### 3. Version Command (`version_cmd`)

Executes commands and extracts version information.

**Config:**

```yaml
config:
  command: ["apache2", "-v"]
  regex: "Apache/(\\d+\\.\\d+\\.\\d+)"
  exit_code: 0 # Optional
```

**Evidence:**

```json
{
  "command": ["apache2", "-v"],
  "exit_code": 0,
  "stdout": "Server version: Apache/2.4.51 (Ubuntu)",
  "stderr": "",
  "regex": "Apache/(\\d+\\.\\d+\\.\\d+)",
  "matched_version": "2.4.51"
}
```

**Security:**

- Commands are executed directly (no shell interpretation)
- 30-second execution timeout
- Captures both stdout and stderr

---

## Command System

### Internal Commands

Commands are registered with canonical prefixes and can have aliases:

| Canonical Command             | Aliases                          | Description                 |
| ----------------------------- | -------------------------------- | --------------------------- |
| `internal:template-scan`      | `scan`, `tscan`, `template-scan` | Run vulnerability templates |
| `internal:scan`               | `inventory`, `software`          | Collect software inventory  |
| `internal:status`             | `status`, `info`                 | Get agent status            |
| `internal:repo`               | `repo`, `repository`             | Repository management       |
| `internal:sync`               | `sync`                           | Sync data with server       |
| `internal:help`               | `help`                           | List available commands     |
| `internal:list-templates`     | `templates`, `list-templates`    | List templates              |
| `internal:discover-templates` | `discover`                       | Discover templates          |

### Command Interface

```go
type Command interface {
    Execute(ctx context.Context, agentInfo AgentInfo,
            commandString string, args string) (output string, err error)
}
```

### Longest Prefix Matching

The dispatcher uses longest prefix matching for command resolution:

```
Registered: "test", "test:long", "scan"

Input: "test:long --verbose"
Match: "test:long" (longer match wins)
Args:  "--verbose"
```

---

## Storage & State Management

### Valkey (Redis-Compatible) Storage

The system uses Valkey for persistent storage:

**Key Patterns:**

| Pattern                               | Purpose                   |
| ------------------------------------- | ------------------------- |
| `template:manifest`                   | Global template manifest  |
| `template:standard:{id}`              | Standard template content |
| `template:custom:{id}`                | Custom template content   |
| `template:meta:{id}`                  | Template metadata         |
| `cmd:response:{agent}:{cmd}`          | Command responses         |
| `sirius:agent-templates:repositories` | Repository configuration  |

### Response Store

Command results are stored in Valkey for UI retrieval:

```go
type CommandResponse struct {
    CommandID   string
    AgentID     string
    Command     string
    Status      string    // pending, completed, failed
    Output      string
    Error       string
    ExitCode    int
    StartTime   time.Time
    EndTime     time.Time
}
```

---

## Template Repository Management

### Multi-Repository Support

The system supports multiple template repositories with priority-based resolution:

```go
type Repository struct {
    ID            string  // Unique identifier
    Name          string  // Display name
    URL           string  // GitHub URL
    Branch        string  // Git branch
    Priority      int     // Higher = higher priority
    Enabled       bool    // Enable/disable sync
    LastSync      *string // Last sync timestamp
    TemplateCount int     // Number of templates
    Status        string  // synced, syncing, error, never_synced
}
```

### Sync Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Template Sync Flow                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌────────────────┐                                                 │
│  │  Sync Request  │  (RabbitMQ: agent.template.sync.jobs)           │
│  └───────┬────────┘                                                 │
│          │                                                          │
│          ▼                                                          │
│  ┌────────────────┐                                                 │
│  │  Repository    │  Load repos from Valkey                         │
│  │  Manager       │                                                 │
│  └───────┬────────┘                                                 │
│          │                                                          │
│          ▼                                                          │
│  ┌────────────────┐                                                 │
│  │  GitHub Sync   │  Download manifest + templates                  │
│  │  Manager       │  from raw.githubusercontent.com                 │
│  └───────┬────────┘                                                 │
│          │                                                          │
│          ▼                                                          │
│  ┌────────────────┐                                                 │
│  │  Valkey        │  Store templates + metadata                     │
│  │  Storage       │                                                 │
│  └───────┬────────┘                                                 │
│          │                                                          │
│          ▼                                                          │
│  ┌────────────────┐                                                 │
│  │  Notify        │  Send sync command to connected agents          │
│  │  Agents        │                                                 │
│  └────────────────┘                                                 │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Repository Manifest Structure

Expected at `https://github.com/{org}/{repo}/main/repository-manifest.json`:

```json
{
  "version": "2.0.0",
  "updated": "2024-01-15T10:30:00Z",
  "components": {
    "templates": {
      "manifest": "templates/manifest.json"
    },
    "scripts": {
      "manifest": "scripts/manifest.json"
    }
  }
}
```

---

## Configuration Reference

### Environment Variables

#### Agent Configuration

| Variable           | Description                        | Default              |
| ------------------ | ---------------------------------- | -------------------- |
| `SERVER_ADDRESS`   | gRPC server address                | `localhost:50051`    |
| `AGENT_ID`         | Unique agent identifier            | hostname             |
| `HOST_ID`          | Host record ID for backend         | `AGENT_ID`           |
| `API_BASE_URL`     | Backend REST API URL               | `http://{host}:9001` |
| `POWERSHELL_PATH`  | Override PowerShell path           | auto-detect          |
| `ENABLE_SCRIPTING` | Enable PowerShell/script execution | `true`               |

#### Server Configuration

| Variable         | Description                | Default          |
| ---------------- | -------------------------- | ---------------- |
| `SERVER_ADDRESS` | Listen address             | `:50051`         |
| `SIRIUS_VALKEY`  | Valkey connection string   | `localhost:6379` |
| `METRICS_ADDR`   | Prometheus metrics address | `:2112`          |

#### Logging

| Variable            | Description            | Default |
| ------------------- | ---------------------- | ------- |
| `SIRIUS_LOG_LEVEL`  | Log verbosity          | `info`  |
| `SIRIUS_LOG_FORMAT` | Log format (text/json) | `text`  |

### Agent Config Structure

```go
type AgentConfig struct {
    ServerAddress   string // gRPC server address
    AgentID         string // Unique agent identifier
    HostID          string // Host record ID
    ApiBaseURL      string // Backend API URL
    PowerShellPath  string // PowerShell executable path
    EnableScripting bool   // Enable script execution
}
```

---

## Development Guide

### Project Structure

```
app-agent/
├── cmd/
│   ├── sirius-agent/          # Main agent binary entry point
│   ├── server/                # Development server entry point
│   ├── template-cli/          # Template management CLI
│   ├── test-discovery/        # Template discovery testing
│   └── test-integration/      # Integration test runner
├── internal/
│   ├── agent/                 # Agent core logic
│   ├── apiclient/             # Backend API client
│   ├── cmd/                   # CLI command implementations (Cobra)
│   ├── command/               # Command message types
│   ├── commands/              # Internal command handlers
│   │   ├── help/
│   │   ├── repo/
│   │   ├── scan/
│   │   ├── status/
│   │   ├── sync/
│   │   ├── template/
│   │   └── templatescan/
│   ├── common/                # Shared utilities
│   │   ├── color/             # ANSI color codes
│   │   ├── errors/            # Custom error types
│   │   ├── files/             # File operations
│   │   └── patterns/          # Regex matching
│   ├── config/                # Configuration loading
│   ├── detect/                # Detection infrastructure
│   ├── modules/               # Detection modules
│   │   ├── filecontent/
│   │   ├── filehash/
│   │   ├── registry/
│   │   └── versioncmd/
│   ├── repository/            # GitHub repository management
│   ├── server/                # Server-side components
│   ├── shell/                 # Shell/PowerShell execution
│   ├── store/                 # Valkey storage adapters
│   ├── sysinfo/               # System information gathering
│   └── template/              # Template system
│       ├── agent/             # Agent-side template management
│       ├── executor/          # Template executor
│       ├── fingerprint/       # System fingerprinting
│       ├── parser/            # YAML parsing
│       ├── reporting/         # Result conversion
│       ├── risk/              # Risk score calculation
│       ├── storage/           # Template storage
│       ├── types/             # Type definitions
│       └── valkey/            # Valkey template storage
├── proto/                     # Protocol Buffer definitions
│   └── hello/
│       ├── hello.proto
│       ├── hello.pb.go
│       └── hello_grpc.pb.go
├── templates/                 # Built-in templates
│   ├── builtin/
│   └── examples/
├── testing/                   # Integration test infrastructure
└── documentation/             # Project documentation
```

### Adding a New Detection Module

1. Create a new package under `internal/modules/`:

```go
// internal/modules/mymodule/mymodule.go
package mymodule

import (
    "context"
    "github.com/SiriusScan/app-agent/internal/modules"
    "github.com/SiriusScan/app-agent/internal/modules/registry"
)

type MyModule struct{}

func (m *MyModule) Execute(ctx context.Context, config modules.StepConfig) (*modules.Result, error) {
    // Implementation
    return &modules.Result{
        Matched:  true,
        Evidence: map[string]interface{}{"key": "value"},
    }, nil
}

func init() {
    descriptor := modules.Descriptor{
        Type:        "my_module",
        Name:        "My Custom Module",
        Description: "Description of what it does",
        Version:     "1.0.0",
        Author:      "Your Name",
        SupportedOS: []string{"linux", "darwin", "windows"},
        ConfigDocs: map[string]string{
            "param1": "Description of param1",
        },
    }
    registry.Register(&MyModule{}, descriptor)
}
```

2. Import the module in `cmd/sirius-agent/main.go`:

```go
import (
    _ "github.com/SiriusScan/app-agent/internal/modules/mymodule"
)
```

### Adding a New Internal Command

1. Create command package:

```go
// internal/commands/mycommand/mycommand.go
package mycommand

import (
    "context"
    "github.com/SiriusScan/app-agent/internal/commands"
)

type MyCommand struct{}

var _ commands.Command = (*MyCommand)(nil)

func init() {
    commands.Register("internal:mycommand", &MyCommand{})
}

func (c *MyCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo,
                            commandString string, args string) (string, error) {
    // Implementation
    return "output", nil
}
```

2. Register aliases in `internal/commands/aliases.go`:

```go
func RegisterBuiltinAliases() {
    // ... existing aliases ...
    safeRegisterAlias("myalias", "internal:mycommand")
}
```

3. Import in `cmd/sirius-agent/main.go`:

```go
import (
    _ "github.com/SiriusScan/app-agent/internal/commands/mycommand"
)
```

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests (requires Docker)
cd testing && make test

# Run specific package tests
go test ./internal/modules/filehash/...

# With verbose output
go test -v ./internal/template/executor/...
```

### Building

```bash
# Development build
go build -o sirius-agent ./cmd/sirius-agent

# With version info
go build -ldflags "-X main.version=v1.0.0" -o sirius-agent ./cmd/sirius-agent

# Cross-platform builds (using GoReleaser)
goreleaser build --snapshot --clean
```

---

## Deployment

### Docker Deployment

**Server (Engine):**

```dockerfile
# Development
docker build --target development -t sirius-engine:dev .
docker run -p 50051:50051 -v $(pwd):/app sirius-engine:dev

# Production
docker build --target final -t sirius-engine:latest .
docker run -p 50051:50051 sirius-engine:latest
```

**Required Services:**

```yaml
# docker-compose.yml excerpt
services:
  sirius-engine:
    build: ./app-agent
    ports:
      - "50051:50051"
    depends_on:
      - valkey
      - rabbitmq
    environment:
      - SIRIUS_VALKEY=valkey:6379

  valkey:
    image: valkey/valkey:latest
    ports:
      - "6379:6379"

  rabbitmq:
    image: rabbitmq:3-management
    ports:
      - "5672:5672"
      - "15672:15672"
```

### Agent Deployment

Download pre-built binaries or build from source:

```bash
# Linux/macOS
curl -LO https://github.com/SiriusScan/app-agent/releases/latest/download/sirius-agent-linux-amd64
chmod +x sirius-agent-linux-amd64
./sirius-agent-linux-amd64

# Windows
Invoke-WebRequest -Uri "https://github.com/SiriusScan/app-agent/releases/latest/download/sirius-agent-windows-amd64.exe" -OutFile "sirius-agent.exe"
.\sirius-agent.exe
```

### systemd Service (Linux)

```ini
# /etc/systemd/system/sirius-agent.service
[Unit]
Description=Sirius Security Agent
After=network.target

[Service]
Type=simple
User=sirius
Environment=SERVER_ADDRESS=sirius-server:50051
Environment=AGENT_ID=%H
ExecStart=/usr/local/bin/sirius-agent
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

---

## Security Considerations

### Template Security

1. **No Shell Interpretation**: Commands are executed directly without shell expansion
2. **Content Scanning**: Templates are scanned for dangerous patterns:
   - `eval(`, `exec(`, `system(`
   - Shell injection patterns (`$(`, `${`)
   - Script injection (`<script`, `javascript:`)
3. **Size Limits**: Maximum template size of 1MB
4. **Regex Timeout**: 5-second timeout prevents ReDoS attacks

### Agent Security

1. **TLS Support**: gRPC supports TLS (currently using insecure for development)
2. **Scripting Control**: PowerShell/script execution can be disabled
3. **Capability Reporting**: Agent reports scripting capability to server

### Communication Security

1. **Metadata Authentication**: Agent ID sent via gRPC metadata
2. **Bidirectional Streaming**: Persistent connection reduces attack surface

---

## Glossary

| Term              | Definition                                                 |
| ----------------- | ---------------------------------------------------------- |
| **Agent**         | Lightweight binary deployed on target hosts                |
| **Engine/Server** | Central server managing agents and templates               |
| **Template**      | YAML file defining vulnerability detection logic           |
| **Module**        | Pluggable detection engine (file_hash, file_content, etc.) |
| **VID**           | Vulnerability ID used in Sirius database                   |
| **Valkey**        | Redis-compatible key-value store                           |
| **gRPC**          | High-performance RPC framework                             |

---

## Related Documentation

- [Template Architect Guide](./README.template-architect-guide.md) - How to write detection templates
- [Agent Commands Reference](./AGENT-COMMANDS-REFERENCE.md) - CLI command documentation
- [Risk Scoring](./RISK-SCORING.md) - Vulnerability severity and scoring
- [PRD](./agent_template_system_PRD.md) - Product requirements document

---

_Last Updated: November 2025_
_Version: 2.0.0_


