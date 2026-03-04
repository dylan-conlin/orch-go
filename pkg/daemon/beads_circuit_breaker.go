package daemon

import (
	"fmt"
	"sync"
	"time"
)

// BeadsCircuitBreaker tracks consecutive bd command failures and provides
// exponential backoff to prevent lock cascade when beads is unhealthy.
//
// When the beads daemon or JSONL lock is stuck, the orch daemon's polling
// creates new bd processes faster than they can complete, causing an
// unkillable lock pileup. The circuit breaker detects this condition
// (consecutive failures) and backs off exponentially.
type BeadsCircuitBreaker struct {
	mu                  sync.Mutex
	consecutiveFailures int
	lastFailure         time.Time
	lastSuccess         time.Time

	// FailureThreshold is the number of consecutive failures before
	// the circuit opens and backoff activates. Default: 3.
	FailureThreshold int

	// MinBackoff is the initial backoff duration when circuit opens. Default: 30s.
	MinBackoff time.Duration

	// MaxBackoff is the maximum backoff cap. Default: 5m.
	MaxBackoff time.Duration
}

// NewBeadsCircuitBreaker creates a circuit breaker with sensible defaults.
func NewBeadsCircuitBreaker() *BeadsCircuitBreaker {
	return &BeadsCircuitBreaker{
		FailureThreshold: 3,
		MinBackoff:       30 * time.Second,
		MaxBackoff:       5 * time.Minute,
	}
}

// RecordSuccess resets the failure counter. Call after any successful bd command.
func (cb *BeadsCircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.consecutiveFailures = 0
	cb.lastSuccess = time.Now()
}

// RecordFailure increments the failure counter. Call after any failed bd command.
func (cb *BeadsCircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.consecutiveFailures++
	cb.lastFailure = time.Now()
}

// IsOpen returns true when consecutive failures have exceeded the threshold,
// indicating beads is unhealthy and the daemon should back off.
func (cb *BeadsCircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.consecutiveFailures >= cb.FailureThreshold
}

// ConsecutiveFailures returns the current failure count.
func (cb *BeadsCircuitBreaker) ConsecutiveFailures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.consecutiveFailures
}

// BackoffDuration returns the current backoff duration based on failure count.
// Returns 0 if circuit is closed (failures below threshold).
// Uses exponential backoff: MinBackoff * 2^(failures - threshold), capped at MaxBackoff.
func (cb *BeadsCircuitBreaker) BackoffDuration() time.Duration {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.consecutiveFailures < cb.FailureThreshold {
		return 0
	}

	exponent := cb.consecutiveFailures - cb.FailureThreshold
	backoff := cb.MinBackoff
	for i := 0; i < exponent; i++ {
		backoff *= 2
		if backoff > cb.MaxBackoff {
			return cb.MaxBackoff
		}
	}
	return backoff
}

// Status returns a human-readable status string for logging.
func (cb *BeadsCircuitBreaker) Status() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.consecutiveFailures < cb.FailureThreshold {
		return "closed"
	}
	return fmt.Sprintf("open (failures=%d, backoff=%s)",
		cb.consecutiveFailures, cb.BackoffDurationLocked())
}

// BackoffDurationLocked is the internal version without mutex (caller must hold lock).
func (cb *BeadsCircuitBreaker) BackoffDurationLocked() time.Duration {
	if cb.consecutiveFailures < cb.FailureThreshold {
		return 0
	}
	exponent := cb.consecutiveFailures - cb.FailureThreshold
	backoff := cb.MinBackoff
	for i := 0; i < exponent; i++ {
		backoff *= 2
		if backoff > cb.MaxBackoff {
			return cb.MaxBackoff
		}
	}
	return backoff
}
