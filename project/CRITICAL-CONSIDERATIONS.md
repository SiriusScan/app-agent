# Critical Considerations & Pitfalls - Template System MVP

## Purpose

This document identifies critical issues, potential pitfalls, and important considerations that could derail the MVP implementation if not addressed proactively.

---

## 🚨 Critical Technical Issues

### 1. Go Module Import Paths

**Issue**: The project uses a replace directive in `go.mod`:

```go
replace github.com/SiriusScan/go-api => ../go-api
```

**Potential Problem**:

- If `go-api` doesn't exist or changes, builds break
- Cross-compilation in container needs this path to work
- New developers might not have this structure

**Solutions**:

- ✅ **Document dependency structure** in README
- ✅ **Check if go-api is needed** for MVP (might not be!)
- ✅ **Consider removing dependency** if not critical
- ✅ **Add validation** to Makefile to check paths

**Action**: Phase 0 - Audit dependencies, remove unused imports

---

### 2. Context Timeouts & Cancellation

**Issue**: Templates and modules need proper timeout handling

**Potential Problems**:

- Long-running templates block worker pool
- No way to cancel stuck executions
- Resource leaks if context not properly propagated

**Solutions**:

- ✅ **Per-step timeout**: 30s default, configurable
- ✅ **Per-template timeout**: 5 minutes maximum
- ✅ **Global timeout**: For entire execution
- ✅ **Proper context propagation** through all layers
- ✅ **Panic recovery** in worker goroutines

**Critical Code Pattern**:

```go
// ALWAYS use context with timeout
ctx, cancel := context.WithTimeout(parentCtx, timeout)
defer cancel()

// ALWAYS propagate context
result, err := module.Execute(ctx, config)

// ALWAYS recover from panics in goroutines
defer func() {
    if r := recover(); r != nil {
        logger.Error("panic in worker", zap.Any("panic", r))
    }
}()
```

**Action**: Phase 1, 4, 7 - Implement timeout handling at each layer

---

### 3. File System Permissions

**Issue**: Agent needs to read system files that may have restricted permissions

**Potential Problems**:

- Permission denied errors on critical files
- Different behavior between dev/prod
- SELinux/AppArmor restrictions
- Root vs non-root execution

**Solutions**:

- ✅ **Graceful error handling** - Don't crash on permission denied
- ✅ **Clear error messages** - Tell user what failed and why
- ✅ **Document privilege requirements** for each module
- ✅ **Test as non-root user** to catch permission issues early

**Critical Error Pattern**:

```go
content, err := os.ReadFile(path)
if err != nil {
    if os.IsPermission(err) {
        return result.WithError("permission_denied",
            fmt.Sprintf("Cannot read %s: permission denied", path))
    }
    return result.WithError("file_error", err.Error())
}
```

**Action**: Phase 3, 5, 9 - Implement proper error handling in each module

---

### 4. Regex Denial of Service (ReDoS)

**Issue**: User-provided regex patterns in templates could cause catastrophic backtracking

**Potential Problems**:

- Malicious or poorly written regex hangs agent
- CPU spike from exponential regex complexity
- No way to detect or prevent bad patterns

**Solutions**:

- ✅ **Timeout on regex matching** - Maximum 5 seconds per match
- ✅ **Regex complexity validation** - Warn on nested quantifiers
- ✅ **Context-aware execution** - Pass timeout to regex matching
- ✅ **Document regex best practices** for template authors

**Critical Pattern**:

```go
// BAD: No timeout
regex.Match(pattern, input)

// GOOD: With timeout via context
func MatchWithTimeout(ctx context.Context, pattern, input string) {
    done := make(chan bool)
    go func() {
        // Do matching
        done <- true
    }()

    select {
    case <-done:
        return result
    case <-ctx.Done():
        return timeout error
    }
}
```

**Action**: Phase 1 - Implement timeout-aware regex matching in `common/patterns`

---

### 5. Concurrent Map Access

**Issue**: Module registry and result collection use maps that could be accessed concurrently

**Potential Problems**:

- Race conditions in module registry
- Concurrent writes to result maps
- Panic: "concurrent map writes"

**Solutions**:

- ✅ **Use sync.RWMutex** for module registry
- ✅ **Channel-based result collection** (no shared map)
- ✅ **Run with -race flag** during development
- ✅ **Test with high worker counts** to expose race conditions

**Critical Pattern**:

```go
// Module registry with mutex
type Registry struct {
    modules map[string]Module
    mu      sync.RWMutex
}

func (r *Registry) Get(name string) (Module, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    m, ok := r.modules[name]
    return m, ok
}
```

**Action**: Phase 1 - Implement thread-safe registry from the start

---

## ⚠️ Architectural Pitfalls

### 6. Module Interface Too Restrictive

**Issue**: If module interface is too rigid, future modules will be painful to implement

**Potential Problem**:

- Can't add new capabilities without breaking interface
- Every module forced into same pattern even if doesn't fit
- Future script module might not fit interface

**Prevention**:

- ✅ **Minimal interface** - Just `Execute()` method
- ✅ **Flexible config** - Use `map[string]interface{}` for step config
- ✅ **Optional capabilities** - Use type assertions for advanced features
- ✅ **Versioning plan** - Think about interface v2 now

**Good Design**:

```go
// Minimal core interface
type Module interface {
    Execute(ctx context.Context, config StepConfig) (*Result, error)
}

// Optional advanced capabilities
type CacheableModule interface {
    Module
    CacheKey(config StepConfig) string
}

type ValidatableModule interface {
    Module
    ValidateConfig(config StepConfig) error
}
```

**Action**: Phase 1 - Design interface with future extensibility in mind

---

### 7. YAML Schema Evolution

**Issue**: Template YAML schema will need to evolve over time

**Potential Problems**:

- Breaking changes to existing templates
- No versioning strategy
- Can't deprecate old fields gracefully

**Prevention**:

- ✅ **Version field in templates** - `schema_version: "1.0"`
- ✅ **Backward compatibility** - Parse old formats
- ✅ **Deprecation warnings** - Warn on old schema usage
- ✅ **Migration tools** - Script to upgrade templates

**Schema Versioning**:

```yaml
# Version 1.0 (MVP)
schema_version: "1.0"
id: template-id
detection:
  logic: all
  steps: [...]

# Future Version 2.0 (might add new features)
schema_version: "2.0"
id: template-id
detection:
  logic: "(step1 AND step2) OR step3"  # Complex expressions
  steps: [...]
```

**Action**: Phase 2 - Add `schema_version` field to template struct NOW

---

### 8. Worker Pool Starvation

**Issue**: If templates take wildly different times, some workers sit idle

**Potential Problems**:

- Slow templates block other fast ones
- Poor parallelism despite worker pool
- Unbalanced work distribution

**Prevention**:

- ✅ **Dynamic work stealing** - Workers grab next available job
- ✅ **Template timeout** - Don't let one template hog workers
- ✅ **Priority queue** - Fast templates first (optional)
- ✅ **Progress monitoring** - Track worker utilization

**Good Pattern** (dynamic work distribution):

```go
// Workers pull from shared channel
jobs := make(chan *Template, len(templates))
results := make(chan *Result, len(templates))

// Queue all templates
for _, template := range templates {
    jobs <- template
}
close(jobs)

// Workers process whatever is available
for i := 0; i < numWorkers; i++ {
    go func() {
        for template := range jobs {
            result := execute(template)
            results <- result
        }
    }()
}
```

**Action**: Phase 7 - Implement dynamic work distribution

---

## 🎯 Development Process Pitfalls

### 9. Container Development Friction

**Issue**: Slow container rebuild cycles kill productivity

**Potential Problem**:

- Rebuilding container for every code change
- Slow iteration on module development
- Frustration leads to skipping container testing

**Prevention**:

- ✅ **Cross-compile on host** - Build binary outside container
- ✅ **Volume mount binary** - Don't rebuild container
- ✅ **Hot reload** - Mount source if possible
- ✅ **Fast Makefile targets** - `make quick` should be < 10 seconds

**Optimal Development Flow**:

```bash
# 1. Edit code on macOS (in IDE)
vim internal/modules/filehash/filehash.go

# 2. Cross-compile (2 seconds)
make build-linux

# 3. Run in container (3 seconds)
make quick

# Total: 5 seconds from code change to result
```

**Action**: Phase 0 - Set up fast iteration workflow from day 1

---

### 10. Testing Without Real Vulnerabilities

**Issue**: How do we test vulnerability detection without real vulnerable systems?

**Potential Problem**:

- Can't test with real malware/vulnerable files
- Fake test data doesn't prove detection works
- False confidence in detection accuracy

**Solution**:

- ✅ **Create fake vulnerable files** - Known-bad hashes, patterns
- ✅ **Mock vulnerable configs** - Bad SSH configs, etc.
- ✅ **Document what is real vs fake** - Clear about test data
- ✅ **Accept limitations** - MVP tests mechanics, not accuracy
- ✅ **Post-MVP validation** - Test against real CVE databases

**Test Data Strategy**:

```bash
testing/test-data/
├── vulnerable-sshd          # Fake binary with known hash
├── vulnerable-apache.conf   # Config with known bad pattern
├── old-openssl-version.txt  # Fake version output
└── README.md                # Documents test data provenance
```

**Action**: Phase 3 - Create comprehensive fake test data

---

### 11. Error Handling Philosophy

**Issue**: How much should we fail vs. continue on errors?

**Critical Decision**:

- **Template-level errors**: Continue processing other templates
- **Step-level errors**: Continue to next step in template
- **Module-level errors**: Return error result, don't crash

**Philosophy**:

```
NEVER CRASH THE AGENT
- Bad template? Log error, skip template, continue
- Bad step? Log error, mark step failed, continue
- Bad module? Log error, return error result, continue
- Panic? Recover, log, continue
```

**Pattern**:

```go
// Template executor
for _, template := range templates {
    result, err := executeTemplate(ctx, template)
    if err != nil {
        // Don't stop! Log and continue
        logger.Error("template failed",
            zap.String("id", template.ID),
            zap.Error(err))
        results = append(results, errorResult(template, err))
        continue
    }
    results = append(results, result)
}
```

**Action**: Phase 4 - Implement graceful error handling throughout

---

## 🔒 Security Considerations

### 12. Command Injection in CommandVersion Module

**Issue**: CommandVersion executes arbitrary commands

**CRITICAL SECURITY RISK**:

- Template could contain malicious commands
- Shell injection if not careful
- Privilege escalation vectors

**Mitigation**:

```go
// ❌ DANGEROUS - Don't do this
cmd := exec.Command("sh", "-c", userCommand)

// ✅ SAFE - Do this
cmd := exec.Command(commandArray[0], commandArray[1:]...)
// No shell interpretation, no injection

// ✅ Even better - Whitelist allowed commands (post-MVP)
allowedCommands := map[string]bool{
    "ssh": true,
    "nginx": true,
    "apache2": true,
}
```

**Template Schema**:

```yaml
# GOOD: Array (no shell)
command: ["ssh", "-V"]

# BAD: String (shell injection risk)
command: "ssh -V"  # Don't support this
```

**Action**: Phase 9 - Implement command execution securely, document risks

---

### 13. Path Traversal in File Operations

**Issue**: Template could specify paths like `../../../../etc/passwd`

**Security Risk**:

- Read arbitrary files outside intended scope
- Information disclosure
- Potential privilege escalation

**Mitigation**:

```go
// Validate paths
func SafePath(path string) (string, error) {
    // Resolve to absolute path
    abs, err := filepath.Abs(path)
    if err != nil {
        return "", err
    }

    // Check for path traversal attempts
    clean := filepath.Clean(abs)
    if !strings.HasPrefix(clean, "/") {
        return "", errors.New("invalid path")
    }

    // Optional: Restrict to certain directories (post-MVP)
    // allowed := []string{"/usr", "/etc", "/var"}

    return clean, nil
}
```

**Action**: Phase 3 - Implement path validation in file operations

---

## 📊 Performance Pitfalls

### 14. Memory Usage with Large Template Sets

**Issue**: Loading 10,000 templates into memory could be problematic

**Potential Problem**:

- High memory usage on agent start
- Template parsing overhead
- Holding all results in memory before output

**Mitigation**:

- ✅ **Lazy loading** - Parse templates on demand (post-MVP)
- ✅ **Streaming output** - JSONL outputs results as they complete
- ✅ **Template caching** - LRU cache for frequently used templates (post-MVP)
- ✅ **Memory profiling** - Test with large template sets early

**Action**: Phase 11 - Performance test with 1,000+ templates

---

### 15. Disk I/O Bottlenecks

**Issue**: Reading thousands of files for hash checking creates I/O bottleneck

**Potential Problem**:

- Slow detection even with parallelism
- Disk becomes bottleneck, not CPU
- SSD vs HDD performance differences

**Mitigation**:

- ✅ **Limit concurrent file operations** - Don't parallelize I/O
- ✅ **File operation worker pool** - Separate from template pool
- ✅ **Skip large files** - Configurable size limits
- ✅ **Caching** - Cache hash results (post-MVP)

**Pattern**:

```go
// Separate pools for CPU-bound and I/O-bound work
templateWorkers := runtime.NumCPU()     // CPU-bound
fileWorkers := 4                        // I/O-bound (limit!)
```

**Action**: Phase 7 - Consider I/O limits in worker pool design

---

## 🎓 Developer Onboarding Pitfalls

### 16. Inadequate Documentation

**Issue**: New developers struggle to understand architecture

**Risk**:

- Long onboarding time
- Inconsistent contributions
- Repeated questions

**Prevention**:

- ✅ **PROJECT-INTRO.md** - Quick start guide ✅ DONE
- ✅ **Architecture diagrams** - Visual system overview
- ✅ **Module examples** - Complete working examples
- ✅ **Troubleshooting guide** - Common issues and solutions

**Action**: Phase 10 - Complete all documentation before release

---

### 17. Insufficient Testing Documentation

**Issue**: Developers don't know how to test their changes

**Risk**:

- Skipping tests
- Breaking existing functionality
- Slow feedback loop

**Prevention**:

- ✅ **Testing workflow docs** - Step-by-step testing guide
- ✅ **Test data documentation** - What test files represent
- ✅ **Makefile targets** - Clear, simple test commands
- ✅ **CI/CD setup** - Automated testing (post-MVP)

**Action**: Phase 0 - Document testing workflow immediately

---

## ✅ Pre-Implementation Checklist

Before writing ANY implementation code:

### Technical Foundation

- [ ] Dependencies audited (go.mod, replace directives)
- [ ] Context timeout strategy defined
- [ ] Thread-safety approach decided (mutexes, channels)
- [ ] Error handling philosophy documented
- [ ] Security mitigations planned

### Development Environment

- [ ] Container workflow tested and fast (<10 sec iteration)
- [ ] Cross-compilation working
- [ ] Test data structure planned
- [ ] Makefile targets defined
- [ ] IDE/editor configured for Go

### Architecture

- [ ] Module interface designed (minimal, extensible)
- [ ] Template schema versioning planned
- [ ] Worker pool strategy decided
- [ ] Result streaming approach defined
- [ ] Deprecation plan documented ✅ DONE

### Documentation

- [ ] PROJECT-INTRO.md written ✅ DONE
- [ ] Implementation plan created ✅ DONE
- [ ] Brainstorming notes complete ✅ DONE
- [ ] Task list structure planned
- [ ] Testing workflow documented

---

## 🚀 Success Criteria Review

MVP is successful if we AVOID these pitfalls:

- ✅ **No race conditions** - Test with `-race` flag
- ✅ **No memory leaks** - Profile with large template sets
- ✅ **No security holes** - Review command execution, file operations
- ✅ **No user frustration** - Fast iteration, clear errors
- ✅ **No architectural regrets** - Extensible design from day 1

---

**This document should be reviewed at each milestone to ensure we're staying on track and avoiding known pitfalls.**
