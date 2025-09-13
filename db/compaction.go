package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
	"rcode/providers"
)

// CompactedMessage represents a compacted section of conversation
type CompactedMessage struct {
	ID                 int       `json:"id"`
	SessionID          string    `json:"session_id"`
	Summary            string    `json:"summary"`
	OriginalMessageIDs []int     `json:"original_message_ids"`
	StartMessageID     int       `json:"start_message_id"`
	EndMessageID       int       `json:"end_message_id"`
	TokenCountBefore   int       `json:"token_count_before"`
	TokenCountAfter    int       `json:"token_count_after"`
	CompactedAt        time.Time `json:"compacted_at"`
	Metadata           JSONMap   `json:"metadata,omitempty"`
}

// CompactionOptions represents options for compacting messages
type CompactionOptions struct {
	PreserveRecent       int    `json:"preserve_recent"`         // Number of recent messages to keep uncompacted
	PreserveInitial      int    `json:"preserve_initial"`        // Number of initial messages to keep uncompacted
	Strategy             string `json:"strategy"`                // "aggressive" or "conservative"
	MaxSummaryTokens     int    `json:"max_summary_tokens"`      // Maximum tokens for each summary
	MinMessagesToCompact int    `json:"min_messages_to_compact"` // Minimum messages required to trigger compaction
}

// DefaultCompactionOptions returns default compaction options
func DefaultCompactionOptions() CompactionOptions {
	return CompactionOptions{
		PreserveRecent:       10, // Keep last 10 messages
		PreserveInitial:      2,  // Keep first 2 messages (including context)
		Strategy:             "conservative",
		MaxSummaryTokens:     500,
		MinMessagesToCompact: 20, // Only compact if we have at least 20 messages
	}
}

// CompactSessionMessages compacts messages in a session to reduce token count
func (db *DB) CompactSessionMessages(sessionID string, opts CompactionOptions) (*CompactedMessage, error) {
	// Get all messages for the session
	messages, err := db.GetMessagesWithMetadata(sessionID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get messages")
	}

	// Check if we have enough messages to compact
	totalMessages := len(messages)
	if totalMessages < opts.MinMessagesToCompact {
		return nil, serr.New(fmt.Sprintf("not enough messages to compact (have %d, need %d)",
			totalMessages, opts.MinMessagesToCompact))
	}

	// Determine which messages to compact
	// We preserve initial and recent messages
	startIdx := opts.PreserveInitial
	endIdx := totalMessages - opts.PreserveRecent

	if startIdx >= endIdx {
		return nil, serr.New("no messages in compactable range")
	}

	// Get messages to compact
	messagesToCompact := messages[startIdx:endIdx]

	// Calculate token count before compaction (approximate)
	tokenCountBefore := 0
	var messageIDs []int
	for _, msg := range messagesToCompact {
		messageIDs = append(messageIDs, msg.ID)
		// Approximate token count (very rough estimate)
		contentStr := fmt.Sprintf("%v", msg.Content)
		tokenCountBefore += len(contentStr) / 4 // Rough approximation: 1 token ≈ 4 characters
	}

	// Generate summary of the compacted messages
	summary, metadata := generateSummary(messagesToCompact, opts)

	// Calculate token count after compaction
	tokenCountAfter := len(summary) / 4 // Rough approximation

	// Begin transaction
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, serr.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Archive original messages
	for _, msg := range messagesToCompact {
		err = archiveMessage(tx, msg, 0) // We'll update compaction_id later
		if err != nil {
			return nil, serr.Wrap(err, "failed to archive message")
		}
	}

	// Create compacted message record
	compactedMsg := &CompactedMessage{
		SessionID:          sessionID,
		Summary:            summary,
		OriginalMessageIDs: messageIDs,
		StartMessageID:     messagesToCompact[0].ID,
		EndMessageID:       messagesToCompact[len(messagesToCompact)-1].ID,
		TokenCountBefore:   tokenCountBefore,
		TokenCountAfter:    tokenCountAfter,
		CompactedAt:        time.Now(),
		Metadata:           metadata,
	}

	// Insert compacted message
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, serr.Wrap(err, "failed to marshal metadata")
	}

	// Convert message IDs to array literal for DuckDB
	idsArray := "["
	for i, id := range messageIDs {
		if i > 0 {
			idsArray += ", "
		}
		idsArray += fmt.Sprintf("%d", id)
	}
	idsArray += "]"

	query := `
		INSERT INTO compacted_messages 
		(session_id, summary, original_message_ids, start_message_id, end_message_id, 
		 token_count_before, token_count_after, metadata)
		VALUES (?, ?, ` + idsArray + `, ?, ?, ?, ?, ?::JSON)
	`

	result, err := tx.Exec(query, sessionID, summary,
		messagesToCompact[0].ID, messagesToCompact[len(messagesToCompact)-1].ID,
		tokenCountBefore, tokenCountAfter, string(metadataJSON))
	if err != nil {
		return nil, serr.Wrap(err, "failed to insert compacted message")
	}
	_ = result

	// Get the inserted ID
	var compactionID int
	err = tx.QueryRow("SELECT currval('compacted_messages_id_seq')").Scan(&compactionID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get compaction ID")
	}
	compactedMsg.ID = compactionID

	// Update archived messages with compaction_id
	_, err = tx.Exec(`
		UPDATE archived_messages 
		SET compaction_id = ? 
		WHERE session_id = ? AND id >= ? AND id <= ?`,
		compactionID, sessionID, messagesToCompact[0].ID, messagesToCompact[len(messagesToCompact)-1].ID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to update archived messages")
	}

	// Delete original messages from messages table
	_, err = tx.Exec(`
		DELETE FROM messages 
		WHERE session_id = ? AND id >= ? AND id <= ?`,
		sessionID, messagesToCompact[0].ID, messagesToCompact[len(messagesToCompact)-1].ID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to delete original messages")
	}

	// Update session metadata
	_, err = tx.Exec(`
		UPDATE sessions 
		SET last_compacted_at = CURRENT_TIMESTAMP,
		    compaction_metadata = ?::JSON
		WHERE id = ?`,
		string(metadataJSON), sessionID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to update session")
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, serr.Wrap(err, "failed to commit transaction")
	}

	logger.Info("Compacted session messages",
		"session_id", sessionID,
		"messages_compacted", len(messagesToCompact),
		"token_savings", tokenCountBefore-tokenCountAfter)

	return compactedMsg, nil
}

// generateSummary creates a summary of the messages to be compacted
func generateSummary(messages []*Message, opts CompactionOptions) (string, JSONMap) {
	var summaryParts []string
	metadata := make(JSONMap)

	// Track key information
	toolsUsed := make(map[string]int)
	filesModified := make(map[string]bool)
	errors := []string{}
	decisions := []string{}

	for _, msg := range messages {
		// Analyze message content for important information
		contentStr := fmt.Sprintf("%v", msg.Content)

		// Track tool usage
		if strings.Contains(contentStr, "tool_use") {
			// Simple extraction - could be enhanced
			if strings.Contains(contentStr, "read_file") {
				toolsUsed["read_file"]++
			}
			if strings.Contains(contentStr, "write_file") {
				toolsUsed["write_file"]++
			}
			if strings.Contains(contentStr, "edit_file") {
				toolsUsed["edit_file"]++
			}
		}

		// Track errors
		if strings.Contains(strings.ToLower(contentStr), "error") {
			// Extract error context (simplified)
			if len(contentStr) > 100 {
				errors = append(errors, contentStr[:100]+"...")
			} else {
				errors = append(errors, contentStr)
			}
		}

		// For user messages, capture key requests
		if msg.Role == "user" && len(contentStr) > 20 {
			if len(contentStr) > 200 {
				summaryParts = append(summaryParts, fmt.Sprintf("User: %s...", contentStr[:200]))
			} else {
				summaryParts = append(summaryParts, fmt.Sprintf("User: %s", contentStr))
			}
		}

		// For assistant messages with significant content
		if msg.Role == "assistant" {
			// Check for code blocks or important responses
			if strings.Contains(contentStr, "```") || len(contentStr) > 500 {
				if opts.Strategy == "aggressive" {
					summaryParts = append(summaryParts, "Assistant: [code/detailed response provided]")
				} else {
					// Conservative: keep more context
					excerpt := contentStr
					if len(excerpt) > 300 {
						excerpt = excerpt[:300] + "..."
					}
					summaryParts = append(summaryParts, fmt.Sprintf("Assistant: %s", excerpt))
				}
			}
		}
	}

	// Build metadata
	metadata["tools_used"] = toolsUsed
	metadata["files_modified"] = filesModified
	if len(errors) > 0 {
		metadata["errors"] = errors
	}
	if len(decisions) > 0 {
		metadata["decisions"] = decisions
	}
	metadata["message_count"] = len(messages)
	metadata["strategy"] = opts.Strategy

	// Create summary
	summary := fmt.Sprintf("=== Compacted Conversation (%d messages) ===\n", len(messages))
	summary += strings.Join(summaryParts, "\n---\n")

	// Add metadata summary
	if len(toolsUsed) > 0 {
		summary += "\n\nTools used in this section: "
		for tool, count := range toolsUsed {
			summary += fmt.Sprintf("%s(%d) ", tool, count)
		}
	}

	return summary, metadata
}

// archiveMessage archives a message before deletion
func archiveMessage(tx *sql.Tx, msg *Message, compactionID int) error {
	contentJSON, err := json.Marshal(msg.Content)
	if err != nil {
		return serr.Wrap(err, "failed to marshal content")
	}

	var usageJSON sql.NullString
	if msg.TokenUsage != nil {
		usage, err := json.Marshal(msg.TokenUsage)
		if err != nil {
			return serr.Wrap(err, "failed to marshal token usage")
		}
		usageJSON = sql.NullString{String: string(usage), Valid: true}
	}

	var model sql.NullString
	if msg.Model != "" {
		model = sql.NullString{String: msg.Model, Valid: true}
	}

	query := `
		INSERT INTO archived_messages 
		(id, session_id, role, content, created_at, model, token_usage, compaction_id)
		VALUES (?, ?, ?, ?::JSON, ?, ?, ?::JSON, ?)
	`

	_, err = tx.Exec(query, msg.ID, msg.SessionID, msg.Role, string(contentJSON),
		msg.CreatedAt, model, usageJSON, sql.NullInt64{Int64: int64(compactionID), Valid: compactionID > 0})

	return err
}

// GetCompactedMessages retrieves compacted messages for a session
func (db *DB) GetCompactedMessages(sessionID string) ([]*CompactedMessage, error) {
	query := `
		SELECT id, session_id, summary, 
		       list_aggregate(original_message_ids, 'string_agg', ',') as ids,
		       start_message_id, end_message_id,
		       token_count_before, token_count_after, 
		       compacted_at, metadata::VARCHAR
		FROM compacted_messages
		WHERE session_id = ?
		ORDER BY start_message_id ASC
	`

	rows, err := db.Query(query, sessionID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to query compacted messages")
	}
	defer rows.Close()

	var compactedMessages []*CompactedMessage
	for rows.Next() {
		var cm CompactedMessage
		var idsStr sql.NullString
		var metadataJSON sql.NullString

		err := rows.Scan(
			&cm.ID, &cm.SessionID, &cm.Summary,
			&idsStr, &cm.StartMessageID, &cm.EndMessageID,
			&cm.TokenCountBefore, &cm.TokenCountAfter,
			&cm.CompactedAt, &metadataJSON,
		)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan compacted message")
		}

		// Parse message IDs
		if idsStr.Valid && idsStr.String != "" {
			parts := strings.Split(idsStr.String, ",")
			for _, part := range parts {
				var id int
				fmt.Sscanf(part, "%d", &id)
				cm.OriginalMessageIDs = append(cm.OriginalMessageIDs, id)
			}
		}

		// Parse metadata
		if metadataJSON.Valid && metadataJSON.String != "" {
			cm.Metadata = make(JSONMap)
			json.Unmarshal([]byte(metadataJSON.String), &cm.Metadata)
		}

		compactedMessages = append(compactedMessages, &cm)
	}

	return compactedMessages, nil
}

// GetMessagesWithCompaction retrieves messages including compacted summaries
func (db *DB) GetMessagesWithCompaction(sessionID string) ([]providers.ChatMessage, error) {
	// Get regular messages
	regularMessages, err := db.GetMessagesWithMetadata(sessionID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get regular messages")
	}

	// Get compacted messages
	compactedMessages, err := db.GetCompactedMessages(sessionID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get compacted messages")
	}

	// If no compacted messages, return regular messages as-is
	if len(compactedMessages) == 0 {
		result := make([]providers.ChatMessage, len(regularMessages))
		for i, msg := range regularMessages {
			result[i] = providers.ChatMessage{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}
		return result, nil
	}

	// Merge regular and compacted messages in correct order
	var result []providers.ChatMessage
	compactedIdx := 0

	for _, msg := range regularMessages {
		// Check if we need to insert a compacted message before this one
		for compactedIdx < len(compactedMessages) &&
			compactedMessages[compactedIdx].EndMessageID < msg.ID {
			// Add compacted message as a system message
			result = append(result, providers.ChatMessage{
				Role:    "system",
				Content: compactedMessages[compactedIdx].Summary,
			})
			compactedIdx++
		}

		// Add the regular message
		result = append(result, providers.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add any remaining compacted messages at the end
	for compactedIdx < len(compactedMessages) {
		result = append(result, providers.ChatMessage{
			Role:    "system",
			Content: compactedMessages[compactedIdx].Summary,
		})
		compactedIdx++
	}

	return result, nil
}

// RestoreCompactedMessages restores archived messages from a compaction
func (db *DB) RestoreCompactedMessages(sessionID string, compactionID int) error {
	// Begin transaction
	tx, err := db.conn.Begin()
	if err != nil {
		return serr.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Get archived messages
	query := `
		SELECT id, session_id, role, content::VARCHAR, created_at, model, token_usage::VARCHAR
		FROM archived_messages
		WHERE session_id = ? AND compaction_id = ?
		ORDER BY id ASC
	`

	rows, err := tx.Query(query, sessionID, compactionID)
	if err != nil {
		return serr.Wrap(err, "failed to query archived messages")
	}
	defer rows.Close()

	// Restore each message
	for rows.Next() {
		var id int
		var sessionID, role, contentJSON string
		var createdAt time.Time
		var model, usageJSON sql.NullString

		err := rows.Scan(&id, &sessionID, &role, &contentJSON, &createdAt, &model, &usageJSON)
		if err != nil {
			return serr.Wrap(err, "failed to scan archived message")
		}

		// Insert back into messages table
		insertQuery := `
			INSERT INTO messages (id, session_id, role, content, created_at, model, token_usage)
			VALUES (?, ?, ?, ?::JSON, ?, ?, ?::JSON)
		`

		_, err = tx.Exec(insertQuery, id, sessionID, role, contentJSON, createdAt, model, usageJSON)
		if err != nil {
			return serr.Wrap(err, "failed to restore message")
		}
	}

	// Delete the compacted message record
	_, err = tx.Exec("DELETE FROM compacted_messages WHERE id = ?", compactionID)
	if err != nil {
		return serr.Wrap(err, "failed to delete compacted message")
	}

	// Delete archived messages
	_, err = tx.Exec("DELETE FROM archived_messages WHERE compaction_id = ?", compactionID)
	if err != nil {
		return serr.Wrap(err, "failed to delete archived messages")
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return serr.Wrap(err, "failed to commit transaction")
	}

	logger.Info("Restored compacted messages", "session_id", sessionID, "compaction_id", compactionID)
	return nil
}

// GetCompactionStats returns statistics about compaction for a session
func (db *DB) GetCompactionStats(sessionID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get session info
	var lastCompacted sql.NullTime
	var autoCompactEnabled bool
	var threshold int
	err := db.QueryRow(`
		SELECT last_compacted_at, auto_compact_enabled, compact_threshold
		FROM sessions WHERE id = ?`, sessionID).Scan(&lastCompacted, &autoCompactEnabled, &threshold)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get session info")
	}

	stats["auto_compact_enabled"] = autoCompactEnabled
	stats["compact_threshold"] = threshold
	if lastCompacted.Valid {
		stats["last_compacted_at"] = lastCompacted.Time
	}

	// Get compaction summary
	var compactionCount int
	var totalTokensSaved int
	err = db.QueryRow(`
		SELECT COUNT(*), 
		       COALESCE(SUM(token_count_before - token_count_after), 0)
		FROM compacted_messages
		WHERE session_id = ?`, sessionID).Scan(&compactionCount, &totalTokensSaved)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get compaction stats")
	}

	stats["compaction_count"] = compactionCount
	stats["total_tokens_saved"] = totalTokensSaved

	// Get current message count and estimated tokens
	messageCount, err := db.GetMessageCount(sessionID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get message count")
	}
	stats["current_message_count"] = messageCount

	return stats, nil
}
