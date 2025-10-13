
# Template‑Based Vulnerability Detection System

## Version and Overview

- **Document version:** 0.1 (Draft)
- **Date:** 13 October 2025
- **Prepared by:** Steering Committee (Sirius Project)

This document defines requirements for adding a **template‑based vulnerability detection** capability to the Sirius scanning agent.  The goal is to support YAML‑defined templates for detecting software vulnerabilities on the host, while ensuring the agent remains modular, maintainable and easy to test.  The agent currently supports script‑based discovery via Python/PowerShell/Bash scripts; the new system will introduce a second mechanism for scanning based on simple YAML templates.  The broader vulnerability scanner will continue to run an agent/server architecture where the agent communicates with a central server over gRPC, but the agent must also support offline operation for rapid iteration.

### Background and Context

1. **Nuclei‑style templates.**  ProjectDiscovery’s Nuclei engine uses **YAML‑based templates** to define the steps required to detect a vulnerability; its templates contain an `id`, metadata and protocol‑specific detection stepshttps://projectdiscovery.io/blog/ultimate-nuclei-guide#:~:text=Nuclei%20is%20a%20fast%2C%20efficient%2C,in%20just%20a%20few%20minutes.  The markup tells the scanner what to send to a host and how to examine the responsehttps://projectdiscovery.io/blog/ultimate-nuclei-guide#:~:text=A%20Nuclei%20template%20is%20a,vulnerable%20to%20a%20certain%20issue.  The new agent should use a similar concept for host‑side checks rather than network requests.
2. **File‑based detection.**  Nuclei’s file mode works with files on the local file system, allowing the scanner to find files and extract data from themhttps://projectdiscovery.io/blog/ultimate-nuclei-guide#:~:text=Works%20with%20files%20on%20the,extraction%20of%20data%20within%20them.  An example template uses a `file:` section that specifies file extensions and a regex extractorhttps://projectdiscovery.io/blog/ultimate-nuclei-guide#:~:text=Works%20with%20files%20on%20the,extraction%20of%20data%20within%20them.  This approach can inspire modules such as file presence, hash and regex matching.
3. **Existing scanning agent.**  The current Sirius agent executes scripts via Python/PowerShell/Bash and reports results back to a gRPC server.  The repository is messy and poorly documented.  We aim to refactor the agent to support modular template processors and provide standalone execution for testing.

## Objectives

The sprint focuses on building a minimum viable template‑based detection system that can be extended over time.  Key objectives:

1. **Modular architecture:** Implement a plugin‑like module for each type of detection (e.g., file hash, version detection, script execution).  Each module lives in its own Go package/directory with all code necessary to execute that detection type.  Shared utilities reside in a common library to adhere to DRY principles.
2. **YAML template DSL:** Design a YAML schema for templates that describe one vulnerability or fingerprinting goal.  Templates contain an ID, metadata (name, author, severity, description, tags, CVE), a list of detection steps and an optional `test_strategy` section (instructions for manual verification).
3. **Standalone execution:** Provide CLI flags (e.g., `--standalone`, `--template <path|id>`) so the agent can run locally without connecting to the server.  This enables rapid development and testing in a controlled container environment.
4. **Containerised development:** Use a standard Ubuntu container as the test environment so that all developers can reproduce scans.  Development tasks will run the agent inside the container, mount test data and iterate quickly.
5. **Documentation alignment:** Document each module and the overall template system in the project’s documentation style (similar to the Task Management System guidelines).  Provide examples and guidelines for writing new templates.
6. **MVP modules:** Deliver only a few basic modules in the first sprint – file presence/hash matching, file content/regex matching, command execution for version detection, and generic script execution (shell/Python/PowerShell).  Document additional module ideas for future sprints.

## Scope

### In Scope

- Designing the YAML template schema and building a parser that converts a template into internal structures.
- Refactoring the agent to support modular detection modules; each module has its own Go package and implements a standard interface.
- Implementing the initial set of modules: **FileHashModule**, **FileContentModule**, **CommandVersionModule**, **ScriptModule**.
- Providing a local template repository on the agent; templates are downloaded from the server or provided via CLI and stored with versioning and signatures.
- Adding standalone execution via CLI flags and ensuring detection results are printed to stdout or saved to a file.
- Creating developer documentation and examples for writing templates and running the agent in both standalone and server mode.

### Out of Scope

- Building UI components for creating or editing templates.  Those will be handled by the central scanner’s web interface.
- Redesigning the server component; the focus is entirely on the agent.
- Implementing advanced modules (e.g., package manager, registry checks, process enumeration) in this sprint; these are documented as future enhancements.

## Requirements

### 1. Module Architecture

- **Separate package per module.**  Each detection type (e.g., file hash, version check) will be implemented as a standalone Go package located under `internal/modules/<module‑name>/`.  The package should contain all logic to parse the module’s specific configuration from a detection step and execute it on the host system.  For example:
  - `internal/modules/filehash` handles reading a file and computing its cryptographic hash.
  - `internal/modules/filecontent` handles opening a file and searching for strings or regex patterns.
  - `internal/modules/versioncmd` runs a command to retrieve version information.
  - `internal/modules/scriptexec` executes arbitrary scripts (shell, Python, PowerShell).
- **Shared utilities.**  Common functions (e.g., OS detection, file operations, regex helpers, hashing) live in `internal/common/`.  Modules depend on these utilities rather than re‑implementing them.
- **Module interface.**  Each module implements a common interface with methods such as `Configure(d DetectionConfig) error`, `Execute(ctx Context) (Result, error)`.  The agent core iterates through detection steps and dispatches them to the appropriate module based on the `type` field.
- **Operating system abstraction.**  Where necessary (e.g., script execution or version detection), modules should abstract away OS differences.  Platform‑specific implementations can live under subdirectories (e.g., `scriptexec/windows/`, `scriptexec/linux/`) and be selected at runtime.
- **Extensibility.**  Adding a new detection type requires creating a new module package and registering it in a module registry.  The YAML schema should not need to change for each new module.

### 2. YAML Template Schema

A template describes a single vulnerability or fingerprinting goal and may contain multiple detection steps.  This structure avoids duplicating metadata across many single‑step templates and lets the system combine evidence from multiple checks.  Templates not tied to specific vulnerabilities (e.g., fingerprinting) can still follow the same structure.

**Top‑level fields**:

```yaml
id: openssh-cve-2024-1117
info:
  name: OpenSSH 6.5.1 RCE
  author: security-team
  severity: critical
  description: Detects OpenSSH version 6.5.1 which is vulnerable to CVE‑2024‑1117.
  tags: [openssh, version, cve-2024-1117]
  cve: CVE-2024-1117

# detection: an array of steps; each step is dispatched to a module

detection:
  - type: file_hash
    path: /usr/sbin/sshd
    hash: "4f5f9..."
  - type: file_content
    path: /usr/sbin/sshd
    regex: "OpenSSH_6\\.5\\.1"
  - type: command_version
    command: ["ssh", "-V"]
    regex: "OpenSSH_6\\.5\\.1"
    exit_code: 0
  - type: script
    interpreter: bash
    script: |
      dpkg-query -W -f='${Version}' openssh-server
    regex: "^6.5.1"

# optional instructions for human testers; not executed by the agent

test_strategy: |
  Run `ssh -V` on the host and verify that the version string matches 6.5.1.  Also
  check installed package version using `dpkg -l openssh-server`.

remediation: |
  Update OpenSSH to the latest patched version or apply vendor hotfix.
```

- **`id`:** Unique identifier for the template.  It must be globally unique and not contain spaces.
- **`info`:** Metadata about the template, including severity and a CVE identifier where applicable.  Nuclei templates include a similar `info` blockhttps://docs.projectdiscovery.io/templates/structure#:~:text=,data%20retrieval%20from%20the%20results.
- **`detection`:** An ordered list of detection steps.  Each entry contains:
  - `type`: Module name.
  - Module‑specific configuration (e.g., `path`, `hash`, `regex`, `command`).
  - Optional fields such as `condition` (`any`/`all`) or `exit_code`.  By default, all steps must succeed for the vulnerability to be considered present.
- **`test_strategy`:** Free‑form text describing how a human operator can manually verify the detection.  This field is **not** executed by the agent; it serves as a guide for security engineers.
- **`remediation`:** Guidance for fixing the vulnerability.

### 3. Template Repository and Storage

- **Local cache.**  The agent maintains a local directory (e.g., `~/.sirius/templates`) containing downloaded templates.  Templates are organised by category or CVE and versioned.  A digital signature field (similar to Nuclei’s template digesthttps://docs.projectdiscovery.io/templates/structure#:~:text=,data%20retrieval%20from%20the%20results) will ensure integrity.
- **Distribution.**  On startup, if connected to the server, the agent checks for updated templates and downloads new versions.  When run in standalone mode, the agent reads templates from the local cache or from a path specified by `--template`.
- **Custom templates.**  Engineers can drop custom templates into the cache directory or supply them via CLI (`--template /path/to/template.yaml`).  The agent validates template structure before execution.

### 4. Agent Execution Flow

1. **Initialisation.**  The agent loads configuration (server address, agent ID, etc.), determines whether to run in standalone mode based on CLI flags, and initializes the module registry.
2. **Template selection.**  In server mode, the agent receives a list of template IDs to run from the server.  In standalone mode, the user specifies a template file or directory.
3. **Parsing.**  The agent parses selected templates into internal structures.  Parsing errors are reported and faulty templates are skipped.
4. **Execution.**  For each detection step, the agent dispatches to the corresponding module.  The module returns success/failure and any extracted data (e.g., matched regex, computed hash).  If any step fails and the template’s logic requires all steps to succeed, the template evaluation stops.
5. **Reporting.**  In server mode, results are transmitted back via gRPC.  In standalone mode, results are output as JSON or human‑readable text.

### 5. Development Mode and Testing

- **Containerised environment.**  Use a standard Ubuntu container for development and testing.  Running the agent inside the container ensures a clean environment and predictable package manager behaviour.  Scripts and command modules can rely on `apt`/`dpkg` for version checks.  Additional containers (e.g., Windows container) can be introduced in future sprints to support cross‑platform modules.
- **Automated testing.**  Each module should include unit tests to validate its behaviour and integration tests that run the agent against known vulnerable and non‑vulnerable samples.  Test templates should be stored in a separate `testdata/` directory.
- **Test strategy field.**  The `test_strategy` section in templates does **not** drive detection.  It is for human guidance; detection logic stays under `detection`.

### 6. MVP Modules

Only a subset of modules will be implemented during this sprint.  Additional ideas are documented for future work.

#### 6.1 FileHashModule

- Reads a specified file and computes a cryptographic hash (SHA‑256 by default).  Compares the result to a list of expected hashes.  Useful for detecting vulnerable binaries or known malicious files.
- Config keys: `path` (string), `hash` (string or list), `algorithm` (optional, default sha256).
- Fails if the file does not exist or the hash does not match.

#### 6.2 FileContentModule

- Reads a file and searches for literal strings or regex patterns.  Inspired by Nuclei’s file extractors and matchershttps://projectdiscovery.io/blog/ultimate-nuclei-guide#:~:text=Works%20with%20files%20on%20the,extraction%20of%20data%20within%20them.  Supports multiple patterns and a `condition` field (`any`/`all`).
- Config keys: `path` (string), `regex` (list of strings), `string` (list of literal strings), `condition` (optional).
- Fails if the file does not exist or no pattern matches.

#### 6.3 CommandVersionModule

- Executes a command (e.g., `ssh -V`) and parses the version from stdout or stderr using regex.  Supports specifying an expected exit code.
- Config keys: `command` (array of strings), `regex` (string or list), `exit_code` (optional), `condition` (optional).
- Fails if the command cannot be executed or the version does not match.

#### 6.4 ScriptModule

- Executes an arbitrary script using a specified interpreter (bash, python, powershell).  Returns success based on exit code and optional regex/string matches in output.
- Config keys: `interpreter` (string), `script` (string), `regex` (list), `exit_code` (optional), `shell` (boolean to run via system shell).  For security, script execution must be constrained and validated.

### 7. Future Module Ideas (Not part of MVP)

1. **PackageManagerModule:** Query package managers (apt, yum, rpm, brew, chocolatey) for installed versions and compare against vulnerable ranges.
2. **ProcessModule:** Enumerate running processes and optionally extract version information.
3. **RegistryModule (Windows):** Check registry keys for software versions or insecure configurations.
4. **EnvVarModule:** Verify environment variables (e.g., insecure settings, presence of secret keys).
5. **CertificateModule:** Validate binary signatures and certificates to detect tampering.
6. **PortModule:** Check open ports and service banners; useful for local network scanning.
7. **CPEFingerprintModule:** Identify installed software using indicator files and generate CPE strings for inventory purposes.

These modules can be prioritized based on user demand and complexity in subsequent sprints.

### 8. Risks and Mitigations

| Risk | Mitigation |
|---|---|
| **Code complexity and fragmentation** | Enforce strict module boundaries; provide clear guidelines on where to place shared functions and how to add new modules.  Periodic code reviews will ensure adherence. |
| **Duplicate or conflicting templates** | Maintain a central template registry with unique IDs; use semantic versioning and digital signatures to avoid confusion.  Provide a process for updating existing templates rather than creating duplicates. |
| **Platform differences (Windows, macOS, Linux)** | Abstract OS‑specific operations behind interfaces; implement platform‑specific modules under separate packages.  Use containerisation for initial Ubuntu testing and expand to other environments later. |
| **Security risks from script execution** | Constrain script modules to run in controlled interpreters; restrict accessible resources; sign templates; audit scripts before deployment. |
| **Performance impact** | Use asynchronous execution where possible; support timeouts; measure overhead of each module. |

### 9. Success Metrics

- **Developer efficiency:** Standalone mode reduces iteration time by at least 50 % compared with deploying through the server.
- **Template adoption:** At least five real vulnerabilities are encoded as templates and successfully detected during this sprint.
- **Modularity:** New modules can be added without modifying existing modules or core logic (verified by implementing one additional module in a follow‑up sprint).
- **Documentation:** Comprehensive documentation and examples are published and reviewed by at least two developers.

## Appendix: Template Library Conventions

To avoid confusion when adding new detections:

- One template should normally correspond to one vulnerability or one logical fingerprinting goal.  It may contain multiple detection steps.  This approach prevents duplication of metadata and keeps templates easy to manage.  However, separate templates for different detection techniques that support the same vulnerability are acceptable if they serve distinct purposes and have distinct IDs.
- All templates must include a `test_strategy` and `remediation` section to guide engineers.
- Template files are stored under directories named after their category (e.g., `cves/2024/`, `fingerprints/`).
- Template IDs follow the pattern `<namespace>-<slug>`, e.g., `openssh-cve-2024-1117`.

---

_This PRD defines the minimum scope and structure for integrating template‑based vulnerability detection into the Sirius scanning agent.  Citations for design inspiration come from the Nuclei template system, which uses YAML files and separates metadata from detection logichttps://projectdiscovery.io/blog/ultimate-nuclei-guide#:~:text=Nuclei%20is%20a%20fast%2C%20efficient%2C,in%20just%20a%20few%20minuteshttps://projectdiscovery.io/blog/ultimate-nuclei-guide#:~:text=A%20Nuclei%20template%20is%20a,vulnerable%20to%20a%20certain%20issuehttps://projectdiscovery.io/blog/ultimate-nuclei-guide#:~:text=Works%20with%20files%20on%20the,extraction%20of%20data%20within%20them._
