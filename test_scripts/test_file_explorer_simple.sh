#!/bin/bash

# Start server in background
echo "Starting server..."
go run main.go &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Test file explorer API endpoints
echo -e "\nTesting File Explorer API..."

# Test 1: Get file tree
echo -e "\n1. Getting file tree:"
curl -v "http://localhost:8000/api/files/tree?depth=1"

# Clean up
echo -e "\n\nStopping server..."
kill $SERVER_PID

echo -e "\nTest completed!"