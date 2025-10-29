#!/bin/bash
#
# Template Sync Diagnostic Script
# This script checks the template synchronization system to identify integration issues
#

set -e

echo "========================================="
echo "Template Sync Diagnostic Report"
echo "========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. Check Repository Configuration
echo "1. Checking Repository Configuration..."
echo "----------------------------------------"
REPO_LIST=$(docker exec sirius-valkey redis-cli GET "sirius:agent-templates:repositories" 2>/dev/null || echo "{}")
if [ "$REPO_LIST" = "{}" ] || [ -z "$REPO_LIST" ]; then
    echo -e "${RED}❌ No repositories configured in Valkey${NC}"
else
    echo -e "${GREEN}✅ Repository configuration found${NC}"
    echo "$REPO_LIST" | python3 -m json.tool 2>/dev/null || echo "$REPO_LIST"
fi
echo ""

# 2. Check Standard Templates
echo "2. Checking Standard Templates in Valkey..."
echo "----------------------------------------"
STANDARD_TEMPLATES=$(docker exec sirius-valkey redis-cli KEYS "template:standard:*" | wc -l)
echo "Standard template keys found: $STANDARD_TEMPLATES"
if [ "$STANDARD_TEMPLATES" -gt 0 ]; then
    echo -e "${GREEN}✅ Standard templates exist${NC}"
    echo "First 10 template IDs:"
    docker exec sirius-valkey redis-cli KEYS "template:standard:*" | head -10 | sed 's/template:standard:/  - /'
else
    echo -e "${RED}❌ No standard templates found${NC}"
fi
echo ""

# 3. Check Template Metadata
echo "3. Checking Template Metadata..."
echo "----------------------------------------"
META_TEMPLATES=$(docker exec sirius-valkey redis-cli KEYS "template:meta:*" | wc -l)
echo "Template metadata keys found: $META_TEMPLATES"
if [ "$META_TEMPLATES" -gt 0 ]; then
    echo -e "${GREEN}✅ Template metadata exists${NC}"
    echo "First 10 metadata IDs:"
    docker exec sirius-valkey redis-cli KEYS "template:meta:*" | head -10 | sed 's/template:meta:/  - /'
else
    echo -e "${RED}❌ No template metadata found${NC}"
fi
echo ""

# 4. Check Template Manifest
echo "4. Checking Template Manifest..."
echo "----------------------------------------"
MANIFEST=$(docker exec sirius-valkey redis-cli GET "template:manifest" 2>/dev/null || echo "")
if [ -z "$MANIFEST" ]; then
    echo -e "${RED}❌ No template manifest found${NC}"
else
    echo -e "${GREEN}✅ Template manifest exists${NC}"
    echo "$MANIFEST" | python3 -m json.tool 2>/dev/null | head -30 || echo "$MANIFEST"
fi
echo ""

# 5. Check Custom Templates
echo "5. Checking Custom Templates..."
echo "----------------------------------------"
CUSTOM_TEMPLATES=$(docker exec sirius-valkey redis-cli KEYS "template:custom:*" | wc -l)
echo "Custom template keys found: $CUSTOM_TEMPLATES"
if [ "$CUSTOM_TEMPLATES" -gt 0 ]; then
    echo -e "${GREEN}✅ Custom templates exist${NC}"
else
    echo -e "${YELLOW}⚠️  No custom templates (this is normal)${NC}"
fi
echo ""

# 6. Check Server Logs
echo "6. Checking Server Sync Logs..."
echo "----------------------------------------"
SYNC_LOGS=$(docker logs sirius-engine 2>&1 | grep -i "repository sync" | tail -20)
if [ -z "$SYNC_LOGS" ]; then
    echo -e "${YELLOW}⚠️  No repository sync logs found${NC}"
else
    echo -e "${GREEN}✅ Sync logs found${NC}"
    echo "$SYNC_LOGS"
fi
echo ""

# 7. Check RabbitMQ Queue
echo "7. Checking RabbitMQ Sync Queue..."
echo "----------------------------------------"
QUEUE_STATUS=$(docker exec sirius-rabbitmq rabbitmqctl list_queues 2>/dev/null | grep "agent.template.sync" || echo "")
if [ -z "$QUEUE_STATUS" ]; then
    echo -e "${YELLOW}⚠️  Sync queue not found or empty${NC}"
else
    echo -e "${GREEN}✅ Sync queue exists${NC}"
    echo "$QUEUE_STATUS"
fi
echo ""

# 8. Sample Template Data
echo "8. Sample Template Data..."
echo "----------------------------------------"
SAMPLE_TEMPLATE=$(docker exec sirius-valkey redis-cli KEYS "template:standard:*" | head -1)
if [ -n "$SAMPLE_TEMPLATE" ]; then
    echo "Sample template: $SAMPLE_TEMPLATE"
    docker exec sirius-valkey redis-cli GET "$SAMPLE_TEMPLATE" | python3 -m json.tool 2>/dev/null | head -30 || echo "Could not parse template"
else
    echo -e "${RED}❌ No templates to sample${NC}"
fi
echo ""

# Summary
echo "========================================="
echo "Diagnostic Summary"
echo "========================================="
echo "Repository Config: $([ "$REPO_LIST" != "{}" ] && echo -e "${GREEN}✅${NC}" || echo -e "${RED}❌${NC}")"
echo "Standard Templates: $STANDARD_TEMPLATES keys"
echo "Template Metadata: $META_TEMPLATES keys"
echo "Template Manifest: $([ -n "$MANIFEST" ] && echo -e "${GREEN}✅${NC}" || echo -e "${RED}❌${NC}")"
echo "Custom Templates: $CUSTOM_TEMPLATES keys"
echo ""

# Recommendations
echo "Recommendations:"
echo "----------------------------------------"
if [ "$STANDARD_TEMPLATES" -eq 0 ]; then
    echo -e "${RED}🔧 Issue: No standard templates found${NC}"
    echo "   Solution: Run repository sync manually"
    echo "   Command: curl -X POST http://localhost:9001/api/agent-templates/repositories/default-sirius-official/sync"
fi

if [ "$META_TEMPLATES" -eq 0 ]; then
    echo -e "${RED}🔧 Issue: No template metadata found${NC}"
    echo "   This means templates aren't being stored properly"
    echo "   Check server logs for errors during sync"
fi

if [ "$STANDARD_TEMPLATES" -gt 0 ] && [ "$META_TEMPLATES" -eq 0 ]; then
    echo -e "${RED}🔧 Critical: Templates exist but metadata is missing${NC}"
    echo "   This indicates a storage layer problem"
    echo "   ValKeyTemplateStorage.StoreTemplate() may not be writing metadata"
fi

if [ "$STANDARD_TEMPLATES" -ne "$META_TEMPLATES" ]; then
    echo -e "${YELLOW}⚠️  Warning: Mismatch between standard ($STANDARD_TEMPLATES) and metadata ($META_TEMPLATES) counts${NC}"
    echo "   Some templates may be incomplete"
fi

echo ""
echo "========================================="
echo "End of Diagnostic Report"
echo "========================================="







