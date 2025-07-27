//go:build ignore

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"rcode/tools"
	"strings"
	"unicode"
)

// Tool is the exported symbol that RCode looks for
var Tool WordCountTool

// Metadata provides information about this plugin
var Metadata = &tools.PluginMetadata{
	Name:            "word_count",
	Version:         "1.0.0",
	Author:          "RCode Team",
	Description:     "Count words, lines, and characters in files",
	MinRCodeVersion: "0.1.0",
}

// WordCountTool implements the ToolPlugin interface
type WordCountTool struct {
	config map[string]interface{}
}

// GetDefinition returns the tool metadata
func (t WordCountTool) GetDefinition() tools.Tool {
	return tools.Tool{
		Name:        "word_count",
		Description: "Count words, lines, and characters in text files",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to analyze",
				},
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Optional glob pattern for multiple files (e.g., '*.txt')",
				},
				"include_stats": map[string]interface{}{
					"type":        "boolean",
					"description": "Include detailed statistics (avg word length, etc.)",
					"default":     false,
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs the tool
func (t WordCountTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	// Extract parameters
	path, _ := input["path"].(string)
	pattern, _ := input["pattern"].(string)
	includeStats, _ := input["include_stats"].(bool)

	// Validate input
	if path == "" && pattern == "" {
		return "", fmt.Errorf("either 'path' or 'pattern' parameter is required")
	}

	// Check context for cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Determine files to process
	var files []string
	if pattern != "" {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return "", fmt.Errorf("invalid glob pattern: %w", err)
		}
		files = matches
	} else {
		files = []string{path}
	}

	if len(files) == 0 {
		return "No files found matching the criteria", nil
	}

	// Process files
	var totalWords, totalLines, totalChars int
	var results []string

	for _, file := range files {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		stats, err := countFile(file)
		if err != nil {
			results = append(results, fmt.Sprintf("Error processing %s: %v", file, err))
			continue
		}

		totalWords += stats.Words
		totalLines += stats.Lines
		totalChars += stats.Characters

		result := fmt.Sprintf("%s: %d words, %d lines, %d characters",
			filepath.Base(file), stats.Words, stats.Lines, stats.Characters)

		if includeStats && stats.Words > 0 {
			avgWordLen := float64(stats.Characters-stats.Lines) / float64(stats.Words)
			result += fmt.Sprintf(" (avg word length: %.1f)", avgWordLen)
		}

		results = append(results, result)
	}

	// Build output
	output := strings.Join(results, "\n")

	if len(files) > 1 {
		output += fmt.Sprintf("\n\nTotal: %d words, %d lines, %d characters across %d files",
			totalWords, totalLines, totalChars, len(files))
	}

	return output, nil
}

// FileStats holds counting statistics for a file
type FileStats struct {
	Words      int
	Lines      int
	Characters int
}

// countFile counts words, lines, and characters in a file
func countFile(path string) (*FileStats, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stats := &FileStats{}
	reader := bufio.NewReader(file)
	inWord := false

	for {
		r, size, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		stats.Characters += size

		if r == '\n' {
			stats.Lines++
		}

		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			stats.Words++
		}
	}

	// Count last line if file doesn't end with newline
	if stats.Characters > 0 && stats.Lines == 0 {
		stats.Lines = 1
	}

	return stats, nil
}

// Initialize sets up the tool
func (t *WordCountTool) Initialize(config map[string]interface{}) error {
	t.config = config
	return nil
}

// Cleanup cleans up resources
func (t WordCountTool) Cleanup() error {
	// No resources to clean up
	return nil
}

// GetCapabilities returns what this tool can do
func (t WordCountTool) GetCapabilities() tools.ToolCapabilities {
	return tools.ToolCapabilities{
		FileRead:      true,  // This tool reads files
		FileWrite:     false, // This tool doesn't write files
		NetworkAccess: false, // This tool doesn't access the network
		ProcessSpawn:  false, // This tool doesn't spawn processes
		WorkingDir:    "",    // No working directory restriction
	}
}
