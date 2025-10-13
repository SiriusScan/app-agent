# Template Management System - Architecture Brainstorm

## Problem Statement

The agent currently hardcodes template paths (`/app-agent/templates`) which:
- ❌ Only works on Linux
- ❌ Requires root/admin permissions on macOS/Windows
- ❌ Assumes templates magically exist
- ❌ Has no template download/sync mechanism
- ❌ No cross-platform path handling

## Requirements

### 1. Cross-Platform Template Storage

**Platform-Specific Default Locations:**

```
Windows:
  System: C:\ProgramData\sirius-agent\templates
  User:   %APPDATA%\sirius-agent\templates
  
macOS:
  System: /Library/Application Support/sirius-agent/templates
  User:   ~/Library/Application Support/sirius-agent/templates
  
Linux:
  System: /var/lib/sirius-agent/templates or /etc/sirius-agent/templates
  User:   ~/.config/sirius-agent/templates or ~/.local/share/sirius-agent/templates
```

**Priority Order:**
1. User-specified (via flag: `--template-dir`)
2. Environment variable (`SIRIUS_TEMPLATE_DIR`)
3. Custom templates (user-installed)
4. Server-synced templates (from Sirius server)
5. Built-in templates (shipped with agent binary)

### 2. Template Sources

**Source Types:**

1. **Built-in Templates**
   - Embedded in binary (Go embed)
   - Always available, read-only
   - Version matches agent version
   - Examples: common CVEs, basic checks

2. **Custom Templates** 
   - User-created/imported
   - Highest priority
   - Mutable (user can edit)
   - Located in `custom/` subdirectory

3. **Server-Synced Templates**
   - Downloaded from Sirius server
   - Automatic sync on connect
   - Version-controlled
   - Located in `server/` subdirectory

4. **Repository Templates** (Future)
   - GitHub/Git repository
   - Community templates
   - Versioned releases
   - Located in `community/` subdirectory

### 3. Template Directory Structure

```
<template-base>/
├── builtin/              # Embedded templates (read-only)
│   ├── cve-2023-*.yaml
│   └── common/*.yaml
├── custom/               # User templates (highest priority)
│   └── my-template.yaml
├── server/               # Server-synced templates
│   ├── .sync-metadata    # Last sync time, versions
│   └── *.yaml
├── community/            # Repository templates (future)
│   └── *.yaml
└── cache/                # Temp/working directory
    └── *.yaml.tmp
```

### 4. Template Management Operations

**Core Operations:**

```go
type TemplateManager interface {
    // Discovery
    DiscoverTemplates(ctx context.Context) ([]*Template, error)
    GetTemplate(id string) (*Template, error)
    ListTemplates(source TemplateSource) ([]*TemplateInfo, error)
    
    // Sync
    SyncFromServer(ctx context.Context, serverAddr string) error
    SyncFromRepository(ctx context.Context, repoURL string) error
    
    // Management
    InstallTemplate(path string, source TemplateSource) error
    UpdateTemplate(id string) error
    RemoveTemplate(id string) error
    ValidateTemplate(path string) error
    
    // Storage
    GetStoragePath() string
    EnsureDirectories() error
}
```

### 5. Template Sync Protocol

**Server Sync Flow:**

```
1. Agent connects to server
2. Agent sends template manifest (hashes, versions)
3. Server compares with available templates
4. Server sends:
   - New templates (full YAML)
   - Updated templates (full YAML)
   - Deleted template IDs
5. Agent applies changes to server/ directory
6. Agent updates .sync-metadata
```

**Sync Metadata Format:**

```yaml
last_sync: "2024-01-15T10:30:00Z"
server_version: "1.2.3"
templates:
  - id: CVE-2023-1234
    version: "1.0.0"
    hash: sha256:abc123...
    last_updated: "2024-01-10T08:00:00Z"
  - id: CVE-2023-5678
    version: "2.1.0"
    hash: sha256:def456...
    last_updated: "2024-01-14T12:00:00Z"
```

## Design Decisions

### Decision 1: Template Storage Location

**Option A: Go-specific directories** (RECOMMENDED)
- Use `os.UserConfigDir()` / `os.UserCacheDir()`
- Cross-platform by default
- Respects OS conventions
- No admin required for user mode

```go
configDir, _ := os.UserConfigDir()
templateDir := filepath.Join(configDir, "sirius-agent", "templates")
```

**Option B: Custom per-platform logic**
- More control but more code
- Harder to maintain

**Decision: Option A** - Use Go standard library

### Decision 2: Built-in Templates

**Option A: Embed in binary** (RECOMMENDED)
```go
//go:embed templates/*.yaml
var embeddedTemplates embed.FS
```

**Option B: Ship as separate files**
- Requires installation step
- Can be lost/modified

**Decision: Option A** - Embed for reliability

### Decision 3: Template Priority/Precedence

When multiple sources have same template ID:

```
1. Custom (user-installed) - HIGHEST
2. Server-synced
3. Repository (community)
4. Built-in - LOWEST
```

Rationale: User's custom templates should always override

### Decision 4: Sync Mechanism

**Option A: On-demand sync** (RECOMMENDED for MVP)
- Agent syncs on:
  - Initial connection to server
  - Explicit `internal:sync-templates` command
  - Scheduled interval (e.g., every 6 hours)
- Simpler to implement
- User controls when sync happens

**Option B: Real-time sync**
- Templates pushed from server immediately
- More complex
- Requires bidirectional protocol

**Decision: Option A** - On-demand for MVP

### Decision 5: Template Validation

**When to validate:**
- On install/import
- On sync from server
- Before execution
- On-demand via CLI

**What to validate:**
- YAML syntax
- Schema compliance
- Required fields
- Module availability
- Circular dependencies (future)

## Implementation Plan

### Phase 1: Template Storage (NEW PHASE 7.6)

**7.6.1: Cross-Platform Path Resolution**
- Create `internal/template/storage/paths.go`
- Implement `GetTemplateBaseDir()` using `os.UserConfigDir()`
- Support environment variable override
- Support command-line flag override
- Create directory structure on first run

**7.6.2: Template Manager Core**
- Create `internal/template/storage/manager.go`
- Implement `TemplateManager` interface
- Template discovery across all sources
- Template loading with precedence
- Basic CRUD operations

**7.6.3: Built-in Templates**
- Embed test templates using `go:embed`
- Load from `builtin/` directory
- Make read-only
- Version tracking

**7.6.4: Update Template-Scan Command**
- Use TemplateManager instead of direct file access
- Remove hardcoded paths
- Support `--template-dir` flag
- Proper cross-platform operation

### Phase 2: Template Sync (FUTURE - After MVP)

**Server-Side Changes:**
- Template repository in server
- Template manifest API
- Template download API
- Version tracking

**Agent-Side Changes:**
- `internal:sync-templates` command
- Sync metadata management
- Incremental updates
- Conflict resolution

### Phase 3: Advanced Features (FUTURE)

- Repository sync (GitHub)
- Template versioning
- Dependency management
- Template marketplace
- Automatic updates

## Cross-Platform Considerations

### Windows Specifics:
- Use `\` path separator (handled by `filepath.Join`)
- Check for admin vs user permissions
- Handle `%APPDATA%` vs `%PROGRAMDATA%`
- Test on Windows Server vs Windows Desktop

### macOS Specifics:
- User templates: `~/Library/Application Support/`
- System templates: `/Library/Application Support/`
- Check for SIP (System Integrity Protection)
- Handle code signing requirements

### Linux Specifics:
- User templates: `~/.config/` or `~/.local/share/`
- System templates: `/var/lib/` or `/etc/`
- Check file permissions
- Handle systemd service deployments

## Testing Strategy

### Unit Tests:
- Path resolution on all platforms
- Template discovery
- Precedence logic
- Validation logic

### Integration Tests:
- Template manager with real files
- Cross-source template loading
- Conflict resolution
- Sync operations

### Platform Tests:
- Windows (PowerShell, cmd)
- macOS (Intel, ARM)
- Linux (Ubuntu, CentOS, Alpine)

## Migration Path

### For Existing Deployments:

1. Detect old hardcoded paths
2. Migrate templates to new location
3. Create symlinks for backward compatibility
4. Log migration warnings
5. Remove old paths after grace period

## Configuration Options

### Environment Variables:
```bash
SIRIUS_TEMPLATE_DIR=/custom/path/templates
SIRIUS_TEMPLATE_SYNC_INTERVAL=3600  # seconds
SIRIUS_TEMPLATE_BUILTIN_ENABLED=true
```

### Config File (future):
```yaml
templates:
  base_dir: /custom/path
  sources:
    builtin: true
    server: true
    community: false
  sync:
    enabled: true
    interval: 3600
    auto_sync_on_connect: true
```

### Command Flags:
```bash
sirius-agent server --template-dir /custom/path
sirius-agent template run-all --template-dir /custom/path
```

## Security Considerations

1. **Template Verification:**
   - Validate templates before execution
   - Check for malicious patterns
   - Sandbox template parsing

2. **Source Trust:**
   - Built-in: Fully trusted (signed binary)
   - Server: Trusted (authenticated connection)
   - Custom: User responsibility
   - Community: Needs verification

3. **File Permissions:**
   - Templates should be readable only
   - Writable only by owner
   - No executable permissions

4. **Path Traversal:**
   - Validate all paths
   - Prevent `../` attacks
   - Sanitize file names

## Success Metrics

- ✅ Agent works on Windows without admin
- ✅ Agent works on macOS without sudo
- ✅ Templates discoverable across all platforms
- ✅ Custom templates override built-in
- ✅ Template sync completes in <5 seconds
- ✅ No hardcoded platform-specific paths
- ✅ Backward compatible with existing deployments

## Open Questions

1. **Q:** Should we support multiple template directories simultaneously?
   **A:** Yes, with precedence order (custom > server > builtin)

2. **Q:** How to handle template conflicts (same ID, different versions)?
   **A:** Use precedence order, optionally warn user

3. **Q:** Should built-in templates be updateable?
   **A:** No, they're read-only. Update via agent update.

4. **Q:** How to handle large template sets (1000s of templates)?
   **A:** Lazy loading, indexing, caching

5. **Q:** Support for template encryption?
   **A:** Future feature, not MVP

