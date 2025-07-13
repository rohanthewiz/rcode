package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/rohanthewiz/serr"
)

// SearchTool implements file search functionality with regex support
type SearchTool struct{}

// SearchResult represents a single search match
type SearchResult struct {
	File       string
	Line       int
	Column     int
	Match      string
	Context    string
}

// GetDefinition returns the tool definition for the AI
func (t *SearchTool) GetDefinition() Tool {
	return Tool{
		Name:        "search",
		Description: "Search for patterns in files using regular expressions. Searches recursively in directories.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File or directory path to search in. If directory, searches recursively.",
				},
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Regular expression pattern to search for",
				},
				"file_pattern": map[string]interface{}{
					"type":        "string",
					"description": "Optional glob pattern to filter files (e.g., '*.go', '*.js')",
				},
				"case_sensitive": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether the search should be case-sensitive (default: true)",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (default: 100)",
				},
				"context_lines": map[string]interface{}{
					"type":        "integer",
					"description": "Number of context lines to show before and after match (default: 2)",
				},
			},
			"required": []string{"path", "pattern"},
		},
	}
}

// Execute performs the search operation
func (t *SearchTool) Execute(input map[string]interface{}) (string, error) {
	// Extract parameters
	searchPath, ok := GetString(input, "path")
	if !ok || searchPath == "" {
		return "", serr.New("path is required")
	}

	// Expand the path to handle ~ for home directory
	expandedPath, err := ExpandPath(searchPath)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand path")
	}
	searchPath = expandedPath

	pattern, ok := GetString(input, "pattern")
	if !ok || pattern == "" {
		return "", serr.New("pattern is required")
	}

	filePattern, _ := GetString(input, "file_pattern")
	
	caseSensitive := true
	if val, exists := input["case_sensitive"]; exists {
		if boolVal, ok := val.(bool); ok {
			caseSensitive = boolVal
		}
	}

	maxResults, ok := GetInt(input, "max_results")
	if !ok {
		maxResults = 100
	}

	contextLines, ok := GetInt(input, "context_lines")
	if !ok {
		contextLines = 2
	}

	// Compile regex
	regexFlags := ""
	if !caseSensitive {
		regexFlags = "(?i)"
	}
	regex, err := regexp.Compile(regexFlags + pattern)
	if err != nil {
		return "", serr.Wrap(err, "Invalid regex pattern")
	}

	// Check if path exists
	info, err := os.Stat(searchPath)
	if err != nil {
		return "", serr.Wrap(err, fmt.Sprintf("Cannot access path: %s", searchPath))
	}

	var results []SearchResult
	var searchedFiles int

	if info.IsDir() {
		// Search in directory
		err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			// Skip directories and binary files
			if info.IsDir() || isBinaryFile(path) {
				return nil
			}

			// Apply file pattern filter if specified
			if filePattern != "" {
				matched, _ := filepath.Match(filePattern, filepath.Base(path))
				if !matched {
					return nil
				}
			}

			// Search in file
			fileResults, err := searchInFile(path, regex, contextLines)
			if err != nil {
				return nil // Skip files we can't read
			}

			searchedFiles++
			results = append(results, fileResults...)

			// Stop if we have enough results
			if len(results) >= maxResults {
				return filepath.SkipAll
			}

			return nil
		})
		if err != nil && err != filepath.SkipAll {
			return "", serr.Wrap(err, "Error walking directory")
		}
	} else {
		// Search in single file
		results, err = searchInFile(searchPath, regex, contextLines)
		if err != nil {
			return "", serr.Wrap(err, "Error searching file")
		}
		searchedFiles = 1
	}

	// Limit results
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Search results for pattern: %s\n", pattern))
	output.WriteString(fmt.Sprintf("Searched %d files, found %d matches\n\n", searchedFiles, len(results)))

	if len(results) == 0 {
		output.WriteString("No matches found.\n")
		return output.String(), nil
	}

	// Group results by file
	fileGroups := make(map[string][]SearchResult)
	for _, result := range results {
		fileGroups[result.File] = append(fileGroups[result.File], result)
	}

	// Sort files
	var files []string
	for file := range fileGroups {
		files = append(files, file)
	}
	sort.Strings(files)

	// Display results
	for _, file := range files {
		relPath := file
		if rel, err := filepath.Rel(searchPath, file); err == nil {
			relPath = rel
		}
		
		output.WriteString(fmt.Sprintf("=== %s ===\n", relPath))
		
		for _, result := range fileGroups[file] {
			output.WriteString(fmt.Sprintf("  Line %d: %s\n", result.Line, result.Match))
			if result.Context != "" {
				output.WriteString(result.Context)
			}
			output.WriteString("\n")
		}
	}

	if len(results) == maxResults {
		output.WriteString(fmt.Sprintf("\n(Results limited to %d matches)\n", maxResults))
	}

	return output.String(), nil
}

// searchInFile searches for pattern in a single file
func searchInFile(path string, regex *regexp.Regexp, contextLines int) ([]SearchResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []SearchResult
	var lines []string
	scanner := bufio.NewScanner(file)
	
	// Read all lines first (for context)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Search for matches
	for i, line := range lines {
		matches := regex.FindAllStringIndex(line, -1)
		for _, match := range matches {
			result := SearchResult{
				File:   path,
				Line:   i + 1,
				Column: match[0] + 1,
				Match:  line[match[0]:match[1]],
			}

			// Add context
			if contextLines > 0 {
				var contextBuilder strings.Builder
				
				// Before context
				startLine := i - contextLines
				if startLine < 0 {
					startLine = 0
				}
				
				// After context
				endLine := i + contextLines
				if endLine >= len(lines) {
					endLine = len(lines) - 1
				}

				// Build context
				for j := startLine; j <= endLine; j++ {
					prefix := "    "
					if j == i {
						prefix = " >> "
					}
					contextBuilder.WriteString(fmt.Sprintf("  %s%d: %s\n", prefix, j+1, lines[j]))
				}
				
				result.Context = contextBuilder.String()
			}

			results = append(results, result)
		}
	}

	return results, nil
}

// isBinaryFile checks if a file is likely binary
func isBinaryFile(path string) bool {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".db": true, ".sqlite": true, ".bin": true, ".dat": true,
	}
	
	if binaryExts[ext] {
		return true
	}

	// Check file content (first 512 bytes)
	file, err := os.Open(path)
	if err != nil {
		return true // Assume binary if we can't read
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return true
	}

	// Check for null bytes
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}

	return false
}