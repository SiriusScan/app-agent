# Template Architect's Guide

This guide provides a comprehensive reference for creating vulnerability detection templates for the Sirius Scan agent. It details the template structure, available detection modules, and best practices for template architects.

## Table of Contents

- [Introduction](#introduction)
- [Template Structure](#template-structure)
  - [Metadata (`info`)](#metadata-info)
  - [Detection Logic (`detection`)](#detection-logic-detection)
- [Detection Strategy Selection](#detection-strategy-selection)
  - [Module Comparison](#module-comparison)
  - [Critical Warning: version_cmd Limitations](#critical-warning-version_cmd-limitations)
  - [Decision Tree for CVE Detection](#decision-tree-for-cve-detection)
- [Detection Modules](#detection-modules)
  - [File Hash Validator (`file_hash`)](#file-hash-validator-file_hash)
  - [File Content Pattern Matcher (`file_content`)](#file-content-pattern-matcher-file_content)
  - [Command Version Extractor (`version_cmd`)](#command-version-extractor-version_cmd)
  - [File Search (`file_search`)](#file-search-file_search)
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

## Detection Strategy Selection

Before choosing a detection module, understand what each module does and its limitations.

### Module Comparison

| Module         | What It Does                                  | Best For                                           | Critical Limitation                                 |
| -------------- | --------------------------------------------- | -------------------------------------------------- | --------------------------------------------------- |
| `file_hash`    | Compares file hash against known hash         | Detecting specific vulnerable binary builds        | Requires pre-computed hashes of vulnerable files    |
| `file_content` | Matches regex pattern in a single file        | Misconfigurations, presence of vulnerable patterns | Pattern complexity, file path must be known         |
| `version_cmd`  | Extracts version string via command            | Detecting software presence, version extraction    | **Does NOT compare versions** - matches ANY version |
| `file_search`  | Recursively searches directories for files    | Supply chain attacks, finding packages across disk | Filesystem scans can be slow on large trees         |

### Critical Warning: version_cmd Limitations

> **⚠️ WARNING**: The `version_cmd` module extracts a version string but **does not compare it against a vulnerable range**. If you use `version_cmd` with a CVE that affects "versions 1.0 through 2.5", the template will match ALL installations of that software, including patched versions.

**version_cmd is appropriate when:**

- You need to detect if software is installed (presence check)
- Version comparison is handled by the Sirius backend after extraction
- You're detecting ANY version of software (not version-specific)

**version_cmd is NOT appropriate when:**

- You need to detect only specific vulnerable versions
- False positives on patched systems would be problematic

### Decision Tree for CVE Detection

```
Is this a version-range CVE? (e.g., "versions 1.0-2.5 vulnerable")
│
├─ YES: Do you know the exact file path?
│       │
│       ├─ YES: Can you obtain file hashes of vulnerable binaries?
│       │       │
│       │       ├─ YES → Use file_hash (most precise)
│       │       │
│       │       └─ NO → Use file_content to match version in a known file
│       │
│       └─ NO: Is the file in an unknown location? (e.g., npm packages, app bundles)
│               │
│               ├─ YES → Use file_search with filename + path + content filters
│               │
│               └─ NO → Use version_cmd BUT:
│                       - Document that version comparison is NOT done in template
│                       - Accept that ALL versions will be flagged
│                       - Version filtering happens in Sirius backend
│
└─ NO: Is this a configuration issue?
        │
        ├─ YES: Do you know the config file path?
        │       │
        │       ├─ YES → Use file_content with specific pattern
        │       │
        │       └─ NO → Use file_search to find config files across the filesystem
        │
        └─ NO → Use appropriate module for detection type
```

### Recommended Approach for CVEs

For CVEs affecting specific version ranges, the **recommended approach** is:

1. **Primary**: Use `file_hash` with hashes of known vulnerable binaries
2. **Fallback**: Use `version_cmd` with `logic: any` to provide multiple detection paths
3. **Document**: Always note in the template description which versions are affected

**Example combining approaches:**

```yaml
detection:
  logic: any # Match if ANY step succeeds
  steps:
    # Primary: Hash of known vulnerable binary
    - type: file_hash
      config:
        path: /usr/lib/libc.so
        hash: abc123... # Hash of vulnerable version
    # Fallback: Version extraction (note: matches ANY version)
    - type: version_cmd
      config:
        command: ["/lib/ld-musl-x86_64.so.1"]
        regex: "Version ([0-9.]+)"
```

---

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

### File Search (`file_search`)

Recursively walks a directory tree, matches files by name (glob) and path (regex), and optionally applies a content regex to each matched file. Returns structured evidence listing every match with path, matched text, and line number. Ideal for supply chain attacks, finding installed packages, and locating configuration files across unknown paths.

**Configuration:**

| Key                | Type     | Required | Description                                                                                                    |
| ------------------ | -------- | -------- | -------------------------------------------------------------------------------------------------------------- |
| `root_path`        | string   | Yes      | Starting directory for the search (e.g., `/`, `/home`, `.`).                                                   |
| `filename_pattern` | string   | No*      | Glob pattern matched against filenames (e.g., `package.json`, `*.lock`).                                       |
| `path_regex`       | string   | No*      | [Go regex](https://pkg.go.dev/regexp/syntax) matched against full file paths (e.g., `node_modules/axios/`).    |
| `content_regex`    | string   | No*      | [Go regex](https://pkg.go.dev/regexp/syntax) applied to file content of files that pass filename/path filters. |
| `max_depth`        | int      | No       | Maximum directory recursion depth. Default `20`.                                                               |
| `max_results`      | int      | No       | Maximum matches to collect. Default `100`.                                                                     |
| `exclude_dirs`     | []string | No       | Directory names to skip entirely. Default `[".git"]`.                                                          |

\* At least one of `filename_pattern`, `path_regex`, or `content_regex` must be provided. All provided criteria are AND-ed: a file must pass every filter to count as a match.

**Evidence output** includes:

- `matches` — array of objects with `path`, `matched_text`, and `line_number`
- `files_scanned` — number of files tested against content regex
- `files_matched` — number of matching files collected
- `truncated` — `true` if `max_results` was reached

**Example — Detect compromised npm package:**

```yaml
- type: file_search
  config:
    root_path: /
    filename_pattern: "package.json"
    path_regex: "node_modules/axios/package\\.json$"
    content_regex: '"version":\s*"(1\.14\.1|0\.30\.4)"'
    max_depth: 30
    exclude_dirs: [".git", "proc", "sys", "dev", "run"]
```

**Example — Find all SSH config files allowing root login:**

```yaml
- type: file_search
  config:
    root_path: /etc
    filename_pattern: "sshd_config"
    content_regex: "^PermitRootLogin\\s+yes"
```

**Example — Locate lockfiles containing a vulnerable package:**

```yaml
- type: file_search
  config:
    root_path: /home
    filename_pattern: "*.lock"
    content_regex: "vulnerable-pkg@2\\.3\\.1"
    max_depth: 10
```

**Performance notes:**

- Scanning from `/` on large filesystems can be slow. Prefer a narrower `root_path` when the target area is known.
- Use `exclude_dirs` to skip virtual filesystems (`proc`, `sys`, `dev`) and large irrelevant trees.
- `max_results` prevents unbounded memory use when many matches exist; evidence includes `truncated: true` so you know results were capped.

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
6.  **Scope File Searches**: When using `file_search`, prefer the narrowest `root_path` that covers the target area. Exclude virtual filesystems (`proc`, `sys`, `dev`) and irrelevant trees to keep scans fast.
7.  **Use file_search for Unknown Paths**: If you don't know where a file lives (e.g., npm packages, user-installed software), use `file_search` instead of guessing paths with `file_content`.

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

### Detect Compromised npm Package (Supply Chain)

```yaml
id: AXIOS-SUPPLY-CHAIN-2026-001
info:
  name: Compromised axios npm Package (Supply Chain Attack)
  severity: critical
  description: >
    Detects compromised axios versions 1.14.1 and 0.30.4 installed in
    node_modules anywhere on the host. See templates/examples/axios-supply-chain.yaml
    for the complete template with metadata and remediation guidance.
detection:
  logic: all
  steps:
    - type: file_search
      config:
        root_path: /
        filename_pattern: "package.json"
        path_regex: "node_modules/axios/package\\.json$"
        content_regex: '"version":\s*"(1\.14\.1|0\.30\.4)"'
        max_depth: 30
        exclude_dirs: [".git", "proc", "sys", "dev", "run"]
```
