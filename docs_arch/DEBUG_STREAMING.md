# Debugging Streaming Issue

## Problem
Messages are not appearing in the UI. The API response shows empty content ("").

## Debug Steps Added

1. **Backend Logging** (web/session.go):
   - Line 352: Log all stream events with type and data presence
   - Line 407: Log parsed content deltas with type and text
   - Line 466: Log when stream completes with content length
   - Lines 577-595: Better error handling for empty responses

2. **Frontend Logging** (web/assets/js/ui.js):
   - Line 213-214: Log SSE event details and sessionId comparison
   - Line 670: Log session IDs when sending messages
   - Line 779: Ensure window.currentSessionId is set

## How to Debug

1. **Run the server** and watch the logs:
   ```bash
   go run main.go 2>&1 | grep -E "(Stream|stream|content|delta)"
   ```

2. **In the browser console**, look for:
   - Session creation logs
   - SSE event logs showing sessionId matching
   - Any JavaScript errors

3. **Expected server logs**:
   ```
   INFO Stream event received type=message_start hasMessage=true hasDelta=false
   INFO Stream event received type=content_block_start hasMessage=false hasDelta=false
   INFO Stream event received type=content_block_delta hasMessage=false hasDelta=true
   INFO Content delta parsed type=text_delta text="Here's the most..."
   INFO Stream event received type=content_block_stop hasMessage=false hasDelta=false
   INFO Stream event received type=message_stop hasMessage=false hasDelta=false
   INFO Stream complete contentLength=1234 toolUses=0
   ```

## Possible Issues

1. **Streaming not working**: If no "content_block_delta" events appear
2. **Delta parsing failing**: If deltas arrive but "Content delta parsed" doesn't show
3. **Session mismatch**: If frontend shows sessionId mismatch in console
4. **API changes**: If Anthropic changed their streaming format

## Next Steps

Based on the logs, we can:
1. Fix delta parsing if structure changed
2. Add more detailed error handling
3. Fall back to non-streaming mode if needed