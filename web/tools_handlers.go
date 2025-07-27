package web

import (
	"encoding/json"
	
	"rcode/db"
	"rcode/tools"
	
	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

// ToolInfo represents tool information with permission status
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Enabled     bool   `json:"enabled"`     // false if denied, true otherwise
	Mode        string `json:"mode"`        // "ask" or "auto"
}

// ToolPermissionUpdate represents a permission update request
type ToolPermissionUpdate struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"` // "ask" or "auto"
}

// getSessionToolsHandler returns all available tools with their current permissions
func getSessionToolsHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")
	
	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}
	
	// Get session to ensure it exists
	session, err := database.GetSession(sessionID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get session"), 500)
	}
	if session == nil {
		return c.WriteError(serr.New("session not found"), 404)
	}
	
	// Get all permissions for this session
	permissions, err := database.GetSessionPermissions(sessionID)
	if err != nil {
		logger.LogErr(err, "failed to get session permissions")
		permissions = []*db.ToolPermission{}
	}
	
	// Create a map of tool permissions for quick lookup
	permMap := make(map[string]*db.ToolPermission)
	for _, perm := range permissions {
		permMap[perm.ToolName] = perm
	}
	
	// Get tool registry
	registry := tools.DefaultRegistry()
	availableTools := registry.GetTools()
	
	// Build tool info list
	toolInfos := make([]ToolInfo, 0, len(availableTools))
	for _, tool := range availableTools {
		info := ToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			Category:    categorizeTools(tool.Name),
			Enabled:     true,  // Default enabled
			Mode:        "ask", // Default ask mode
		}
		
		// Check if we have a permission for this tool
		if perm, exists := permMap[tool.Name]; exists {
			switch perm.PermissionType {
			case db.PermissionDenied:
				info.Enabled = false
				info.Mode = "ask" // Mode doesn't matter when disabled
			case db.PermissionAllowed:
				info.Enabled = true
				info.Mode = "auto"
			case db.PermissionAsk:
				info.Enabled = true
				info.Mode = "ask"
			}
		}
		
		toolInfos = append(toolInfos, info)
	}
	
	return c.WriteJSON(toolInfos)
}

// updateToolPermissionHandler updates a tool's permission for a session
func updateToolPermissionHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")
	toolName := c.Request().Param("tool")
	
	// Parse request body
	body := c.Request().Body()
	var update ToolPermissionUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}
	
	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}
	
	// Determine permission type based on enabled and mode
	var permType db.PermissionType
	if !update.Enabled {
		permType = db.PermissionDenied
	} else if update.Mode == "auto" {
		permType = db.PermissionAllowed
	} else {
		permType = db.PermissionAsk
	}
	
	// Update permission in database
	err = database.SetToolPermission(sessionID, toolName, permType, nil, 0)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to update tool permission"), 500)
	}
	
	logger.Info("Updated tool permission", "session_id", sessionID, "tool", toolName, "permission", permType)
	
	// Broadcast the permission update
	BroadcastToolPermissionUpdate(sessionID, toolName, update.Enabled, update.Mode)
	
	return c.WriteJSON(map[string]interface{}{
		"success": true,
		"tool":    toolName,
		"enabled": update.Enabled,
		"mode":    update.Mode,
	})
}

// categorizeTools returns a category for grouping tools in the UI
func categorizeTools(toolName string) string {
	categories := map[string]string{
		// File operations
		"read_file":  "File Operations",
		"write_file": "File Operations",
		"edit_file":  "File Operations",
		"search":     "File Operations",
		
		// Directory operations
		"list_dir": "Directory Operations",
		"tree":     "Directory Operations",
		"make_dir": "Directory Operations",
		"remove":   "Directory Operations",
		"move":     "Directory Operations",
		
		// Git operations
		"git_status":   "Git Operations",
		"git_diff":     "Git Operations",
		"git_log":      "Git Operations",
		"git_branch":   "Git Operations",
		"git_add":      "Git Operations",
		"git_commit":   "Git Operations",
		"git_push":     "Git Operations",
		"git_pull":     "Git Operations",
		"git_checkout": "Git Operations",
		"git_merge":    "Git Operations",
		
		// System operations
		"bash": "System Operations",
		
		// Web operations
		"web_search": "Web Operations",
		"web_fetch":  "Web Operations",
	}
	
	if category, exists := categories[toolName]; exists {
		return category
	}
	
	// Default for custom tools
	return "Custom Tools"
}