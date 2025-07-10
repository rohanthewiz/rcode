package tools

import (
	"fmt"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// EnhancedRegistry wraps the standard registry with validation and enhanced features
type EnhancedRegistry struct {
	*Registry
	validator      *ToolValidator
	metrics        *ToolMetrics
	beforeExecute  []BeforeExecuteHook
	afterExecute   []AfterExecuteHook
}

// BeforeExecuteHook is called before tool execution
type BeforeExecuteHook func(toolName string, params map[string]interface{}) error

// AfterExecuteHook is called after tool execution
type AfterExecuteHook func(toolName string, params map[string]interface{}, result *ToolResult, err error)

// ToolMetrics tracks tool usage metrics
type ToolMetrics struct {
	executions    map[string]int
	totalDuration map[string]int64 // milliseconds
	failures      map[string]int
}

// NewEnhancedRegistry creates a new enhanced registry
func NewEnhancedRegistry() *EnhancedRegistry {
	return &EnhancedRegistry{
		Registry:      NewRegistry(),
		validator:     NewToolValidator(),
		metrics:       NewToolMetrics(),
		beforeExecute: make([]BeforeExecuteHook, 0),
		afterExecute:  make([]AfterExecuteHook, 0),
	}
}

// NewToolMetrics creates a new metrics tracker
func NewToolMetrics() *ToolMetrics {
	return &ToolMetrics{
		executions:    make(map[string]int),
		totalDuration: make(map[string]int64),
		failures:      make(map[string]int),
	}
}

// Execute runs a tool with validation and hooks
func (r *EnhancedRegistry) Execute(toolUse ToolUse) (*ToolResult, error) {
	// Validate parameters
	if err := r.validator.Validate(toolUse.Name, toolUse.Input); err != nil {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Validation error: %v", err),
		}, serr.Wrap(err, "parameter validation failed")
	}

	// Run before-execute hooks
	for _, hook := range r.beforeExecute {
		if err := hook(toolUse.Name, toolUse.Input); err != nil {
			return &ToolResult{
				Type:      "tool_result",
				ToolUseID: toolUse.ID,
				Content:   fmt.Sprintf("Pre-execution hook error: %v", err),
			}, serr.Wrap(err, "before-execute hook failed")
		}
	}

	// Track execution start time
	startTime := time.Now()

	// Execute the tool
	result, err := r.Registry.Execute(toolUse)

	// Calculate duration
	duration := time.Since(startTime).Milliseconds()

	// Update metrics
	r.metrics.RecordExecution(toolUse.Name, duration, err != nil)

	// Run after-execute hooks
	for _, hook := range r.afterExecute {
		hook(toolUse.Name, toolUse.Input, result, err)
	}

	// Log execution details
	if err != nil {
		logger.LogErr(err, fmt.Sprintf("Tool execution failed: %s (duration: %dms)", toolUse.Name, duration))
	} else {
		logger.Debug(fmt.Sprintf("Tool executed successfully: %s (duration: %dms)", toolUse.Name, duration))
	}

	return result, err
}

// RegisterWithValidation registers a tool with automatic validation setup
func (r *EnhancedRegistry) RegisterWithValidation(tool Tool, executor Executor) {
	// Register the tool
	r.Register(tool, executor)

	// If the tool has a schema, ensure validation rules are set up
	if tool.InputSchema != nil {
		// The validator already has default rules, but we can enhance them
		// based on the tool's schema if needed
		logger.Debug("Registered tool with validation: " + tool.Name)
	}
}

// AddBeforeExecuteHook adds a hook to run before tool execution
func (r *EnhancedRegistry) AddBeforeExecuteHook(hook BeforeExecuteHook) {
	r.beforeExecute = append(r.beforeExecute, hook)
}

// AddAfterExecuteHook adds a hook to run after tool execution
func (r *EnhancedRegistry) AddAfterExecuteHook(hook AfterExecuteHook) {
	r.afterExecute = append(r.afterExecute, hook)
}

// GetMetrics returns tool usage metrics
func (r *EnhancedRegistry) GetMetrics() map[string]interface{} {
	return r.metrics.GetSummary()
}

// GetToolSchema returns the enhanced schema for a tool
func (r *EnhancedRegistry) GetToolSchema(toolName string) map[string]interface{} {
	// First check if validator has a schema
	if schema := r.validator.GetSchema(toolName); schema != nil {
		return schema
	}

	// Fall back to tool's own schema
	for _, tool := range r.GetTools() {
		if tool.Name == toolName {
			return tool.InputSchema
		}
	}

	return nil
}

// ValidateParams validates parameters for a tool without executing it
func (r *EnhancedRegistry) ValidateParams(toolName string, params map[string]interface{}) error {
	return r.validator.Validate(toolName, params)
}

// ToolMetrics methods

// RecordExecution records a tool execution
func (m *ToolMetrics) RecordExecution(toolName string, durationMs int64, failed bool) {
	m.executions[toolName]++
	m.totalDuration[toolName] += durationMs
	
	if failed {
		m.failures[toolName]++
	}
}

// GetSummary returns a summary of metrics
func (m *ToolMetrics) GetSummary() map[string]interface{} {
	summary := make(map[string]interface{})
	
	for tool, count := range m.executions {
		avgDuration := int64(0)
		if count > 0 {
			avgDuration = m.totalDuration[tool] / int64(count)
		}
		
		successRate := 100.0
		if count > 0 {
			successRate = float64(count-m.failures[tool]) / float64(count) * 100
		}
		
		summary[tool] = map[string]interface{}{
			"executions":      count,
			"failures":        m.failures[tool],
			"success_rate":    fmt.Sprintf("%.1f%%", successRate),
			"avg_duration_ms": avgDuration,
			"total_time_ms":   m.totalDuration[tool],
		}
	}
	
	return summary
}

// DefaultEnhancedRegistry creates an enhanced registry with all default tools
func DefaultEnhancedRegistry() *EnhancedRegistry {
	registry := NewEnhancedRegistry()

	// Register all default tools
	readTool := &ReadFileTool{}
	registry.RegisterWithValidation(readTool.GetDefinition(), readTool)

	writeTool := &WriteFileTool{}
	registry.RegisterWithValidation(writeTool.GetDefinition(), writeTool)

	bashTool := &BashTool{}
	registry.RegisterWithValidation(bashTool.GetDefinition(), bashTool)

	editTool := &EditFileTool{}
	registry.RegisterWithValidation(editTool.GetDefinition(), editTool)

	searchTool := &SearchTool{}
	registry.RegisterWithValidation(searchTool.GetDefinition(), searchTool)

	// Directory operations
	listDirTool := &ListDirTool{}
	registry.RegisterWithValidation(listDirTool.GetDefinition(), listDirTool)

	makeDirTool := &MakeDirTool{}
	registry.RegisterWithValidation(makeDirTool.GetDefinition(), makeDirTool)

	removeTool := &RemoveTool{}
	registry.RegisterWithValidation(removeTool.GetDefinition(), removeTool)

	treeTool := &TreeTool{}
	registry.RegisterWithValidation(treeTool.GetDefinition(), treeTool)

	moveTool := &MoveTool{}
	registry.RegisterWithValidation(moveTool.GetDefinition(), moveTool)

	// Git tools
	gitStatusTool := &GitStatusTool{}
	registry.RegisterWithValidation(gitStatusTool.GetDefinition(), gitStatusTool)

	gitDiffTool := &GitDiffTool{}
	registry.RegisterWithValidation(gitDiffTool.GetDefinition(), gitDiffTool)

	gitLogTool := &GitLogTool{}
	registry.RegisterWithValidation(gitLogTool.GetDefinition(), gitLogTool)

	gitBranchTool := &GitBranchTool{}
	registry.RegisterWithValidation(gitBranchTool.GetDefinition(), gitBranchTool)

	// Add default hooks
	registry.AddBeforeExecuteHook(func(toolName string, params map[string]interface{}) error {
		// Log tool execution
		logger.Debug("Executing tool: " + toolName)
		return nil
	})

	registry.AddAfterExecuteHook(func(toolName string, params map[string]interface{}, result *ToolResult, err error) {
		// Log result
		if err != nil {
			logger.Debug(fmt.Sprintf("Tool execution failed: %s - %v", toolName, err))
		}
	})

	return registry
}