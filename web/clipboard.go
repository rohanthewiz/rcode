package web

import (
	"sync"
	"time"
)

// ClipboardMode represents the type of clipboard operation
type ClipboardMode string

const (
	ClipboardModeCopy ClipboardMode = "copy"
	ClipboardModeCut  ClipboardMode = "cut"
)

// FileInfo represents a file or directory in the clipboard
type FileInfo struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	IsDir    bool      `json:"isDir"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// Clipboard holds files for copy/cut operations
type Clipboard struct {
	SessionID string        `json:"sessionId"`
	Mode      ClipboardMode `json:"mode"`
	Files     []FileInfo    `json:"files"`
	Timestamp time.Time     `json:"timestamp"`
}

// ClipboardManager manages session-based clipboards
type ClipboardManager struct {
	mu         sync.RWMutex
	clipboards map[string]*Clipboard // sessionID -> clipboard
}

// NewClipboardManager creates a new clipboard manager
func NewClipboardManager() *ClipboardManager {
	return &ClipboardManager{
		clipboards: make(map[string]*Clipboard),
	}
}

// Set updates the clipboard for a session
func (cm *ClipboardManager) Set(sessionID string, mode ClipboardMode, files []FileInfo) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.clipboards[sessionID] = &Clipboard{
		SessionID: sessionID,
		Mode:      mode,
		Files:     files,
		Timestamp: time.Now(),
	}
}

// Get retrieves the clipboard for a session
func (cm *ClipboardManager) Get(sessionID string) *Clipboard {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	clipboard, exists := cm.clipboards[sessionID]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	return &Clipboard{
		SessionID: clipboard.SessionID,
		Mode:      clipboard.Mode,
		Files:     append([]FileInfo{}, clipboard.Files...),
		Timestamp: clipboard.Timestamp,
	}
}

// Clear removes the clipboard for a session
func (cm *ClipboardManager) Clear(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.clipboards, sessionID)
}

// HasContent checks if a session has clipboard content
func (cm *ClipboardManager) HasContent(sessionID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	clipboard, exists := cm.clipboards[sessionID]
	return exists && len(clipboard.Files) > 0
}

// CleanupOld removes clipboards older than the specified duration
func (cm *ClipboardManager) CleanupOld(maxAge time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	for sessionID, clipboard := range cm.clipboards {
		if now.Sub(clipboard.Timestamp) > maxAge {
			delete(cm.clipboards, sessionID)
		}
	}
}

// Global clipboard manager instance
var clipboardManager = NewClipboardManager()

// GetClipboardManager returns the global clipboard manager
func GetClipboardManager() *ClipboardManager {
	return clipboardManager
}
