package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// Session represents a chat session in the database
type Session struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	InitialPrompts   []string   `json:"initial_prompts"`
	ModelPreference  string     `json:"model_preference,omitempty"`
	Metadata         JSONMap    `json:"metadata,omitempty"`
}

// JSONMap is a helper type for JSON columns
type JSONMap map[string]interface{}

// Scan implements sql.Scanner interface for JSONMap
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(JSONMap)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

// Value implements driver.Valuer interface for JSONMap
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// SessionOptions represents options for creating a session
type SessionOptions struct {
	Title            string
	InitialPrompts   []string
	ModelPreference  string
	Metadata         JSONMap
}

// CreateSession creates a new session in the database
func (db *DB) CreateSession(opts SessionOptions) (*Session, error) {
	now := time.Now()
	id := fmt.Sprintf("session-%d", now.Unix())

	// Set default title if not provided
	if opts.Title == "" {
		opts.Title = "New Chat"
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(opts.Metadata)
	if err != nil {
		return nil, serr.Wrap(err, "failed to marshal metadata")
	}

	// Build array literal for DuckDB
	var promptsArray string
	if len(opts.InitialPrompts) == 0 {
		promptsArray = "[]"
	} else {
		quotedPrompts := make([]string, len(opts.InitialPrompts))
		for i, p := range opts.InitialPrompts {
			// Escape single quotes and wrap in single quotes
			escaped := strings.ReplaceAll(p, "'", "''")
			quotedPrompts[i] = "'" + escaped + "'"
		}
		promptsArray = "[" + strings.Join(quotedPrompts, ", ") + "]"
	}

	// Use direct array literal
	query := `
		INSERT INTO sessions (id, title, created_at, updated_at, initial_prompts, model_preference, metadata)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ` + promptsArray + `, ?, ?::JSON)
	`

	_, err = db.Exec(query, id, opts.Title, opts.ModelPreference, string(metadataJSON))
	if err != nil {
		return nil, serr.Wrap(err, "failed to create session")
	}

	session := &Session{
		ID:              id,
		Title:           opts.Title,
		CreatedAt:       now,
		UpdatedAt:       now,
		InitialPrompts:  opts.InitialPrompts,
		ModelPreference: opts.ModelPreference,
		Metadata:        opts.Metadata,
	}

	logger.Info("Created session", "id", id, "title", opts.Title)
	return session, nil
}

// GetSession retrieves a session by ID
func (db *DB) GetSession(id string) (*Session, error) {
	query := `
		SELECT id, title, created_at, updated_at, 
		       list_aggregate(initial_prompts, 'string_agg', '|||') as prompts,
		       model_preference, metadata
		FROM sessions
		WHERE id = ?
	`

	var session Session
	var promptsStr sql.NullString
	var modelPref sql.NullString
	var metadataJSON sql.NullString

	err := db.QueryRow(query, id).Scan(
		&session.ID,
		&session.Title,
		&session.CreatedAt,
		&session.UpdatedAt,
		&promptsStr,
		&modelPref,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, serr.Wrap(err, "failed to get session")
	}

	// Parse prompts
	if promptsStr.Valid && promptsStr.String != "" {
		session.InitialPrompts = strings.Split(promptsStr.String, "|||")
	}

	// Set model preference
	if modelPref.Valid {
		session.ModelPreference = modelPref.String
	}

	// Parse metadata
	if metadataJSON.Valid && metadataJSON.String != "" {
		session.Metadata = make(JSONMap)
		if err := json.Unmarshal([]byte(metadataJSON.String), &session.Metadata); err != nil {
			logger.LogErr(err, "failed to parse session metadata")
		}
	}

	return &session, nil
}

// ListSessions retrieves all sessions
func (db *DB) ListSessions() ([]*Session, error) {
	query := `
		SELECT id, title, created_at, updated_at,
		       list_aggregate(initial_prompts, 'string_agg', '|||') as prompts,
		       model_preference, metadata
		FROM sessions
		ORDER BY updated_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		var promptsStr sql.NullString
		var modelPref sql.NullString
		var metadataJSON sql.NullString

		err := rows.Scan(
			&session.ID,
			&session.Title,
			&session.CreatedAt,
			&session.UpdatedAt,
			&promptsStr,
			&modelPref,
			&metadataJSON,
		)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan session row")
		}

		// Parse prompts
		if promptsStr.Valid && promptsStr.String != "" {
			session.InitialPrompts = strings.Split(promptsStr.String, "|||")
		}

		// Set model preference
		if modelPref.Valid {
			session.ModelPreference = modelPref.String
		}

		// Parse metadata
		if metadataJSON.Valid && metadataJSON.String != "" {
			session.Metadata = make(JSONMap)
			if err := json.Unmarshal([]byte(metadataJSON.String), &session.Metadata); err != nil {
				logger.LogErr(err, "failed to parse session metadata")
			}
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// UpdateSession updates a session's title and/or metadata
func (db *DB) UpdateSession(id string, title string, metadata JSONMap) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return serr.Wrap(err, "failed to marshal metadata")
	}

	query := `
		UPDATE sessions 
		SET title = ?, metadata = ?::JSON, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := db.Exec(query, title, string(metadataJSON), id)
	if err != nil {
		return serr.Wrap(err, "failed to update session")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return serr.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return serr.New("session not found")
	}

	logger.Info("Updated session", "id", id, "title", title)
	return nil
}

// DeleteSession deletes a session and all its messages
func (db *DB) DeleteSession(id string) error {
	result, err := db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return serr.Wrap(err, "failed to delete session")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return serr.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return serr.New("session not found")
	}

	logger.Info("Deleted session", "id", id)
	return nil
}

// SearchSessions searches sessions by content
func (db *DB) SearchSessions(searchTerm string) ([]*Session, error) {
	// Search in session titles and message content
	query := `
		SELECT DISTINCT s.id, s.title, s.created_at, s.updated_at,
		       list_aggregate(s.initial_prompts, 'string_agg', '|||') as prompts,
		       s.model_preference, s.metadata
		FROM sessions s
		LEFT JOIN messages m ON s.id = m.session_id
		WHERE s.title ILIKE ? 
		   OR m.content::TEXT ILIKE ?
		   OR list_contains(s.initial_prompts, ?)
		ORDER BY s.updated_at DESC
	`

	searchPattern := "%" + searchTerm + "%"
	rows, err := db.Query(query, searchPattern, searchPattern, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		var promptsStr sql.NullString
		var modelPref sql.NullString
		var metadataJSON sql.NullString

		err := rows.Scan(
			&session.ID,
			&session.Title,
			&session.CreatedAt,
			&session.UpdatedAt,
			&promptsStr,
			&modelPref,
			&metadataJSON,
		)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan session row")
		}

		// Parse prompts
		if promptsStr.Valid && promptsStr.String != "" {
			session.InitialPrompts = strings.Split(promptsStr.String, "|||")
		}

		// Set model preference
		if modelPref.Valid {
			session.ModelPreference = modelPref.String
		}

		// Parse metadata
		if metadataJSON.Valid && metadataJSON.String != "" {
			session.Metadata = make(JSONMap)
			if err := json.Unmarshal([]byte(metadataJSON.String), &session.Metadata); err != nil {
				logger.LogErr(err, "failed to parse session metadata")
			}
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// Helper function to quote strings for DuckDB array
func quoteStrings(strs []string) []string {
	quoted := make([]string, len(strs))
	for i, s := range strs {
		// Escape single quotes and wrap in single quotes
		escaped := strings.ReplaceAll(s, "'", "''")
		quoted[i] = "'" + escaped + "'"
	}
	return quoted
}