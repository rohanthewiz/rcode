# Diff Visualization End-to-End Test Plan

## Test Environment Setup
1. Ensure RCode server is running with all components:
   - Backend diff services (DiffService, SnapshotManager)
   - SSE broadcasting enabled
   - Frontend with Monaco Editor loaded

## Test Scenarios

### Scenario 1: Basic File Modification Flow
1. **Setup**:
   - Start RCode server
   - Login and create a new session
   - Open File Explorer

2. **Test Steps**:
   ```bash
   # Create a test file
   echo "Hello World" > test_file.txt
   
   # Wait for file to appear in File Explorer
   # Modify the file using RCode tools
   # Check for diff_available SSE event
   ```

3. **Expected Results**:
   - File Explorer shows orange dot indicator next to modified file
   - System message appears: "📝 Changes detected in test_file.txt"
   - Right-click context menu shows "View Changes" option

### Scenario 2: Diff Viewer Functionality
1. **Test Steps**:
   - Right-click on modified file
   - Select "View Changes"
   - Test all view modes:
     - Monaco (default)
     - Side-by-Side
     - Inline
     - Unified

2. **Expected Results**:
   - Diff modal opens with correct filename
   - Addition/deletion counts display correctly
   - All view modes render properly
   - Synchronized scrolling works in Monaco and Side-by-Side views

### Scenario 3: Multiple File Changes
1. **Test Steps**:
   ```bash
   # Create multiple files
   echo "File 1 content" > file1.txt
   echo "File 2 content" > file2.txt
   echo "File 3 content" > file3.txt
   
   # Modify all files via RCode
   # Check File Explorer updates
   ```

2. **Expected Results**:
   - All modified files show orange indicators
   - Latest diff for each file is accessible
   - Switching between diffs works correctly

### Scenario 4: Apply/Revert Changes
1. **Test Steps**:
   - Open diff viewer for a modified file
   - Test "Apply Changes" button
   - Test "Revert" button
   - Test "Copy Diff" button

2. **Expected Results**:
   - Apply: Changes are saved, indicator removed
   - Revert: Original content restored, indicator removed
   - Copy: Diff copied to clipboard in unified format

### Scenario 5: SSE Reconnection
1. **Test Steps**:
   - Modify a file
   - Simulate network interruption
   - Wait for SSE reconnection
   - Modify another file

2. **Expected Results**:
   - Connection status indicator updates
   - New modifications still trigger diff_available events
   - File Explorer updates continue working

### Scenario 6: Theme and Options
1. **Test Steps**:
   - Open diff viewer
   - Toggle between Dark/Light themes
   - Enable/disable word wrap
   - Test with different file types (JS, Go, Python)

2. **Expected Results**:
   - Theme changes apply to Monaco editor
   - Word wrap toggles correctly
   - Syntax highlighting matches file type

### Scenario 7: Performance Test
1. **Test Steps**:
   - Create a large file (>1000 lines)
   - Make multiple changes throughout the file
   - Open diff viewer

2. **Expected Results**:
   - Diff loads within 2 seconds
   - Scrolling remains smooth
   - View mode switching is responsive

## Manual Test Script

```javascript
// Paste this in browser console to simulate file changes
async function testDiffVisualization() {
    console.log('Starting diff visualization tests...');
    
    // Test 1: Check if diff viewer is initialized
    if (!window.diffViewer) {
        console.error('❌ Diff viewer not initialized');
        return;
    }
    console.log('✅ Diff viewer initialized');
    
    // Test 2: Check File Explorer integration
    if (!window.FileExplorer) {
        console.error('❌ File Explorer not initialized');
        return;
    }
    console.log('✅ File Explorer initialized');
    
    // Test 3: Simulate diff_available event
    const testEvent = {
        type: 'diff_available',
        sessionId: window.currentSessionId,
        data: {
            diffId: 'test-diff-123',
            path: 'test/sample.js',
            stats: { additions: 5, deletions: 3 }
        }
    };
    
    // Trigger the event handler
    if (window.handleSSEMessage) {
        window.handleSSEMessage({ data: JSON.stringify(testEvent) });
        console.log('✅ SSE event triggered');
    }
    
    // Test 4: Check if file is marked as modified
    setTimeout(() => {
        if (window.FileExplorer.isFileModified('test/sample.js')) {
            console.log('✅ File marked as modified');
        } else {
            console.error('❌ File not marked as modified');
        }
    }, 100);
}

// Run the test
testDiffVisualization();
```

## Automated Test Using Bash

```bash
#!/bin/bash
# save as test_diff_flow.sh

echo "Testing RCode Diff Visualization Feature"

# 1. Create test directory
TEST_DIR="test_diff_viz_$(date +%s)"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# 2. Create initial files
echo "Creating test files..."
cat > main.go << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
EOF

cat > utils.js << 'EOF'
function calculateSum(a, b) {
    return a + b;
}

module.exports = { calculateSum };
EOF

echo "✅ Test files created"

# 3. Start monitoring (this would need to be done in RCode)
echo "Please ensure RCode is running and you have a session open"
echo "Press Enter when ready..."
read

# 4. Make modifications
echo "Making file modifications..."
cat > main.go << 'EOF'
package main

import (
    "fmt"
    "time"
)

func main() {
    fmt.Println("Hello, RCode!")
    fmt.Printf("Current time: %v\n", time.Now())
}
EOF

cat > utils.js << 'EOF'
function calculateSum(a, b) {
    // Add input validation
    if (typeof a !== 'number' || typeof b !== 'number') {
        throw new Error('Both arguments must be numbers');
    }
    return a + b;
}

function calculateProduct(a, b) {
    return a * b;
}

module.exports = { calculateSum, calculateProduct };
EOF

echo "✅ Files modified"

# 5. Create a new file
cat > README.md << 'EOF'
# Test Project

This is a test project for diff visualization.
EOF

echo "✅ New file created"

# 6. Summary
echo ""
echo "Test Summary:"
echo "- Created 2 files initially"
echo "- Modified both files"
echo "- Created 1 new file"
echo ""
echo "Expected in RCode:"
echo "1. main.go and utils.js should show orange indicators"
echo "2. System messages for each file change"
echo "3. Right-click -> View Changes should work"
echo "4. README.md should appear without indicator"

# Cleanup option
echo ""
echo "To cleanup test files, run: rm -rf ../$TEST_DIR"
```

## Integration Points to Verify

1. **Backend → SSE**:
   - DiffService creates snapshot
   - SSE broadcaster sends diff_available event
   - Event includes correct diffId and path

2. **SSE → Frontend**:
   - ui.js receives and processes event
   - System message displayed
   - Event forwarded to FileExplorer

3. **FileExplorer → DiffViewer**:
   - File marked as modified
   - Context menu shows "View Changes"
   - Click opens diff viewer with correct diffId

4. **DiffViewer → Backend**:
   - Fetch diff data via API
   - Apply/Revert operations work
   - Updates reflected in FileExplorer

## Success Criteria

- [ ] All modified files show visual indicators
- [ ] System messages appear for each modification
- [ ] Context menu integration works
- [ ] All diff view modes render correctly
- [ ] Synchronized scrolling works
- [ ] Apply/Revert operations complete successfully
- [ ] No console errors during operation
- [ ] Performance remains acceptable (< 2s load time)

## Known Issues to Watch For

1. **Monaco Loading**: Ensure Monaco is fully loaded before opening diffs
2. **SSE Reconnection**: Check connection status after network issues
3. **Large Files**: Monitor performance with files > 10,000 lines
4. **Concurrent Modifications**: Test rapid successive changes
5. **Session Switching**: Verify diff state when switching sessions