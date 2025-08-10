package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
	"rcode/providers"
)

// Message represents a chat message in the database
type Message struct {
	ID         int              `json:"id"`
	SessionID  string           `json:"session_id"`
	Role       string           `json:"role"`
	Content    interface{}      `json:"content"`
	CreatedAt  time.Time        `json:"created_at"`
	Model      string           `json:"model,omitempty"`
	TokenUsage *providers.Usage `json:"token_usage,omitempty"`
}

// AddMessageWithID adds a message to a session and returns the message ID
func (db *DB) AddMessageWithID(sessionID string, msg providers.ChatMessage, model string, usage *providers.Usage) (*int, error) {
	// Convert content to JSON
	contentJSON, err := json.Marshal(msg.Content)
	if err != nil {
		return nil, serr.Wrap(err, "failed to marshal message content")
	}

	// Convert token usage to JSON if present
	var usageJSONStr string
	if usage != nil {
		usageJSON, err := json.Marshal(usage)
		if err != nil {
			return nil, serr.Wrap(err, "failed to marshal token usage")
		}
		usageJSONStr = string(usageJSON)
	} else {
		usageJSONStr = "null"
	}

	// Handle empty model
	if model == "" {
		model = "null"
	}

	query := `
		INSERT INTO messages (session_id, role, content, model, token_usage, created_at)
		VALUES (?, ?, ?::JSON, NULLIF(?, 'null'), ?::JSON, CURRENT_TIMESTAMP)
	`

	result, err := db.Exec(query, sessionID, msg.Role, string(contentJSON), model, usageJSONStr)
	if err != nil {
		return nil, serr.Wrap(err, "failed to add message")
	}
	_ = result // Suppress unused variable warning

	// Get the last insert ID using DuckDB's method
	var messageID int
	err = db.QueryRow("SELECT currval('messages_id_seq')").Scan(&messageID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get message ID")
	}

	// Update session's updated_at timestamp
	_, err = db.Exec("UPDATE sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", sessionID)
	if err != nil {
		logger.LogErr(err, "failed to update session timestamp")
	}

	logger.Debug("Added message to session", "session_id", sessionID, "role", msg.Role, "message_id", messageID)
	return &messageID, nil
}

// AddMessage adds a message to a session (wrapper for backward compatibility)
func (db *DB) AddMessage(sessionID string, msg providers.ChatMessage, model string, usage *providers.Usage) error {
	_, err := db.AddMessageWithID(sessionID, msg, model, usage)
	return err
}

// GetMessages retrieves all messages for a session
func (db *DB) GetMessages(sessionID string) ([]providers.ChatMessage, error) {
	query := `
		SELECT role, content::VARCHAR
		FROM messages
		WHERE session_id = ?
		ORDER BY created_at ASC
	`

	rows, err := db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []providers.ChatMessage
	for rows.Next() {
		var role string
		var contentJSON string

		err := rows.Scan(&role, &contentJSON)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan message row")
		}

		// Parse content based on type
		var content interface{}

		// Try to parse as JSON first
		var jsonContent interface{}
		if err := json.Unmarshal([]byte(contentJSON), &jsonContent); err == nil {
			content = jsonContent
		} else {
			// If not valid JSON, treat as plain string
			content = contentJSON
		}

		messages = append(messages, providers.ChatMessage{
			Role:    role,
			Content: content,
		})
	}

	return messages, nil
}

// GetMessagesWithMetadata retrieves messages with full metadata
func (db *DB) GetMessagesWithMetadata(sessionID string) ([]*Message, error) {
	query := `
		SELECT id, session_id, role, content::VARCHAR, created_at, model, token_usage::VARCHAR
		FROM messages
		WHERE session_id = ?
		ORDER BY created_at ASC
	`

	rows, err := db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		var contentJSON string
		var model sql.NullString
		var usageJSON sql.NullString

		err := rows.Scan(
			&msg.ID,
			&msg.SessionID,
			&msg.Role,
			&contentJSON,
			&msg.CreatedAt,
			&model,
			&usageJSON,
		)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan message row")
		}

		// Parse content
		var content interface{}
		if err := json.Unmarshal([]byte(contentJSON), &content); err == nil {
			msg.Content = content
		} else {
			msg.Content = contentJSON
		}

		// Set model if present
		if model.Valid {
			msg.Model = model.String
		}

		// Parse token usage if present
		if usageJSON.Valid && usageJSON.String != "" {
			var usage providers.Usage
			if err := json.Unmarshal([]byte(usageJSON.String), &usage); err == nil {
				msg.TokenUsage = &usage
			}
		}

		messages = append(messages, &msg)
	}

	return messages, nil
}

// DeleteMessagesBySession deletes all messages for a session
func (db *DB) DeleteMessagesBySession(sessionID string) error {
	_, err := db.Exec("DELETE FROM messages WHERE session_id = ?", sessionID)
	if err != nil {
		return serr.Wrap(err, "failed to delete messages")
	}

	logger.Debug("Deleted messages for session", "session_id", sessionID)
	return nil
}

// GetMessageCount returns the number of messages in a session
func (db *DB) GetMessageCount(sessionID string) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM messages WHERE session_id = ?", sessionID).Scan(&count)
	if err != nil {
		return 0, serr.Wrap(err, "failed to get message count")
	}
	return count, nil
}

// GetSessionStats returns statistics for a session
func (db *DB) GetSessionStats(sessionID string) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as message_count,
			COUNT(DISTINCT model) as models_used,
			SUM(CASE WHEN token_usage IS NOT NULL THEN 
				(token_usage::JSON->>'input_tokens')::INT + 
				(token_usage::JSON->>'output_tokens')::INT 
			ELSE 0 END) as total_tokens
		FROM messages
		WHERE session_id = ?
	`

	var messageCount int
	var modelsUsed int
	var totalTokens sql.NullInt64

	err := db.QueryRow(query, sessionID).Scan(&messageCount, &modelsUsed, &totalTokens)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get session stats")
	}

	stats := map[string]interface{}{
		"message_count": messageCount,
		"models_used":   modelsUsed,
		"total_tokens":  0,
	}

	if totalTokens.Valid {
		stats["total_tokens"] = totalTokens.Int64
	}

	return stats, nil
}
