package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// DiffSnapshot represents a stored file snapshot in the database
type DiffSnapshot struct {
	ID              int64     `json:"id"`
	SessionID       string    `json:"sessionId"`
	FilePath        string    `json:"filePath"`
	Content         string    `json:"content"`
	Hash            string    `json:"hash"`
	CreatedAt       time.Time `json:"createdAt"`
	ToolExecutionID string    `json:"toolExecutionId,omitempty"`
	ToolName        string    `json:"toolName,omitempty"`
}

// Diff represents a stored diff in the database
type Diff struct {
	ID               int64           `json:"id"`
	SessionID        string          `json:"sessionId"`
	FilePath         string          `json:"filePath"`
	BeforeSnapshotID *int64          `json:"beforeSnapshotId,omitempty"`
	AfterSnapshotID  *int64          `json:"afterSnapshotId,omitempty"`
	DiffData         json.RawMessage `json:"diffData"`
	CreatedAt        time.Time       `json:"createdAt"`
	ToolExecutionID  string          `json:"toolExecutionId,omitempty"`
	IsApplied        bool            `json:"isApplied"`
}

// DiffView represents a record of viewing a diff
type DiffView struct {
	SessionID string    `json:"sessionId"`
	DiffID    int64     `json:"diffId"`
	ViewedAt  time.Time `json:"viewedAt"`
	ViewMode  string    `json:"viewMode"`
}

// DiffPreferences represents user preferences for diff viewing
type DiffPreferences struct {
	UserID           string    `json:"userId"`
	DefaultMode      string    `json:"defaultMode"`
	ContextLines     int       `json:"contextLines"`
	WordWrap         bool      `json:"wordWrap"`
	SyntaxHighlight  bool      `json:"syntaxHighlight"`
	ShowLineNumbers  bool      `json:"showLineNumbers"`
	Theme            string    `json:"theme"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// SaveDiffSnapshot stores a file snapshot in the database.
// Returns the ID of the created snapshot.
func (db *DB) SaveDiffSnapshot(snapshot *DiffSnapshot) (int64, error) {
	query := `
		INSERT INTO diff_snapshots (session_id, file_path, content, hash, created_at, tool_execution_id, tool_name)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`

	var id int64
	err := db.QueryRow(query,
		snapshot.SessionID,
		snapshot.FilePath,
		snapshot.Content,
		snapshot.Hash,
		snapshot.CreatedAt,
		nullableString(snapshot.ToolExecutionID),
		nullableString(snapshot.ToolName),
	).Scan(&id)

	if err != nil {
		return 0, serr.Wrap(err, "failed to save diff snapshot")
	}

	logger.Debug("Saved diff snapshot",
		"id", id,
		"sessionId", snapshot.SessionID,
		"filePath", snapshot.FilePath,
		"hash", snapshot.Hash[:8],
	)

	return id, nil
}

// GetDiffSnapshot retrieves a snapshot by ID.
func (db *DB) GetDiffSnapshot(id int64) (*DiffSnapshot, error) {
	query := `
		SELECT id, session_id, file_path, content, hash, created_at, tool_execution_id, tool_name
		FROM diff_snapshots
		WHERE id = ?
	`

	var snapshot DiffSnapshot
	var toolExecutionID, toolName sql.NullString

	err := db.QueryRow(query, id).Scan(
		&snapshot.ID,
		&snapshot.SessionID,
		&snapshot.FilePath,
		&snapshot.Content,
		&snapshot.Hash,
		&snapshot.CreatedAt,
		&toolExecutionID,
		&toolName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, serr.Wrap(err, "failed to get diff snapshot")
	}

	snapshot.ToolExecutionID = toolExecutionID.String
	snapshot.ToolName = toolName.String

	return &snapshot, nil
}

// SaveDiff stores a diff in the database.
// Returns the ID of the created diff.
func (db *DB) SaveDiff(diff *Diff) (int64, error) {
	query := `
		INSERT INTO diffs (session_id, file_path, before_snapshot_id, after_snapshot_id, diff_data, created_at, tool_execution_id, is_applied)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`

	var id int64
	err := db.QueryRow(query,
		diff.SessionID,
		diff.FilePath,
		nullableInt64(diff.BeforeSnapshotID),
		nullableInt64(diff.AfterSnapshotID),
		diff.DiffData,
		diff.CreatedAt,
		nullableString(diff.ToolExecutionID),
		diff.IsApplied,
	).Scan(&id)

	if err != nil {
		return 0, serr.Wrap(err, "failed to save diff")
	}

	logger.Debug("Saved diff",
		"id", id,
		"sessionId", diff.SessionID,
		"filePath", diff.FilePath,
		"beforeSnapshot", diff.BeforeSnapshotID,
		"afterSnapshot", diff.AfterSnapshotID,
	)

	return id, nil
}

// GetDiff retrieves a diff by ID.
func (db *DB) GetDiff(id int64) (*Diff, error) {
	query := `
		SELECT id, session_id, file_path, before_snapshot_id, after_snapshot_id, diff_data, created_at, tool_execution_id, is_applied
		FROM diffs
		WHERE id = ?
	`

	var diff Diff
	var beforeSnapshotID, afterSnapshotID sql.NullInt64
	var toolExecutionID sql.NullString

	err := db.QueryRow(query, id).Scan(
		&diff.ID,
		&diff.SessionID,
		&diff.FilePath,
		&beforeSnapshotID,
		&afterSnapshotID,
		&diff.DiffData,
		&diff.CreatedAt,
		&toolExecutionID,
		&diff.IsApplied,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, serr.Wrap(err, "failed to get diff")
	}

	if beforeSnapshotID.Valid {
		diff.BeforeSnapshotID = &beforeSnapshotID.Int64
	}
	if afterSnapshotID.Valid {
		diff.AfterSnapshotID = &afterSnapshotID.Int64
	}
	diff.ToolExecutionID = toolExecutionID.String

	return &diff, nil
}

// GetSessionDiffs retrieves all diffs for a session.
// Ordered by creation time descending (newest first).
func (db *DB) GetSessionDiffs(sessionID string) ([]*Diff, error) {
	query := `
		SELECT id, session_id, file_path, before_snapshot_id, after_snapshot_id, diff_data, created_at, tool_execution_id, is_applied
		FROM diffs
		WHERE session_id = ?
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, sessionID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get session diffs")
	}
	defer rows.Close()

	var diffs []*Diff
	for rows.Next() {
		var diff Diff
		var beforeSnapshotID, afterSnapshotID sql.NullInt64
		var toolExecutionID sql.NullString

		err := rows.Scan(
			&diff.ID,
			&diff.SessionID,
			&diff.FilePath,
			&beforeSnapshotID,
			&afterSnapshotID,
			&diff.DiffData,
			&diff.CreatedAt,
			&toolExecutionID,
			&diff.IsApplied,
		)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan diff")
		}

		if beforeSnapshotID.Valid {
			diff.BeforeSnapshotID = &beforeSnapshotID.Int64
		}
		if afterSnapshotID.Valid {
			diff.AfterSnapshotID = &afterSnapshotID.Int64
		}
		diff.ToolExecutionID = toolExecutionID.String

		diffs = append(diffs, &diff)
	}

	return diffs, nil
}

// GetFileDiffs retrieves all diffs for a specific file in a session.
// Ordered by creation time descending (newest first).
func (db *DB) GetFileDiffs(sessionID, filePath string) ([]*Diff, error) {
	query := `
		SELECT id, session_id, file_path, before_snapshot_id, after_snapshot_id, diff_data, created_at, tool_execution_id, is_applied
		FROM diffs
		WHERE session_id = ? AND file_path = ?
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, sessionID, filePath)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get file diffs")
	}
	defer rows.Close()

	var diffs []*Diff
	for rows.Next() {
		var diff Diff
		var beforeSnapshotID, afterSnapshotID sql.NullInt64
		var toolExecutionID sql.NullString

		err := rows.Scan(
			&diff.ID,
			&diff.SessionID,
			&diff.FilePath,
			&beforeSnapshotID,
			&afterSnapshotID,
			&diff.DiffData,
			&diff.CreatedAt,
			&toolExecutionID,
			&diff.IsApplied,
		)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan diff")
		}

		if beforeSnapshotID.Valid {
			diff.BeforeSnapshotID = &beforeSnapshotID.Int64
		}
		if afterSnapshotID.Valid {
			diff.AfterSnapshotID = &afterSnapshotID.Int64
		}
		diff.ToolExecutionID = toolExecutionID.String

		diffs = append(diffs, &diff)
	}

	return diffs, nil
}

// MarkDiffViewed records that a diff has been viewed.
func (db *DB) MarkDiffViewed(sessionID string, diffID int64, viewMode string) error {
	query := `
		INSERT INTO diff_views (session_id, diff_id, viewed_at, view_mode)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (session_id, diff_id) DO UPDATE
		SET viewed_at = ?, view_mode = ?
	`

	now := time.Now()
	_, err := db.Exec(query, sessionID, diffID, now, viewMode, now, viewMode)
	if err != nil {
		return serr.Wrap(err, "failed to mark diff as viewed")
	}

	return nil
}

// GetDiffPreferences retrieves user preferences for diff viewing.
// Returns default preferences if none exist.
func (db *DB) GetDiffPreferences(userID string) (*DiffPreferences, error) {
	query := `
		SELECT user_id, default_mode, context_lines, word_wrap, syntax_highlight, show_line_numbers, theme, updated_at
		FROM diff_preferences
		WHERE user_id = ?
	`

	var prefs DiffPreferences
	err := db.QueryRow(query, userID).Scan(
		&prefs.UserID,
		&prefs.DefaultMode,
		&prefs.ContextLines,
		&prefs.WordWrap,
		&prefs.SyntaxHighlight,
		&prefs.ShowLineNumbers,
		&prefs.Theme,
		&prefs.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default preferences
			return &DiffPreferences{
				UserID:          userID,
				DefaultMode:     "side-by-side",
				ContextLines:    3,
				WordWrap:        false,
				SyntaxHighlight: true,
				ShowLineNumbers: true,
				Theme:           "dark",
				UpdatedAt:       time.Now(),
			}, nil
		}
		return nil, serr.Wrap(err, "failed to get diff preferences")
	}

	return &prefs, nil
}

// SaveDiffPreferences stores user preferences for diff viewing.
func (db *DB) SaveDiffPreferences(prefs *DiffPreferences) error {
	query := `
		INSERT INTO diff_preferences (user_id, default_mode, context_lines, word_wrap, syntax_highlight, show_line_numbers, theme, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE
		SET default_mode = ?, context_lines = ?, word_wrap = ?, syntax_highlight = ?, show_line_numbers = ?, theme = ?, updated_at = ?
	`

	now := time.Now()
	_, err := db.Exec(query,
		prefs.UserID,
		prefs.DefaultMode,
		prefs.ContextLines,
		prefs.WordWrap,
		prefs.SyntaxHighlight,
		prefs.ShowLineNumbers,
		prefs.Theme,
		now,
		prefs.DefaultMode,
		prefs.ContextLines,
		prefs.WordWrap,
		prefs.SyntaxHighlight,
		prefs.ShowLineNumbers,
		prefs.Theme,
		now,
	)

	if err != nil {
		return serr.Wrap(err, "failed to save diff preferences")
	}

	return nil
}

// nullableString converts an empty string to sql.NullString.
func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullableInt64 converts a nil *int64 to sql.NullInt64.
func nullableInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}