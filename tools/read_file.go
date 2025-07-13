package tools

import (
	"fmt"
	"os"
	"strings"

	"github.com/rohanthewiz/serr"
)

// ReadFileTool implements file reading functionality
type ReadFileTool struct{}

// GetDefinition returns the tool definition for the AI
func (t *ReadFileTool) GetDefinition() Tool {
	return Tool{
		Name:        "read_file",
		Description: "Read the contents of a file at the specified path",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The path to the file to read",
				},
			},
			"required": []string{"path"},
		},
	}
}

// Execute reads the file and returns its contents
func (t *ReadFileTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		return "", serr.New("path is required")
	}

	// Expand the path to handle ~ for home directory
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand path")
	}

	// Read the file
	content, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", serr.New(fmt.Sprintf("File not found: %s", path))
		}
		return "", serr.Wrap(err, fmt.Sprintf("Failed to read file: %s", path))
	}

	// Add line numbers like the TypeScript version
	lines := strings.Split(string(content), "\n")
	numberedLines := make([]string, len(lines))
	for i, line := range lines {
		numberedLines[i] = fmt.Sprintf("%d\t%s", i+1, line)
	}

	result := strings.Join(numberedLines, "\n")

	// Truncate if too long (similar to TypeScript version)
	const maxLength = 30000
	if len(result) > maxLength {
		result = result[:maxLength] + "\n\n[Content truncated...]"
	}

	return result, nil
}
