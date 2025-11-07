#!/bin/bash
#
# Simple Template Metadata Migration
# Adds is_custom field to existing metadata
#

echo "========================================="
echo "Template Metadata Migration (Simple)"
echo "========================================="
echo ""

# Fix DOCKER-001 (standard)
echo "Fixing DOCKER-001 (standard)..."
docker exec sirius-valkey redis-cli GET "template:meta:DOCKER-001" | \
  python3 -c "import sys, json; d=json.load(sys.stdin); d['is_custom']=False; print(json.dumps(d))" | \
  docker exec -i sirius-valkey redis-cli -x SET "template:meta:DOCKER-001"
echo "✅ DOCKER-001 updated"

# Fix NGINX-001 (standard)
echo "Fixing NGINX-001 (standard)..."
docker exec sirius-valkey redis-cli GET "template:meta:NGINX-001" | \
  python3 -c "import sys, json; d=json.load(sys.stdin); d['is_custom']=False; print(json.dumps(d))" | \
  docker exec -i sirius-valkey redis-cli -x SET "template:meta:NGINX-001"
echo "✅ NGINX-001 updated"

# Fix APACHE-001 (standard)
echo "Fixing APACHE-001 (standard)..."
docker exec sirius-valkey redis-cli GET "template:meta:APACHE-001" | \
  python3 -c "import sys, json; d=json.load(sys.stdin); d['is_custom']=False; print(json.dumps(d))" | \
  docker exec -i sirius-valkey redis-cli -x SET "template:meta:APACHE-001"
echo "✅ APACHE-001 updated"

# Fix test (custom)
echo "Fixing test (custom)..."
docker exec sirius-valkey redis-cli GET "template:meta:test" 2>/dev/null | \
  python3 -c "import sys, json; d=json.load(sys.stdin); d['is_custom']=True; print(json.dumps(d))" | \
  docker exec -i sirius-valkey redis-cli -x SET "template:meta:test" || echo "⚠️  No metadata for test"

# Fix live-test (custom)
echo "Fixing live-test (custom)..."
docker exec sirius-valkey redis-cli GET "template:meta:live-test" 2>/dev/null | \
  python3 -c "import sys, json; d=json.load(sys.stdin); d['is_custom']=True; print(json.dumps(d))" | \
  docker exec -i sirius-valkey redis-cli -x SET "template:meta:live-test" || echo "⚠️  No metadata for live-test"

echo ""
echo "========================================="
echo "Migration Complete!"
echo "========================================="
echo ""
echo "Now restart sirius-engine:"
echo "  docker restart sirius-engine"









