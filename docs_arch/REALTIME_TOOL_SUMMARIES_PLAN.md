# Real-time Tool Call Summaries Plan

## Overview
Enhance RCode to display tool call summaries in real-time as they execute, showing them immediately after the "Thinking..." indicator disappears. These summaries will remain visible in the chat history as a permanent record of what tools were used.

## Current State
- Tool summaries are displayed AFTER execution completes
- Format: "🛠️ TOOL USE" header with individual tool summaries like "✓ Created directory reverse"
- Summaries appear between the user message and assistant response
- "Thinking..." indicator is removed when content starts streaming

## Proposed Enhancement

### Visual Flow
1. User sends message
2. "Thinking..." indicator appears
3. When tool execution starts:
   - "Thinking..." changes to "🛠️ Executing tools..."
   - Tool summaries appear in real-time as each tool starts/completes
   - Each tool shows status: pending → executing → complete/failed
4. Tool summaries remain visible in chat history
5. Assistant's response streams below the tool summaries

### UI Design

```
You: Create a new Go file that adds two numbers

Assistant
🛠️ Executing tools...

┌─ Tool Activity ─────────────────────────────────┐
│ ⏳ make_dir: Creating directory "calculator"... │
│ ✓ make_dir: Created directory "calculator"     │
│ ⏳ write_file: Writing main.go...               │
│ ✓ write_file: Wrote main.go (523 bytes)        │
└─────────────────────────────────────────────────┘

[Assistant response streams here...]
```

### Real-time Status Indicators
- ⏳ Pending/Executing (animated spinner)
- ✓ Success (green)
- ❌ Failed (red)
- ⚠️ Warning (yellow)
- 🔄 Retrying (for transient failures)

## Implementation Plan

### Phase 1: Backend Infrastructure
1. **Enhanced SSE Events**
   - Add `tool_execution_start` event when tool begins
   - Add `tool_execution_progress` for long-running tools
   - Add `tool_execution_complete` with result summary
   - Include execution metrics (start time, duration, retry count)

2. **Tool Execution Tracking**
   - Track tool state: pending, executing, complete, failed, retrying
   - Include queue position for multiple tools
   - Add progress reporting for tools that support it

### Phase 2: Frontend Display
1. **Tool Activity Container**
   - Replace "Thinking..." with "🛠️ Executing tools..." header
   - Create expandable/collapsible tool activity box
   - Show tool queue and execution order
   - Display real-time status updates

2. **Animation and Transitions**
   - Smooth transitions between states
   - Progress bars for long-running operations
   - Fade-in animation for new tool entries
   - Success/failure color transitions

3. **Persistent Display**
   - Tool summaries remain in chat history
   - Collapsible by default after completion
   - Click to expand and see detailed execution log
   - Timestamp for each tool execution

### Phase 3: Enhanced Features
1. **Tool Progress Reporting**
   - File operations: bytes written/read progress
   - Search operations: files scanned counter
   - Git operations: commits processed
   - Web operations: download progress

2. **Tool Execution Insights**
   - Show tool dependencies (e.g., mkdir before write_file)
   - Display parallel vs sequential execution
   - Highlight failed tools that block execution
   - Show retry attempts and reasons

3. **Interactive Features**
   - Click on tool to see detailed input/output
   - Copy tool commands for manual execution
   - Re-run failed tools with modified parameters
   - Cancel long-running operations

## Technical Implementation

### Backend Changes

#### 1. Modify `web/session.go`
```go
// Broadcast when tool execution starts
BroadcastToolExecutionStart(sessionID, map[string]interface{}{
    "toolId": toolUse.ID,
    "toolName": toolUse.Name,
    "status": "executing",
    "startTime": time.Now().Unix(),
})

// During execution, send progress updates
BroadcastToolExecutionProgress(sessionID, map[string]interface{}{
    "toolId": toolUse.ID,
    "progress": 45,
    "message": "Processed 45 of 100 files",
})

// After completion
BroadcastToolExecutionComplete(sessionID, map[string]interface{}{
    "toolId": toolUse.ID,
    "status": "success",
    "summary": summary,
    "duration": durationMs,
    "metrics": executionMetrics,
})
```

#### 2. Add to `web/sse.go`
```go
func BroadcastToolExecutionStart(sessionID string, data interface{}) {
    event := SSEEvent{
        Type:      "tool_execution_start",
        SessionID: sessionID,
        Data:      data,
    }
    sseHub.Broadcast(event)
}
```

### Frontend Changes

#### 1. Update `web/assets/js/ui.js`
```javascript
// Track active tool executions
const activeTools = new Map();

// Handle tool execution events
function handleToolExecutionStart(event) {
    // Remove thinking indicator
    removeThinkingIndicator();
    
    // Show tool execution container
    showToolExecutionContainer();
    
    // Add tool to active list
    activeTools.set(event.data.toolId, {
        name: event.data.toolName,
        status: 'executing',
        startTime: event.data.startTime
    });
    
    // Update UI
    updateToolExecutionDisplay();
}

// Create real-time tool display
function createToolExecutionDisplay() {
    const container = document.createElement('div');
    container.className = 'tool-execution-container';
    container.innerHTML = `
        <div class="tool-execution-header">
            <span class="tool-icon">🛠️</span>
            <span class="tool-title">Executing tools...</span>
            <button class="tool-toggle" onclick="toggleToolDetails()">▼</button>
        </div>
        <div class="tool-execution-list">
            <!-- Tool items added here dynamically -->
        </div>
    `;
    return container;
}
```

#### 2. Add to `web/assets/css/ui.css`
```css
.tool-execution-container {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 8px;
    margin: 1rem 0;
    overflow: hidden;
    transition: all 0.3s ease;
}

.tool-execution-header {
    padding: 0.75rem 1rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
    user-select: none;
}

.tool-execution-list {
    padding: 0 1rem 1rem;
    max-height: 300px;
    overflow-y: auto;
}

.tool-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem;
    margin: 0.25rem 0;
    background: var(--bg-primary);
    border-radius: 4px;
    transition: all 0.2s ease;
}

.tool-item.executing {
    border-left: 3px solid var(--primary);
    animation: pulse 1.5s ease-in-out infinite;
}

.tool-item.success {
    border-left: 3px solid var(--success);
}

.tool-item.failed {
    border-left: 3px solid var(--error);
}

.tool-status-icon {
    font-size: 1.2rem;
    animation: spin 1s linear infinite;
}

.tool-item.executing .tool-status-icon {
    animation: spin 1s linear infinite;
}

@keyframes spin {
    to { transform: rotate(360deg); }
}

@keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.8; }
}

.tool-progress {
    flex: 1;
    height: 4px;
    background: var(--bg-tertiary);
    border-radius: 2px;
    overflow: hidden;
    margin: 0 0.5rem;
}

.tool-progress-bar {
    height: 100%;
    background: var(--primary);
    transition: width 0.3s ease;
}

.tool-metrics {
    font-size: 0.85rem;
    color: var(--text-secondary);
    margin-left: auto;
}
```

## Benefits

1. **Better User Experience**
   - Immediate feedback on what's happening
   - No anxiety during long operations
   - Clear indication of progress and status

2. **Transparency**
   - Users see exactly what tools are being used
   - Execution order and dependencies are visible
   - Failed operations are clearly marked

3. **Debugging**
   - Persistent record of tool executions
   - Timing information for performance analysis
   - Clear error messages for failures

4. **Professional Feel**
   - Matches modern CI/CD pipeline interfaces
   - Similar to GitHub Actions or GitLab CI visualization
   - Polished, real-time feedback

## Future Enhancements

1. **Tool Execution History**
   - View all tool executions for a session
   - Filter by tool type, status, or time
   - Export execution logs

2. **Tool Analytics**
   - Most used tools
   - Average execution times
   - Success/failure rates
   - Performance trends

3. **Advanced Features**
   - Tool execution templates
   - Batch operations
   - Conditional execution
   - Tool chaining visualization

## Success Metrics

- Tool status updates appear within 100ms of execution start
- All tool executions are captured and displayed
- No UI freezing during long operations
- Clear visual distinction between states
- Smooth animations and transitions
- Tool summaries remain accessible in chat history