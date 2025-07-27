# Auto-Switch to Files Tab on First LLM Response

## Overview
This document outlines the implementation plan for automatically switching to the Files tab when the first response from the LLM is received. This enhancement improves user experience by immediately showing file changes and activity when the assistant starts working.

## Current Architecture

### Tab System
- **Location**: The tab UI is rendered server-side in `/Users/ro/projs/go/rcode/web/file_tree_component.go`
- **Component**: `FileExplorerTabs` struct with `ActiveTab` field (can be "sessions", "files", or "tools")
- **JavaScript**: Tab switching is handled by `switchTab()` function in `/Users/ro/projs/go/rcode/web/assets/js/fileExplorer.js`

### Event System
- **SSE Events**: Server sends events via Server-Sent Events (SSE)
- **Event Handler**: `handleServerEvent()` in `ui.js` processes all incoming events
- **Key Events**:
  - `message_start`: Indicates message streaming has started
  - `content_start`: Indicates actual content (text or tool) has started
  - `tool_execution_start`: Indicates a tool is beginning execution
  - `message_delta`: Contains streaming text chunks

### Module Structure
- **FileExplorer**: IIFE module exposed as `window.FileExplorer`
- **Current API**: The module currently exports several functions but NOT `switchTab`
- **Initialization**: FileExplorer is initialized in `ui.go` after DOM is ready

## Implementation Plan

### Step 1: Export switchTab Function
In `/Users/ro/projs/go/rcode/web/assets/js/fileExplorer.js`, modify the return statement (around line 777):

```javascript
// Public API
return {
    init,
    loadFileTree,
    openFile,
    getOpenFiles: () => openFiles,
    getActiveFile: () => activeFile,
    refreshTree: () => renderFileTree(),
    handleFileEvent,
    refreshPath,
    markFileModified,
    unmarkFileModified,
    isFileModified,
    switchTab  // ADD THIS LINE
};
```

### Step 2: Add State Tracking
In `/Users/ro/projs/go/rcode/web/assets/js/ui.js`, add at the top with other state variables:

```javascript
let hasReceivedFirstResponse = false;  // Track first response per message
```

### Step 3: Implement Auto-Switch Logic
In `handleServerEvent()` function in `ui.js`, add logic to detect first response:

```javascript
// Inside handleServerEvent function
if (event.sessionId === currentSessionId) {
    // Check for first content/response events
    if (!hasReceivedFirstResponse && 
        (event.type === 'content_start' || 
         event.type === 'tool_execution_start' ||
         (event.type === 'message_delta' && event.data && event.data.delta))) {
        
        // Switch to Files tab on first response
        if (window.FileExplorer && window.FileExplorer.switchTab) {
            window.FileExplorer.switchTab('files');
            hasReceivedFirstResponse = true;
        }
    }
}
```

### Step 4: Reset Flag on New Messages
In `sendMessage()` function, reset the flag before sending:

```javascript
async function sendMessage() {
    // ... existing code ...
    
    // Reset first response flag for new message
    hasReceivedFirstResponse = false;
    
    // ... rest of the function ...
}
```

## Testing Scenarios

1. **Basic Test**:
   - Send a message to the LLM
   - Verify Files tab activates when response starts
   - Verify tab doesn't switch on subsequent responses

2. **Tool Usage Test**:
   - Send a message that triggers tool use
   - Verify Files tab activates when tool execution starts

3. **Session Switching**:
   - Switch between sessions
   - Verify flag resets properly for each session

4. **Manual Override**:
   - Manually switch to another tab after auto-switch
   - Verify system respects user's manual selection

## Additional Considerations

1. **User Preference**: Could add a setting to disable auto-switch behavior
2. **Animation**: Could add smooth transition animation when switching tabs
3. **Notification**: Could show a subtle indicator that tab was auto-switched
4. **Smart Detection**: Could only switch if files are actually being modified

## Related Files
- `/Users/ro/projs/go/rcode/web/assets/js/fileExplorer.js` - Contains switchTab function
- `/Users/ro/projs/go/rcode/web/assets/js/ui.js` - Main UI logic and event handling
- `/Users/ro/projs/go/rcode/web/file_tree_component.go` - Server-side tab rendering
- `/Users/ro/projs/go/rcode/web/ui.go` - Main UI template and JS initialization