package tools

import (
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"syscall"
	"testing"
)

// TestErrorTypes tests the basic error type implementations
func TestErrorTypes(t *testing.T) {
	t.Run("RetryableError", func(t *testing.T) {
		baseErr := errors.New("base error")
		err := NewRetryableError(baseErr, "network issue")

		if !IsRetryableError(err) {
			t.Error("Expected RetryableError to be retryable")
		}

		var retryable *RetryableError
		if !errors.As(err, &retryable) {
			t.Error("Expected error to be RetryableError type")
		}

		if !errors.Is(err, baseErr) {
			t.Error("Expected error to wrap base error")
		}

		expectedMsg := "retryable error (network issue): base error"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("PermanentError", func(t *testing.T) {
		baseErr := errors.New("base error")
		err := NewPermanentError(baseErr, "invalid input")

		if IsRetryableError(err) {
			t.Error("Expected PermanentError to not be retryable")
		}

		var permanent *PermanentError
		if !errors.As(err, &permanent) {
			t.Error("Expected error to be PermanentError type")
		}

		expectedMsg := "permanent error (invalid input): base error"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("RateLimitError", func(t *testing.T) {
		baseErr := errors.New("too many requests")
		err := NewRateLimitError(baseErr, 30)

		if !IsRetryableError(err) {
			t.Error("Expected RateLimitError to be retryable")
		}

		var rateLimit *RateLimitError
		if !errors.As(err, &rateLimit) {
			t.Error("Expected error to be RateLimitError type")
		}

		if rateLimit.RetryAfter != 30 {
			t.Errorf("Expected RetryAfter to be 30, got %d", rateLimit.RetryAfter)
		}

		expectedMsg := "rate limit exceeded (retry after 30s): too many requests"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})
}

// TestNetworkErrorClassification tests network error detection
func TestNetworkErrorClassification(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "timeout error",
			err:       &net.OpError{Op: "dial", Err: &timeoutError{}},
			retryable: true,
		},
		{
			name:      "DNS temporary error",
			err:       &net.DNSError{IsTemporary: true},
			retryable: true,
		},
		{
			name:      "DNS timeout",
			err:       &net.DNSError{IsTimeout: true},
			retryable: true,
		},
		{
			name:      "DNS permanent error",
			err:       &net.DNSError{IsTemporary: false, IsTimeout: false},
			retryable: false,
		},
		{
			name:      "URL error wrapping timeout",
			err:       &url.Error{Err: &timeoutError{}},
			retryable: true,
		},
		{
			name:      "Connection refused",
			err:       &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			retryable: true,
		},
		{
			name:      "EOF error",
			err:       io.EOF,
			retryable: true,
		},
		{
			name:      "Unexpected EOF",
			err:       io.ErrUnexpectedEOF,
			retryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableNetworkError(tt.err)
			if result != tt.retryable {
				t.Errorf("isRetryableNetworkError(%v) = %v, want %v", tt.err, result, tt.retryable)
			}
		})
	}
}

// TestFileSystemErrorClassification tests file system error detection
func TestFileSystemErrorClassification(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "EAGAIN",
			err:       &os.PathError{Err: syscall.EAGAIN},
			retryable: true,
		},
		{
			name:      "EBUSY",
			err:       &os.PathError{Err: syscall.EBUSY},
			retryable: true,
		},
		{
			name:      "EINTR",
			err:       &os.PathError{Err: syscall.EINTR},
			retryable: true,
		},
		{
			name:      "ENOMEM",
			err:       &os.PathError{Err: syscall.ENOMEM},
			retryable: true,
		},
		{
			name:      "EACCES (permission denied)",
			err:       &os.PathError{Err: syscall.EACCES},
			retryable: false,
		},
		{
			name:      "ENOENT (not found)",
			err:       &os.PathError{Err: syscall.ENOENT},
			retryable: false,
		},
		{
			name:      "Link error with EBUSY",
			err:       &os.LinkError{Err: syscall.EBUSY},
			retryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableFileSystemError(tt.err)
			if result != tt.retryable {
				t.Errorf("isRetryableFileSystemError(%v) = %v, want %v", tt.err, result, tt.retryable)
			}
		})
	}
}

// TestErrorMessageClassification tests error message pattern matching
func TestErrorMessageClassification(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "timeout message",
			err:       errors.New("operation timed out"),
			retryable: true,
		},
		{
			name:      "temporary failure",
			err:       errors.New("temporary failure in connection"),
			retryable: true,
		},
		{
			name:      "connection refused",
			err:       errors.New("dial tcp: connection refused"),
			retryable: true,
		},
		{
			name:      "connection reset",
			err:       errors.New("read: connection reset by peer"),
			retryable: true,
		},
		{
			name:      "service unavailable",
			err:       errors.New("503 service unavailable"),
			retryable: true,
		},
		{
			name:      "rate limit",
			err:       errors.New("rate limit exceeded"),
			retryable: true,
		},
		{
			name:      "permission denied",
			err:       errors.New("permission denied"),
			retryable: false,
		},
		{
			name:      "access denied",
			err:       errors.New("access denied to resource"),
			retryable: false,
		},
		{
			name:      "not found",
			err:       errors.New("file not found"),
			retryable: false,
		},
		{
			name:      "invalid request",
			err:       errors.New("invalid request format"),
			retryable: false,
		},
		{
			name:      "unauthorized",
			err:       errors.New("401 unauthorized"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableByMessage(tt.err)
			if result != tt.retryable {
				t.Errorf("isRetryableByMessage(%v) = %v, want %v", tt.err, result, tt.retryable)
			}
		})
	}
}

// TestClassifyError tests the overall error classification
func TestClassifyError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		err := ClassifyError(nil)
		if err != nil {
			t.Error("Expected nil for nil input")
		}
	})

	t.Run("already classified", func(t *testing.T) {
		retryable := NewRetryableError(errors.New("test"), "test")
		classified := ClassifyError(retryable)
		if classified != retryable {
			t.Error("Expected already classified error to be returned as-is")
		}
	})

	t.Run("unclassified retryable", func(t *testing.T) {
		err := errors.New("connection timeout")
		classified := ClassifyError(err)

		var retryable *RetryableError
		if !errors.As(classified, &retryable) {
			t.Error("Expected timeout error to be classified as retryable")
		}
	})

	t.Run("unclassified permanent", func(t *testing.T) {
		err := errors.New("permission denied")
		classified := ClassifyError(err)

		var permanent *PermanentError
		if !errors.As(classified, &permanent) {
			t.Error("Expected permission error to be classified as permanent")
		}
	})
}

// TestWrappers tests the wrapper functions
func TestWrappers(t *testing.T) {
	t.Run("WrapNetworkError", func(t *testing.T) {
		// Retryable network error
		timeoutErr := &net.OpError{Op: "dial", Err: &timeoutError{}}
		wrapped := WrapNetworkError(timeoutErr)

		var retryable *RetryableError
		if !errors.As(wrapped, &retryable) {
			t.Error("Expected timeout to be wrapped as retryable")
		}

		// Non-retryable network error
		permErr := errors.New("permission denied")
		wrapped = WrapNetworkError(permErr)

		var permanent *PermanentError
		if !errors.As(wrapped, &permanent) {
			t.Error("Expected permission error to be wrapped as permanent")
		}
	})

	t.Run("WrapFileSystemError", func(t *testing.T) {
		// Retryable FS error
		busyErr := &os.PathError{Err: syscall.EBUSY}
		wrapped := WrapFileSystemError(busyErr)

		var retryable *RetryableError
		if !errors.As(wrapped, &retryable) {
			t.Error("Expected EBUSY to be wrapped as retryable")
		}

		// Non-retryable FS error
		notFoundErr := &os.PathError{Err: syscall.ENOENT}
		wrapped = WrapFileSystemError(notFoundErr)

		var permanent *PermanentError
		if !errors.As(wrapped, &permanent) {
			t.Error("Expected ENOENT to be wrapped as permanent")
		}
	})
}

// Mock timeout error for testing
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
