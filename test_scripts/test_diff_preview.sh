#!/bin/bash

# Test script for diff preview in permissions
# This script helps test the diff preview functionality

echo "🧪 Testing Diff Preview in Permission Dialogs"
echo "==========================================="
echo ""
echo "Prerequisites:"
echo "1. Start the RCode server: go run main.go"
echo "2. Login and create a session"
echo "3. Set write_file and edit_file tools to 'Ask' mode in the Tools tab"
echo ""
echo "Test Cases:"
echo ""
echo "1. Test write_file with existing file:"
echo "   - Send: 'Please update the README.md file to add a new section about testing'"
echo "   - Expected: Permission dialog shows diff preview with additions"
echo ""
echo "2. Test write_file with new file:"
echo "   - Send: 'Create a new file test_new.go with a simple hello world program'"
echo "   - Expected: Permission dialog shows diff preview (all additions)"
echo ""
echo "3. Test edit_file:"
echo "   - Send: 'Edit main.go to add a comment at line 10'"
echo "   - Expected: Permission dialog shows diff preview with the edit"
echo ""
echo "4. Test complex edit_file:"
echo "   - Send: 'Replace the first function in main.go with an improved version'"
echo "   - Expected: Permission dialog shows diff preview with deletions and additions"
echo ""
echo "Visual Verification:"
echo "✓ Diff preview section appears for file modification tools"
echo "✓ 'View Changes' button is clickable and shows stats"
echo "✓ Diff content displays with proper syntax highlighting"
echo "✓ Added lines show in green"
echo "✓ Deleted lines show in red"
echo "✓ Context lines show in gray"
echo "✓ Diff can be expanded/collapsed"
echo ""
echo "Browser Console Commands:"
echo "You can manually trigger permission requests in the browser console:"
echo ""
echo "// Simulate a write_file permission request"
echo 'handlePermissionRequest({'
echo '  requestId: "test-123",'
echo '  toolName: "write_file",'
echo '  parameters: { path: "test.txt", content: "new content" },'
echo '  diffPreview: {'
echo '    stats: { added: 5, deleted: 2 },'
echo '    hunks: [{'
echo '      oldStart: 1, oldLines: 3,'
echo '      newStart: 1, newLines: 4,'
echo '      lines: ['
echo '        { type: "context", content: "line 1" },'
echo '        { type: "delete", content: "old line 2" },'
echo '        { type: "add", content: "new line 2" },'
echo '        { type: "add", content: "extra line" },'
echo '        { type: "context", content: "line 3" }'
echo '      ]'
echo '    }]'
echo '  }'
echo '})'