# Usage Tracking and Display Implementation Plan

## Analysis Summary

The rcode codebase already has excellent foundations for usage tracking:

### Current Infrastructure:
- **API Response Parsing**: `providers.Usage` struct captures `InputTokens` and `OutputTokens` from Anthropic API
- **Database Storage**: `token_usage` JSON field in `messages` table already stores usage data per message
- **Existing Methods**: `GetSessionStats()` and `GetMessagesWithMetadata()` can extract token data
- **SSE System**: Real-time event broadcasting system ready for usage updates
- **Frontend Structure**: Message rendering system and tool summary display patterns established

### Missing Pieces:
- Frontend display of usage information
- Real-time usage updates via SSE
- API endpoints for usage statistics
- UI components for usage visualization

## Implementation Plan

### Phase 1: Backend API Endpoints (30 min)
1. **Add Usage Statistics Endpoints**:
   - `GET /api/session/:id/usage` - Get detailed usage stats for a session
   - `GET /api/usage/global` - Get global usage statistics across all sessions
   - Extend existing session stats endpoint with token breakdowns

2. **Enhance SSE Events**:
   - Add `usage_update` event type to broadcast real-time token consumption
   - Modify `sendMessageHandler` to broadcast usage after each API response

### Phase 2: Frontend Display Components (45 min)
1. **Session Usage Panel**:
   - Add collapsible usage panel in chat interface header
   - Display: current session tokens, estimated cost, model-specific stats
   - Real-time updates via SSE events

2. **Message-Level Usage Display**:
   - Add small usage badge next to assistant messages showing tokens used
   - Format: "📊 1.2K in / 850 out" style display
   - Only show for messages with usage data

3. **Global Usage Dashboard**:
   - Add usage statistics tab to sidebar (alongside sessions/files)
   - Show: daily usage, model breakdowns, cost estimates, usage trends
   - Implement usage history visualization

### Phase 3: Enhanced Features (15 min)
1. **Usage Alerts**:
   - Configurable usage thresholds with visual warnings
   - Session-level and daily usage tracking

2. **Cost Estimation**:
   - Add Anthropic pricing data for accurate cost calculations
   - Display estimated costs alongside token counts

## File Changes Required

### Backend Files:
- `web/routes.go` - Add new usage endpoints
- `web/session.go` - Add usage broadcast in message handler  
- `web/usage_handlers.go` - New file for usage API handlers
- `web/sse.go` - Add usage event broadcast functions
- `db/message.go` - Enhance existing usage query methods

### Frontend Files:
- `web/assets/js/ui.js` - Add usage display logic and SSE handlers
- `web/assets/css/ui.css` - Style usage components
- `web/ui.go` - Add usage panel to main UI structure

## Technical Approach

### Real-time Updates:
- Broadcast usage via SSE immediately after API response
- Update session totals and message-specific displays
- Use existing SSE infrastructure for consistency

### Data Flow:
1. API response contains usage → Store in message record
2. Broadcast usage_update SSE event → Frontend updates displays
3. Usage endpoints provide aggregated statistics on demand
4. Frontend polls usage stats for dashboard updates

### UI Integration:
- Follow existing patterns (tool summaries, message structure)
- Use CSS variables for consistent theming
- Implement progressive enhancement (graceful degradation)

This plan leverages the existing robust infrastructure while adding comprehensive usage tracking and visualization capabilities.