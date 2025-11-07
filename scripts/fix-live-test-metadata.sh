#!/bin/bash
#
# Fix live-test Template Metadata
# Creates proper metadata for the live-test custom template
#

set -e

echo "Fixing live-test metadata..."

# Get template content
TEMPLATE_CONTENT=$(docker exec sirius-valkey redis-cli GET "template:custom:live-test")

# Calculate checksum
CHECKSUM=$(echo "$TEMPLATE_CONTENT" | shasum -a 256 | awk '{print $1}')
SIZE=$(echo "$TEMPLATE_CONTENT" | wc -c | tr -d ' ')

# Get current timestamp
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%S.%NZ")

# Create metadata JSON
cat <<EOF | docker exec -i sirius-valkey redis-cli -x SET "template:meta:live-test"
{
  "id": "live-test",
  "version": "1.0",
  "checksum": "sha256:${CHECKSUM}",
  "size": ${SIZE},
  "severity": "critical",
  "platforms": ["darwin"],
  "detection_type": "file-hash",
  "author": "",
  "created": "${TIMESTAMP}",
  "updated": "${TIMESTAMP}",
  "vulnerability_ids": null,
  "is_custom": true
}
EOF

echo "✅ Metadata created for live-test"
echo ""
echo "Verification:"
docker exec sirius-valkey redis-cli GET "template:meta:live-test" | python3 -m json.tool
echo ""
echo "Next step: Trigger agent sync"
echo "  [agent] internal:template sync"









