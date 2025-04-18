#!/bin/bash

echo "==== Debugging gRPC Connections ===="

# Check if the agent is running
echo "Checking for running agent processes:"
ps aux | grep "cmd/agent/main.go" | grep -v grep

# Check current network connections on port 50051
echo "Checking for connections on port 50051:"
netstat -an | grep 50051

# Check if anything is listening on port 50051
echo "Checking for listeners on port 50051:"
lsof -i :50051 2>/dev/null || echo "No processes listening on port 50051"

# Docker network inspection
echo "Checking Docker network:"
docker inspect sirius-engine -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' 2>/dev/null || echo "Couldn't get Docker IP"

# Check server logs
echo "Checking server logs:"
docker exec -it sirius-engine sh -c 'tail -n 20 /app-agent/logs/server.log' 2>/dev/null || echo "No server logs found"

# Run a test connection
echo "Testing gRPC connection:"
nc -zv localhost 50051 2>&1 || echo "Failed to connect to gRPC server"

echo "==== Debug Complete ====" 