#!/bin/bash

# Create a new prompt
echo "Creating a new prompt..."
CREATE_RESPONSE=$(curl -s -X POST http://localhost:8000/api/prompts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Prompt",
    "description": "A test prompt to verify updates work",
    "content": "This is the initial content",
    "includes_permissions": false,
    "is_active": true,
    "is_default": false
  }')

echo "Create response: $CREATE_RESPONSE"

# Extract the ID from the response
PROMPT_ID=$(echo "$CREATE_RESPONSE" | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
echo "Created prompt with ID: $PROMPT_ID"

# Update the prompt
echo -e "\nUpdating the prompt..."
UPDATE_RESPONSE=$(curl -s -X PUT "http://localhost:8000/api/prompts/$PROMPT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Test Prompt",
    "description": "This description has been updated",
    "content": "This is the UPDATED content",
    "includes_permissions": true,
    "is_active": true,
    "is_default": true
  }')

echo "Update response: $UPDATE_RESPONSE"

# Verify the update
echo -e "\nVerifying the update..."
GET_RESPONSE=$(curl -s -X GET "http://localhost:8000/api/prompts/$PROMPT_ID")
echo "Get response: $GET_RESPONSE"

# Clean up - delete the test prompt
echo -e "\nCleaning up..."
DELETE_RESPONSE=$(curl -s -X DELETE "http://localhost:8000/api/prompts/$PROMPT_ID")
echo "Delete response: $DELETE_RESPONSE"