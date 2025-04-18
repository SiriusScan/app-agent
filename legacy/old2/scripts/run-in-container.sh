#!/bin/bash

# Run the server and agent inside the Docker container

# Check if container is running
if ! docker ps | grep -q "sirius-engine"; then
  echo "❌ sirius-engine container is not running!"
  echo "Please start it first with: docker start sirius-engine"
  exit 1
fi

# Show help
show_help() {
  echo "Usage: $0 [options]"
  echo "Options:"
  echo "  -s, --server-only    Run only the server"
  echo "  -a, --agent-only     Run only the agent"
  echo "  -p, --port PORT      Specify the port (default: 50051)"
  echo "  -h, --help           Show this help message"
  echo ""
  echo "Without options, both server and agent will be run."
}

# Default values
RUN_SERVER=true
RUN_AGENT=true
PORT=50051

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    -s|--server-only)
      RUN_SERVER=true
      RUN_AGENT=false
      shift
      ;;
    -a|--agent-only)
      RUN_SERVER=false
      RUN_AGENT=true
      shift
      ;;
    -p|--port)
      PORT="$2"
      shift 2
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      show_help
      exit 1
      ;;
  esac
done

# Run the server
if $RUN_SERVER; then
  echo "🚀 Starting server on port $PORT..."
  docker exec -d sirius-engine sh -c "cd /app-agent && go run cmd/app-agent/main.go"
  echo "✅ Server started in background"
  
  # Wait for server to start
  sleep 2
fi

# Run the agent
if $RUN_AGENT; then
  echo "🚀 Starting agent connected to localhost:$PORT..."
  docker exec -it sirius-engine sh -c "cd /app-agent && go run cmd/agent/main.go --server=localhost:$PORT"
  echo "✅ Agent finished"
fi

echo "👋 Done" 