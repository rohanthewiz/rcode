package web

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
)

const sseStdMsgType = "message" // note that JS EventSource only pickup on "message" event type

// SSEEvent represents a server-sent event
type SSEEvent struct {
	Type      string      `json:"type"`
	SessionId string      `json:"sessionId,omitempty"`
	Data      interface{} `json:"data"`
}

// SSEHub manages SSE connections
type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan any]bool
}

// Global SSE hub
var sseHub = &SSEHub{
	clients: make(map[chan any]bool),
}

// Register adds a new SSE client
func (h *SSEHub) Register(client chan any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
}

// Unregister removes an SSE client
func (h *SSEHub) Unregister(client chan any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client)
	close(client)
}

// Broadcast sends an event to all connected clients
func (h *SSEHub) Broadcast(event SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	logger.F("Broadcasting SSE event: type=%s, sessionID=%s, nbrOfClients=%d", event.Type, event.SessionId, len(h.clients))

	// Prepare the payload
	data := map[string]interface{}{
		"type":      event.Type,
		"sessionId": event.SessionId,
		"data":      event.Data,
	}

	bytPayload, err := json.Marshal(data)
	if err != nil {
		logger.LogErr(err, "On broadcast, failed to marshal SSE event")
		return
	}

	rEvent := rweb.SSEvent{
		Type: sseStdMsgType, // Type fixed here bc that's what EventSource expects // event.Type,
		Data: string(bytPayload),
	}

	for client := range h.clients {
		select {
		case client <- rEvent:
		default:
			// Client's channel is full, skip
			logger.Log("warn", "SSE client channel full, skipping")
		}
	}
}

// BroadcastSessionUpdate broadcasts a session update event
func BroadcastSessionUpdate(sessionID string, updateType string, data interface{}) {
	event := SSEEvent{
		Type:      updateType,
		SessionId: sessionID,
		Data:      data,
	}
	sseHub.Broadcast(event)
}

// BroadcastMessage broadcasts a message event
func BroadcastMessage(sessionID string, message interface{}) {
	BroadcastSessionUpdate(sessionID, "message", message)
}

// BroadcastSessionList broadcasts when sessions are created/deleted
func BroadcastSessionList() {
	event := SSEEvent{
		Type: "session_list_updated",
		Data: nil,
	}
	sseHub.Broadcast(event)
}

// BroadcastToolUsage broadcasts a tool usage summary event
func BroadcastToolUsage(sessionID string, toolName string, summary string) {
	event := SSEEvent{
		Type:      "tool_usage",
		SessionId: sessionID,
		Data: map[string]interface{}{
			"tool":    toolName,
			"summary": summary,
		},
	}
	logger.Info("BroadcastToolUsage", "sessionID", sessionID, "tool", toolName, "summary", summary)
	sseHub.Broadcast(event)
}

// broadcastJSON broadcasts a generic JSON event
func broadcastJSON(eventType string, data interface{}) {
	event := SSEEvent{
		Type: eventType,
		Data: data,
	}
	sseHub.Broadcast(event)
}

// BroadcastFileEvent broadcasts file-related events
func BroadcastFileEvent(eventType string, data interface{}) {
	event := SSEEvent{
		Type: eventType,
		Data: data,
	}
	sseHub.Broadcast(event)
}

// BroadcastFileOpened broadcasts when a file is opened
func BroadcastFileOpened(sessionID string, filePath string) {
	BroadcastFileEvent("file_opened", map[string]interface{}{
		"sessionId": sessionID,
		"path":      filePath,
		"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
	})
}

// BroadcastFileChanged broadcasts when a file is modified
func BroadcastFileChanged(filePath string, changeType string) {
	BroadcastFileEvent("file_changed", map[string]interface{}{
		"path":       filePath,
		"changeType": changeType, // "created", "modified", "deleted", "renamed"
		"timestamp":  fmt.Sprintf("%d", time.Now().Unix()),
	})
}

// BroadcastFileTreeUpdate broadcasts when the file tree needs refresh
func BroadcastFileTreeUpdate(path string) {
	BroadcastFileEvent("file_tree_update", map[string]interface{}{
		"path":      path,
		"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
	})
}

// BroadcastDiffAvailable broadcasts when a new diff is available
func BroadcastDiffAvailable(sessionID string, diffID int64, filePath string, stats interface{}, toolName string) {
	event := SSEEvent{
		Type:      "diff_available",
		SessionId: sessionID,
		Data: map[string]interface{}{
			"diffId":   diffID,
			"filePath": filePath,
			"stats":    stats,
			"toolName": toolName,
		},
	}
	sseHub.Broadcast(event)
}

// BroadcastToolPermissionUpdate broadcasts when a tool permission is changed
func BroadcastToolPermissionUpdate(sessionID string, toolName string, enabled bool, mode string) {
	event := SSEEvent{
		Type:      "tool_permission_update",
		SessionId: sessionID,
		Data: map[string]interface{}{
			"toolName": toolName,
			"enabled":  enabled,
			"mode":     mode,
		},
	}
	sseHub.Broadcast(event)
}

// BroadcastMessageStart broadcasts when a message starts streaming
func BroadcastMessageStart(sessionID string) {
	event := SSEEvent{
		Type:      "message_start",
		SessionId: sessionID,
		Data:      nil,
	}
	sseHub.Broadcast(event)
}

// BroadcastMessageDelta broadcasts a chunk of streaming text
func BroadcastMessageDelta(sessionID string, delta string) {
	event := SSEEvent{
		Type:      "message_delta",
		SessionId: sessionID,
		Data: map[string]interface{}{
			"delta": delta,
		},
	}
	sseHub.Broadcast(event)
}

// BroadcastMessageStop broadcasts when a message finishes streaming
func BroadcastMessageStop(sessionID string) {
	event := SSEEvent{
		Type:      "message_stop",
		SessionId: sessionID,
		Data:      nil,
	}
	sseHub.Broadcast(event)
}

// BroadcastToolUseStart broadcasts when tool use is starting (removes thinking indicator)
func BroadcastToolUseStart(sessionID string) {
	event := SSEEvent{
		Type:      "tool_use_start",
		SessionId: sessionID,
		Data:      nil,
	}
	sseHub.Broadcast(event)
}

// BroadcastContentStart broadcasts when content starts (either text or tool)
func BroadcastContentStart(sessionID string) {
	event := SSEEvent{
		Type:      "content_start",
		SessionId: sessionID,
		Data:      nil,
	}
	sseHub.Broadcast(event)
}

// BroadcastToolExecutionStart broadcasts when a tool begins execution
func BroadcastToolExecutionStart(sessionID string, toolID string, toolName string) {
	event := SSEEvent{
		Type:      "tool_execution_start",
		SessionId: sessionID,
		Data: map[string]interface{}{
			"toolId":    toolID,
			"toolName":  toolName,
			"status":    "executing",
			"startTime": time.Now().Unix(),
		},
	}
	sseHub.Broadcast(event)
}

// BroadcastToolExecutionProgress broadcasts progress updates for long-running tools
func BroadcastToolExecutionProgress(sessionID string, toolID string, progress int, message string) {
	event := SSEEvent{
		Type:      "tool_execution_progress",
		SessionId: sessionID,
		Data: map[string]interface{}{
			"toolId":   toolID,
			"progress": progress,
			"message":  message,
		},
	}
	sseHub.Broadcast(event)
}

// BroadcastToolExecutionComplete broadcasts when a tool finishes execution
func BroadcastToolExecutionComplete(sessionID string, toolName string, toolID string, status string, summary string, durationMs int64, metrics map[string]interface{}) {
	event := SSEEvent{
		Type:      "tool_execution_complete",
		SessionId: sessionID,
		Data: map[string]interface{}{
			"toolName": toolName,
			"toolId":   toolID,
			"status":   status, // "success", "failed", "cancelled"
			"summary":  summary,
			"duration": durationMs,
			"metrics":  metrics,
		},
	}
	sseHub.Broadcast(event)
}

// BroadcastPermissionRequest broadcasts a tool permission request to the frontend
func BroadcastPermissionRequest(request *PermissionRequest) {
	// Format parameters for display
	paramDisplay := FormatParametersForDisplay(request.ToolName, request.Parameters)

	eventData := map[string]interface{}{
		"requestId":        request.ID,
		"toolName":         request.ToolName,
		"parameters":       request.Parameters,
		"parameterDisplay": paramDisplay,
		"timestamp":        request.Timestamp.Unix(),
	}

	// Include diff preview if available
	if request.DiffPreview != nil {
		eventData["diffPreview"] = request.DiffPreview
	}

	event := SSEEvent{
		Type:      "permission_request",
		SessionId: request.SessionID,
		Data:      eventData,
	}
	sseHub.Broadcast(event)
}

// BroadcastPermissionTimeout broadcasts when a permission request times out
func BroadcastPermissionTimeout(sessionID string, requestID string) {
	event := SSEEvent{
		Type:      "permission_timeout",
		SessionId: sessionID,
		Data: map[string]interface{}{
			"requestId": requestID,
		},
	}
	sseHub.Broadcast(event)
}
