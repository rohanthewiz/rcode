package web

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// PermissionRequest represents a pending tool permission request
type PermissionRequest struct {
	ID         string                 `json:"id"`
	SessionID  string                 `json:"sessionId"`
	ToolName   string                 `json:"toolName"`
	Parameters map[string]interface{} `json:"parameters"`
	Timestamp  time.Time              `json:"timestamp"`
	ResponseCh chan PermissionResponse
}

// PermissionResponse represents a user's response to a permission request
type PermissionResponse struct {
	RequestID      string `json:"requestId"`
	Approved       bool   `json:"approved"`
	RememberChoice bool   `json:"rememberChoice"`
	Error          error  `json:"-"`
}

// PermissionManager manages pending permission requests
type PermissionManager struct {
	mu       sync.RWMutex
	requests map[string]*PermissionRequest
	timeout  time.Duration
}

// Global permission manager instance
var permissionManager = NewPermissionManager(30 * time.Second)

// NewPermissionManager creates a new permission manager
func NewPermissionManager(timeout time.Duration) *PermissionManager {
	pm := &PermissionManager{
		requests: make(map[string]*PermissionRequest),
		timeout:  timeout,
	}

	// Start cleanup goroutine
	go pm.cleanupExpiredRequests()

	return pm
}

// CreateRequest creates a new permission request and returns its ID
func (pm *PermissionManager) CreateRequest(sessionID, toolName string, parameters map[string]interface{}) (*PermissionRequest, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	request := &PermissionRequest{
		ID:         uuid.New().String(),
		SessionID:  sessionID,
		ToolName:   toolName,
		Parameters: parameters,
		Timestamp:  time.Now(),
		ResponseCh: make(chan PermissionResponse, 1),
	}

	pm.requests[request.ID] = request

	logger.Info("Created permission request",
		"id", request.ID,
		"session", sessionID,
		"tool", toolName)

	return request, nil
}

// WaitForResponse waits for a response to the given request with timeout
func (pm *PermissionManager) WaitForResponse(requestID string) (PermissionResponse, error) {
	pm.mu.RLock()
	request, exists := pm.requests[requestID]
	pm.mu.RUnlock()

	if !exists {
		return PermissionResponse{}, serr.New("request not found")
	}

	// Wait for response or timeout
	select {
	case response := <-request.ResponseCh:
		// Clean up the request
		pm.removeRequest(requestID)
		return response, response.Error

	case <-time.After(pm.timeout):
		// Timeout occurred
		pm.removeRequest(requestID)
		return PermissionResponse{}, serr.New("permission request timed out")
	}
}

// HandleResponse processes a permission response from the frontend
func (pm *PermissionManager) HandleResponse(response PermissionResponse) error {
	pm.mu.RLock()
	request, exists := pm.requests[response.RequestID]
	pm.mu.RUnlock()

	if !exists {
		return serr.New("request not found or already processed")
	}

	// Send response through channel
	select {
	case request.ResponseCh <- response:
		logger.Info("Permission response processed",
			"id", response.RequestID,
			"approved", response.Approved,
			"remember", response.RememberChoice)
		return nil
	default:
		// Channel already has a response (shouldn't happen)
		return serr.New("request already has a response")
	}
}

// GetRequest retrieves a request by ID (for validation)
func (pm *PermissionManager) GetRequest(requestID string) (*PermissionRequest, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	request, exists := pm.requests[requestID]
	return request, exists
}

// removeRequest removes a request from the manager
func (pm *PermissionManager) removeRequest(requestID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if request, exists := pm.requests[requestID]; exists {
		close(request.ResponseCh)
		delete(pm.requests, requestID)
	}
}

// cleanupExpiredRequests periodically removes expired requests
func (pm *PermissionManager) cleanupExpiredRequests() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pm.mu.Lock()
		now := time.Now()

		for id, request := range pm.requests {
			if now.Sub(request.Timestamp) > pm.timeout {
				logger.Info("Cleaning up expired permission request", "id", id)
				close(request.ResponseCh)
				delete(pm.requests, id)
			}
		}

		pm.mu.Unlock()
	}
}

// GetPendingRequests returns all pending requests for a session
func (pm *PermissionManager) GetPendingRequests(sessionID string) []*PermissionRequest {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var pending []*PermissionRequest
	for _, request := range pm.requests {
		if request.SessionID == sessionID {
			pending = append(pending, request)
		}
	}

	return pending
}

// CancelSessionRequests cancels all pending requests for a session
func (pm *PermissionManager) CancelSessionRequests(sessionID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for id, request := range pm.requests {
		if request.SessionID == sessionID {
			// Send cancellation response
			select {
			case request.ResponseCh <- PermissionResponse{
				RequestID: id,
				Approved:  false,
				Error:     serr.New("session cancelled all pending requests"),
			}:
			default:
			}

			close(request.ResponseCh)
			delete(pm.requests, id)
		}
	}
}

// FormatParametersForDisplay formats tool parameters for user-friendly display
func FormatParametersForDisplay(toolName string, params map[string]interface{}) string {
	switch toolName {
	case "write_file", "edit_file":
		if path, ok := params["path"].(string); ok {
			return fmt.Sprintf("File: %s", path)
		}
	case "bash":
		if cmd, ok := params["command"].(string); ok {
			// Truncate long commands
			if len(cmd) > 100 {
				cmd = cmd[:97] + "..."
			}
			return fmt.Sprintf("Command: %s", cmd)
		}
	case "remove":
		if path, ok := params["path"].(string); ok {
			return fmt.Sprintf("Delete: %s", path)
		}
	case "make_dir":
		if path, ok := params["path"].(string); ok {
			return fmt.Sprintf("Create directory: %s", path)
		}
	}

	// Default: show first few parameters
	var parts []string
	count := 0
	for k, v := range params {
		if count >= 3 {
			parts = append(parts, "...")
			break
		}
		parts = append(parts, fmt.Sprintf("%s: %v", k, v))
		count++
	}

	return fmt.Sprintf("Parameters: %v", parts)
}
