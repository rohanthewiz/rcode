#!/bin/bash
# Build script for word_count plugin example

TOOL_NAME="word_count"
OUTPUT_DIR="$HOME/.rcode/tools"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Building word_count plugin example"
echo "================================="

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

# Build the plugin
echo -e "${YELLOW}Compiling plugin...${NC}"
go build -buildmode=plugin -o "$OUTPUT_DIR/${TOOL_NAME}_tool.so" word_count_tool.go

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Successfully built word_count plugin${NC}"
    echo -e "${GREEN}✓ Plugin location: $OUTPUT_DIR/${TOOL_NAME}_tool.so${NC}"
    echo ""
    echo "To test this plugin:"
    echo "1. export RCODE_CUSTOM_TOOLS_ENABLED=true"
    echo "2. Restart RCode"
    echo "3. Use the tool: word_count with path or pattern parameter"
else
    echo "Failed to build plugin"
    exit 1
fi