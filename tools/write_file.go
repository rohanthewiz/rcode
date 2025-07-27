package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rohanthewiz/serr"
)

// WriteFileTool implements file writing functionality
type WriteFileTool struct{}

// GetDefinition returns the tool definition for the AI
func (t *WriteFileTool) GetDefinition() Tool {
	return Tool{
		Name:        "write_file",
		Description: "Write content to a file at the specified path. Creates the file if it doesn't exist.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The path where the file should be written",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
	}
}

// Execute writes the content to the file
func (t *WriteFileTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		return "", serr.New("path is required")
	}

	content, ok := GetString(input, "content")
	if !ok {
		return "", serr.New("content is required")
	}

	// Expand the path to handle ~ for home directory
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand path")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		if os.IsPermission(err) {
			return "", NewPermanentError(serr.Wrap(err, fmt.Sprintf("Permission denied creating directory: %s", dir)), "permission denied")
		}
		// Other errors might be temporary (disk full, system resources, etc)
		return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Failed to create directory: %s", dir)))
	}

	// Check if file exists (to determine if it's a create or modify)
	fileExists := false
	if _, err := os.Stat(expandedPath); err == nil {
		fileExists = true
	}

	// Write the file
	if err := os.WriteFile(expandedPath, []byte(content), 0644); err != nil {
		if os.IsPermission(err) {
			return "", NewPermanentError(serr.Wrap(err, fmt.Sprintf("Permission denied writing file: %s", path)), "permission denied")
		}
		// Other errors might be temporary (disk full, file locked, etc)
		return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Failed to write file: %s", path)))
	}

	// Notify file change
	if fileExists {
		NotifyFileChange(path, "modified")
	} else {
		NotifyFileChange(path, "created")
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}
