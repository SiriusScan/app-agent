# Sirius Template Development Guide

This guide explains how to create custom vulnerability detection templates for the Sirius Agent template-based scanning system.

## Overview

Sirius Agent uses YAML-based templates to detect vulnerabilities through multiple detection methods:

- **File Hash Detection**: Identify vulnerable files by their cryptographic hash
- **Registry Detection**: Scan Windows registry for vulnerable configurations
- **Configuration File Detection**: Use regex patterns to find misconfigurations

## Template Structure

All templates follow a standardized YAML format with the following sections:

```yaml
---
id: "UNIQUE-TEMPLATE-ID"
info:
  name: "Template Name"
  author: "Your Name"
  severity: "high|medium|low"
  description: "Description of what this template detects"
  references:
    - "https://reference-url.com"
  cve: "CVE-YYYY-NNNN" # Optional
  tags:
    - "tag1"
    - "tag2"
  version: "1.0"

detection:
  type: "file-hash|registry|config-file"
  method: "sha256|sha1|md5" # For file-hash only
  targets:
    # Detection targets (see sections below)

  conditions:
    # Detection conditions
    - type: "file_exists|hash_match|key_exists|value_matches_pattern"
      value: true|false|"regex_pattern"

  metadata:
    confidence: 0.95 # 0.0 to 1.0
    impact: "Description of potential impact"

remediation:
  description: "How to fix this vulnerability"
  commands:
    linux: "command to run on Linux"
    windows: "command to run on Windows"
    darwin: "command to run on macOS"
  verification:
    command: "command to verify fix"
    expected_exit_code: 0
---
```

## Detection Types

### 1. File Hash Detection

Detect vulnerable files by their cryptographic hash:

```yaml
detection:
  type: "file-hash"
  method: "sha256" # sha256, sha1, or md5
  targets:
    - path: "/path/to/vulnerable/file"
      hash: "a1b2c3d4e5f6..." # SHA256 hash
      description: "Description of the vulnerable file"
      platform: ["linux", "darwin"] # Optional platform targeting
```

**Example**: Detect a vulnerable Apache configuration file

```yaml
detection:
  type: "file-hash"
  method: "sha256"
  targets:
    - path: "/etc/apache2/apache2.conf"
      hash: "a724b4564df7efb37dba9ffb18fd857212d2412279c86c6a1425fa2b393ac1a7"
      description: "Vulnerable Apache configuration file"
      platform: ["linux"]
```

### 2. Registry Detection

Scan Windows registry for vulnerable configurations:

```yaml
detection:
  type: "registry"
  targets:
    - key: "HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System"
      value_name: "EnableLUA"
      expected_value: "0"
      description: "UAC disabled"
      platform: ["windows"]
```

**Example**: Detect disabled Windows Defender

```yaml
detection:
  type: "registry"
  targets:
    - key: "HKLM\\SOFTWARE\\Microsoft\\Windows Defender"
      value_name: "DisableAntiSpyware"
      expected_value: "1"
      description: "Windows Defender disabled"
      platform: ["windows"]
```

### 3. Configuration File Detection

Use regex patterns to find misconfigurations:

```yaml
detection:
  type: "config-file"
  targets:
    - path: "/etc/ssh/sshd_config"
      patterns:
        - pattern: "PermitRootLogin\\s+yes"
          description: "Root login enabled"
      description: "SSH configuration file"
      platform: ["linux", "darwin"]
```

**Example**: Detect weak SSH configuration

```yaml
detection:
  type: "config-file"
  targets:
    - path: "/etc/ssh/sshd_config"
      patterns:
        - pattern: "PasswordAuthentication\\s+yes"
          description: "Password authentication enabled"
        - pattern: "PermitEmptyPasswords\\s+yes"
          description: "Empty passwords allowed"
      description: "SSH security configuration"
      platform: ["linux", "darwin"]
```

## Conditions

Conditions define when a template should trigger:

```yaml
conditions:
  - type: "file_exists"
    value: true
  - type: "hash_match"
    value: true
  - type: "key_exists"
    value: true
  - type: "value_matches_pattern"
    value: "regex_pattern"
```

## Platform Targeting

Templates can target specific platforms:

```yaml
platform: ["linux", "windows", "darwin"]  # All platforms
platform: ["linux"]                        # Linux only
platform: ["windows", "darwin"]           # Windows and macOS
```

## Severity Levels

- **high**: Critical vulnerabilities that require immediate attention
- **medium**: Important vulnerabilities that should be addressed soon
- **low**: Minor issues that can be addressed during regular maintenance

## Remediation

Provide clear instructions for fixing detected vulnerabilities:

```yaml
remediation:
  description: "Remove the vulnerable file"
  commands:
    linux: "rm /path/to/vulnerable/file"
    windows: "Remove-Item 'C:\\path\\to\\vulnerable\\file'"
    darwin: "rm /path/to/vulnerable/file"
  verification:
    command: "ls /path/to/vulnerable/file"
    expected_exit_code: 2 # File not found
```

## Template Best Practices

### 1. Unique IDs

Use descriptive, unique IDs:

```yaml
id: "CVE-2024-1234-APACHE-CONFIG"
id: "WINDOWS-DEFENDER-DISABLED"
id: "SSH-WEAK-CONFIG-001"
```

### 2. Comprehensive Metadata

Include all relevant information:

```yaml
info:
  name: "Apache ServerTokens Information Disclosure"
  author: "Security Team"
  severity: "medium"
  description: "Apache ServerTokens directive reveals version information"
  references:
    - "https://httpd.apache.org/docs/2.4/mod/core.html#servertokens"
    - "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2024-1234"
  cve: "CVE-2024-1234"
  tags:
    - "apache"
    - "information-disclosure"
    - "web-server"
  version: "1.0"
```

### 3. Accurate Confidence Scoring

Set confidence based on detection reliability:

- **0.9-1.0**: High confidence (exact hash match, specific registry key)
- **0.7-0.9**: Medium confidence (pattern match, multiple conditions)
- **0.5-0.7**: Lower confidence (heuristic detection)

### 4. Clear Remediation Steps

Provide specific, actionable remediation:

```yaml
remediation:
  description: "Disable ServerTokens to prevent information disclosure"
  commands:
    linux: "echo 'ServerTokens Prod' >> /etc/apache2/apache2.conf && systemctl reload apache2"
    darwin: "echo 'ServerTokens Prod' >> /etc/apache2/httpd.conf && apachectl restart"
  verification:
    command: "grep 'ServerTokens Prod' /etc/apache2/apache2.conf"
    expected_exit_code: 0
```

## Template Validation

Before submitting templates, validate them:

1. **YAML Syntax**: Ensure valid YAML format
2. **Required Fields**: Verify all required fields are present
3. **Hash Accuracy**: For file-hash templates, verify hash values
4. **Platform Compatibility**: Test on target platforms
5. **Remediation Testing**: Verify remediation commands work

## Template Repository Structure

Templates are organized in the `sirius-agent-modules` repository:

```
templates/
├── hash-based/          # File hash detection templates
├── registry-based/      # Windows registry templates
└── config-based/        # Configuration file templates
```

## Submitting Templates

1. **Fork** the `sirius-agent-modules` repository
2. **Create** your template in the appropriate directory
3. **Test** the template thoroughly
4. **Submit** a pull request with:
   - Template file
   - Updated manifest (if needed)
   - Description of the vulnerability
   - Testing results

## Example Templates

### File Hash Template

```yaml
---
id: "CVE-2024-1234-APACHE-CONFIG"
info:
  name: "Apache Vulnerable Configuration File"
  author: "Security Team"
  severity: "high"
  description: "Detects vulnerable Apache configuration file"
  references:
    - "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2024-1234"
  cve: "CVE-2024-1234"
  tags:
    - "apache"
    - "web-server"
    - "configuration"
  version: "1.0"

detection:
  type: "file-hash"
  method: "sha256"
  targets:
    - path: "/etc/apache2/apache2.conf"
      hash: "a724b4564df7efb37dba9ffb18fd857212d2412279c86c6a1425fa2b393ac1a7"
      description: "Vulnerable Apache configuration file"
      platform: ["linux"]

  conditions:
    - type: "file_exists"
      value: true
    - type: "hash_match"
      value: true

  metadata:
    confidence: 0.99
    impact: "Apache configuration vulnerability could lead to security compromise"

remediation:
  description: "Replace vulnerable Apache configuration file"
  commands:
    linux: "cp /etc/apache2/apache2.conf /etc/apache2/apache2.conf.backup && echo 'ServerTokens Prod' > /etc/apache2/apache2.conf"
  verification:
    command: "grep 'ServerTokens Prod' /etc/apache2/apache2.conf"
    expected_exit_code: 0
---
```

### Registry Template

```yaml
---
id: "WINDOWS-DEFENDER-DISABLED"
info:
  name: "Windows Defender Disabled"
  author: "Security Team"
  severity: "high"
  description: "Detects when Windows Defender is disabled"
  references:
    - "https://docs.microsoft.com/en-us/windows/security/threat-protection/windows-defender-antivirus/"
  tags:
    - "windows"
    - "antivirus"
    - "security"
  version: "1.0"

detection:
  type: "registry"
  targets:
    - key: "HKLM\\SOFTWARE\\Microsoft\\Windows Defender"
      value_name: "DisableAntiSpyware"
      expected_value: "1"
      description: "Windows Defender disabled"
      platform: ["windows"]

  conditions:
    - type: "key_exists"
      value: true
    - type: "value_matches_pattern"
      value: "1"

  metadata:
    confidence: 0.95
    impact: "System vulnerable to malware without antivirus protection"

remediation:
  description: "Enable Windows Defender"
  commands:
    windows: "Set-MpPreference -DisableRealtimeMonitoring $false"
  verification:
    command: "Get-MpPreference | Select-Object DisableRealtimeMonitoring"
    expected_exit_code: 0
---
```

### Configuration File Template

```yaml
---
id: "SSH-WEAK-CONFIG"
info:
  name: "SSH Weak Configuration"
  author: "Security Team"
  severity: "medium"
  description: "Detects weak SSH configuration settings"
  references:
    - "https://www.openssh.com/manual.html"
  tags:
    - "ssh"
    - "configuration"
    - "security"
  version: "1.0"

detection:
  type: "config-file"
  targets:
    - path: "/etc/ssh/sshd_config"
      patterns:
        - pattern: "PermitRootLogin\\s+yes"
          description: "Root login enabled"
        - pattern: "PasswordAuthentication\\s+yes"
          description: "Password authentication enabled"
      description: "SSH configuration file"
      platform: ["linux", "darwin"]

  conditions:
    - type: "file_exists"
      value: true

  metadata:
    confidence: 0.8
    impact: "Weak SSH configuration could allow unauthorized access"

remediation:
  description: "Harden SSH configuration"
  commands:
    linux: "echo 'PermitRootLogin no' >> /etc/ssh/sshd_config && echo 'PasswordAuthentication no' >> /etc/ssh/sshd_config && systemctl reload sshd"
    darwin: "echo 'PermitRootLogin no' >> /etc/ssh/sshd_config && echo 'PasswordAuthentication no' >> /etc/ssh/sshd_config && launchctl unload /System/Library/LaunchDaemons/ssh.plist && launchctl load /System/Library/LaunchDaemons/ssh.plist"
  verification:
    command: "grep 'PermitRootLogin no' /etc/ssh/sshd_config"
    expected_exit_code: 0
---
```

## Testing Templates

1. **Create test files** that match your template
2. **Run the scan command** to verify detection
3. **Test remediation** commands manually
4. **Verify false positives** don't occur

## Support

For questions or issues with template development:

- **Repository**: https://github.com/SiriusScan/sirius-agent-modules
- **Documentation**: This guide and inline comments
- **Issues**: Use GitHub Issues for bug reports and feature requests

## Contributing

We welcome contributions! Please:

1. Follow the template structure and best practices
2. Test thoroughly before submitting
3. Include clear documentation
4. Provide example test cases
5. Update manifests when adding new templates

---

_This guide is maintained by the Sirius Security Team. For updates and improvements, please submit pull requests to the documentation repository._
