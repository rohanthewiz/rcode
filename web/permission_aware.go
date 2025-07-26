package web

import (
	"fmt"
	"strings"

	"rcode/db"
	"rcode/tools"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// PermissionAwareExecutor wraps tool execution with permission checks
type PermissionAwareExecutor struct {
	executor     *tools.ContextAwareExecutor
	database     *db.DB
	onAskHandler func(sessionID, toolName string, params map[string]interface{}) (bool, error)
}

// NewPermissionAwareExecutor creates a new permission-aware executor
func NewPermissionAwareExecutor(executor *tools.ContextAwareExecutor, database *db.DB) *PermissionAwareExecutor {
	return &PermissionAwareExecutor{
		executor: executor,
		database: database,
	}
}

// SetAskHandler sets the handler for tools that require confirmation
func (e *PermissionAwareExecutor) SetAskHandler(handler func(sessionID, toolName string, params map[string]interface{}) (bool, error)) {
	e.onAskHandler = handler
}

// Execute runs a tool with permission checks
func (e *PermissionAwareExecutor) Execute(toolUse tools.ToolUse) (*tools.ToolResult, error) {
	// Extract session ID from input
	sessionID, ok := toolUse.Input["_sessionId"].(string)
	if !ok || sessionID == "" {
		// No session context, execute without permission check
		logger.Debug("No session ID in tool use, executing without permission check", "tool", toolUse.Name)
		return e.executor.Execute(toolUse)
	}

	// Check tool permission
	permType, scope, err := e.database.CheckToolPermission(sessionID, toolUse.Name)
	if err != nil {
		logger.LogErr(err, "failed to check tool permission", "tool", toolUse.Name, "session", sessionID)
		// On error, default to ask mode
		permType = db.PermissionAsk
	}

	logger.Debug("Checking tool permission", "tool", toolUse.Name, "session", sessionID, "permission", permType)

	switch permType {
	case db.PermissionDenied:
		// Tool is denied
		return &tools.ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Tool '%s' is disabled for this session. Please enable it in the Tools tab to use it.", toolUse.Name),
		}, serr.New("tool is disabled")

	case db.PermissionAsk:
		// Tool requires confirmation
		if e.onAskHandler != nil {
			// Create a copy of params without internal fields
			cleanParams := make(map[string]interface{})
			for k, v := range toolUse.Input {
				if !strings.HasPrefix(k, "_") {
					cleanParams[k] = v
				}
			}

			approved, err := e.onAskHandler(sessionID, toolUse.Name, cleanParams)
			if err != nil {
				return &tools.ToolResult{
					Type:      "tool_result",
					ToolUseID: toolUse.ID,
					Content:   fmt.Sprintf("Error requesting permission: %v", err),
				}, err
			}

			if !approved {
				return &tools.ToolResult{
					Type:      "tool_result",
					ToolUseID: toolUse.ID,
					Content:   fmt.Sprintf("Tool '%s' execution was not approved by user.", toolUse.Name),
				}, serr.New("tool execution not approved")
			}
		} else {
			// No ask handler configured, log warning and proceed
			logger.Warn("Tool requires ask permission but no handler configured", "tool", toolUse.Name)
		}

	case db.PermissionAllowed:
		// Tool is allowed, proceed with execution
		logger.Debug("Tool allowed, executing", "tool", toolUse.Name)
	}

	// Apply scope restrictions if any
	if scope != nil {
		if err := e.applyScopeRestrictions(toolUse, scope); err != nil {
			return &tools.ToolResult{
				Type:      "tool_result",
				ToolUseID: toolUse.ID,
				Content:   fmt.Sprintf("Scope restriction error: %v", err),
			}, err
		}
	}

	// Execute the tool
	return e.executor.Execute(toolUse)
}

// applyScopeRestrictions applies permission scope restrictions to tool parameters
func (e *PermissionAwareExecutor) applyScopeRestrictions(toolUse tools.ToolUse, scope *db.PermissionScope) error {
	// Check path restrictions for file tools
	if len(scope.Paths) > 0 {
		if path, ok := tools.GetString(toolUse.Input, "path"); ok {
			allowed := false
			for _, allowedPath := range scope.Paths {
				if strings.HasPrefix(path, allowedPath) {
					allowed = true
					break
				}
			}
			if !allowed {
				return serr.New("path not allowed by permission scope")
			}
		}
	}

	// Check file size restrictions
	if scope.MaxFileSize > 0 {
		if toolUse.Name == "write_file" {
			if content, ok := tools.GetString(toolUse.Input, "content"); ok {
				if int64(len(content)) > scope.MaxFileSize {
					return serr.New("file size exceeds permission scope limit")
				}
			}
		}
	}

	// Check allowed commands for bash tool
	if toolUse.Name == "bash" && len(scope.AllowedCmds) > 0 {
		if cmd, ok := tools.GetString(toolUse.Input, "command"); ok {
			allowed := false
			for _, allowedCmd := range scope.AllowedCmds {
				if strings.HasPrefix(cmd, allowedCmd) {
					allowed = true
					break
				}
			}
			if !allowed {
				return serr.New("command not allowed by permission scope")
			}
		}
	}

	return nil
}

// HandleAskPermission handles ask permission requests via SSE
// It sends a permission request to the frontend and waits for the user's response
func HandleAskPermission(sessionID, toolName string, params map[string]interface{}) (bool, error) {
	// Create a permission request
	request, err := permissionManager.CreateRequest(sessionID, toolName, params)
	if err != nil {
		return false, serr.Wrap(err, "failed to create permission request")
	}

	// Broadcast the permission request to the frontend
	BroadcastPermissionRequest(request)

	// Wait for the response
	response, err := permissionManager.WaitForResponse(request.ID)
	if err != nil {
		// Check if it was a timeout
		if err.Error() == "permission request timed out" {
			// Broadcast timeout event
			BroadcastPermissionTimeout(sessionID, request.ID)
		}
		return false, err
	}

	return response.Approved, nil
}