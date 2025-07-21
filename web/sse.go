package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
)

// SSEEvent represents a server-sent event
type SSEEvent struct {
	Type      string      `json:"type"`
	SessionID string      `json:"sessionId,omitempty"`
	Data      interface{} `json:"data"`
}

// SSEHub manages SSE connections
type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan SSEEvent]bool
}

// Global SSE hub
var sseHub = &SSEHub{
	clients: make(map[chan SSEEvent]bool),
}

// Register adds a new SSE client
func (h *SSEHub) Register(client chan SSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
}

// Unregister removes an SSE client
func (h *SSEHub) Unregister(client chan SSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client)
	close(client)
}

// Broadcast sends an event to all connected clients
func (h *SSEHub) Broadcast(event SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	logger.F("Broadcasting SSE event: type=%s, sessionID=%s, clients=%d", event.Type, event.SessionID, len(h.clients))

	for client := range h.clients {
		select {
		case client <- event:
			// logger.Debug("Event sent to client")
		default:
			// Client's channel is full, skip
			logger.Log("warn", "SSE client channel full, skipping")
		}
	}
}

// Updated eventsHandler with proper SSE implementation
func eventsHandler(c rweb.Context) error {
	// Set SSE headers
	c.Response().SetHeader("Content-Type", "text/event-stream")
	c.Response().SetHeader("Cache-Control", "no-cache")
	c.Response().SetHeader("Connection", "keep-alive")
	c.Response().SetHeader("Access-Control-Allow-Origin", "*")

	// Create client channel
	clientChan := make(chan SSEEvent, 10)
	sseHub.Register(clientChan)

	// Ensure cleanup on disconnect
	defer func() {
		sseHub.Unregister(clientChan)
	}()

	// Send initial connection event
	_, _ = fmt.Fprintf(c.Response(), "event: connected\ndata: {}\n\n")
	if flusher, ok := c.Response().(http.Flusher); ok {
		flusher.Flush()
	}

	// Listen for events
	for {
		select {
		case event, ok := <-clientChan:
			if !ok {
				// Channel closed, client disconnected
				return nil
			}

			// Marshal event data
			data, err := json.Marshal(event)
			if err != nil {
				logger.LogErr(err, "failed to marshal SSE event")
				continue
			}

			// Send event
			_, _ = fmt.Fprintf(c.Response(), "data: %s\n\n", string(data))

			// Flush the response
			if flusher, ok := c.Response().(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// BroadcastSessionUpdate broadcasts a session update event
func BroadcastSessionUpdate(sessionID string, updateType string, data interface{}) {
	event := SSEEvent{
		Type:      updateType,
		SessionID: sessionID,
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
		SessionID: sessionID,
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
		SessionID: sessionID,
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
		SessionID: sessionID,
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
		SessionID: sessionID,
		Data:      nil,
	}
	sseHub.Broadcast(event)
}

// BroadcastMessageDelta broadcasts a chunk of streaming text
func BroadcastMessageDelta(sessionID string, delta string) {
	event := SSEEvent{
		Type:      "message_delta",
		SessionID: sessionID,
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
		SessionID: sessionID,
		Data:      nil,
	}
	sseHub.Broadcast(event)
}

// BroadcastToolUseStart broadcasts when tool use is starting (removes thinking indicator)
func BroadcastToolUseStart(sessionID string) {
	event := SSEEvent{
		Type:      "tool_use_start",
		SessionID: sessionID,
		Data:      nil,
	}
	sseHub.Broadcast(event)
}

// BroadcastContentStart broadcasts when content starts (either text or tool)
func BroadcastContentStart(sessionID string) {
	event := SSEEvent{
		Type:      "content_start",
		SessionID: sessionID,
		Data:      nil,
	}
	sseHub.Broadcast(event)
}

// BroadcastToolExecutionStart broadcasts when a tool begins execution
func BroadcastToolExecutionStart(sessionID string, toolID string, toolName string) {
	event := SSEEvent{
		Type:      "tool_execution_start",
		SessionID: sessionID,
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
		SessionID: sessionID,
		Data: map[string]interface{}{
			"toolId":   toolID,
			"progress": progress,
			"message":  message,
		},
	}
	sseHub.Broadcast(event)
}

// BroadcastToolExecutionComplete broadcasts when a tool finishes execution
func BroadcastToolExecutionComplete(sessionID string, toolID string, status string, summary string, durationMs int64, metrics map[string]interface{}) {
	event := SSEEvent{
		Type:      "tool_execution_complete",
		SessionID: sessionID,
		Data: map[string]interface{}{
			"toolId":   toolID,
			"status":   status, // "success", "failed", "cancelled"
			"summary":  summary,
			"duration": durationMs,
			"metrics":  metrics,
		},
	}
	sseHub.Broadcast(event)
}
