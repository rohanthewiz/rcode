package tools

import (
	"fmt"
	"os"
	"strings"

	"github.com/rohanthewiz/serr"
)

// EditFileTool implements line-based file editing functionality
type EditFileTool struct{}

// GetDefinition returns the tool definition for the AI
func (t *EditFileTool) GetDefinition() Tool {
	return Tool{
		Name:        "edit_file",
		Description: "Edit a file by replacing specific lines or line ranges. Supports single line edits, multi-line replacements, and insertions.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The path to the file to edit",
				},
				"start_line": map[string]interface{}{
					"type":        "integer",
					"description": "Starting line number (1-based) for the edit",
				},
				"end_line": map[string]interface{}{
					"type":        "integer",
					"description": "Ending line number (inclusive) for the edit. If not provided, only start_line is edited.",
				},
				"new_content": map[string]interface{}{
					"type":        "string",
					"description": "The new content to replace the specified lines. Use empty string to delete lines.",
				},
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "Operation type: 'replace' (default), 'insert_before', 'insert_after'",
					"enum":        []string{"replace", "insert_before", "insert_after"},
				},
			},
			"required": []string{"path", "start_line", "new_content"},
		},
	}
}

// Execute performs the file edit operation
func (t *EditFileTool) Execute(input map[string]interface{}) (string, error) {
	// Extract parameters
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		return "", serr.New("path is required")
	}

	// Expand the path to handle ~ for home directory
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand path")
	}

	startLine, ok := GetInt(input, "start_line")
	if !ok || startLine < 1 {
		return "", serr.New("start_line is required and must be >= 1")
	}

	endLine, hasEndLine := GetInt(input, "end_line")
	if !hasEndLine {
		endLine = startLine
	}
	if endLine < startLine {
		return "", serr.New("end_line must be >= start_line")
	}

	newContent, ok := GetString(input, "new_content")
	if !ok {
		return "", serr.New("new_content is required")
	}

	operation, ok := GetString(input, "operation")
	if !ok {
		operation = "replace"
	}

	// Read the original file
	originalContent, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", NewPermanentError(serr.New(fmt.Sprintf("File not found: %s", path)), "file not found")
		}
		if os.IsPermission(err) {
			return "", NewPermanentError(serr.Wrap(err, fmt.Sprintf("Permission denied reading file: %s", path)), "permission denied")
		}
		return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Failed to read file: %s", path)))
	}

	// Split into lines
	lines := strings.Split(string(originalContent), "\n")
	
	// Handle case where file ends with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Validate line numbers
	if startLine > len(lines) {
		return "", serr.New(fmt.Sprintf("start_line %d exceeds file length %d", startLine, len(lines)))
	}
	if endLine > len(lines) {
		return "", serr.New(fmt.Sprintf("end_line %d exceeds file length %d", endLine, len(lines)))
	}

	// Prepare the new content lines
	newLines := strings.Split(newContent, "\n")
	if len(newLines) > 0 && newLines[len(newLines)-1] == "" {
		newLines = newLines[:len(newLines)-1]
	}

	// Perform the operation
	var result []string
	switch operation {
	case "insert_before":
		result = append(result, lines[:startLine-1]...)
		result = append(result, newLines...)
		result = append(result, lines[startLine-1:]...)
	case "insert_after":
		result = append(result, lines[:endLine]...)
		result = append(result, newLines...)
		result = append(result, lines[endLine:]...)
	case "replace":
		fallthrough
	default:
		result = append(result, lines[:startLine-1]...)
		if newContent != "" {
			result = append(result, newLines...)
		}
		result = append(result, lines[endLine:]...)
	}

	// Write the modified content back
	modifiedContent := strings.Join(result, "\n")
	if len(result) > 0 {
		modifiedContent += "\n"
	}

	err = os.WriteFile(expandedPath, []byte(modifiedContent), 0644)
	if err != nil {
		if os.IsPermission(err) {
			return "", NewPermanentError(serr.Wrap(err, fmt.Sprintf("Permission denied writing file: %s", path)), "permission denied")
		}
		return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Failed to write file: %s", path)))
	}

	// Notify file change
	NotifyFileChange(path, "modified")

	// Generate diff-like output for confirmation
	var diffOutput strings.Builder
	diffOutput.WriteString(fmt.Sprintf("File edited: %s\n", path))
	diffOutput.WriteString(fmt.Sprintf("Operation: %s\n", operation))
	
	// Show what was changed
	if operation == "replace" {
		diffOutput.WriteString(fmt.Sprintf("\nLines %d-%d replaced:\n", startLine, endLine))
		diffOutput.WriteString("--- Before:\n")
		for i := startLine - 1; i < endLine && i < len(lines); i++ {
			diffOutput.WriteString(fmt.Sprintf("%d: %s\n", i+1, lines[i]))
		}
		diffOutput.WriteString("+++ After:\n")
		if newContent != "" {
			for i, line := range newLines {
				diffOutput.WriteString(fmt.Sprintf("%d: %s\n", startLine+i, line))
			}
		} else {
			diffOutput.WriteString("(lines deleted)\n")
		}
	} else {
		diffOutput.WriteString(fmt.Sprintf("\nInserted at line %d:\n", startLine))
		for i, line := range newLines {
			diffOutput.WriteString(fmt.Sprintf("+%d: %s\n", startLine+i, line))
		}
	}

	// Add summary
	oldLineCount := len(lines)
	newLineCount := len(result)
	diffOutput.WriteString(fmt.Sprintf("\nFile line count: %d → %d (%+d lines)\n", 
		oldLineCount, newLineCount, newLineCount-oldLineCount))

	return diffOutput.String(), nil
}