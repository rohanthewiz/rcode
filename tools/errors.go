package tools

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
)

// Error types for classification
type (
	// RetryableError represents an error that can be retried
	RetryableError struct {
		Err    error
		Reason string
	}

	// PermanentError represents an error that should not be retried
	PermanentError struct {
		Err    error
		Reason string
	}

	// RateLimitError represents a rate limit error with retry-after information
	RateLimitError struct {
		Err        error
		RetryAfter int // seconds to wait before retry
		Reason     string
	}
)

// Error implementations
func (e *RetryableError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("retryable error (%s): %v", e.Reason, e.Err)
	}
	return fmt.Sprintf("retryable error: %v", e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

func (e *PermanentError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("permanent error (%s): %v", e.Reason, e.Err)
	}
	return fmt.Sprintf("permanent error: %v", e.Err)
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded (retry after %ds): %v", e.RetryAfter, e.Err)
}

func (e *RateLimitError) Unwrap() error {
	return e.Err
}

// Constructor functions
func NewRetryableError(err error, reason string) error {
	return &RetryableError{Err: err, Reason: reason}
}

func NewPermanentError(err error, reason string) error {
	return &PermanentError{Err: err, Reason: reason}
}

func NewRateLimitError(err error, retryAfter int) error {
	return &RateLimitError{Err: err, RetryAfter: retryAfter, Reason: "rate limit exceeded"}
}

// IsRetryableError determines if an error should trigger a retry
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's explicitly marked as retryable
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return true
	}

	// Check if it's explicitly marked as permanent
	var permanent *PermanentError
	if errors.As(err, &permanent) {
		return false
	}

	// Check for rate limit errors (always retryable)
	var rateLimit *RateLimitError
	if errors.As(err, &rateLimit) {
		return true
	}

	// Check for network errors
	if isRetryableNetworkError(err) {
		return true
	}

	// Check for file system errors
	if isRetryableFileSystemError(err) {
		return true
	}

	// Check for specific error messages
	if isRetryableByMessage(err) {
		return true
	}

	// Default to not retryable
	return false
}

// isRetryableNetworkError checks if a network error is retryable
func isRetryableNetworkError(err error) bool {
	// Check for net.Error
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Timeout errors are retryable
		if netErr.Timeout() {
			return true
		}
		// Temporary errors are retryable
		if netErr.Temporary() {
			return true
		}
	}

	// Check for specific network errors
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		// URL errors often wrap other errors
		return IsRetryableError(urlErr.Err)
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		// Temporary DNS failures are retryable
		return dnsErr.IsTemporary || dnsErr.IsTimeout
	}

	// Check for connection errors
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		// Connection refused, reset, etc. are often transient
		if opErr.Op == "dial" || opErr.Op == "read" || opErr.Op == "write" {
			return true
		}
	}

	// Check for EOF errors (connection closed)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	return false
}

// isRetryableFileSystemError checks if a file system error is retryable
func isRetryableFileSystemError(err error) bool {
	// Check for path errors
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		// Check the underlying error
		return isRetryableSyscallError(pathErr.Err)
	}

	// Check for link errors
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		// Check the underlying error
		return isRetryableSyscallError(linkErr.Err)
	}

	// Direct syscall errors
	return isRetryableSyscallError(err)
}

// isRetryableSyscallError checks if a syscall error is retryable
func isRetryableSyscallError(err error) bool {
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}

	// List of retryable syscall errors
	switch errno {
	case syscall.EAGAIN, // Resource temporarily unavailable
		syscall.EINTR,     // Interrupted system call
		syscall.EBUSY,     // Device or resource busy
		syscall.ENFILE,    // Too many open files in system
		syscall.EMFILE,    // Too many open files
		syscall.ENOMEM,    // Out of memory
		syscall.ENOBUFS,   // No buffer space available
		syscall.ETIMEDOUT: // Operation timed out
		return true
	}

	return false
}

// isRetryableByMessage checks error messages for retryable patterns
func isRetryableByMessage(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	// Common retryable patterns
	retryablePatterns := []string{
		"timeout",
		"timed out",
		"temporary failure",
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"i/o timeout",
		"deadline exceeded",
		"service unavailable",
		"too many requests",
		"rate limit",
		"throttled",
		"try again",
		"resource busy",
		"locked",
		"concurrent",
		"conflict",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}

	// Common permanent error patterns
	permanentPatterns := []string{
		"permission denied",
		"access denied",
		"forbidden",
		"unauthorized",
		"not found",
		"invalid",
		"bad request",
		"malformed",
		"unsupported",
		"not implemented",
	}

	for _, pattern := range permanentPatterns {
		if strings.Contains(msg, pattern) {
			return false
		}
	}

	return false
}

// ClassifyError wraps an error with appropriate retry classification
func ClassifyError(err error) error {
	if err == nil {
		return nil
	}

	// Already classified
	var retryable *RetryableError
	var permanent *PermanentError
	var rateLimit *RateLimitError
	if errors.As(err, &retryable) || errors.As(err, &permanent) || errors.As(err, &rateLimit) {
		return err
	}

	// Classify based on error type
	if IsRetryableError(err) {
		return NewRetryableError(err, "transient failure")
	}

	return NewPermanentError(err, "non-retryable failure")
}

// WrapNetworkError classifies network errors appropriately
func WrapNetworkError(err error) error {
	if err == nil {
		return nil
	}

	if isRetryableNetworkError(err) {
		return NewRetryableError(err, "network error")
	}

	return NewPermanentError(err, "network error")
}

// WrapFileSystemError classifies file system errors appropriately
func WrapFileSystemError(err error) error {
	if err == nil {
		return nil
	}

	if isRetryableFileSystemError(err) {
		return NewRetryableError(err, "filesystem error")
	}

	return NewPermanentError(err, "filesystem error")
}
