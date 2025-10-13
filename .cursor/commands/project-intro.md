# Sirius Agent - Template-Based Vulnerability Detection System

## Developer Onboarding Guide

Welcome to the Sirius Agent project! This document will get you up to speed quickly.

---

## What is This?

**Sirius Agent** is a cross-platform vulnerability detection agent that uses YAML-based templates to identify security vulnerabilities on host systems. Think of it as "Nuclei for host-side vulnerability detection" rather than network scanning.

### Key Concepts:

1. **Templates** (YAML files) define what to look for
2. **Modules** (Go code) perform the actual detection
3. **Agent** executes templates and reports results
4. **Worker Pool** runs thousands of templates in parallel

---

## Quick Start

### Prerequisites:

- Go 1.24+ installed
- Docker installed (for Linux testing)
- Basic understanding of YAML and Go

### Get Running in 5 Minutes:

```bash
# 1. Clone and navigate
cd app-agent

# 2. Build the agent
GOOS=linux GOARCH=amd64 go build -o bin/agent cmd/agent/main.go

# 3. Run a test template (in Linux container)
cd testing
make build-linux
make quick

# You should see JSON output with detection results
```

---

## Project Architecture

### High-Level Flow:

```
Templates (YAML)
    ↓
Agent parses templates
    ↓
Worker pool (parallel execution)
    ↓
Each template → Sequential detection steps
    ↓
Modules execute (FileHash, FileContent, etc.)
    ↓
Results (JSON/JSONL output)
```

### Directory Structure:

```
app-agent/
├── cmd/agent/                  # Main entry point
├── internal/
│   ├── modules/                # Detection modules
│   │   ├── registry/           # Module registration system
│   │   ├── filehash/           # Hash-based detection
│   │   ├── filecontent/        # Content/regex detection
│   │   └── versioncmd/         # Command version detection
│   ├── common/                 # Shared libraries
│   │   ├── files/              # File operations
│   │   ├── os/                 # OS detection
│   │   ├── patterns/           # Regex matching
│   │   └── results/            # Result builders
│   ├── template/               # Template parsing & execution
│   └── commands/               # CLI commands (Cobra)
├── testing/                    # Container-based testing
│   ├── test-data/              # Fake vulnerable files
│   └── test-templates/         # Test templates
└── templates/                  # Example templates
```

---

## Core Components

### 1. Templates (YAML)

Templates define WHAT to detect:

```yaml
id: openssh-cve-2024-1117
info:
  name: OpenSSH 6.5.1 RCE
  severity: critical
  description: Detects vulnerable OpenSSH version

detection:
  logic: all # AND logic (all steps must match)
  steps:
    - type: file_hash
      path: /usr/sbin/sshd
      hash: "4f5f9..."
      weight: 0.8

    - type: file_content
      path: /usr/sbin/sshd
      regex: "OpenSSH_6\\.5\\.1"
      weight: 0.3
```

### 2. Modules (Go)

Modules define HOW to detect:

```go
// FileHashModule calculates and compares file hashes
type FileHashModule struct{}

func (m *FileHashModule) Execute(ctx context.Context, config StepConfig) (*Result, error) {
    // Extract config
    path := config.GetString("path")
    expectedHash := config.GetString("hash")

    // Calculate hash using shared library
    actualHash, err := files.CalculateHash(path, "sha256")

    // Build and return result
    return results.Build(matched, confidence, evidence), nil
}
```

### 3. CLI Commands (Cobra)

User interface for the agent:

```bash
# Standalone mode - run templates
./agent template run /path/to/template.yaml
./agent template run-all /templates/

# Introspection
./agent module list
./agent module info file_hash

# Server mode (connects to gRPC server)
./agent server
```

---

## Development Workflow

### 1. Working on macOS, Targeting Linux

```bash
# Edit code on macOS (your IDE)
vim internal/modules/filehash/filehash.go

# Cross-compile for Linux
cd testing
make build-linux

# Run in Linux container
make quick

# See results immediately
```

### 2. Adding a New Module

```go
// 1. Create module directory
internal/modules/mymodule/mymodule.go

// 2. Implement Module interface
type MyModule struct{}

func (m *MyModule) Execute(ctx context.Context, config StepConfig) (*Result, error) {
    // Your detection logic here
    return result, nil
}

// 3. Register in init()
func init() {
    modules.Register(&MyModule{}, modules.Descriptor{
        Type:        "my_module",
        Name:        "My Module",
        Description: "What this module does",
        SupportedOS: []string{"linux", "darwin"},
    })
}
```

### 3. Testing Your Module

```yaml
# Create test template
# testing/test-templates/test-my-module.yaml
id: test-my-module
info:
  name: Test My Module
  severity: info

detection:
  steps:
    - type: my_module
      # your config fields
```

```bash
# Run the test
cd testing
make build-linux
make test-template TEMPLATE=test-my-module.yaml
```

---

## Key Design Principles

### 1. **MVP Focus**

- Simple AND/OR logic (no complex expressions)
- Four core modules (FileHash, FileContent, VersionCmd, [Script deferred])
- Linux-first (macOS/Windows later)
- Outcome-focused testing (not coverage metrics)

### 2. **Modularity**

- Each module is self-contained
- Modules use shared libraries (DRY principle)
- No module-to-module dependencies
- Easy to add new modules

### 3. **Scalability**

- Worker pool for parallel template execution
- Sequential steps within templates
- 10,000+ templates should complete in ~20 minutes

### 4. **Developer Experience**

- Fast iteration (no container rebuilds)
- Clear error messages
- Simple testing workflow
- Good documentation

---

## Common Tasks

### Run a Single Template:

```bash
./agent template run /path/to/template.yaml
```

### Run All Templates in Directory:

```bash
./agent template run-all /templates/
# Outputs JSONL (one JSON object per line)
```

### List Registered Modules:

```bash
./agent module list
```

### Get Module Details:

```bash
./agent module info file_hash
# Shows: description, supported OS, config fields
```

### Validate Template Syntax:

```bash
./agent template validate /path/to/template.yaml
```

---

## Testing Philosophy

We focus on **integration testing** (does it work end-to-end?) rather than unit testing (does each function work?).

### Test Pattern:

1. Create fake vulnerable file in `testing/test-data/`
2. Write template that should detect it
3. Run template in Linux container
4. Check JSON output - did it detect correctly?
5. If not, debug, fix, repeat

**No complex test frameworks. Just run it and see if it works.**

---

## Documentation Structure

### For Users:

- `README.md` - Project overview
- `documentation/agent_template_system_PRD.md` - Product requirements
- Template writing guides (TBD)

### For Developers:

- `project/PROJECT-INTRO.md` - This file (onboarding)
- `project/BRAINSTORM.template-system-notes.md` - Design decisions
- `project/PLAN.agent-template-system-implementation.md` - Implementation plan
- Module-specific READMEs in each module directory

---

## FAQ

**Q: Why templates instead of code?**
A: Security engineers can write templates without knowing Go. Templates are portable, shareable, and version-controlled separately from the agent.

**Q: Why not just use Nuclei?**
A: Nuclei is for network scanning. We need host-side detection (file hashes, installed packages, running services, etc.)

**Q: Why Go?**
A: Cross-platform, fast, good concurrency, single binary distribution, widely used in security tools.

**Q: Why Linux-first?**
A: Most servers run Linux. Get it working there first, then expand to macOS/Windows.

**Q: Where's the Script module?**
A: Deferred to post-MVP. Will use embedded Lua for true cross-platform scripting.

**Q: How do I add a new detection type?**
A: Create a new module implementing the Module interface, register it in init(), document its config fields.

**Q: Where do templates come from?**
A: Security team writes them, community contributes them, they're version-controlled in a template repository (separate from agent code).

---

## Getting Help

1. **Read the brainstorming notes**: `project/BRAINSTORM.template-system-notes.md`
2. **Check the implementation plan**: `project/PLAN.agent-template-system-implementation.md`
3. **Look at existing modules**: `internal/modules/*/` for examples
4. **Ask questions**: Tag @security-team in your questions

---

## Next Steps

1. Read the [Implementation Plan](../project/PLAN.agent-template-system-implementation.md)
2. Review [Brainstorming Notes](../project/BRAINSTORM.template-system-notes.md) for design rationale
3. Set up your development environment (Docker, Go)
4. Pick a task from `tasks/template-system-mvp.json`
5. Start coding!

---

**Welcome to the team! Let's build something great.** 🚀
