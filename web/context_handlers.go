package web

import (
	"encoding/json"
	"os"
	"strconv"

	"rcode/context"
	"rcode/tools"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

// Global context manager instance (shared across sessions)
var globalContextManager *context.Manager

// GetContextManager returns the global context manager, initializing if needed
func GetContextManager() *context.Manager {
	if globalContextManager == nil {
		globalContextManager = context.NewManager()
		
		// Try to initialize with current directory
		workDir, err := os.Getwd()
		if err != nil {
			logger.LogErr(err, "failed to get working directory")
			workDir = "."
		}
		
		if _, err := globalContextManager.ScanProject(workDir); err != nil {
			logger.LogErr(err, "failed to scan project on startup")
		}
	}
	return globalContextManager
}

// getProjectContextHandler returns the current project context
func getProjectContextHandler(c rweb.Context) error {
	cm := GetContextManager()
	
	if !cm.IsInitialized() {
		return c.WriteJSON(map[string]interface{}{
			"initialized": false,
			"message":     "Project context not initialized",
		})
	}
	
	ctx := cm.GetContext()
	if ctx == nil {
		return c.WriteError(serr.New("context not available"), 500)
	}
	
	// Convert to JSON-safe structure
	response := map[string]interface{}{
		"initialized": true,
		"root_path":   ctx.RootPath,
		"language":    ctx.Language,
		"framework":   ctx.Framework,
		"statistics":  ctx.Statistics,
		"patterns":    ctx.Patterns,
		"recent_files": ctx.RecentFiles,
		"modified_files": func() []string {
			files := make([]string, 0, len(ctx.ModifiedFiles))
			for file := range ctx.ModifiedFiles {
				files = append(files, file)
			}
			return files
		}(),
	}
	
	return c.WriteJSON(response)
}

// initializeProjectContextHandler initializes the project context
func initializeProjectContextHandler(c rweb.Context) error {
	// Parse request body
	var req struct {
		Path string `json:"path"`
	}
	
	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}
	
	// Default to current directory
	if req.Path == "" {
		var err error
		req.Path, err = os.Getwd()
		if err != nil {
			return c.WriteError(serr.Wrap(err, "failed to get working directory"), 500)
		}
	}
	
	cm := GetContextManager()
	projectCtx, err := cm.ScanProject(req.Path)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to scan project"), 500)
	}
	
	logger.Info("Initialized project context", 
		"language", projectCtx.Language,
		"framework", projectCtx.Framework,
		"files", projectCtx.Statistics.TotalFiles,
	)
	
	return c.WriteJSON(map[string]interface{}{
		"success": true,
		"context": projectCtx,
	})
}

// getRelevantFilesHandler returns files relevant to a task
func getRelevantFilesHandler(c rweb.Context) error {
	// Parse request body
	var req struct {
		Task     string `json:"task"`
		MaxFiles int    `json:"max_files"`
	}
	
	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}
	
	if req.MaxFiles <= 0 {
		req.MaxFiles = 20
	}
	
	cm := GetContextManager()
	if !cm.IsInitialized() {
		return c.WriteError(serr.New("context not initialized"), 400)
	}
	
	files, err := cm.PrioritizeFiles(req.Task)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to prioritize files"), 500)
	}
	
	// Limit to requested max
	if len(files) > req.MaxFiles {
		files = files[:req.MaxFiles]
	}
	
	return c.WriteJSON(map[string]interface{}{
		"files": files,
		"count": len(files),
	})
}

// getChangeTrackingHandler returns recent file changes
func getChangeTrackingHandler(c rweb.Context) error {
	cm := GetContextManager()
	
	// Get limit from query parameter
	limit := 50
	if limitStr := c.Request().QueryParam("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	
	changes := cm.GetRecentChanges(limit)
	
	return c.WriteJSON(map[string]interface{}{
		"changes": changes,
		"count":   len(changes),
	})
}

// getContextStatsHandler returns context statistics
func getContextStatsHandler(c rweb.Context) error {
	cm := GetContextManager()
	
	if !cm.IsInitialized() {
		return c.WriteJSON(map[string]interface{}{
			"initialized": false,
		})
	}
	
	ctx := cm.GetContext()
	if ctx == nil {
		return c.WriteError(serr.New("context not available"), 500)
	}
	
	// Get change tracking stats
	changeStats := context.ChangeStats{} // Default empty stats for now
	
	stats := map[string]interface{}{
		"initialized": true,
		"project": map[string]interface{}{
			"language":    ctx.Language,
			"framework":   ctx.Framework,
			"total_files": ctx.Statistics.TotalFiles,
			"total_lines": ctx.Statistics.TotalLines,
			"files_by_language": ctx.Statistics.FilesByLanguage,
		},
		"session": map[string]interface{}{
			"total_changes":    changeStats.TotalChanges,
			"files_changed":    changeStats.FileCount,
			"creates":          changeStats.CreateCount,
			"modifications":    changeStats.ModifyCount,
			"deletions":        changeStats.DeleteCount,
			"renames":          changeStats.RenameCount,
			"session_duration": "N/A", // Will implement session duration tracking later
		},
	}
	
	return c.WriteJSON(stats)
}

// suggestToolsHandler suggests tools based on a task description
func suggestToolsHandler(c rweb.Context) error {
	// Parse request body
	var req struct {
		Task string `json:"task"`
	}
	
	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}
	
	if req.Task == "" {
		return c.WriteError(serr.New("task description required"), 400)
	}
	
	// Create context-aware executor
	cm := GetContextManager()
	toolRegistry := tools.DefaultRegistry()
	contextExecutor := tools.NewContextAwareExecutor(toolRegistry, cm)
	
	// Get tool suggestions
	suggestions := contextExecutor.SuggestTools(req.Task)
	
	return c.WriteJSON(map[string]interface{}{
		"suggestions": suggestions,
		"count":       len(suggestions),
	})
}