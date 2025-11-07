# Agent Template Risk Scoring System

## Overview

The Sirius agent template system supports **four different risk scoring mechanisms** with a **priority-based calculation system**. This flexible approach allows template authors to be as specific or general as they choose when assigning risk scores to vulnerabilities.

## Scoring Mechanisms

### Priority System

Risk scores are calculated using the following priority order:

1. **Custom Risk Score** (`risk_score`) - Highest priority
2. **CVSS Vector** (`cvss_vector`) - Second priority
3. **CVSS Score** (`cvss_score`) - Third priority
4. **Severity Mapping** (`severity`) - Lowest priority (fallback)

The system uses the highest-priority method available in the template. If a higher-priority method fails or is not provided, it falls back to the next available method.

## Method 1: Custom Risk Score (Highest Priority)

Assign an arbitrary numerical risk score from 0.0 to 10.0.

**Use Case:** When you have domain-specific knowledge or custom risk assessment that doesn't fit standard CVSS metrics.

**Example:**

```yaml
id: CUSTOM-VULN-001
info:
  name: Critical Business Logic Flaw
  author: Security Team
  severity: critical
  risk_score: 9.8 # Custom assessment
  description: Exploitable business logic vulnerability
```

**Validation:**

- Must be between 0.0 and 10.0
- Can use decimal precision (e.g., 8.5, 9.2)
- Overrides all other scoring methods

## Method 2: CVSS Vector (Second Priority)

Provide a CVSS v3.x vector string. The system automatically calculates the base score.

**Use Case:** When you want to provide detailed CVSS metrics and have the system calculate the score.

**Example:**

```yaml
id: CVE-2024-12345
info:
  name: Remote Code Execution
  author: Security Team
  severity: critical
  cvss_vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H"
  description: Network-accessible RCE vulnerability
```

**Supported Formats:**

- CVSS v3.0: `CVSS:3.0/...`
- CVSS v3.1: `CVSS:3.1/...`

**Required Metrics:**

- `AV` - Attack Vector (N/A/L/P)
- `AC` - Attack Complexity (L/H)
- `PR` - Privileges Required (N/L/H)
- `UI` - User Interaction (N/R)
- `S` - Scope (U/C)
- `C` - Confidentiality Impact (H/L/N)
- `I` - Integrity Impact (H/L/N)
- `A` - Availability Impact (H/L/N)

**Score Calculation:**
The system implements the official CVSS v3 base score formula. Temporal and environmental metrics are not supported.

## Method 3: CVSS Score (Third Priority)

Provide a pre-calculated CVSS score from 0.0 to 10.0.

**Use Case:** When you have a CVSS score from an external source (e.g., NVD) but don't need the full vector string.

**Example:**

```yaml
id: CVE-2024-67890
info:
  name: Authentication Bypass
  author: Security Team
  severity: high
  cvss_score: 8.1
  description: Bypass authentication controls
  references:
    - https://nvd.nist.gov/vuln/detail/CVE-2024-67890
```

**Validation:**

- Must be between 0.0 and 10.0
- Used as-is without calculation

## Method 4: Severity Mapping (Fallback)

Use only the qualitative severity level. The system maps it to a numerical score.

**Use Case:** Simple templates where detailed risk scoring isn't necessary. Maintains backward compatibility with existing templates.

**Example:**

```yaml
id: INFO-VULN-001
info:
  name: Information Disclosure
  author: Security Team
  severity: medium # Maps to 5.0
  description: Sensitive information in logs
```

**Severity Mappings:**

- `critical` → 9.5
- `high` → 7.5
- `medium` → 5.0
- `low` → 2.0
- `info` → 0.0

## Template Examples

### Example 1: Custom Score for Business-Specific Risk

```yaml
id: BUSINESS-CRITICAL-001
info:
  name: Financial Data Exposure
  author: Internal Security
  severity: critical
  risk_score: 9.9 # Business critical - highest score
  description: Direct exposure of customer financial records
  tags:
    - financial
    - pii
    - critical-business-impact

detection:
  logic: all
  steps:
    - type: file_content
      config:
        path: /var/www/api/users.php
        regex: "SELECT.*credit_card.*FROM users WHERE.*"
```

### Example 2: CVSS Vector for Standard CVE

```yaml
id: CVE-2024-APACHE-001
info:
  name: Apache HTTP Server RCE
  author: Security Research Team
  severity: critical
  cvss_vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H"
  description: Remote code execution in Apache HTTP Server
  cve:
    - CVE-2024-APACHE-001
  references:
    - https://httpd.apache.org/security/vulnerabilities_24.html
  tags:
    - apache
    - rce
    - network

detection:
  logic: all
  steps:
    - type: file_hash
      platforms:
        - linux
      config:
        path: /usr/sbin/httpd
        hash: abc123def456...
        algorithm: sha256
```

### Example 3: CVSS Score from NVD

```yaml
id: CVE-2024-OPENSSH-001
info:
  name: OpenSSH Authentication Bypass
  author: Security Team
  severity: high
  cvss_score: 7.8
  description: Authentication bypass in OpenSSH
  cve:
    - CVE-2024-OPENSSH-001
  references:
    - https://nvd.nist.gov/vuln/detail/CVE-2024-OPENSSH-001
    - https://www.openssh.com/security.html

detection:
  logic: any
  steps:
    - type: command_version
      platforms:
        - linux
        - darwin
      config:
        command: ["ssh", "-V"]
        regex: "OpenSSH_8\\.[0-3]"
```

### Example 4: Severity-Only (Simple Template)

```yaml
id: LOG-EXPOSURE-001
info:
  name: Sensitive Log File Exposure
  author: Security Team
  severity: low # Simple severity, maps to 2.0
  description: Sensitive information in world-readable log files
  tags:
    - logging
    - information-disclosure

detection:
  logic: all
  steps:
    - type: file_content
      config:
        path: /var/log/app.log
        regex: "API_KEY|SECRET|PASSWORD"
```

## Validation Rules

### General Validation

- At least one scoring method must be present (risk_score, cvss_vector, cvss_score, or severity)
- `severity` field is always required (used for categorization even if not used for scoring)
- Invalid or out-of-range values will cause template validation to fail

### Risk Score Validation

```yaml
# ✅ Valid
risk_score: 8.5
risk_score: 0.0
risk_score: 10.0

# ❌ Invalid
risk_score: -1.0   # Below minimum
risk_score: 10.5   # Above maximum
risk_score: null   # Use absence instead
```

### CVSS Vector Validation

```yaml
# ✅ Valid
cvss_vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H"
cvss_vector: "CVSS:3.0/AV:L/AC:H/PR:L/UI:R/S:U/C:L/I:L/A:N"

# ❌ Invalid
cvss_vector: "CVSS:2.0/..."  # Only v3.x supported
cvss_vector: "CVSS:3.1/AV:X"  # Invalid metric value
cvss_vector: "CVSS:3.1/AV:N"  # Missing required metrics
```

### CVSS Score Validation

```yaml
# ✅ Valid
cvss_score: 7.5
cvss_score: 0.0
cvss_score: 10.0

# ❌ Invalid
cvss_score: -0.1  # Below minimum
cvss_score: 10.1  # Above maximum
```

## How Priority Works

### Example Scenario 1: Multiple Methods Specified

```yaml
info:
  severity: high # Priority 4: Maps to 7.5
  cvss_score: 8.1 # Priority 3
  cvss_vector: "CVSS:3.1/..." # Priority 2: Calculates to 9.2
  risk_score: 9.8 # Priority 1 - WINS

# Final Risk Score: 9.8 (from risk_score)
```

### Example Scenario 2: CVSS Vector with Invalid Score

```yaml
info:
  severity: critical # Priority 4: Maps to 9.5
  cvss_score: 15.0 # Priority 3: INVALID (> 10.0)
  cvss_vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H"
# cvss_score is invalid, skipped
# cvss_vector is valid, calculates to 10.0
# Final Risk Score: 10.0 (from cvss_vector)
```

### Example Scenario 3: Severity Only

```yaml
info:
  severity: medium # Priority 4: Maps to 5.0

# No other scoring methods provided
# Final Risk Score: 5.0 (from severity mapping)
```

## Integration with Vulnerability Database

Risk scores calculated from templates are stored in the vulnerability database:

- **`risk_score` field**: The calculated numerical score (0.0-10.0)
- **`severity` field**: The qualitative severity string
- **`cvss_score` field**: Populated with the calculated risk score
- **`cvss_vector` field**: Stored if provided in template
- **`agent_metadata` field**: Contains detailed scoring information including the scoring method used

## Best Practices

### When to Use Each Method

1. **Custom Risk Score**: Use for:

   - Business-specific vulnerabilities
   - Internal risk assessments
   - Quick prototyping
   - Non-standard vulnerabilities

2. **CVSS Vector**: Use for:

   - Standard CVEs with full CVSS analysis
   - Publicly disclosed vulnerabilities
   - When you want detailed metrics documentation
   - Compliance requirements needing CVSS vectors

3. **CVSS Score**: Use for:

   - Importing from external sources (NVD, vendor advisories)
   - When you have the score but not the full vector
   - Third-party vulnerability data

4. **Severity Only**: Use for:
   - Simple templates
   - Low-priority findings
   - Information-gathering templates
   - Backward compatibility with existing templates

### Recommendations

- **Consistency**: Within a template repository, try to use the same scoring method for similar types of vulnerabilities
- **Documentation**: Always include `references` to document the source of your risk assessment
- **Testing**: Validate templates after creation to ensure risk scores calculate correctly
- **Updates**: If CVSS scores change (e.g., after further analysis), update templates accordingly

## Troubleshooting

### Template Validation Fails

**Error:** `info.risk_score 12.0 is invalid (must be between 0.0 and 10.0)`

**Solution:** Risk scores must be between 0.0 and 10.0. Adjust the value.

### CVSS Vector Parse Error

**Error:** `info.cvss_vector is invalid: missing required base metric: PR`

**Solution:** Ensure all required CVSS base metrics are included in the vector string.

### Unexpected Risk Score

**Problem:** Template shows different risk score than expected.

**Solution:** Check which scoring method is being used by examining the `agent_metadata` in scan results. The priority system may be using a different method than you expected.

## Future Enhancements

The following features are planned for future releases:

- **Scanner-Level Risk Calculation**: Custom risk calculation engines at the scanner level
- **Environmental CVSS Metrics**: Support for environmental scoring adjustments
- **Temporal CVSS Metrics**: Support for exploit maturity and remediation levels
- **Risk Score Overrides**: Per-host or per-scan risk score adjustments
- **Risk Trending**: Track risk score changes over time

## See Also

- [Template Creation Guide](README.md)
- [CVSS v3.1 Specification](https://www.first.org/cvss/v3.1/specification-document)
- [CVSS Calculator](https://www.first.org/cvss/calculator/3.1)
- [NVD Vulnerability Database](https://nvd.nist.gov/)











