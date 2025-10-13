# Code Deprecation Analysis - Template System MVP

## Purpose

This document identifies existing code that will become obsolete during the template system refactor. The goal is to systematically replace or remove these components during implementation, not immediately, but as we build the new architecture.

---

## Critical Principle

**Do not delete code until replacement is functional.** The existing code is our working proof-of-concept and serves as a valuable reference. We will deprecate functionality progressively as new modules are implemented.

---

## Code Deprecation Matrix

### 🔴 COMPLETE REPLACEMENT - Remove During Implementation

Code that will be entirely replaced by new architecture:

#### 1. Script Execution System

**Location**: `internal/detect/script/`

**Files to Remove**:

- `script/executor.go` (~528 lines) - Platform-specific script execution
- `script/bash.go` - Bash script handling
- `script/powershell.go` - PowerShell script handling
- `script/metadata.go` - Script metadata extraction
- `script/repository.go` - Script repository management

**Replaced By**:

- Three core modules (FileHash, FileContent, CommandVersion)
- Post-MVP: Embedded Lua module (not Bash/PowerShell/Python)

**Deprecation Timeline**:

- **Phase 3**: After FileHash module is working
- **Phase 5**: After FileContent module is working
- **Phase 9**: After CommandVersion module is working
- **Phase 11**: Final removal during cleanup

**Rationale**:

- Current implementation uses system interpreters (not cross-platform)
- Security concerns with Bash/PowerShell/Python execution
- No sandboxing capability
- Three core modules replace 90% of script use cases
- Future Lua module will be properly sandboxed

---

#### 2. Old Template System

**Location**: `internal/detect/template/`

**Files to Remove**:

- `template/parser.go` - Old YAML parser
- `template/executor.go` - Old template executor
- `template/discovery.go` - Old template discovery
- `template/valkey_adapter.go` - Valkey integration (may defer)
- `template/valkey_schema.go` - Valkey schema (may defer)

**Replaced By**:

- New `internal/template/parser/` - New YAML parser
- New `internal/template/executor/` - New template executor with worker pool
- New `internal/template/types/` - Updated type system

**Deprecation Timeline**:

- **Phase 2**: After new parser is implemented
- **Phase 4**: After new executor is implemented
- **Phase 7**: After worker pool is implemented
- **Phase 11**: Final removal during cleanup

**Rationale**:

- Old schema doesn't support AND/OR logic
- No step-based execution model
- No worker pool parallelism
- Doesn't support per-step platform filtering
- Type system doesn't match new YAML schema

---

#### 3. Old Type System

**Location**: `internal/detect/types.go`

**Types to Remove**:

```go
// Lines 7-26: Old VulnTemplate struct
type VulnTemplate struct { ... }

// Lines 55-77: Old DetectionConfig struct
type DetectionConfig struct { ... }

// Lines 79-89: Old DetectionType constants
const DetectionTypeFileHash DetectionType = "file-hash"

// Lines 91-104: Old DetectionTarget struct
type DetectionTarget struct { ... }

// Lines 199-242: DetectionScript struct (script system)
type DetectionScript struct { ... }

// Lines 244-252: ScriptLanguage constants
type ScriptLanguage string
const (
    ScriptLanguagePowerShell ScriptLanguage = "powershell"
    ScriptLanguageBash       ScriptLanguage = "bash"
    ScriptLanguagePython     ScriptLanguage = "python"
)
```

**Replaced By**:

- New `internal/template/types/types.go` - New type system
- New `internal/modules/types.go` - Module interfaces

**Deprecation Timeline**:

- **Phase 1**: After new type system is defined
- **Phase 2**: After parser uses new types
- **Phase 11**: Final removal of old types

**Keep These Types** (they're still useful):

```go
// Lines 272-280: HashAlgorithm constants (KEEP - still relevant)
type HashAlgorithm string
const (
    HashAlgorithmSHA256 HashAlgorithm = "sha256"
    HashAlgorithmSHA1   HashAlgorithm = "sha1"
    HashAlgorithmMD5    HashAlgorithm = "md5"
    HashAlgorithmSHA512 HashAlgorithm = "sha512"
)
```

---

#### 4. Old Scan Command

**Location**: `internal/commands/scan/scan_command.go`

**What to Refactor**:

- Lines 1-100: Current scan logic heavily tied to old system
- Script execution integration (lines ~50-100)
- Old template execution (lines ~86-150)
- Fingerprinting integration (may move to common/)

**Replaced By**:

- New template execution through `template run-all`
- Direct module execution (no complex scan orchestration for MVP)
- Simplified scan focused on template execution

**Deprecation Timeline**:

- **Phase 6**: After CLI with Cobra is implemented
- **Phase 8**: After result output format is complete
- **Phase 11**: Refactor or remove during cleanup

**Decision Needed**:

- Do we keep a `scan` command in MVP, or just use `template run-all`?
- If we keep it, dramatically simplify it

---

### 🟡 PARTIAL REPLACEMENT - Refactor During Implementation

Code that has useful parts but needs restructuring:

#### 5. Hash Detection

**Location**: `internal/detect/hash/`

**Files**:

- `hash/calculator.go` - Hash calculation logic
- `hash/types.go` - Hash types

**Action**:

- **MOVE** to `internal/common/files/hash.go`
- **SIMPLIFY** - Remove complex types
- **KEEP** core hash calculation functions

**Timeline**: Phase 1 (Shared Libraries - Files)

---

#### 6. Fingerprinting Utilities

**Location**: `internal/fingerprint/`

**Files**:

- `fingerprint/system.go` - OS detection
- `fingerprint/network.go` - Network info
- `fingerprint/ports.go` - Port scanning
- `fingerprint/services.go` - Service detection
- `fingerprint/certificates.go` - SSL cert info
- `fingerprint/users.go` - User enumeration

**Action**:

- **EVALUATE** which utilities are useful
- **MOVE** OS detection to `internal/common/os/`
- **DEFER** network/port/service scanning (not MVP)
- **DEFER** certificate analysis (not MVP)
- **DEFER** user enumeration (not MVP)

**Timeline**: Phase 1 (evaluate), Phase 11 (cleanup unused)

---

#### 7. Old Template Commands

**Location**: `internal/commands/template/`

**Files**:

- `template/discover_command.go` - Template discovery
- `template/list_command.go` - Template listing

**Action**:

- **REFERENCE** for new CLI implementation
- **REPLACE** with Cobra-based commands
- **KEEP** discovery logic patterns

**Timeline**: Phase 6 (CLI implementation)

---

### 🟢 EVALUATE FOR MVP - May Keep or Defer

Code that may or may not be needed for MVP:

#### 8. Repository Integration

**Location**: `internal/repository/`

**Files**:

- `repository/github_manager.go` - GitHub template downloads
- `repository/integration.go` - Repository integration
- `repository/interfaces.go` - Repository interfaces
- `repository/types.go` - Repository types

**Decision Needed**:

- **Option A**: Defer entirely to post-MVP (templates are local files)
- **Option B**: Keep for downloading template repositories
- **Option C**: Remove and implement differently later

**Recommendation**: **Defer to post-MVP**. Start with local templates only.

**Timeline**: Phase 11 (final decision)

---

#### 9. Server-Side Components

**Location**: `internal/server/`

**Files**:

- `server/server.go` - gRPC server
- `server/custom_content_consumer.go` - Custom content sync
- `server/custom_storage.go` - Custom storage
- `server/script_sync.go` - Script syncing
- `server/template_sync.go` - Template syncing
- `server/script_valkey_store.go` - Valkey storage

**Decision Needed**:

- **MVP Focus**: Standalone mode only
- **Future**: Server mode for centralized management

**Action**: **Keep for now**, but don't use in MVP implementation

**Timeline**: No changes during MVP

---

#### 10. Store/Database Integration

**Location**: `internal/store/`

**Files**:

- `store/store.go` - Database operations
- `store/adapter.go` - Database adapter
- `store/config.go` - Store configuration
- `store/response_store.go` - Response storage

**Decision Needed**:

- Do we need to store results in MVP?
- Or just output JSON/JSONL to stdout?

**Recommendation**: **Defer to post-MVP**. Output to stdout only for MVP.

**Timeline**: Phase 11 (final decision)

---

## Deprecation Strategy

### Phase-by-Phase Removal Plan

#### Phase 0: Cleanup (Week 1)

- Delete test commands
- Remove POC files
- Archive old documentation
- **No code deprecation yet**

#### Phase 1: Core Architecture (Week 1)

- Implement new type system
- Mark old types as deprecated (code comments)
- Begin moving hash utilities to common/

#### Phase 2-3: First Module (Week 2)

- Implement new template parser (don't delete old one yet)
- Implement FileHash module
- **Can now stop using** `internal/detect/hash/` (but don't delete)

#### Phase 4-5: Second Module (Week 2-3)

- Implement template executor
- Implement FileContent module
- **Can now stop using** old template executor (but don't delete)

#### Phase 6: CLI (Week 3)

- Implement Cobra CLI
- New template commands replace old ones
- Mark old commands as deprecated

#### Phase 7-9: Complete Modules (Week 4)

- Implement worker pool
- Implement CommandVersion module
- **All three MVP modules functional**
- Can now deprecate script system entirely

#### Phase 10-11: Polish & Cleanup (Week 5)

- **NOW SAFE TO DELETE**:
  - `internal/detect/script/` (entire directory)
  - `internal/detect/template/` (entire directory)
  - Old types from `internal/detect/types.go`
  - Old command implementations
- Refactor remaining code
- Update imports
- Run `go mod tidy`

---

## Safe Deprecation Checklist

Before removing ANY code:

- [ ] ✅ Replacement functionality is implemented
- [ ] ✅ Replacement is tested and working
- [ ] ✅ All imports are updated to new code
- [ ] ✅ No other code depends on deprecated code
- [ ] ✅ Tests pass without deprecated code
- [ ] ✅ Documentation updated
- [ ] ✅ Git commit with clear message

---

## Code Deprecation Guidelines

### 1. Mark as Deprecated First

Add deprecation comments before removing:

```go
// DEPRECATED: This will be removed in favor of internal/template/parser
// Use template.ParseTemplate() instead
// Removal scheduled for Phase 11
func OldParseTemplate(...) { ... }
```

### 2. Progressive Replacement

```bash
# Week 1-2: New code alongside old code
internal/
├── detect/
│   ├── template/  # OLD - still here
│   └── types.go   # OLD - still here
├── template/      # NEW - being built
└── modules/       # NEW - being built

# Week 5-6: Old code removed
internal/
├── template/      # NEW - fully functional
└── modules/       # NEW - fully functional
```

### 3. Import Cleanup

After removing deprecated code:

```bash
# Find all imports of deprecated packages
grep -r "internal/detect/script" internal/

# Update imports to new locations
# Run go mod tidy
go mod tidy

# Verify compilation
go build ./...
```

---

## Risk Mitigation

### Backup Strategy

Before removing any significant code:

```bash
# Create deprecation snapshot branch
git checkout -b snapshot/pre-deprecation
git add -A
git commit -m "snapshot: code before deprecation cleanup"
git checkout feature/template-system-mvp

# Now safe to remove code on feature branch
```

### Gradual Removal

Don't remove everything at once:

1. **Week 4**: Remove script system (after all modules work)
2. **Week 5**: Remove old template system (after new system works)
3. **Week 5**: Remove old types (after all code updated)
4. **Week 6**: Final cleanup

### Verification After Removal

```bash
# After each removal, verify:
go build ./...               # Compiles
go test ./...                # Tests pass
cd testing && make test-all  # Integration tests pass
```

---

## Documentation Updates

When deprecating code, update these docs:

- [ ] `README.md` - Remove references to old features
- [ ] `project/PLAN.agent-template-system-implementation.md` - Note completion
- [ ] `documentation/agent_template_system_PRD.md` - Update if needed
- [ ] Module documentation - Ensure new modules are documented
- [ ] API documentation - Update for new interfaces

---

## Summary

### Total Lines to Remove: ~3,000+ lines

**Major Removals**:

- Script execution system: ~1,500 lines
- Old template system: ~800 lines
- Old type system: ~300 lines
- Old scan command: ~400 lines
- Miscellaneous: ~500 lines

**Timeline**:

- Weeks 1-4: Build new system alongside old
- Week 5: Remove deprecated code
- Week 6: Final cleanup and verification

**Safety**:

- Never remove code until replacement works
- Mark as deprecated before removing
- Test after each removal
- Keep git history (don't squash these commits)

---

**This deprecation plan ensures we maintain working code throughout implementation while systematically replacing obsolete functionality.**
