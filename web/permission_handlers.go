package web

import (
	"encoding/json"

	"rcode/db"
	"rcode/providers"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

// handlePermissionResponseHandler handles permission responses from the frontend
func handlePermissionResponseHandler(c rweb.Context) error {
	// Parse request body
	body := c.Request().Body()
	var response PermissionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}

	// Validate the request exists
	request, exists := permissionManager.GetRequest(response.RequestID)
	if !exists {
		return c.WriteError(serr.New("permission request not found or expired"), 404)
	}

	// Validate that the session making the response owns the request
	// This prevents cross-session attacks where one session could approve
	// permission requests from another session
	if response.SessionID == "" {
		return c.WriteError(serr.New("session ID is required"), 400)
	}

	// Verify the session ID matches the request's session ID
	// This ensures only the session that triggered the permission request
	// can approve or deny it, pr
	if response.SessionID != request.SessionID {
		logger.Warn("Session mismatch in permission response",
			"responseSessionID", response.SessionID,
			"requestSessionID", request.SessionID,
			"requestID", response.RequestID)
		return c.WriteError(serr.New("unauthorized: session does not own this permission request"), 403)
	}

	logger.Info("Received permission response",
		"requestId", response.RequestID,
		"approved", response.Approved,
		"remember", response.RememberChoice)

	// If user chose to remember, update the database
	if response.RememberChoice {
		database, err := db.GetDB()
		if err != nil {
			logger.LogErr(err, "failed to get database")
		} else {
			// Determine permission type based on approval
			permType := db.PermissionDenied
			if response.Approved {
				permType = db.PermissionAllowed
			}

			// Update the tool permission (no expiration for remembered choices)
			err = database.SetToolPermission(request.SessionID, request.ToolName, permType, nil, 0)
			if err != nil {
				logger.LogErr(err, "failed to update tool permission")
			} else {
				logger.Info("Updated tool permission based on remember choice",
					"tool", request.ToolName,
					"permission", permType)

				// Broadcast the permission update
				BroadcastToolPermissionUpdate(request.SessionID, request.ToolName, response.Approved, string(permType))
			}
		}
	}

	// Handle the response
	if err := permissionManager.HandleResponse(response); err != nil {
		return c.WriteError(err, 400)
	}

	return c.WriteJSON(map[string]interface{}{
		"success": true,
		"message": "Permission response processed",
	})
}

// PermissionAbortRequest represents an abort request from the frontend
type PermissionAbortRequest struct {
	SessionID string `json:"session_id"`
	RequestID string `json:"request_id,omitempty"` // Optional, abort specific request
}

// handlePermissionAbortHandler sends an abort message to the LLM for the current session
// This allows the user to interrupt the current operation and tell the LLM to stop
func handlePermissionAbortHandler(c rweb.Context) error {
	// Parse request body
	body := c.Request().Body()
	var abortReq PermissionAbortRequest
	if err := json.Unmarshal(body, &abortReq); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}

	// Validate session ID is provided
	if abortReq.SessionID == "" {
		return c.WriteError(serr.New("session ID is required"), 400)
	}

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Verify the session exists
	session, err := database.GetSession(abortReq.SessionID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get session"), 500)
	}
	if session == nil {
		return c.WriteError(serr.New("session not found"), 404)
	}

	// If a specific request ID was provided, cancel that permission request
	if abortReq.RequestID != "" {
		request, exists := permissionManager.GetRequest(abortReq.RequestID)
		if exists && request.SessionID == abortReq.SessionID {
			// Create a denial response for the permission request
			response := PermissionResponse{
				RequestID:      abortReq.RequestID,
				SessionID:      abortReq.SessionID,
				Approved:       false,
				RememberChoice: false,
			}

			if err := permissionManager.HandleResponse(response); err != nil {
				logger.LogErr(err, "failed to cancel permission request")
			} else {
				logger.Info("Cancelled permission request via abort",
					"requestId", abortReq.RequestID,
					"sessionId", abortReq.SessionID)
			}
		}
	}

	// Add an abort message to the session for the LLM to see
	// This message will be visible to the LLM and should cause it to stop what it's doing
	abortMessage := providers.ChatMessage{
		Role:    "user",
		Content: "IMPORTANT: User has requested to abort the current operation. Please stop immediately and acknowledge.",
	}

	err = database.AddMessage(abortReq.SessionID, abortMessage, "", nil)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to add abort message"), 500)
	}

	// Broadcast the abort message to the UI so it appears in the chat
	BroadcastMessage(abortReq.SessionID, map[string]interface{}{
		"role":    abortMessage.Role,
		"content": abortMessage.Content,
	})

	logger.Info("Sent abort message to session",
		"sessionId", abortReq.SessionID,
		"requestId", abortReq.RequestID)

	return c.WriteJSON(map[string]interface{}{
		"success": true,
		"message": "Abort message sent to LLM",
	})
}
