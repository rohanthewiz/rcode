package db

import (
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// FileAccess represents a file access record
type FileAccess struct {
	ID         int64     `json:"id"`
	SessionID  string    `json:"session_id"`
	FilePath   string    `json:"file_path"`
	AccessedAt time.Time `json:"accessed_at"`
	AccessType string    `json:"access_type"` // open, edit, create, delete
}

// SessionFile represents a file currently open in a session
type SessionFile struct {
	SessionID    string     `json:"session_id"`
	FilePath     string     `json:"file_path"`
	OpenedAt     time.Time  `json:"opened_at"`
	LastViewedAt *time.Time `json:"last_viewed_at,omitempty"`
	IsActive     bool       `json:"is_active"`
}

// TrackFileAccess records a file access event
func (db *DB) TrackFileAccess(sessionID, filePath, accessType string) error {
	_, err := db.Exec(`
		INSERT INTO file_access (session_id, file_path, access_type)
		VALUES (?, ?, ?)
	`, sessionID, filePath, accessType)
	
	if err != nil {
		return serr.Wrap(err, "failed to track file access")
	}
	
	logger.Info("File access tracked", "session", sessionID, "file", filePath, "type", accessType)
	return nil
}

// OpenFileInSession marks a file as open in a session
func (db *DB) OpenFileInSession(sessionID, filePath string) error {
	now := time.Now()
	
	// First, check if the file is already in the session
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM session_files 
			WHERE session_id = ? AND file_path = ?
		)
	`, sessionID, filePath).Scan(&exists)
	
	if err != nil {
		return serr.Wrap(err, "failed to check file existence")
	}
	
	if exists {
		// Update existing record
		_, err = db.Exec(`
			UPDATE session_files 
			SET last_viewed_at = ?, is_active = TRUE
			WHERE session_id = ? AND file_path = ?
		`, now, sessionID, filePath)
	} else {
		// Insert new record
		_, err = db.Exec(`
			INSERT INTO session_files (session_id, file_path, opened_at, last_viewed_at, is_active)
			VALUES (?, ?, ?, ?, TRUE)
		`, sessionID, filePath, now, now)
	}
	
	if err != nil {
		return serr.Wrap(err, "failed to open file in session")
	}
	
	// Also track this as a file access
	return db.TrackFileAccess(sessionID, filePath, "open")
}

// CloseFileInSession marks a file as closed in a session
func (db *DB) CloseFileInSession(sessionID, filePath string) error {
	_, err := db.Exec(`
		UPDATE session_files 
		SET is_active = FALSE
		WHERE session_id = ? AND file_path = ?
	`, sessionID, filePath)
	
	if err != nil {
		return serr.Wrap(err, "failed to close file in session")
	}
	
	return nil
}

// GetSessionFiles returns all files currently open in a session
func (db *DB) GetSessionFiles(sessionID string, activeOnly bool) ([]SessionFile, error) {
	query := `
		SELECT session_id, file_path, opened_at, last_viewed_at, is_active
		FROM session_files
		WHERE session_id = ?
	`
	
	if activeOnly {
		query += " AND is_active = TRUE"
	}
	
	query += " ORDER BY last_viewed_at DESC NULLS LAST, opened_at DESC"
	
	rows, err := db.Query(query, sessionID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get session files")
	}
	defer rows.Close()
	
	var files []SessionFile
	for rows.Next() {
		var f SessionFile
		err := rows.Scan(&f.SessionID, &f.FilePath, &f.OpenedAt, &f.LastViewedAt, &f.IsActive)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan session file")
		}
		files = append(files, f)
	}
	
	return files, nil
}

// GetRecentFiles returns recently accessed files for a session
func (db *DB) GetRecentFiles(sessionID string, limit int) ([]FileAccess, error) {
	if limit <= 0 {
		limit = 20
	}
	
	rows, err := db.Query(`
		SELECT id, session_id, file_path, accessed_at, access_type
		FROM file_access
		WHERE session_id = ?
		ORDER BY accessed_at DESC
		LIMIT ?
	`, sessionID, limit)
	
	if err != nil {
		return nil, serr.Wrap(err, "failed to get recent files")
	}
	defer rows.Close()
	
	var files []FileAccess
	for rows.Next() {
		var f FileAccess
		err := rows.Scan(&f.ID, &f.SessionID, &f.FilePath, &f.AccessedAt, &f.AccessType)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan file access")
		}
		files = append(files, f)
	}
	
	return files, nil
}

// GetFileAccessHistory returns the access history for a specific file
func (db *DB) GetFileAccessHistory(filePath string, limit int) ([]FileAccess, error) {
	if limit <= 0 {
		limit = 50
	}
	
	rows, err := db.Query(`
		SELECT id, session_id, file_path, accessed_at, access_type
		FROM file_access
		WHERE file_path = ?
		ORDER BY accessed_at DESC
		LIMIT ?
	`, filePath, limit)
	
	if err != nil {
		return nil, serr.Wrap(err, "failed to get file access history")
	}
	defer rows.Close()
	
	var accesses []FileAccess
	for rows.Next() {
		var a FileAccess
		err := rows.Scan(&a.ID, &a.SessionID, &a.FilePath, &a.AccessedAt, &a.AccessType)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan file access")
		}
		accesses = append(accesses, a)
	}
	
	return accesses, nil
}