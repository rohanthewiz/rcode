package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rohanthewiz/serr"
)

// RipgrepTool implements high-performance file search using ripgrep
// This tool provides multiple output modes for token efficiency:
// - files_only: Just file paths (minimal tokens)
// - count: Match counts per file  
// - content: Matches with configurable context
// - json: Structured data for parsing
type RipgrepTool struct{}

// RipgrepMatch represents a single match in JSON output mode
type RipgrepMatch struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		Lines struct {
			Text string `json:"text"`
		} `json:"lines"`
		LineNumber   int `json:"line_number"`
		AbsoluteOffset int `json:"absolute_offset"`
		Submatches []struct {
			Match struct {
				Text string `json:"text"`
			} `json:"match"`
			Start int `json:"start"`
			End   int `json:"end"`
		} `json:"submatches"`
	} `json:"data"`
}

// GetDefinition returns the tool definition for the AI
func (t *RipgrepTool) GetDefinition() Tool {
	return Tool{
		Name:        "ripgrep",
		Description: "High-performance file search using ripgrep. Offers multiple output modes for token efficiency.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Regular expression pattern to search for",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File or directory path to search in (defaults to current directory)",
				},
				"output_mode": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"files_only", "count", "content", "json"},
					"description": "Output mode: files_only (just paths), count (match counts), content (with context), json (structured)",
					"default":     "files_only",
				},
				"file_type": map[string]interface{}{
					"type":        "string",
					"description": "Limit search to file type (e.g., 'go', 'js', 'py', 'rust', 'java')",
				},
				"glob": map[string]interface{}{
					"type":        "string",
					"description": "Glob pattern to filter files (e.g., '*.test.go', 'src/**/*.js')",
				},
				"case_sensitive": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether the search should be case-sensitive (default: true)",
					"default":     true,
				},
				"context_lines": map[string]interface{}{
					"type":        "integer",
					"description": "Number of context lines before and after matches (only for content mode, default: 2)",
					"default":     2,
					"minimum":     0,
					"maximum":     10,
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (default: 100)",
					"default":     100,
				},
				"include_hidden": map[string]interface{}{
					"type":        "boolean",
					"description": "Include hidden files and directories (default: false)",
					"default":     false,
				},
				"follow_symlinks": map[string]interface{}{
					"type":        "boolean",
					"description": "Follow symbolic links (default: false)",
					"default":     false,
				},
				"multiline": map[string]interface{}{
					"type":        "boolean",
					"description": "Enable multiline mode where . matches newlines (default: false)",
					"default":     false,
				},
			},
			"required": []string{"pattern"},
		},
	}
}

// Execute performs the ripgrep search operation
func (t *RipgrepTool) Execute(input map[string]interface{}) (string, error) {
	// Check if ripgrep is available
	if _, err := exec.LookPath("rg"); err != nil {
		// Fallback message with helpful instructions
		return "", NewPermanentError(
			serr.New("ripgrep (rg) is not installed. Please install it for optimal search performance. " +
				"On macOS: brew install ripgrep, On Ubuntu: apt-get install ripgrep"),
			"ripgrep not found",
		)
	}

	// Extract and validate parameters
	pattern, ok := GetString(input, "pattern")
	if !ok || pattern == "" {
		return "", serr.New("pattern is required")
	}

	searchPath, _ := GetString(input, "path")
	if searchPath == "" {
		searchPath = "."
	}

	// Expand the path to handle ~ for home directory
	expandedPath, err := ExpandPath(searchPath)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand path")
	}
	searchPath = expandedPath

	outputMode, _ := GetString(input, "output_mode")
	if outputMode == "" {
		outputMode = "files_only"
	}

	// Build ripgrep command arguments
	args := t.buildArgs(input, pattern, searchPath, outputMode)

	// Create context with timeout (30 seconds max for ripgrep)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute ripgrep
	cmd := exec.CommandContext(ctx, "rg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	
	// Handle timeout
	if ctx.Err() == context.DeadlineExceeded {
		return "", NewRetryableError(serr.New("ripgrep search timed out after 30 seconds"), "timeout")
	}

	// Check for errors (exit code 1 means no matches, which is not an error)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// No matches found
				return t.formatNoMatches(pattern, searchPath, outputMode), nil
			}
			// Other error
			if stderr.Len() > 0 {
				return "", serr.New(fmt.Sprintf("ripgrep error: %s", stderr.String()))
			}
		}
		return "", serr.Wrap(err, "failed to execute ripgrep")
	}

	// Format output based on mode
	output := stdout.String()
	return t.formatOutput(output, pattern, searchPath, outputMode)
}

// buildArgs constructs the ripgrep command arguments based on input parameters
func (t *RipgrepTool) buildArgs(input map[string]interface{}, pattern, searchPath, outputMode string) []string {
	args := []string{}

	// Set output mode specific flags
	switch outputMode {
	case "files_only":
		args = append(args, "--files-with-matches")
	case "count":
		args = append(args, "--count")
	case "json":
		args = append(args, "--json")
	case "content":
		// Default behavior, no special flag needed
		args = append(args, "--heading", "--line-number")
		
		// Add context lines if specified
		if contextLines, ok := GetInt(input, "context_lines"); ok && contextLines > 0 {
			args = append(args, "-C", strconv.Itoa(contextLines))
		} else {
			args = append(args, "-C", "2") // Default context
		}
	}

	// Case sensitivity
	caseSensitive := true
	if val, exists := input["case_sensitive"]; exists {
		if boolVal, ok := val.(bool); ok {
			caseSensitive = boolVal
		}
	}
	if !caseSensitive {
		args = append(args, "--ignore-case")
	}

	// File type filter
	if fileType, ok := GetString(input, "file_type"); ok && fileType != "" {
		args = append(args, "--type", fileType)
	}

	// Glob pattern filter
	if glob, ok := GetString(input, "glob"); ok && glob != "" {
		args = append(args, "--glob", glob)
	}

	// Max results
	maxResults := 100
	if val, ok := GetInt(input, "max_results"); ok && val > 0 {
		maxResults = val
	}
	args = append(args, "--max-count", strconv.Itoa(maxResults))

	// Include hidden files
	if includeHidden, ok := input["include_hidden"].(bool); ok && includeHidden {
		args = append(args, "--hidden")
	}

	// Follow symlinks
	if followSymlinks, ok := input["follow_symlinks"].(bool); ok && followSymlinks {
		args = append(args, "--follow")
	}

	// Multiline mode
	if multiline, ok := input["multiline"].(bool); ok && multiline {
		args = append(args, "--multiline", "--multiline-dotall")
	}

	// Add pattern and path
	args = append(args, pattern)
	args = append(args, searchPath)

	return args
}

// formatOutput formats the ripgrep output based on the output mode
func (t *RipgrepTool) formatOutput(output, pattern, searchPath, outputMode string) (string, error) {
	if output == "" {
		return t.formatNoMatches(pattern, searchPath, outputMode), nil
	}

	var result strings.Builder
	lines := strings.Split(strings.TrimSpace(output), "\n")

	switch outputMode {
	case "files_only":
		result.WriteString(fmt.Sprintf("Files containing pattern '%s':\n", pattern))
		result.WriteString(fmt.Sprintf("Found %d files\n\n", len(lines)))
		for _, line := range lines {
			result.WriteString(fmt.Sprintf("  %s\n", line))
		}

	case "count":
		result.WriteString(fmt.Sprintf("Match counts for pattern '%s':\n", pattern))
		totalMatches := 0
		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				if count, err := strconv.Atoi(parts[1]); err == nil {
					totalMatches += count
				}
				result.WriteString(fmt.Sprintf("  %s: %s matches\n", parts[0], parts[1]))
			}
		}
		result.WriteString(fmt.Sprintf("\nTotal: %d matches across %d files\n", totalMatches, len(lines)))

	case "json":
		// Parse and format JSON output
		result.WriteString(fmt.Sprintf("JSON search results for pattern '%s':\n\n", pattern))
		matchCount := 0
		fileSet := make(map[string]bool)
		
		for _, line := range lines {
			var match RipgrepMatch
			if err := json.Unmarshal([]byte(line), &match); err == nil && match.Type == "match" {
				matchCount++
				fileSet[match.Data.Path.Text] = true
			}
		}
		
		result.WriteString(fmt.Sprintf("Found %d matches in %d files\n", matchCount, len(fileSet)))
		result.WriteString("\nRaw JSON output:\n")
		result.WriteString(output)

	case "content":
		result.WriteString(fmt.Sprintf("Search results for pattern '%s':\n", pattern))
		result.WriteString(fmt.Sprintf("Path: %s\n\n", searchPath))
		result.WriteString(output)
		
		// Add summary at the end
		fileCount := 0
		for _, line := range lines {
			if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "-") {
				fileCount++
			}
		}
		if fileCount > 0 {
			result.WriteString(fmt.Sprintf("\n\nFound matches in %d file(s)\n", fileCount))
		}
	}

	return result.String(), nil
}

// formatNoMatches returns a formatted message when no matches are found
func (t *RipgrepTool) formatNoMatches(pattern, searchPath, outputMode string) string {
	switch outputMode {
	case "files_only":
		return fmt.Sprintf("No files found containing pattern '%s' in %s\n", pattern, searchPath)
	case "count":
		return fmt.Sprintf("No matches found for pattern '%s' in %s\n", pattern, searchPath)
	case "json":
		return fmt.Sprintf("No matches found for pattern '%s' in %s (JSON mode)\n", pattern, searchPath)
	default:
		return fmt.Sprintf("No matches found for pattern '%s' in %s\n", pattern, searchPath)
	}
}