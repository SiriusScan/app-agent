#!/bin/bash
#
# Fix Template Metadata Migration
# This script adds the missing 'is_custom' field to existing template metadata in Valkey
#

set -e

echo "========================================="
echo "Template Metadata Migration"
echo "========================================="
echo ""

# Get all metadata keys
META_KEYS=$(docker exec sirius-valkey redis-cli KEYS "template:meta:*")

if [ -z "$META_KEYS" ]; then
    echo "No metadata keys found. Nothing to migrate."
    exit 0
fi

echo "Found template metadata keys to migrate:"
echo "$META_KEYS"
echo ""

# For each metadata key, check if corresponding custom template exists
for META_KEY in $META_KEYS; do
    # Extract template ID
    TEMPLATE_ID=$(echo "$META_KEY" | sed 's/template:meta://')
    
    echo "Processing: $TEMPLATE_ID"
    
    # Check if custom template exists
    CUSTOM_EXISTS=$(docker exec sirius-valkey redis-cli EXISTS "template:custom:$TEMPLATE_ID")
    STANDARD_EXISTS=$(docker exec sirius-valkey redis-cli EXISTS "template:standard:$TEMPLATE_ID")
    
    # Get current metadata
    METADATA=$(docker exec sirius-valkey redis-cli GET "$META_KEY")
    
    # Determine if it's custom
    if [ "$CUSTOM_EXISTS" = "1" ]; then
        echo "  → Custom template detected"
        IS_CUSTOM="true"
    elif [ "$STANDARD_EXISTS" = "1" ]; then
        echo "  → Standard template detected"
        IS_CUSTOM="false"
    else
        echo "  ⚠️  Warning: No content found for $TEMPLATE_ID, skipping"
        continue
    fi
    
    # Add is_custom field to metadata using Python
    UPDATED_METADATA=$(echo "$METADATA" | python3 -c "
import sys
import json
try:
    data = json.load(sys.stdin)
    data['is_custom'] = $IS_CUSTOM
    print(json.dumps(data))
except Exception as e:
    print('ERROR', file=sys.stderr)
    sys.exit(1)
" 2>&1)
    
    if echo "$UPDATED_METADATA" | grep -q "ERROR"; then
        echo "  ❌ Failed to parse metadata for $TEMPLATE_ID"
        continue
    fi
    
    # Update metadata in Valkey
    echo "$UPDATED_METADATA" | docker exec -i sirius-valkey redis-cli -x SET "$META_KEY" > /dev/null
    
    echo "  ✅ Updated metadata with is_custom=$IS_CUSTOM"
done

echo ""
echo "========================================="
echo "Migration Complete"
echo "========================================="
echo ""
echo "Restart the sirius-engine container to apply changes:"
echo "  docker restart sirius-engine"
echo ""
echo "Then trigger a template sync on the agent:"
echo "  [agent] internal:template sync"







