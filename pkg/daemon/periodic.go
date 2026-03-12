// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/identity"
)

// ShouldRunReflection returns true if periodic reflection should run.
// This checks if reflection is enabled and enough time has elapsed since the last run.
func (d *Daemon) ShouldRunReflection() bool {
	return d.Scheduler.IsDue(TaskReflect)
}

// RunPeriodicReflection runs the periodic reflection analysis if due.
// Returns the result if reflection was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicReflection() *ReflectResult {
	if !d.ShouldRunReflection() {
		return nil
	}

	reflector := d.Reflector
	if reflector == nil {
		reflector = &defaultReflector{}
	}
	result, err := reflector.Reflect(d.Config.ReflectCreateIssues)
	if err != nil {
		return &ReflectResult{
			Error:   err,
			Message: fmt.Sprintf("Reflection failed: %v", err),
		}
	}

	if d.Config.ReflectOpenEnabled {
		if err := reflector.ReflectOpen(); err != nil {
			return &ReflectResult{
				Suggestions: result.Suggestions,
				Saved:       result.Saved,
				Error:       err,
				Message:     fmt.Sprintf("Reflection open failed: %v", err),
			}
		}
	}

	// Update last reflect time on success
	d.Scheduler.MarkRun(TaskReflect)

	return result
}

// LastReflectTime returns when reflection was last run.
// Returns zero time if reflection has never run.
func (d *Daemon) LastReflectTime() time.Time {
	return d.Scheduler.LastRunTime(TaskReflect)
}

// NextReflectTime returns when the next reflection is scheduled.
// Returns zero time if reflection is disabled.
func (d *Daemon) NextReflectTime() time.Time {
	return d.Scheduler.NextRunTime(TaskReflect)
}

// ShouldRunCleanup returns true if periodic session cleanup should run.
// This checks if cleanup is enabled and enough time has elapsed since the last run.
func (d *Daemon) ShouldRunCleanup() bool {
	return d.Scheduler.IsDue(TaskCleanup)
}

// CleanupResult contains the result of a cleanup operation.
type CleanupResult struct {
	Deleted int
	Error   error
	Message string
}

// RunPeriodicCleanup runs periodic cleanup if due.
// OpenCode handles session cleanup via TTL, so this only closes stale tmux windows.
func (d *Daemon) RunPeriodicCleanup() *CleanupResult {
	if !d.ShouldRunCleanup() {
		return nil
	}

	cleaner := d.Cleaner
	if cleaner == nil {
		cleaner = &defaultSessionCleaner{}
	}

	deleted, message, err := cleaner.Cleanup(d.Config)
	if err != nil {
		return &CleanupResult{
			Deleted: deleted,
			Error:   err,
			Message: message,
		}
	}

	d.Scheduler.MarkRun(TaskCleanup)

	return &CleanupResult{
		Deleted: deleted,
		Message: message,
	}
}

// LastCleanupTime returns when cleanup was last run.
// Returns zero time if cleanup has never run.
func (d *Daemon) LastCleanupTime() time.Time {
	return d.Scheduler.LastRunTime(TaskCleanup)
}

// NextCleanupTime returns when the next cleanup is scheduled.
// Returns zero time if cleanup is disabled.
func (d *Daemon) NextCleanupTime() time.Time {
	return d.Scheduler.NextRunTime(TaskCleanup)
}

// ShouldRunRecovery returns true if periodic recovery should run.
// This checks if recovery is enabled and enough time has elapsed since the last run.
func (d *Daemon) ShouldRunRecovery() bool {
	return d.Scheduler.IsDue(TaskRecovery)
}

// RecoveryResult contains the result of a recovery operation.
type RecoveryResult struct {
	ResumedCount int
	SkippedCount int
	Error        error
	Message      string
}

// RunPeriodicRecovery runs the periodic stuck agent recovery if due.
// Returns the result if recovery was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicRecovery() *RecoveryResult {
	if !d.ShouldRunRecovery() {
		return nil
	}

	// Get list of active agents via workspace + OpenCode discovery
	agents, err := GetActiveAgents()
	if err != nil {
		return &RecoveryResult{
			ResumedCount: 0,
			SkippedCount: 0,
			Error:        err,
			Message:      fmt.Sprintf("Recovery failed to list agents: %v", err),
		}
	}

	resumed := 0
	skipped := 0
	now := time.Now()

	for _, agent := range agents {
		// Skip agents without beads ID (can't resume without ID)
		if agent.BeadsID == "" {
			skipped++
			continue
		}

		// Skip agents that already reported Phase: Complete
		// (they're waiting for orchestrator review, not stuck)
		if strings.EqualFold(agent.Phase, "complete") {
			skipped++
			continue
		}

		// Check if agent is idle long enough to trigger recovery
		idleTime := now.Sub(agent.UpdatedAt)
		if idleTime < d.Config.RecoveryIdleThreshold {
			skipped++
			continue
		}

		// Check if we've attempted resume recently (rate limiting)
		if lastAttempt, exists := d.resumeAttempts[agent.BeadsID]; exists {
			timeSinceLastAttempt := now.Sub(lastAttempt)
			if timeSinceLastAttempt < d.Config.RecoveryRateLimit {
				skipped++
				if d.Config.Verbose {
					fmt.Printf("  Skipping %s: resumed %v ago (rate limit: %v)\n",
						agent.BeadsID, timeSinceLastAttempt.Round(time.Minute), d.Config.RecoveryRateLimit)
				}
				continue
			}
		}

		// Attempt to resume the agent
		if d.Config.Verbose {
			fmt.Printf("  Attempting recovery for %s (idle for %v)\n",
				agent.BeadsID, idleTime.Round(time.Minute))
		}

		if err := ResumeAgentByBeadsID(agent.BeadsID); err != nil {
			if d.Config.Verbose {
				fmt.Printf("  Failed to resume %s: %v\n", agent.BeadsID, err)
			}
			// Don't count failures toward resumed count, but don't retry immediately
			d.resumeAttempts[agent.BeadsID] = now
			skipped++
			continue
		}

		// Record successful resume attempt
		d.resumeAttempts[agent.BeadsID] = now
		resumed++

		if d.Config.Verbose {
			fmt.Printf("  Resumed %s successfully\n", agent.BeadsID)
		}
	}

	// Update last recovery time on success
	d.Scheduler.MarkRun(TaskRecovery)

	return &RecoveryResult{
		ResumedCount: resumed,
		SkippedCount: skipped,
		Error:        nil,
		Message:      fmt.Sprintf("Recovery attempted: %d resumed, %d skipped", resumed, skipped),
	}
}

// RegistryRefreshResult contains the result of a registry refresh operation.
type RegistryRefreshResult struct {
	Changed bool     // Whether the registry was updated
	Added   []string // Prefixes of newly discovered projects
	Removed []string // Prefixes of projects no longer found
	Error   error
	Message string
}

// RunPeriodicRegistryRefresh rebuilds the project registry if due.
// Returns non-nil result if refresh was run (even if no changes found).
func (d *Daemon) RunPeriodicRegistryRefresh() *RegistryRefreshResult {
	if !d.Scheduler.IsDue(TaskRegistryRefresh) {
		return nil
	}

	newRegistry, err := NewProjectRegistryWithGroups()
	if err != nil {
		// Fall back to kb-only registry
		newRegistry, err = NewProjectRegistry()
		if err != nil {
			return &RegistryRefreshResult{
				Error:   err,
				Message: fmt.Sprintf("Registry refresh failed: %v", err),
			}
		}
	}

	d.Scheduler.MarkRun(TaskRegistryRefresh)

	// Check if registry changed
	if d.ProjectRegistry != nil && d.ProjectRegistry.Equal(newRegistry) {
		return &RegistryRefreshResult{
			Changed: false,
			Message: "Registry unchanged",
		}
	}

	// Compute diff before updating
	added, removed := d.ProjectRegistry.Diff(newRegistry)

	// Update the registry and dependent state
	d.ProjectRegistry = newRegistry
	d.ProjectDirNames = BuildProjectDirNames(newRegistry)

	return &RegistryRefreshResult{
		Changed: true,
		Added:   added,
		Removed: removed,
		Message: fmt.Sprintf("Registry updated: +%d -%d projects", len(added), len(removed)),
	}
}

// NewProjectRegistryWithGroups delegates to identity.NewProjectRegistryWithGroups.
func NewProjectRegistryWithGroups() (*ProjectRegistry, error) {
	return identity.NewProjectRegistryWithGroups()
}

// LastRecoveryTime returns when recovery was last run.
// Returns zero time if recovery has never run.
func (d *Daemon) LastRecoveryTime() time.Time {
	return d.Scheduler.LastRunTime(TaskRecovery)
}

// NextRecoveryTime returns when the next recovery is scheduled.
// Returns zero time if recovery is disabled.
func (d *Daemon) NextRecoveryTime() time.Time {
	return d.Scheduler.NextRunTime(TaskRecovery)
}
