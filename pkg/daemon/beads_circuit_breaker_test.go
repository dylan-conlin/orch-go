package daemon

import (
	"testing"
	"time"
)

func TestBeadsCircuitBreaker_InitialState(t *testing.T) {
	cb := NewBeadsCircuitBreaker()

	if cb.IsOpen() {
		t.Error("new circuit breaker should be closed")
	}
	if cb.ConsecutiveFailures() != 0 {
		t.Errorf("consecutive failures = %d, want 0", cb.ConsecutiveFailures())
	}
	if cb.BackoffDuration() != 0 {
		t.Errorf("backoff = %s, want 0", cb.BackoffDuration())
	}
}

func TestBeadsCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewBeadsCircuitBreaker()

	// Record failures below threshold
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.IsOpen() {
		t.Error("circuit breaker should be closed with 2 failures (threshold=3)")
	}

	// Third failure should open it
	cb.RecordFailure()
	if !cb.IsOpen() {
		t.Error("circuit breaker should be open after 3 failures")
	}
	if cb.ConsecutiveFailures() != 3 {
		t.Errorf("consecutive failures = %d, want 3", cb.ConsecutiveFailures())
	}
}

func TestBeadsCircuitBreaker_SuccessResets(t *testing.T) {
	cb := NewBeadsCircuitBreaker()

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	if !cb.IsOpen() {
		t.Fatal("expected circuit to be open")
	}

	// Success should reset
	cb.RecordSuccess()
	if cb.IsOpen() {
		t.Error("circuit should be closed after success")
	}
	if cb.ConsecutiveFailures() != 0 {
		t.Errorf("consecutive failures = %d, want 0", cb.ConsecutiveFailures())
	}
	if cb.BackoffDuration() != 0 {
		t.Errorf("backoff = %s, want 0", cb.BackoffDuration())
	}
}

func TestBeadsCircuitBreaker_ExponentialBackoff(t *testing.T) {
	cb := NewBeadsCircuitBreaker()
	cb.MinBackoff = 30 * time.Second
	cb.MaxBackoff = 5 * time.Minute

	// Below threshold: no backoff
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.BackoffDuration() != 0 {
		t.Errorf("expected 0 backoff below threshold, got %s", cb.BackoffDuration())
	}

	// At threshold (3): MinBackoff
	cb.RecordFailure()
	if cb.BackoffDuration() != 30*time.Second {
		t.Errorf("backoff = %s, want 30s", cb.BackoffDuration())
	}

	// 4th failure: 2x MinBackoff
	cb.RecordFailure()
	if cb.BackoffDuration() != 60*time.Second {
		t.Errorf("backoff = %s, want 60s", cb.BackoffDuration())
	}

	// 5th failure: 4x MinBackoff
	cb.RecordFailure()
	if cb.BackoffDuration() != 120*time.Second {
		t.Errorf("backoff = %s, want 120s", cb.BackoffDuration())
	}

	// 6th failure: 8x MinBackoff = 240s
	cb.RecordFailure()
	if cb.BackoffDuration() != 240*time.Second {
		t.Errorf("backoff = %s, want 240s", cb.BackoffDuration())
	}

	// 7th failure: would be 480s but capped at MaxBackoff (300s)
	cb.RecordFailure()
	if cb.BackoffDuration() != 5*time.Minute {
		t.Errorf("backoff = %s, want 5m (max)", cb.BackoffDuration())
	}

	// More failures stay capped
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.BackoffDuration() != 5*time.Minute {
		t.Errorf("backoff = %s, want 5m (max cap)", cb.BackoffDuration())
	}
}

func TestBeadsCircuitBreaker_IntermittentSuccess(t *testing.T) {
	cb := NewBeadsCircuitBreaker()

	// Build up failures
	cb.RecordFailure()
	cb.RecordFailure()
	// Success resets
	cb.RecordSuccess()
	if cb.ConsecutiveFailures() != 0 {
		t.Error("expected 0 failures after success")
	}

	// Start building again
	cb.RecordFailure()
	if cb.ConsecutiveFailures() != 1 {
		t.Errorf("consecutive failures = %d, want 1", cb.ConsecutiveFailures())
	}
	if cb.IsOpen() {
		t.Error("expected closed after 1 failure")
	}
}

func TestBeadsCircuitBreaker_Status(t *testing.T) {
	cb := NewBeadsCircuitBreaker()

	status := cb.Status()
	if status != "closed" {
		t.Errorf("status = %q, want %q", status, "closed")
	}

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	status = cb.Status()
	if status == "closed" {
		t.Error("status should not be 'closed' after 3 failures")
	}
}
