# Server Integration Analysis

## Current Architecture Assessment

### Existing Components

**Agent Side (`internal/agent/agent.go`):**
- ✅ gRPC bidirectional stream connection to server
- ✅ Heartbeat mechanism (every 30s)
- ✅ Command dispatch system via `commands.Dispatch()`
- ✅ Result reporting back to server via stream
- ✅ Handles `internal:status`, `internal:scan`, script execution

**Server Side (`internal/server/server.go`):**
- ✅ Maintains bidirectional streams with agents
- ✅ Tracks connected agents
- ✅ Can send commands to agents
- ✅ Receives results from agents
- ✅ Queue integration (RabbitMQ) for external commands
- ✅ Stores results in Valkey/ResponseStore

**Command System (`internal/commands/`):**
- ✅ Command registry with prefix matching
- ✅ Commands implement `Command` interface with `Execute()`
- ✅ OLD template commands exist (discover-templates, list-templates)
  - These use the OLD POC template system from `internal/detect/template`
  - NOT compatible with our new template system

**Protobuf Messages (`proto/hello/hello.proto`):**
- ✅ `CommandRequest` / `CommandResult` - generic command/result
- ✅ `AgentMessage` / `ServerMessage` - stream messages
- ✅ `MessageType` enum (HEARTBEAT, COMMAND, RESULT, ACKNOWLEDGMENT)
- ✅ Command results include: output, error, exit_code, execution_time

### What We Built (Phase 0-7)

**New Template System:**
- ✅ Module registry (file_hash, file_content)
- ✅ Template type system (YAML-based vulnerability detection)
- ✅ Template parser & validator
- ✅ Template executor (confidence scoring, AND/OR logic)
- ✅ Worker pool (parallel execution)
- ✅ Cobra CLI (`sirius-agent` binary)
  - `template run <file>` - execute single template
  - `template run-all <dir>` - execute all templates
  - `template list` - list templates
  - `template validate` - validate template
  - `module list` - list modules
  - `module info <type>` - module details

**Key Difference:**
- OLD system: `internal/detect/template` (POC, Valkey-backed)
- NEW system: `internal/template/*` + `internal/modules/*` (MVP, production-ready)

## Integration Plan

### Goal
Enable agents to receive template execution commands from the server and report detailed vulnerability detection results back.

### Required Changes

#### 1. New Command: `internal:template-scan`

**Create:** `internal/commands/templatescan/scan_command.go`

**Functionality:**
- Parse arguments: `--directory <path>` or `--template <file>` or `--all`
- Use our new `executor.ExecuteTemplatesParallelWithConfig()`
- Format results as JSON
- Return results via command output

**Arguments:**
```
internal:template-scan                    # Scan with default templates
internal:template-scan --all              # Scan all templates
internal:template-scan --directory ./templates/  # Scan specific directory
internal:template-scan --template file.yaml      # Scan single template
internal:template-scan --workers 10       # Configure workers
internal:template-scan --format json      # Output format
```

**Output Format:**
```json
{
  "summary": {
    "total_templates": 7,
    "matched": 5,
    "execution_time_ms": 1234
  },
  "results": [
    {
      "template_id": "CVE-2023-1234",
      "template_name": "Vulnerable SSH Daemon",
      "severity": "critical",
      "matched": true,
      "confidence": 1.0,
      "steps": [...]
    }
  ]
}
```

#### 2. Enhanced Proto Messages (Optional)

**Current:** Generic `CommandResult` works fine
- `output` field can contain JSON
- `error` field for errors
- `exit_code` for success/failure
- `execution_time` for performance tracking

**Optional Enhancement (Phase 12):**
Add dedicated template result message type:
```protobuf
message TemplateExecutionResult {
  string template_id = 1;
  string template_name = 2;
  string severity = 3;
  bool matched = 4;
  double confidence = 5;
  repeated StepResult steps = 6;
}

message TemplateScanResult {
  int32 total_templates = 1;
  int32 matched_count = 2;
  int64 execution_time_ms = 3;
  repeated TemplateExecutionResult results = 4;
}
```

**Decision:** Start with generic `CommandResult` (no upstream changes needed), enhance later if needed.

#### 3. Agent Integration

**Minimal Changes:**
- Add `_ "github.com/SiriusScan/app-agent/internal/commands/templatescan"` import to `cmd/agent/main.go`
- Command will auto-register via `init()`
- Agent dispatch system will route `internal:template-scan` to our new command

#### 4. Server Integration

**No Changes Needed:**
- Server already sends commands via stream
- Server already receives results via stream
- Server already stores results in Valkey/ResponseStore
- Queue integration already works

#### 5. Testing Strategy

**Unit Tests:**
- `TestTemplateScanCommand_Execute()` - command logic
- `TestTemplateScanCommand_ParseArgs()` - argument parsing
- `TestTemplateScanCommand_OutputFormats()` - JSON/text output

**Integration Tests:**
- Agent receives command from server
- Agent executes templates
- Agent sends results back to server
- Server stores results correctly

**End-to-End Test:**
```bash
# 1. Start server
go run cmd/server/main.go

# 2. Start agent
go run cmd/agent/main.go

# 3. Send command from server console
> agent-123 internal:template-scan --all

# 4. Verify results appear in server console and Valkey
```

### Implementation Phases

**Phase 7.5: Server Command Integration (NEW)**
- Task 7.5.1: Create `internal/commands/templatescan/scan_command.go`
- Task 7.5.2: Add agent integration tests
- Task 7.5.3: Add end-to-end server communication tests

### Upstream API Changes

**None Required** for MVP. The generic `CommandResult` message is sufficient for template scan results in JSON format.

**Future Enhancement (Phase 12):** Consider dedicated protobuf messages for richer type safety and better tooling support.

### Benefits of This Approach

1. **Minimal Disruption:** Uses existing infrastructure
2. **No Breaking Changes:** No upstream API changes required
3. **Backward Compatible:** Old commands still work
4. **Flexible:** Can add more template commands later
5. **Production Ready:** Leverages battle-tested gRPC streaming

### Security Considerations

1. **Template Source Trust:** Only execute templates from trusted directories
2. **Resource Limits:** Worker pool already has limits (1-50 workers)
3. **Timeout Protection:** Per-template timeout prevents hangs
4. **Result Size:** JSON output size should be monitored
5. **Access Control:** Server should validate agent permissions (future)

### Performance Considerations

1. **Parallel Execution:** Worker pool provides optimal performance
2. **Result Streaming:** Consider chunking large result sets (future)
3. **Network Efficiency:** JSON is compressed over gRPC
4. **Memory Usage:** Worker pool limits concurrent executions

## Summary

**What Works Now:**
- ✅ Local CLI execution (`sirius-agent template run-all`)
- ✅ All template system components tested and working
- ✅ Worker pool for parallel execution

**What's Needed:**
- 🔨 New `internal:template-scan` command
- 🔨 Agent imports new command
- 🔨 Integration tests
- 🔨 End-to-end testing

**What Doesn't Need Changes:**
- ✅ Server infrastructure (already works)
- ✅ Agent infrastructure (already works)
- ✅ Protobuf definitions (generic messages work)
- ✅ Queue integration (already works)
- ✅ Result storage (already works)

**Estimated Effort:** 3-4 subtasks, ~2-3 hours of work

**Risk Level:** Low - leverages existing patterns and infrastructure

