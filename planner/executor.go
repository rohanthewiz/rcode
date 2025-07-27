package planner

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rohanthewiz/serr"
	"rcode/tools"
)

// StepExecutor executes individual task steps
type StepExecutor struct {
	toolRegistry *tools.Registry
}

// NewStepExecutor creates a new step executor
func NewStepExecutor() *StepExecutor {
	return &StepExecutor{
		toolRegistry: tools.DefaultRegistry(),
	}
}

// NewStepExecutorWithRegistry creates a new step executor with a custom registry
func NewStepExecutorWithRegistry(registry *tools.Registry) *StepExecutor {
	return &StepExecutor{
		toolRegistry: registry,
	}
}

// Execute executes a single step
func (e *StepExecutor) Execute(step *TaskStep, context *TaskContext) (*StepResult, error) {
	startTime := time.Now()
	
	result := &StepResult{
		Success: false,
		Retries: 0,
	}

	// Validate tool exists
	toolFound := false
	for _, tool := range e.toolRegistry.GetTools() {
		if tool.Name == step.Tool {
			toolFound = true
			break
		}
	}

	if !toolFound {
		result.Error = fmt.Sprintf("unknown tool: %s", step.Tool)
		return result, serr.New(result.Error)
	}

	// Prepare parameters with variable substitution
	params := e.prepareParams(step.Params, context)

	// Create tool use request
	toolUse := tools.ToolUse{
		Type:  "tool_use",
		ID:    step.ID,
		Name:  step.Tool,
		Input: params,
	}

	// Execute the tool
	toolResult, err := e.toolRegistry.Execute(toolUse)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	// Parse result
	result.Success = !strings.Contains(toolResult.Content, "Error:")
	result.Output = toolResult.Content
	result.Duration = time.Since(startTime)

	if !result.Success {
		result.Error = toolResult.Content
		return result, serr.New(result.Error)
	}

	return result, nil
}

// prepareParams prepares parameters with variable substitution
func (e *StepExecutor) prepareParams(params map[string]interface{}, context *TaskContext) map[string]interface{} {
	prepared := make(map[string]interface{})
	
	for key, value := range params {
		// Check if value is a variable reference
		if strVal, ok := value.(string); ok && strings.HasPrefix(strVal, "${") && strings.HasSuffix(strVal, "}") {
			varName := strVal[2 : len(strVal)-1]
			if varValue, exists := context.Variables[varName]; exists {
				prepared[key] = varValue
			} else {
				prepared[key] = value // Keep original if variable not found
			}
		} else {
			prepared[key] = value
		}
	}

	return prepared
}

// SetToolRegistry allows setting a custom tool registry
func (e *StepExecutor) SetToolRegistry(registry *tools.Registry) {
	e.toolRegistry = registry
}

// ValidateStep validates that a step can be executed
func (e *StepExecutor) ValidateStep(step *TaskStep) error {
	if step.Tool == "" {
		return serr.New("tool name is required")
	}

	if step.ID == "" {
		return serr.New("step ID is required")
	}

	// Check if tool exists
	toolFound := false
	var toolDef tools.Tool
	
	for _, tool := range e.toolRegistry.GetTools() {
		if tool.Name == step.Tool {
			toolFound = true
			toolDef = tool
			break
		}
	}

	if !toolFound {
		return serr.New(fmt.Sprintf("unknown tool: %s", step.Tool))
	}

	// Validate parameters against tool schema
	if toolDef.InputSchema != nil {
		if err := e.validateParams(step.Params, toolDef.InputSchema); err != nil {
			return serr.Wrap(err, "invalid parameters")
		}
	}

	return nil
}

// validateParams validates parameters against a schema
func (e *StepExecutor) validateParams(params map[string]interface{}, schema map[string]interface{}) error {
	// Get required fields
	required := make(map[string]bool)
	if reqArray, ok := schema["required"].([]string); ok {
		for _, field := range reqArray {
			required[field] = true
		}
	}

	// Get properties schema
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return nil // No properties to validate
	}

	// Check required fields
	for field := range required {
		if _, exists := params[field]; !exists {
			return serr.New(fmt.Sprintf("required field '%s' is missing", field))
		}
	}

	// Validate field types
	for field, value := range params {
		if propSchema, ok := properties[field].(map[string]interface{}); ok {
			if err := e.validateFieldType(field, value, propSchema); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateFieldType validates a single field against its schema
func (e *StepExecutor) validateFieldType(field string, value interface{}, schema map[string]interface{}) error {
	expectedType, ok := schema["type"].(string)
	if !ok {
		return nil // No type constraint
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return serr.New(fmt.Sprintf("field '%s' must be a string", field))
		}
	case "integer":
		switch value.(type) {
		case int, int32, int64, float64:
			// Accept numeric types
		default:
			return serr.New(fmt.Sprintf("field '%s' must be an integer", field))
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return serr.New(fmt.Sprintf("field '%s' must be a boolean", field))
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return serr.New(fmt.Sprintf("field '%s' must be an object", field))
		}
	case "array":
		switch value.(type) {
		case []interface{}, []string, []int:
			// Accept array types
		default:
			return serr.New(fmt.Sprintf("field '%s' must be an array", field))
		}
	}

	return nil
}

// DryRun performs a dry run of a step without executing it
func (e *StepExecutor) DryRun(step *TaskStep, context *TaskContext) (*StepResult, error) {
	// Validate the step
	if err := e.ValidateStep(step); err != nil {
		return nil, err
	}

	// Prepare parameters
	params := e.prepareParams(step.Params, context)

	// Create a dry run result
	result := &StepResult{
		Success: true,
		Output: fmt.Sprintf("DRY RUN: Would execute tool '%s' with parameters: %s",
			step.Tool, e.formatParams(params)),
		Retries:  0,
		Duration: 0,
	}

	return result, nil
}

// formatParams formats parameters for display
func (e *StepExecutor) formatParams(params map[string]interface{}) string {
	// Convert to JSON for readable display
	data, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", params)
	}
	return string(data)
}

// GetAvailableTools returns a list of available tools
func (e *StepExecutor) GetAvailableTools() []ToolInfo {
	tools := e.toolRegistry.GetTools()
	result := make([]ToolInfo, 0, len(tools))

	for _, tool := range tools {
		info := ToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  make([]ParameterInfo, 0),
		}

		// Extract parameter information
		if props, ok := tool.InputSchema["properties"].(map[string]interface{}); ok {
			required := make(map[string]bool)
			if reqArray, ok := tool.InputSchema["required"].([]string); ok {
				for _, field := range reqArray {
					required[field] = true
				}
			}

			for name, schema := range props {
				if paramSchema, ok := schema.(map[string]interface{}); ok {
					param := ParameterInfo{
						Name:        name,
						Required:    required[name],
					}

					if typeStr, ok := paramSchema["type"].(string); ok {
						param.Type = typeStr
					}
					if desc, ok := paramSchema["description"].(string); ok {
						param.Description = desc
					}

					info.Parameters = append(info.Parameters, param)
				}
			}
		}

		result = append(result, info)
	}

	return result
}

// ToolInfo contains information about a tool
type ToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []ParameterInfo `json:"parameters"`
}

// ParameterInfo contains information about a tool parameter
type ParameterInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}