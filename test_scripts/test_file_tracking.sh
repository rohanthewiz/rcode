#!/bin/bash

# Test file tracking functionality

echo "Testing File Tracking API..."

# Start server in background
echo "Starting server..."
go run main.go &
SERVER_PID=$!

# Wait for server to start
sleep 5

# Create a test session
echo -e "\n1. Creating test session:"
SESSION_RESPONSE=$(curl -s -X POST "http://localhost:8000/api/session" \
  -H "Content-Type: application/json" \
  -d '{"name": "File Tracking Test"}')

SESSION_ID=$(echo "$SESSION_RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
echo "Created session: $SESSION_ID"

# Open some files
echo -e "\n2. Opening files in session:"
curl -s -X POST "http://localhost:8000/api/session/$SESSION_ID/files/open" \
  -H "Content-Type: application/json" \
  -d '{"path": "main.go"}' | jq '.'

curl -s -X POST "http://localhost:8000/api/session/$SESSION_ID/files/open" \
  -H "Content-Type: application/json" \
  -d '{"path": "web/routes.go"}' | jq '.'

# Get open files
echo -e "\n3. Getting open files:"
curl -s "http://localhost:8000/api/session/$SESSION_ID/files/open" | jq '.'

# Get recent files
echo -e "\n4. Getting recent files:"
curl -s "http://localhost:8000/api/session/$SESSION_ID/files/recent" | jq '.'

# Close a file
echo -e "\n5. Closing a file:"
curl -s -X POST "http://localhost:8000/api/session/$SESSION_ID/files/close" \
  -H "Content-Type: application/json" \
  -d '{"path": "main.go"}' | jq '.'

# Get open files again
echo -e "\n6. Getting open files after close:"
curl -s "http://localhost:8000/api/session/$SESSION_ID/files/open" | jq '.'

# Clean up
echo -e "\n\nCleaning up..."
kill $SERVER_PID 2>/dev/null

echo -e "\nFile tracking test completed!"