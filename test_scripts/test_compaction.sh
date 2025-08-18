#!/bin/bash

# Test script for conversation compaction feature

echo "Testing Conversation Compaction Feature"
echo "======================================="

# Get first session ID
SESSION_ID=$(curl -s http://localhost:8000/api/session | jq -r '.[0].id')
echo "Using session: $SESSION_ID"

# Get compaction stats
echo -e "\n1. Getting compaction stats..."
curl -s http://localhost:8000/api/session/$SESSION_ID/compaction/stats | jq '.'

# Get message count
echo -e "\n2. Getting message count..."
MSG_COUNT=$(curl -s http://localhost:8000/api/session/$SESSION_ID/messages | jq '. | length')
echo "Current message count: $MSG_COUNT"

# Try to compact (may fail if not enough messages)
echo -e "\n3. Attempting to compact conversation..."
RESPONSE=$(curl -s -X POST http://localhost:8000/api/session/$SESSION_ID/compact \
  -H "Content-Type: application/json" \
  -d '{
    "preserve_recent": 5,
    "preserve_initial": 2,
    "strategy": "conservative",
    "min_messages_to_compact": 10
  }')

echo "Compaction response:"
echo "$RESPONSE" | jq '.'

# Check if there are any compacted messages
echo -e "\n4. Checking for compacted messages..."
curl -s http://localhost:8000/api/session/$SESSION_ID/compaction/messages | jq '.'

echo -e "\nTest complete!"