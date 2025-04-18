#!/bin/bash

# Check results of a specific task ID
if [ -n "$1" ]; then
  echo "Checking results for task ID: $1"
  go run cmd/result-retriever/main.go --command="$1" --format=table
else
  # Check all results for test-agent-1
  echo "Checking all results for test-agent-1"
  go run cmd/result-retriever/main.go --agent="test-agent-1" --format=table
fi 