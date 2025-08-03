package web

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"rcode/db"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

var cacheTTL = 7 * time.Second

// FileNode represents a file or directory in the tree
type FileNode struct {
	Path     string     `json:"path"`
	Name     string     `json:"name"`
	IsDir    bool       `json:"isDir"`
	Size     int64      `json:"size,omitempty"`
	ModTime  time.Time  `json:"modTime"`
	Children []FileNode `json:"children,omitempty"`
	IsOpen   bool       `json:"isOpen,omitempty"`
	Icon     string     `json:"icon,omitempty"`
}

// FileExplorerService manages file system operations
type FileExplorerService struct {
	rootPath       string
	ignorePatterns []string
	cache          map[string]*FileNode
	cacheMutex     sync.RWMutex
	cacheTTL       time.Duration
	cacheTimestamp map[string]time.Time
}

// NewFileExplorerService creates a new file explorer service
func NewFileExplorerService(rootPath string) (*FileExplorerService, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get absolute path")
	}

	// Verify the path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, serr.Wrap(err, "failed to stat root path")
	}
	if !info.IsDir() {
		return nil, serr.New("root path is not a directory")
	}

	service := &FileExplorerService{
		rootPath:       absPath,
		cache:          make(map[string]*FileNode),
		cacheTimestamp: make(map[string]time.Time),
		cacheTTL:       cacheTTL,
		ignorePatterns: getIgnorePatterns(absPath),
	}

	return service, nil
}

// getIgnorePatterns reads .gitignore and .rcodeIgnore files
func getIgnorePatterns(rootPath string) []string {
	patterns := []string{
		".git", ".idea", ".vscode", "node_modules", "__pycache__",
		"*.pyc", "*.pyo", "*.pyd", ".DS_Store", "Thumbs.db",
		"*.log", "*.tmp", "*.temp", "*.cache", "*.swp", "*.swo",
		".env", ".env.local", ".env.*.local",
	}

	// Read .gitignore
	gitignorePath := filepath.Join(rootPath, ".gitignore")
	if data, err := os.ReadFile(gitignorePath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				patterns = append(patterns, line)
			}
		}
	}

	// Read .rcodeIgnore
	rcodeIgnorePath := filepath.Join(rootPath, ".rcodeIgnore")
	if data, err := os.ReadFile(rcodeIgnorePath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				patterns = append(patterns, line)
			}
		}
	}

	return patterns
}

// shouldIgnore checks if a path should be ignored
func (s *FileExplorerService) shouldIgnore(path string) bool {
	base := filepath.Base(path)

	for _, pattern := range s.ignorePatterns {
		// Simple pattern matching (can be enhanced with proper glob matching)
		if strings.Contains(pattern, "*") {
			// Basic wildcard matching
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(base, prefix) {
				return true
			}
		} else if base == pattern {
			return true
		}
	}

	return false
}

// GetTree returns the directory tree starting from a given path
func (s *FileExplorerService) GetTree(relativePath string, depth int) (*FileNode, error) {
	// Validate and clean the path
	cleanPath := filepath.Clean(relativePath)
	if cleanPath == "" || cleanPath == "." {
		cleanPath = ""
	}

	fullPath := filepath.Join(s.rootPath, cleanPath)

	// Security check: ensure path is within root
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return nil, serr.New("access denied: path outside project root")
	}

	// Check cache
	s.cacheMutex.RLock()
	if cached, ok := s.cache[fullPath]; ok {
		if timestamp, exists := s.cacheTimestamp[fullPath]; exists {
			if time.Since(timestamp) < s.cacheTTL {
				s.cacheMutex.RUnlock()
				return cached, nil
			}
		}
	}
	s.cacheMutex.RUnlock()

	// Build tree
	node, err := s.buildTree(fullPath, depth, 0)
	if err != nil {
		return nil, err
	}

	// Update cache
	s.cacheMutex.Lock()
	s.cache[fullPath] = node
	s.cacheTimestamp[fullPath] = time.Now()
	s.cacheMutex.Unlock()

	return node, nil
}

// buildTree recursively builds the file tree
func (s *FileExplorerService) buildTree(path string, maxDepth, currentDepth int) (*FileNode, error) {
	if currentDepth > maxDepth && maxDepth > 0 {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, serr.Wrap(err, "failed to stat path")
	}

	// Get relative path for display
	relPath, err := filepath.Rel(s.rootPath, path)
	if err != nil {
		relPath = path
	}
	if relPath == "." {
		relPath = ""
	}

	node := &FileNode{
		Path:    relPath,
		Name:    filepath.Base(path),
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}

	// Set icon based on file type
	node.Icon = getFileIcon(node.Name, node.IsDir)

	// If it's a directory and we haven't reached max depth, get children
	if info.IsDir() && (maxDepth == 0 || currentDepth < maxDepth) {
		entries, err := os.ReadDir(path)
		if err != nil {
			return node, nil // Return node without children on error
		}

		var children []FileNode
		for _, entry := range entries {
			childPath := filepath.Join(path, entry.Name())

			// Skip ignored files
			if s.shouldIgnore(childPath) {
				continue
			}

			childNode, err := s.buildTree(childPath, maxDepth, currentDepth+1)
			if err != nil {
				continue // Skip files we can't read
			}
			if childNode != nil {
				children = append(children, *childNode)
			}
		}

		// Sort children: directories first, then by name
		sort.Slice(children, func(i, j int) bool {
			if children[i].IsDir != children[j].IsDir {
				return children[i].IsDir
			}
			return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
		})

		node.Children = children
	}

	return node, nil
}

// getFileIcon returns an appropriate icon for the file type
func getFileIcon(name string, isDir bool) string {
	if isDir {
		return "folder"
	}

	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go":
		return "go"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cxx", ".cc", ".hpp":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".php":
		return "php"
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	case ".sql":
		return "database"
	case ".sh", ".bash":
		return "shell"
	case ".vim":
		return "vim"
	case ".git":
		return "git"
	case ".env":
		return "env"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico":
		return "image"
	case ".mp4", ".avi", ".mov", ".wmv":
		return "video"
	case ".mp3", ".wav", ".flac", ".ogg":
		return "audio"
	case ".zip", ".tar", ".gz", ".rar", ".7z":
		return "archive"
	case ".pdf":
		return "pdf"
	case ".doc", ".docx":
		return "word"
	case ".xls", ".xlsx":
		return "excel"
	case ".ppt", ".pptx":
		return "powerpoint"
	default:
		return "file"
	}
}

// GetFileContent returns the content of a file
func (s *FileExplorerService) GetFileContent(relativePath string) (map[string]interface{}, error) {
	// Validate and clean the path
	cleanPath := filepath.Clean(relativePath)
	fullPath := filepath.Join(s.rootPath, cleanPath)

	// Security check: ensure path is within root
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return nil, serr.New("access denied: path outside project root")
	}

	// Check if file exists and is not a directory
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, serr.Wrap(err, "file not found")
	}
	if info.IsDir() {
		return nil, serr.New("path is a directory, not a file")
	}

	// Check file size (limit to 10MB)
	if info.Size() > 10*1024*1024 {
		return nil, serr.New("file too large (max 10MB)")
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, serr.Wrap(err, "failed to read file")
	}

	// Detect if file is binary
	isBinary := isBinaryContent(content)

	result := map[string]interface{}{
		"path":     cleanPath,
		"name":     filepath.Base(fullPath),
		"size":     info.Size(),
		"modTime":  info.ModTime(),
		"isBinary": isBinary,
	}

	if isBinary {
		result["content"] = ""
		result["error"] = "Binary file"
	} else {
		result["content"] = string(content)
	}

	return result, nil
}

// isBinaryContent checks if content appears to be binary
func isBinaryContent(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Check first 512 bytes for null bytes
	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}

	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}

	return false
}

// CreateFile creates a new file with optional content
func (s *FileExplorerService) CreateFile(relativePath string, content string) error {
	// Validate and clean the path
	cleanPath := filepath.Clean(relativePath)
	fullPath := filepath.Join(s.rootPath, cleanPath)

	// Security check: ensure path is within root
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return serr.New("access denied: path outside project root")
	}

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		return serr.New("file already exists")
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return serr.Wrap(err, "failed to create parent directories")
	}

	// Create the file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return serr.Wrap(err, "failed to create file")
	}

	// Clear cache for parent directory
	s.clearCacheForPath(filepath.Dir(cleanPath))

	return nil
}

// CreateDirectory creates a new directory
func (s *FileExplorerService) CreateDirectory(relativePath string) error {
	// Validate and clean the path
	cleanPath := filepath.Clean(relativePath)
	fullPath := filepath.Join(s.rootPath, cleanPath)

	// Security check: ensure path is within root
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return serr.New("access denied: path outside project root")
	}

	// Check if directory already exists
	if _, err := os.Stat(fullPath); err == nil {
		return serr.New("directory already exists")
	}

	// Create the directory (including parent directories)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return serr.Wrap(err, "failed to create directory")
	}

	// Clear cache for parent directory
	s.clearCacheForPath(filepath.Dir(cleanPath))

	return nil
}

// RenameFile renames a file or directory
func (s *FileExplorerService) RenameFile(oldPath, newName string) error {
	// Validate and clean the paths
	cleanOldPath := filepath.Clean(oldPath)
	fullOldPath := filepath.Join(s.rootPath, cleanOldPath)

	// Build new path (in same directory as old file)
	dir := filepath.Dir(cleanOldPath)
	cleanNewPath := filepath.Join(dir, newName)
	fullNewPath := filepath.Join(s.rootPath, cleanNewPath)

	// Security checks
	if !strings.HasPrefix(fullOldPath, s.rootPath) || !strings.HasPrefix(fullNewPath, s.rootPath) {
		return serr.New("access denied: path outside project root")
	}

	// Check if old path exists
	if _, err := os.Stat(fullOldPath); err != nil {
		return serr.Wrap(err, "source file/directory not found")
	}

	// Check if new path already exists
	if _, err := os.Stat(fullNewPath); err == nil {
		return serr.New("destination already exists")
	}

	// Validate new name
	if strings.ContainsAny(newName, "/\\") {
		return serr.New("invalid file name: cannot contain path separators")
	}

	// Perform the rename
	if err := os.Rename(fullOldPath, fullNewPath); err != nil {
		return serr.Wrap(err, "failed to rename")
	}

	// Clear cache for parent directory
	s.clearCacheForPath(dir)

	return nil
}

// DeleteFile deletes a file or directory
func (s *FileExplorerService) DeleteFile(relativePath string) error {
	// Validate and clean the path
	cleanPath := filepath.Clean(relativePath)
	fullPath := filepath.Join(s.rootPath, cleanPath)

	// Security check: ensure path is within root
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return serr.New("access denied: path outside project root")
	}

	// Prevent deletion of critical files
	base := filepath.Base(fullPath)
	criticalFiles := []string{".git", "go.mod", "go.sum", "package.json", "package-lock.json", "yarn.lock", "Gemfile", "Gemfile.lock"}
	for _, critical := range criticalFiles {
		if base == critical {
			return serr.New("cannot delete critical project file")
		}
	}

	// Check if path exists
	info, err := os.Stat(fullPath)
	if err != nil {
		return serr.Wrap(err, "file/directory not found")
	}

	// Delete the file or directory
	if info.IsDir() {
		// For directories, use RemoveAll for recursive deletion
		if err := os.RemoveAll(fullPath); err != nil {
			return serr.Wrap(err, "failed to delete directory")
		}
	} else {
		// For files, use Remove
		if err := os.Remove(fullPath); err != nil {
			return serr.Wrap(err, "failed to delete file")
		}
	}

	// Clear cache for parent directory
	s.clearCacheForPath(filepath.Dir(cleanPath))

	return nil
}

// clearCacheForPath clears the cache for a specific path and its parents
func (s *FileExplorerService) clearCacheForPath(relativePath string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// Clear cache for the specific path
	fullPath := filepath.Join(s.rootPath, relativePath)
	delete(s.cache, fullPath)
	delete(s.cacheTimestamp, fullPath)

	// Clear cache for parent directories up to root
	current := relativePath
	for current != "" && current != "." && current != "/" {
		parent := filepath.Dir(current)
		parentFullPath := filepath.Join(s.rootPath, parent)
		delete(s.cache, parentFullPath)
		delete(s.cacheTimestamp, parentFullPath)

		if parent == current || parent == "." {
			break
		}
		current = parent
	}

	// Also clear root cache
	delete(s.cache, s.rootPath)
	delete(s.cacheTimestamp, s.rootPath)
}

// SearchFiles searches for files by name or content
func (s *FileExplorerService) SearchFiles(query string, searchContent bool) ([]FileNode, error) {
	var results []FileNode
	query = strings.ToLower(query)

	err := filepath.WalkDir(s.rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip paths with errors
		}

		// Skip ignored paths
		if s.shouldIgnore(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check name match
		if strings.Contains(strings.ToLower(d.Name()), query) {
			relPath, _ := filepath.Rel(s.rootPath, path)
			info, err := d.Info()
			if err == nil {
				results = append(results, FileNode{
					Path:    relPath,
					Name:    d.Name(),
					IsDir:   d.IsDir(),
					Size:    info.Size(),
					ModTime: info.ModTime(),
					Icon:    getFileIcon(d.Name(), d.IsDir()),
				})
			}
		}

		// Check content if requested and it's a file
		if searchContent && !d.IsDir() {
			content, err := os.ReadFile(path)
			if err == nil && !isBinaryContent(content) {
				if strings.Contains(strings.ToLower(string(content)), query) {
					relPath, _ := filepath.Rel(s.rootPath, path)
					info, _ := d.Info()

					// Check if already added by name match
					alreadyAdded := false
					for _, r := range results {
						if r.Path == relPath {
							alreadyAdded = true
							break
						}
					}

					if !alreadyAdded {
						results = append(results, FileNode{
							Path:    relPath,
							Name:    d.Name(),
							IsDir:   false,
							Size:    info.Size(),
							ModTime: info.ModTime(),
							Icon:    getFileIcon(d.Name(), false),
						})
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, serr.Wrap(err, "search failed")
	}

	// Sort results: directories first, then by name
	sort.Slice(results, func(i, j int) bool {
		if results[i].IsDir != results[j].IsDir {
			return results[i].IsDir
		}
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	// Limit results to 100
	if len(results) > 100 {
		results = results[:100]
	}

	return results, nil
}

// Global file explorer service instance
var fileExplorer *FileExplorerService

// InitFileExplorer initializes the file explorer service
func InitFileExplorer(rootPath string) error {
	service, err := NewFileExplorerService(rootPath)
	if err != nil {
		return err
	}
	fileExplorer = service
	return nil
}

// File Explorer API Handlers

// getFileTreeHandler returns the directory tree
func getFileTreeHandler(c rweb.Context) error {
	if fileExplorer == nil {
		return c.WriteError(serr.New("file explorer not initialized"), 500)
	}

	path := c.Request().QueryParam("path")
	depthStr := c.Request().QueryParam("depth")
	if depthStr == "" {
		depthStr = "2"
	}
	depth := 2
	if d, err := strconv.Atoi(depthStr); err == nil && d >= 0 && d <= 5 {
		depth = d
	}

	tree, err := fileExplorer.GetTree(path, depth)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get tree"), 400)
	}

	// Add the absolute root path to the response
	// If path is empty, we're at the root
	absolutePath := fileExplorer.rootPath
	if path != "" && path != "." {
		absolutePath = filepath.Join(fileExplorer.rootPath, path)
	}

	// Abbreviate home directory to ~ for display
	displayPath := absolutePath
	homeDir, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(displayPath, homeDir) {
		displayPath = "~" + strings.TrimPrefix(displayPath, homeDir)
	}

	// Create a wrapper response with the working directory
	response := map[string]interface{}{
		"path":        absolutePath,
		"displayPath": displayPath,
		"children":    tree.Children,
		"name":        tree.Name,
		"isDir":       tree.IsDir,
	}

	return c.WriteJSON(response)
}

// getCurrentWorkingDirectoryHandler returns the current working directory
func getCurrentWorkingDirectoryHandler(c rweb.Context) error {
	if fileExplorer == nil {
		return c.WriteError(serr.New("file explorer not initialized"), 500)
	}

	// Abbreviate home directory to ~ for display
	displayPath := fileExplorer.rootPath
	homeDir, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(displayPath, homeDir) {
		displayPath = "~" + strings.TrimPrefix(displayPath, homeDir)
	}

	return c.WriteJSON(map[string]string{
		"path":        fileExplorer.rootPath,
		"displayPath": displayPath,
	})
}

// getFileContentHandler returns file content
func getFileContentHandler(c rweb.Context) error {
	if fileExplorer == nil {
		return c.WriteError(serr.New("file explorer not initialized"), 500)
	}

	// Get the path from the URL after /api/files/content/
	fullPath := c.Request().Path()
	prefix := "/api/files/content/"
	if !strings.HasPrefix(fullPath, prefix) {
		return c.WriteError(serr.New("invalid path"), 400)
	}

	path := strings.TrimPrefix(fullPath, prefix)
	if path == "" {
		return c.WriteError(serr.New("path parameter required"), 400)
	}

	content, err := fileExplorer.GetFileContent(path)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get file content"), 400)
	}

	return c.WriteJSON(content)
}

// searchFilesHandler searches for files
func searchFilesHandler(c rweb.Context) error {
	if fileExplorer == nil {
		return c.WriteError(serr.New("file explorer not initialized"), 500)
	}

	var req struct {
		Query         string `json:"query"`
		SearchContent bool   `json:"searchContent"`
	}

	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.New("invalid request body"), 400)
	}

	if req.Query == "" {
		return c.WriteError(serr.New("query parameter required"), 400)
	}

	results, err := fileExplorer.SearchFiles(req.Query, req.SearchContent)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "search failed"), 500)
	}

	return c.WriteJSON(map[string]interface{}{
		"results": results,
		"count":   len(results),
		"query":   req.Query,
	})
}

// openFileHandler tracks opened files in session
func openFileHandler(c rweb.Context) error {
	sessionId := c.Request().Param("id")
	if sessionId == "" {
		return c.WriteError(serr.New("session ID required"), 400)
	}

	var req struct {
		Path string `json:"path"`
	}

	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.New("invalid request body"), 400)
	}

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Track file opening in database
	if err := database.OpenFileInSession(sessionId, req.Path); err != nil {
		logger.LogErr(err, "failed to track file opening")
		// Don't fail the request, just log the error
	}

	// Broadcast file opened event
	BroadcastFileOpened(sessionId, req.Path)

	return c.WriteJSON(map[string]interface{}{
		"status": "ok",
		"path":   req.Path,
	})
}

// getRecentFilesHandler returns recently accessed files for a session
func getRecentFilesHandler(c rweb.Context) error {
	sessionId := c.Request().Param("id")
	if sessionId == "" {
		return c.WriteError(serr.New("session ID required"), 400)
	}

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Get recent files from database
	recentFiles, err := database.GetRecentFiles(sessionId, 20)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get recent files"), 500)
	}

	// Convert to FileNode format for consistency
	var fileNodes []FileNode
	for _, file := range recentFiles {
		// Get file info if it exists
		if info, err := os.Stat(filepath.Join(fileExplorer.rootPath, file.FilePath)); err == nil {
			fileNodes = append(fileNodes, FileNode{
				Path:    file.FilePath,
				Name:    filepath.Base(file.FilePath),
				IsDir:   info.IsDir(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Icon:    getFileIcon(filepath.Base(file.FilePath), info.IsDir()),
			})
		}
	}

	return c.WriteJSON(map[string]interface{}{
		"files": fileNodes,
		"count": len(fileNodes),
	})
}

// createFileHandler creates a new file or directory
func createFileHandler(c rweb.Context) error {
	if fileExplorer == nil {
		return c.WriteError(serr.New("file explorer not initialized"), 500)
	}

	var req struct {
		Path    string `json:"path"`
		Type    string `json:"type"` // "file" or "directory"
		Content string `json:"content,omitempty"`
	}

	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.New("invalid request body"), 400)
	}

	if req.Path == "" {
		return c.WriteError(serr.New("path parameter required"), 400)
	}

	if req.Type != "file" && req.Type != "directory" {
		return c.WriteError(serr.New("type must be 'file' or 'directory'"), 400)
	}

	var err error
	if req.Type == "file" {
		err = fileExplorer.CreateFile(req.Path, req.Content)
	} else {
		err = fileExplorer.CreateDirectory(req.Path)
	}

	if err != nil {
		return c.WriteError(err, 400)
	}

	// Broadcast file tree update event
	BroadcastFileTreeUpdate("", filepath.Dir(req.Path))

	return c.WriteJSON(map[string]interface{}{
		"status": "ok",
		"path":   req.Path,
		"type":   req.Type,
	})
}

// renameFileHandler renames a file or directory
func renameFileHandler(c rweb.Context) error {
	if fileExplorer == nil {
		return c.WriteError(serr.New("file explorer not initialized"), 500)
	}

	var req struct {
		OldPath string `json:"oldPath"`
		NewName string `json:"newName"`
	}

	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.New("invalid request body"), 400)
	}

	if req.OldPath == "" || req.NewName == "" {
		return c.WriteError(serr.New("oldPath and newName parameters required"), 400)
	}

	err := fileExplorer.RenameFile(req.OldPath, req.NewName)
	if err != nil {
		return c.WriteError(err, 400)
	}

	// Build new path for response
	dir := filepath.Dir(req.OldPath)
	newPath := filepath.Join(dir, req.NewName)

	// Broadcast file tree update event
	BroadcastFileTreeUpdate("", dir)

	return c.WriteJSON(map[string]interface{}{
		"status":  "ok",
		"oldPath": req.OldPath,
		"newPath": newPath,
	})
}

// deleteFileHandler deletes a file or directory
func deleteFileHandler(c rweb.Context) error {
	if fileExplorer == nil {
		return c.WriteError(serr.New("file explorer not initialized"), 500)
	}

	var req struct {
		Path string `json:"path"`
	}

	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.New("invalid request body"), 400)
	}

	if req.Path == "" {
		return c.WriteError(serr.New("path parameter required"), 400)
	}

	err := fileExplorer.DeleteFile(req.Path)
	if err != nil {
		return c.WriteError(err, 400)
	}

	// Broadcast file tree update event
	BroadcastFileTreeUpdate("", filepath.Dir(req.Path))

	return c.WriteJSON(map[string]interface{}{
		"status": "ok",
		"path":   req.Path,
	})
}
