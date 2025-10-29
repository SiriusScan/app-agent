#!/bin/bash
#
# Error Scenario Testing Script
# Tests various error conditions to verify graceful handling
#

set -e

echo "🧪 Testing Error Scenarios for Template System"
echo "=============================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counter for tests
PASS=0
FAIL=0

# Test helper
test_scenario() {
    local test_name="$1"
    local command="$2"
    local expected_pattern="$3"
    
    echo -e "${YELLOW}Testing: $test_name${NC}"
    
    if output=$($command 2>&1); then
        result_code=0
    else
        result_code=$?
    fi
    
    if echo "$output" | grep -q "$expected_pattern"; then
        echo -e "${GREEN}✅ PASS${NC} - Found expected error pattern: $expected_pattern"
        ((PASS++))
    else
        echo -e "${RED}❌ FAIL${NC} - Expected pattern not found: $expected_pattern"
        echo "Output: $output"
        ((FAIL++))
    fi
    echo ""
}

# Cleanup function
cleanup() {
    rm -f /tmp/test-*.yaml
    rm -rf /tmp/empty-templates
    rm -rf /tmp/test-templates
}

# Ensure cleanup on exit
trap cleanup EXIT

echo "📝 Setting up test environment..."
mkdir -p /tmp/test-templates

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 1: Invalid YAML Syntax"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cat > /tmp/test-invalid.yaml <<EOF
id: TEST-001
invalid: {
  unclosed: bracket
EOF

test_scenario "Invalid YAML" \
    "./sirius-agent template run --template /tmp/test-invalid.yaml" \
    "unmarshal"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 2: Missing Required Fields"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cat > /tmp/test-incomplete.yaml <<EOF
id: TEST-002
detection:
  steps: []
EOF

test_scenario "Missing required fields" \
    "./sirius-agent template run --template /tmp/test-incomplete.yaml" \
    "info.name is required"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 3: Non-existent Module Type"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cat > /tmp/test-bad-module.yaml <<EOF
id: TEST-003
info:
  name: Bad Module Test
  severity: high
detection:
  steps:
    - type: non-existent-module
      config: {}
EOF

test_scenario "Non-existent module" \
    "./sirius-agent template run --template /tmp/test-bad-module.yaml" \
    "not found in registry"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 4: Invalid Severity Level"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cat > /tmp/test-bad-severity.yaml <<EOF
id: TEST-004
info:
  name: Bad Severity Test
  severity: super-duper-high
detection:
  steps:
    - type: file-hash
      config:
        path: /etc/passwd
EOF

test_scenario "Invalid severity level" \
    "./sirius-agent template run --template /tmp/test-bad-severity.yaml" \
    "severity.*invalid"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 5: Invalid Detection Logic"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cat > /tmp/test-bad-logic.yaml <<EOF
id: TEST-005
info:
  name: Bad Logic Test
  severity: high
detection:
  logic: maybe
  steps:
    - type: file-hash
      config:
        path: /etc/passwd
EOF

test_scenario "Invalid detection logic" \
    "./sirius-agent template run --template /tmp/test-bad-logic.yaml" \
    "detection.logic.*invalid"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 6: No Detection Steps"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cat > /tmp/test-no-steps.yaml <<EOF
id: TEST-006
info:
  name: No Steps Test
  severity: high
detection:
  steps: []
EOF

test_scenario "No detection steps" \
    "./sirius-agent template run --template /tmp/test-no-steps.yaml" \
    "detection.steps must contain at least one step"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 7: Empty Directory"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
mkdir -p /tmp/empty-templates

test_scenario "Empty template directory" \
    "./sirius-agent template run-all --directory /tmp/empty-templates" \
    "no valid templates found"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 8: Non-existent Directory"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
test_scenario "Non-existent directory" \
    "./sirius-agent template run-all --directory /tmp/nonexistent-dir-12345" \
    "does not exist"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 9: Permission Denied"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "id: TEST-007" > /tmp/test-noperm.yaml
chmod 000 /tmp/test-noperm.yaml

test_scenario "Permission denied" \
    "./sirius-agent template run --template /tmp/test-noperm.yaml" \
    "permission denied"

# Restore permissions for cleanup
chmod 644 /tmp/test-noperm.yaml

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 10: Invalid Worker Count"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
test_scenario "Invalid worker count (too high)" \
    "./sirius-agent template run-all --workers 100" \
    "worker count must be between"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 11: Invalid Timeout Value"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
test_scenario "Invalid timeout value" \
    "./sirius-agent template run-all --timeout abc" \
    "invalid timeout value"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 12: Invalid Format"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
test_scenario "Invalid format" \
    "./sirius-agent template run-all --format xml" \
    "invalid format"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test 13: Conflicting Arguments"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
test_scenario "Conflicting --template and --directory" \
    "./sirius-agent template run-all --template /tmp/test.yaml --directory /tmp/test-templates" \
    "cannot specify both"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  📊 Test Results Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
TOTAL=$((PASS + FAIL))
echo "Total Tests: $TOTAL"
echo -e "${GREEN}Passed: $PASS${NC}"
if [ $FAIL -gt 0 ]; then
    echo -e "${RED}Failed: $FAIL${NC}"
else
    echo "Failed: $FAIL"
fi
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✅ All error scenarios handled correctly!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed. Review error handling.${NC}"
    exit 1
fi

