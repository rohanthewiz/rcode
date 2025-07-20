package tools

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestRetryWithSuccess tests successful operation after retries
func TestRetryWithSuccess(t *testing.T) {
	attempts := 0
	operation := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil // Success on third attempt
	}

	policy := RetryPolicy{
		MaxAttempts:     3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		Multiplier:      2,
		Jitter:          false,
		RetryableErrors: func(err error) bool { return true },
	}

	result := Retry(context.Background(), policy, operation)

	if !result.Success {
		t.Errorf("Expected success, got failure: %v", result.LastError)
	}
	if result.Attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", result.Attempts)
	}
}

// TestRetryWithPermanentError tests that permanent errors don't retry
func TestRetryWithPermanentError(t *testing.T) {
	attempts := 0
	permanentErr := NewPermanentError(errors.New("permanent error"), "test")

	operation := func(ctx context.Context) error {
		attempts++
		return permanentErr
	}

	policy := RetryPolicy{
		MaxAttempts:     3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		Multiplier:      2,
		Jitter:          false,
		RetryableErrors: IsRetryableError,
	}

	result := Retry(context.Background(), policy, operation)

	if result.Success {
		t.Error("Expected failure for permanent error")
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt for permanent error, got %d", attempts)
	}
}

// TestRetryWithContextCancellation tests context cancellation
func TestRetryWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	operation := func(ctx context.Context) error {
		attempts++
		if attempts == 2 {
			cancel() // Cancel on second attempt
		}
		return errors.New("temporary failure")
	}

	policy := RetryPolicy{
		MaxAttempts:     5,
		InitialDelay:    50 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		Multiplier:      2,
		Jitter:          false,
		RetryableErrors: func(err error) bool { return true },
	}

	result := Retry(ctx, policy, operation)

	if result.Success {
		t.Error("Expected failure due to cancellation")
	}
	if attempts > 3 {
		t.Errorf("Expected at most 3 attempts before cancellation, got %d", attempts)
	}
	if !errors.Is(result.LastError, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", result.LastError)
	}
}

// TestExponentialBackoff tests the exponential backoff timing
func TestExponentialBackoff(t *testing.T) {
	attempts := 0
	startTime := time.Now()
	var attemptTimes []time.Duration

	operation := func(ctx context.Context) error {
		attempts++
		attemptTimes = append(attemptTimes, time.Since(startTime))
		if attempts < 4 {
			return errors.New("temporary failure")
		}
		return nil
	}

	policy := RetryPolicy{
		MaxAttempts:     3,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        1000 * time.Millisecond,
		Multiplier:      2,
		Jitter:          false,
		RetryableErrors: func(err error) bool { return true },
	}

	result := Retry(context.Background(), policy, operation)

	if !result.Success {
		t.Errorf("Expected success, got failure: %v", result.LastError)
	}

	// Check delays between attempts (should be approximately 100ms, 200ms, 400ms)
	if len(attemptTimes) < 4 {
		t.Fatalf("Expected 4 attempt times, got %d", len(attemptTimes))
	}

	// First attempt should be immediate
	if attemptTimes[0] > 10*time.Millisecond {
		t.Errorf("First attempt should be immediate, got %v", attemptTimes[0])
	}

	// Check subsequent delays
	expectedDelays := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}
	for i := 1; i < len(attemptTimes); i++ {
		actualDelay := attemptTimes[i] - attemptTimes[i-1]
		expectedDelay := expectedDelays[i-1]

		// Allow 20% tolerance for timing
		minDelay := expectedDelay * 8 / 10
		maxDelay := expectedDelay * 12 / 10

		if actualDelay < minDelay || actualDelay > maxDelay {
			t.Errorf("Attempt %d: expected delay ~%v, got %v", i+1, expectedDelay, actualDelay)
		}
	}
}

// TestRetryableErrorClassification tests error classification
func TestRetryableErrorClassification(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "retryable error",
			err:       NewRetryableError(errors.New("network timeout"), "timeout"),
			retryable: true,
		},
		{
			name:      "permanent error",
			err:       NewPermanentError(errors.New("permission denied"), "auth"),
			retryable: false,
		},
		{
			name:      "rate limit error",
			err:       NewRateLimitError(errors.New("too many requests"), 60),
			retryable: true,
		},
		{
			name:      "timeout error message",
			err:       errors.New("operation timed out"),
			retryable: true,
		},
		{
			name:      "permission denied message",
			err:       errors.New("permission denied"),
			retryable: false,
		},
		{
			name:      "connection refused message",
			err:       errors.New("connection refused"),
			retryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.retryable {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, result, tt.retryable)
			}
		})
	}
}

// TestJitter tests that jitter adds randomness to delays
func TestJitter(t *testing.T) {
	delays := make([]time.Duration, 5)

	for i := 0; i < 5; i++ {
		attempts := 0
		operation := func(ctx context.Context) error {
			attempts++
			if attempts == 2 {
				return nil // Success on second attempt
			}
			return errors.New("temporary failure")
		}

		policy := RetryPolicy{
			MaxAttempts:     3,
			InitialDelay:    100 * time.Millisecond,
			MaxDelay:        1000 * time.Millisecond,
			Multiplier:      2,
			Jitter:          true,
			RetryableErrors: func(err error) bool { return true },
		}

		startTime := time.Now()
		Retry(context.Background(), policy, operation)
		delays[i] = time.Since(startTime)
	}

	// Check that not all delays are exactly the same (jitter is working)
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("Expected jitter to produce different delays, but all were the same")
	}
}

// TestNetworkRetryPolicy tests the network-specific retry policy
func TestNetworkRetryPolicy(t *testing.T) {
	if NetworkRetryPolicy.MaxAttempts != 5 {
		t.Errorf("Expected NetworkRetryPolicy.MaxAttempts to be 5, got %d", NetworkRetryPolicy.MaxAttempts)
	}
	if NetworkRetryPolicy.InitialDelay != 500*time.Millisecond {
		t.Errorf("Expected NetworkRetryPolicy.InitialDelay to be 500ms, got %v", NetworkRetryPolicy.InitialDelay)
	}
	if NetworkRetryPolicy.MaxDelay != 30*time.Second {
		t.Errorf("Expected NetworkRetryPolicy.MaxDelay to be 30s, got %v", NetworkRetryPolicy.MaxDelay)
	}
}

// TestFileSystemRetryPolicy tests the file system-specific retry policy
func TestFileSystemRetryPolicy(t *testing.T) {
	if FileSystemRetryPolicy.MaxAttempts != 2 {
		t.Errorf("Expected FileSystemRetryPolicy.MaxAttempts to be 2, got %d", FileSystemRetryPolicy.MaxAttempts)
	}
	if FileSystemRetryPolicy.InitialDelay != 50*time.Millisecond {
		t.Errorf("Expected FileSystemRetryPolicy.InitialDelay to be 50ms, got %v", FileSystemRetryPolicy.InitialDelay)
	}
	if FileSystemRetryPolicy.MaxDelay != 500*time.Millisecond {
		t.Errorf("Expected FileSystemRetryPolicy.MaxDelay to be 500ms, got %v", FileSystemRetryPolicy.MaxDelay)
	}
}

// TestRetryWithDefault tests the convenience function
func TestRetryWithDefault(t *testing.T) {
	attempts := 0
	operation := func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	}

	result := RetryWithDefault(context.Background(), operation)

	if result.LastError != nil {
		t.Errorf("Expected success with default policy, got error: %v", result.LastError)
	}
	if !result.Success {
		t.Error("Expected success with default policy")
	}
}

// TestWithRetryWrapper tests the wrapper functions
func TestWithRetryWrapper(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	}

	policy := RetryPolicy{
		MaxAttempts:     3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		Multiplier:      2,
		RetryableErrors: func(err error) bool { return true },
	}

	wrappedFn := WithRetry(policy, fn)
	err := wrappedFn()

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

// BenchmarkRetry benchmarks the retry mechanism
func BenchmarkRetry(b *testing.B) {
	operation := func(ctx context.Context) error {
		return nil // Always succeed immediately
	}

	policy := DefaultRetryPolicy

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Retry(context.Background(), policy, operation)
	}
}
