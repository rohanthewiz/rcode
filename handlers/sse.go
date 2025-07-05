package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

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
	fmt.Fprintf(c.Response(), "event: connected\ndata: {}\n\n")
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
			fmt.Fprintf(c.Response(), "data: %s\n\n", string(data))

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
