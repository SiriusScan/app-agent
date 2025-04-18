#!/bin/bash

# Kill any existing server and agent processes
echo "Stopping any existing processes..."
pkill -f "server-app" || true
pkill -f "agent-app" || true

# Set port for the server
export PORT=50051

# Start the server in the background
echo "Starting server on port $PORT..."
./server-app &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Start the agent connected to the server
echo "Starting agent connected to localhost:$PORT..."
./agent-app --server=localhost:$PORT

# Cleanup when agent exits
echo "Agent exited, shutting down server..."
kill $SERVER_PID 