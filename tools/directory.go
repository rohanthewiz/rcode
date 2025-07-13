package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rohanthewiz/serr"
)

// ListDirTool implements directory listing functionality
type ListDirTool struct{}

// GetDefinition returns the tool definition for directory listing
func (t *ListDirTool) GetDefinition() Tool {
	return Tool{
		Name:        "list_dir",
		Description: "List contents of a directory with file information",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The directory path to list (defaults to current directory)",
				},
				"all": map[string]interface{}{
					"type":        "boolean",
					"description": "Include hidden files (starting with .)",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "List subdirectories recursively",
				},
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Glob pattern to filter files (e.g., '*.go')",
				},
			},
			"required": []string{},
		},
	}
}

// Execute lists the directory contents
func (t *ListDirTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		path = "."
	}

	// Expand the path to handle ~ for home directory
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand path")
	}
	path = expandedPath

	showAll := false
	if val, exists := input["all"]; exists {
		if boolVal, ok := val.(bool); ok {
			showAll = boolVal
		}
	}

	recursive := false
	if val, exists := input["recursive"]; exists {
		if boolVal, ok := val.(bool); ok {
			recursive = boolVal
		}
	}

	pattern, _ := GetString(input, "pattern")

	// Check if path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		return "", serr.Wrap(err, fmt.Sprintf("Cannot access path: %s", path))
	}
	if !info.IsDir() {
		return "", serr.New(fmt.Sprintf("Path is not a directory: %s", path))
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Directory: %s\n\n", path))

	if recursive {
		err = listDirRecursive(&output, path, "", showAll, pattern)
	} else {
		err = listDirFlat(&output, path, showAll, pattern)
	}

	if err != nil {
		return "", serr.Wrap(err, "Error listing directory")
	}

	return output.String(), nil
}

// listDirFlat lists directory contents non-recursively
func listDirFlat(output *strings.Builder, path string, showAll bool, pattern string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Filter and sort entries
	var filtered []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		
		// Skip hidden files if not showing all
		if !showAll && strings.HasPrefix(name, ".") {
			continue
		}

		// Apply pattern filter if specified
		if pattern != "" {
			matched, _ := filepath.Match(pattern, name)
			if !matched {
				continue
			}
		}

		filtered = append(filtered, entry)
	}

	// Sort entries (directories first, then by name)
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].IsDir() != filtered[j].IsDir() {
			return filtered[i].IsDir()
		}
		return filtered[i].Name() < filtered[j].Name()
	})

	// Display entries
	for _, entry := range filtered {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		typeStr := "file"
		sizeStr := fmt.Sprintf("%10d", info.Size())
		if entry.IsDir() {
			typeStr = "dir "
			sizeStr = "          "
		}

		output.WriteString(fmt.Sprintf("%s %s %s %s\n",
			typeStr,
			info.Mode().String(),
			sizeStr,
			entry.Name(),
		))
	}

	return nil
}

// listDirRecursive lists directory contents recursively as a tree
func listDirRecursive(output *strings.Builder, basePath, prefix string, showAll bool, pattern string) error {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return err
	}

	// Filter entries
	var filtered []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		
		// Skip hidden files if not showing all
		if !showAll && strings.HasPrefix(name, ".") {
			continue
		}

		// For directories, always include (we'll filter contents)
		// For files, apply pattern filter
		if !entry.IsDir() && pattern != "" {
			matched, _ := filepath.Match(pattern, name)
			if !matched {
				continue
			}
		}

		filtered = append(filtered, entry)
	}

	// Sort entries
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name() < filtered[j].Name()
	})

	// Display entries
	for i, entry := range filtered {
		isLast := i == len(filtered)-1
		
		// Tree drawing characters
		connector := "├── "
		childPrefix := "│   "
		if isLast {
			connector = "└── "
			childPrefix = "    "
		}

		output.WriteString(prefix + connector + entry.Name())
		
		if entry.IsDir() {
			output.WriteString("/\n")
			
			// Recurse into subdirectory
			subPath := filepath.Join(basePath, entry.Name())
			err := listDirRecursive(output, subPath, prefix+childPrefix, showAll, pattern)
			if err != nil {
				// Continue even if we can't read a subdirectory
				output.WriteString(prefix + childPrefix + "(error reading directory)\n")
			}
		} else {
			output.WriteString("\n")
		}
	}

	return nil
}

// MakeDirTool implements directory creation functionality
type MakeDirTool struct{}

// GetDefinition returns the tool definition for directory creation
func (t *MakeDirTool) GetDefinition() Tool {
	return Tool{
		Name:        "make_dir",
		Description: "Create a new directory",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The directory path to create",
				},
				"parents": map[string]interface{}{
					"type":        "boolean",
					"description": "Create parent directories if they don't exist (like mkdir -p)",
				},
			},
			"required": []string{"path"},
		},
	}
}

// Execute creates the directory
func (t *MakeDirTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		return "", serr.New("path is required")
	}

	// Expand the path to handle ~ for home directory
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand path")
	}
	path = expandedPath

	createParents := false
	if val, exists := input["parents"]; exists {
		if boolVal, ok := val.(bool); ok {
			createParents = boolVal
		}
	}

	// Check if already exists
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return fmt.Sprintf("Directory already exists: %s", path), nil
		}
		return "", serr.New(fmt.Sprintf("Path exists but is not a directory: %s", path))
	}

	// Create directory
	if createParents {
		err = os.MkdirAll(path, 0755)
	} else {
		err = os.Mkdir(path, 0755)
	}

	if err != nil {
		return "", serr.Wrap(err, fmt.Sprintf("Failed to create directory: %s", path))
	}

	return fmt.Sprintf("Directory created: %s", path), nil
}

// RemoveTool implements file/directory removal functionality
type RemoveTool struct{}

// GetDefinition returns the tool definition for removal
func (t *RemoveTool) GetDefinition() Tool {
	return Tool{
		Name:        "remove",
		Description: "Remove a file or directory",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file or directory path to remove",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "Remove directories and their contents recursively",
				},
				"force": map[string]interface{}{
					"type":        "boolean",
					"description": "Force removal without confirmation",
				},
			},
			"required": []string{"path"},
		},
	}
}

// Execute removes the file or directory
func (t *RemoveTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		return "", serr.New("path is required")
	}

	// Expand the path to handle ~ for home directory
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand path")
	}
	path = expandedPath

	recursive := false
	if val, exists := input["recursive"]; exists {
		if boolVal, ok := val.(bool); ok {
			recursive = boolVal
		}
	}

	// Safety check - prevent removing important directories
	abspath, err := filepath.Abs(path)
	if err == nil {
		dangerous := []string{"/", "/etc", "/usr", "/var", "/bin", "/sbin", "/home", "/Users"}
		for _, d := range dangerous {
			if abspath == d || abspath == d+"/" {
				return "", serr.New(fmt.Sprintf("Refusing to remove dangerous path: %s", path))
			}
		}
	}

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Path does not exist: %s", path), nil
		}
		return "", serr.Wrap(err, fmt.Sprintf("Cannot access path: %s", path))
	}

	// Remove the path
	if info.IsDir() && recursive {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}

	if err != nil {
		return "", serr.Wrap(err, fmt.Sprintf("Failed to remove: %s", path))
	}

	typeStr := "File"
	if info.IsDir() {
		typeStr = "Directory"
	}

	return fmt.Sprintf("%s removed: %s", typeStr, path), nil
}

// TreeTool implements tree-like directory visualization
type TreeTool struct{}

// GetDefinition returns the tool definition for tree visualization
func (t *TreeTool) GetDefinition() Tool {
	return Tool{
		Name:        "tree",
		Description: "Display directory structure as a tree",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The directory path to display (defaults to current directory)",
				},
				"max_depth": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum depth to traverse (default: 5)",
				},
				"all": map[string]interface{}{
					"type":        "boolean",
					"description": "Include hidden files",
				},
				"dirs_only": map[string]interface{}{
					"type":        "boolean",
					"description": "Show only directories",
				},
			},
			"required": []string{},
		},
	}
}

// Execute displays the directory tree
func (t *TreeTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		path = "."
	}

	// Expand the path to handle ~ for home directory
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand path")
	}
	path = expandedPath

	maxDepth, ok := GetInt(input, "max_depth")
	if !ok {
		maxDepth = 5
	}

	showAll := false
	if val, exists := input["all"]; exists {
		if boolVal, ok := val.(bool); ok {
			showAll = boolVal
		}
	}

	dirsOnly := false
	if val, exists := input["dirs_only"]; exists {
		if boolVal, ok := val.(bool); ok {
			dirsOnly = boolVal
		}
	}

	// Check if path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		return "", serr.Wrap(err, fmt.Sprintf("Cannot access path: %s", path))
	}
	if !info.IsDir() {
		return "", serr.New(fmt.Sprintf("Path is not a directory: %s", path))
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("%s\n", path))

	stats := &treeStats{}
	err = buildTree(&output, path, "", 0, maxDepth, showAll, dirsOnly, stats)
	if err != nil {
		return "", serr.Wrap(err, "Error building tree")
	}

	// Add summary
	output.WriteString(fmt.Sprintf("\n%d directories", stats.dirs))
	if !dirsOnly {
		output.WriteString(fmt.Sprintf(", %d files", stats.files))
	}
	output.WriteString("\n")

	return output.String(), nil
}

type treeStats struct {
	dirs  int
	files int
}

// buildTree recursively builds the tree structure
func buildTree(output *strings.Builder, basePath, prefix string, depth, maxDepth int, showAll, dirsOnly bool, stats *treeStats) error {
	if depth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return err
	}

	// Filter entries
	var filtered []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		
		// Skip hidden files if not showing all
		if !showAll && strings.HasPrefix(name, ".") {
			continue
		}

		// Skip files if dirs only
		if dirsOnly && !entry.IsDir() {
			continue
		}

		filtered = append(filtered, entry)
	}

	// Sort entries
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name() < filtered[j].Name()
	})

	// Display entries
	for i, entry := range filtered {
		isLast := i == len(filtered)-1
		
		// Tree drawing characters
		connector := "├── "
		childPrefix := "│   "
		if isLast {
			connector = "└── "
			childPrefix = "    "
		}

		output.WriteString(prefix + connector + entry.Name())
		
		if entry.IsDir() {
			output.WriteString("/\n")
			stats.dirs++
			
			// Recurse into subdirectory
			subPath := filepath.Join(basePath, entry.Name())
			err := buildTree(output, subPath, prefix+childPrefix, depth+1, maxDepth, showAll, dirsOnly, stats)
			if err != nil {
				// Continue even if we can't read a subdirectory
				continue
			}
		} else {
			output.WriteString("\n")
			stats.files++
		}
	}

	return nil
}

// MoveTool implements file/directory moving functionality
type MoveTool struct{}

// GetDefinition returns the tool definition for moving files
func (t *MoveTool) GetDefinition() Tool {
	return Tool{
		Name:        "move",
		Description: "Move or rename a file or directory",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"source": map[string]interface{}{
					"type":        "string",
					"description": "Source file or directory path",
				},
				"destination": map[string]interface{}{
					"type":        "string",
					"description": "Destination path",
				},
			},
			"required": []string{"source", "destination"},
		},
	}
}

// Execute moves the file or directory
func (t *MoveTool) Execute(input map[string]interface{}) (string, error) {
	source, ok := GetString(input, "source")
	if !ok || source == "" {
		return "", serr.New("source is required")
	}

	destination, ok := GetString(input, "destination")
	if !ok || destination == "" {
		return "", serr.New("destination is required")
	}

	// Expand paths to handle ~ for home directory
	expandedSource, err := ExpandPath(source)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand source path")
	}
	
	expandedDestination, err := ExpandPath(destination)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand destination path")
	}

	// Check if source exists
	sourceInfo, err := os.Stat(expandedSource)
	if err != nil {
		return "", serr.Wrap(err, fmt.Sprintf("Cannot access source: %s", source))
	}

	// Check if destination exists
	destInfo, err := os.Stat(expandedDestination)
	if err == nil && destInfo.IsDir() && !sourceInfo.IsDir() {
		// Moving file into directory
		expandedDestination = filepath.Join(expandedDestination, filepath.Base(expandedSource))
	}

	// Perform the move
	err = os.Rename(expandedSource, expandedDestination)
	if err != nil {
		return "", serr.Wrap(err, fmt.Sprintf("Failed to move %s to %s", source, destination))
	}

	return fmt.Sprintf("Moved: %s → %s", source, destination), nil
}