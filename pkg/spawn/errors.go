// Package spawn provides error types for spawn failures with clear messages and recovery guidance.
package spawn

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"
)

// SpawnErrorKind categorizes spawn failures for user-friendly messaging.
type SpawnErrorKind int

const (
	// ErrKindConnection indicates network/connection failures.
	ErrKindConnection SpawnErrorKind = iota
	// ErrKindServer indicates OpenCode server is not responding.
	ErrKindServer
	// ErrKindSession indicates session creation/extraction failed.
	ErrKindSession
	// ErrKindProcess indicates opencode process failed to start.
	ErrKindProcess
	// ErrKindTimeout indicates operation timed out.
	ErrKindTimeout
	// ErrKindUnknown indicates unknown error type.
	ErrKindUnknown
)

// SpawnError wraps errors with user-friendly messages and recovery guidance.
type SpawnError struct {
	Kind      SpawnErrorKind
	Message   string
	Cause     error
	Retryable bool
}

func (e *SpawnError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *SpawnError) Unwrap() error {
	return e.Cause
}

// RecoveryGuidance returns actionable steps for the user.
func (e *SpawnError) RecoveryGuidance() string {
	switch e.Kind {
	case ErrKindConnection:
		return `Connection failed. Try:
  1. Check if OpenCode server is running: ps aux | grep opencode
  2. Start the server: opencode serve --port 4096
  3. Verify server URL with: orch status`

	case ErrKindServer:
		return `OpenCode server not responding. Try:
  1. Restart OpenCode: opencode serve --port 4096
  2. Check server logs for errors
  3. Verify port is not blocked: lsof -i :4096`

	case ErrKindSession:
		return `Failed to establish agent session. Try:
  1. Retry the spawn command
  2. Check OpenCode logs for errors
  3. Try with --inline flag for debugging: orch spawn --inline SKILL "task"`

	case ErrKindProcess:
		return `OpenCode process failed to start. Try:
  1. Check if opencode CLI is installed: which opencode
  2. Verify opencode is in PATH
  3. Try running directly: opencode run "test prompt"`

	case ErrKindTimeout:
		return `Operation timed out. Try:
  1. Check network connectivity
  2. Increase timeout if needed
  3. Check OpenCode server load with: orch status`

	default:
		return `Unknown error. Try:
  1. Check orch and OpenCode logs
  2. Run with --inline flag for debugging
  3. Check GitHub issues: https://github.com/sst/opencode/issues`
	}
}

// ClassifyError determines the error kind from the underlying error.
func ClassifyError(err error) SpawnErrorKind {
	if err == nil {
		return ErrKindUnknown
	}

	errStr := err.Error()

	// Check for connection refused / network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return ErrKindTimeout
		}
		return ErrKindConnection
	}

	// Check for syscall errors (e.g., connection refused)
	var syscallErr syscall.Errno
	if errors.As(err, &syscallErr) {
		if syscallErr == syscall.ECONNREFUSED {
			return ErrKindConnection
		}
	}

	// String-based detection for common patterns
	if strings.Contains(errStr, "connection refused") {
		return ErrKindConnection
	}
	if strings.Contains(errStr, "no such host") {
		return ErrKindConnection
	}
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return ErrKindTimeout
	}
	if strings.Contains(errStr, "session") || strings.Contains(errStr, "session ID") {
		return ErrKindSession
	}
	if strings.Contains(errStr, "failed to start") || strings.Contains(errStr, "exec:") {
		return ErrKindProcess
	}
	if strings.Contains(errStr, "status code") || strings.Contains(errStr, "unexpected status") {
		return ErrKindServer
	}

	return ErrKindUnknown
}

// IsRetryable returns true if the error is likely transient and retryable.
func IsRetryable(err error) bool {
	kind := ClassifyError(err)
	switch kind {
	case ErrKindConnection, ErrKindTimeout:
		return true
	case ErrKindServer:
		// Server errors may be retryable (e.g., 503 Service Unavailable)
		errStr := err.Error()
		if strings.Contains(errStr, "503") || strings.Contains(errStr, "502") {
			return true
		}
		return false
	default:
		return false
	}
}

// WrapSpawnError wraps an error with spawn-specific context.
func WrapSpawnError(err error, message string) *SpawnError {
	kind := ClassifyError(err)
	return &SpawnError{
		Kind:      kind,
		Message:   message,
		Cause:     err,
		Retryable: IsRetryable(err),
	}
}

// NewConnectionError creates a connection-specific spawn error.
func NewConnectionError(err error) *SpawnError {
	return &SpawnError{
		Kind:      ErrKindConnection,
		Message:   "Failed to connect to OpenCode server",
		Cause:     err,
		Retryable: true,
	}
}

// NewSessionError creates a session-specific spawn error.
func NewSessionError(err error) *SpawnError {
	return &SpawnError{
		Kind:      ErrKindSession,
		Message:   "Failed to establish agent session",
		Cause:     err,
		Retryable: false,
	}
}

// RetryConfig configures retry behavior for spawn operations.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	BackoffRate  float64 // Multiplier for exponential backoff
}

// DefaultRetryConfig returns sensible defaults for spawn retries.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		BackoffRate:  2.0,
	}
}

// RetryResult contains the result of a retry operation.
type RetryResult struct {
	Attempts int
	LastErr  error
}

// Retry executes fn with retries for transient errors.
// Returns the result of fn, or the last error if all retries fail.
func Retry[T any](cfg RetryConfig, fn func() (T, error)) (T, *RetryResult) {
	var lastErr error
	var zero T
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			return result, &RetryResult{Attempts: attempt}
		}

		lastErr = err

		// Don't retry non-retryable errors
		if !IsRetryable(err) {
			return zero, &RetryResult{Attempts: attempt, LastErr: lastErr}
		}

		// Don't sleep after the last attempt
		if attempt < cfg.MaxAttempts {
			time.Sleep(delay)
			// Exponential backoff with max cap
			delay = time.Duration(float64(delay) * cfg.BackoffRate)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}
	}

	return zero, &RetryResult{Attempts: cfg.MaxAttempts, LastErr: lastErr}
}

// FormatSpawnError formats a spawn error with recovery guidance for CLI output.
func FormatSpawnError(err error) string {
	var spawnErr *SpawnError
	if errors.As(err, &spawnErr) {
		var sb strings.Builder
		sb.WriteString("❌ Spawn failed: ")
		sb.WriteString(spawnErr.Message)
		if spawnErr.Cause != nil {
			sb.WriteString(fmt.Sprintf("\n   Cause: %v", spawnErr.Cause))
		}
		sb.WriteString("\n\n")
		sb.WriteString(spawnErr.RecoveryGuidance())
		return sb.String()
	}
	return fmt.Sprintf("❌ Spawn failed: %v", err)
}
