package tools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rohanthewiz/serr"
	"rcode/context"
)

// ContextAwareExecutor wraps tool execution with context awareness
type ContextAwareExecutor struct {
	registry       *Registry
	contextManager *context.Manager
}

// NewContextAwareExecutor creates a new context-aware executor
func NewContextAwareExecutor(registry *Registry, contextManager *context.Manager) *ContextAwareExecutor {
	return &ContextAwareExecutor{
		registry:       registry,
		contextManager: contextManager,
	}
}

// Execute runs a tool with context awareness
func (e *ContextAwareExecutor) Execute(toolUse ToolUse) (*ToolResult, error) {
	// Pre-execution context updates
	e.preExecute(toolUse)
	
	// Execute the tool
	result, err := e.registry.Execute(toolUse)
	
	// Post-execution context updates
	e.postExecute(toolUse, result, err)
	
	return result, err
}

// preExecute performs context updates before tool execution
func (e *ContextAwareExecutor) preExecute(toolUse ToolUse) {
	if e.contextManager == nil {
		return
	}

	// Track tool usage patterns
	switch toolUse.Name {
	case "read_file":
		if path, ok := GetString(toolUse.Input, "path"); ok {
			e.contextManager.AddRecentFile(path)
		}
	case "search":
		// Could track search patterns for better future suggestions
	}
}

// postExecute performs context updates after tool execution
func (e *ContextAwareExecutor) postExecute(toolUse ToolUse, result *ToolResult, err error) {
	if e.contextManager == nil || err != nil {
		return
	}

	// Track file changes with detailed information
	switch toolUse.Name {
	case "write_file":
		if path, ok := GetString(toolUse.Input, "path"); ok {
			details := make(map[string]interface{})
			if content, ok := GetString(toolUse.Input, "content"); ok {
				details["size"] = len(content)
				details["lines"] = countLines(content)
			}
			
			change := context.FileChange{
				Path:    path,
				Type:    context.ChangeTypeCreate,
				Tool:    toolUse.Name,
				Details: details,
			}
			e.contextManager.TrackChangeWithDetails(change)
			e.contextManager.AddRecentFile(path)
		}
		
	case "edit_file":
		if path, ok := GetString(toolUse.Input, "path"); ok {
			details := make(map[string]interface{})
			
			// Extract edit details
			if edits, ok := toolUse.Input["edits"].([]interface{}); ok {
				details["edit_count"] = len(edits)
				details["operations"] = extractEditOperations(edits)
			} else if editType, ok := GetString(toolUse.Input, "edit_type"); ok {
				details["edit_type"] = editType
			}
			
			change := context.FileChange{
				Path:    path,
				Type:    context.ChangeTypeModify,
				Tool:    toolUse.Name,
				Details: details,
			}
			e.contextManager.TrackChangeWithDetails(change)
			e.contextManager.AddRecentFile(path)
		}
		
	case "remove":
		if path, ok := GetString(toolUse.Input, "path"); ok {
			details := make(map[string]interface{})
			if recursive, ok := toolUse.Input["recursive"].(bool); ok {
				details["recursive"] = recursive
			}
			
			change := context.FileChange{
				Path:    path,
				Type:    context.ChangeTypeDelete,
				Tool:    toolUse.Name,
				Details: details,
			}
			e.contextManager.TrackChangeWithDetails(change)
		}
		
	case "move":
		if source, ok := GetString(toolUse.Input, "source"); ok {
			if dest, ok := GetString(toolUse.Input, "destination"); ok {
				details := make(map[string]interface{})
				details["destination"] = dest
				
				change := context.FileChange{
					Path:    dest,
					OldPath: source,
					Type:    context.ChangeTypeRename,
					Tool:    toolUse.Name,
					Details: details,
				}
				e.contextManager.TrackChangeWithDetails(change)
				e.contextManager.AddRecentFile(dest)
			}
		}
		
	case "make_dir":
		if path, ok := GetString(toolUse.Input, "path"); ok {
			details := make(map[string]interface{})
			if parents, ok := toolUse.Input["parents"].(bool); ok {
				details["create_parents"] = parents
			}
			
			change := context.FileChange{
				Path:    path,
				Type:    context.ChangeTypeCreate,
				Tool:    toolUse.Name,
				Details: details,
			}
			e.contextManager.TrackChangeWithDetails(change)
		}
		
	case "git_add", "git_commit", "git_push", "git_pull", "git_merge":
		// Track git operations
		details := make(map[string]interface{})
		details["command"] = toolUse.Name
		
		// Extract relevant parameters
		for key, value := range toolUse.Input {
			if key == "files" || key == "message" || key == "branch" || key == "remote" {
				details[key] = value
			}
		}
		
		// For git operations, track as a special change on the repository
		change := context.FileChange{
			Path:    ".git",
			Type:    context.ChangeTypeModify,
			Tool:    toolUse.Name,
			Details: details,
		}
		e.contextManager.TrackChangeWithDetails(change)
	}
}

// SuggestTools suggests tools based on the current context and task
func (e *ContextAwareExecutor) SuggestTools(task string) []ToolSuggestion {
	suggestions := make([]ToolSuggestion, 0)
	
	taskLower := strings.ToLower(task)
	
	// Basic pattern matching for tool suggestions
	patterns := []struct {
		keywords []string
		tool     string
		reason   string
	}{
		{
			keywords: []string{"read", "view", "show", "display"},
			tool:     "read_file",
			reason:   "Task involves reading or viewing files",
		},
		{
			keywords: []string{"create", "write", "new file"},
			tool:     "write_file",
			reason:   "Task involves creating new files",
		},
		{
			keywords: []string{"edit", "modify", "change", "update"},
			tool:     "edit_file",
			reason:   "Task involves editing existing files",
		},
		{
			keywords: []string{"search", "find", "grep", "look for"},
			tool:     "search",
			reason:   "Task involves searching for content",
		},
		{
			keywords: []string{"list", "directory", "files in"},
			tool:     "list_dir",
			reason:   "Task involves listing directory contents",
		},
		{
			keywords: []string{"structure", "tree", "hierarchy"},
			tool:     "tree",
			reason:   "Task involves viewing directory structure",
		},
		{
			keywords: []string{"run", "execute", "command", "test"},
			tool:     "bash",
			reason:   "Task involves running commands",
		},
		{
			keywords: []string{"git", "commit", "diff", "status"},
			tool:     "git_status",
			reason:   "Task involves git operations",
		},
	}
	
	// Check each pattern
	for _, pattern := range patterns {
		for _, keyword := range pattern.keywords {
			if strings.Contains(taskLower, keyword) {
				suggestions = append(suggestions, ToolSuggestion{
					Tool:     pattern.tool,
					Reason:   pattern.reason,
					Priority: calculatePriority(pattern.tool, task),
				})
				break
			}
		}
	}
	
	// Sort by priority
	for i := 0; i < len(suggestions)-1; i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[j].Priority > suggestions[i].Priority {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}
	
	return suggestions
}

// EnhanceToolParams enhances tool parameters with context information
func (e *ContextAwareExecutor) EnhanceToolParams(toolName string, params map[string]interface{}) map[string]interface{} {
	if e.contextManager == nil || !e.contextManager.IsInitialized() {
		return params
	}
	
	enhanced := make(map[string]interface{})
	for k, v := range params {
		enhanced[k] = v
	}
	
	ctx := e.contextManager.GetContext()
	if ctx == nil {
		return enhanced
	}
	
	// Enhance parameters based on tool and context
	switch toolName {
	case "search":
		// If no path specified, use project root
		if _, hasPath := enhanced["path"]; !hasPath {
			enhanced["path"] = ctx.RootPath
		}
		
		// Add file pattern based on language
		if _, hasPattern := enhanced["file_pattern"]; !hasPattern {
			switch ctx.Language {
			case "go":
				enhanced["file_pattern"] = "*.go"
			case "javascript", "typescript":
				enhanced["file_pattern"] = "*.{js,jsx,ts,tsx}"
			case "python":
				enhanced["file_pattern"] = "*.py"
			}
		}
		
	case "bash":
		// Add context-aware command suggestions
		if cmd, ok := enhanced["command"].(string); ok {
			enhanced["command"] = e.enhanceCommand(cmd, ctx)
		}
	}
	
	return enhanced
}

// enhanceCommand enhances a bash command with context-aware replacements
func (e *ContextAwareExecutor) enhanceCommand(cmd string, ctx *context.ProjectContext) string {
	// Replace common placeholders
	replacements := map[string]string{
		"${PROJECT_ROOT}": ctx.RootPath,
		"${LANGUAGE}":     ctx.Language,
		"${FRAMEWORK}":    ctx.Framework,
	}
	
	// Language-specific replacements
	switch ctx.Language {
	case "go":
		replacements["${TEST_CMD}"] = "go test ./..."
		replacements["${BUILD_CMD}"] = "go build"
		replacements["${RUN_CMD}"] = "go run ."
	case "javascript", "typescript":
		replacements["${TEST_CMD}"] = "npm test"
		replacements["${BUILD_CMD}"] = "npm run build"
		replacements["${RUN_CMD}"] = "npm start"
	case "python":
		replacements["${TEST_CMD}"] = "pytest"
		replacements["${BUILD_CMD}"] = "python setup.py build"
		replacements["${RUN_CMD}"] = "python main.py"
	}
	
	// Apply replacements
	result := cmd
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	
	return result
}

// GetContextualHelp provides context-aware help for tools
func (e *ContextAwareExecutor) GetContextualHelp(toolName string) string {
	if e.contextManager == nil || !e.contextManager.IsInitialized() {
		return ""
	}
	
	ctx := e.contextManager.GetContext()
	if ctx == nil {
		return ""
	}
	
	var help strings.Builder
	
	switch toolName {
	case "search":
		help.WriteString(fmt.Sprintf("Search in %s project", ctx.Language))
		if ctx.Framework != "" {
			help.WriteString(fmt.Sprintf(" (%s)", ctx.Framework))
		}
		help.WriteString("\nCommon patterns:\n")
		
		switch ctx.Language {
		case "go":
			help.WriteString("- Function definitions: `func\\s+\\w+`\n")
			help.WriteString("- Struct definitions: `type\\s+\\w+\\s+struct`\n")
			help.WriteString("- Imports: `import\\s+\"`\n")
		case "javascript", "typescript":
			help.WriteString("- Function definitions: `function\\s+\\w+`\n")
			help.WriteString("- Class definitions: `class\\s+\\w+`\n")
			help.WriteString("- Imports: `import\\s+.*from`\n")
		case "python":
			help.WriteString("- Function definitions: `def\\s+\\w+`\n")
			help.WriteString("- Class definitions: `class\\s+\\w+`\n")
			help.WriteString("- Imports: `import\\s+\\w+`\n")
		}
		
	case "bash":
		help.WriteString("Available context variables:\n")
		help.WriteString("- ${PROJECT_ROOT}: Project root directory\n")
		help.WriteString("- ${LANGUAGE}: Detected language\n")
		help.WriteString("- ${TEST_CMD}: Language-specific test command\n")
		help.WriteString("- ${BUILD_CMD}: Language-specific build command\n")
	}
	
	return help.String()
}

// ValidateToolUse validates tool usage in the current context
func (e *ContextAwareExecutor) ValidateToolUse(toolUse ToolUse) error {
	if e.contextManager == nil || !e.contextManager.IsInitialized() {
		return nil // No context, no validation
	}
	
	// Validate file paths are within project
	if path, ok := GetString(toolUse.Input, "path"); ok {
		if !e.isPathInProject(path) {
			return serr.New(fmt.Sprintf("path '%s' is outside project scope", path))
		}
	}
	
	// Additional tool-specific validations
	switch toolUse.Name {
	case "remove":
		if path, ok := GetString(toolUse.Input, "path"); ok {
			if e.isCriticalFile(path) {
				return serr.New(fmt.Sprintf("cannot remove critical file: %s", path))
			}
		}
	}
	
	return nil
}

// isPathInProject checks if a path is within the project
func (e *ContextAwareExecutor) isPathInProject(path string) bool {
	ctx := e.contextManager.GetContext()
	if ctx == nil {
		return true // No context, allow all
	}
	
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	
	return strings.HasPrefix(absPath, ctx.RootPath)
}

// isCriticalFile checks if a file is critical and shouldn't be removed
func (e *ContextAwareExecutor) isCriticalFile(path string) bool {
	basename := filepath.Base(path)
	criticalFiles := []string{
		"go.mod", "go.sum", "package.json", "package-lock.json",
		"requirements.txt", "Cargo.toml", "pom.xml", "build.gradle",
		".git", ".gitignore", "README.md", "LICENSE",
	}
	
	for _, critical := range criticalFiles {
		if basename == critical {
			return true
		}
	}
	
	return false
}

// ToolSuggestion represents a suggested tool for a task
type ToolSuggestion struct {
	Tool     string
	Reason   string
	Priority int
}

// calculatePriority calculates priority for a tool suggestion
func calculatePriority(tool, task string) int {
	priority := 50 // Base priority
	
	// Boost priority for exact tool mentions
	if strings.Contains(strings.ToLower(task), tool) {
		priority += 30
	}
	
	// Tool-specific priority adjustments
	switch tool {
	case "read_file":
		if strings.Contains(task, "understand") || strings.Contains(task, "analyze") {
			priority += 10
		}
	case "edit_file":
		if strings.Contains(task, "fix") || strings.Contains(task, "update") {
			priority += 15
		}
	case "search":
		if strings.Contains(task, "find") || strings.Contains(task, "locate") {
			priority += 20
		}
	}
	
	return priority
}

// countLines counts the number of lines in a string
func countLines(s string) int {
	if s == "" {
		return 0
	}
	lines := 1
	for _, ch := range s {
		if ch == '\n' {
			lines++
		}
	}
	return lines
}

// extractEditOperations extracts operation types from edit list
func extractEditOperations(edits []interface{}) []string {
	operations := make([]string, 0)
	operationMap := make(map[string]bool)
	
	for _, edit := range edits {
		if editMap, ok := edit.(map[string]interface{}); ok {
			if editType, ok := GetString(editMap, "edit_type"); ok {
				if !operationMap[editType] {
					operations = append(operations, editType)
					operationMap[editType] = true
				}
			}
		}
	}
	
	return operations
}