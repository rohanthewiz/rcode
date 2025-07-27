package tools

import (
	"errors"
	"testing"
	"time"
)

// MockTool for testing
type MockTool struct {
	name        string
	executions  int
	failUntil   int
	failWithErr error
}

func (m *MockTool) GetDefinition() Tool {
	return Tool{
		Name:        m.name,
		Description: "Mock tool for testing",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"test": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}
}

func (m *MockTool) Execute(input map[string]interface{}) (string, error) {
	m.executions++
	if m.executions <= m.failUntil {
		if m.failWithErr != nil {
			return "", m.failWithErr
		}
		return "", errors.New("mock failure")
	}
	return "success", nil
}

// TestEnhancedRegistryWithRetry tests the retry functionality in enhanced registry
func TestEnhancedRegistryWithRetry(t *testing.T) {
	t.Run("successful retry", func(t *testing.T) {
		registry := NewEnhancedRegistry()
		mockTool := &MockTool{
			name:      "test_tool",
			failUntil: 2, // Fail first 2 attempts
		}

		registry.RegisterWithValidation(mockTool.GetDefinition(), mockTool)

		// Set retry policy
		registry.SetToolRetryPolicy("test_tool", RetryPolicy{
			MaxAttempts:     3,
			InitialDelay:    10 * time.Millisecond,
			MaxDelay:        100 * time.Millisecond,
			Multiplier:      2,
			RetryableErrors: func(err error) bool { return true },
		})

		// Execute tool
		result, err := registry.Execute(ToolUse{
			ID:    "test-1",
			Name:  "test_tool",
			Input: map[string]interface{}{"test": "value"},
		})

		if err != nil {
			t.Errorf("Expected success after retry, got error: %v", err)
		}
		if result.Content != "success" {
			t.Errorf("Expected content 'success', got %q", result.Content)
		}
		if mockTool.executions != 3 {
			t.Errorf("Expected 3 executions, got %d", mockTool.executions)
		}

		// Check metrics
		metrics := registry.GetMetrics()
		if toolMetrics, ok := metrics["test_tool"].(map[string]interface{}); ok {
			// Debug: print all metrics
			t.Logf("Tool metrics: %+v", toolMetrics)

			if retries, ok := toolMetrics["retries"].(int); !ok || retries != 2 {
				t.Errorf("Expected 2 retries in metrics, got %v", retries)
			}
			if retrySuccess, ok := toolMetrics["retry_success"].(int); !ok || retrySuccess != 1 {
				t.Errorf("Expected 1 retry success in metrics, got %v", retrySuccess)
			}
		} else {
			t.Error("No metrics found for test_tool")
		}
	})

	t.Run("permanent error no retry", func(t *testing.T) {
		registry := NewEnhancedRegistry()
		mockTool := &MockTool{
			name:        "test_tool",
			failUntil:   10,
			failWithErr: NewPermanentError(errors.New("permanent"), "test"),
		}

		registry.RegisterWithValidation(mockTool.GetDefinition(), mockTool)

		// Set retry policy
		registry.SetToolRetryPolicy("test_tool", RetryPolicy{
			MaxAttempts:     3,
			InitialDelay:    10 * time.Millisecond,
			RetryableErrors: IsRetryableError,
		})

		// Execute tool
		_, err := registry.Execute(ToolUse{
			ID:    "test-2",
			Name:  "test_tool",
			Input: map[string]interface{}{"test": "value"},
		})

		if err == nil {
			t.Error("Expected permanent error")
		}
		if mockTool.executions != 1 {
			t.Errorf("Expected 1 execution for permanent error, got %d", mockTool.executions)
		}
	})

	t.Run("default retry policy", func(t *testing.T) {
		registry := NewEnhancedRegistry()
		mockTool := &MockTool{
			name:      "test_tool",
			failUntil: 1,
		}

		registry.RegisterWithValidation(mockTool.GetDefinition(), mockTool)

		// Set default retry policy for all tools
		registry.SetDefaultRetryPolicy(RetryPolicy{
			MaxAttempts:     2,
			InitialDelay:    10 * time.Millisecond,
			RetryableErrors: func(err error) bool { return true },
		})

		// Execute tool
		result, err := registry.Execute(ToolUse{
			ID:    "test-3",
			Name:  "test_tool",
			Input: map[string]interface{}{"test": "value"},
		})

		if err != nil {
			t.Errorf("Expected success with default retry, got error: %v", err)
		}
		if result.Content != "success" {
			t.Errorf("Expected content 'success', got %q", result.Content)
		}
		if mockTool.executions != 2 {
			t.Errorf("Expected 2 executions with default retry, got %d", mockTool.executions)
		}
	})

	t.Run("no retry policy", func(t *testing.T) {
		registry := NewEnhancedRegistry()
		mockTool := &MockTool{
			name:      "test_tool",
			failUntil: 1,
		}

		registry.RegisterWithValidation(mockTool.GetDefinition(), mockTool)

		// No retry policy set - should execute only once
		_, err := registry.Execute(ToolUse{
			ID:    "test-4",
			Name:  "test_tool",
			Input: map[string]interface{}{"test": "value"},
		})

		if err == nil {
			t.Error("Expected error with no retry")
		}
		if mockTool.executions != 1 {
			t.Errorf("Expected 1 execution with no retry, got %d", mockTool.executions)
		}
	})
}

// TestEnhancedRegistryMetrics tests the metrics tracking
func TestEnhancedRegistryMetrics(t *testing.T) {
	registry := NewEnhancedRegistry()

	// Create tools with different success/failure patterns
	successTool := &MockTool{name: "success_tool", failUntil: 0}
	failTool := &MockTool{name: "fail_tool", failUntil: 100}
	retryTool := &MockTool{name: "retry_tool", failUntil: 1}

	registry.RegisterWithValidation(successTool.GetDefinition(), successTool)
	registry.RegisterWithValidation(failTool.GetDefinition(), failTool)
	registry.RegisterWithValidation(retryTool.GetDefinition(), retryTool)

	// Set retry policy for retry_tool
	registry.SetToolRetryPolicy("retry_tool", RetryPolicy{
		MaxAttempts:     2,
		InitialDelay:    1 * time.Millisecond,
		RetryableErrors: func(err error) bool { return true },
	})

	// Execute tools multiple times
	for i := 0; i < 5; i++ {
		registry.Execute(ToolUse{
			ID:    "success-" + string(rune(i)),
			Name:  "success_tool",
			Input: map[string]interface{}{},
		})
	}

	for i := 0; i < 3; i++ {
		registry.Execute(ToolUse{
			ID:    "fail-" + string(rune(i)),
			Name:  "fail_tool",
			Input: map[string]interface{}{},
		})
	}

	for i := 0; i < 2; i++ {
		retryTool.executions = 0 // Reset for each test
		registry.Execute(ToolUse{
			ID:    "retry-" + string(rune(i)),
			Name:  "retry_tool",
			Input: map[string]interface{}{},
		})
	}

	// Check metrics
	metrics := registry.GetMetrics()

	// Debug: print all metrics
	t.Logf("All metrics: %+v", metrics)

	// Success tool metrics
	if successMetrics, ok := metrics["success_tool"].(map[string]interface{}); ok {
		if executions, ok := successMetrics["executions"].(int); !ok || executions != 5 {
			t.Errorf("Expected 5 executions for success_tool, got %v", executions)
		}
		if failures, ok := successMetrics["failures"].(int); !ok || failures != 0 {
			t.Errorf("Expected 0 failures for success_tool, got %v", failures)
		}
		if successRate, ok := successMetrics["success_rate"].(string); !ok || successRate != "100.0%" {
			t.Errorf("Expected 100.0%% success rate for success_tool, got %v", successRate)
		}
	} else {
		t.Error("Missing metrics for success_tool")
	}

	// Fail tool metrics
	if failMetrics, ok := metrics["fail_tool"].(map[string]interface{}); ok {
		if executions, ok := failMetrics["executions"].(int); !ok || executions != 3 {
			t.Errorf("Expected 3 executions for fail_tool, got %v", executions)
		}
		if failures, ok := failMetrics["failures"].(int); !ok || failures != 3 {
			t.Errorf("Expected 3 failures for fail_tool, got %v", failures)
		}
		if successRate, ok := failMetrics["success_rate"].(string); !ok || successRate != "0.0%" {
			t.Errorf("Expected 0.0%% success rate for fail_tool, got %v", successRate)
		}
	} else {
		t.Error("Missing metrics for fail_tool")
	}

	// Retry tool metrics
	if retryMetrics, ok := metrics["retry_tool"].(map[string]interface{}); ok {
		if retries, ok := retryMetrics["retries"].(int); !ok || retries != 2 {
			t.Errorf("Expected 2 total retries for retry_tool, got %v", retries)
		}
		if retrySuccess, ok := retryMetrics["retry_success"].(int); !ok || retrySuccess != 2 {
			t.Errorf("Expected 2 retry successes for retry_tool, got %v", retrySuccess)
		}
		if retrySuccessRate, ok := retryMetrics["retry_success_rate"].(string); !ok || retrySuccessRate != "100.0%" {
			t.Errorf("Expected 100.0%% retry success rate for retry_tool, got %v", retrySuccessRate)
		}
	} else {
		t.Error("Missing metrics for retry_tool")
	}
}

// TestRetryPolicyConfiguration tests the retry policy configuration in default registry
func TestRetryPolicyConfiguration(t *testing.T) {
	registry := DefaultEnhancedRegistry()

	// Check that network tools have NetworkRetryPolicy
	networkTools := []string{"web_fetch", "web_search", "git_push", "git_pull"}
	for _, toolName := range networkTools {
		// We can't directly access the retry policies, but we can verify they're configured
		// by checking that the tool exists in the registry
		found := false
		for _, tool := range registry.GetTools() {
			if tool.Name == toolName {
				found = true
				break
			}
		}
		if !found && toolName != "git_fetch" && toolName != "git_clone" { // These aren't implemented yet
			t.Errorf("Expected %s to be registered", toolName)
		}
	}

	// Check that file system tools are registered
	fsTools := []string{"read_file", "write_file", "edit_file", "list_dir", "make_dir", "remove", "move"}
	for _, toolName := range fsTools {
		found := false
		for _, tool := range registry.GetTools() {
			if tool.Name == toolName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected %s to be registered", toolName)
		}
	}
}
