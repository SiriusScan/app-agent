# Sirius Agent - Command Reference

## Unified Binary: `sirius-agent`

### Server Mode (Agent Mode)

Connect to Sirius server and wait for commands:

```bash
# Start agent in server mode
sirius-agent server

# With custom server address
sirius-agent server --address localhost:50051

# With custom agent ID
sirius-agent server --agent-id my-agent-123

# With environment variables
export SERVER_ADDRESS=localhost:50051
export AGENT_ID=my-laptop
sirius-agent server
```

### CLI Mode (Local Execution)

Execute templates locally without server:

```bash
# Run single template
sirius-agent template run ./templates/my-template.yaml

# Run all templates in directory
sirius-agent template run-all ./templates/

# Run with parallel workers
sirius-agent template run-all ./templates/ --workers 10

# List available templates
sirius-agent template list --directory ./templates/

# Validate template
sirius-agent template validate ./templates/my-template.yaml

# List available detection modules
sirius-agent module list

# Get module details
sirius-agent module info file_hash
```

## Server Commands

When connected to server, these commands can be sent from the server console or via queue:

### Template Scanning

**⚠️ IMPORTANT:** Command name includes `internal:` prefix!

```bash
# Basic scan (uses default template directory)
<agent-id> internal:template-scan

# Scan all templates
<agent-id> internal:template-scan --all

# Scan specific directory
<agent-id> internal:template-scan --directory /app-agent/templates

# Scan single template
<agent-id> internal:template-scan --template /app-agent/templates/cve-2023-1234.yaml

# Configure workers
<agent-id> internal:template-scan --workers 10

# Adjust timeout (seconds)
<agent-id> internal:template-scan --timeout 60

# Text format output
<agent-id> internal:template-scan --format text

# Combined options
<agent-id> internal:template-scan --all --workers 5 --timeout 120
```

### Other Internal Commands

```bash
# Get agent status
<agent-id> internal:status

# Run vulnerability scan (legacy)
<agent-id> internal:scan
```

## Command Output Format

### JSON Output (Default)

```json
{
  "summary": {
    "total_templates": 7,
    "matched": 5,
    "execution_time_ms": 1234,
    "workers": 8
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
  ],
  "discovery_errors": [],
  "execution_errors": []
}
```

### Text Output

```
🔍 Template Scan Results
==================================================

📊 Summary:
  Total Templates: 7
  Matched: 5
  Execution Time: 1.234s

📋 Template Results:

[1] ✅ Vulnerable SSH Daemon (ID: CVE-2023-1234)
    Severity: critical | Confidence: 1.00
    Matched Steps: 1/1

[2] ❌ File Hash Mismatch Test (ID: TEST-002)
    Severity: low | Confidence: 0.00
...
```

## Troubleshooting

### Command Not Found

**Error:** `Command not found in internal registry, attempting script execution`

**Solution:** Use the correct command name with `internal:` prefix:

- ❌ `template-scan`
- ❌ `scan`
- ✅ `internal:template-scan`

### No Templates Found

**Error:** `no templates found in "/app-agent/templates"`

**Solutions:**

1. Check template directory exists: `ls /app-agent/templates`
2. Ensure templates are valid YAML
3. Use `--directory` flag to specify correct path
4. Check file permissions

### Worker Count Error

**Error:** `worker count must be between 1 and 50`

**Solution:** Use valid worker count:

```bash
internal:template-scan --workers 10  # Valid
```

## Architecture

```
sirius-agent (unified binary)
├── server mode          → connects to Sirius server
│   ├── receives commands via gRPC stream
│   ├── executes internal:* commands
│   └── sends results back via stream
└── CLI mode             → standalone local execution
    ├── template run/run-all/list/validate
    └── module list/info
```

## Environment Variables

```bash
# Server mode configuration
export SERVER_ADDRESS=localhost:50051   # gRPC server address
export AGENT_ID=my-agent-123           # Unique agent identifier

# PowerShell configuration (Windows/macOS)
export POWERSHELL_PATH=/usr/bin/pwsh   # Custom PowerShell path
export ENABLE_SCRIPTING=true           # Enable script execution
```

## Detection Modules

Currently available modules:

- **file_hash** - SHA256/SHA1/MD5/SHA512 file hash comparison
- **file_content** - Regex pattern matching in files

View details:

```bash
sirius-agent module info file_hash
sirius-agent module info file_content
```

## Examples

### Local Template Development

```bash
# 1. Create template
cat > my-template.yaml << 'EOF'
id: TEST-001
info:
  name: My Test Template
  severity: low
detection:
  logic: all
  steps:
    - type: file_hash
      config:
        path: /etc/passwd
        hash: abc123...
EOF

# 2. Validate template
sirius-agent template validate my-template.yaml

# 3. Test execution
sirius-agent template run my-template.yaml --format text
```

### Production Agent Deployment

```bash
# 1. Configure environment
export SERVER_ADDRESS=prod-server:50051
export AGENT_ID=$(hostname)

# 2. Start agent
sirius-agent server

# 3. From server console:
$(hostname) internal:template-scan --all
```

## Version

```bash
sirius-agent version
# Output: sirius-agent version 1.0.0-mvp
```
