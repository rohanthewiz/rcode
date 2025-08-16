package web

import (
	"encoding/json"
	"fmt"
	"strings"

	"rcode/db"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

// CompactionRequest represents a request to compact a session
type CompactionRequest struct {
	PreserveRecent       int    `json:"preserve_recent,omitempty"`
	PreserveInitial      int    `json:"preserve_initial,omitempty"`
	Strategy             string `json:"strategy,omitempty"` // "aggressive" or "conservative"
	MaxSummaryTokens     int    `json:"max_summary_tokens,omitempty"`
	MinMessagesToCompact int    `json:"min_messages_to_compact,omitempty"`
}

// compactSessionHandler handles requests to compact a session's messages
func compactSessionHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")
	
	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Check if session exists
	session, err := database.GetSession(sessionID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get session"), 500)
	}
	if session == nil {
		return c.WriteError(serr.New("session not found"), 404)
	}

	// Parse request body for options
	var req CompactionRequest
	body := c.Request().Body()
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			logger.LogErr(err, "failed to parse compaction request, using defaults")
		}
	}

	// Build compaction options
	opts := db.DefaultCompactionOptions()
	if req.PreserveRecent > 0 {
		opts.PreserveRecent = req.PreserveRecent
	}
	if req.PreserveInitial > 0 {
		opts.PreserveInitial = req.PreserveInitial
	}
	if req.Strategy != "" {
		opts.Strategy = req.Strategy
	}
	if req.MaxSummaryTokens > 0 {
		opts.MaxSummaryTokens = req.MaxSummaryTokens
	}
	if req.MinMessagesToCompact > 0 {
		opts.MinMessagesToCompact = req.MinMessagesToCompact
	}

	// Perform compaction
	compactedMsg, err := database.CompactSessionMessages(sessionID, opts)
	if err != nil {
		// Check if it's a "not enough messages" error
		errStr := err.Error()
		if strings.Contains(errStr, "not enough messages") || strings.Contains(errStr, "no messages in compactable range") {
			return c.WriteError(err, 400) // Bad request - not enough messages
		}
		return c.WriteError(serr.Wrap(err, "failed to compact messages"), 500)
	}

	// Broadcast session update
	BroadcastSessionUpdate(sessionID, "session_compacted", map[string]interface{}{
		"compaction_id": compactedMsg.ID,
	})

	// Return compaction result
	return c.WriteJSON(map[string]interface{}{
		"success":            true,
		"compacted_message":  compactedMsg,
		"messages_compacted": len(compactedMsg.OriginalMessageIDs),
		"tokens_saved":       compactedMsg.TokenCountBefore - compactedMsg.TokenCountAfter,
	})
}

// getCompactionStatsHandler returns compaction statistics for a session
func getCompactionStatsHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Get compaction stats
	stats, err := database.GetCompactionStats(sessionID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get compaction stats"), 500)
	}

	return c.WriteJSON(stats)
}

// restoreCompactedMessagesHandler restores archived messages from a compaction
func restoreCompactedMessagesHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")
	compactionID := c.Request().Param("compactionId")

	// Convert compactionID to int
	var compID int
	if _, err := fmt.Sscanf(compactionID, "%d", &compID); err != nil {
		return c.WriteError(serr.New("invalid compaction ID"), 400)
	}

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Restore messages
	err = database.RestoreCompactedMessages(sessionID, compID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to restore messages"), 500)
	}

	// Broadcast session update
	BroadcastSessionUpdate(sessionID, "messages_restored", map[string]interface{}{
		"compaction_id": compID,
	})

	return c.WriteJSON(map[string]interface{}{
		"success": true,
		"message": "Messages restored successfully",
	})
}

// getCompactedMessagesHandler retrieves compacted messages for a session
func getCompactedMessagesHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Get compacted messages
	compactedMessages, err := database.GetCompactedMessages(sessionID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get compacted messages"), 500)
	}

	return c.WriteJSON(compactedMessages)
}

// AutoCompactSettings represents settings for auto-compaction
type AutoCompactSettings struct {
	Enabled   bool `json:"enabled"`
	Threshold int  `json:"threshold"` // Token threshold
}

// updateAutoCompactHandler updates auto-compaction settings for a session
func updateAutoCompactHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")

	// Parse request body
	var settings AutoCompactSettings
	body := c.Request().Body()
	if err := json.Unmarshal(body, &settings); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Update session settings
	query := `
		UPDATE sessions 
		SET auto_compact_enabled = ?, 
		    compact_threshold = ?
		WHERE id = ?
	`
	
	_, err = database.Exec(query, settings.Enabled, settings.Threshold, sessionID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to update auto-compact settings"), 500)
	}

	logger.Info("Updated auto-compact settings", 
		"session_id", sessionID, 
		"enabled", settings.Enabled, 
		"threshold", settings.Threshold)

	return c.WriteJSON(map[string]interface{}{
		"success": true,
		"settings": settings,
	})
}