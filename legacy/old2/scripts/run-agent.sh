#!/bin/bash

# Get the Docker container's IP address
DOCKER_IP=$(docker inspect sirius-engine -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' 2>/dev/null)
if [ -z "$DOCKER_IP" ]; then
  DOCKER_IP="localhost"
  echo "Warning: Could not get Docker container IP, using localhost instead."
fi

# Default values
AGENT_ID="test-agent-1"
SERVER_ADDR="$DOCKER_IP:50051"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --id=*)
      AGENT_ID="${1#*=}"
      shift
      ;;
    --server=*)
      SERVER_ADDR="${1#*=}"
      shift
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [--id=agent-id] [--server=addr:port]"
      exit 1
      ;;
  esac
done

# Set environment variables
export AGENT_ID="$AGENT_ID"

# Kill any already running agents
pkill -f "go run cmd/agent/main.go" 2>/dev/null || true

# Print configuration
echo "Starting agent with configuration:"
echo "  Agent ID: $AGENT_ID"
echo "  Server: $SERVER_ADDR"

# Run the agent with debug logging
SIRIUS_LOG_LEVEL=debug go run cmd/agent/main.go --server="$SERVER_ADDR" 