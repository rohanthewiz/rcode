package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// PermissionType represents the type of permission
type PermissionType string

const (
	PermissionAllowed PermissionType = "allowed"
	PermissionDenied  PermissionType = "denied"
	PermissionAsk     PermissionType = "ask"
)

// PermissionScope represents the scope of a permission
type PermissionScope struct {
	Paths        []string               `json:"paths,omitempty"`
	MaxFileSize  int64                  `json:"max_file_size,omitempty"`
	AllowedCmds  []string               `json:"allowed_cmds,omitempty"`
	CustomRules  map[string]interface{} `json:"custom_rules,omitempty"`
}

// ToolPermission represents a tool permission in the database
type ToolPermission struct {
	ID             int             `json:"id"`
	SessionID      string          `json:"session_id"`
	ToolName       string          `json:"tool_name"`
	PermissionType PermissionType  `json:"permission_type"`
	GrantedAt      *time.Time      `json:"granted_at,omitempty"`
	ExpiresAt      *time.Time      `json:"expires_at,omitempty"`
	Scope          *PermissionScope `json:"scope,omitempty"`
}

// SetToolPermission sets or updates a tool permission for a session
func (db *DB) SetToolPermission(sessionID, toolName string, permType PermissionType, scope *PermissionScope, expiresIn time.Duration) error {
	var scopeJSON []byte
	var err error
	
	if scope != nil {
		scopeJSON, err = json.Marshal(scope)
		if err != nil {
			return serr.Wrap(err, "failed to marshal permission scope")
		}
	}

	var expiresAt *time.Time
	if expiresIn > 0 {
		expires := time.Now().Add(expiresIn)
		expiresAt = &expires
	}

	grantedAt := time.Now()

	// Use UPSERT to insert or update
	query := `
		INSERT INTO tool_permissions (session_id, tool_name, permission_type, granted_at, expires_at, scope)
		VALUES (?, ?, ?, ?, ?, ?::JSON)
		ON CONFLICT (session_id, tool_name) 
		DO UPDATE SET 
			permission_type = EXCLUDED.permission_type,
			granted_at = EXCLUDED.granted_at,
			expires_at = EXCLUDED.expires_at,
			scope = EXCLUDED.scope
	`

	_, err = db.Exec(query, sessionID, toolName, string(permType), grantedAt, expiresAt, string(scopeJSON))
	if err != nil {
		return serr.Wrap(err, "failed to set tool permission")
	}

	logger.Info("Set tool permission", "session_id", sessionID, "tool", toolName, "permission", permType)
	return nil
}

// GetToolPermission retrieves a tool permission for a session
func (db *DB) GetToolPermission(sessionID, toolName string) (*ToolPermission, error) {
	query := `
		SELECT id, session_id, tool_name, permission_type, granted_at, expires_at, scope
		FROM tool_permissions
		WHERE session_id = ? AND tool_name = ?
		  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
	`

	var perm ToolPermission
	var grantedAt sql.NullTime
	var expiresAt sql.NullTime
	var scopeJSON sql.NullString

	err := db.QueryRow(query, sessionID, toolName).Scan(
		&perm.ID,
		&perm.SessionID,
		&perm.ToolName,
		&perm.PermissionType,
		&grantedAt,
		&expiresAt,
		&scopeJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, serr.Wrap(err, "failed to get tool permission")
	}

	// Set nullable fields
	if grantedAt.Valid {
		perm.GrantedAt = &grantedAt.Time
	}
	if expiresAt.Valid {
		perm.ExpiresAt = &expiresAt.Time
	}

	// Parse scope if present
	if scopeJSON.Valid && scopeJSON.String != "" {
		var scope PermissionScope
		if err := json.Unmarshal([]byte(scopeJSON.String), &scope); err == nil {
			perm.Scope = &scope
		}
	}

	return &perm, nil
}

// GetSessionPermissions retrieves all permissions for a session
func (db *DB) GetSessionPermissions(sessionID string) ([]*ToolPermission, error) {
	query := `
		SELECT id, session_id, tool_name, permission_type, granted_at, expires_at, scope
		FROM tool_permissions
		WHERE session_id = ?
		  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		ORDER BY tool_name
	`

	rows, err := db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []*ToolPermission
	for rows.Next() {
		var perm ToolPermission
		var grantedAt sql.NullTime
		var expiresAt sql.NullTime
		var scopeJSON sql.NullString

		err := rows.Scan(
			&perm.ID,
			&perm.SessionID,
			&perm.ToolName,
			&perm.PermissionType,
			&grantedAt,
			&expiresAt,
			&scopeJSON,
		)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan permission row")
		}

		// Set nullable fields
		if grantedAt.Valid {
			perm.GrantedAt = &grantedAt.Time
		}
		if expiresAt.Valid {
			perm.ExpiresAt = &expiresAt.Time
		}

		// Parse scope if present
		if scopeJSON.Valid && scopeJSON.String != "" {
			var scope PermissionScope
			if err := json.Unmarshal([]byte(scopeJSON.String), &scope); err == nil {
				perm.Scope = &scope
			}
		}

		permissions = append(permissions, &perm)
	}

	return permissions, nil
}

// RevokeToolPermission revokes a tool permission
func (db *DB) RevokeToolPermission(sessionID, toolName string) error {
	result, err := db.Exec(
		"DELETE FROM tool_permissions WHERE session_id = ? AND tool_name = ?",
		sessionID, toolName,
	)
	if err != nil {
		return serr.Wrap(err, "failed to revoke tool permission")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return serr.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return serr.New("permission not found")
	}

	logger.Info("Revoked tool permission", "session_id", sessionID, "tool", toolName)
	return nil
}

// CleanupExpiredPermissions removes expired permissions
func (db *DB) CleanupExpiredPermissions() error {
	result, err := db.Exec(
		"DELETE FROM tool_permissions WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP",
	)
	if err != nil {
		return serr.Wrap(err, "failed to cleanup expired permissions")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return serr.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected > 0 {
		logger.Info("Cleaned up expired permissions", "count", rowsAffected)
	}

	return nil
}

// CheckToolPermission checks if a tool is allowed for a session
func (db *DB) CheckToolPermission(sessionID, toolName string) (PermissionType, *PermissionScope, error) {
	perm, err := db.GetToolPermission(sessionID, toolName)
	if err != nil {
		return PermissionAsk, nil, err
	}

	// If no permission found, default to ask
	if perm == nil {
		return PermissionAsk, nil, nil
	}

	return perm.PermissionType, perm.Scope, nil
}