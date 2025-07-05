package tool

import (
	"context"
	"fmt"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// Executor handles the execution of tools within the context of a message.
// It manages parameter validation, timeout handling, and error recovery.
type Executor struct {
	registry *Registry
}

// NewExecutor creates a new tool executor
func NewExecutor(registry *Registry) *Executor {
	return &Executor{
		registry: registry,
	}
}

// Execute runs a tool with the given parameters.
// It handles validation, timeout, and error handling.
func (e *Executor) Execute(toolID string, params map[string]any, ctx Context) (Result, error) {
	// Retrieve the tool from registry
	tool, exists := e.registry.Get(toolID)
	if !exists {
		return Result{}, serr.New("tool not found: %s", toolID)
	}
	
	// Validate parameters against schema
	if err := tool.Parameters().Validate(params); err != nil {
		return Result{}, serr.Wrap(err, "parameter validation failed")
	}
	
	// Create execution context with timeout
	execCtx, cancel := context.WithCancel(ctx.Abort)
	defer cancel()
	
	// Track execution time
	start := time.Now()
	
	// Channel for collecting results
	resultChan := make(chan Result, 1)
	errChan := make(chan error, 1)
	
	// Execute tool in goroutine to handle cancellation
	go func() {
		defer func() {
			// Recover from panics in tool execution
			if r := recover(); r != nil {
				errChan <- serr.New("tool panicked: %v", r)
			}
		}()
		
		// Execute the tool
		result, err := tool.Execute(Context{
			SessionID: ctx.SessionID,
			MessageID: ctx.MessageID,
			Abort:     execCtx,
			Metadata:  ctx.Metadata,
		}, params)
		
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()
	
	// Wait for result or cancellation
	select {
	case result := <-resultChan:
		duration := time.Since(start)
		logger.Info("tool executed successfully",
			"tool", toolID,
			"duration", duration,
			"session", ctx.SessionID,
		)
		
		// Add execution metadata
		if result.Metadata == nil {
			result.Metadata = make(map[string]any)
		}
		result.Metadata["duration_ms"] = duration.Milliseconds()
		
		return result, nil
		
	case err := <-errChan:
		duration := time.Since(start)
		logger.LogErr(err, "tool execution failed",
			"tool", toolID,
			"duration", duration,
			"session", ctx.SessionID,
		)
		
		// Return error as output for AI to see
		return Result{
			Output: fmt.Sprintf("Error: %v", err),
			Metadata: map[string]any{
				"error":       err.Error(),
				"duration_ms": duration.Milliseconds(),
			},
		}, nil
		
	case <-execCtx.Done():
		// Execution was cancelled
		logger.Info("tool execution cancelled",
			"tool", toolID,
			"session", ctx.SessionID,
		)
		
		return Result{
			Output: "Execution cancelled",
			Metadata: map[string]any{
				"cancelled": true,
			},
		}, nil
	}
}

// ExecuteMultiple runs multiple tools concurrently.
// This is useful when the AI requests multiple tool executions at once.
func (e *Executor) ExecuteMultiple(executions []ToolExecution, ctx Context) []ExecutionResult {
	results := make([]ExecutionResult, len(executions))
	var wg sync.WaitGroup
	
	for i, exec := range executions {
		wg.Add(1)
		go func(idx int, execution ToolExecution) {
			defer wg.Done()
			
			// Create a unique metadata callback for this execution
			metadataCallback := func(meta map[string]any) {
				// Include tool call ID in metadata updates
				meta["tool_call_id"] = execution.CallID
				ctx.Metadata(meta)
			}
			
			// Execute the tool
			result, err := e.Execute(
				execution.ToolID,
				execution.Parameters,
				Context{
					SessionID: ctx.SessionID,
					MessageID: ctx.MessageID,
					Abort:     ctx.Abort,
					Metadata:  metadataCallback,
				},
			)
			
			results[idx] = ExecutionResult{
				CallID: execution.CallID,
				Result: result,
				Error:  err,
			}
		}(i, exec)
	}
	
	wg.Wait()
	return results
}

// ToolExecution represents a single tool execution request
type ToolExecution struct {
	CallID     string
	ToolID     string
	Parameters map[string]any
}

// ExecutionResult contains the result of a tool execution
type ExecutionResult struct {
	CallID string
	Result Result
	Error  error
}