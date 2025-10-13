# Project Directory Cleanup Analysis

## Current State Assessment

The `app-agent` project has evolved from a proof-of-concept into a production-ready template-based vulnerability detection system. The current directory structure contains significant POC/test code that should be cleaned up before beginning the MVP implementation.

## Analysis Categories

### 🟢 KEEP - Core Production Code

**Essential Directories:**
```
cmd/agent/              # Main agent entry point - KEEP
internal/agent/         # Core agent logic - KEEP & REFACTOR
internal/config/        # Configuration management - KEEP
internal/shell/         # Shell execution utilities - KEEP
internal/sysinfo/       # System information gathering - KEEP
proto/hello/            # gRPC protocol definitions - KEEP
```

**Essential Files:**
```
go.mod, go.sum          # Dependencies - KEEP
README.md               # Main documentation - UPDATE
documentation/agent_template_system_PRD.md  # Project PRD - KEEP & UPDATE
project/BRAINSTORM.template-system-notes.md # Design decisions - KEEP
```

---

### 🟡 REFACTOR - Needs Restructuring

**internal/detect/** - Has good foundations but needs major refactoring:
- `detect/template/` - Template parsing logic (use as reference, rewrite)
- `detect/script/` - Script execution (reference only, will use Lua post-MVP)
- `detect/hash/` - Hash utilities (move to `internal/common/files/`)
- `detect/types.go` - Type definitions (update for new schema)

**internal/commands/** - Command registry pattern is good:
- `commands/command.go` - KEEP (good pattern)
- `commands/registry.go` - KEEP (matches our module design)
- `commands/scan/` - REFACTOR heavily (old scan logic)
- `commands/template/` - Use as reference for new implementation
- `commands/repo/`, `commands/sync/`, `commands/status/` - Evaluate individually

**internal/fingerprint/** - System fingerprinting utilities:
- May move to `internal/common/system/` for use by modules
- Good utilities for OS detection, network info, etc.

**internal/repository/** - GitHub template distribution:
- May keep for template repository management
- Evaluate if needed for MVP

**internal/server/** - Server-side components:
- Evaluate if needed for MVP (we're focusing on agent standalone mode)
- May defer to post-MVP

**internal/store/** - Valkey/database integration:
- Evaluate if needed for MVP
- May defer to post-MVP

---

### 🔴 DELETE - POC/Test Code

**Test Command Directories (DELETE ALL):**
```
cmd/command-receiver/       # Test code
cmd/command-sender/         # Test code
cmd/repo-test/              # Test code
cmd/template-test/          # Test code
cmd/test-github-download/   # Test code
cmd/test-grpc-client/       # Test code
cmd/test-repo/              # Test code
cmd/test-repo-cli/          # Test code
cmd/test-repo-cli-live/     # Test code
cmd/test-repo-commands/     # Test code
cmd/test-repo-integration/  # Test code (empty)
cmd/test-scan/              # Test code
cmd/test-template-discovery/# Test code
```

**Root-Level Clutter (DELETE):**
```
agent                   # Old compiled binary
agent-cpe               # Old binary
agent-windows.exe       # Old binary
main                    # Old binary
test-grpc-client        # Old binary
agent.log               # Log file
server_commands.log     # Log file
custom-manifest.json    # POC file
manifest.json           # POC file
custom-scripts/         # POC test scripts
custom-templates/       # POC templates
dev-exec.sh             # Dev script
template-discovery-test.json  # Test file
test_message.json       # Test file
tmp/                    # Temporary files
tests/                  # Empty directory
sirius-agent-modules/   # Empty directory
```

**Bin Directory (DELETE):**
```
bin/                    # All old compiled binaries
```

**Outdated Documentation (DELETE/ARCHIVE):**
```
docs/PRD-Phase1.md           # Superseded by new PRD
docs/queue/                  # Old queue implementation docs
docs/SCRIPT-DISTRIBUTION-PLAN.md  # Old planning
docs/TEMPLATE-DEV-GUIDE.md   # Will be rewritten
DEVELOPMENT.md               # Outdated
PROJECT-STRUCTURE.md         # Outdated
README-agent-server.md       # Outdated
```

**Old Project Documentation (ARCHIVE):**
```
project/docs/                # Archive all old docs
project/tasks/               # Archive old task tracking
```

---

## Cleanup Strategy

### Phase 1: Safe Removal (Immediate)
Files that are definitively not needed:

```bash
# Delete test commands
rm -rf cmd/command-receiver cmd/command-sender cmd/repo-test
rm -rf cmd/template-test cmd/test-*

# Delete old binaries
rm -rf bin/
rm -f agent agent-cpe agent-windows.exe main test-grpc-client

# Delete log files
rm -f *.log

# Delete POC files
rm -f custom-manifest.json manifest.json
rm -rf custom-scripts/ custom-templates/
rm -f dev-exec.sh template-discovery-test.json test_message.json

# Delete empty/temp directories
rm -rf tmp/ tests/ sirius-agent-modules/

# Archive old documentation
mkdir -p project/archive/docs
mv docs/* project/archive/docs/
mv project/docs/* project/archive/docs/
rm -rf docs/
rmdir project/docs/

# Archive old tasks
mkdir -p project/archive/tasks
mv project/tasks/* project/archive/tasks/
rmdir project/tasks/
```

### Phase 2: Evaluation & Refactoring
Review and decide on these during implementation:

**Evaluate for MVP:**
- `internal/server/` - Server-side code (needed for MVP?)
- `internal/store/` - Database integration (needed for MVP?)
- `internal/repository/` - GitHub integration (needed for MVP?)
- `internal/fingerprint/` - System utilities (integrate into common/)
- `cmd/server/` - Server binary (needed for MVP?)

**Refactor & Restructure:**
- `internal/detect/` → Reference for new `internal/modules/` and `internal/template/`
- `internal/commands/scan/` → Integrate into new template execution
- Old type definitions → Update for new YAML schema

### Phase 3: New Structure Creation
Create directories for new architecture:

```bash
mkdir -p internal/modules/filehash
mkdir -p internal/modules/filecontent
mkdir -p internal/modules/versioncmd
mkdir -p internal/common/files
mkdir -p internal/common/os
mkdir -p internal/common/patterns
mkdir -p internal/common/results
mkdir -p internal/common/errors
mkdir -p internal/template
mkdir -p testing/test-data
mkdir -p testing/test-templates
mkdir -p templates/examples
```

---

## Post-Cleanup Directory Structure

### Desired Final Structure:
```
app-agent/
├── cmd/
│   ├── agent/              # Main agent binary
│   └── server/             # Server binary (if needed)
├── internal/
│   ├── agent/              # Core agent logic (refactored)
│   ├── config/             # Configuration management
│   ├── modules/            # NEW: Detection modules
│   │   ├── registry/       # Module registry
│   │   ├── filehash/       # FileHashModule
│   │   ├── filecontent/    # FileContentModule
│   │   └── versioncmd/     # CommandVersionModule
│   ├── common/             # NEW: Shared libraries
│   │   ├── files/          # File operations
│   │   ├── os/             # OS detection
│   │   ├── patterns/       # Regex matching
│   │   ├── results/        # Result builders
│   │   └── errors/         # Error types
│   ├── template/           # NEW: Template parsing & execution
│   │   ├── parser/         # YAML parser
│   │   ├── executor/       # Template executor
│   │   └── types/          # Template types
│   ├── commands/           # Command registry (existing pattern)
│   ├── shell/              # Shell utilities
│   ├── sysinfo/            # System info
│   └── [evaluate others]   # Review server/, store/, repository/
├── testing/
│   ├── Dockerfile.linux    # Test container
│   ├── Makefile            # Test commands
│   ├── test-data/          # Fake vulnerable files
│   └── test-templates/     # Test templates
├── templates/
│   └── examples/           # Example templates
├── documentation/
│   └── agent_template_system_PRD.md
├── project/
│   ├── BRAINSTORM.template-system-notes.md
│   ├── PLAN.agent-template-system-implementation.md
│   ├── PROJECT-INTRO.md
│   └── archive/            # Archived old docs
├── proto/
│   └── hello/              # gRPC definitions
├── go.mod
├── go.sum
└── README.md

REMOVED:
- All cmd/test-* directories
- All bin/ binaries
- All root-level test files
- Old docs/ directory
- tmp/, tests/ empty directories
```

---

## Cleanup Checklist

Before implementation begins:

- [ ] **Phase 1**: Delete all test commands and binaries
- [ ] **Phase 1**: Remove POC files and log files
- [ ] **Phase 1**: Archive old documentation
- [ ] **Phase 1**: Clean up root directory clutter
- [ ] **Phase 2**: Evaluate server/, store/, repository/ for MVP
- [ ] **Phase 2**: Review and refactor internal/detect/ code
- [ ] **Phase 2**: Update go.mod dependencies
- [ ] **Phase 3**: Create new directory structure
- [ ] **Phase 3**: Stub out new packages with README files
- [ ] **Phase 3**: Update main README.md with new project overview
- [ ] **Verification**: Run `go mod tidy` and ensure project compiles
- [ ] **Verification**: No broken imports remain
- [ ] **Verification**: Git status shows intentional changes only

---

## Risk Mitigation

**Before deleting anything:**
1. ✅ Create feature branch: `git checkout -b cleanup/project-restructure`
2. ✅ Commit current state: `git add -A && git commit -m "snapshot before cleanup"`
3. ✅ Verify nothing critical in test commands (manual review)
4. ✅ Backup any potentially useful code snippets to project/archive/

**During cleanup:**
- Review each directory before deletion
- Keep git history (don't force push)
- Document any discoveries in cleanup notes
- If unsure about code, move to project/archive/ instead of deleting

---

## Timeline Estimate

- **Phase 1 (Safe Removal)**: 30 minutes
- **Phase 2 (Evaluation)**: 2 hours (review code, make decisions)
- **Phase 3 (New Structure)**: 1 hour
- **Verification**: 30 minutes

**Total**: ~4 hours of cleanup before implementation begins

---

## Success Criteria

Cleanup is complete when:
- ✅ No test commands remain in cmd/
- ✅ Root directory contains only production files
- ✅ Old documentation is archived, not deleted
- ✅ New directory structure is stubbed out
- ✅ Project compiles without errors
- ✅ README.md accurately reflects new structure
- ✅ Git history is clean and intentional

