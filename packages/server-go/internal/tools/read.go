package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rohanthewiz/serr"
	"github.com/sst/opencode/server-go/internal/schema"
	"github.com/sst/opencode/server-go/internal/tool"
)

// ReadTool implements file reading functionality.
// It reads files with line numbers and supports offset/limit for large files.
type ReadTool struct {
	description string
}

// NewReadTool creates a new read tool instance
func NewReadTool() *ReadTool {
	return &ReadTool{
		description: `Reads a file from the local filesystem with line numbers.
Supports reading entire files or specific sections using offset and limit parameters.
Returns content in a format similar to 'cat -n' with line numbers.`,
	}
}

func (t *ReadTool) ID() string {
	return "read"
}

func (t *ReadTool) Description() string {
	return t.description
}

func (t *ReadTool) Parameters() tool.Schema {
	return schema.Object(map[string]tool.Schema{
		"filePath": schema.String().Describe("The path to the file to read"),
		"offset":   schema.Optional(schema.Number().Describe("The line number to start reading from (1-based)")),
		"limit":    schema.Optional(schema.Number().Describe("The maximum number of lines to read")),
	}, "filePath")
}

func (t *ReadTool) Execute(ctx tool.Context, params map[string]any) (tool.Result, error) {
	// Extract parameters
	filePath, _ := params["filePath"].(string)
	
	// Handle optional offset and limit
	offset := 1
	if offsetVal, ok := params["offset"].(float64); ok {
		offset = int(offsetVal)
		if offset < 1 {
			offset = 1
		}
	}
	
	limit := -1 // -1 means no limit
	if limitVal, ok := params["limit"].(float64); ok {
		limit = int(limitVal)
	}
	
	// Clean the file path
	cleanPath := filepath.Clean(filePath)
	
	// Check if file exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return tool.Result{}, serr.New("file not found: %s", cleanPath)
		}
		return tool.Result{}, serr.Wrap(err, "failed to stat file")
	}
	
	// Check if it's a directory
	if info.IsDir() {
		return tool.Result{}, serr.New("path is a directory, not a file: %s", cleanPath)
	}
	
	// Open the file
	file, err := os.Open(cleanPath)
	if err != nil {
		return tool.Result{}, serr.Wrap(err, "failed to open file")
	}
	defer file.Close()
	
	// Read the file with line numbers
	scanner := bufio.NewScanner(file)
	var lines []string
	lineNum := 1
	linesRead := 0
	
	// Skip lines before offset
	for lineNum < offset && scanner.Scan() {
		lineNum++
	}
	
	// Read lines from offset up to limit
	for scanner.Scan() {
		if limit > 0 && linesRead >= limit {
			break
		}
		
		line := scanner.Text()
		// Format with line number (similar to cat -n)
		lines = append(lines, fmt.Sprintf("%6d\t%s", lineNum, line))
		
		lineNum++
		linesRead++
	}
	
	if err := scanner.Err(); err != nil {
		return tool.Result{}, serr.Wrap(err, "error reading file")
	}
	
	// Join lines for output
	output := strings.Join(lines, "\n")
	
	// Create preview for metadata (first 100 chars or first line)
	preview := ""
	if len(lines) > 0 {
		firstLine := lines[0]
		// Remove line number prefix for preview
		parts := strings.SplitN(firstLine, "\t", 2)
		if len(parts) > 1 {
			preview = parts[1]
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
		}
	}
	
	// Get relative path for display
	cwd, _ := os.Getwd()
	relativePath, _ := filepath.Rel(cwd, cleanPath)
	if relativePath == "" || strings.HasPrefix(relativePath, "..") {
		relativePath = cleanPath
	}
	
	// Send metadata update
	ctx.Metadata(map[string]any{
		"title":   relativePath,
		"preview": preview,
		"lines":   linesRead,
		"offset":  offset,
	})
	
	return tool.Result{
		Output: output,
		Metadata: map[string]any{
			"title":      relativePath,
			"preview":    preview,
			"totalLines": linesRead,
			"offset":     offset,
			"fileSize":   info.Size(),
		},
	}, nil
}