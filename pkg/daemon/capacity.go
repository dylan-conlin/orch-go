// Package daemon provides autonomous overnight processing capabilities.
package daemon

// AvailableSlots returns the number of agent slots available for spawning.
// Returns a high number if no limit is set.
func (d *Daemon) AvailableSlots() int {
	// Use pool if available
	if d.Pool != nil {
		return d.Pool.Available()
	}
	// If no pool and no limit, unlimited slots available
	if d.Config.MaxAgents <= 0 {
		return 100 // No limit
	}
	// If pool not configured, assume unlimited
	return 100
}

// AtCapacity returns true if the daemon cannot spawn more agents.
func (d *Daemon) AtCapacity() bool {
	// Use pool if available
	if d.Pool != nil {
		return d.Pool.AtCapacity()
	}
	// If no pool configured, never at capacity
	return false
}

// ActiveCount returns the number of currently active agents.
func (d *Daemon) ActiveCount() int {
	if d.Pool != nil {
		return d.Pool.Active()
	}
	// If no pool configured, return 0 (cannot track active count without pool)
	return 0
}

// PoolStatus returns the current worker pool status for monitoring.
// Returns nil if no pool is configured.
func (d *Daemon) PoolStatus() *PoolStatus {
	if d.Pool == nil {
		return nil
	}
	status := d.Pool.Status()
	return &status
}

// RateLimitStatus returns the current rate limiter status for monitoring.
// Returns nil if no rate limiter is configured.
func (d *Daemon) RateLimitStatus() *RateLimiterStatus {
	if d.RateLimiter == nil {
		return nil
	}
	status := d.RateLimiter.Status()
	return &status
}

// RateLimited returns true if the daemon cannot spawn due to hourly rate limit.
func (d *Daemon) RateLimited() bool {
	if d.RateLimiter == nil {
		return false
	}
	canSpawn, _, _ := d.RateLimiter.CanSpawn()
	return !canSpawn
}

// RateLimitMessage returns a message if rate limited, or empty string if not.
func (d *Daemon) RateLimitMessage() string {
	if d.RateLimiter == nil {
		return ""
	}
	_, _, msg := d.RateLimiter.CanSpawn()
	return msg
}

// ReconcileWithOpenCode synchronizes the worker pool with actual OpenCode sessions.
// This prevents the pool from becoming stuck at capacity when agents complete
// without the daemon knowing (e.g., overnight runs, crashes, manual kills).
//
// Also cleans up stale entries from the spawned issue tracker.
//
// Should be called at the start of each poll cycle.
// Returns the number of slots freed due to reconciliation, or 0 if no pool.
func (d *Daemon) ReconcileWithOpenCode() int {
	// Clean up stale spawned issue entries (older than TTL)
	if d.SpawnedIssues != nil {
		d.SpawnedIssues.CleanStale()
	}

	if d.Pool == nil {
		return 0
	}

	// Get actual count from OpenCode API
	actualCount := DefaultActiveCount()

	// Reconcile pool with actual count
	return d.Pool.Reconcile(actualCount)
}
