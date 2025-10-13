# Testing the Template System

This guide shows you how to manually test the Template System MVP that has been implemented.

## Quick Start

The template system is accessible via the `template-cli` command-line tool.

### Build the CLI

```bash
go build -o bin/template-cli cmd/template-cli/main.go
```

## Available Commands

```
template-cli run <template-file>        # Run a single template
template-cli run-all <directory>        # Run all templates in directory
template-cli validate <template-file>   # Validate a template
template-cli list-modules               # List available detection modules
template-cli module-info <type>         # Show module information
template-cli help                       # Show help message
```

## Testing Workflow

### 1. List Available Modules

See what detection modules are registered:

```bash
./bin/template-cli list-modules
```

**Expected Output:**
```
📦 Available Detection Modules
===============================

Found 1 module(s):

1. file_hash
   Name: File Hash Validator
   Description: Compares file cryptographic hashes to detect specific vulnerable file versions
   Supported OS: [linux darwin windows]
```

### 2. View Module Details

Get detailed information about a module:

```bash
./bin/template-cli module-info file_hash
```

**Expected Output:**
```
📦 Module Information: file_hash
====================================

Type: file_hash
Name: File Hash Validator
Version: 1.0.0
Author: Sirius Scan

Description:
  Compares file cryptographic hashes to detect specific vulnerable file versions

Supported OS:
  - linux
  - darwin
  - windows

Configuration Fields:
  - path: Path to the file to check
  - hash: Expected hash value to compare against
  - algorithm: Hash algorithm to use (sha256, sha1, md5, sha512). Defaults to sha256.
```

### 3. Validate a Template

Check if a template is valid:

```bash
./bin/template-cli validate testing/test-templates/01-file-hash.yaml
```

**Expected Output:**
```
🔍 Validating template: testing/test-templates/01-file-hash.yaml
================================

✓ Parsed successfully
  ID: CVE-2024-TEST-001
  Name: Vulnerable SSH Daemon Detection
  Severity: critical

✅ Template is valid!
  Steps: 1
  Logic: all
```

### 4. Run a Single Template

Execute a template against your system:

```bash
./bin/template-cli run testing/test-templates/01-file-hash.yaml
```

**Expected Output:**
```
🔍 Running template: testing/test-templates/01-file-hash.yaml
========================================

✓ Parsed template: CVE-2024-TEST-001
  Name: Vulnerable SSH Daemon Detection
  Severity: critical
  Steps: 1

✓ Template is valid

🚀 Executing template...

📊 Results:
-----------
✅ MATCHED - Vulnerability detected!

Confidence: 1.00
Executed Steps: 1
Errors: 0
Host: your-hostname
Timestamp: 2024-01-15 10:30:45

📋 Step Details:
  Step 1 (file_hash):
    Matched: true
    Duration: 1.234ms

📄 JSON Output:
---------------
{
  "template_id": "CVE-2024-TEST-001",
  "template_name": "Vulnerable SSH Daemon Detection",
  "severity": "critical",
  "matched": true,
  "confidence": 1,
  "steps": [...],
  "timestamp": "2024-01-15T10:30:45Z",
  "host": "your-hostname"
}
```

### 5. Run All Templates

Execute all templates in a directory:

```bash
./bin/template-cli run-all testing/test-templates/
```

**Expected Output:**
```
🔍 Discovering templates in: testing/test-templates/
=========================================

📊 Discovery Results:
  Valid templates: 3
  Errors: 0

🚀 Executing templates...
-------------------------

[1/3] Vulnerable SSH Daemon Detection (CVE-2024-TEST-001)
  ✅ MATCHED (Confidence: 1.00)

[2/3] File Hash Mismatch Test (CVE-2024-TEST-002)
  ❌ NOT MATCHED

[3/3] Missing File Test (CVE-2024-TEST-003)
  ❌ NOT MATCHED

=========================================
Summary: 1/3 templates matched
⚠️  Vulnerabilities detected!
```

## Test Templates

The following test templates are available in `testing/test-templates/`:

### 01-file-hash.yaml
- **CVE**: CVE-2024-TEST-001
- **Purpose**: Tests file hash detection (should match)
- **File**: testing/test-data/vulnerable-sshd
- **Expected**: MATCHED

### 02-file-hash-mismatch.yaml
- **CVE**: CVE-2024-TEST-002
- **Purpose**: Tests hash mismatch (should NOT match)
- **File**: testing/test-data/vulnerable-sshd
- **Expected**: NOT MATCHED

### 03-file-hash-missing.yaml
- **CVE**: CVE-2024-TEST-003
- **Purpose**: Tests missing file error handling
- **File**: /nonexistent/file/that/does/not/exist
- **Expected**: Error (file not found)

## Creating Your Own Template

Create a YAML file with the following structure:

```yaml
id: my-test-001
info:
  name: My Test Template
  author: Your Name
  severity: high
  description: Description of what this detects
  version: "1.0"

detection:
  logic: all  # or "any"
  steps:
    - type: file_hash
      platforms:
        - linux
        - darwin
      weight: 1.0
      config:
        path: /path/to/file
        hash: abc123...
        algorithm: sha256
```

Then test it:

```bash
./bin/template-cli validate my-template.yaml
./bin/template-cli run my-template.yaml
```

## What's Been Implemented

✅ **Core Architecture**
- Module Registry (thread-safe)
- Template Type System
- Shared Libraries (Files, Patterns, Results)

✅ **Template Parser**
- YAML parsing
- Template validation
- Template discovery (recursive directory scanning)

✅ **Template Executor**
- Sequential step execution
- Platform filtering (linux/darwin/windows)
- AND/OR logic evaluation
- Confidence calculation
- Error handling

✅ **Detection Modules**
- FileHash module (SHA256, SHA1, MD5, SHA512)

## Test Coverage

- **Parser Tests**: 46/46 passing ✓
- **FileHash Tests**: 13/13 passing ✓
- **Executor Tests**: 12/12 passing ✓
- **Total**: 71 unit tests passing

## Troubleshooting

### "Module not found" error
Make sure the module is imported in the CLI. Check `cmd/template-cli/main.go` for the import:
```go
_ "github.com/SiriusScan/app-agent/internal/modules/filehash"
```

### "Template not found" error
Check that the file path is correct and the file exists.

### "Permission denied" error
Ensure you have read permissions for the template and test files.

## Next Steps

The following features are planned but not yet implemented:
- Additional detection modules (file_content, command_version)
- Parallel execution (worker pool)
- Output formats (JSON, JSONL, text with colors)
- Full Cobra CLI integration

