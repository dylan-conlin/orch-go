package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// bdSubprocessLimiter prevents bd subprocess stampede in orch serve.
//
// Problem: Under load (12+ agents), multiple dashboard polls can trigger hundreds
// of concurrent bd subprocesses (bd ready, bd stats, bd list, bd show, bd dep list,
// bd comments, etc.) which locks up the entire system.
//
// Solution: Two-layer protection:
//  1. singleflight.Group per operation type — deduplicates concurrent identical requests.
//     If 10 dashboard polls all want bd stats at the same time, only 1 subprocess runs.
//  2. Semaphore (max 5 concurrent) — hard cap on total bd subprocesses from serve.
//     Even with cache misses across different operation types, we never exceed 5.
//
// This is more correct than pure rate-limiting because singleflight serves ALL
// concurrent waiters the same result from a single subprocess call.
type bdLimiter struct {
	// Singleflight groups — one per operation category.
	// Using separate groups ensures that a bd stats call doesn't block a bd ready call.
	statsGroup     singleflight.Group // bd stats
	readyGroup     singleflight.Group // bd ready
	listGroup      singleflight.Group // bd list (for graph)
	showGroup      singleflight.Group // bd show <id>
	depGroup       singleflight.Group // bd dep list <id>
	commentsGroup  singleflight.Group // bd comments <id>
	frontierGroup  singleflight.Group // frontier.CalculateFrontier
	questionsGroup singleflight.Group // handleQuestions (bd list --type question)
	attemptsGroup  singleflight.Group // collectAttemptHistory

	// Hard concurrency limit — semaphore pattern.
	// No more than maxConcurrent bd subprocesses at any time from serve.
	sem           chan struct{}
	maxConcurrent int

	// Metrics for observability
	inflight   atomic.Int64 // Current number of bd subprocesses running
	totalCalls atomic.Int64 // Total bd subprocess calls made
	dedupCalls atomic.Int64 // Calls served from singleflight dedup (avoided subprocess)
}

// Global bd limiter, initialized in runServe
var globalBdLimiter *bdLimiter

const (
	// maxBdConcurrent is the hard cap on concurrent bd subprocesses from serve.
	// With 12+ agents and 5s polling, even cache misses should never exceed this.
	// If all 5 slots are full, new requests wait (with timeout) rather than spawning more.
	maxBdConcurrent = 5

	// bdAcquireTimeout is how long to wait for a semaphore slot before giving up.
	// Dashboard can tolerate stale data — better to return cached/error than deadlock.
	bdAcquireTimeout = 10 * time.Second
)

// newBdLimiter creates a new bd subprocess limiter with the configured concurrency cap.
func newBdLimiter() *bdLimiter {
	return &bdLimiter{
		sem:           make(chan struct{}, maxBdConcurrent),
		maxConcurrent: maxBdConcurrent,
	}
}

// acquire reserves a slot for a bd subprocess call.
// Returns a release function that MUST be called when the subprocess completes.
// Returns error if the timeout expires before a slot is available.
func (l *bdLimiter) acquire(ctx context.Context) (func(), error) {
	select {
	case l.sem <- struct{}{}:
		l.inflight.Add(1)
		l.totalCalls.Add(1)
		return func() {
			<-l.sem
			l.inflight.Add(-1)
		}, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("bd limiter: timeout waiting for slot (inflight=%d, max=%d)", l.inflight.Load(), l.maxConcurrent)
	}
}

// acquireWithTimeout is a convenience wrapper using the default timeout.
func (l *bdLimiter) acquireWithTimeout() (func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), bdAcquireTimeout)
	defer cancel()
	return l.acquire(ctx)
}

// stats returns current limiter metrics for diagnostics.
func (l *bdLimiter) stats() (inflight, total, deduped int64) {
	return l.inflight.Load(), l.totalCalls.Load(), l.dedupCalls.Load()
}

// bdLimitedFunc wraps a function that spawns bd subprocesses with the concurrency limiter.
// The function f will only be called when a semaphore slot is available.
// This is used as the inner function for singleflight.Do — singleflight deduplicates,
// and when the single call actually runs, it acquires a semaphore slot.
func bdLimitedFunc[T any](l *bdLimiter, f func() (T, error)) (T, error) {
	release, err := l.acquireWithTimeout()
	if err != nil {
		var zero T
		return zero, err
	}
	defer release()
	return f()
}

// bdSingleflightDo executes f through both singleflight dedup and concurrency limiting.
// This is the primary entry point for bd subprocess calls in serve handlers.
//
// Parameters:
//   - group: the singleflight.Group for this operation type
//   - key: dedup key (e.g., "stats:default" or "show:orch-go-12345")
//   - f: the function that actually spawns the bd subprocess
//
// Returns the result, error, and whether this was a shared result (deduped).
func bdSingleflightDo[T any](l *bdLimiter, group *singleflight.Group, key string, f func() (T, error)) (T, error, bool) {
	result, err, shared := group.Do(key, func() (interface{}, error) {
		return bdLimitedFunc(l, f)
	})

	if shared {
		l.dedupCalls.Add(1)
	}

	if err != nil {
		var zero T
		return zero, err, shared
	}

	return result.(T), nil, shared
}

// logLimiterStats periodically logs limiter metrics for observability.
// Call this as a goroutine during serve startup.
func logLimiterStats(l *bdLimiter, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastTotal, lastDeduped int64

	for {
		select {
		case <-ticker.C:
			inflight, total, deduped := l.stats()
			// Only log if there's been activity since last log
			if total != lastTotal {
				log.Printf("[bd-limiter] inflight=%d total=%d deduped=%d (saved %d%% subprocess calls)",
					inflight, total, deduped, safePct(deduped, total))
				lastTotal = total
				lastDeduped = deduped
			}
			_ = lastDeduped // suppress unused
		case <-stop:
			return
		}
	}
}

// safePct calculates a percentage safely, avoiding division by zero.
func safePct(part, total int64) int64 {
	if total == 0 {
		return 0
	}
	return (part * 100) / total
}

// bdLimiterStatsResponse is the JSON structure for the bd limiter stats API.
// Exposed via /health endpoint for observability.
type bdLimiterStatsResponse struct {
	Inflight      int64 `json:"inflight"`
	MaxConcurrent int   `json:"max_concurrent"`
	TotalCalls    int64 `json:"total_calls"`
	DedupedCalls  int64 `json:"deduped_calls"`
	DedupPct      int64 `json:"dedup_pct"`
}

// getLimiterStats returns current limiter stats for API response.
func getLimiterStats() *bdLimiterStatsResponse {
	if globalBdLimiter == nil {
		return nil
	}
	inflight, total, deduped := globalBdLimiter.stats()
	return &bdLimiterStatsResponse{
		Inflight:      inflight,
		MaxConcurrent: globalBdLimiter.maxConcurrent,
		TotalCalls:    total,
		DedupedCalls:  deduped,
		DedupPct:      safePct(deduped, total),
	}
}

// --- Wrapper functions for each bd subprocess call site ---
// These replace direct calls to beads.FallbackStats(), beads.FallbackReady(), etc.
// Each one deduplicates via singleflight and rate-limits via semaphore.

// bdLimitedStats wraps a bd stats call with singleflight + concurrency limiter.
// key should identify the project (e.g., "" for default or the projectDir).
func bdLimitedStats(key string, f func() (interface{}, error)) (interface{}, error, bool) {
	if globalBdLimiter == nil {
		result, err := f()
		return result, err, false
	}
	return bdSingleflightDo(globalBdLimiter, &globalBdLimiter.statsGroup, "stats:"+key, func() (interface{}, error) {
		return f()
	})
}

// bdLimitedReady wraps a bd ready call with singleflight + concurrency limiter.
func bdLimitedReady(key string, f func() (interface{}, error)) (interface{}, error, bool) {
	if globalBdLimiter == nil {
		result, err := f()
		return result, err, false
	}
	return bdSingleflightDo(globalBdLimiter, &globalBdLimiter.readyGroup, "ready:"+key, func() (interface{}, error) {
		return f()
	})
}

// bdLimitedList wraps a bd list call with singleflight + concurrency limiter.
func bdLimitedList(key string, f func() (interface{}, error)) (interface{}, error, bool) {
	if globalBdLimiter == nil {
		result, err := f()
		return result, err, false
	}
	return bdSingleflightDo(globalBdLimiter, &globalBdLimiter.listGroup, "list:"+key, func() (interface{}, error) {
		return f()
	})
}

// bdLimitedShow wraps a bd show call with singleflight + concurrency limiter.
func bdLimitedShow(key string, f func() (interface{}, error)) (interface{}, error, bool) {
	if globalBdLimiter == nil {
		result, err := f()
		return result, err, false
	}
	return bdSingleflightDo(globalBdLimiter, &globalBdLimiter.showGroup, "show:"+key, func() (interface{}, error) {
		return f()
	})
}

// bdLimitedDep wraps a bd dep list call with singleflight + concurrency limiter.
func bdLimitedDep(key string, f func() (interface{}, error)) (interface{}, error, bool) {
	if globalBdLimiter == nil {
		result, err := f()
		return result, err, false
	}
	return bdSingleflightDo(globalBdLimiter, &globalBdLimiter.depGroup, "dep:"+key, func() (interface{}, error) {
		return f()
	})
}

// bdLimitedComments wraps a bd comments call with singleflight + concurrency limiter.
func bdLimitedComments(key string, f func() (interface{}, error)) (interface{}, error, bool) {
	if globalBdLimiter == nil {
		result, err := f()
		return result, err, false
	}
	return bdSingleflightDo(globalBdLimiter, &globalBdLimiter.commentsGroup, "comments:"+key, func() (interface{}, error) {
		return f()
	})
}

// bdLimitedFrontier wraps a frontier calculation with singleflight + concurrency limiter.
func bdLimitedFrontier(f func() (interface{}, error)) (interface{}, error, bool) {
	if globalBdLimiter == nil {
		result, err := f()
		return result, err, false
	}
	return bdSingleflightDo(globalBdLimiter, &globalBdLimiter.frontierGroup, "frontier", func() (interface{}, error) {
		return f()
	})
}

// bdLimitedQuestions wraps a questions fetch with singleflight + concurrency limiter.
func bdLimitedQuestions(f func() (interface{}, error)) (interface{}, error, bool) {
	if globalBdLimiter == nil {
		result, err := f()
		return result, err, false
	}
	return bdSingleflightDo(globalBdLimiter, &globalBdLimiter.questionsGroup, "questions", func() (interface{}, error) {
		return f()
	})
}

// bdLimitedAttempts wraps attempt history collection with singleflight + concurrency limiter.
func bdLimitedAttempts(key string, f func() (interface{}, error)) (interface{}, error, bool) {
	if globalBdLimiter == nil {
		result, err := f()
		return result, err, false
	}
	return bdSingleflightDo(globalBdLimiter, &globalBdLimiter.attemptsGroup, "attempts:"+key, func() (interface{}, error) {
		return f()
	})
}

// bdLimitedCreate wraps a bd create call with ONLY the concurrency limiter (no singleflight).
// Creates are unique operations — we never want to deduplicate them.
func bdLimitedCreate(f func() (interface{}, error)) (interface{}, error) {
	if globalBdLimiter == nil {
		return f()
	}
	return bdLimitedFunc(globalBdLimiter, func() (interface{}, error) {
		return f()
	})
}

// compileSentinel is checked at compile time to ensure this package compiles.
// The sync import is used by singleflight groups (embedded sync.Mutex).
var _ sync.Mutex
