#!/bin/bash

# Test script for real-time tool summaries

echo "Testing Real-time Tool Summaries..."
echo "==================================="

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create a test session
echo -e "${BLUE}Creating test session...${NC}"
SESSION_RESPONSE=$(curl -s -X POST http://localhost:8000/api/session -H "Content-Type: application/json" -d '{"title": "Real-time Tool Test"}')
SESSION_ID=$(echo $SESSION_RESPONSE | jq -r '.id')
echo "Session ID: $SESSION_ID"

# Function to send a message and wait for response
send_message() {
    local content="$1"
    echo -e "\n${BLUE}Sending: $content${NC}"
    
    # Send the message
    curl -s -X POST "http://localhost:8000/api/session/$SESSION_ID/message" \
        -H "Content-Type: application/json" \
        -d "{\"content\": \"$content\"}" | jq .
    
    # Give it time to process
    sleep 3
}

# Test 1: Simple file operations
echo -e "\n${GREEN}Test 1: File Operations${NC}"
send_message "Create a new directory called 'test_realtime' and write a simple hello.txt file in it with the content 'Hello from real-time test!'"

# Test 2: Multiple tools in sequence
echo -e "\n${GREEN}Test 2: Multiple Tools${NC}"
send_message "List the contents of the test_realtime directory, then read the hello.txt file"

# Test 3: Search operation
echo -e "\n${GREEN}Test 3: Search Operation${NC}"
send_message "Search for files containing the word 'real-time' in the current directory"

# Test 4: Git operations
echo -e "\n${GREEN}Test 4: Git Operations${NC}"
send_message "Show the git status and the latest 3 commits"

# Test 5: Cleanup
echo -e "\n${GREEN}Test 5: Cleanup${NC}"
send_message "Remove the test_realtime directory"

echo -e "\n${GREEN}Tests completed!${NC}"
echo "Check the browser at http://localhost:8000 to see the real-time tool summaries in action."