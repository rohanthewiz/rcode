package context

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/rohanthewiz/serr"
)

// Manager handles project context and file tracking
type Manager struct {
	mu           sync.RWMutex
	context      *ProjectContext
	changes      []FileChange
	scanner      *ProjectScanner
	prioritizer  *FilePrioritizer
	changeTracker *ChangeTracker
}

// NewManager creates a new context manager
func NewManager() *Manager {
	return &Manager{
		changes:       make([]FileChange, 0),
		scanner:       NewProjectScanner(),
		prioritizer:   NewFilePrioritizer(),
		changeTracker: NewChangeTracker(),
	}
}

// ScanProject scans a project directory and builds context
func (m *Manager) ScanProject(path string) (*ProjectContext, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use the scanner to analyze the project
	ctx, err := m.scanner.Scan(path)
	if err != nil {
		return nil, serr.Wrap(err, "failed to scan project")
	}

	m.context = ctx
	return ctx, nil
}

// GetContext returns the current project context
func (m *Manager) GetContext() *ProjectContext {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.context
}

// PrioritizeFiles returns files prioritized for a given task
func (m *Manager) PrioritizeFiles(task string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.context == nil {
		return nil, serr.New("no project context available")
	}

	taskCtx := &TaskContext{
		Task:     task,
		MaxFiles: 20, // Default max files
	}

	return m.prioritizer.Prioritize(m.context, taskCtx)
}

// TrackChange records a file change
func (m *Manager) TrackChange(filepath string, changeType ChangeType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	change := FileChange{
		Path:      filepath,
		Type:      changeType,
		Timestamp: time.Now(),
	}

	m.changes = append(m.changes, change)
	
	// Update modified files in context
	if m.context != nil && m.context.ModifiedFiles != nil {
		if changeType == ChangeTypeDelete {
			delete(m.context.ModifiedFiles, filepath)
		} else {
			m.context.ModifiedFiles[filepath] = time.Now()
		}
	}

	// Track in change tracker
	m.changeTracker.Track(change)
}

// GetRelevantContext returns context relevant to a specific task
func (m *Manager) GetRelevantContext(task string) (*TaskContext, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.context == nil {
		return nil, serr.New("no project context available")
	}

	// Extract search terms from the task
	searchTerms := extractSearchTerms(task)

	// Get prioritized files
	taskCtx := &TaskContext{
		Task:        task,
		SearchTerms: searchTerms,
		MaxFiles:    20,
		FileScores:  make(map[string]float64),
	}

	files, err := m.prioritizer.Prioritize(m.context, taskCtx)
	if err != nil {
		return nil, serr.Wrap(err, "failed to prioritize files")
	}

	taskCtx.RelevantFiles = files
	return taskCtx, nil
}

// GetRecentChanges returns recent file changes
func (m *Manager) GetRecentChanges(limit int) []FileChange {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.changes) {
		limit = len(m.changes)
	}

	// Return most recent changes
	start := len(m.changes) - limit
	if start < 0 {
		start = 0
	}

	result := make([]FileChange, limit)
	copy(result, m.changes[start:])
	return result
}

// UpdateFileMetadata updates metadata for a specific file
func (m *Manager) UpdateFileMetadata(path string, metadata FileMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.context == nil || m.context.FileTree == nil {
		return serr.New("no project context available")
	}

	// Find the file node
	node := findFileNode(m.context.FileTree, path)
	if node == nil {
		return serr.New("file not found in context")
	}

	node.Metadata = metadata
	return nil
}

// RefreshFile refreshes information about a specific file
func (m *Manager) RefreshFile(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.context == nil {
		return serr.New("no project context available")
	}

	// Re-scan just this file
	return m.scanner.RefreshFile(m.context, path)
}

// GetContextWindow returns an optimized context window for the AI
func (m *Manager) GetContextWindow(files []string, maxTokens int) (*ContextWindow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	window := &ContextWindow{
		Files:     make([]ContextFile, 0),
		MaxTokens: maxTokens,
		Priority:  "relevance",
	}

	// TODO: Implement token counting and optimization
	// For now, just include all requested files
	for _, file := range files {
		// Read file content and estimate tokens
		contextFile := ContextFile{
			Path:     file,
			Score:    1.0,
			Included: true,
		}
		window.Files = append(window.Files, contextFile)
	}

	return window, nil
}

// Helper function to find a file node in the tree
func findFileNode(root *FileNode, path string) *FileNode {
	if root.Path == path {
		return root
	}

	if root.Children != nil {
		for _, child := range root.Children {
			if node := findFileNode(child, path); node != nil {
				return node
			}
		}
	}

	return nil
}

// Helper function to extract search terms from a task description
func extractSearchTerms(task string) []string {
	// TODO: Implement proper NLP-based term extraction
	// For now, return empty slice
	return []string{}
}

// IsInitialized returns whether the context manager has been initialized
func (m *Manager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.context != nil
}

// GetProjectRoot returns the project root path
func (m *Manager) GetProjectRoot() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.context == nil {
		return ""
	}
	return m.context.RootPath
}

// AddRecentFile adds a file to the recent files list
func (m *Manager) AddRecentFile(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.context == nil {
		return
	}

	// Remove if already exists
	for i, f := range m.context.RecentFiles {
		if f == path {
			m.context.RecentFiles = append(m.context.RecentFiles[:i], m.context.RecentFiles[i+1:]...)
			break
		}
	}

	// Add to front
	m.context.RecentFiles = append([]string{path}, m.context.RecentFiles...)

	// Keep only last 50 files
	if len(m.context.RecentFiles) > 50 {
		m.context.RecentFiles = m.context.RecentFiles[:50]
	}
}