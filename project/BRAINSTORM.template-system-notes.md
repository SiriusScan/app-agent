# Template System MVP - Brainstorming Notes

## Sprint Information

- **Sprint Name**: `template-system-mvp`
- **Duration Estimate**: 4-6 weeks
- **Focus**: Core MVP modules + standalone mode + containerized testing

## Project Approach

- **Code Strategy**: Refactor aggressively - treat existing code as POC/reference, not sacred
- **Testing Philosophy**: Integration/outcome-focused testing, not unit test coverage
- **Container Setup**: Get testing environment ready immediately, test in parallel with development
- **Dependencies**: No module-to-module dependencies - modules use shared libraries only
- **Error Handling**: Log errors, avoid panics in production code

---

## 1. Module Registry Pattern

### Decision: Struct-Based Registration (Idiomatic Go Pattern)

**Research Summary:**

- Analyzed Go standard library patterns (database/sql, image, crypto packages)
- Compared builder pattern vs struct-based registration
- Builder pattern rejected as un-idiomatic for Go
- Struct-based registration matches Go stdlib conventions

**Implementation Pattern:**

```go
// Module interface - minimal, just execution
type Module interface {
    Execute(ctx context.Context, config StepConfig) (*Result, error)
}

// Descriptor holds module metadata
type Descriptor struct {
    Type        string            // YAML type field (required)
    Name        string            // Human-readable name (required)
    Description string            // What it does (required)
    Version     string            // Semantic version (optional)
    Author      string            // Who wrote it (optional)
    SupportedOS []string          // Empty = all OS (optional)
    ConfigDocs  map[string]string // Config field documentation (optional)
}

// Registration in init()
func init() {
    modules.Register(&MyModule{}, modules.Descriptor{
        Type:        "my_module",
        Name:        "My Module",
        Description: "What this module does",
        // ... other fields
    })
}
```

**Key Design Decisions:**

- ✅ Single method interface: `Execute()` only
- ✅ Validation happens at template parsing, not per-module
- ✅ Thread-safe registry with `sync.RWMutex`
- ✅ Registration errors are logged, not panicked
- ✅ Registration in `init()` follows Go stdlib pattern
- ✅ Descriptor struct for metadata (not builder pattern)

**Shared Libraries Structure:**

```
internal/
  common/           # Shared utilities for modules
    files/          # File operations (read, hash, exists, permissions)
    os/             # OS detection and utilities
    patterns/       # Regex and pattern matching
    results/        # Result builders and evidence formatting
    errors/         # Standard error types
```

**Module Philosophy:**

- Modules should be thin wrappers around shared libraries
- Modules extract config, call shared libs, build results
- Shared libraries handle OS-specific logic, error patterns, safety checks
- No dependencies between modules (only on shared libraries)

**Third-Party Developer Experience:**

1. Copy existing module as template
2. Implement single `Execute()` method
3. Fill in Descriptor struct
4. Register in `init()`
5. Module is done

**Documentation Required:**

- [ ] Module Development Guide (how to create a module)
- [ ] Module Interface Reference (Execute method, StepConfig, Result types)
- [ ] Shared Libraries API Documentation (what utilities are available)
- [ ] Module Registry Documentation (how registration works)
- [ ] Example Module Walkthrough (annotated real example)
- [ ] Third-Party Module Guidelines (best practices, dos/don'ts)

---

## 2. CLI Structure for Standalone Mode

### Decision: Cobra Library with Subcommands

**Research Summary:**

- Analyzed industry standard Go CLI tools (Docker, Kubernetes, Helm, GitHub CLI)
- All major Go CLI tools use Cobra library for subcommand structure
- Standard library `flag` package suitable only for simple single-purpose tools
- Cobra provides: subcommands, automatic help, future shell completion support

**CLI Library Choice:**

- **Library**: Cobra (github.com/spf13/cobra)
- **Rationale**: Industry standard, only external dependency for CLI, universally known by Go developers
- **Trade-off**: One external dependency vs significant functionality gain

**Command Structure:**

```bash
# Server mode (default - backward compatible)
./agent                           # Reads SERVER_ADDRESS, AGENT_ID from env

# Explicit server mode
./agent server [--address=:50051]

# Standalone template operations
./agent template run <path-or-id>         # Run single template
./agent template run-all <directory>      # Run all in directory
./agent template list                     # List available templates
./agent template validate <path>          # Validate syntax
./agent template info <id>                # Show template details

# Module introspection
./agent module list                       # List registered modules
./agent module info <type>                # Show module details

# Global flags (before subcommand)
./agent --log-level debug template run file.yaml
./agent --output json template run file.yaml
```

**Template Resolution Priority:**

1. **File path check**: If argument is valid filesystem path (absolute or relative) → use it
2. **Template ID lookup**: Search `~/.sirius/templates/<id>.yaml`
3. **Error**: Template not found

**Examples:**

```bash
./agent template run /absolute/path/template.yaml       # Filesystem
./agent template run ./relative/path/template.yaml      # Filesystem
./agent template run openssh-cve-2024-1117              # ID lookup
./agent template run-all /path/to/templates/            # Directory
```

**Output Defaults:**

- **Default format**: JSON to stdout (machine-readable, pipeable)
- **Text format**: `--format text` for human-readable output
- **File output**: `--output filename` to save results to file
- **Both**: Can combine `--output` and stdout

**Examples:**

```bash
./agent template run file.yaml                          # JSON to stdout
./agent template run file.yaml --format text            # Human-readable
./agent template run file.yaml --output results.json    # Save to file
./agent template run file.yaml | jq .                   # Pipe to jq
```

**Backward Compatibility:**

- `./agent` with no args = server mode (daemon behavior)
- Maintains existing environment variable usage (SERVER_ADDRESS, AGENT_ID)
- Matches industry patterns: Prometheus, Telegraf, etc.

**Key Design Decisions:**

- ✅ Cobra library for subcommands (industry standard)
- ✅ No `-f` flag needed - positional arguments for file paths
- ✅ Intelligent template resolution (filesystem → ID lookup)
- ✅ JSON stdout default (pipeable, machine-friendly)
- ✅ Default `./agent` behavior unchanged (server mode)

**Documentation Required:**

- [ ] CLI Usage Guide (all commands and flags)
- [ ] Template Resolution Documentation (how paths/IDs are found)
- [ ] Output Format Reference (JSON schema, text format)
- [ ] Migration Guide (if changing existing behavior)
- [ ] Examples and Common Use Cases
- [ ] Shell Completion Setup Instructions

---

## 3. Test Container Setup

### Decision: Container as Linux Execution Environment

**Development Context:**

- Primary development on macOS
- Target platform: Linux (Ubuntu 22.04 LTS)
- Need fast iteration without container rebuilds
- Container is execution environment, NOT development environment

**Philosophy:**

- Code stays on macOS (edit in IDE)
- Cross-compile for Linux (`GOOS=linux GOARCH=amd64`)
- Mount compiled binary + test files into container
- Run agent in Linux environment
- No container rebuild needed for code changes

**Container Setup:**

```dockerfile
# Dockerfile.linux - Minimal runtime environment
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y ca-certificates
WORKDIR /workspace
CMD ["/bin/bash"]
```

**Docker Compose Configuration:**

```yaml
services:
  linux-test:
    build:
      context: testing
      dockerfile: Dockerfile.linux
    volumes:
      - ./bin:/workspace/bin:ro # Mount compiled binary
      - ./testing:/workspace/testing:ro # Mount test files/templates
    working_dir: /workspace
```

**Directory Structure:**

```
testing/
  Dockerfile.linux         # Minimal Ubuntu container
  Makefile                 # Build and test commands
  test-data/               # Fake vulnerable files for testing
    vulnerable-sshd        # Binary with known hash
    vulnerable-config.conf # Config with vulnerable pattern
    vulnerable-script.sh   # Script with security issue
  test-templates/          # Templates that detect above files
    01-file-hash.yaml      # FileHashModule test
    02-config-file.yaml    # FileContentModule test
    03-script-exec.yaml    # ScriptModule test
    04-multi-step.yaml     # Multiple detection steps
```

**Development Workflow:**

```bash
# Fast iteration loop (on macOS)
cd testing
make build-linux         # Cross-compile for Linux
make quick               # Run one test template

# Edit code on macOS
make quick               # Rebuild and test again

# Interactive debugging
make shell               # Drop into Linux container
./bin/agent template run testing/test-templates/01-file-hash.yaml
```

**Key Makefile Targets:**

- `build-linux` - Cross-compile agent for Linux
- `quick` - Build and run single test (fastest iteration)
- `test-template TEMPLATE=name.yaml` - Run specific template
- `test-all` - Run all test templates
- `shell` - Interactive Linux shell for debugging
- `clean` - Remove build artifacts

**Testing Approach:**

- **Milestone-based**: Each template tests a specific capability
- **Outcome-focused**: Does it run? Does output look right?
- **No complex assertions**: Visual inspection of JSON output
- **Fast feedback**: See results immediately after code changes

**Integration Test Pattern:**

1. Create fake vulnerable file in `test-data/`
2. Write template that should detect it
3. Build agent: `make build-linux`
4. Run template: `make test-template TEMPLATE=01-file-hash.yaml`
5. Check JSON output - did it detect the vulnerability?
6. If not, debug, fix code, repeat from step 3

**Key Design Decisions:**

- ✅ Container is execution environment only
- ✅ Cross-compile on macOS (no Go in container)
- ✅ Mount binaries and test files (read-only)
- ✅ No container rebuilds during iteration
- ✅ Simple Makefile for all operations
- ✅ Ubuntu 22.04 LTS as target platform
- ✅ Test data: fake vulnerable files for detection

**Documentation Required:**

- [ ] Container Setup Guide (how to get started)
- [ ] Cross-Compilation Instructions (GOOS/GOARCH)
- [ ] Makefile Target Reference (all commands explained)
- [ ] Test Data Creation Guide (how to add new test files)
- [ ] Template Testing Guide (how to write test templates)
- [ ] Debugging in Container (interactive workflow)

---

## 4. Template Execution Results Format

### Decision: Hybrid Format with JSONL for Multiple Results

**Result Structure: Hybrid (Structured + Context)**

Single template result format:

```json
{
  "template_id": "openssh-cve-2024-1117",
  "template_name": "OpenSSH 6.5.1 RCE",
  "severity": "critical",
  "matched": true,
  "confidence": 1.0,
  "timestamp": "2025-10-13T10:30:45Z",
  "host": {
    "os": "linux",
    "version": "Ubuntu 22.04",
    "agent_id": "agent-001"
  },
  "detection_steps": [
    {
      "step": 1,
      "type": "file_hash",
      "matched": true,
      "path": "/usr/sbin/sshd",
      "evidence": {
        "expected_hash": "4f5f9...",
        "actual_hash": "4f5f9...",
        "algorithm": "sha256"
      }
    },
    {
      "step": 2,
      "type": "file_content",
      "matched": true,
      "path": "/usr/sbin/sshd",
      "evidence": {
        "pattern": "OpenSSH_6\\.5\\.1",
        "matched_line": 42,
        "matched_text": "OpenSSH_6.5.1p1"
      }
    }
  ],
  "summary": {
    "total_steps": 4,
    "matched_steps": 2,
    "failed_steps": 0,
    "execution_time_ms": 125
  },
  "errors": []
}
```

**Multiple Templates: JSONL (Industry Standard)**

Default output format for multiple templates (JSONL - JSON Lines):

```json
{"template_id":"openssh-cve-2024-1117","matched":true,...}
{"template_id":"apache-cve-2021-44228","matched":false,...}
{"template_id":"nginx-misconfiguration","matched":true,...}
```

**Rationale for JSONL:**

- Streaming results (see output as templates complete)
- Parseable line-by-line (`cat output.json | jq .`)
- Resilient to interruption (partial results valid)
- No memory overhead (process line-by-line)
- Industry standard (Nuclei, Trivy, security tools)

**Alternative Format: JSON Array**

```bash
./agent template run-all dir/ --format json-array
```

Output:

```json
{
  "scan_id": "scan-12345",
  "timestamp": "2025-10-13T10:30:45Z",
  "total_templates": 10,
  "vulnerable_count": 3,
  "results": [...]
}
```

**Text Format (Human-Readable)**

```
[CRITICAL] OpenSSH 6.5.1 RCE (openssh-cve-2024-1117)
Status: VULNERABLE
Confidence: 100%

Detection Steps:
  ✓ Step 1: file_hash - MATCHED
    Path: /usr/sbin/sshd
    Hash: 4f5f9... (sha256)

  ✓ Step 2: file_content - MATCHED
    Path: /usr/sbin/sshd
    Pattern: OpenSSH_6\.5\.1
    Line 42: OpenSSH_6.5.1p1

Summary:
  Matched: 2/4 steps
  Execution time: 125ms

---
```

**Error Handling Approach:**

Errors are **separate from matching** for better analytics:

```json
{
  "template_id": "test-template",
  "matched": false,
  "confidence": 0.0,
  "detection_steps": [
    {
      "step": 1,
      "type": "file_hash",
      "matched": false,
      "error": "permission denied: /usr/sbin/sshd"
    }
  ],
  "errors": [
    {
      "step": 1,
      "type": "permission_error",
      "message": "Cannot read file: /usr/sbin/sshd",
      "details": "permission denied"
    }
  ]
}
```

**Key Distinction:**

- `matched: false` = template did NOT detect vulnerability
- `errors: [...]` = something went wrong during execution
- Step can have both: `matched: false` AND `error: "..."`

**Output Flags:**

- `--format json` (default) - JSONL for multiple, JSON for single
- `--format json-array` - Wrapped JSON array with summary
- `--format text` - Human-readable output
- `--output <file>` - Save to file (otherwise stdout)

**Key Design Decisions:**

- ✅ Hybrid format (structure + context + evidence)
- ✅ JSONL default for multiple templates (streaming)
- ✅ JSON array option for wrapped results
- ✅ Errors separate from matching (analytics)
- ✅ Per-step evidence for debugging
- ✅ Text format for human consumption

**Documentation Required:**

- [ ] Result Format Reference (JSON schema)
- [ ] JSONL vs JSON Array explanation
- [ ] Error Handling Guide (error types and meanings)
- [ ] Evidence Structure Documentation
- [ ] Output Format Examples (all formats)
- [ ] Parsing Results Guide (how to consume output)

---

## 5. YAML Template Schema Design

### Decision: Simple AND Logic with Optional Weights (MVP)

**MVP Philosophy:**

- Keep schema simple and focused
- Support 90% of use cases with minimal complexity
- Design for future enhancement without refactors
- Let template authors control confidence/weights

**Core YAML Schema:**

```yaml
id: openssh-cve-2024-1117
info:
  name: OpenSSH 6.5.1 RCE
  author: security-team
  severity: critical # critical, high, medium, low, info
  description: Detects OpenSSH version 6.5.1 vulnerable to CVE-2024-1117
  tags: [openssh, version, cve-2024-1117, rce]
  cve: CVE-2024-1117
  references:
    - https://nvd.nist.gov/vuln/detail/CVE-2024-1117

detection:
  logic: all # optional: "all" (AND - default) or "any" (OR)
  steps:
    - type: file_hash
      platforms: [linux, darwin] # optional: skip on other OS
      weight: 0.8 # optional: confidence weight (0.0-1.0)
      path: /usr/sbin/sshd
      hash: "4f5f9..."
      algorithm: sha256

    - type: file_content
      platforms: [linux, darwin]
      weight: 0.3
      path: /usr/sbin/sshd
      regex: "OpenSSH_6\\.5\\.1"

test_strategy: |
  Manual verification steps:
  1. Run `ssh -V` on the host
  2. Verify version string contains "OpenSSH_6.5.1"

remediation: |
  Update OpenSSH to latest patched version:
  - Linux: apt-get update && apt-get install openssh-server
  - macOS: brew upgrade openssh
```

**Detection Logic (MVP):**

- **Default: AND** - All steps must match
- **Optional: OR** - Any step matching = vulnerable
- **Future**: Complex expressions like `(step1 AND step2) OR step3`

**Platform Filtering:**

- **Per-step** - Each step can specify `platforms: [linux, darwin]`
- **Empty/omitted** - Runs on all platforms
- **Template-level filtering** - Future enhancement if needed

**Confidence Scoring:**

- **Template author assigns weights** - Each step has optional `weight: 0.0-1.0`
- **Default weight: 1.0** - If not specified
- **AND logic**: Confidence = min(matched weights) or average
- **OR logic**: Confidence = max(matched weights)
- **Template authors control** - Can adjust by editing weights

**Module-Specific Fields:**
Each module type uses different config fields:

```yaml
# FileHashModule
- type: file_hash
  path: /path/to/file
  hash: "abc123..."
  algorithm: sha256 # optional: sha256 (default), sha1, md5, sha512

# FileContentModule
- type: file_content
  path: /path/to/file
  regex: "pattern"
  # Future: string, strings (literal matches)

# CommandVersionModule
- type: command_version
  command: ["ssh", "-V"]
  regex: "OpenSSH_6\\.5\\.1"
  exit_code: 0 # optional: expected exit code

# ScriptModule
- type: script
  interpreter: bash # bash, python, powershell
  script: |
    #!/bin/bash
    dpkg-query -W -f='${Version}' openssh-server
  regex: "^6.5.1"
  exit_code: 0 # optional
```

**Field Documentation:**

- Each module registers with `ConfigDocs` explaining its fields
- Template validation checks required fields per module type
- `agent module info <type>` shows available fields

**Required Fields (All Templates):**

- ✅ `id` - Unique template identifier
- ✅ `info.name` - Human-readable name
- ✅ `info.severity` - critical|high|medium|low|info
- ✅ `info.description` - What this detects
- ✅ `detection.steps` - At least one detection step

**Optional Fields:**

- `info.author` - Template author
- `info.tags` - Classification tags
- `info.cve` - CVE identifier
- `info.references` - URLs for more info
- `detection.logic` - "all" (default) or "any"
- `detection.steps[].platforms` - OS filter
- `detection.steps[].weight` - Confidence weight
- `test_strategy` - Manual verification steps
- `remediation` - Fix instructions

**MVP Scope:**

- ✅ AND/OR logic only (no complex expressions)
- ✅ Per-step platform filtering
- ✅ Template author weights
- ✅ Four core modules (FileHash, FileContent, CommandVersion, Script)
- ✅ Basic field validation
- ❌ Conditional logic (if/then)
- ❌ Variable substitution
- ❌ Step dependencies
- ❌ Dynamic step generation

**Future Enhancements (Post-MVP):**

- Complex logic expressions: `(step1 AND step2) OR step3`
- Template variables: `${OS}`, `${ARCH}`
- Conditional steps: `if: platform == linux`
- Step results as inputs: Use step1 output in step2
- Template inheritance: Extend base templates

**Validation:**

- Parser validates YAML syntax
- Required fields checked
- Module type must exist in registry
- Module-specific fields validated against ConfigDocs
- Platform names validated (linux, darwin, windows)
- Weight values must be 0.0-1.0
- Severity must be valid value

**Key Design Decisions:**

- ✅ Simple AND/OR logic (MVP)
- ✅ Per-step platform filtering
- ✅ Template author assigns weights
- ✅ Module ConfigDocs for field documentation
- ✅ Extensible schema (add fields without breaking)
- ✅ Focus on 90% of use cases
- ✅ Future enhancements planned but not implemented

**Documentation Required:**

- [ ] Template Schema Reference (all fields explained)
- [ ] Module Field Reference (per-module config options)
- [ ] Template Writing Guide (how to create templates)
- [ ] Template Examples (common patterns)
- [ ] Validation Rules Documentation
- [ ] Best Practices Guide (weight assignment, platform filtering)

---

## 6. Module Execution Model

### Decision: Sequential Steps, Parallel Templates (Worker Pool)

**Critical Distinction:**

- **Single template**: Steps execute sequentially (one after another)
- **Multiple templates**: Templates execute in parallel (worker pool)

**Rationale:**

- Individual template speed is not critical
- But running 10,000+ templates sequentially would take forever
- Worker pool provides controlled parallelism and resource management

**Single Template Execution (Sequential Steps):**

```
Template: openssh-cve-2024-1117
├── Step 1: file_hash      → Executes (100ms)
├── Step 2: file_content   → Executes (50ms)
├── Step 3: command_version → Executes (200ms)
└── Step 4: script         → Executes (500ms)
Total: ~850ms
```

**Multiple Templates Execution (Parallel with Worker Pool):**

```
Worker Pool (10 concurrent workers)
├── Worker 1: template-001 (steps run sequentially)
├── Worker 2: template-002 (steps run sequentially)
├── Worker 3: template-003 (steps run sequentially)
├── ...
└── Worker 10: template-010 (steps run sequentially)

Queue: templates 11-10,000 waiting
```

**Step Execution Model (Within Single Template):**

```go
// Pseudocode for single template execution
func ExecuteTemplate(template Template) Result {
    // 1. Parse and validate
    if err := ValidateTemplate(template); err != nil {
        return ErrorResult(err)
    }

    // 2. Filter steps by current platform
    steps := FilterStepsByPlatform(template.Detection.Steps, runtime.GOOS)

    // 3. Execute steps sequentially
    stepResults := []StepResult{}
    for i, step := range steps {
        // Get module for this step type
        module := registry.Get(step.Type)

        // Execute with timeout (30s default per step)
        ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
        result := module.Execute(ctx, step.Config)
        cancel()

        stepResults = append(stepResults, result)

        // Continue even if step errored (collect all results)
    }

    // 4. Evaluate detection logic (AND/OR)
    matched := EvaluateLogic(template.Detection.Logic, stepResults)

    // 5. Calculate confidence from weights
    confidence := CalculateConfidence(stepResults, template.Detection.Steps)

    // 6. Build final result
    return BuildResult(template, matched, confidence, stepResults)
}
```

**Template Pool Execution Model (Multiple Templates):**

```go
// Pseudocode for template pool execution
func ExecuteTemplates(templates []Template, workerCount int) []Result {
    // Create worker pool
    jobs := make(chan Template, len(templates))
    results := make(chan Result, len(templates))

    // Start worker goroutines
    for w := 1; w <= workerCount; w++ {
        go worker(w, jobs, results)
    }

    // Send templates to workers
    for _, template := range templates {
        jobs <- template
    }
    close(jobs)

    // Collect results
    allResults := []Result{}
    for i := 0; i < len(templates); i++ {
        result := <-results
        allResults = append(allResults, result)
    }

    return allResults
}

func worker(id int, jobs <-chan Template, results chan<- Result) {
    for template := range jobs {
        // Execute template (steps run sequentially)
        result := ExecuteTemplate(template)
        results <- result
    }
}
```

**Worker Pool Configuration:**

- **Default workers**: `runtime.NumCPU()` (use all available CPU cores)
- **Configurable**: `--workers 20` flag to adjust
- **Max workers**: 50 (safety limit to prevent resource exhaustion)
- **Queue size**: Unbuffered (backpressure if workers are busy)

**Execution Flow:**

```
agent template run-all /templates/

1. Discover all templates in directory (10,000 templates)
2. Validate templates (parallel validation)
3. Create worker pool (10 workers)
4. Queue templates to workers
5. Each worker:
   - Takes next template from queue
   - Executes steps sequentially
   - Returns result
   - Takes next template
6. Collect all results as JSONL stream
7. Exit when all templates processed
```

**Timeouts:**

- **Per-step**: 30 seconds (prevents stuck operations)
- **Per-template**: 5 minutes total (prevents runaway templates)
- **Global**: None (let it run until all templates complete)

**Failure Handling:**

- **Step failure**: Continue to next step, log error
- **Template failure**: Return error result, don't crash worker
- **Worker panic**: Recover, log error, worker continues
- **Overall**: One bad template doesn't affect others

**Progress Reporting (Optional Enhancement):**

```
Scanning with 10 workers...
Progress: 1,247 / 10,000 templates (12.47%)
Vulnerable: 23 detected
Errors: 5 templates
Time elapsed: 2m 15s
```

**Platform Filtering:**

- Happens **after** template is pulled from queue
- Each worker filters steps based on `runtime.GOOS`
- Templates with zero applicable steps are skipped quickly

**Key Design Decisions:**

- ✅ Sequential step execution within template (simple, debuggable)
- ✅ Parallel template execution with worker pool (scalable)
- ✅ Worker count = CPU count (good default)
- ✅ Configurable worker count (flexibility)
- ✅ Continue-on-error within template (complete picture)
- ✅ Recover from panics (resilient workers)
- ✅ JSONL streaming output (see results as they complete)
- ✅ Per-step and per-template timeouts (prevent hangs)

**Scalability:**

- **10 templates**: ~1 second (sequential would be fine)
- **100 templates**: ~10 seconds (worker pool shows benefit)
- **1,000 templates**: ~2 minutes (significant improvement)
- **10,000 templates**: ~20 minutes (vs hours sequentially)

**Documentation Required:**

- [ ] Worker Pool Architecture Documentation
- [ ] Performance Tuning Guide (worker count optimization)
- [ ] Timeout Configuration Guide
- [ ] Error Handling and Recovery Documentation
- [ ] Progress Reporting Implementation (if added)

---

## 7. MVP Module Implementations

### Decision: Three Core Modules + Script Module Deferred

**MVP Modules (Linux-First):**

1. **FileHashModule** - Hash-based file detection
2. **FileContentModule** - Regex/pattern matching in files
3. **CommandVersionModule** - Version extraction from command output

**Post-MVP Enhancement:** 4. **ScriptModule** - Embedded Lua scripting (deferred)

### Rationale for Deferring ScriptModule:

- Cross-platform scripting requires embedded interpreter (not system interpreters)
- Current POC uses Bash/PowerShell/Python - not truly cross-platform
- Embedded Lua (via gopher-lua) is the right solution
- Three core modules cover 90% of use cases for MVP
- Post-MVP will implement Lua with proper sandboxing

### Documentation Required:

- [ ] FileHashModule Reference
- [ ] FileContentModule Reference
- [ ] CommandVersionModule Reference
- [ ] Module Development Guide
- [ ] Shared Libraries API Documentation
- [ ] Script Module Plan (post-MVP roadmap)

---

## Summary: Ready for Implementation

All major architectural decisions have been made:

1. ✅ Module Registry Pattern (struct-based, idiomatic Go)
2. ✅ CLI Structure (Cobra with subcommands)
3. ✅ Test Container Setup (Linux execution environment)
4. ✅ Results Format (Hybrid JSON with JSONL streaming)
5. ✅ YAML Schema (Simple AND/OR logic, template author weights)
6. ✅ Execution Model (Sequential steps, parallel templates)
7. ✅ MVP Modules (Three core modules, Script deferred)

**Next Step**: Create exhaustive task list → Developer handoff

---

## Final Decisions Summary

### Outstanding Questions - RESOLVED:

1. **Cobra Dependency**: ✅ APPROVED - Use Cobra for CLI
2. **Server Mode**: ✅ KEEP - Will need updates post-MVP, but defer for now
3. **Repository Integration**: ✅ IMPORTANT - Templates will sync from server in production, but use local files for MVP testing
4. **Testing Approach**: ✅ CONFIRMED - Integration tests, no code coverage requirements
5. **Platform Support**: ✅ LINUX-FIRST - Strictly Linux for MVP, macOS/Windows post-MVP

### Scope Clarification:

**MVP Focus:**

- Standalone mode (local template execution)
- Linux-only (Ubuntu 22.04 LTS)
- Local template files for testing
- Three core modules

**Post-MVP:**

- Server mode integration (template sync)
- Template repository sync mechanism
- macOS support
- Windows support
- Embedded Lua script module

### Ready for Implementation: ✅

All architectural decisions finalized. Task list created. Developer handoff prepared.
