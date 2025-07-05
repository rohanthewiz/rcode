#!/bin/bash

# Script to run OpenCode server and web UI together

echo "Starting OpenCode server and web UI..."

# Function to kill background processes on exit
cleanup() {
    echo "Stopping services..."
    kill $SERVER_PID $UI_PID 2>/dev/null
    exit
}

trap cleanup EXIT INT TERM

# Start OpenCode server in background
echo "Starting OpenCode server on port 4096..."
(cd ../opencode && bun run dev) &
SERVER_PID=$!

# Wait a bit for server to start
sleep 2

# Start web UI
echo "Starting web UI on port 3000..."
bun run dev &
UI_PID=$!

echo "Services started!"
echo "- OpenCode server: http://localhost:4096"
echo "- Web UI: http://localhost:3000"
echo ""
echo "Press Ctrl+C to stop both services"

# Wait for background processes
wait