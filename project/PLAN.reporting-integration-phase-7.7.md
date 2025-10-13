# Phase 7.7: REST API Reporting Integration - Implementation Plan

## Overview

**Goal**: Integrate template scan results with the server's REST API to persist vulnerability data in the database and enable UI visibility.

**Approach**: Hybrid (Phase 7.7.1 for quick wins, Phase 7.7.2 for complete feature parity)

---

## Phase 7.7.1: Basic REST API Reporting (RECOMMENDED FOR MVP)

### Priority: HIGH (Should complete before finalizing MVP)

### Task Breakdown

#### Task 7.7.1.1: Basic Host Fingerprinting

**File**: `internal/template/fingerprint/fingerprint.go` (NEW)

**Functionality**:

```go
type HostFingerprint struct {
    OS          string  // runtime.GOOS
    OSVersion   string  // platform-specific detection
    Hostname    string  // os.Hostname()
    PrimaryIP   string  // first non-loopback IPv4
    AgentID     string  // from config
}

func CollectBasicFingerprint(ctx context.Context, cfg *config.AgentConfig) (*HostFingerprint, error)
```

**Platform Detection**:

- **Windows**: `wmic os get caption,version` or registry
- **Linux**: `/etc/os-release` or `uname -r`
- **macOS**: `sw_vers` or `system_profiler`

**Implementation Steps**:

1. Create `internal/template/fingerprint/` directory
2. Implement `fingerprint.go` with OS detection
3. Implement `fingerprint_windows.go`, `fingerprint_linux.go`, `fingerprint_darwin.go`
4. Add tests for each platform
5. Handle errors gracefully (return partial data if some fails)

**Success Criteria**:

- ✅ Returns OS name on all platforms
- ✅ Returns hostname on all platforms
- ✅ Returns primary IP (best effort)
- ✅ Handles errors without crashing
- ✅ Completes in < 1 second

**Test Strategy**:

```bash
# Unit tests
go test ./internal/template/fingerprint/...

# Manual test
go run cmd/test-fingerprint/main.go
```

---

#### Task 7.7.1.2: Template Result Conversion

**File**: `internal/template/reporting/converter.go` (NEW)

**Functionality**:

```go
// Convert template results to sirius.Vulnerability format
func ConvertTemplateResultsToVulnerabilities(results []*types.Result) []sirius.Vulnerability

// Build sirius.Host from fingerprint and vulnerabilities
func BuildHostData(fingerprint *fingerprint.HostFingerprint, vulns []sirius.Vulnerability) sirius.Host

// Build agent metadata with template results
func BuildAgentMetadata(results []*types.Result, executionTime time.Duration) map[string]interface{}
```

**Conversion Logic**:

```go
// For each matched template result:
Vulnerability{
    VID:         result.TemplateID,          // e.g., "CVE-2024-TEST-001"
    Title:       result.TemplateName,        // e.g., "Vulnerable SSH Daemon Detection"
    Description: result.TemplateName,        // Use name if no description
    RiskScore:   severityToRiskScore(result.Severity),
}

// Severity → RiskScore mapping:
// critical → 9.0-10.0
// high     → 7.0-8.9
// medium   → 4.0-6.9
// low      → 0.1-3.9
// info     → 0.0
```

**Agent Metadata Structure**:

```json
{
  "agent_version": "1.0.0-mvp",
  "scan_duration": 1234,
  "template_results": [
    {
      "template_id": "CVE-2024-TEST-001",
      "vulnerability_id": "CVE-2024-TEST-001",
      "vulnerable": true,
      "confidence": 1.0,
      "severity": "critical",
      "steps": 1,
      "matched_steps": 1
    }
  ],
  "template_count": 5,
  "matched_count": 3,
  "worker_count": 10
}
```

**Implementation Steps**:

1. Create `internal/template/reporting/` directory
2. Implement `converter.go` with conversion functions
3. Add `converter_test.go` with comprehensive tests
4. Handle edge cases (empty results, partial matches)
5. Validate output matches `sirius.Host` schema

**Success Criteria**:

- ✅ Converts all matched templates to vulnerabilities
- ✅ Preserves confidence scores
- ✅ Maps severity to risk scores correctly
- ✅ Builds valid sirius.Host structure
- ✅ Includes template metadata

**Test Strategy**:

```go
func TestConvertTemplateResults(t *testing.T) {
    // Test with matched template
    // Test with unmatched template
    // Test with multiple templates
    // Test severity mapping
    // Test metadata generation
}
```

---

#### Task 7.7.1.3: API Submission Integration

**File**: `internal/commands/templatescan/scan_command.go` (MODIFY)

**Changes**:

**After template execution** (line ~126):

```go
results, execErrors := executor.ExecuteTemplatesParallelWithConfig(templates, poolConfig)
executionTime := time.Since(startTime)

// NEW: Report to REST API if we have results
if shouldSubmitToAPI(agentInfo, results) {
    agentInfo.Logger.Info("Submitting template results to REST API")

    // Async submission (don't block command completion)
    go submitTemplateResultsToAPI(ctx, agentInfo, results, executionTime)
}

// Continue with existing output generation
outputData := ScanCommandOutput{...}
```

**New Helper Functions**:

```go
// shouldSubmitToAPI checks if we should submit to API
func shouldSubmitToAPI(agentInfo commands.AgentInfo, results []*types.Result) bool {
    // Don't submit if:
    // - APIClient is nil
    // - ApiBaseURL is empty
    // - No matched results
    // - Configuration disables API reporting

    if agentInfo.APIClient == nil || agentInfo.Config.ApiBaseURL == "" {
        return false
    }

    // Check if any templates matched
    for _, r := range results {
        if r != nil && r.Matched {
            return true
        }
    }

    return false
}

// submitTemplateResultsToAPI performs the API submission
func submitTemplateResultsToAPI(
    ctx context.Context,
    agentInfo commands.AgentInfo,
    results []*types.Result,
    executionTime time.Duration,
) {
    startTime := time.Now()

    // 1. Collect host fingerprint
    hostFingerprint, err := fingerprint.CollectBasicFingerprint(ctx, agentInfo.Config)
    if err != nil {
        agentInfo.Logger.Warn("Failed to collect host fingerprint, using partial data",
            zap.Error(err))
        // Continue with partial fingerprint
    }

    // 2. Convert template results to vulnerabilities
    vulnerabilities := reporting.ConvertTemplateResultsToVulnerabilities(results)
    agentInfo.Logger.Debug("Converted template results to vulnerabilities",
        zap.Int("vulnerability_count", len(vulnerabilities)))

    // 3. Build sirius.Host data
    hostData := reporting.BuildHostData(hostFingerprint, vulnerabilities)

    // 4. Submit to API
    apiCtx := context.Background() // Use background context for async call
    err = agentInfo.APIClient.UpdateHostRecord(apiCtx, agentInfo.Config.ApiBaseURL, hostData)

    submissionTime := time.Since(startTime)

    if err != nil {
        agentInfo.Logger.Error("Failed to submit template results to API",
            zap.Error(err),
            zap.Duration("submission_time", submissionTime))
    } else {
        agentInfo.Logger.Info("Successfully submitted template results to API",
            zap.Int("vulnerabilities", len(vulnerabilities)),
            zap.Duration("submission_time", submissionTime),
            zap.String("host_id", agentInfo.Config.HostID))
    }
}
```

**Implementation Steps**:

1. Add imports for fingerprint and reporting packages
2. Add helper functions to scan_command.go
3. Integrate API submission after template execution
4. Add comprehensive logging
5. Handle errors gracefully (don't fail command if API fails)
6. Add configuration flag to enable/disable API reporting

**Success Criteria**:

- ✅ API submission is async (doesn't block command)
- ✅ Command succeeds even if API fails
- ✅ Comprehensive logging of API submission
- ✅ Respects configuration (can be disabled)
- ✅ Includes all matched vulnerabilities

---

#### Task 7.7.1.4: Integration Testing

**Test Scenarios**:

1. **End-to-End Flow**

   - Start server and agent
   - Run `internal:template-scan --all`
   - Verify data appears in database
   - Check UI displays vulnerabilities

2. **Database Verification**

   - Query hosts table for agent's host
   - Query vulnerabilities table for template CVEs
   - Query host_vulnerabilities junction table
   - Verify all fields populated correctly

3. **API Failure Handling**

   - Simulate API unavailability
   - Verify command still completes
   - Check error logging
   - Verify gRPC response still sent

4. **Cross-Platform**
   - Test on Windows
   - Test on macOS
   - Test on Linux
   - Verify fingerprinting works on all platforms

**Test Commands**:

```bash
# Start infrastructure
docker-compose up -d

# Run agent
./bin/sirius-agent server --agent-id test-agent

# Trigger scan from server
test-agent internal:template-scan --all

# Query database
psql -h localhost -U postgres -d sirius -c "
  SELECT h.hostname, h.os, v.vid, v.title
  FROM hosts h
  JOIN host_vulnerabilities hv ON h.id = hv.host_id
  JOIN vulnerabilities v ON hv.vulnerability_id = v.id
  WHERE h.hostname = '<agent-hostname>';
"

# Check UI
# Navigate to http://localhost:3000/scanner
# Verify host appears with vulnerabilities
```

**Success Criteria**:

- ✅ Vulnerabilities appear in database
- ✅ Host record created/updated
- ✅ Data visible in UI
- ✅ API failures don't crash agent
- ✅ Works on all platforms

---

## Phase 7.7.2: Enhanced Reporting (FUTURE SPRINT)

### Priority: MEDIUM (Post-MVP enhancement)

### Additional Components

#### Task 7.7.2.1: Software Package Enumeration

**Platforms**:

- **Windows**: Registry, winget, wmic
- **Linux**: dpkg, rpm, apt, yum
- **macOS**: brew, system_profiler

**Data Structure**:

```json
{
  "software_inventory": {
    "packages": [
      {
        "name": "openssh-server",
        "version": "8.2p1",
        "architecture": "amd64",
        "source": "dpkg",
        "cpe": "cpe:2.3:a:openbsd:openssh:8.2p1"
      }
    ],
    "package_count": 150,
    "collected_at": "2024-10-13T12:00:00Z"
  }
}
```

#### Task 7.7.2.2: System Fingerprinting

**Components**:

- Hardware: CPU, memory, storage
- Network: interfaces, IPs, MACs, DNS
- Services: running services, status, PIDs
- Users: accounts, groups, shells
- Certificates: SSL/TLS certificates in stores

#### Task 7.7.2.3: Enhanced API Client

**New Method**:

```go
func (a *apiClientAdapter) UpdateHostRecordWithEnhancedData(
    ctx context.Context,
    apiBaseURL string,
    hostData sirius.Host,
    softwareInventory map[string]interface{},
    systemFingerprint map[string]interface{},
    agentMetadata map[string]interface{},
) error
```

**API Endpoint**: `POST {apiBaseURL}/host/enhanced` (or extend existing `/host`)

---

## Configuration

### New Config Fields

**File**: `internal/config/config.go`

```go
type AgentConfig struct {
    // ... existing fields ...

    // API Reporting Configuration
    EnableAPIReporting   bool   `env:"ENABLE_API_REPORTING" envDefault:"true"`
    ApiBaseURL          string `env:"API_BASE_URL" envDefault:"http://localhost:9001"`
    ApiTimeout          int    `env:"API_TIMEOUT" envDefault:"15"` // seconds

    // Fingerprinting Configuration
    EnableFingerprinting bool   `env:"ENABLE_FINGERPRINTING" envDefault:"true"`
    CollectPackages      bool   `env:"COLLECT_PACKAGES" envDefault:"false"` // Phase 7.7.2
    CollectSystemInfo    bool   `env:"COLLECT_SYSTEM_INFO" envDefault:"false"` // Phase 7.7.2
}
```

### Environment Variables

```bash
# Enable/disable API reporting
export ENABLE_API_REPORTING=true

# API endpoint
export API_BASE_URL=http://localhost:9001

# Enable/disable fingerprinting
export ENABLE_FINGERPRINTING=true

# Future: Package enumeration (Phase 7.7.2)
export COLLECT_PACKAGES=false

# Future: System fingerprinting (Phase 7.7.2)
export COLLECT_SYSTEM_INFO=false
```

---

## Testing Strategy

### Unit Tests

```bash
# Fingerprinting tests
go test ./internal/template/fingerprint/...

# Conversion tests
go test ./internal/template/reporting/...

# Command integration
go test ./internal/commands/templatescan/...
```

### Integration Tests

```bash
# Start test infrastructure
docker-compose -f testing/docker-compose.test.yaml up -d

# Run integration test
go test -tags=integration ./internal/commands/templatescan/...

# Verify database
psql -h localhost -U testuser -d testdb -f testing/verify-reporting.sql
```

### Manual E2E Tests

1. Start server and database
2. Run agent with template scan
3. Check server logs for API requests
4. Query database for vulnerability data
5. Verify UI displays results
6. Test on all platforms

---

## Success Metrics

### Phase 7.7.1 Success Criteria

- ✅ 100% of matched templates create vulnerabilities
- ✅ All vulnerabilities visible in database
- ✅ Host fingerprint collected on all platforms
- ✅ < 1 second overhead for API submission
- ✅ 0% command failure rate due to API issues
- ✅ 100% test coverage for conversion logic

### KPIs to Track

- API submission success rate
- Average API submission time
- Vulnerability data accuracy
- Platform compatibility (Windows/macOS/Linux)
- User adoption of template scanning

---

## Rollout Plan

### Development

1. **Week 1**: Tasks 7.7.1.1 & 7.7.1.2

   - Fingerprinting implementation
   - Conversion utilities
   - Unit tests

2. **Week 2**: Tasks 7.7.1.3 & 7.7.1.4
   - API integration
   - Integration tests
   - E2E validation

### Testing

1. **Local Testing**: Developer machines
2. **Integration Testing**: Test infrastructure
3. **UAT**: Staging environment with real data
4. **Production**: Gradual rollout

### Monitoring

- API submission success rate
- Database growth (vulnerabilities, hosts)
- Error logs from API failures
- User feedback on UI data quality

---

## Risks & Mitigations

| Risk                    | Impact | Mitigation                                |
| ----------------------- | ------ | ----------------------------------------- |
| API structure mismatch  | High   | Validate with server team first           |
| Performance degradation | Medium | Async submission, timeout limits          |
| Cross-platform issues   | Medium | Comprehensive testing, graceful fallbacks |
| Database schema changes | High   | Coordinate with API team                  |
| API unavailability      | Low    | Async call, error handling                |

---

## Dependencies

### Code Dependencies

- `github.com/SiriusScan/go-api/sirius` - Host and Vulnerability structs
- `internal/config` - Agent configuration
- `internal/template/types` - Template result types
- `internal/apiclient` - REST API client

### Infrastructure Dependencies

- Sirius API running at configured URL
- PostgreSQL database accessible
- Network connectivity from agent to API

### Team Dependencies

- API team: Confirm `/host` endpoint structure
- Database team: Confirm schema compatibility
- UI team: Verify data display requirements

---

## Documentation Updates

### Files to Update

1. **AGENT-COMMANDS-REFERENCE.md**

   - Add API reporting behavior
   - Document configuration options

2. **PHASE-7.6-COMPLETION-SUMMARY.md**

   - Add note about API reporting
   - Update architecture diagram

3. **README.md**

   - Update feature list
   - Add API integration documentation

4. **Configuration Guide** (NEW)
   - Environment variables
   - API endpoint setup
   - Troubleshooting

---

## Next Steps

1. **Review this plan** with team
2. **Confirm API endpoint** structure with server team
3. **Create tasks** in task-system-mvp.json
4. **Begin implementation** of Phase 7.7.1.1
5. **Schedule testing** with QA team

---

## Questions for Team

1. Is `POST /host` the correct API endpoint?
2. Does the `sirius.Host` structure match expectations?
3. Should we implement Phase 7.7.1 before MVP release?
4. Is the agent_metadata structure acceptable?
5. Any security requirements for API authentication?

---

**Status**: Ready for Implementation
**Estimated Effort**: Phase 7.7.1 = 2-3 days, Phase 7.7.2 = 1-2 weeks
**Priority**: High (Phase 7.7.1), Medium (Phase 7.7.2)
