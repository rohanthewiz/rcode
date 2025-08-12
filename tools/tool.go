package tools

import (
	"encoding/json"
)

// Tool represents a tool that can be used by the AI
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ToolUse represents a tool use request from the AI
type ToolUse struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResult represents the result of executing a tool
type ToolResult struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

// Executor is the interface for tool execution
type Executor interface {
	Execute(input map[string]interface{}) (string, error)
}

// Registry holds all available tools
type Registry struct {
	tools     map[string]Tool
	executors map[string]Executor
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools:     make(map[string]Tool),
		executors: make(map[string]Executor),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool, executor Executor) {
	r.tools[tool.Name] = tool
	r.executors[tool.Name] = executor
}

// GetTools returns all registered tools
func (r *Registry) GetTools() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Execute runs a tool and returns the result
func (r *Registry) Execute(toolUse ToolUse) (*ToolResult, error) {
	executor, exists := r.executors[toolUse.Name]
	if !exists {
		return nil, &ToolError{Message: "Unknown tool: " + toolUse.Name}
	}

	result, err := executor.Execute(toolUse.Input)
	if err != nil {
		// Return both the error result and the error itself
		// This allows the enhanced registry to handle retries
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   "Error: " + err.Error(),
		}, err
	}

	return &ToolResult{
		Type:      "tool_result",
		ToolUseID: toolUse.ID,
		Content:   result,
	}, nil
}

// ToolError represents a tool execution error
type ToolError struct {
	Message string
}

func (e *ToolError) Error() string {
	return e.Message
}

// Helper function to get string from interface{}
func GetString(input map[string]interface{}, key string) (string, bool) {
	val, exists := input[key]
	if !exists {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// Helper function to get int from interface{}
func GetInt(input map[string]interface{}, key string) (int, bool) {
	val, exists := input[key]
	if !exists {
		return 0, false
	}

	// Handle both int and float64 (JSON numbers are float64)
	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// Helper function to get bool from interface{}
func GetBool(input map[string]interface{}, key string) (bool, bool) {
	val, exists := input[key]
	if !exists {
		return false, false
	}
	boolVal, ok := val.(bool)
	return boolVal, ok
}

// MarshalJSON for proper JSON encoding
func (t Tool) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		InputSchema map[string]interface{} `json:"input_schema"`
	}{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: t.InputSchema,
	})
}
