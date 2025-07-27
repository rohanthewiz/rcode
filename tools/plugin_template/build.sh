#!/bin/bash
# Build script for RCode custom tool plugins

# Tool name (change this to match your tool)
TOOL_NAME="example_tool"

# Output directory for the compiled plugin
OUTPUT_DIR="$HOME/.rcode/tools"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Building RCode custom tool plugin: $TOOL_NAME"
echo "============================================"

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

# Check if we're in the right directory
if [ ! -f "$TOOL_NAME.go" ]; then
    echo -e "${RED}Error: $TOOL_NAME.go not found in current directory${NC}"
    exit 1
fi

# Build the plugin
echo -e "${YELLOW}Compiling plugin...${NC}"
go build -buildmode=plugin -o "$OUTPUT_DIR/$TOOL_NAME.so" "$TOOL_NAME.go"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Successfully built $TOOL_NAME plugin${NC}"
    echo -e "${GREEN}✓ Plugin location: $OUTPUT_DIR/$TOOL_NAME.so${NC}"
    echo ""
    echo "To use this plugin:"
    echo "1. Set the environment variable: export RCODE_CUSTOM_TOOLS_ENABLED=true"
    echo "2. Restart RCode"
    echo "3. The tool '$TOOL_NAME' will be available in your sessions"
else
    echo -e "${RED}✗ Failed to build plugin${NC}"
    echo "Common issues:"
    echo "- Make sure you have Go installed and in your PATH"
    echo "- Check that the plugin imports match RCode's structure"
    echo "- Ensure you're using compatible Go versions"
    exit 1
fi