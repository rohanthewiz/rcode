package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/rohanthewiz/serr"
)

// SmartEditTool provides multiple efficient file editing modes with token optimization
// Modes include: patch (unified diff), replace (pattern-based), sed (stream editor), line (line-based)
// Response modes: minimal, summary, diff, full
type SmartEditTool struct{}

// EditStats tracks statistics about the edit operation
type EditStats struct {
	LinesAdded    int
	LinesDeleted  int
	LinesModified int
	Replacements  int
	FilesAffected int
}

// GetDefinition returns the tool definition for the AI
func (t *SmartEditTool) GetDefinition() Tool {
	return Tool{
		Name:        "smart_edit",
		Description: "Efficiently edit files using multiple modes: patch (unified diff), replace (pattern-based), sed (commands), or line (line numbers). Optimized for minimal token usage.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to edit",
				},
				"mode": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"patch", "replace", "sed", "line"},
					"description": "Edit mode: patch (apply unified diff), replace (pattern replacement), sed (stream editor commands), line (line-based editing)",
					"default":     "replace",
				},
				"response_mode": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"minimal", "summary", "diff", "full"},
					"description": "Response detail level: minimal (5-10 tokens), summary (20-50 tokens), diff (50-200 tokens), full (complete before/after)",
					"default":     "minimal",
				},
				// Mode-specific parameters
				// For "patch" mode
				"diff": map[string]interface{}{
					"type":        "string",
					"description": "Unified diff to apply (for patch mode)",
				},
				// For "replace" mode
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Pattern to search for (regex supported in replace mode)",
				},
				"replacement": map[string]interface{}{
					"type":        "string",
					"description": "Replacement text (supports backreferences like $1, $2)",
				},
				"replace_all": map[string]interface{}{
					"type":        "boolean",
					"description": "Replace all occurrences (default: true)",
					"default":     true,
				},
				"case_sensitive": map[string]interface{}{
					"type":        "boolean",
					"description": "Case-sensitive matching (default: true)",
					"default":     true,
				},
				// For "sed" mode
				"commands": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Array of sed commands to execute (e.g., ['s/old/new/g', '/pattern/d'])",
				},
				// For "line" mode
				"start_line": map[string]interface{}{
					"type":        "integer",
					"description": "Starting line number for line mode",
				},
				"end_line": map[string]interface{}{
					"type":        "integer",
					"description": "Ending line number for line mode",
				},
				"new_content": map[string]interface{}{
					"type":        "string",
					"description": "New content for line mode",
				},
				"operation": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"replace", "insert_before", "insert_after", "delete"},
					"description": "Operation for line mode",
					"default":     "replace",
				},
				// Common options
				"dry_run": map[string]interface{}{
					"type":        "boolean",
					"description": "Preview changes without applying them",
					"default":     false,
				},
				"backup": map[string]interface{}{
					"type":        "boolean",
					"description": "Create backup before editing (.bak extension)",
					"default":     false,
				},
			},
			"required": []string{"path", "mode"},
		},
	}
}

// Execute performs the smart edit operation based on the selected mode
func (t *SmartEditTool) Execute(input map[string]interface{}) (string, error) {
	// Extract common parameters
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		return "", serr.New("path is required")
	}

	// Expand the path
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand path")
	}

	mode, ok := GetString(input, "mode")
	if !ok {
		mode = "replace"
	}

	responseMode, ok := GetString(input, "response_mode")
	if !ok {
		responseMode = "minimal"
	}

	dryRun := false
	if val, exists := input["dry_run"]; exists {
		if boolVal, ok := val.(bool); ok {
			dryRun = boolVal
		}
	}

	backup := false
	if val, exists := input["backup"]; exists {
		if boolVal, ok := val.(bool); ok {
			backup = boolVal
		}
	}

	// Create backup if requested
	if backup && !dryRun {
		if err := t.createBackup(expandedPath); err != nil {
			return "", serr.Wrap(err, "failed to create backup")
		}
	}

	// Store original content for comparison
	originalContent, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", NewPermanentError(serr.New(fmt.Sprintf("File not found: %s", path)), "file not found")
		}
		return "", WrapFileSystemError(serr.Wrap(err, "failed to read file"))
	}

	// Execute based on mode
	var stats EditStats
	var result string

	switch mode {
	case "patch":
		stats, err = t.applyPatchMode(expandedPath, input, dryRun)
	case "replace":
		stats, err = t.applyReplaceMode(expandedPath, input, dryRun)
	case "sed":
		stats, err = t.applySedMode(expandedPath, input, dryRun)
	case "line":
		stats, err = t.applyLineMode(expandedPath, input, dryRun)
	default:
		return "", serr.New(fmt.Sprintf("unknown mode: %s", mode))
	}

	if err != nil {
		return "", err
	}

	// Read new content for comparison (unless dry run)
	var newContent []byte
	if !dryRun {
		newContent, _ = os.ReadFile(expandedPath)
		// Notify file change
		NotifyFileChange(path, "modified")
	} else {
		newContent = originalContent // For dry run, content unchanged
	}

	// Format response based on response mode
	result = t.formatResponse(responseMode, path, originalContent, newContent, stats, dryRun)

	return result, nil
}

// applyPatchMode applies a unified diff patch to the file
func (t *SmartEditTool) applyPatchMode(path string, input map[string]interface{}, dryRun bool) (EditStats, error) {
	diff, ok := GetString(input, "diff")
	if !ok || diff == "" {
		return EditStats{}, serr.New("diff is required for patch mode")
	}

	// Create temporary file with the diff
	tmpFile, err := os.CreateTemp("", "patch-*.diff")
	if err != nil {
		return EditStats{}, serr.Wrap(err, "failed to create temp file")
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(diff); err != nil {
		tmpFile.Close()
		return EditStats{}, serr.Wrap(err, "failed to write diff")
	}
	tmpFile.Close()

	// Build patch command
	args := []string{"-u", path, "-i", tmpFile.Name()}
	if dryRun {
		args = append([]string{"--dry-run"}, args...)
	}

	// Apply patch
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "patch", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return EditStats{}, serr.New(fmt.Sprintf("patch failed: %s\nOutput: %s", err, output))
	}

	// Parse patch output for statistics
	stats := parsePatchOutput(string(output))
	stats.FilesAffected = 1

	return stats, nil
}

// applyReplaceMode applies pattern-based replacements
func (t *SmartEditTool) applyReplaceMode(path string, input map[string]interface{}, dryRun bool) (EditStats, error) {
	pattern, ok := GetString(input, "pattern")
	if !ok || pattern == "" {
		return EditStats{}, serr.New("pattern is required for replace mode")
	}

	replacement, ok := GetString(input, "replacement")
	if !ok {
		replacement = ""
	}

	replaceAll := true
	if val, exists := input["replace_all"]; exists {
		if boolVal, ok := val.(bool); ok {
			replaceAll = boolVal
		}
	}

	caseSensitive := true
	if val, exists := input["case_sensitive"]; exists {
		if boolVal, ok := val.(bool); ok {
			caseSensitive = boolVal
		}
	}

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return EditStats{}, serr.Wrap(err, "failed to read file")
	}

	// Compile regex
	regexPattern := pattern
	if !caseSensitive {
		regexPattern = "(?i)" + regexPattern
	}

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return EditStats{}, serr.Wrap(err, "invalid regex pattern")
	}

	// Perform replacement
	var newContent string
	replacements := 0

	if replaceAll {
		matches := re.FindAllStringIndex(string(content), -1)
		replacements = len(matches)
		newContent = re.ReplaceAllString(string(content), replacement)
	} else {
		// Replace only first occurrence
		loc := re.FindStringIndex(string(content))
		if loc != nil {
			replacements = 1
			newContent = string(content[:loc[0]]) + replacement + string(content[loc[1]:])
		} else {
			newContent = string(content)
		}
	}

	// Write back if not dry run
	if !dryRun && replacements > 0 {
		if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
			return EditStats{}, serr.Wrap(err, "failed to write file")
		}
	}

	// Calculate line changes
	oldLines := strings.Count(string(content), "\n")
	newLines := strings.Count(newContent, "\n")

	stats := EditStats{
		Replacements:  replacements,
		FilesAffected: 1,
		LinesModified: replacements, // Approximate
	}

	if newLines > oldLines {
		stats.LinesAdded = newLines - oldLines
	} else if oldLines > newLines {
		stats.LinesDeleted = oldLines - newLines
	}

	return stats, nil
}

// applySedMode applies sed commands to the file
func (t *SmartEditTool) applySedMode(path string, input map[string]interface{}, dryRun bool) (EditStats, error) {
	commands, ok := input["commands"].([]interface{})
	if !ok || len(commands) == 0 {
		return EditStats{}, serr.New("commands array is required for sed mode")
	}

	// Convert commands to strings
	sedCommands := make([]string, 0, len(commands))
	for _, cmd := range commands {
		if cmdStr, ok := cmd.(string); ok {
			sedCommands = append(sedCommands, cmdStr)
		}
	}

	if len(sedCommands) == 0 {
		return EditStats{}, serr.New("no valid sed commands provided")
	}

	// Build sed command
	args := []string{}
	if !dryRun {
		args = append(args, "-i", "") // In-place edit
	} else {
		args = append(args, "-n") // No output for dry run check
	}

	// Add each command
	for _, cmd := range sedCommands {
		args = append(args, "-e", cmd)
	}
	args = append(args, path)

	// Execute sed
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Store original for comparison
	originalContent, _ := os.ReadFile(path)

	cmd := exec.CommandContext(ctx, "sed", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return EditStats{}, serr.New(fmt.Sprintf("sed failed: %s\nOutput: %s", err, output))
	}

	// Calculate statistics
	stats := EditStats{FilesAffected: 1}

	if !dryRun {
		newContent, _ := os.ReadFile(path)
		stats = calculateEditStats(originalContent, newContent)
	}

	return stats, nil
}

// applyLineMode applies line-based edits (similar to original EditFileTool)
func (t *SmartEditTool) applyLineMode(path string, input map[string]interface{}, dryRun bool) (EditStats, error) {
	startLine, ok := GetInt(input, "start_line")
	if !ok || startLine < 1 {
		return EditStats{}, serr.New("start_line is required for line mode")
	}

	endLine, hasEndLine := GetInt(input, "end_line")
	if !hasEndLine {
		endLine = startLine
	}

	newContent, _ := GetString(input, "new_content")
	operation, _ := GetString(input, "operation")
	if operation == "" {
		operation = "replace"
	}

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return EditStats{}, serr.Wrap(err, "failed to read file")
	}

	// Split into lines
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Validate line numbers
	if startLine > len(lines) {
		return EditStats{}, serr.New(fmt.Sprintf("start_line %d exceeds file length %d", startLine, len(lines)))
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}

	// Prepare new content lines
	newLines := []string{}
	if newContent != "" {
		newLines = strings.Split(newContent, "\n")
		if len(newLines) > 0 && newLines[len(newLines)-1] == "" {
			newLines = newLines[:len(newLines)-1]
		}
	}

	// Apply operation
	var result []string
	stats := EditStats{FilesAffected: 1}

	switch operation {
	case "insert_before":
		result = append(result, lines[:startLine-1]...)
		result = append(result, newLines...)
		result = append(result, lines[startLine-1:]...)
		stats.LinesAdded = len(newLines)

	case "insert_after":
		result = append(result, lines[:endLine]...)
		result = append(result, newLines...)
		result = append(result, lines[endLine:]...)
		stats.LinesAdded = len(newLines)

	case "delete":
		result = append(result, lines[:startLine-1]...)
		result = append(result, lines[endLine:]...)
		stats.LinesDeleted = endLine - startLine + 1

	case "replace":
		fallthrough
	default:
		result = append(result, lines[:startLine-1]...)
		if newContent != "" {
			result = append(result, newLines...)
		}
		result = append(result, lines[endLine:]...)
		
		oldCount := endLine - startLine + 1
		newCount := len(newLines)
		if newCount > oldCount {
			stats.LinesAdded = newCount - oldCount
			stats.LinesModified = oldCount
		} else if oldCount > newCount {
			stats.LinesDeleted = oldCount - newCount
			stats.LinesModified = newCount
		} else {
			stats.LinesModified = newCount
		}
	}

	// Write back if not dry run
	if !dryRun {
		modifiedContent := strings.Join(result, "\n")
		if len(result) > 0 {
			modifiedContent += "\n"
		}

		if err := os.WriteFile(path, []byte(modifiedContent), 0644); err != nil {
			return EditStats{}, serr.Wrap(err, "failed to write file")
		}
	}

	return stats, nil
}

// formatResponse formats the output based on response mode
func (t *SmartEditTool) formatResponse(mode, path string, originalContent, newContent []byte, stats EditStats, dryRun bool) string {
	dryRunPrefix := ""
	if dryRun {
		dryRunPrefix = "[DRY RUN] "
	}

	switch mode {
	case "minimal":
		// Ultra-concise response (5-10 tokens)
		if stats.Replacements > 0 {
			return fmt.Sprintf("%s%d replacements", dryRunPrefix, stats.Replacements)
		}
		if stats.LinesAdded > 0 || stats.LinesDeleted > 0 {
			return fmt.Sprintf("%s+%d/-%d lines", dryRunPrefix, stats.LinesAdded, stats.LinesDeleted)
		}
		if stats.LinesModified > 0 {
			return fmt.Sprintf("%s%d lines modified", dryRunPrefix, stats.LinesModified)
		}
		return fmt.Sprintf("%sFile unchanged", dryRunPrefix)

	case "summary":
		// Brief summary (20-50 tokens)
		var summary strings.Builder
		summary.WriteString(fmt.Sprintf("%sEdited: %s\n", dryRunPrefix, path))
		
		if stats.Replacements > 0 {
			summary.WriteString(fmt.Sprintf("Replacements: %d\n", stats.Replacements))
		}
		if stats.LinesAdded > 0 {
			summary.WriteString(fmt.Sprintf("Lines added: %d\n", stats.LinesAdded))
		}
		if stats.LinesDeleted > 0 {
			summary.WriteString(fmt.Sprintf("Lines deleted: %d\n", stats.LinesDeleted))
		}
		if stats.LinesModified > 0 {
			summary.WriteString(fmt.Sprintf("Lines modified: %d\n", stats.LinesModified))
		}
		
		oldSize := len(originalContent)
		newSize := len(newContent)
		if oldSize != newSize {
			summary.WriteString(fmt.Sprintf("Size: %d → %d bytes (%+d)\n", oldSize, newSize, newSize-oldSize))
		}
		
		return summary.String()

	case "diff":
		// Show diff output (50-200 tokens)
		// Generate simple diff
		diff := generateSimpleDiff(string(originalContent), string(newContent))
		return fmt.Sprintf("%sFile: %s\n%s", dryRunPrefix, path, diff)

	case "full":
		// Complete before/after (current EditFileTool behavior)
		var output strings.Builder
		output.WriteString(fmt.Sprintf("%sFile edited: %s\n", dryRunPrefix, path))
		output.WriteString("\n--- Before:\n")
		output.WriteString(string(originalContent))
		output.WriteString("\n+++ After:\n")
		output.WriteString(string(newContent))
		output.WriteString(fmt.Sprintf("\n\nStatistics:\n"))
		output.WriteString(fmt.Sprintf("Lines: +%d/-%d, Modified: %d\n", 
			stats.LinesAdded, stats.LinesDeleted, stats.LinesModified))
		return output.String()

	default:
		return fmt.Sprintf("%sEdit completed", dryRunPrefix)
	}
}

// Helper functions

// createBackup creates a backup of the file with .bak extension
func (t *SmartEditTool) createBackup(path string) error {
	backupPath := path + ".bak"
	input, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(backupPath, input, 0644)
}

// parsePatchOutput parses patch command output for statistics
func parsePatchOutput(output string) EditStats {
	stats := EditStats{}
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		// Look for hunks applied
		if strings.Contains(line, "Hunk") && strings.Contains(line, "succeeded") {
			stats.LinesModified++
		}
		// Parse added/deleted lines from patch output
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			stats.LinesAdded++
		}
		if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			stats.LinesDeleted++
		}
	}
	
	return stats
}

// calculateEditStats calculates edit statistics by comparing content
func calculateEditStats(before, after []byte) EditStats {
	beforeLines := strings.Split(string(before), "\n")
	afterLines := strings.Split(string(after), "\n")
	
	stats := EditStats{
		FilesAffected: 1,
	}
	
	// Simple line count comparison
	if len(afterLines) > len(beforeLines) {
		stats.LinesAdded = len(afterLines) - len(beforeLines)
	} else if len(beforeLines) > len(afterLines) {
		stats.LinesDeleted = len(beforeLines) - len(afterLines)
	}
	
	// Count modified lines (simplified)
	minLen := len(beforeLines)
	if len(afterLines) < minLen {
		minLen = len(afterLines)
	}
	
	for i := 0; i < minLen; i++ {
		if beforeLines[i] != afterLines[i] {
			stats.LinesModified++
		}
	}
	
	return stats
}

// generateSimpleDiff generates a simple unified-style diff
func generateSimpleDiff(before, after string) string {
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")
	
	var diff strings.Builder
	diff.WriteString("@@ Changes @@\n")
	
	// Simple line-by-line comparison (not a full diff algorithm)
	maxLines := len(beforeLines)
	if len(afterLines) > maxLines {
		maxLines = len(afterLines)
	}
	
	const maxDiffLines = 20 // Limit diff output for token efficiency
	diffLines := 0
	
	for i := 0; i < maxLines && diffLines < maxDiffLines; i++ {
		beforeLine := ""
		afterLine := ""
		
		if i < len(beforeLines) {
			beforeLine = beforeLines[i]
		}
		if i < len(afterLines) {
			afterLine = afterLines[i]
		}
		
		if beforeLine != afterLine {
			if beforeLine != "" && afterLine == "" {
				diff.WriteString(fmt.Sprintf("-%d: %s\n", i+1, truncateLine(beforeLine, 80)))
				diffLines++
			} else if beforeLine == "" && afterLine != "" {
				diff.WriteString(fmt.Sprintf("+%d: %s\n", i+1, truncateLine(afterLine, 80)))
				diffLines++
			} else if beforeLine != afterLine {
				diff.WriteString(fmt.Sprintf("-%d: %s\n", i+1, truncateLine(beforeLine, 80)))
				diff.WriteString(fmt.Sprintf("+%d: %s\n", i+1, truncateLine(afterLine, 80)))
				diffLines += 2
			}
		}
	}
	
	if diffLines >= maxDiffLines {
		diff.WriteString("... (diff truncated for brevity)\n")
	}
	
	return diff.String()
}

// truncateLine truncates a line to max length for display
func truncateLine(line string, maxLen int) string {
	if len(line) <= maxLen {
		return line
	}
	return line[:maxLen-3] + "..."
}

