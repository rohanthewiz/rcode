package diff

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// DiffService manages file snapshots and diff generation for the diff visualization feature.
// It captures file states before modifications and generates diffs to show changes.
type DiffService struct {
	// snapshots stores file snapshots keyed by session:path
	snapshots map[string]*FileSnapshot
	mu        sync.RWMutex
}

// FileSnapshot represents a point-in-time snapshot of a file's content.
// Used for generating diffs between file states.
type FileSnapshot struct {
	SessionID string    `json:"sessionId"`
	Path      string    `json:"path"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Hash      string    `json:"hash"` // SHA256 hash of content
	ToolID    string    `json:"toolId,omitempty"` // ID of tool that created snapshot
}

// DiffResult contains the complete diff information between two file states.
// Includes both the raw content and parsed diff hunks for display.
type DiffResult struct {
	SessionID string     `json:"sessionId"`
	Path      string     `json:"path"`
	Before    string     `json:"before"`    // Original content
	After     string     `json:"after"`     // Modified content
	Hunks     []DiffHunk `json:"hunks"`     // Parsed diff hunks
	Stats     DiffStats  `json:"stats"`     // Change statistics
	Timestamp time.Time  `json:"timestamp"` // When diff was generated
}

// DiffHunk represents a contiguous section of changes in a diff.
// Contains line-by-line change information for display.
type DiffHunk struct {
	OldStart int        `json:"oldStart"` // Starting line in original file
	OldLines int        `json:"oldLines"` // Number of lines in original
	NewStart int        `json:"newStart"` // Starting line in new file
	NewLines int        `json:"newLines"` // Number of lines in new
	Lines    []DiffLine `json:"lines"`    // Individual line changes
}

// DiffLine represents a single line in a diff with its change type.
// Tracks both old and new line numbers for side-by-side display.
type DiffLine struct {
	Type    string `json:"type"`              // "add", "delete", "context"
	OldLine *int   `json:"oldLine,omitempty"` // Line number in original (nil for added lines)
	NewLine *int   `json:"newLine,omitempty"` // Line number in new (nil for deleted lines)
	Content string `json:"content"`           // Line content (without newline)
}

// DiffStats provides summary statistics for a diff.
// Used for quick overview of changes.
type DiffStats struct {
	Added    int `json:"added"`    // Lines added
	Deleted  int `json:"deleted"`  // Lines removed
	Modified int `json:"modified"` // Lines changed (counted as delete + add)
}

// NewDiffService creates a new diff service instance.
// Initializes the snapshot storage for tracking file states.
func NewDiffService() *DiffService {
	return &DiffService{
		snapshots: make(map[string]*FileSnapshot),
	}
}

// CreateSnapshot captures the current state of a file for later diff generation.
// Called before file modifications to preserve the original state.
func (ds *DiffService) CreateSnapshot(sessionID, path, content, toolID string) (*FileSnapshot, error) {
	// Calculate content hash for change detection
	hash := sha256.Sum256([]byte(content))
	hashStr := hex.EncodeToString(hash[:])

	snapshot := &FileSnapshot{
		SessionID: sessionID,
		Path:      path,
		Content:   content,
		Timestamp: time.Now(),
		Hash:      hashStr,
		ToolID:    toolID,
	}

	// Store snapshot with composite key
	key := fmt.Sprintf("%s:%s", sessionID, path)
	ds.mu.Lock()
	ds.snapshots[key] = snapshot
	ds.mu.Unlock()

	logger.Debug("Created file snapshot",
		"sessionId", sessionID,
		"path", path,
		"hash", hashStr[:8],
		"toolId", toolID,
	)

	return snapshot, nil
}

// GetSnapshot retrieves a previously captured snapshot.
// Returns nil if no snapshot exists for the given session and path.
func (ds *DiffService) GetSnapshot(sessionID, path string) *FileSnapshot {
	key := fmt.Sprintf("%s:%s", sessionID, path)
	ds.mu.RLock()
	snapshot := ds.snapshots[key]
	ds.mu.RUnlock()
	return snapshot
}

// GenerateDiff creates a diff between a snapshot and new content.
// Uses the diff algorithm to produce hunks and statistics.
func (ds *DiffService) GenerateDiff(sessionID, path, newContent string) (*DiffResult, error) {
	// Retrieve the original snapshot
	snapshot := ds.GetSnapshot(sessionID, path)
	if snapshot == nil {
		return nil, serr.New("no snapshot found for file",
			"sessionId", sessionID,
			"path", path,
		)
	}

	// Generate diff hunks using diff algorithm
	hunks, err := ds.computeDiff(snapshot.Content, newContent)
	if err != nil {
		return nil, serr.Wrap(err, "failed to compute diff")
	}

	// Calculate statistics
	stats := ds.calculateStats(hunks)

	result := &DiffResult{
		SessionID: sessionID,
		Path:      path,
		Before:    snapshot.Content,
		After:     newContent,
		Hunks:     hunks,
		Stats:     stats,
		Timestamp: time.Now(),
	}

	logger.Debug("Generated diff",
		"sessionId", sessionID,
		"path", path,
		"hunks", len(hunks),
		"added", stats.Added,
		"deleted", stats.Deleted,
	)

	return result, nil
}

// computeDiff generates diff hunks between two text strings.
// Uses our line-based diff algorithm with 3 lines of context by default.
func (ds *DiffService) computeDiff(before, after string) ([]DiffHunk, error) {
	algo := &diffAlgorithm{}
	return algo.ComputeLineDiff(before, after, 3) // 3 lines of context
}

// calculateStats computes diff statistics from hunks.
// Counts added, deleted, and modified lines.
func (ds *DiffService) calculateStats(hunks []DiffHunk) DiffStats {
	stats := DiffStats{}
	
	for _, hunk := range hunks {
		for _, line := range hunk.Lines {
			switch line.Type {
			case "add":
				stats.Added++
			case "delete":
				stats.Deleted++
			}
		}
	}
	
	// Modified lines are typically represented as delete + add
	// We'll refine this logic when implementing the actual diff algorithm
	return stats
}

// ClearSnapshot removes a snapshot from memory.
// Called after diff is persisted or no longer needed.
func (ds *DiffService) ClearSnapshot(sessionID, path string) {
	key := fmt.Sprintf("%s:%s", sessionID, path)
	ds.mu.Lock()
	delete(ds.snapshots, key)
	ds.mu.Unlock()
}

// ClearSessionSnapshots removes all snapshots for a session.
// Called when a session ends or is cleaned up.
func (ds *DiffService) ClearSessionSnapshots(sessionID string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	// Remove all snapshots belonging to the session
	for key := range ds.snapshots {
		if len(key) > len(sessionID) && key[:len(sessionID)] == sessionID {
			delete(ds.snapshots, key)
		}
	}
}

// GetSessionSnapshots returns all snapshots for a session.
// Useful for showing all modified files in a session.
func (ds *DiffService) GetSessionSnapshots(sessionID string) []*FileSnapshot {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	var snapshots []*FileSnapshot
	for key, snapshot := range ds.snapshots {
		if len(key) > len(sessionID) && key[:len(sessionID)] == sessionID {
			snapshots = append(snapshots, snapshot)
		}
	}
	
	return snapshots
}

// HasChanges checks if the new content differs from the snapshot.
// Uses hash comparison for efficiency.
func (ds *DiffService) HasChanges(sessionID, path, newContent string) bool {
	snapshot := ds.GetSnapshot(sessionID, path)
	if snapshot == nil {
		return true // No snapshot means it's a new file
	}
	
	// Compare hashes to detect changes
	hash := sha256.Sum256([]byte(newContent))
	newHash := hex.EncodeToString(hash[:])
	return snapshot.Hash != newHash
}