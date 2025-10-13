# Agent Enhancement Project Structure

**Status**: ✅ Task 0.4 COMPLETED - Project structure established and ready for implementation

This document outlines the complete project structure established for the Sirius Agent Enhancement initiative, implementing the comprehensive vulnerability detection framework described in the PRD.

## 📁 Directory Structure

```
app-agent/
├── internal/
│   ├── detect/                    # Core vulnerability detection framework
│   │   ├── interfaces.go          # Detection engine interfaces
│   │   ├── types.go               # Detection data structures
│   │   ├── template/              # YAML template processing
│   │   ├── script/                # Script execution framework
│   │   └── hash/                  # File hash detection
│   ├── fingerprint/               # System fingerprinting
│   │   ├── interfaces.go          # Fingerprinting interfaces
│   │   ├── types.go               # System information structures
│   │   ├── system.go              # Hardware detection (placeholder)
│   │   ├── network.go             # Network configuration (placeholder)
│   │   ├── users.go               # User enumeration (placeholder)
│   │   └── services.go            # Service detection (placeholder)
│   └── repository/                # Template/script repository management
│       ├── interfaces.go          # Repository management interfaces
│       ├── types.go               # Repository data structures
│       ├── manager.go             # Repository operations (placeholder)
│       ├── updater.go             # Remote synchronization (placeholder)
│       └── validator.go           # Content validation (placeholder)
├── templates/                     # Vulnerability detection templates
│   ├── hash-based/               # File hash detection templates
│   │   └── example-apache-vuln.yaml  # Example Apache vulnerability template
│   ├── registry-based/           # Windows registry detection templates
│   └── config-based/             # Configuration file detection templates
├── scripts/                      # Vulnerability detection scripts
│   ├── windows/                  # PowerShell detection scripts
│   │   └── example-service-permissions.ps1  # Example Windows script
│   ├── linux/                    # Bash detection scripts
│   │   └── example-suid-analysis.sh         # Example Linux script
│   └── cross-platform/          # Python/other cross-platform scripts
├── go.mod                        # Updated with YAML and crypto dependencies
└── PROJECT-STRUCTURE.md         # This documentation
```

## 🔧 Core Components Established

### 1. Detection Framework (`internal/detect/`)

**Interfaces Created:**

- `DetectionEngine` - Main orchestrator for vulnerability detection
- `TemplateEngine` - YAML template parsing and execution
- `ScriptEngine` - Script execution with security controls
- `HashEngine` - File hash calculations and matching

**Data Structures:**

- `DetectionReport` - Comprehensive detection results
- `DetectionResult` - Individual detection method results
- `Evidence` - Proof structures for vulnerability findings
- `VulnTemplate` - YAML template representation
- `DetectionScript` - Script metadata and execution info

### 2. Fingerprinting Framework (`internal/fingerprint/`)

**Interfaces Created:**

- `SystemFingerprinter` - Main system information collector
- `HardwareCollector` - CPU, memory, storage detection
- `NetworkCollector` - Network interfaces and configuration
- `UserCollector` - User accounts and privileges
- `ServiceCollector` - Running services and processes
- `CertificateCollector` - Certificate store enumeration

**Data Structures:**

- `SystemFingerprint` - Complete system profile
- `HardwareInfo`, `NetworkInfo`, `UserInfo`, etc. - Detailed component information
- Cross-platform type definitions supporting Windows, Linux, and macOS

### 3. Repository Management (`internal/repository/`)

**Interfaces Created:**

- `RepositoryManager` - Template/script repository operations
- `UpdateManager` - Remote synchronization and versioning
- `SignatureValidator` - GPG signature verification
- `ContentValidator` - Security and integrity validation
- `RemoteRepository` - Remote repository communication

**Data Structures:**

- `Manifest` - Repository manifest with file metadata
- `UpdateResult` - Update operation results
- `ValidationResult` - Content validation outcomes
- `RepositoryConfiguration` - Repository settings and security controls

## 🎯 Template System

### YAML Template Format

Created comprehensive template structure supporting:

- **File Hash Detection** - SHA256/SHA1/MD5 verification
- **Registry Detection** - Windows registry key/value checking
- **Configuration Analysis** - Regex pattern matching in config files
- **Cross-platform Targeting** - Platform-specific detection logic
- **Remediation Guidance** - Automated fix recommendations

### Example Template (`templates/hash-based/example-apache-vuln.yaml`)

```yaml
id: "EXAMPLE-2024-001"
info:
  name: "Vulnerable Apache Binary Detection"
  severity: "high"
  cve: "CVE-2023-25690"
detection:
  type: "file-hash"
  targets:
    - path: "/usr/sbin/apache2"
      hash: "a1b2c3d4e5f6789abcdef..."
      platform: ["linux"]
```

## 🔒 Script Framework

### Security-First Design

- **Execution Sandboxing** - Privilege dropping and resource limits
- **Timeout Controls** - Prevent runaway execution
- **Parameter Validation** - Type checking and sanitization
- **Standardized Output** - JSON result format for all scripts

### Example Scripts

- **Windows PowerShell** (`scripts/windows/example-service-permissions.ps1`)
  - Service permission vulnerability detection
  - Uses `icacls` for permission analysis
  - Configurable service targeting
- **Linux Bash** (`scripts/linux/example-suid-analysis.sh`)
  - SUID binary analysis for privilege escalation
  - Pattern matching for dangerous binaries
  - Comprehensive evidence collection

## 📦 Dependencies Added

Updated `go.mod` with required libraries:

```go
require (
    // ... existing dependencies ...
    gopkg.in/yaml.v3 v3.0.1        // YAML template parsing
    golang.org/x/crypto v0.32.0    // Cryptographic operations
)
```

## 🚀 Implementation Readiness

### ✅ Completed in Task 0.4

- [x] Complete interface definitions for all major components
- [x] Comprehensive type system supporting all detection methods
- [x] Directory structure matching PRD architecture
- [x] Example templates and scripts demonstrating format
- [x] Updated dependencies for YAML and cryptographic operations
- [x] Cross-platform data structures (Windows/Linux/macOS)
- [x] Security-focused script execution framework
- [x] Repository management with GPG signature support

### 🔄 Ready for Next Phases

**Phase 1: Enhanced SBOM and System Fingerprinting**

- Interfaces defined in `internal/fingerprint/`
- Database integration structures ready
- Platform-specific implementation placeholders created

**Phase 2: Template-Based Vulnerability Detection**

- Template engine interfaces and types complete
- Example YAML templates demonstrate format
- Hash, registry, and config detection structures ready

**Phase 3: Script-Based Vulnerability Detection**

- Script execution framework interfaces defined
- Security sandbox architecture planned
- Example PowerShell and Bash scripts created

**Phase 4: Repository Management**

- Complete repository management interfaces
- Update, validation, and signature verification ready
- Manifest format and versioning system defined

## 🔍 Next Steps

With Task 0.4 complete, the project is ready to proceed with:

1. **Task 1.1**: Extend scan command data structures
2. **Task 1.2**: Implement system hardware fingerprinting
3. **Task 1.3**: Implement network configuration detection
4. **Task 1.4**: Implement certificate store inventory
5. **Task 1.5**: Enhance package detection with metadata
6. **Task 1.6**: Integrate SBOM data with database

The foundation is now in place for all planned agent enhancement features, with a clear separation of concerns, comprehensive type safety, and security-first design principles throughout.

---

**Task 0.4 Status**: ✅ **COMPLETED**  
**Next Task**: Task 1.1 - Extend Scan Command Data Structures  
**Dependencies**: All Phase 0 tasks completed, ready for Phase 1 implementation
