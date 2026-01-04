// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"time"
)

// RateLimiter tracks spawn history to enforce hourly rate limits.
type RateLimiter struct {
	// MaxPerHour is the maximum spawns allowed per hour (0 = no limit).
	MaxPerHour int
	// SpawnHistory tracks timestamps of recent spawns.
	SpawnHistory []time.Time
	// nowFunc allows injecting time for testing.
	nowFunc func() time.Time
}

// NewRateLimiter creates a new rate limiter with the given limit.
func NewRateLimiter(maxPerHour int) *RateLimiter {
	return &RateLimiter{
		MaxPerHour:   maxPerHour,
		SpawnHistory: make([]time.Time, 0),
		nowFunc:      time.Now,
	}
}

// CanSpawn returns true if spawning is allowed under the hourly rate limit.
// Returns (allowed bool, spawnsInLastHour int, message string).
func (r *RateLimiter) CanSpawn() (bool, int, string) {
	if r.MaxPerHour <= 0 {
		return true, 0, ""
	}

	now := r.nowFunc()
	oneHourAgo := now.Add(-time.Hour)

	// Count spawns in the last hour
	count := 0
	for _, t := range r.SpawnHistory {
		if t.After(oneHourAgo) {
			count++
		}
	}

	if count >= r.MaxPerHour {
		return false, count, fmt.Sprintf("Rate limit reached: %d/%d spawns in the last hour", count, r.MaxPerHour)
	}

	return true, count, ""
}

// RecordSpawn records a spawn at the current time.
func (r *RateLimiter) RecordSpawn() {
	now := r.nowFunc()
	r.SpawnHistory = append(r.SpawnHistory, now)
	r.prune()
}

// prune removes spawn history older than 1 hour to prevent unbounded growth.
func (r *RateLimiter) prune() {
	now := r.nowFunc()
	oneHourAgo := now.Add(-time.Hour)

	// Find first entry that's within the hour
	cutoff := 0
	for i, t := range r.SpawnHistory {
		if t.After(oneHourAgo) {
			cutoff = i
			break
		}
		cutoff = i + 1 // All entries are old
	}

	if cutoff > 0 {
		r.SpawnHistory = r.SpawnHistory[cutoff:]
	}
}

// SpawnsRemaining returns how many spawns are available before hitting the limit.
// Returns a high number if no limit is set.
func (r *RateLimiter) SpawnsRemaining() int {
	if r.MaxPerHour <= 0 {
		return 100 // No limit
	}

	_, count, _ := r.CanSpawn()
	remaining := r.MaxPerHour - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RateLimiterStatus returns current rate limiter status for monitoring.
type RateLimiterStatus struct {
	MaxPerHour      int
	SpawnsLastHour  int
	SpawnsRemaining int
	LimitReached    bool
}

// Status returns the current rate limiter status.
func (r *RateLimiter) Status() RateLimiterStatus {
	canSpawn, count, _ := r.CanSpawn()
	remaining := r.MaxPerHour - count
	if remaining < 0 {
		remaining = 0
	}
	return RateLimiterStatus{
		MaxPerHour:      r.MaxPerHour,
		SpawnsLastHour:  count,
		SpawnsRemaining: remaining,
		LimitReached:    !canSpawn,
	}
}
