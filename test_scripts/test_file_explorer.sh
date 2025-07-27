#!/bin/bash

# Test file explorer API endpoints

echo "Testing File Explorer API..."

# Test 1: Get file tree
echo -e "\n1. Getting file tree (depth=2):"
curl -s "http://localhost:8000/api/files/tree?depth=2" | jq '.'

# Test 2: Get specific path
echo -e "\n2. Getting web directory:"
curl -s "http://localhost:8000/api/files/tree?path=web&depth=1" | jq '.'

# Test 3: Search for files
echo -e "\n3. Searching for 'ui' files:"
curl -s -X POST "http://localhost:8000/api/files/search" \
  -H "Content-Type: application/json" \
  -d '{"query": "ui", "searchContent": false}' | jq '.'

# Test 4: Get file content
echo -e "\n4. Getting README.md content:"
curl -s "http://localhost:8000/api/files/content/README.md" | jq '.name, .size'

echo -e "\nAll tests completed!"