#!/bin/bash
#
# Template Sync Fix Verification Script
# Run this to verify the bug fix is working
#

set -e

echo "========================================="
echo "Template Sync Fix Verification"
echo "========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0

# Test 1: Verify metadata has is_custom field
echo "Test 1: Checking metadata for is_custom field..."
echo "----------------------------------------------"
DOCKER_META=$(docker exec sirius-valkey redis-cli GET "template:meta:DOCKER-001")
if echo "$DOCKER_META" | grep -q '"is_custom"'; then
    if echo "$DOCKER_META" | grep -q '"is_custom":false' || echo "$DOCKER_META" | grep -q '"is_custom": false'; then
        echo -e "${GREEN}✅ PASS${NC} - DOCKER-001 has is_custom=false"
        ((PASS++))
    else
        echo -e "${RED}❌ FAIL${NC} - DOCKER-001 has wrong is_custom value"
        ((FAIL++))
    fi
else
    echo -e "${RED}❌ FAIL${NC} - DOCKER-001 missing is_custom field"
    ((FAIL++))
fi

TEST_META=$(docker exec sirius-valkey redis-cli GET "template:meta:test" 2>/dev/null || echo "")
if [ -n "$TEST_META" ]; then
    if echo "$TEST_META" | grep -q '"is_custom"'; then
        if echo "$TEST_META" | grep -q '"is_custom":true' || echo "$TEST_META" | grep -q '"is_custom": true'; then
            echo -e "${GREEN}✅ PASS${NC} - test has is_custom=true"
            ((PASS++))
        else
            echo -e "${RED}❌ FAIL${NC} - test has wrong is_custom value"
            ((FAIL++))
        fi
    else
        echo -e "${RED}❌ FAIL${NC} - test missing is_custom field"
        ((FAIL++))
    fi
else
    echo -e "${YELLOW}⚠️  SKIP${NC} - test template has no metadata"
fi
echo ""

# Test 2: Verify template counts
echo "Test 2: Verifying template counts in Valkey..."
echo "----------------------------------------------"
STANDARD_COUNT=$(docker exec sirius-valkey redis-cli KEYS "template:standard:*" | wc -l)
CUSTOM_COUNT=$(docker exec sirius-valkey redis-cli KEYS "template:custom:*" | wc -l)

echo "Standard templates: $STANDARD_COUNT"
echo "Custom templates: $CUSTOM_COUNT"

if [ "$STANDARD_COUNT" -ge 3 ]; then
    echo -e "${GREEN}✅ PASS${NC} - Standard templates exist ($STANDARD_COUNT)"
    ((PASS++))
else
    echo -e "${RED}❌ FAIL${NC} - Not enough standard templates ($STANDARD_COUNT)"
    ((FAIL++))
fi

if [ "$CUSTOM_COUNT" -ge 1 ]; then
    echo -e "${GREEN}✅ PASS${NC} - Custom templates exist ($CUSTOM_COUNT)"
    ((PASS++))
else
    echo -e "${YELLOW}⚠️  WARN${NC} - No custom templates found"
fi
echo ""

# Test 3: Verify server is running
echo "Test 3: Checking server status..."
echo "----------------------------------------------"
if docker ps | grep -q sirius-engine; then
    echo -e "${GREEN}✅ PASS${NC} - sirius-engine container running"
    ((PASS++))
else
    echo -e "${RED}❌ FAIL${NC} - sirius-engine container not running"
    ((FAIL++))
fi

if docker logs sirius-engine 2>&1 | tail -50 | grep -q "Server listening"; then
    echo -e "${GREEN}✅ PASS${NC} - Server is listening"
    ((PASS++))
else
    echo -e "${RED}❌ FAIL${NC} - Server not responding"
    ((FAIL++))
fi
echo ""

# Test 4: Check agent cache directories
echo "Test 4: Checking agent cache structure..."
echo "----------------------------------------------"
AGENT_CACHE_DIR="$HOME/Library/Application Support/sirius-agent/templates"
if [ -d "$AGENT_CACHE_DIR" ]; then
    echo -e "${GREEN}✅ PASS${NC} - Agent cache directory exists"
    ((PASS++))
    
    if [ -d "$AGENT_CACHE_DIR/server" ]; then
        echo -e "${GREEN}✅ PASS${NC} - Server templates directory exists"
        ((PASS++))
    else
        echo -e "${RED}❌ FAIL${NC} - Server templates directory missing"
        ((FAIL++))
    fi
    
    if [ -d "$AGENT_CACHE_DIR/custom" ]; then
        echo -e "${GREEN}✅ PASS${NC} - Custom templates directory exists"
        ((PASS++))
    else
        echo -e "${RED}❌ FAIL${NC} - Custom templates directory missing"
        ((FAIL++))
    fi
else
    echo -e "${YELLOW}⚠️  INFO${NC} - Agent cache directory not found (agent hasn't synced yet)"
fi
echo ""

# Summary
echo "========================================="
echo "Verification Summary"
echo "========================================="
echo -e "${GREEN}PASSED: $PASS${NC}"
echo -e "${RED}FAILED: $FAIL${NC}"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✅ ALL CHECKS PASSED!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Run agent: ./sirius-agent connect"
    echo "2. Trigger sync: internal:template sync"
    echo "3. Verify agent receives templates"
    echo "4. Run scan: scan --all"
else
    echo -e "${RED}❌ SOME CHECKS FAILED${NC}"
    echo ""
    echo "Review the failures above and:"
    echo "1. Check if migration script ran successfully"
    echo "2. Verify server restarted after code changes"
    echo "3. Check docker logs for errors"
fi

echo ""
echo "========================================="
exit $FAIL







