# Template Architect's Guide

This guide provides a comprehensive reference for creating vulnerability detection templates for the Sirius Scan agent. It details the template structure, available detection modules, and best practices for template architects.

## Table of Contents

- [Introduction](#introduction)
- [Template Structure](#template-structure)
  - [Metadata (`info`)](#metadata-info)
  - [Detection Logic (`detection`)](#detection-logic-detection)
- [Detection Modules](#detection-modules)
  - [File Hash Validator (`file_hash`)](#file-hash-validator-file_hash)
  - [File Content Pattern Matcher (`file_content`)](#file-content-pattern-matcher-file_content)
  - [Command Version Extractor (`version_cmd`)](#command-version-extractor-version_cmd)
- [Risk Scoring & Severity](#risk-scoring--severity)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Introduction

Sirius Scan templates are YAML files that define how to detect specific vulnerabilities or security misconfigurations on a host system. The agent parses these templates and executes the defined detection steps locally.

## Template Structure

A valid template consists of three main sections: `id`, `info`, and `detection`.

```yaml
id: UNIQUE-TEMPLATE-ID
info:
  name: Template Name
  author: Author Name
  severity: critical
  description: Description of what this detects.
detection:
  logic: all
  steps:
    - type: module_name
      config:
        key: value
```

### Metadata (`info`)

The `info` block contains metadata about the vulnerability.

| Field         | Type   | Required | Description                                             |
| ------------- | ------ | -------- | ------------------------------------------------------- |
| `name`        | string | Yes      | Human-readable name of the vulnerability.               |
| `author`      | string | No       | Name of the template author or organization.            |
| `severity`    | string | Yes      | One of: `critical`, `high`, `medium`, `low`, `info`.    |
| `description` | string | Yes      | Detailed description of the issue.                      |
| `references`  | list   | No       | URLs to external documentation or advisories.           |
| `cve`         | list   | No       | List of CVE IDs (e.g., `CVE-2024-1234`).                |
| `tags`        | list   | No       | Keywords for categorization (e.g., `ssh`, `rce`).       |
| `version`     | string | No       | Template version (e.g., `1.0`).                         |
| `cvss_vector` | string | No       | CVSS v3.x vector string (e.g., `CVSS:3.1/...`).         |
| `cvss_score`  | float  | No       | Pre-calculated CVSS score (0.0 - 10.0).                 |
| `risk_score`  | float  | No       | Custom risk score (0.0 - 10.0). Overrides other scores. |

### Detection Logic (`detection`)

The `detection` block defines the rules for identifying the vulnerability.

| Field   | Type   | Default | Description                                                                              |
| ------- | ------ | ------- | ---------------------------------------------------------------------------------------- |
| `logic` | string | `all`   | `all` (AND) requires all steps to match. `any` (OR) requires at least one step to match. |
| `steps` | list   | -       | A list of detection steps to execute.                                                    |

#### Detection Steps

Each step in the `steps` list requires:

- **`type`**: The name of the module to use (see [Detection Modules](#detection-modules)).
- **`config`**: Module-specific configuration map.
- **`platforms`** (optional): List of supported OSs (`linux`, `darwin`, `windows`). If omitted, runs on all.
- **`weight`** (optional): Float (0.0-1.0) indicating the step's importance (default 1.0).

## Detection Modules

The following modules are built-in and available for use in templates.

### File Hash Validator (`file_hash`)

Calculates the cryptographic hash of a file and compares it to a known hash. Useful for detecting specific vulnerable binary versions.

**Configuration:**

| Key         | Type   | Required | Description                                                  |
| ----------- | ------ | -------- | ------------------------------------------------------------ |
| `path`      | string | Yes      | Absolute path to the file.                                   |
| `hash`      | string | Yes      | Expected hash string.                                        |
| `algorithm` | string | No       | Hash algorithm: `sha256` (default), `sha1`, `md5`, `sha512`. |

**Example:**

```yaml
- type: file_hash
  config:
    path: /usr/sbin/sshd
    hash: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
    algorithm: sha256
```

### File Content Pattern Matcher (`file_content`)

Searches for regular expression patterns within a file's content. Useful for checking configuration files.

**Configuration:**

| Key         | Type   | Required | Description                                                           |
| ----------- | ------ | -------- | --------------------------------------------------------------------- |
| `path`      | string | Yes      | Absolute path to the file.                                            |
| `regex`     | string | Yes      | [Go-compatible regular expression](https://pkg.go.dev/regexp/syntax). |
| `multiline` | bool   | No       | Set to `true` to enable multiline matching. Default `false`.          |

**Example:**

```yaml
- type: file_content
  config:
    path: /etc/ssh/sshd_config
    regex: "^PermitRootLogin\\s+yes"
```

### Command Version Extractor (`version_cmd`)

Executes a system command and extracts version information from stdout/stderr using a regex.

**Configuration:**

| Key         | Type   | Required | Description                                                         |
| ----------- | ------ | -------- | ------------------------------------------------------------------- |
| `command`   | list   | Yes      | Command and arguments as a list of strings. **No shell expansion.** |
| `regex`     | string | Yes      | Regex with a capture group `()` for the version string.             |
| `exit_code` | int    | No       | Expected exit code. If matched, checks regex.                       |

**Example:**

```yaml
- type: version_cmd
  config:
    command: ["ssh", "-V"]
    regex: "OpenSSH_([0-9.]+)"
    exit_code: 0
```

## Risk Scoring & Severity

The agent calculates a final risk score for each finding. The priority for score calculation is:

1.  **`risk_score`**: If explicitly set, this value is used directly.
2.  **`cvss_vector`**: If valid, the base score is calculated from the vector.
3.  **`cvss_score`**: If no vector, this pre-calculated score is used.
4.  **`severity`**: If no numeric scores are present, a default score is assigned based on the severity level (Critical=9.0, High=7.0, etc.).

## Best Practices

1.  **Use Specific Regex**: Avoid overly broad regex patterns (like `.*`) that can cause performance issues (ReDoS).
2.  **Define Platforms**: If a check is OS-specific (e.g., `/etc/passwd`), always specify `platforms: [linux]`.
3.  **Prioritize File Hashes**: Hash checks are faster and more reliable than version string parsing. Use them when targeting specific binaries.
4.  **Avoid Shell Injection**: The `version_cmd` module executes commands directly, not through a shell. Do not use pipes `|`, redirects `>`, or shell built-ins.
5.  **Test Your Templates**: Verify templates on a test system before deploying. Use the agent's CLI to run a single template against a target.

## Examples

### Check for Weak SSH Configuration

```yaml
id: SSH-WEAK-CONFIG-001
info:
  name: SSH Root Login Enabled
  severity: high
  description: Detects if SSH root login is permitted.
detection:
  logic: all
  steps:
    - type: file_content
      platforms: [linux]
      config:
        path: /etc/ssh/sshd_config
        regex: "^PermitRootLogin\\s+yes"
```

### Check for Specific Vulnerable Binary (Hash)

```yaml
id: VULN-BIN-001
info:
  name: Vulnerable Service Binary
  severity: critical
  description: Detects a known vulnerable version of service_x via hash.
detection:
  logic: all
  steps:
    - type: file_hash
      config:
        path: /opt/service_x/bin/server
        hash: 5f4dcc3b5aa765d61d8327deb882cf99
```
