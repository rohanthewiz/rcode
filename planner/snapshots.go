package planner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// SnapshotStore defines the interface for snapshot persistence
type SnapshotStore interface {
	SaveSnapshot(snapshot *FileSnapshot) error
	GetSnapshots(checkpointID string) ([]*FileSnapshot, error)
	GetSnapshotByHash(hash string) (*FileSnapshot, error)
}

// FileSnapshot represents a file snapshot for rollback
type FileSnapshot struct {
	ID           int       `json:"id"`
	SnapshotID   string    `json:"snapshot_id"`
	PlanID       string    `json:"plan_id"`
	CheckpointID string    `json:"checkpoint_id"`
	FilePath     string    `json:"file_path"`
	Content      string    `json:"content"`
	Hash         string    `json:"hash"`
	FileMode     int       `json:"file_mode,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// SnapshotManager manages file snapshots for rollback capabilities
type SnapshotManager struct {
	baseDir string
	store   SnapshotStore
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(store SnapshotStore) *SnapshotManager {
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".local", "share", "rcode", "snapshots")
	
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		logger.LogErr(err, "failed to create snapshots directory")
	}
	
	return &SnapshotManager{
		baseDir: baseDir,
		store:   store,
	}
}

// CreateSnapshot creates snapshots of specified files
func (sm *SnapshotManager) CreateSnapshot(planID, checkpointID string, files []string) error {
	logger.Info("Creating snapshots", "plan_id", planID, "checkpoint_id", checkpointID, "files", len(files))
	
	for _, file := range files {
		if err := sm.snapshotFile(planID, checkpointID, file); err != nil {
			// Log error but continue with other files
			logger.LogErr(err, "failed to snapshot file", "file", file)
		}
	}
	
	return nil
}

// snapshotFile creates a snapshot of a single file
func (sm *SnapshotManager) snapshotFile(planID, checkpointID, filePath string) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, skip it
			return nil
		}
		return serr.Wrap(err, "failed to read file")
	}
	
	// Get file info for permissions
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return serr.Wrap(err, "failed to stat file")
	}
	
	// Calculate content hash
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])
	
	// Check if we already have this content stored
	existingSnapshot, err := sm.store.GetSnapshotByHash(hashStr)
	if err == nil && existingSnapshot != nil {
		// Content already exists, just reference it
		logger.Debug("Content already exists in snapshots", "hash", hashStr[:8])
	} else {
		// Store content using content-addressed storage
		snapPath := filepath.Join(sm.baseDir, hashStr[:2], hashStr)
		snapDir := filepath.Dir(snapPath)
		
		if err := os.MkdirAll(snapDir, 0755); err != nil {
			return serr.Wrap(err, "failed to create snapshot directory")
		}
		
		if err := os.WriteFile(snapPath, content, 0644); err != nil {
			return serr.Wrap(err, "failed to write snapshot file")
		}
	}
	
	// Save snapshot metadata to database
	snapshot := &FileSnapshot{
		SnapshotID:   uuid.New().String(),
		PlanID:       planID,
		CheckpointID: checkpointID,
		FilePath:     filePath,
		Content:      string(content), // Store content in DB for quick access
		Hash:         hashStr,
		FileMode:     int(fileInfo.Mode().Perm()),
		CreatedAt:    time.Now(),
	}
	
	if err := sm.store.SaveSnapshot(snapshot); err != nil {
		return serr.Wrap(err, "failed to save snapshot metadata")
	}
	
	logger.Debug("Created snapshot", "file", filePath, "hash", hashStr[:8])
	return nil
}

// RestoreSnapshot restores files from a checkpoint
func (sm *SnapshotManager) RestoreSnapshot(checkpointID string) error {
	logger.Info("Restoring snapshot", "checkpoint_id", checkpointID)
	
	snapshots, err := sm.store.GetSnapshots(checkpointID)
	if err != nil {
		return serr.Wrap(err, "failed to get snapshots")
	}
	
	restoredCount := 0
	for _, snapshot := range snapshots {
		if err := sm.restoreFile(snapshot); err != nil {
			logger.LogErr(err, "failed to restore file", "file", snapshot.FilePath)
			return err
		}
		restoredCount++
	}
	
	logger.Info("Restored files from snapshot", "checkpoint_id", checkpointID, "files", restoredCount)
	return nil
}

// restoreFile restores a single file from snapshot
func (sm *SnapshotManager) restoreFile(snapshot *FileSnapshot) error {
	// Create backup of current file if it exists
	if _, err := os.Stat(snapshot.FilePath); err == nil {
		backupPath := snapshot.FilePath + ".backup." + time.Now().Format("20060102150405")
		if err := sm.copyFile(snapshot.FilePath, backupPath); err != nil {
			logger.LogErr(err, "failed to create backup", "file", snapshot.FilePath)
		}
	}
	
	// Ensure directory exists
	dir := filepath.Dir(snapshot.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return serr.Wrap(err, "failed to create directory")
	}
	
	// Restore file content
	fileMode := os.FileMode(0644)
	if snapshot.FileMode > 0 {
		fileMode = os.FileMode(snapshot.FileMode)
	}
	
	if err := os.WriteFile(snapshot.FilePath, []byte(snapshot.Content), fileMode); err != nil {
		return serr.Wrap(err, "failed to restore file")
	}
	
	logger.Debug("Restored file", "file", snapshot.FilePath, "mode", fileMode)
	return nil
}

// RestoreFile restores a specific file to a specific snapshot version
func (sm *SnapshotManager) RestoreFile(filePath, snapshotID string) error {
	logger.Info("Restoring specific file", "file", filePath, "snapshot_id", snapshotID)
	
	// Get snapshot by ID
	snapshots, err := sm.store.GetSnapshots(snapshotID)
	if err != nil {
		return serr.Wrap(err, "failed to get snapshot")
	}
	
	// Find the specific file snapshot
	for _, snapshot := range snapshots {
		if snapshot.FilePath == filePath {
			return sm.restoreFile(snapshot)
		}
	}
	
	return serr.New("snapshot not found for file")
}

// GetFileHistory returns the history of snapshots for a file
func (sm *SnapshotManager) GetFileHistory(planID, filePath string) ([]*FileSnapshot, error) {
	// This would require an additional DB query method
	// For now, return empty list
	return []*FileSnapshot{}, nil
}

// CleanupOldSnapshots removes snapshots older than the retention period
func (sm *SnapshotManager) CleanupOldSnapshots(retentionDays int) error {
	logger.Info("Cleaning up old snapshots", "retention_days", retentionDays)
	
	// Walk through snapshot directory
	deletedCount := 0
	err := filepath.Walk(sm.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check age
		age := time.Since(info.ModTime())
		if age > time.Duration(retentionDays)*24*time.Hour {
			if err := os.Remove(path); err != nil {
				logger.LogErr(err, "failed to remove old snapshot", "path", path)
			} else {
				deletedCount++
			}
		}
		
		return nil
	})
	
	if err != nil {
		return serr.Wrap(err, "failed to walk snapshot directory")
	}
	
	logger.Info("Cleaned up old snapshots", "deleted", deletedCount)
	return nil
}

// copyFile copies a file from src to dst
func (sm *SnapshotManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}
	
	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err == nil {
		err = os.Chmod(dst, sourceInfo.Mode())
	}
	
	return err
}

// GetSnapshotSize returns the total size of all snapshots
func (sm *SnapshotManager) GetSnapshotSize() (int64, error) {
	var totalSize int64
	
	err := filepath.Walk(sm.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	
	return totalSize, err
}

// VerifySnapshot verifies that a snapshot's content matches its hash
func (sm *SnapshotManager) VerifySnapshot(snapshot *FileSnapshot) error {
	// Calculate hash of stored content
	hash := sha256.Sum256([]byte(snapshot.Content))
	hashStr := hex.EncodeToString(hash[:])
	
	if hashStr != snapshot.Hash {
		return fmt.Errorf("snapshot verification failed: hash mismatch")
	}
	
	// Also check file on disk if it exists
	snapPath := filepath.Join(sm.baseDir, snapshot.Hash[:2], snapshot.Hash)
	if content, err := os.ReadFile(snapPath); err == nil {
		diskHash := sha256.Sum256(content)
		diskHashStr := hex.EncodeToString(diskHash[:])
		if diskHashStr != snapshot.Hash {
			return fmt.Errorf("disk snapshot verification failed: hash mismatch")
		}
	}
	
	return nil
}