package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// ToolUsage represents a tool usage record
type ToolUsage struct {
	ID         int                    `json:"id"`
	SessionID  string                 `json:"session_id"`
	ToolName   string                 `json:"tool_name"`
	Input      map[string]interface{} `json:"input"`
	Output     string                 `json:"output,omitempty"`
	ExecutedAt time.Time              `json:"executed_at"`
	DurationMs int                    `json:"duration_ms,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// LogToolUsage logs a tool execution
func (db *DB) LogToolUsage(sessionID, toolName string, input map[string]interface{}, output string, durationMs int, toolError error) error {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return serr.Wrap(err, "failed to marshal tool input")
	}

	var errorMsg string
	if toolError != nil {
		errorMsg = toolError.Error()
	}

	query := `
		INSERT INTO tool_usage (session_id, tool_name, input, output, executed_at, duration_ms, error)
		VALUES (?, ?, ?::JSON, ?, CURRENT_TIMESTAMP, ?, ?)
	`

	_, err = db.Exec(query, sessionID, toolName, string(inputJSON), output, durationMs, errorMsg)
	if err != nil {
		return serr.Wrap(err, "failed to log tool usage")
	}

	logger.Debug("Logged tool usage", "session_id", sessionID, "tool", toolName, "duration_ms", durationMs)
	return nil
}

// GetToolUsage retrieves tool usage for a session
func (db *DB) GetToolUsage(sessionID string) ([]*ToolUsage, error) {
	query := `
		SELECT id, session_id, tool_name, input, output, executed_at, duration_ms, error
		FROM tool_usage
		WHERE session_id = ?
		ORDER BY executed_at DESC
	`

	rows, err := db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usages []*ToolUsage
	for rows.Next() {
		var usage ToolUsage
		var inputJSON string
		var output sql.NullString
		var durationMs sql.NullInt64
		var errorMsg sql.NullString

		err := rows.Scan(
			&usage.ID,
			&usage.SessionID,
			&usage.ToolName,
			&inputJSON,
			&output,
			&usage.ExecutedAt,
			&durationMs,
			&errorMsg,
		)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan tool usage row")
		}

		// Parse input JSON
		usage.Input = make(map[string]interface{})
		if err := json.Unmarshal([]byte(inputJSON), &usage.Input); err != nil {
			logger.LogErr(err, "failed to parse tool input")
		}

		// Set nullable fields
		if output.Valid {
			usage.Output = output.String
		}
		if durationMs.Valid {
			usage.DurationMs = int(durationMs.Int64)
		}
		if errorMsg.Valid {
			usage.Error = errorMsg.String
		}

		usages = append(usages, &usage)
	}

	return usages, nil
}

// GetToolUsageStats retrieves statistics about tool usage
func (db *DB) GetToolUsageStats(sessionID string) (map[string]interface{}, error) {
	query := `
		SELECT 
			tool_name,
			COUNT(*) as usage_count,
			AVG(duration_ms) as avg_duration_ms,
			MAX(duration_ms) as max_duration_ms,
			SUM(CASE WHEN error IS NOT NULL AND error != '' THEN 1 ELSE 0 END) as error_count
		FROM tool_usage
		WHERE session_id = ?
		GROUP BY tool_name
	`

	rows, err := db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	toolStats := make([]map[string]interface{}, 0)

	for rows.Next() {
		var toolName string
		var usageCount int
		var avgDuration sql.NullFloat64
		var maxDuration sql.NullInt64
		var errorCount int

		err := rows.Scan(&toolName, &usageCount, &avgDuration, &maxDuration, &errorCount)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan tool stats row")
		}

		toolStat := map[string]interface{}{
			"tool_name":    toolName,
			"usage_count":  usageCount,
			"error_count":  errorCount,
		}

		if avgDuration.Valid {
			toolStat["avg_duration_ms"] = avgDuration.Float64
		}
		if maxDuration.Valid {
			toolStat["max_duration_ms"] = maxDuration.Int64
		}

		toolStats = append(toolStats, toolStat)
	}

	// Get overall statistics
	var totalUsage int
	var totalErrors int
	err = db.QueryRow(`
		SELECT 
			COUNT(*) as total_usage,
			SUM(CASE WHEN error IS NOT NULL AND error != '' THEN 1 ELSE 0 END) as total_errors
		FROM tool_usage
		WHERE session_id = ?
	`, sessionID).Scan(&totalUsage, &totalErrors)
	
	if err != nil && err != sql.ErrNoRows {
		return nil, serr.Wrap(err, "failed to get overall tool stats")
	}

	stats["total_usage"] = totalUsage
	stats["total_errors"] = totalErrors
	stats["by_tool"] = toolStats

	return stats, nil
}

// GetGlobalToolUsageStats retrieves tool usage statistics across all sessions
func (db *DB) GetGlobalToolUsageStats() (map[string]interface{}, error) {
	query := `
		SELECT 
			tool_name,
			COUNT(*) as usage_count,
			COUNT(DISTINCT session_id) as session_count,
			AVG(duration_ms) as avg_duration_ms,
			SUM(CASE WHEN error IS NOT NULL AND error != '' THEN 1 ELSE 0 END) as error_count
		FROM tool_usage
		GROUP BY tool_name
		ORDER BY usage_count DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	toolStats := make([]map[string]interface{}, 0)

	for rows.Next() {
		var toolName string
		var usageCount int
		var sessionCount int
		var avgDuration sql.NullFloat64
		var errorCount int

		err := rows.Scan(&toolName, &usageCount, &sessionCount, &avgDuration, &errorCount)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan global tool stats row")
		}

		toolStat := map[string]interface{}{
			"tool_name":     toolName,
			"usage_count":   usageCount,
			"session_count": sessionCount,
			"error_count":   errorCount,
		}

		if avgDuration.Valid {
			toolStat["avg_duration_ms"] = avgDuration.Float64
		}

		toolStats = append(toolStats, toolStat)
	}

	stats["tools"] = toolStats
	return stats, nil
}

// CleanupOldToolUsage removes tool usage records older than the specified duration
func (db *DB) CleanupOldToolUsage(olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	
	result, err := db.Exec(
		"DELETE FROM tool_usage WHERE executed_at < ?",
		cutoffTime,
	)
	if err != nil {
		return serr.Wrap(err, "failed to cleanup old tool usage")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return serr.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected > 0 {
		logger.Info("Cleaned up old tool usage records", "count", rowsAffected)
	}

	return nil
}