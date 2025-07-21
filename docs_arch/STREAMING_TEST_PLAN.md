# RCode Streaming Implementation Test Plan

## Overview
This document outlines the test scenarios to verify that the real-time message streaming feature works correctly in RCode.

## What Was Implemented
1. **Backend streaming** is already enabled (web/session.go line 338: `request.Stream = true`)
2. **SSE events** for streaming are already implemented:
   - `BroadcastMessageStart` - signals start of streaming
   - `BroadcastMessageDelta` - sends text chunks
   - `BroadcastMessageStop` - signals end of streaming
3. **Frontend handlers** already exist for these events
4. **Fixed thinking indicator** - now removed by SSE events, not API response
5. **Enhanced streaming UI** - improved cursor animation and text appearance

## Test Scenarios

### 1. Basic Text Streaming
- Send a message asking for a simple explanation (e.g., "Explain how JavaScript promises work")
- **Expected behavior:**
  - "Thinking..." indicator appears briefly
  - Thinking indicator disappears when streaming starts
  - Text appears character by character with animated cursor
  - Cursor disappears when message completes

### 2. Streaming with Code Blocks
- Send: "Write a Python function to calculate fibonacci numbers"
- **Expected behavior:**
  - Code blocks render correctly during streaming
  - Syntax highlighting applies after code block completes
  - No layout jumps or text overflow

### 3. Streaming with Tool Usage
- Send: "Read the contents of main.go"
- **Expected behavior:**
  - Thinking indicator appears
  - Tool usage summary appears: "🛠️ TOOL USE - ✓ Read main.go (X lines)"
  - Thinking indicator is replaced by streaming response
  - Tool summaries remain visible above the response

### 4. Multiple Tool Uses
- Send: "List all Go files in the project and read the first one"
- **Expected behavior:**
  - Multiple tool summaries appear in sequence
  - Each tool summary shows before its execution
  - Streaming response appears after all tools complete

### 5. Long Response Streaming
- Send: "Explain the entire architecture of this RCode project in detail"
- **Expected behavior:**
  - Smooth scrolling as content streams
  - No performance issues with long content
  - Cursor remains visible at the end of text

### 6. Error Handling
- Disconnect network briefly during streaming
- **Expected behavior:**
  - SSE reconnects automatically
  - Partial message is preserved
  - Error message if reconnection fails

## How to Run Tests
1. Start the RCode server: `go run main.go`
2. Open http://localhost:8000 in your browser
3. Open browser developer console to see debug logs
4. Execute each test scenario above
5. Verify expected behaviors

## Debug Information
The browser console will show:
- "Message streaming started" - when streaming begins
- "Message delta received: [text]" - for each chunk
- "Message streaming stopped" - when complete
- SSE connection status messages

## Success Criteria
- ✅ No more "Thinking..." during message display
- ✅ Smooth, character-by-character text appearance  
- ✅ Tool summaries display correctly with streaming
- ✅ No UI glitches or layout issues
- ✅ Good performance with long responses