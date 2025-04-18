#!/bin/bash

# Function to check if command exists
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Check if tmux is installed
if ! command_exists tmux; then
  echo "tmux is not installed. Please install it or use another method to view logs."
  exit 1
fi

# Kill any existing tmux session with the same name
tmux kill-session -t agent_logs 2>/dev/null || true

# Create a new tmux session
tmux new-session -d -s agent_logs

# Split the window horizontally
tmux split-window -h -t agent_logs

# In the left pane, show agent logs
tmux send-keys -t agent_logs:0.0 "echo 'Agent Logs:'; tail -f $(ls -t *.log | head -1)" C-m

# In the right pane, show server logs from Docker
tmux send-keys -t agent_logs:0.1 "echo 'Server Logs:'; docker logs -f sirius-engine 2>&1 | grep -v 'go: downloading' | grep -v '^$'" C-m

# Attach to the session
tmux attach-session -t agent_logs

echo "Log tracking stopped." 