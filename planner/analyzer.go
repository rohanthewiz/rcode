package planner

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rohanthewiz/serr"
)

// TaskAnalyzer analyzes task descriptions and creates execution plans
type TaskAnalyzer struct {
	patterns []TaskPattern
}

// TaskPattern represents a pattern for task analysis
type TaskPattern struct {
	Keywords    []string
	ToolChain   []string
	Description string
}

// NewTaskAnalyzer creates a new task analyzer
func NewTaskAnalyzer() *TaskAnalyzer {
	return &TaskAnalyzer{
		patterns: initializePatterns(),
	}
}

// AnalyzeTask analyzes a task description and returns steps
func (a *TaskAnalyzer) AnalyzeTask(description string) ([]TaskStep, error) {
	description = strings.ToLower(description)
	
	// Try to match patterns
	for _, pattern := range a.patterns {
		if a.matchesPattern(description, pattern) {
			return a.createStepsFromPattern(description, pattern)
		}
	}

	// If no pattern matches, try to infer steps
	return a.inferSteps(description)
}

// matchesPattern checks if a description matches a pattern
func (a *TaskAnalyzer) matchesPattern(description string, pattern TaskPattern) bool {
	matchCount := 0
	for _, keyword := range pattern.Keywords {
		if strings.Contains(description, keyword) {
			matchCount++
		}
	}
	// Match if at least half of keywords are present
	return matchCount >= len(pattern.Keywords)/2
}

// createStepsFromPattern creates steps based on a pattern
func (a *TaskAnalyzer) createStepsFromPattern(description string, pattern TaskPattern) ([]TaskStep, error) {
	steps := make([]TaskStep, 0, len(pattern.ToolChain))

	for i, tool := range pattern.ToolChain {
		step := TaskStep{
			ID:          fmt.Sprintf("step_%d_%s", i+1, uuid.New().String()[:8]),
			Description: a.generateStepDescription(tool, description),
			Tool:        tool,
			Params:      a.inferParams(tool, description),
			Retryable:   true,
			MaxRetries:  3,
			Status:      StepStatusPending,
		}

		// Add dependencies
		if i > 0 && a.requiresDependency(tool) {
			step.Dependencies = []string{steps[i-1].ID}
		}

		steps = append(steps, step)
	}

	return steps, nil
}

// inferSteps tries to infer steps from the description
func (a *TaskAnalyzer) inferSteps(description string) ([]TaskStep, error) {
	steps := make([]TaskStep, 0)

	// Common task patterns
	if strings.Contains(description, "create") && strings.Contains(description, "file") {
		steps = append(steps, a.createFileStep(description))
	}

	if strings.Contains(description, "edit") || strings.Contains(description, "modify") {
		steps = append(steps, a.createEditStep(description))
	}

	if strings.Contains(description, "search") || strings.Contains(description, "find") {
		steps = append(steps, a.createSearchStep(description))
	}

	if strings.Contains(description, "test") {
		steps = append(steps, a.createTestStep(description))
	}

	if strings.Contains(description, "commit") || strings.Contains(description, "git") {
		steps = append(steps, a.createGitSteps(description)...)
	}

	if len(steps) == 0 {
		return nil, serr.New("unable to analyze task - no recognizable patterns found")
	}

	// Set IDs and dependencies
	for i := range steps {
		steps[i].ID = fmt.Sprintf("step_%d_%s", i+1, uuid.New().String()[:8])
		if i > 0 && steps[i].Tool != "git_status" {
			steps[i].Dependencies = []string{steps[i-1].ID}
		}
	}

	return steps, nil
}

// Step creation helpers

func (a *TaskAnalyzer) createFileStep(description string) TaskStep {
	return TaskStep{
		Description: "Create new file",
		Tool:        "write_file",
		Params:      map[string]interface{}{},
		Retryable:   true,
		MaxRetries:  3,
		Status:      StepStatusPending,
	}
}

func (a *TaskAnalyzer) createEditStep(description string) TaskStep {
	return TaskStep{
		Description: "Edit file",
		Tool:        "edit_file",
		Params:      map[string]interface{}{},
		Retryable:   true,
		MaxRetries:  3,
		Status:      StepStatusPending,
	}
}

func (a *TaskAnalyzer) createSearchStep(description string) TaskStep {
	return TaskStep{
		Description: "Search for content",
		Tool:        "search",
		Params:      map[string]interface{}{},
		Retryable:   true,
		MaxRetries:  3,
		Status:      StepStatusPending,
	}
}

func (a *TaskAnalyzer) createTestStep(description string) TaskStep {
	return TaskStep{
		Description: "Run tests",
		Tool:        "bash",
		Params: map[string]interface{}{
			"command": "${test_command}",
		},
		Retryable:   true,
		MaxRetries:  3,
		Status:      StepStatusPending,
	}
}

func (a *TaskAnalyzer) createGitSteps(description string) []TaskStep {
	steps := make([]TaskStep, 0)

	// Always start with status
	steps = append(steps, TaskStep{
		Description: "Check git status",
		Tool:        "git_status",
		Params:      map[string]interface{}{},
		Retryable:   false,
		MaxRetries:  1,
		Status:      StepStatusPending,
	})

	if strings.Contains(description, "diff") {
		steps = append(steps, TaskStep{
			Description: "Show git diff",
			Tool:        "git_diff",
			Params:      map[string]interface{}{},
			Retryable:   false,
			MaxRetries:  1,
			Status:      StepStatusPending,
		})
	}

	return steps
}

// generateStepDescription generates a description for a step
func (a *TaskAnalyzer) generateStepDescription(tool, taskDescription string) string {
	switch tool {
	case "read_file":
		return "Read file contents"
	case "write_file":
		return "Create or write file"
	case "edit_file":
		return "Edit file contents"
	case "search":
		return "Search for content"
	case "list_dir":
		return "List directory contents"
	case "tree":
		return "Show directory structure"
	case "bash":
		return "Execute command"
	case "git_status":
		return "Check git status"
	case "git_diff":
		return "Show git changes"
	default:
		return fmt.Sprintf("Execute %s", tool)
	}
}

// inferParams tries to infer parameters from the description
func (a *TaskAnalyzer) inferParams(tool, description string) map[string]interface{} {
	params := make(map[string]interface{})

	// Extract quoted strings as potential parameter values
	quotes := extractQuotedStrings(description)

	switch tool {
	case "read_file", "write_file", "edit_file":
		if len(quotes) > 0 {
			// First quoted string might be file path
			params["path"] = quotes[0]
		}
	case "search":
		if len(quotes) > 0 {
			// First quoted string might be search pattern
			params["pattern"] = quotes[0]
		}
		params["path"] = "."
	case "bash":
		if len(quotes) > 0 {
			params["command"] = quotes[0]
		}
	}

	return params
}

// requiresDependency checks if a tool typically requires a previous step
func (a *TaskAnalyzer) requiresDependency(tool string) bool {
	// Tools that typically depend on previous steps
	dependentTools := map[string]bool{
		"edit_file":   true,
		"git_diff":    true,
		"git_commit":  true,
	}
	return dependentTools[tool]
}

// extractQuotedStrings extracts quoted strings from text
func extractQuotedStrings(text string) []string {
	var result []string
	inQuote := false
	quoteChar := rune(0)
	current := strings.Builder{}

	for _, ch := range text {
		if !inQuote {
			if ch == '"' || ch == '\'' || ch == '`' {
				inQuote = true
				quoteChar = ch
			}
		} else {
			if ch == quoteChar {
				inQuote = false
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(ch)
			}
		}
	}

	return result
}

// initializePatterns sets up common task patterns
func initializePatterns() []TaskPattern {
	return []TaskPattern{
		{
			Keywords:    []string{"refactor", "clean", "improve"},
			ToolChain:   []string{"search", "read_file", "edit_file"},
			Description: "Code refactoring",
		},
		{
			Keywords:    []string{"add", "feature", "implement"},
			ToolChain:   []string{"tree", "write_file", "edit_file"},
			Description: "Feature implementation",
		},
		{
			Keywords:    []string{"fix", "bug", "error"},
			ToolChain:   []string{"search", "read_file", "edit_file", "bash"},
			Description: "Bug fixing",
		},
		{
			Keywords:    []string{"test", "unit test", "integration"},
			ToolChain:   []string{"write_file", "bash"},
			Description: "Test creation",
		},
		{
			Keywords:    []string{"analyze", "review", "understand"},
			ToolChain:   []string{"tree", "search", "read_file"},
			Description: "Code analysis",
		},
		{
			Keywords:    []string{"document", "readme", "docs"},
			ToolChain:   []string{"read_file", "write_file"},
			Description: "Documentation",
		},
	}
}

// BreakdownTask provides a more detailed task breakdown
func (a *TaskAnalyzer) BreakdownTask(description string) (*TaskBreakdown, error) {
	breakdown := &TaskBreakdown{
		OriginalTask: description,
		Subtasks:     make([]Subtask, 0),
		Dependencies: make(map[string][]string),
	}

	// Analyze task complexity
	complexity := a.assessComplexity(description)
	breakdown.Complexity = complexity

	// Extract entities (files, functions, etc.)
	entities := a.extractEntities(description)
	breakdown.Entities = entities

	// Generate subtasks based on complexity and entities
	if complexity == "simple" {
		// Single subtask
		subtask := Subtask{
			ID:          "subtask_1",
			Description: description,
			Priority:    1,
		}
		breakdown.Subtasks = append(breakdown.Subtasks, subtask)
	} else {
		// Break down into multiple subtasks
		subtasks := a.generateSubtasks(description, entities)
		breakdown.Subtasks = subtasks
		
		// Establish dependencies
		for i := 1; i < len(subtasks); i++ {
			breakdown.Dependencies[subtasks[i].ID] = []string{subtasks[i-1].ID}
		}
	}

	return breakdown, nil
}

// assessComplexity assesses the complexity of a task
func (a *TaskAnalyzer) assessComplexity(description string) string {
	wordCount := len(strings.Fields(description))
	
	// Count complexity indicators
	complexIndicators := []string{"multiple", "several", "various", "refactor", "migrate", "integrate"}
	indicatorCount := 0
	
	descLower := strings.ToLower(description)
	for _, indicator := range complexIndicators {
		if strings.Contains(descLower, indicator) {
			indicatorCount++
		}
	}

	if wordCount < 10 && indicatorCount == 0 {
		return "simple"
	} else if wordCount < 25 && indicatorCount <= 1 {
		return "moderate"
	} else {
		return "complex"
	}
}

// extractEntities extracts relevant entities from the description
func (a *TaskAnalyzer) extractEntities(description string) []Entity {
	entities := make([]Entity, 0)

	// Extract file paths
	words := strings.Fields(description)
	for _, word := range words {
		if strings.Contains(word, "/") || strings.Contains(word, ".") {
			// Potential file path
			entities = append(entities, Entity{
				Type:  "file",
				Name:  strings.Trim(word, ".,!?"),
				Value: strings.Trim(word, ".,!?"),
			})
		}
	}

	// Extract function/class names (simple heuristic)
	// Look for CamelCase or snake_case words
	for _, word := range words {
		cleaned := strings.Trim(word, ".,!?()")
		if isCamelCase(cleaned) || isSnakeCase(cleaned) {
			entities = append(entities, Entity{
				Type:  "identifier",
				Name:  cleaned,
				Value: cleaned,
			})
		}
	}

	return entities
}

// generateSubtasks generates subtasks based on the description and entities
func (a *TaskAnalyzer) generateSubtasks(description string, entities []Entity) []Subtask {
	subtasks := make([]Subtask, 0)
	priority := 1

	// Always start with understanding the current state
	subtasks = append(subtasks, Subtask{
		ID:          fmt.Sprintf("subtask_%d", priority),
		Description: "Analyze current implementation",
		Priority:    priority,
	})
	priority++

	// Add entity-specific subtasks
	for _, entity := range entities {
		if entity.Type == "file" {
			subtasks = append(subtasks, Subtask{
				ID:          fmt.Sprintf("subtask_%d", priority),
				Description: fmt.Sprintf("Process file: %s", entity.Name),
				Priority:    priority,
			})
			priority++
		}
	}

	// Add final verification step
	subtasks = append(subtasks, Subtask{
		ID:          fmt.Sprintf("subtask_%d", priority),
		Description: "Verify changes and test",
		Priority:    priority,
	})

	return subtasks
}

// Helper functions

func isCamelCase(word string) bool {
	if len(word) < 2 {
		return false
	}
	
	hasUpper := false
	hasLower := false
	
	for _, ch := range word {
		if ch >= 'A' && ch <= 'Z' {
			hasUpper = true
		} else if ch >= 'a' && ch <= 'z' {
			hasLower = true
		}
	}
	
	return hasUpper && hasLower
}

func isSnakeCase(word string) bool {
	return strings.Contains(word, "_") && !strings.HasPrefix(word, "_") && !strings.HasSuffix(word, "_")
}

// Types for task breakdown

type TaskBreakdown struct {
	OriginalTask string              `json:"original_task"`
	Complexity   string              `json:"complexity"`
	Entities     []Entity            `json:"entities"`
	Subtasks     []Subtask           `json:"subtasks"`
	Dependencies map[string][]string `json:"dependencies"`
}

type Entity struct {
	Type  string `json:"type"`  // "file", "function", "class", "identifier"
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Subtask struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
}