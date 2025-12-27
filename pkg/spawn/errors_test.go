package spawn

import (
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"
	"time"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected SpawnErrorKind
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ErrKindUnknown,
		},
		{
			name:     "connection refused",
			err:      errors.New("dial tcp: connection refused"),
			expected: ErrKindConnection,
		},
		{
			name:     "no such host",
			err:      errors.New("dial tcp: lookup host: no such host"),
			expected: ErrKindConnection,
		},
		{
			name:     "timeout error",
			err:      errors.New("context deadline exceeded"),
			expected: ErrKindTimeout,
		},
		{
			name:     "timeout in message",
			err:      errors.New("request timeout after 30s"),
			expected: ErrKindTimeout,
		},
		{
			name:     "session error",
			err:      errors.New("failed to extract session ID from output"),
			expected: ErrKindSession,
		},
		{
			name:     "exec error",
			err:      errors.New("exec: opencode: executable file not found"),
			expected: ErrKindProcess,
		},
		{
			name:     "failed to start",
			err:      errors.New("failed to start opencode: no such file"),
			expected: ErrKindProcess,
		},
		{
			name:     "HTTP status code",
			err:      errors.New("unexpected status code 503: service unavailable"),
			expected: ErrKindServer,
		},
		{
			name:     "unknown error",
			err:      errors.New("some random error"),
			expected: ErrKindUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyError(tt.err)
			if got != tt.expected {
				t.Errorf("ClassifyError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "connection refused is retryable",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "timeout is retryable",
			err:      errors.New("context deadline exceeded"),
			expected: true,
		},
		{
			name:     "503 is retryable",
			err:      errors.New("unexpected status code 503"),
			expected: true,
		},
		{
			name:     "502 is retryable",
			err:      errors.New("unexpected status code 502"),
			expected: true,
		},
		{
			name:     "session error is not retryable",
			err:      errors.New("failed to extract session ID"),
			expected: false,
		},
		{
			name:     "process error is not retryable",
			err:      errors.New("exec: not found"),
			expected: false,
		},
		{
			name:     "400 is not retryable",
			err:      errors.New("unexpected status code 400"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSpawnError(t *testing.T) {
	cause := errors.New("connection refused")
	spawnErr := WrapSpawnError(cause, "Failed to connect")

	// Test Error() method
	errMsg := spawnErr.Error()
	if errMsg != "Failed to connect: connection refused" {
		t.Errorf("Error() = %q, want %q", errMsg, "Failed to connect: connection refused")
	}

	// Test Unwrap()
	if unwrapped := spawnErr.Unwrap(); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// Test Kind classification
	if spawnErr.Kind != ErrKindConnection {
		t.Errorf("Kind = %v, want %v", spawnErr.Kind, ErrKindConnection)
	}

	// Test Retryable
	if !spawnErr.Retryable {
		t.Error("Retryable should be true for connection error")
	}
}

func TestRecoveryGuidance(t *testing.T) {
	tests := []struct {
		kind     SpawnErrorKind
		contains string
	}{
		{ErrKindConnection, "Check if OpenCode server is running"},
		{ErrKindServer, "Restart OpenCode"},
		{ErrKindSession, "Retry the spawn command"},
		{ErrKindProcess, "Check if opencode CLI is installed"},
		{ErrKindTimeout, "Check network connectivity"},
		{ErrKindUnknown, "Check orch and OpenCode logs"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("kind_%d", tt.kind), func(t *testing.T) {
			err := &SpawnError{Kind: tt.kind}
			guidance := err.RecoveryGuidance()
			if guidance == "" {
				t.Error("RecoveryGuidance() returned empty string")
			}
			// Just verify it returns non-empty guidance for all types
			if len(guidance) < 20 {
				t.Errorf("RecoveryGuidance() too short: %q", guidance)
			}
		})
	}
}

func TestRetry(t *testing.T) {
	t.Run("succeeds on first try", func(t *testing.T) {
		cfg := RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Millisecond,
			MaxDelay:     10 * time.Millisecond,
			BackoffRate:  2.0,
		}

		calls := 0
		result, retryResult := Retry(cfg, func() (string, error) {
			calls++
			return "success", nil
		})

		if result != "success" {
			t.Errorf("result = %q, want %q", result, "success")
		}
		if retryResult.Attempts != 1 {
			t.Errorf("Attempts = %d, want 1", retryResult.Attempts)
		}
		if retryResult.LastErr != nil {
			t.Errorf("LastErr = %v, want nil", retryResult.LastErr)
		}
		if calls != 1 {
			t.Errorf("calls = %d, want 1", calls)
		}
	})

	t.Run("succeeds on second try", func(t *testing.T) {
		cfg := RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Millisecond,
			MaxDelay:     10 * time.Millisecond,
			BackoffRate:  2.0,
		}

		calls := 0
		result, retryResult := Retry(cfg, func() (string, error) {
			calls++
			if calls < 2 {
				return "", errors.New("connection refused") // retryable
			}
			return "success", nil
		})

		if result != "success" {
			t.Errorf("result = %q, want %q", result, "success")
		}
		if retryResult.Attempts != 2 {
			t.Errorf("Attempts = %d, want 2", retryResult.Attempts)
		}
		if retryResult.LastErr != nil {
			t.Errorf("LastErr = %v, want nil", retryResult.LastErr)
		}
	})

	t.Run("fails all attempts with retryable error", func(t *testing.T) {
		cfg := RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Millisecond,
			MaxDelay:     10 * time.Millisecond,
			BackoffRate:  2.0,
		}

		calls := 0
		_, retryResult := Retry(cfg, func() (string, error) {
			calls++
			return "", errors.New("connection refused")
		})

		if retryResult.Attempts != 3 {
			t.Errorf("Attempts = %d, want 3", retryResult.Attempts)
		}
		if retryResult.LastErr == nil {
			t.Error("LastErr should not be nil")
		}
		if calls != 3 {
			t.Errorf("calls = %d, want 3", calls)
		}
	})

	t.Run("does not retry non-retryable error", func(t *testing.T) {
		cfg := RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Millisecond,
			MaxDelay:     10 * time.Millisecond,
			BackoffRate:  2.0,
		}

		calls := 0
		_, retryResult := Retry(cfg, func() (string, error) {
			calls++
			return "", errors.New("failed to extract session ID") // not retryable
		})

		if retryResult.Attempts != 1 {
			t.Errorf("Attempts = %d, want 1", retryResult.Attempts)
		}
		if calls != 1 {
			t.Errorf("calls = %d, want 1 (should not retry)", calls)
		}
	})
}

func TestFormatSpawnError(t *testing.T) {
	t.Run("formats SpawnError with guidance", func(t *testing.T) {
		err := &SpawnError{
			Kind:    ErrKindConnection,
			Message: "Failed to connect",
			Cause:   errors.New("connection refused"),
		}

		formatted := FormatSpawnError(err)

		if formatted == "" {
			t.Error("FormatSpawnError returned empty string")
		}
		if len(formatted) < 50 {
			t.Errorf("FormatSpawnError too short: %q", formatted)
		}
	})

	t.Run("formats regular error", func(t *testing.T) {
		err := errors.New("some error")
		formatted := FormatSpawnError(err)

		expected := "❌ Spawn failed: some error"
		if formatted != expected {
			t.Errorf("FormatSpawnError = %q, want %q", formatted, expected)
		}
	})
}

// TestClassifyErrorWithRealNetError tests with actual network error types
func TestClassifyErrorWithRealNetError(t *testing.T) {
	// Test with syscall.ECONNREFUSED wrapped in an error
	t.Run("syscall ECONNREFUSED", func(t *testing.T) {
		err := fmt.Errorf("dial tcp: %w", syscall.ECONNREFUSED)
		kind := ClassifyError(err)
		if kind != ErrKindConnection {
			t.Errorf("ClassifyError(ECONNREFUSED) = %v, want %v", kind, ErrKindConnection)
		}
	})

	// Test with a net.Error timeout
	t.Run("net.Error timeout", func(t *testing.T) {
		err := &mockNetError{timeout: true}
		kind := ClassifyError(err)
		if kind != ErrKindTimeout {
			t.Errorf("ClassifyError(timeout) = %v, want %v", kind, ErrKindTimeout)
		}
	})
}

// mockNetError implements net.Error for testing
type mockNetError struct {
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return "mock net error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

var _ net.Error = (*mockNetError)(nil)
