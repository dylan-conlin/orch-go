package main

import (
	"errors"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestShouldRetrySessionIDExtraction(t *testing.T) {
	tests := []struct {
		name       string
		retry      *spawn.RetryResult
		expectedOK bool
	}{
		{
			name:       "nil retry result",
			retry:      nil,
			expectedOK: false,
		},
		{
			name: "nil last error",
			retry: &spawn.RetryResult{
				Attempts: 1,
				LastErr:  nil,
			},
			expectedOK: false,
		},
		{
			name: "spawn session error with extraction message",
			retry: &spawn.RetryResult{
				Attempts: 1,
				LastErr: &spawn.SpawnError{
					Kind:    spawn.ErrKindSession,
					Message: "Failed to extract session ID: no session ID found in output",
					Cause:   errors.New("no session ID found in output"),
				},
			},
			expectedOK: true,
		},
		{
			name: "spawn session error with unrelated message",
			retry: &spawn.RetryResult{
				Attempts: 1,
				LastErr: &spawn.SpawnError{
					Kind:    spawn.ErrKindSession,
					Message: "Failed to establish agent session",
					Cause:   errors.New("session not found"),
				},
			},
			expectedOK: false,
		},
		{
			name: "plain error string still detected",
			retry: &spawn.RetryResult{
				Attempts: 1,
				LastErr:  errors.New("Failed to extract session ID"),
			},
			expectedOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRetrySessionIDExtraction(tt.retry)
			if got != tt.expectedOK {
				t.Fatalf("shouldRetrySessionIDExtraction() = %v, want %v", got, tt.expectedOK)
			}
		})
	}
}

func TestRetryHeadlessSpawnAfterSessionIDExtractionFailure(t *testing.T) {
	t.Run("successful retry clears error and increments attempts", func(t *testing.T) {
		retryResult := &spawn.RetryResult{
			Attempts: 1,
			LastErr: &spawn.SpawnError{
				Kind:    spawn.ErrKindSession,
				Message: "Failed to extract session ID: no session ID found in output",
				Cause:   errors.New("no session ID found in output"),
			},
		}

		initial := &headlessSpawnResult{SessionID: "initial"}
		calls := 0
		recovered, retried := retryHeadlessSpawnAfterSessionIDExtractionFailure(initial, retryResult, 0, func() (*headlessSpawnResult, error) {
			calls++
			return &headlessSpawnResult{SessionID: "recovered"}, nil
		})

		if !retried {
			t.Fatal("expected retried=true")
		}
		if calls != 1 {
			t.Fatalf("retry function calls = %d, want 1", calls)
		}
		if retryResult.Attempts != 2 {
			t.Fatalf("Attempts = %d, want 2", retryResult.Attempts)
		}
		if retryResult.LastErr != nil {
			t.Fatalf("LastErr = %v, want nil", retryResult.LastErr)
		}
		if recovered == nil || recovered.SessionID != "recovered" {
			t.Fatalf("recovered result = %#v, want sessionID 'recovered'", recovered)
		}
	})

	t.Run("failed retry keeps failure and increments attempts", func(t *testing.T) {
		retryResult := &spawn.RetryResult{
			Attempts: 1,
			LastErr: &spawn.SpawnError{
				Kind:    spawn.ErrKindSession,
				Message: "Failed to extract session ID: no session ID found in output",
				Cause:   errors.New("no session ID found in output"),
			},
		}

		initial := &headlessSpawnResult{SessionID: "initial"}
		retryErr := errors.New("retry attempt failed")
		recovered, retried := retryHeadlessSpawnAfterSessionIDExtractionFailure(initial, retryResult, 0, func() (*headlessSpawnResult, error) {
			return nil, retryErr
		})

		if !retried {
			t.Fatal("expected retried=true")
		}
		if retryResult.Attempts != 2 {
			t.Fatalf("Attempts = %d, want 2", retryResult.Attempts)
		}
		if !errors.Is(retryResult.LastErr, retryErr) {
			t.Fatalf("LastErr = %v, want retry error %v", retryResult.LastErr, retryErr)
		}
		if recovered == nil || recovered.SessionID != "initial" {
			t.Fatalf("recovered result = %#v, want initial result", recovered)
		}
	})

	t.Run("non-extraction errors are not retried", func(t *testing.T) {
		retryResult := &spawn.RetryResult{
			Attempts: 1,
			LastErr:  errors.New("connection refused"),
		}

		initial := &headlessSpawnResult{SessionID: "initial"}
		calls := 0
		recovered, retried := retryHeadlessSpawnAfterSessionIDExtractionFailure(initial, retryResult, 0, func() (*headlessSpawnResult, error) {
			calls++
			return &headlessSpawnResult{SessionID: "should-not-run"}, nil
		})

		if retried {
			t.Fatal("expected retried=false")
		}
		if calls != 0 {
			t.Fatalf("retry function calls = %d, want 0", calls)
		}
		if retryResult.Attempts != 1 {
			t.Fatalf("Attempts = %d, want 1", retryResult.Attempts)
		}
		if recovered == nil || recovered.SessionID != "initial" {
			t.Fatalf("recovered result = %#v, want initial result", recovered)
		}
	})

	t.Run("applies configured retry delay before reattempt", func(t *testing.T) {
		retryResult := &spawn.RetryResult{
			Attempts: 1,
			LastErr: &spawn.SpawnError{
				Kind:    spawn.ErrKindSession,
				Message: "Failed to extract session ID: no session ID found in output",
				Cause:   errors.New("no session ID found in output"),
			},
		}

		initial := &headlessSpawnResult{SessionID: "initial"}
		wantDelay := 175 * time.Millisecond

		originalSleep := sessionIDExtractionRetrySleep
		defer func() {
			sessionIDExtractionRetrySleep = originalSleep
		}()

		var gotDelay time.Duration
		sessionIDExtractionRetrySleep = func(delay time.Duration) {
			gotDelay = delay
		}

		_, retried := retryHeadlessSpawnAfterSessionIDExtractionFailure(initial, retryResult, wantDelay, func() (*headlessSpawnResult, error) {
			return &headlessSpawnResult{SessionID: "recovered"}, nil
		})

		if !retried {
			t.Fatal("expected retried=true")
		}
		if gotDelay != wantDelay {
			t.Fatalf("delay = %s, want %s", gotDelay, wantDelay)
		}
	})
}
