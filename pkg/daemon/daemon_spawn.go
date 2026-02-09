// daemon_spawn.go contains spawn execution, capacity management, and pool/rate methods.
package daemon

import (
	"fmt"
	"net/http"
	"time"
)

// AvailableSlots returns the number of agent slots available for spawning.
// Returns a high number if no limit is set.
func (d *Daemon) AvailableSlots() int {
	// Use pool if available
	if d.Pool != nil {
		return d.Pool.Available()
	}
	// Fallback to legacy activeCountFunc
	if d.Config.MaxAgents <= 0 {
		return 100 // No limit
	}
	active := d.activeCountFunc()
	available := d.Config.MaxAgents - active
	if available < 0 {
		return 0
	}
	return available
}

// AtCapacity returns true if the daemon cannot spawn more agents.
func (d *Daemon) AtCapacity() bool {
	// Use pool if available
	if d.Pool != nil {
		return d.Pool.AtCapacity()
	}
	// Fallback to legacy activeCountFunc
	if d.Config.MaxAgents <= 0 {
		return false // No limit
	}
	return d.activeCountFunc() >= d.Config.MaxAgents
}

// ActiveCount returns the number of currently active agents.
func (d *Daemon) ActiveCount() int {
	if d.Pool != nil {
		return d.Pool.Active()
	}
	return d.activeCountFunc()
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

// CheckServerHealth checks if the OpenCode server is reachable and updates
// the server recovery state. This enables detection of server restarts by
// tracking when the server goes down and comes back up.
//
// Should be called at the start of each poll cycle, before RunServerRecovery.
// Returns true if the server is reachable, false otherwise.
func (d *Daemon) CheckServerHealth() bool {
	if d.serverRecoveryState == nil {
		return true // No recovery state to update
	}

	serverURL := d.Config.CleanupServerURL
	if serverURL == "" {
		serverURL = "http://127.0.0.1:4096"
	}

	// Make a simple HTTP request to check if server is reachable
	// Use a short timeout to avoid blocking the poll loop
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(serverURL + "/session")
	available := err == nil && resp != nil && resp.StatusCode == http.StatusOK
	if resp != nil {
		resp.Body.Close()
	}

	// Update the recovery state with server health
	d.serverRecoveryState.UpdateServerHealth(available)

	return available
}

// ReconcileWithOpenCode synchronizes the worker pool with actual active agents.
// This prevents the pool from becoming stuck at capacity when agents complete
// without the daemon knowing (e.g., overnight runs, crashes, manual kills).
//
// The counting method depends on the configured backend:
// - "docker": counts running Docker containers with claude-code-mcp image
// - "opencode" or others: queries OpenCode API for active sessions
//
// Also cleans up stale entries from the spawned issue tracker, and clears
// entries for issues that have been abandoned (allowing them to be respawned).
//
// Should be called at the start of each poll cycle.
// Returns the number of slots freed due to reconciliation, or 0 if no pool.
func (d *Daemon) ReconcileWithOpenCode() int {
	// Reload ProcessedCache from disk so cache-clear commands from other processes
	// are reflected without restarting the daemon.
	if d.ProcessedCache != nil {
		if err := d.ProcessedCache.Reload(); err != nil && d.Config.Verbose {
			fmt.Printf("  DEBUG: Failed to reload processed cache: %v\n", err)
		}
	}

	// Clean up stale spawned issue entries (older than TTL)
	if d.SpawnedIssues != nil {
		d.SpawnedIssues.CleanStale()

		// Clear entries for issues that were abandoned via `orch abandon`.
		// This allows the daemon to respawn issues after they're abandoned and
		// re-labeled with triage:ready. We look at abandon events from the last
		// 7 hours (slightly longer than the 6h TTL) to ensure we catch all
		// abandons that occurred within the tracker's TTL window.
		abandonedIDs, err := GetRecentlyAbandonedIssues(7)
		if err == nil && len(abandonedIDs) > 0 {
			cleared := d.SpawnedIssues.ClearAbandoned(abandonedIDs)
			if cleared > 0 && d.Config.Verbose {
				fmt.Printf("  DEBUG: Cleared %d abandoned issues from spawn tracker\n", cleared)
			}
		}
	}

	if d.Pool == nil {
		return 0
	}

	// Get actual count using the configured counting function
	// (DockerActiveCount for docker backend, DefaultActiveCount otherwise)
	// Fall back to DefaultActiveCount if activeCountFunc is not set (e.g., in tests).
	countFunc := d.activeCountFunc
	if countFunc == nil {
		countFunc = DefaultActiveCount
	}
	actualCount := countFunc()

	// Reconcile pool with actual count
	return d.Pool.Reconcile(actualCount)
}

// Once processes a single issue from the queue and returns.
// If a worker pool is configured, it acquires a slot before spawning.
// Note: The slot is NOT automatically released when the agent completes.
// Use OnceWithSlot() for explicit slot management, or ReleaseSlot() manually.
func (d *Daemon) Once() (*OnceResult, error) {
	return d.OnceExcluding(nil)
}

// OnceExcluding processes a single issue from the queue, excluding skipped issues.
// This allows the daemon to skip issues that failed to spawn (e.g., due to failure
// report gate) and continue processing other issues in the queue.
//
// The skip map should contain issue IDs that should be skipped this cycle.
// If a worker pool is configured, it acquires a slot before spawning.
// If a rate limiter is configured, it checks the hourly limit before spawning.
func (d *Daemon) OnceExcluding(skip map[string]bool) (*OnceResult, error) {
	// Check rate limit first (before fetching issues)
	if d.RateLimiter != nil {
		canSpawn, count, msg := d.RateLimiter.CanSpawn()
		if !canSpawn {
			if d.Config.Verbose {
				fmt.Printf("  Rate limited: %s\n", msg)
			}
			return &OnceResult{
				Processed: false,
				Message:   fmt.Sprintf("Rate limited: %d/%d spawns in the last hour", count, d.RateLimiter.MaxPerHour),
			}, nil
		}
	}

	// Create extended skip set that includes issues skipped due to session/completion checks.
	// This fixes the bug where the daemon stops looking if the highest-priority
	// issue has an existing session or Phase: Complete.
	extendedSkip := make(map[string]bool)
	for k, v := range skip {
		extendedSkip[k] = v
	}

	var issue *Issue
	var skill string
	var skippedReasons []string

	for {
		var err error
		issue, err = d.NextIssueExcluding(extendedSkip)
		if err != nil {
			return nil, err
		}

		if issue == nil {
			// No more issues to try
			if len(skippedReasons) > 0 {
				return &OnceResult{
					Processed: false,
					Message:   fmt.Sprintf("No spawnable issues (skipped: %v)", skippedReasons),
				}, nil
			}
			return &OnceResult{
				Processed: false,
				Message:   "No spawnable issues in queue",
			}, nil
		}

		var skillErr error
		skill, skillErr = InferSkillFromIssue(issue)
		if skillErr != nil {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (failed to infer skill: %v)\n", issue.ID, skillErr)
			}
			extendedSkip[issue.ID] = true
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s: failed to infer skill", issue.ID))
			continue
		}

		// Unified dedup check: Use ProcessedCache to consolidate three checks:
		// 1. Persistent cache (survives daemon restart)
		// 2. Session dedup (checks OpenCode sessions)
		// 3. Phase Complete (checks beads comments)
		if d.ProcessedCache != nil && !d.ProcessedCache.ShouldProcess(issue.ID) {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (blocked by ProcessedCache)\n", issue.ID)
			}
			// Emit telemetry event when cache blocks spawn
			if d.EventLogger != nil {
				_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
					"beads_id":    issue.ID,
					"dedup_layer": "processed_cache",
					"reason":      "Issue blocked by unified ProcessedCache",
				})
			}
			extendedSkip[issue.ID] = true
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s: already processed", issue.ID))
			continue
		}

		// Synthesis completion check: prevent spawning for synthesis topics
		// that already have a guide/decision. Defense-in-depth against kb-cli
		// dedup failure (JSON parse errors cause "no duplicate" to be returned).
		// See: orch-go-qu8fj, orch-go-bn6io
		if reason := CheckSynthesisCompletion(issue, getProjectDir()); reason != "" {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (%s)\n", issue.ID, reason)
			}
			if d.EventLogger != nil {
				_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
					"beads_id":    issue.ID,
					"dedup_layer": "synthesis_completion",
					"reason":      reason,
				})
			}
			extendedSkip[issue.ID] = true
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s: %s", issue.ID, reason))
			continue
		}

		// Found an issue that passes all checks
		break
	}

	// If pool is configured, acquire a slot first
	var slot *Slot
	if d.Pool != nil {
		slot = d.Pool.TryAcquire()
		if slot == nil {
			return &OnceResult{
				Processed: false,
				Issue:     issue,
				Skill:     skill,
				Message:   "At capacity - no slots available",
			}, nil
		}
		slot.BeadsID = issue.ID
	}

	// Mark in legacy tracker before spawn to preserve the race-window dedup behavior.
	// ProcessedCache is marked only after confirmed successful spawn.
	if d.SpawnedIssues != nil {
		d.SpawnedIssues.MarkSpawned(issue.ID)
	}

	// Spawn the work
	if err := d.spawnFunc(issue.ID); err != nil {
		if d.SpawnedIssues != nil {
			d.SpawnedIssues.Unmark(issue.ID)
		}
		// Release slot on spawn failure
		if d.Pool != nil && slot != nil {
			d.Pool.Release(slot)
		}
		return &OnceResult{
			Processed: false,
			Issue:     issue,
			Skill:     skill,
			Error:     err,
			Message:   fmt.Sprintf("Failed to spawn: %v", err),
		}, nil
	}

	// Mark in persistent processed cache only after successful spawn.
	if d.ProcessedCache != nil {
		if err := d.ProcessedCache.MarkProcessed(issue.ID); err != nil {
			fmt.Printf("warning: failed to mark issue as processed: %v\n", err)
		}
	}

	// Record successful spawn for rate limiting
	if d.RateLimiter != nil {
		d.RateLimiter.RecordSpawn()
	}

	return &OnceResult{
		Processed: true,
		Issue:     issue,
		Skill:     skill,
		Message:   fmt.Sprintf("Spawned work on %s", issue.ID),
	}, nil
}

// OnceWithSlot processes a single issue and returns the acquired slot.
// The caller is responsible for releasing the slot when the agent completes.
// Returns (result, slot, error). Slot will be nil if no pool is configured or if spawn failed.
func (d *Daemon) OnceWithSlot() (*OnceResult, *Slot, error) {
	// Check rate limit first (before fetching issues)
	if d.RateLimiter != nil {
		canSpawn, count, msg := d.RateLimiter.CanSpawn()
		if !canSpawn {
			if d.Config.Verbose {
				fmt.Printf("  Rate limited: %s\n", msg)
			}
			return &OnceResult{
				Processed: false,
				Message:   fmt.Sprintf("Rate limited: %d/%d spawns in the last hour", count, d.RateLimiter.MaxPerHour),
			}, nil, nil
		}
	}

	issue, err := d.NextIssue()
	if err != nil {
		return nil, nil, err
	}

	if issue == nil {
		return &OnceResult{
			Processed: false,
			Message:   "No spawnable issues in queue",
		}, nil, nil
	}

	skill, err := InferSkillFromIssue(issue)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to infer skill: %w", err)
	}

	// Unified dedup check: Use ProcessedCache to consolidate three checks
	if d.ProcessedCache != nil && !d.ProcessedCache.ShouldProcess(issue.ID) {
		if d.Config.Verbose {
			fmt.Printf("  DEBUG: Skipping %s (blocked by ProcessedCache)\n", issue.ID)
		}
		// Emit telemetry event when cache blocks spawn
		if d.EventLogger != nil {
			_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
				"beads_id":    issue.ID,
				"dedup_layer": "processed_cache",
				"reason":      "Issue blocked by unified ProcessedCache",
			})
		}
		return &OnceResult{
			Processed: false,
			Issue:     issue,
			Skill:     skill,
			Message:   fmt.Sprintf("Skipping %s: already processed", issue.ID),
		}, nil, nil
	}

	// If pool is configured, acquire a slot first
	var slot *Slot
	if d.Pool != nil {
		slot = d.Pool.TryAcquire()
		if slot == nil {
			return &OnceResult{
				Processed: false,
				Issue:     issue,
				Skill:     skill,
				Message:   "At capacity - no slots available",
			}, nil, nil
		}
		slot.BeadsID = issue.ID
	}

	// Mark in legacy tracker before spawn to preserve the race-window dedup behavior.
	// ProcessedCache is marked only after confirmed successful spawn.
	if d.SpawnedIssues != nil {
		d.SpawnedIssues.MarkSpawned(issue.ID)
	}

	// Spawn the work
	if err := d.spawnFunc(issue.ID); err != nil {
		if d.SpawnedIssues != nil {
			d.SpawnedIssues.Unmark(issue.ID)
		}
		// Release slot on spawn failure
		if d.Pool != nil && slot != nil {
			d.Pool.Release(slot)
		}
		return &OnceResult{
			Processed: false,
			Issue:     issue,
			Skill:     skill,
			Error:     err,
			Message:   fmt.Sprintf("Failed to spawn: %v", err),
		}, nil, nil
	}

	// Mark in persistent processed cache only after successful spawn.
	if d.ProcessedCache != nil {
		if err := d.ProcessedCache.MarkProcessed(issue.ID); err != nil {
			fmt.Printf("warning: failed to mark issue as processed: %v\n", err)
		}
	}

	// Record successful spawn for rate limiting
	if d.RateLimiter != nil {
		d.RateLimiter.RecordSpawn()
	}

	return &OnceResult{
		Processed: true,
		Issue:     issue,
		Skill:     skill,
		Message:   fmt.Sprintf("Spawned work on %s", issue.ID),
	}, slot, nil
}

// ReleaseSlot releases a previously acquired slot.
// Safe to call with nil slot.
func (d *Daemon) ReleaseSlot(slot *Slot) {
	if d.Pool != nil && slot != nil {
		d.Pool.Release(slot)
	}
}

// Run processes issues in a loop until the queue is empty or maxIterations is reached.
// Returns a slice of results for each processed issue.
func (d *Daemon) Run(maxIterations int) ([]*OnceResult, error) {
	var results []*OnceResult

	for i := 0; i < maxIterations; i++ {
		result, err := d.Once()
		if err != nil {
			return results, err
		}

		// Queue is empty
		if !result.Processed {
			break
		}

		results = append(results, result)
	}

	return results, nil
}
