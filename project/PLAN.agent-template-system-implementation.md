# Template System MVP - Implementation Plan

## Project Overview

**Sprint Name**: `template-system-mvp`  
**Duration**: 4-6 weeks  
**Branch**: `feature/template-system-mvp`  
**Target**: Linux-first, cross-platform template-based vulnerability detection

---

## Implementation Strategy

### Approach:
1. **Clean first, then build** - Remove POC code, establish clean structure
2. **Foundation before features** - Core architecture before modules
3. **Vertical slices** - Complete one module end-to-end before starting next
4. **Test continuously** - Verify each milestone in Linux container
5. **Document as we go** - Update docs with each completed phase

### Success Criteria:
- ✅ Agent runs templates in standalone mode
- ✅ Four core modules functional (FileHash, FileContent, VersionCmd, + Module registry)
- ✅ Worker pool executes 1,000+ templates efficiently
- ✅ JSON/JSONL output format working
- ✅ Container-based testing workflow established
- ✅ Example templates demonstrate all capabilities

---

## Phase 0: Project Cleanup & Foundation (Week 1, Days 1-2)

**Goal**: Clean project structure, establish development environment

### Tasks:

#### 0.1: Repository Cleanup
- [ ] Create feature branch: `feature/template-system-mvp`
- [ ] Snapshot current state (commit)
- [ ] Delete test commands (`cmd/test-*`, `cmd/command-*`, etc.)
- [ ] Remove old binaries (`bin/`, root-level binaries)
- [ ] Clean POC files (custom-scripts, custom-templates, test files)
- [ ] Archive old documentation (`docs/` → `project/archive/docs/`)
- [ ] Remove empty directories (`tmp/`, `tests/`, `sirius-agent-modules/`)
- [ ] Commit: "chore: clean up POC code and test artifacts"

**Test Strategy**: `git status` shows only intentional deletions

#### 0.2: Directory Structure Setup
- [ ] Create `internal/modules/registry/`
- [ ] Create `internal/common/{files,os,patterns,results,errors}/`
- [ ] Create `internal/template/{parser,executor,types}/`
- [ ] Create `testing/{test-data,test-templates}/`
- [ ] Create `templates/examples/`
- [ ] Add README.md stub in each new directory
- [ ] Commit: "chore: create new directory structure for template system"

**Test Strategy**: All new directories exist with README stubs

#### 0.3: Container Development Environment
- [ ] Create `testing/Dockerfile.linux` (Ubuntu 22.04)
- [ ] Create `testing/docker-compose.dev.yaml`
- [ ] Create `testing/Makefile` with targets:
  - `build-linux` - Cross-compile agent
  - `shell` - Interactive container
  - `quick` - Build + run one test
  - `test-template` - Run specific template
  - `clean` - Remove build artifacts
- [ ] Test: Build container, compile agent, run in container
- [ ] Commit: "feat: add container-based development environment"

**Test Strategy**: `make build-linux && make shell` works, agent binary runs in container

---

## Phase 1: Core Architecture (Week 1, Days 3-5 + Weekend)

**Goal**: Module registry, type system, shared libraries foundation

### Tasks:

#### 1.1: Module Registry System
- [ ] Define `Module` interface in `internal/modules/types.go`
- [ ] Define `Descriptor` struct for module metadata
- [ ] Implement module registry in `internal/modules/registry/registry.go`
- [ ] Add `Register()`, `Get()`, `List()` functions
- [ ] Add thread-safe registry with `sync.RWMutex`
- [ ] Add registration validation (check required fields)
- [ ] Write registration tests
- [ ] Commit: "feat: implement module registry system"

**Test Strategy**: Register dummy module, retrieve it, list all modules

#### 1.2: Template Type System
- [ ] Define template types in `internal/template/types/types.go`:
  - `Template` struct
  - `TemplateInfo` struct
  - `DetectionConfig` struct
  - `DetectionStep` struct
  - `Result` struct
  - `StepResult` struct
- [ ] Define severity levels (critical, high, medium, low, info)
- [ ] Define platform types (linux, darwin, windows)
- [ ] Add JSON/YAML tags for serialization
- [ ] Commit: "feat: define template type system"

**Test Strategy**: Create test templates, marshal/unmarshal JSON/YAML

#### 1.3: Shared Libraries - Files
- [ ] Implement `internal/common/files/read.go` - Safe file reading
- [ ] Implement `internal/common/files/hash.go` - Hash calculation (SHA256, SHA1, MD5, SHA512)
- [ ] Implement `internal/common/files/exists.go` - File existence checks
- [ ] Add size limits, timeout handling
- [ ] Add unit tests for each function
- [ ] Commit: "feat: implement file operations shared library"

**Test Strategy**: Create test files, verify hash calculations, test size limits

#### 1.4: Shared Libraries - Patterns & Results
- [ ] Implement `internal/common/patterns/match.go` - Regex matching
- [ ] Implement `internal/common/results/builder.go` - Result construction
- [ ] Implement `internal/common/errors/types.go` - Standard error types
- [ ] Add helper functions for evidence formatting
- [ ] Add tests
- [ ] Commit: "feat: implement patterns and results shared libraries"

**Test Strategy**: Test regex matching, result building with various evidence types

---

## Phase 2: Template Parser (Week 2, Days 1-3)

**Goal**: Parse and validate YAML templates

### Tasks:

#### 2.1: YAML Parser
- [ ] Implement `internal/template/parser/parser.go`
- [ ] Add `ParseTemplate(path string) (*Template, error)`
- [ ] Add `ParseTemplateBytes(data []byte) (*Template, error)`
- [ ] Handle YAML unmarshaling
- [ ] Set metadata (FilePath, LoadedAt)
- [ ] Add error handling for malformed YAML
- [ ] Commit: "feat: implement template YAML parser"

**Test Strategy**: Parse valid and invalid YAML templates

#### 2.2: Template Validation
- [ ] Implement `ValidateTemplate(t *Template) error`
- [ ] Validate required fields (id, info.name, info.severity, detection.steps)
- [ ] Validate severity levels
- [ ] Validate platform names
- [ ] Validate weight values (0.0-1.0)
- [ ] Validate detection logic ("all" or "any")
- [ ] Add comprehensive validation tests
- [ ] Commit: "feat: implement template validation"

**Test Strategy**: Validate correct and incorrect templates, check error messages

#### 2.3: Template Discovery
- [ ] Implement `DiscoverTemplates(dir string) ([]*Template, error)`
- [ ] Walk directory recursively
- [ ] Find all .yaml/.yml files
- [ ] Parse and validate each template
- [ ] Collect errors but continue processing
- [ ] Return all valid templates + error list
- [ ] Add context timeouts for large directories
- [ ] Commit: "feat: implement template discovery"

**Test Strategy**: Create directory with mix of valid/invalid templates, verify discovery

---

## Phase 3: FileHash Module (Week 2, Days 4-5)

**Goal**: First complete module implementation (vertical slice)

### Tasks:

#### 3.1: FileHash Module Implementation
- [ ] Create `internal/modules/filehash/filehash.go`
- [ ] Implement `FileHashModule` struct
- [ ] Implement `Execute(ctx context.Context, config StepConfig) (*Result, error)`
- [ ] Extract config: path, hash, algorithm
- [ ] Calculate hash using `common/files`
- [ ] Compare hashes
- [ ] Build result with evidence
- [ ] Handle errors (file not found, permission denied)
- [ ] Add timeout handling
- [ ] Commit: "feat: implement FileHash module"

**Test Strategy**: Create test file, calculate hash, verify detection

#### 3.2: FileHash Module Registration
- [ ] Add `init()` function to register module
- [ ] Define module descriptor:
  - Type: "file_hash"
  - Name: "File Hash Validator"
  - Description: "..."
  - SupportedOS: [linux, darwin, windows]
  - ConfigDocs: {path, hash, algorithm}
- [ ] Commit: "feat: register FileHash module"

**Test Strategy**: List modules, verify FileHash appears with correct metadata

#### 3.3: FileHash Integration Test
- [ ] Create test file in `testing/test-data/vulnerable-sshd`
- [ ] Calculate known hash
- [ ] Create template in `testing/test-templates/01-file-hash.yaml`
- [ ] Build agent for Linux
- [ ] Run template in container
- [ ] Verify JSON output shows detection
- [ ] Commit: "test: add FileHash integration test"

**Test Strategy**: `make test-template TEMPLATE=01-file-hash.yaml` succeeds

---

## Phase 4: Template Executor (Week 3, Days 1-2)

**Goal**: Execute templates and produce results

### Tasks:

#### 4.1: Single Template Executor
- [ ] Implement `internal/template/executor/executor.go`
- [ ] Implement `ExecuteTemplate(ctx context.Context, template *Template) (*Result, error)`
- [ ] Filter steps by current platform
- [ ] Execute steps sequentially
- [ ] Get module from registry for each step
- [ ] Call module.Execute() with timeout (30s per step)
- [ ] Collect step results
- [ ] Continue on error (don't stop)
- [ ] Evaluate detection logic (AND/OR)
- [ ] Calculate confidence from weights
- [ ] Build final result
- [ ] Commit: "feat: implement single template executor"

**Test Strategy**: Execute template with multiple steps, verify all run

#### 4.2: Logic Evaluation
- [ ] Implement `EvaluateLogic(logic string, steps []StepResult) bool`
- [ ] Handle "all" logic (AND - all must match)
- [ ] Handle "any" logic (OR - at least one matches)
- [ ] Default to "all" if not specified
- [ ] Add unit tests for both logic types
- [ ] Commit: "feat: implement detection logic evaluation"

**Test Strategy**: Test AND/OR logic with various step results

#### 4.3: Confidence Calculation
- [ ] Implement `CalculateConfidence(steps []StepResult, weights []float64) float64`
- [ ] AND logic: minimum of matched weights
- [ ] OR logic: maximum of matched weights
- [ ] Default weight: 1.0
- [ ] Handle zero matched steps
- [ ] Add unit tests
- [ ] Commit: "feat: implement confidence calculation"

**Test Strategy**: Test confidence with various weight combinations

---

## Phase 5: FileContent Module (Week 3, Days 3-4)

**Goal**: Second module implementation

### Tasks:

#### 5.1: FileContent Module Implementation
- [ ] Create `internal/modules/filecontent/filecontent.go`
- [ ] Implement `FileContentModule` struct
- [ ] Implement `Execute()` method
- [ ] Extract config: path, regex
- [ ] Read file using `common/files`
- [ ] Match regex using `common/patterns`
- [ ] Build result with evidence (matched text, line number)
- [ ] Handle errors gracefully
- [ ] Register module in `init()`
- [ ] Commit: "feat: implement FileContent module"

**Test Strategy**: Create test file with patterns, verify regex matching

#### 5.2: FileContent Integration Test
- [ ] Create test file with vulnerable config pattern
- [ ] Create template `testing/test-templates/02-file-content.yaml`
- [ ] Test in container
- [ ] Verify detection and evidence
- [ ] Commit: "test: add FileContent integration test"

**Test Strategy**: `make test-template TEMPLATE=02-file-content.yaml` succeeds

---

## Phase 6: CLI with Cobra (Week 3, Day 5 + Weekend)

**Goal**: User interface for template execution

### Tasks:

#### 6.1: Cobra Setup
- [ ] Add Cobra dependency: `go get github.com/spf13/cobra`
- [ ] Refactor `cmd/agent/main.go` to use Cobra
- [ ] Create root command
- [ ] Add global flags: `--log-level`, `--output`, `--format`
- [ ] Implement version command
- [ ] Commit: "feat: integrate Cobra CLI framework"

**Test Strategy**: `./agent --help` shows commands

#### 6.2: Template Commands
- [ ] Create `template` command group
- [ ] Implement `template run <path-or-id>` subcommand
- [ ] Implement `template run-all <directory>` subcommand
- [ ] Implement `template list` subcommand
- [ ] Implement `template validate <path>` subcommand
- [ ] Implement `template info <id>` subcommand
- [ ] Add template resolution (filesystem → ID lookup)
- [ ] Commit: "feat: implement template CLI commands"

**Test Strategy**: Run each command, verify output

#### 6.3: Module Commands
- [ ] Create `module` command group
- [ ] Implement `module list` subcommand
- [ ] Implement `module info <type>` subcommand
- [ ] Format output nicely (table format)
- [ ] Commit: "feat: implement module CLI commands"

**Test Strategy**: `./agent module list` shows all registered modules

---

## Phase 7: Worker Pool & Parallel Execution (Week 4, Days 1-2)

**Goal**: Execute thousands of templates efficiently

### Tasks:

#### 7.1: Worker Pool Implementation
- [ ] Create `internal/template/executor/pool.go`
- [ ] Implement `ExecuteTemplatesParallel(templates []*Template, workers int) []Result`
- [ ] Create job channel and result channel
- [ ] Start worker goroutines
- [ ] Queue templates to workers
- [ ] Each worker calls `ExecuteTemplate()` sequentially
- [ ] Collect results
- [ ] Add worker panic recovery
- [ ] Commit: "feat: implement worker pool for parallel execution"

**Test Strategy**: Run 100 templates, verify all execute

#### 7.2: Worker Pool Configuration
- [ ] Add `--workers` flag (default: `runtime.NumCPU()`)
- [ ] Add worker count validation (max: 50)
- [ ] Add per-template timeout (5 minutes)
- [ ] Add graceful shutdown on Ctrl+C
- [ ] Commit: "feat: add worker pool configuration"

**Test Strategy**: Test with different worker counts, verify parallelism

---

## Phase 8: Result Output Format (Week 4, Day 3)

**Goal**: JSON and JSONL output

### Tasks:

#### 8.1: JSON Output
- [ ] Implement result serialization to JSON
- [ ] Single template: Pretty-printed JSON
- [ ] Multiple templates: JSONL (one per line)
- [ ] Add `--format json-array` for wrapped array output
- [ ] Stream results as they complete (JSONL)
- [ ] Commit: "feat: implement JSON/JSONL output"

**Test Strategy**: Run templates, pipe to `jq`, verify parsing

#### 8.2: Text Output
- [ ] Implement `--format text` for human-readable output
- [ ] Color-code severity (critical=red, high=orange, etc.)
- [ ] Format evidence nicely
- [ ] Show summary (matched X/Y steps)
- [ ] Commit: "feat: implement text output format"

**Test Strategy**: Run templates with `--format text`, verify readability

---

## Phase 9: CommandVersion Module (Week 4, Days 4-5)

**Goal**: Third module implementation

### Tasks:

#### 9.1: CommandVersion Module
- [ ] Create `internal/modules/versioncmd/versioncmd.go`
- [ ] Implement `CommandVersionModule` struct
- [ ] Implement `Execute()` method
- [ ] Extract config: command (array), regex, exit_code
- [ ] Execute command with timeout
- [ ] Capture stdout/stderr
- [ ] Match version regex
- [ ] Verify exit code if specified
- [ ] Build result with evidence
- [ ] Register module
- [ ] Commit: "feat: implement CommandVersion module"

**Test Strategy**: Run version commands (`ssh -V`, etc.), verify parsing

#### 9.2: CommandVersion Integration Test
- [ ] Create template for SSH version detection
- [ ] Test in container
- [ ] Verify version extraction
- [ ] Commit: "test: add CommandVersion integration test"

**Test Strategy**: `make test-template TEMPLATE=03-version-cmd.yaml` succeeds

---

## Phase 10: Example Templates & Documentation (Week 5, Days 1-2)

**Goal**: Complete documentation and examples

### Tasks:

#### 10.1: Example Templates
- [ ] Create `templates/examples/openssh-vulnerable.yaml`
- [ ] Create `templates/examples/apache-misconfiguration.yaml`
- [ ] Create `templates/examples/nginx-version-detection.yaml`
- [ ] Create `templates/examples/multi-step-detection.yaml`
- [ ] Test all examples in container
- [ ] Commit: "docs: add example templates"

**Test Strategy**: All example templates execute successfully

#### 10.2: Template Writing Guide
- [ ] Write `documentation/TEMPLATE-WRITING-GUIDE.md`
- [ ] Explain YAML schema
- [ ] Document each module's config fields
- [ ] Provide examples for each module
- [ ] Explain AND/OR logic
- [ ] Explain weight/confidence scoring
- [ ] Commit: "docs: add template writing guide"

#### 10.3: Module Development Guide
- [ ] Write `documentation/MODULE-DEVELOPMENT-GUIDE.md`
- [ ] Explain Module interface
- [ ] Show how to register modules
- [ ] Document shared libraries
- [ ] Provide complete module example
- [ ] Commit: "docs: add module development guide"

#### 10.4: Update Main README
- [ ] Update `README.md` with new architecture
- [ ] Add quick start guide
- [ ] Add CLI usage examples
- [ ] Link to detailed documentation
- [ ] Commit: "docs: update main README for template system"

---

## Phase 11: Integration & Polish (Week 5, Days 3-5)

**Goal**: End-to-end testing and refinement

### Tasks:

#### 11.1: Comprehensive Integration Tests
- [ ] Create 20+ test templates covering all modules
- [ ] Create corresponding test data files
- [ ] Run full test suite: `make test-all`
- [ ] Verify all templates execute correctly
- [ ] Document any failures
- [ ] Fix bugs found during testing
- [ ] Commit: "test: add comprehensive integration test suite"

**Test Strategy**: `make test-all` runs all tests, reports pass/fail

#### 11.2: Performance Testing
- [ ] Generate 1,000 dummy templates
- [ ] Time execution with different worker counts
- [ ] Verify parallelism is working
- [ ] Optimize if needed
- [ ] Document performance characteristics
- [ ] Commit: "test: add performance benchmarks"

**Test Strategy**: 1,000 templates complete in < 5 minutes

#### 11.3: Error Handling Review
- [ ] Review all error paths
- [ ] Ensure graceful error messages
- [ ] Test error scenarios (bad templates, missing files, etc.)
- [ ] Verify errors don't crash agent
- [ ] Commit: "fix: improve error handling throughout"

#### 11.4: Code Cleanup
- [ ] Run `gofmt` on all files
- [ ] Run `go mod tidy`
- [ ] Remove debug logging
- [ ] Add package comments
- [ ] Review and clean up any TODOs
- [ ] Commit: "chore: code cleanup and formatting"

---

## Phase 12: Final Testing & Documentation (Week 6)

**Goal**: Production-ready release

### Tasks:

#### 12.1: Final Testing
- [ ] Fresh clone of repository
- [ ] Follow README from scratch
- [ ] Verify all commands work
- [ ] Run full test suite
- [ ] Test on clean Ubuntu 22.04 container
- [ ] Document any issues

#### 12.2: Documentation Review
- [ ] Review all documentation for accuracy
- [ ] Check all links work
- [ ] Verify code examples are correct
- [ ] Get documentation reviewed by another developer
- [ ] Make corrections

#### 12.3: Release Preparation
- [ ] Update CHANGELOG.md (if exists)
- [ ] Tag release: `v1.0.0-mvp`
- [ ] Create release notes
- [ ] Merge feature branch to main

---

## Milestones & Checkpoints

### Milestone 1: Foundation Complete (End of Week 1)
- ✅ Project cleaned up
- ✅ Directory structure established
- ✅ Container development environment working
- ✅ Module registry implemented
- ✅ Type system defined

### Milestone 2: First Module Working (End of Week 2)
- ✅ Template parser complete
- ✅ FileHash module functional
- ✅ Can execute single template
- ✅ JSON output working

### Milestone 3: CLI & Multiple Modules (End of Week 3)
- ✅ Cobra CLI implemented
- ✅ FileContent module working
- ✅ Template executor complete
- ✅ Integration tests passing

### Milestone 4: Production Features (End of Week 4)
- ✅ Worker pool functional
- ✅ CommandVersion module working
- ✅ All output formats implemented
- ✅ Performance acceptable

### Milestone 5: Complete & Documented (End of Week 5)
- ✅ All modules documented
- ✅ Example templates created
- ✅ Comprehensive tests passing
- ✅ Ready for wider testing

### Milestone 6: Production Ready (End of Week 6)
- ✅ Final testing complete
- ✅ Documentation reviewed
- ✅ Release prepared
- ✅ Merged to main

---

## Risk Management

### Technical Risks:

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Module interface too rigid | Medium | High | Keep interface minimal, use StepConfig map |
| Template schema changes needed | High | Medium | Version templates, support migration |
| Performance issues with 10K templates | Medium | High | Worker pool, test early with large sets |
| Cross-platform issues | Low | Medium | Focus on Linux first, defer others |

### Schedule Risks:

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Cleanup takes longer than expected | Medium | Low | Timebox to 1 day, defer if needed |
| Module complexity underestimated | Medium | Medium | Start with simplest (FileHash) first |
| Testing reveals major issues | Medium | High | Test continuously, catch issues early |
| Documentation takes too long | High | Low | Write docs as we implement |

---

## Success Criteria

MVP is complete when:

- [ ] Agent runs in standalone mode
- [ ] Three detection modules work (FileHash, FileContent, CommandVersion)
- [ ] Templates can use AND/OR logic
- [ ] Worker pool executes 1,000+ templates efficiently
- [ ] JSON/JSONL output format correct
- [ ] Container-based testing workflow established
- [ ] Example templates demonstrate all features
- [ ] Documentation complete and reviewed
- [ ] Integration tests pass
- [ ] Code merged to main branch

---

## Post-MVP Enhancements

Features to consider after MVP is complete:

1. **Embedded Lua scripting** - True cross-platform script module
2. **Template variables** - `${OS}`, `${ARCH}` substitution
3. **Complex logic expressions** - `(step1 AND step2) OR step3`
4. **macOS support** - Test and fix platform-specific issues
5. **Windows support** - Add Windows-specific modules
6. **Template repository integration** - Download templates from GitHub
7. **Server mode improvements** - gRPC communication enhancements
8. **Progress reporting** - Real-time status during scans
9. **Result caching** - Avoid re-running identical templates
10. **Template signing** - Verify template authenticity

---

**This plan is a living document. Update it as implementation progresses.**

