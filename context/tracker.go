package context

import (
	"sync"
	"time"
)

// ChangeTracker tracks file changes during a session
type ChangeTracker struct {
	mu       sync.RWMutex
	changes  map[string][]FileChange // path -> changes
	sessionStart time.Time
	stats    ChangeStats
}

// ChangeStats contains statistics about file changes
type ChangeStats struct {
	TotalChanges   int
	FileCount      int
	CreateCount    int
	ModifyCount    int
	DeleteCount    int
	RenameCount    int
	LastChangeTime time.Time
}

// NewChangeTracker creates a new change tracker
func NewChangeTracker() *ChangeTracker {
	return &ChangeTracker{
		changes:      make(map[string][]FileChange),
		sessionStart: time.Now(),
	}
}

// Track records a file change
func (ct *ChangeTracker) Track(change FileChange) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Initialize slice if needed
	if _, exists := ct.changes[change.Path]; !exists {
		ct.changes[change.Path] = make([]FileChange, 0)
		ct.stats.FileCount++
	}

	// Add change
	ct.changes[change.Path] = append(ct.changes[change.Path], change)
	ct.stats.TotalChanges++
	ct.stats.LastChangeTime = change.Timestamp

	// Update type-specific stats
	switch change.Type {
	case ChangeTypeCreate:
		ct.stats.CreateCount++
	case ChangeTypeModify:
		ct.stats.ModifyCount++
	case ChangeTypeDelete:
		ct.stats.DeleteCount++
		// Remove from changes map on delete
		delete(ct.changes, change.Path)
		ct.stats.FileCount--
	case ChangeTypeRename:
		ct.stats.RenameCount++
		// Move changes from old path to new path
		if change.OldPath != "" && change.OldPath != change.Path {
			if oldChanges, exists := ct.changes[change.OldPath]; exists {
				ct.changes[change.Path] = append(oldChanges, change)
				delete(ct.changes, change.OldPath)
			}
		}
	}
}

// GetChanges returns all changes for a specific file
func (ct *ChangeTracker) GetChanges(path string) []FileChange {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	if changes, exists := ct.changes[path]; exists {
		// Return a copy
		result := make([]FileChange, len(changes))
		copy(result, changes)
		return result
	}
	return nil
}

// GetAllChanges returns all tracked changes
func (ct *ChangeTracker) GetAllChanges() map[string][]FileChange {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	// Return a copy
	result := make(map[string][]FileChange)
	for path, changes := range ct.changes {
		changesCopy := make([]FileChange, len(changes))
		copy(changesCopy, changes)
		result[path] = changesCopy
	}
	return result
}

// GetRecentChanges returns changes made in the last duration
func (ct *ChangeTracker) GetRecentChanges(duration time.Duration) []FileChange {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	result := make([]FileChange, 0)

	for _, changes := range ct.changes {
		for _, change := range changes {
			if change.Timestamp.After(cutoff) {
				result = append(result, change)
			}
		}
	}

	return result
}

// GetModifiedFiles returns a list of files modified during the session
func (ct *ChangeTracker) GetModifiedFiles() []string {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	files := make([]string, 0, len(ct.changes))
	for path := range ct.changes {
		files = append(files, path)
	}
	return files
}

// GetStats returns change statistics
func (ct *ChangeTracker) GetStats() ChangeStats {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.stats
}

// GetSessionDuration returns how long the session has been active
func (ct *ChangeTracker) GetSessionDuration() time.Duration {
	return time.Since(ct.sessionStart)
}

// HasChanges returns true if there are any tracked changes
func (ct *ChangeTracker) HasChanges() bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.stats.TotalChanges > 0
}

// GetLastChange returns the most recent change for a file
func (ct *ChangeTracker) GetLastChange(path string) *FileChange {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	if changes, exists := ct.changes[path]; exists && len(changes) > 0 {
		// Return a copy of the last change
		lastChange := changes[len(changes)-1]
		return &lastChange
	}
	return nil
}

// GetChangesSummary returns a summary of changes by type
func (ct *ChangeTracker) GetChangesSummary() map[ChangeType]int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	return map[ChangeType]int{
		ChangeTypeCreate: ct.stats.CreateCount,
		ChangeTypeModify: ct.stats.ModifyCount,
		ChangeTypeDelete: ct.stats.DeleteCount,
		ChangeTypeRename: ct.stats.RenameCount,
	}
}

// GetFrequentlyModifiedFiles returns files sorted by modification frequency
func (ct *ChangeTracker) GetFrequentlyModifiedFiles(limit int) []FileFrequency {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	// Count modifications per file
	frequencies := make(map[string]int)
	for path, changes := range ct.changes {
		count := 0
		for _, change := range changes {
			if change.Type == ChangeTypeModify {
				count++
			}
		}
		if count > 0 {
			frequencies[path] = count
		}
	}

	// Sort by frequency
	result := make([]FileFrequency, 0, len(frequencies))
	for path, count := range frequencies {
		result = append(result, FileFrequency{
			Path:  path,
			Count: count,
		})
	}

	// Sort descending by count
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Count > result[i].Count {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	// Limit results
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result
}

// FileFrequency represents a file and its modification count
type FileFrequency struct {
	Path  string
	Count int
}

// Clear removes all tracked changes
func (ct *ChangeTracker) Clear() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.changes = make(map[string][]FileChange)
	ct.stats = ChangeStats{}
	ct.sessionStart = time.Now()
}

// GetUndoableChanges returns changes that can potentially be undone
func (ct *ChangeTracker) GetUndoableChanges(limit int) []FileChange {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	undoable := make([]FileChange, 0)
	
	// Collect all changes that are undoable (creates and modifies)
	for _, changes := range ct.changes {
		for _, change := range changes {
			if change.Type == ChangeTypeCreate || change.Type == ChangeTypeModify {
				undoable = append(undoable, change)
			}
		}
	}

	// Sort by timestamp (most recent first)
	for i := 0; i < len(undoable)-1; i++ {
		for j := i + 1; j < len(undoable); j++ {
			if undoable[j].Timestamp.After(undoable[i].Timestamp) {
				undoable[i], undoable[j] = undoable[j], undoable[i]
			}
		}
	}

	// Limit results
	if limit > 0 && len(undoable) > limit {
		undoable = undoable[:limit]
	}

	return undoable
}