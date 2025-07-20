package tools

import (
	"context"
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
	retryPolicies  map[string]RetryPolicy // Tool-specific retry policies
	defaultRetry   RetryPolicy            // Default retry policy for all tools
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
	retries       map[string]int   // Number of retry attempts
	retrySuccess  map[string]int   // Successful retries
}

// NewEnhancedRegistry creates a new enhanced registry
func NewEnhancedRegistry() *EnhancedRegistry {
	return &EnhancedRegistry{
		Registry:      NewRegistry(),
		validator:     NewToolValidator(),
		metrics:       NewToolMetrics(),
		beforeExecute: make([]BeforeExecuteHook, 0),
		afterExecute:  make([]AfterExecuteHook, 0),
		retryPolicies: make(map[string]RetryPolicy),
		defaultRetry:  RetryPolicy{}, // No retry by default
	}
}

// NewToolMetrics creates a new metrics tracker
func NewToolMetrics() *ToolMetrics {
	return &ToolMetrics{
		executions:    make(map[string]int),
		totalDuration: make(map[string]int64),
		failures:      make(map[string]int),
		retries:       make(map[string]int),
		retrySuccess:  make(map[string]int),
	}
}

// Execute runs a tool with validation, retries, and hooks
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

	// Get retry policy for this tool
	retryPolicy := r.getRetryPolicy(toolUse.Name)

	// Track overall execution start time
	overallStartTime := time.Now()

	// Create the operation to retry
	var result *ToolResult
	var lastErr error
	
	operation := func(ctx context.Context) error {
		// Execute the tool
		res, err := r.Registry.Execute(toolUse)
		result = res
		lastErr = err
		
		return err
	}

	// Execute with retry if policy is configured
	if retryPolicy.MaxAttempts > 0 {
		ctx := context.Background()
		retryResult := Retry(ctx, retryPolicy, operation)
		
		// Update retry metrics
		if retryResult.Attempts > 1 {
			r.metrics.retries[toolUse.Name] += retryResult.Attempts - 1
			if retryResult.Success {
				r.metrics.retrySuccess[toolUse.Name]++
			}
		}
		
		// Log retry details if there were retries
		if retryResult.Attempts > 1 {
			if retryResult.Success {
				logger.Info(fmt.Sprintf("Tool %s succeeded after %d attempts (total duration: %v)",
					toolUse.Name, retryResult.Attempts, retryResult.TotalDuration),
					"tool", toolUse.Name,
					"attempts", retryResult.Attempts)
			} else {
				logger.LogErr(retryResult.LastError, fmt.Sprintf("Tool %s failed after %d attempts (total duration: %v)",
					toolUse.Name, retryResult.Attempts, retryResult.TotalDuration))
			}
		}
		
		lastErr = retryResult.LastError
	} else {
		// No retry policy, execute once
		err := operation(context.Background())
		lastErr = err
	}

	// Calculate overall duration
	overallDuration := time.Since(overallStartTime).Milliseconds()

	// Update overall metrics
	r.metrics.RecordExecution(toolUse.Name, overallDuration, lastErr != nil)

	// Run after-execute hooks
	for _, hook := range r.afterExecute {
		hook(toolUse.Name, toolUse.Input, result, lastErr)
	}

	// Log execution details
	if lastErr != nil {
		logger.LogErr(lastErr, fmt.Sprintf("Tool execution failed: %s (duration: %dms)", toolUse.Name, overallDuration))
	} else {
		logger.Debug(fmt.Sprintf("Tool executed successfully: %s (duration: %dms)", toolUse.Name, overallDuration),
			"tool", toolUse.Name,
			"duration_ms", overallDuration)
	}

	return result, lastErr
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

// SetDefaultRetryPolicy sets the default retry policy for all tools
func (r *EnhancedRegistry) SetDefaultRetryPolicy(policy RetryPolicy) {
	r.defaultRetry = policy
}

// SetToolRetryPolicy sets a specific retry policy for a tool
func (r *EnhancedRegistry) SetToolRetryPolicy(toolName string, policy RetryPolicy) {
	r.retryPolicies[toolName] = policy
}

// getRetryPolicy returns the retry policy for a specific tool
func (r *EnhancedRegistry) getRetryPolicy(toolName string) RetryPolicy {
	// Check for tool-specific policy
	if policy, exists := r.retryPolicies[toolName]; exists {
		return policy
	}
	// Return default policy
	return r.defaultRetry
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
		
		retryRate := 0.0
		retrySuccessRate := 0.0
		if retries := m.retries[tool]; retries > 0 {
			retryRate = float64(retries) / float64(count) * 100
			if retrySuccess := m.retrySuccess[tool]; retrySuccess > 0 {
				retrySuccessRate = float64(retrySuccess) / float64(retries) * 100
			}
		}
		
		summary[tool] = map[string]interface{}{
			"executions":         count,
			"failures":           m.failures[tool],
			"success_rate":       fmt.Sprintf("%.1f%%", successRate),
			"avg_duration_ms":    avgDuration,
			"total_time_ms":      m.totalDuration[tool],
			"retries":            m.retries[tool],
			"retry_success":      m.retrySuccess[tool],
			"retry_rate":         fmt.Sprintf("%.1f%%", retryRate),
			"retry_success_rate": fmt.Sprintf("%.1f%%", retrySuccessRate),
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

	gitAddTool := &GitAddTool{}
	registry.RegisterWithValidation(gitAddTool.GetDefinition(), gitAddTool)

	gitCommitTool := &GitCommitTool{}
	registry.RegisterWithValidation(gitCommitTool.GetDefinition(), gitCommitTool)

	gitPushTool := &GitPushTool{}
	registry.RegisterWithValidation(gitPushTool.GetDefinition(), gitPushTool)

	gitPullTool := &GitPullTool{}
	registry.RegisterWithValidation(gitPullTool.GetDefinition(), gitPullTool)

	gitCheckoutTool := &GitCheckoutTool{}
	registry.RegisterWithValidation(gitCheckoutTool.GetDefinition(), gitCheckoutTool)

	gitMergeTool := &GitMergeTool{}
	registry.RegisterWithValidation(gitMergeTool.GetDefinition(), gitMergeTool)

	// Web tools
	webSearchTool := &WebSearchTool{}
	registry.RegisterWithValidation(webSearchTool.GetDefinition(), webSearchTool)

	webFetchTool := &WebFetchTool{}
	registry.RegisterWithValidation(webFetchTool.GetDefinition(), webFetchTool)

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

	// Setup diff integration for file modification tracking
	diffIntegration, err := NewDiffIntegration()
	if err != nil {
		logger.LogErr(err, "failed to initialize diff integration")
		// Continue without diff integration - it's not critical
	} else {
		diffIntegration.SetupDiffHooks(registry)
		logger.Debug("Diff integration hooks registered")
	}

	// Configure retry policies for tools that benefit from retries
	
	// Network-based tools get more aggressive retry
	registry.SetToolRetryPolicy("web_fetch", NetworkRetryPolicy)
	registry.SetToolRetryPolicy("web_search", NetworkRetryPolicy)
	registry.SetToolRetryPolicy("git_push", NetworkRetryPolicy)
	registry.SetToolRetryPolicy("git_pull", NetworkRetryPolicy)
	registry.SetToolRetryPolicy("git_fetch", NetworkRetryPolicy)
	registry.SetToolRetryPolicy("git_clone", NetworkRetryPolicy)
	
	// File system tools get lighter retry
	registry.SetToolRetryPolicy("read_file", FileSystemRetryPolicy)
	registry.SetToolRetryPolicy("write_file", FileSystemRetryPolicy)
	registry.SetToolRetryPolicy("edit_file", FileSystemRetryPolicy)
	registry.SetToolRetryPolicy("list_dir", FileSystemRetryPolicy)
	registry.SetToolRetryPolicy("make_dir", FileSystemRetryPolicy)
	registry.SetToolRetryPolicy("remove", FileSystemRetryPolicy)
	registry.SetToolRetryPolicy("move", FileSystemRetryPolicy)
	
	// Git local operations might need retry for lock issues
	registry.SetToolRetryPolicy("git_status", FileSystemRetryPolicy)
	registry.SetToolRetryPolicy("git_diff", FileSystemRetryPolicy)
	registry.SetToolRetryPolicy("git_add", FileSystemRetryPolicy)
	registry.SetToolRetryPolicy("git_commit", FileSystemRetryPolicy)
	
	// Bash commands don't retry by default (could be destructive)
	// But users can configure specific retry if needed

	return registry
}