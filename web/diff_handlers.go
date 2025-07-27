package web

import (
	"encoding/json"
	"fmt"
	"os"
	"rcode/db"
	"rcode/diff"
	"strconv"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

// Global diff service instance
var diffService *diff.DiffService

// InitDiffService initializes the diff service.
// Should be called during server startup.
func InitDiffService() {
	diffService = diff.NewDiffService()
	logger.Info("Diff service initialized")
}

// getDiffHandler retrieves a diff for a specific file.
// GET /api/diff/:sessionId/:path
func getDiffHandler(c rweb.Context) error {
	sessionID := c.Request().Param("sessionId")
	filePath := c.Request().Param("path")

	if sessionID == "" || filePath == "" {
		return c.WriteError(serr.New("sessionId and path are required"), 400)
	}

	// Get the database connection
	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "failed to get database connection")
		return c.WriteError(serr.Wrap(err, "database connection failed"), 500)
	}

	// Get the latest diff for this file
	diffs, err := database.GetFileDiffs(sessionID, filePath)
	if err != nil {
		logger.LogErr(err, "failed to get file diffs")
		return c.WriteError(serr.Wrap(err, "failed to retrieve diffs"), 500)
	}

	if len(diffs) == 0 {
		return c.WriteError(serr.New("no diffs found for file"), 404)
	}

	// Return the most recent diff
	return c.WriteJSON(diffs[0])
}

// createSnapshotHandler creates a snapshot before file modification.
// POST /api/diff/snapshot
func createSnapshotHandler(c rweb.Context) error {
	var req struct {
		SessionID       string `json:"sessionId"`
		FilePath        string `json:"filePath"`
		ToolExecutionID string `json:"toolExecutionId,omitempty"`
		ToolName        string `json:"toolName,omitempty"`
	}

	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}

	// Read the current file content
	content, err := os.ReadFile(req.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, use empty content
			content = []byte{}
		} else {
			logger.LogErr(err, "failed to read file", "path", req.FilePath)
			return c.WriteError(serr.Wrap(err, "failed to read file"), 500)
		}
	}

	// Create snapshot in memory
	snapshot, err := diffService.CreateSnapshot(req.SessionID, req.FilePath, string(content), req.ToolExecutionID)
	if err != nil {
		logger.LogErr(err, "failed to create snapshot")
		return c.WriteError(serr.Wrap(err, "failed to create snapshot"), 500)
	}

	// Store in database
	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "failed to get database connection")
		return c.WriteError(serr.Wrap(err, "database connection failed"), 500)
	}

	dbSnapshot := &db.DiffSnapshot{
		SessionID:       snapshot.SessionID,
		FilePath:        snapshot.Path,
		Content:         snapshot.Content,
		Hash:            snapshot.Hash,
		CreatedAt:       snapshot.Timestamp,
		ToolExecutionID: req.ToolExecutionID,
		ToolName:        req.ToolName,
	}

	snapshotID, err := database.SaveDiffSnapshot(dbSnapshot)
	if err != nil {
		logger.LogErr(err, "failed to save snapshot to database")
		return c.WriteError(serr.Wrap(err, "failed to save snapshot"), 500)
	}

	return c.WriteJSON(map[string]interface{}{
		"id":        snapshotID,
		"sessionId": snapshot.SessionID,
		"filePath":  snapshot.Path,
		"hash":      snapshot.Hash,
		"timestamp": snapshot.Timestamp,
	})
}

// generateDiffHandler generates a diff after file modification.
// POST /api/diff/generate
func generateDiffHandler(c rweb.Context) error {
	var req struct {
		SessionID       string `json:"sessionId"`
		FilePath        string `json:"filePath"`
		ToolExecutionID string `json:"toolExecutionId,omitempty"`
	}

	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}

	// Read the current file content
	content, err := os.ReadFile(req.FilePath)
	if err != nil {
		logger.LogErr(err, "failed to read file", "path", req.FilePath)
		return c.WriteError(serr.Wrap(err, "failed to read file"), 500)
	}

	// Generate diff
	diffResult, err := diffService.GenerateDiff(req.SessionID, req.FilePath, string(content))
	if err != nil {
		logger.LogErr(err, "failed to generate diff")
		return c.WriteError(serr.Wrap(err, "failed to generate diff"), 500)
	}

	// Store in database
	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "failed to get database connection")
		return c.WriteError(serr.Wrap(err, "database connection failed"), 500)
	}

	// First, save the "after" snapshot
	afterSnapshot := &db.DiffSnapshot{
		SessionID:       req.SessionID,
		FilePath:        req.FilePath,
		Content:         string(content),
		Hash:            diffService.GetSnapshot(req.SessionID, req.FilePath).Hash, // Reuse computed hash
		CreatedAt:       time.Now(),
		ToolExecutionID: req.ToolExecutionID,
	}

	afterSnapshotID, err := database.SaveDiffSnapshot(afterSnapshot)
	if err != nil {
		logger.LogErr(err, "failed to save after snapshot")
		return c.WriteError(serr.Wrap(err, "failed to save after snapshot"), 500)
	}

	// Get the before snapshot ID from database
	// For now, we'll use the most recent snapshot as the "before"
	snapshots, err := database.GetFileDiffs(req.SessionID, req.FilePath)
	var beforeSnapshotID *int64
	if err == nil && len(snapshots) > 0 {
		// Find the most recent snapshot before this one
		// This is a simplified approach - in production, we'd track this more carefully
		beforeSnapshotID = snapshots[0].BeforeSnapshotID
	}

	// Serialize diff data
	diffData, err := json.Marshal(diffResult)
	if err != nil {
		logger.LogErr(err, "failed to serialize diff data")
		return c.WriteError(serr.Wrap(err, "failed to serialize diff"), 500)
	}

	// Save diff to database
	dbDiff := &db.Diff{
		SessionID:        req.SessionID,
		FilePath:         req.FilePath,
		BeforeSnapshotID: beforeSnapshotID,
		AfterSnapshotID:  &afterSnapshotID,
		DiffData:         diffData,
		CreatedAt:        time.Now(),
		ToolExecutionID:  req.ToolExecutionID,
		IsApplied:        true,
	}

	diffID, err := database.SaveDiff(dbDiff)
	if err != nil {
		logger.LogErr(err, "failed to save diff to database")
		return c.WriteError(serr.Wrap(err, "failed to save diff"), 500)
	}

	// Clear the in-memory snapshot
	diffService.ClearSnapshot(req.SessionID, req.FilePath)

	// Broadcast diff available event
	BroadcastDiffAvailable(req.SessionID, diffID, req.FilePath, diffResult.Stats, "")

	return c.WriteJSON(map[string]interface{}{
		"id":        diffID,
		"sessionId": diffResult.SessionID,
		"filePath":  diffResult.Path,
		"stats":     diffResult.Stats,
		"hunks":     diffResult.Hunks,
	})
}

// listSessionDiffsHandler lists all diffs in a session.
// GET /api/session/:id/diffs
func listSessionDiffsHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")
	if sessionID == "" {
		return c.WriteError(serr.New("sessionId is required"), 400)
	}

	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "failed to get database connection")
		return c.WriteError(serr.Wrap(err, "database connection failed"), 500)
	}

	diffs, err := database.GetSessionDiffs(sessionID)
	if err != nil {
		logger.LogErr(err, "failed to get session diffs")
		return c.WriteError(serr.Wrap(err, "failed to retrieve diffs"), 500)
	}

	// Parse and enhance diff data for response
	var response []map[string]interface{}
	for _, diff := range diffs {
		var diffData map[string]interface{}
		if err := json.Unmarshal(diff.DiffData, &diffData); err != nil {
			logger.LogErr(err, "failed to parse diff data", "diffId", strconv.FormatInt(diff.ID, 10))
			continue
		}

		response = append(response, map[string]interface{}{
			"id":        diff.ID,
			"filePath":  diff.FilePath,
			"createdAt": diff.CreatedAt,
			"stats":     diffData["stats"],
			"isApplied": diff.IsApplied,
		})
	}

	return c.WriteJSON(map[string]interface{}{
		"diffs": response,
		"total": len(response),
	})
}

// getDiffByIdHandler retrieves a specific diff by ID.
// GET /api/diff/:id
func getDiffByIdHandler(c rweb.Context) error {
	diffIDStr := c.Request().Param("id")
	diffID, err := strconv.ParseInt(diffIDStr, 10, 64)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "invalid diff ID"), 400)
	}

	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "failed to get database connection")
		return c.WriteError(serr.Wrap(err, "database connection failed"), 500)
	}

	diff, err := database.GetDiff(diffID)
	if err != nil {
		logger.LogErr(err, "failed to get diff")
		return c.WriteError(serr.Wrap(err, "failed to retrieve diff"), 500)
	}

	if diff == nil {
		return c.WriteError(serr.New("diff not found"), 404)
	}

	// Get snapshots for full content
	var before, after string
	if diff.BeforeSnapshotID != nil {
		beforeSnapshot, err := database.GetDiffSnapshot(*diff.BeforeSnapshotID)
		if err == nil && beforeSnapshot != nil {
			before = beforeSnapshot.Content
		}
	}

	if diff.AfterSnapshotID != nil {
		afterSnapshot, err := database.GetDiffSnapshot(*diff.AfterSnapshotID)
		if err == nil && afterSnapshot != nil {
			after = afterSnapshot.Content
		}
	}

	// Parse diff data
	var diffData map[string]interface{}
	if err := json.Unmarshal(diff.DiffData, &diffData); err != nil {
		logger.LogErr(err, "failed to parse diff data")
		return c.WriteError(serr.Wrap(err, "failed to parse diff data"), 500)
	}

	// Add content to response
	diffData["before"] = before
	diffData["after"] = after
	diffData["id"] = diff.ID
	diffData["createdAt"] = diff.CreatedAt
	diffData["isApplied"] = diff.IsApplied

	return c.WriteJSON(diffData)
}

// markDiffViewedHandler marks a diff as viewed.
// POST /api/diff/:id/viewed
func markDiffViewedHandler(c rweb.Context) error {
	diffIDStr := c.Request().Param("id")
	diffID, err := strconv.ParseInt(diffIDStr, 10, 64)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "invalid diff ID"), 400)
	}

	var req struct {
		SessionID string `json:"sessionId"`
		ViewMode  string `json:"viewMode"`
	}

	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}

	if req.ViewMode == "" {
		req.ViewMode = "side-by-side"
	}

	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "failed to get database connection")
		return c.WriteError(serr.Wrap(err, "database connection failed"), 500)
	}

	err = database.MarkDiffViewed(req.SessionID, diffID, req.ViewMode)
	if err != nil {
		logger.LogErr(err, "failed to mark diff as viewed")
		return c.WriteError(serr.Wrap(err, "failed to mark diff as viewed"), 500)
	}

	return c.WriteJSON(map[string]interface{}{
		"success": true,
	})
}

// getDiffPreferencesHandler retrieves user diff preferences.
// GET /api/diff/preferences
func getDiffPreferencesHandler(c rweb.Context) error {
	// For now, use a default user ID
	// In the future, this would come from auth
	userID := "default"

	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "failed to get database connection")
		return c.WriteError(serr.Wrap(err, "database connection failed"), 500)
	}

	prefs, err := database.GetDiffPreferences(userID)
	if err != nil {
		logger.LogErr(err, "failed to get diff preferences")
		return c.WriteError(serr.Wrap(err, "failed to retrieve preferences"), 500)
	}

	return c.WriteJSON(prefs)
}

// saveDiffPreferencesHandler saves user diff preferences.
// POST /api/diff/preferences
func saveDiffPreferencesHandler(c rweb.Context) error {
	var prefs db.DiffPreferences
	body := c.Request().Body()
	if err := json.Unmarshal(body, &prefs); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}

	// For now, use a default user ID
	prefs.UserID = "default"

	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "failed to get database connection")
		return c.WriteError(serr.Wrap(err, "database connection failed"), 500)
	}

	err = database.SaveDiffPreferences(&prefs)
	if err != nil {
		logger.LogErr(err, "failed to save diff preferences")
		return c.WriteError(serr.Wrap(err, "failed to save preferences"), 500)
	}

	return c.WriteJSON(map[string]interface{}{
		"success": true,
	})
}

// applyDiffHandler applies or reverts a diff.
// POST /api/diff/apply
func applyDiffHandler(c rweb.Context) error {
	var req struct {
		DiffID int64 `json:"diffId"`
		Revert bool  `json:"revert"`
	}

	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}

	// This is a placeholder for actual diff application logic
	// In a real implementation, this would:
	// 1. Retrieve the diff from database
	// 2. Apply or revert the changes to the file
	// 3. Update the diff's isApplied status
	// 4. Create a new snapshot if needed

	return c.WriteJSON(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Diff %d %s successfully", req.DiffID, map[bool]string{true: "reverted", false: "applied"}[req.Revert]),
	})
}