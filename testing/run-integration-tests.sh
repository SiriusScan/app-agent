#!/bin/bash
set -e

echo "🧪 Running FileHash Module Integration Tests"
echo "=============================================="

# Build the test program
echo ""
echo "📦 Building integration test program..."
go build -o bin/test-integration cmd/test-integration/main.go
echo "✓ Built bin/test-integration"

# Test 1: Valid hash match
echo ""
echo "Test 1: Valid Hash Match"
echo "------------------------"
./bin/test-integration 2>&1 | tail -n 10

echo ""
echo "✅ All integration tests completed!"

