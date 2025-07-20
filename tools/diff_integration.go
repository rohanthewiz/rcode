package tools

import (
	"fmt"
	"os"
	"rcode/diff"
	"strings"
	"time"

	"github.com/rohanthewiz/logger"
)

// DiffIntegration handles diff-related functionality for tools
type DiffIntegration struct {
	diffService *diff.DiffService
}

// NewDiffIntegration creates a new diff integration handler
func NewDiffIntegration() (*DiffIntegration, error) {
	return &DiffIntegration{
		diffService: diff.NewDiffService(),
	}, nil
}

// SetupDiffHooks registers before/after execution hooks for diff capture.
// Should be called during tool registry initialization.
func (di *DiffIntegration) SetupDiffHooks(registry *EnhancedRegistry) {
	// Add before-execute hook to capture snapshots
	registry.AddBeforeExecuteHook(di.beforeFileModification)

	// Add after-execute hook to generate diffs
	registry.AddAfterExecuteHook(di.afterFileModification)
}

// beforeFileModification captures file snapshots before modification tools run.
func (di *DiffIntegration) beforeFileModification(toolName string, params map[string]interface{}) error {
	// Only capture snapshots for file modification tools
	if !isFileModificationTool(toolName) {
		return nil
	}

	// Extract session ID from context (if available)
	sessionID, _ := params["_sessionId"].(string)
	if sessionID == "" {
		// No session context, skip snapshot
		return nil
	}

	// Extract file path based on tool
	filePath := extractFilePath(toolName, params)
	if filePath == "" {
		return nil
	}

	// Expand the path
	expandedPath, err := ExpandPath(filePath)
	if err != nil {
		logger.LogErr(err, "failed to expand path for snapshot", "path", filePath)
		return nil // Don't fail the tool execution
	}

	// Read current file content (if exists)
	content := ""
	if data, err := os.ReadFile(expandedPath); err == nil {
		content = string(data)
	} else if !os.IsNotExist(err) {
		logger.LogErr(err, "failed to read file for snapshot", "path", expandedPath)
		return nil // Don't fail the tool execution
	}

	// Create snapshot in memory
	toolID := fmt.Sprintf("%s_%d", toolName, time.Now().UnixNano())
	_, err = di.diffService.CreateSnapshot(sessionID, filePath, content, toolID)
	if err != nil {
		logger.LogErr(err, "failed to create snapshot")
		return nil // Don't fail the tool execution
	}

	// Store tool execution ID for later use
	params["_toolExecutionId"] = toolID

	logger.Debug("Created file snapshot before modification",
		"tool", toolName,
		"path", filePath,
		"sessionId", sessionID,
	)

	return nil
}

// afterFileModification generates diffs after file modification tools complete.
func (di *DiffIntegration) afterFileModification(toolName string, params map[string]interface{}, result *ToolResult, err error) {
	// Only process successful file modifications
	if err != nil || !isFileModificationTool(toolName) {
		return
	}

	// Extract session ID
	sessionID, _ := params["_sessionId"].(string)
	if sessionID == "" {
		return
	}

	// Extract file path
	filePath := extractFilePath(toolName, params)
	if filePath == "" {
		return
	}

	// Check if we have a snapshot
	if !di.diffService.HasChanges(sessionID, filePath, "") {
		// No snapshot or no changes
		return
	}

	// Read the new file content
	expandedPath, err := ExpandPath(filePath)
	if err != nil {
		logger.LogErr(err, "failed to expand path for diff", "path", filePath)
		return
	}

	newContent := ""
	if data, err := os.ReadFile(expandedPath); err == nil {
		newContent = string(data)
	} else {
		logger.LogErr(err, "failed to read file for diff", "path", expandedPath)
		return
	}

	// Generate diff
	diffResult, err := di.diffService.GenerateDiff(sessionID, filePath, newContent)
	if err != nil {
		logger.LogErr(err, "failed to generate diff")
		return
	}

	// For now, we'll use a simple ID based on timestamp
	// In production, this would be saved to database
	diffID := time.Now().UnixNano()
	toolExecutionID, _ := params["_toolExecutionId"].(string)

	// Clear the in-memory snapshot
	di.diffService.ClearSnapshot(sessionID, filePath)

	// Broadcast diff available event with temporary ID
	diff.BroadcastDiffAvailable(sessionID, diffID, filePath, diffResult.Stats, toolName)

	logger.Debug("Generated diff after file modification",
		"tool", toolName,
		"path", filePath,
		"sessionId", sessionID,
		"toolExecutionId", toolExecutionID,
		"added", diffResult.Stats.Added,
		"deleted", diffResult.Stats.Deleted,
	)
}

// isFileModificationTool checks if a tool modifies files.
func isFileModificationTool(toolName string) bool {
	switch toolName {
	case "write_file", "edit_file", "move", "remove":
		return true
	default:
		return false
	}
}

// extractFilePath extracts the file path from tool parameters.
func extractFilePath(toolName string, params map[string]interface{}) string {
	switch toolName {
	case "write_file", "edit_file", "remove":
		if path, ok := params["path"].(string); ok {
			return path
		}
	case "move":
		// For move, we track the destination
		if dest, ok := params["destination"].(string); ok {
			// Check if destination is a file
			if !strings.HasSuffix(dest, "/") {
				return dest
			}
		}
	}
	return ""
}
