# Manual Reconnection Test Steps - FIXED

## Issues Fixed
1. **Manual reconnection now properly closes existing EventSource**
2. **Attempt counter shows correctly (1/5 instead of 0/5)**
3. **404 errors handled gracefully when sessions are lost**
4. **Sessions refresh automatically after reconnection**

## Test Procedure

1. **Open browser**: http://localhost:8000
2. **Open Developer Console** (F12)
3. **Login if needed**

## Test 1: Basic Manual Reconnection (Backend Running)
1. In console, type: `disconnectSSE()`
2. Verify status shows "Disconnected"
3. Click "Reconnect" link
4. **Expected**: 
   - Console shows "Manual SSE reconnection requested"
   - Console shows "Closing existing EventSource before reconnecting" (if applicable)
   - Status immediately shows "Reconnecting... (1/5)"
   - Connection re-establishes
   - Status indicator disappears

## Test 2: Manual Reconnection (Backend Down)
1. Stop the backend (Ctrl+C)
2. Wait for auto-reconnection to exhaust (5 attempts)
3. Click "Reconnect" link
4. **Expected**:
   - Console shows "Manual SSE reconnection requested"
   - Status shows "Reconnecting... (1/5)"
   - Attempts continue with exponential backoff
   - After 5 attempts, shows "Connection lost. Reconnect" again

## Test 3: Manual Reconnection (Backend Restored)
1. From Test 2 state (backend down, max attempts reached)
2. Start backend: `go run main.go`
3. Click "Reconnect" link
4. **Expected**:
   - Immediate connection success
   - Status indicator disappears
   - Console shows "SSE connection established"

## Console Commands for Testing
```javascript
// Force disconnect
disconnectSSE()

// Check current state
console.log({
  eventSource,
  reconnectAttempts,
  connectionStatus,
  isManuallyDisconnected
})

// Manual reconnect
reconnectSSE()
```