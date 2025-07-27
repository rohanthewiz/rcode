package web

import (
	"encoding/json"

	"rcode/db"

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
