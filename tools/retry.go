package tools

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/rohanthewiz/logger"
)

// RetryPolicy defines the configuration for retry attempts
type RetryPolicy struct {
	// MaxAttempts is the maximum number of retry attempts (excluding the initial attempt)
	MaxAttempts int
	// InitialDelay is the initial delay between retries
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration
	// Multiplier is the factor by which the delay is multiplied after each retry
	Multiplier float64
	// Jitter adds randomness to the delay to avoid thundering herd
	Jitter bool
	// RetryableErrors defines which errors should trigger a retry
	RetryableErrors func(error) bool
}

// DefaultRetryPolicy provides sensible defaults for most operations
var DefaultRetryPolicy = RetryPolicy{
	MaxAttempts:     3,
	InitialDelay:    100 * time.Millisecond,
	MaxDelay:        5 * time.Second,
	Multiplier:      2.0,
	Jitter:          true,
	RetryableErrors: IsRetryableError,
}

// NetworkRetryPolicy is optimized for network operations
var NetworkRetryPolicy = RetryPolicy{
	MaxAttempts:     5,
	InitialDelay:    500 * time.Millisecond,
	MaxDelay:        30 * time.Second,
	Multiplier:      2.0,
	Jitter:          true,
	RetryableErrors: IsRetryableError,
}

// FileSystemRetryPolicy is optimized for file system operations
var FileSystemRetryPolicy = RetryPolicy{
	MaxAttempts:     2,
	InitialDelay:    50 * time.Millisecond,
	MaxDelay:        500 * time.Millisecond,
	Multiplier:      2.0,
	Jitter:          false,
	RetryableErrors: IsRetryableError,
}

// RetryResult contains information about the retry operation
type RetryResult struct {
	// Attempts is the total number of attempts made
	Attempts int
	// Success indicates whether the operation eventually succeeded
	Success bool
	// LastError is the error from the final attempt
	LastError error
	// TotalDuration is the total time spent including all retries
	TotalDuration time.Duration
}

// RetryOperation represents a function that can be retried
type RetryOperation func(ctx context.Context) error

// Retry executes an operation with the specified retry policy
func Retry(ctx context.Context, policy RetryPolicy, operation RetryOperation) RetryResult {
	start := time.Now()
	result := RetryResult{Attempts: 1}

	// Execute the operation for the first time
	err := operation(ctx)
	if err == nil {
		result.Success = true
		result.TotalDuration = time.Since(start)
		return result
	}

	// Check if the error is retryable
	if policy.RetryableErrors != nil && !policy.RetryableErrors(err) {
		result.LastError = err
		result.TotalDuration = time.Since(start)
		logger.Debug("Error is not retryable", "error", err.Error())
		return result
	}

	// Perform retries
	delay := policy.InitialDelay
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		result.Attempts++

		// Calculate delay with optional jitter
		currentDelay := delay
		if policy.Jitter {
			// Add up to 20% jitter
			jitter := float64(delay) * 0.2 * rand.Float64()
			currentDelay = time.Duration(float64(delay) + jitter)
		}

		logger.Debug("Retrying operation",
			"attempt", attempt,
			"max_attempts", policy.MaxAttempts,
			"delay", currentDelay.String(),
			"previous_error", err.Error())

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			result.LastError = fmt.Errorf("retry cancelled: %w", ctx.Err())
			result.TotalDuration = time.Since(start)
			return result
		case <-time.After(currentDelay):
		}

		// Retry the operation
		err = operation(ctx)
		if err == nil {
			result.Success = true
			result.TotalDuration = time.Since(start)
			logger.Debug("Retry succeeded", "attempt", attempt)
			return result
		}

		// Check if the new error is retryable
		if policy.RetryableErrors != nil && !policy.RetryableErrors(err) {
			result.LastError = err
			result.TotalDuration = time.Since(start)
			logger.Debug("Error after retry is not retryable", "error", err.Error())
			return result
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * policy.Multiplier)
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
	}

	result.LastError = fmt.Errorf("operation failed after %d attempts: %w", result.Attempts, err)
	result.TotalDuration = time.Since(start)
	return result
}

// RetryWithDefault executes an operation with the default retry policy
func RetryWithDefault(ctx context.Context, operation RetryOperation) RetryResult {
	return Retry(ctx, DefaultRetryPolicy, operation)
}

// calculateBackoffDelay calculates the exponential backoff delay
func calculateBackoffDelay(attempt int, initialDelay time.Duration, maxDelay time.Duration, multiplier float64) time.Duration {
	if attempt <= 0 {
		return initialDelay
	}

	delay := float64(initialDelay) * math.Pow(multiplier, float64(attempt-1))
	if delay > float64(maxDelay) {
		return maxDelay
	}

	return time.Duration(delay)
}

// WithRetry wraps a function to add retry capability
func WithRetry(policy RetryPolicy, fn func() error) func() error {
	return func() error {
		result := Retry(context.Background(), policy, func(ctx context.Context) error {
			return fn()
		})
		return result.LastError
	}
}

// WithRetryContext wraps a context-aware function to add retry capability
func WithRetryContext(policy RetryPolicy, fn func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		result := Retry(ctx, policy, fn)
		return result.LastError
	}
}