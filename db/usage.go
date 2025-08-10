package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/rohanthewiz/serr"
	"rcode/providers"
)

// UsageRecord represents a usage tracking record in the database
type UsageRecord struct {
	ID            int                      `json:"id"`
	SessionID     string                   `json:"session_id"`
	MessageID     sql.NullInt64            `json:"message_id,omitempty"`
	Model         string                   `json:"model"`
	InputTokens   int                      `json:"input_tokens"`
	OutputTokens  int                      `json:"output_tokens"`
	RateLimitInfo *providers.RateLimitInfo `json:"rate_limit_info,omitempty"`
	CreatedAt     time.Time                `json:"created_at"`
}

// CreateUsageTable creates the usage tracking table if it doesn't exist
func (db *DB) CreateUsageTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS usage_tracking (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			message_id INTEGER,
			model TEXT NOT NULL,
			input_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			rate_limits JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_id) REFERENCES sessions(id),
			FOREIGN KEY (message_id) REFERENCES messages(id)
		)
	`
	_, err := db.conn.Exec(query)
	if err != nil {
		return serr.Wrap(err, "failed to create usage_tracking table")
	}

	// Create index for faster queries
	_, err = db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_usage_session ON usage_tracking(session_id)`)
	if err != nil {
		return serr.Wrap(err, "failed to create usage index")
	}

	_, err = db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_usage_created ON usage_tracking(created_at)`)
	if err != nil {
		return serr.Wrap(err, "failed to create usage timestamp index")
	}

	return nil
}

// RecordUsage records token usage and rate limit information
func (db *DB) RecordUsage(sessionID string, messageID *int, model string, usage *providers.Usage, rateLimits *providers.RateLimitInfo) error {
	if usage == nil {
		return nil // Nothing to record
	}

	var rateLimitsJSON string
	if rateLimits != nil {
		data, err := json.Marshal(rateLimits)
		if err != nil {
			return serr.Wrap(err, "failed to marshal rate limits")
		}
		rateLimitsJSON = string(data)
	} else {
		rateLimitsJSON = "null"
	}

	var msgID sql.NullInt64
	if messageID != nil {
		msgID = sql.NullInt64{Int64: int64(*messageID), Valid: true}
	}

	query := `
		INSERT INTO usage_tracking (session_id, message_id, model, input_tokens, output_tokens, rate_limits)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := db.conn.Exec(query, sessionID, msgID, model, usage.InputTokens, usage.OutputTokens, rateLimitsJSON)
	if err != nil {
		return serr.Wrap(err, "failed to record usage")
	}

	return nil
}

// GetSessionUsage gets total usage for a session
func (db *DB) GetSessionUsage(sessionID string) (totalInput int, totalOutput int, latestRateLimits *providers.RateLimitInfo, err error) {
	// Get total tokens
	query := `
		SELECT 
			COALESCE(SUM(input_tokens), 0) as total_input,
			COALESCE(SUM(output_tokens), 0) as total_output
		FROM usage_tracking
		WHERE session_id = ?
	`
	err = db.conn.QueryRow(query, sessionID).Scan(&totalInput, &totalOutput)
	if err != nil {
		return 0, 0, nil, serr.Wrap(err, "failed to get session usage")
	}

	// Get latest rate limits
	var rateLimitsJSON sql.NullString
	query = `
		SELECT rate_limits
		FROM usage_tracking
		WHERE session_id = ? AND rate_limits IS NOT NULL AND rate_limits != 'null'
		ORDER BY created_at DESC
		LIMIT 1
	`
	err = db.conn.QueryRow(query, sessionID).Scan(&rateLimitsJSON)
	if err != nil && err != sql.ErrNoRows {
		return totalInput, totalOutput, nil, serr.Wrap(err, "failed to get latest rate limits")
	}

	if rateLimitsJSON.Valid && rateLimitsJSON.String != "null" {
		latestRateLimits = &providers.RateLimitInfo{}
		if err := json.Unmarshal([]byte(rateLimitsJSON.String), latestRateLimits); err != nil {
			// Log error but don't fail the whole operation
			// logger.LogErr(err, "failed to unmarshal rate limits")
		}
	}

	return totalInput, totalOutput, latestRateLimits, nil
}

// GetDailyUsage gets usage statistics for today
func (db *DB) GetDailyUsage() (map[string]struct{ Input, Output int }, error) {
	query := `
		SELECT 
			model,
			COALESCE(SUM(input_tokens), 0) as total_input,
			COALESCE(SUM(output_tokens), 0) as total_output
		FROM usage_tracking
		WHERE DATE(created_at) = DATE('now')
		GROUP BY model
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get daily usage")
	}
	defer rows.Close()

	usage := make(map[string]struct{ Input, Output int })
	for rows.Next() {
		var model string
		var input, output int
		if err := rows.Scan(&model, &input, &output); err != nil {
			return nil, serr.Wrap(err, "failed to scan usage row")
		}
		usage[model] = struct{ Input, Output int }{Input: input, Output: output}
	}

	return usage, nil
}

// GetGlobalUsage gets total usage across all sessions
func (db *DB) GetGlobalUsage() (map[string]struct{ Input, Output int }, *providers.RateLimitInfo, error) {
	// Get total usage by model
	query := `
		SELECT 
			model,
			COALESCE(SUM(input_tokens), 0) as total_input,
			COALESCE(SUM(output_tokens), 0) as total_output
		FROM usage_tracking
		GROUP BY model
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, nil, serr.Wrap(err, "failed to get global usage")
	}
	defer rows.Close()

	usage := make(map[string]struct{ Input, Output int })
	for rows.Next() {
		var model string
		var input, output int
		if err := rows.Scan(&model, &input, &output); err != nil {
			return nil, nil, serr.Wrap(err, "failed to scan usage row")
		}
		usage[model] = struct{ Input, Output int }{Input: input, Output: output}
	}

	// Get latest rate limits
	var rateLimitsJSON sql.NullString
	query = `
		SELECT rate_limits
		FROM usage_tracking
		WHERE rate_limits IS NOT NULL AND rate_limits != 'null'
		ORDER BY created_at DESC
		LIMIT 1
	`
	err = db.conn.QueryRow(query).Scan(&rateLimitsJSON)
	if err != nil && err != sql.ErrNoRows {
		return usage, nil, serr.Wrap(err, "failed to get latest rate limits")
	}

	var latestRateLimits *providers.RateLimitInfo
	if rateLimitsJSON.Valid && rateLimitsJSON.String != "null" {
		latestRateLimits = &providers.RateLimitInfo{}
		if err := json.Unmarshal([]byte(rateLimitsJSON.String), latestRateLimits); err != nil {
			// Log error but don't fail
			latestRateLimits = nil
		}
	}

	return usage, latestRateLimits, nil
}
