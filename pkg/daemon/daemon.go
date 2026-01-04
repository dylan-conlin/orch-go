// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// Config holds configuration for the daemon.
type Config struct {
	// PollInterval is the time between polling cycles (0 = run once).
	PollInterval time.Duration

	// MaxAgents is the maximum number of concurrent agents (0 = no limit).
	MaxAgents int

	// MaxSpawnsPerHour is the maximum number of spawns allowed per hour (0 = no limit).
	// This prevents runaway spawning when many issues are batch-labeled as triage:ready.
	MaxSpawnsPerHour int

	// Label filters issues to only those with this label (empty = no filter).
	Label string

	// SpawnDelay is the delay between spawns to avoid rate limits.
	SpawnDelay time.Duration

	// DryRun shows what would be processed without spawning.
	DryRun bool

	// Verbose enables detailed output.
	Verbose bool
}

// DefaultConfig returns sensible defaults for daemon configuration.
func DefaultConfig() Config {
	return Config{
		PollInterval:     time.Minute,
		MaxAgents:        3,
		MaxSpawnsPerHour: 20, // Prevents runaway spawning
		Label:            "triage:ready",
		SpawnDelay:       10 * time.Second,
		DryRun:           false,
		Verbose:          false,
	}
}

// PreviewResult contains the result of a preview operation.
type PreviewResult struct {
	Issue           *Issue
	Skill           string
	Message         string
	RateLimited     bool             // True if rate limit would prevent spawning
	RateStatus      string           // Rate limit status message (e.g., "5/20 spawns in last hour")
	HotspotWarnings []HotspotWarning // Warnings about hotspot areas this issue may touch
}

// HasHotspotWarnings returns true if there are any hotspot warnings.
func (r *PreviewResult) HasHotspotWarnings() bool {
	return len(r.HotspotWarnings) > 0
}

// HasCriticalHotspots returns true if any hotspot warning is critical (score >= 10).
func (r *PreviewResult) HasCriticalHotspots() bool {
	for _, w := range r.HotspotWarnings {
		if w.IsCritical() {
			return true
		}
	}
	return false
}

// OnceResult contains the result of processing one issue.
type OnceResult struct {
	Processed bool
	Issue     *Issue
	Skill     string
	Message   string
	Error     error
}

// Daemon manages autonomous issue processing.
type Daemon struct {
	// Config holds the daemon configuration.
	Config Config

	// Pool is the worker pool for concurrency control.
	// If set, it is used instead of activeCountFunc.
	Pool *WorkerPool

	// RateLimiter tracks spawn history for hourly rate limiting.
	RateLimiter *RateLimiter

	// HotspotChecker checks for hotspot areas before spawning.
	// If set, Preview will include hotspot warnings.
	HotspotChecker HotspotChecker

	// listIssuesFunc is used for testing - allows mocking bd list
	listIssuesFunc func() ([]Issue, error)
	// spawnFunc is used for testing - allows mocking orch work
	spawnFunc func(beadsID string) error
	// activeCountFunc is used for testing - allows mocking active agent count
	// Deprecated: Use Pool for concurrency control instead.
	activeCountFunc func() int
	// listCompletedAgentsFunc is used for testing - allows mocking completed agents list
	listCompletedAgentsFunc func(CompletionConfig) ([]CompletedAgent, error)
}

// New creates a new Daemon instance with default configuration.
func New() *Daemon {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new Daemon instance with the given configuration.
func NewWithConfig(config Config) *Daemon {
	d := &Daemon{
		Config:          config,
		listIssuesFunc:  ListReadyIssues,
		spawnFunc:       SpawnWork,
		activeCountFunc: DefaultActiveCount,
	}
	// Initialize worker pool if MaxAgents is set
	if config.MaxAgents > 0 {
		d.Pool = NewWorkerPool(config.MaxAgents)
	}
	// Initialize rate limiter if MaxSpawnsPerHour is set
	if config.MaxSpawnsPerHour > 0 {
		d.RateLimiter = NewRateLimiter(config.MaxSpawnsPerHour)
	}
	return d
}

// NewWithPool creates a new Daemon instance with an explicit worker pool.
// This is useful for sharing a pool across daemon instances or for testing.
func NewWithPool(config Config, pool *WorkerPool) *Daemon {
	d := &Daemon{
		Config:          config,
		Pool:            pool,
		listIssuesFunc:  ListReadyIssues,
		spawnFunc:       SpawnWork,
		activeCountFunc: DefaultActiveCount,
	}
	// Initialize rate limiter if MaxSpawnsPerHour is set
	if config.MaxSpawnsPerHour > 0 {
		d.RateLimiter = NewRateLimiter(config.MaxSpawnsPerHour)
	}
	return d
}

// NextIssue returns the next spawnable issue from the queue.
// Returns nil if no spawnable issues are available.
// Issues are sorted by priority (0 = highest priority).
// If a label filter is configured, only issues with that label are considered.
func (d *Daemon) NextIssue() (*Issue, error) {
	return d.NextIssueExcluding(nil)
}

// NextIssueExcluding returns the next spawnable issue from the queue,
// excluding any issues in the skip set. This allows the daemon to skip
// issues that failed to spawn (e.g., due to failure report gate) and
// continue processing other issues in the queue.
//
// Returns nil if no spawnable issues are available after excluding skipped ones.
// Issues are sorted by priority (0 = highest priority).
// If a label filter is configured, only issues with that label are considered.
func (d *Daemon) NextIssueExcluding(skip map[string]bool) (*Issue, error) {
	issues, err := d.listIssuesFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	if d.Config.Verbose {
		fmt.Printf("  DEBUG: Found %d open issues\n", len(issues))
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Priority < issues[j].Priority
	})

	for _, issue := range issues {
		// Skip issues in the skip set (failed to spawn this cycle)
		if skip != nil && skip[issue.ID] {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (failed to spawn this cycle)\n", issue.ID)
			}
			continue
		}
		// Skip non-spawnable types
		if !IsSpawnableType(issue.IssueType) {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (type %s not spawnable)\n", issue.ID, issue.IssueType)
			}
			continue
		}
		// Skip blocked issues
		if issue.Status == "blocked" {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (blocked)\n", issue.ID)
			}
			continue
		}
		// Skip in_progress issues (already being worked on)
		if issue.Status == "in_progress" {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (already in_progress)\n", issue.ID)
			}
			continue
		}
		// Skip issues without required label (if filter is set)
		if d.Config.Label != "" && !issue.HasLabel(d.Config.Label) {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (missing label %s, has %v)\n", issue.ID, d.Config.Label, issue.Labels)
			}
			continue
		}
		// Skip issues with blocking dependencies (open/in_progress dependencies)
		blockers, err := beads.CheckBlockingDependencies(issue.ID)
		if err != nil {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Warning: could not check dependencies for %s: %v\n", issue.ID, err)
			}
			// Continue checking - don't skip issue just because we can't check dependencies
		} else if len(blockers) > 0 {
			if d.Config.Verbose {
				var blockerIDs []string
				for _, b := range blockers {
					blockerIDs = append(blockerIDs, fmt.Sprintf("%s (%s)", b.ID, b.Status))
				}
				fmt.Printf("  DEBUG: Skipping %s (blocked by dependencies: %s)\n", issue.ID, strings.Join(blockerIDs, ", "))
			}
			continue
		}
		if d.Config.Verbose {
			fmt.Printf("  DEBUG: Selected %s (type=%s, labels=%v)\n", issue.ID, issue.IssueType, issue.Labels)
		}
		return &issue, nil
	}

	return nil, nil
}

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

// ReconcileWithOpenCode synchronizes the worker pool with actual OpenCode sessions.
// This prevents the pool from becoming stuck at capacity when agents complete
// without the daemon knowing (e.g., overnight runs, crashes, manual kills).
//
// Should be called at the start of each poll cycle.
// Returns the number of slots freed due to reconciliation, or 0 if no pool.
func (d *Daemon) ReconcileWithOpenCode() int {
	if d.Pool == nil {
		return 0
	}

	// Get actual count from OpenCode API
	actualCount := DefaultActiveCount()

	// Reconcile pool with actual count
	return d.Pool.Reconcile(actualCount)
}

// Preview shows what would be processed next without actually processing.
func (d *Daemon) Preview() (*PreviewResult, error) {
	result := &PreviewResult{}

	// Check rate limit status
	if d.RateLimiter != nil {
		canSpawn, count, msg := d.RateLimiter.CanSpawn()
		result.RateLimited = !canSpawn
		if d.RateLimiter.MaxPerHour > 0 {
			result.RateStatus = fmt.Sprintf("%d/%d spawns in last hour", count, d.RateLimiter.MaxPerHour)
		}
		if !canSpawn {
			result.Message = msg
			return result, nil
		}
	}

	issue, err := d.NextIssue()
	if err != nil {
		return nil, err
	}

	if issue == nil {
		result.Message = "No spawnable issues in queue"
		return result, nil
	}

	skill, err := InferSkillFromIssue(issue)
	if err != nil {
		return nil, fmt.Errorf("failed to infer skill: %w", err)
	}

	result.Issue = issue
	result.Skill = skill

	// Check for hotspot warnings if checker is configured
	if d.HotspotChecker != nil {
		result.HotspotWarnings = CheckHotspotsForIssue(issue, d.HotspotChecker)
	}

	return result, nil
}

// FormatPreview formats an issue for preview display.
func FormatPreview(issue *Issue) string {
	return fmt.Sprintf(`Issue:    %s
Title:    %s
Type:     %s
Priority: P%d
Status:   %s
Description: %s`,
		issue.ID,
		issue.Title,
		issue.IssueType,
		issue.Priority,
		issue.Status,
		truncate(issue.Description, 100),
	)
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
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

	issue, err := d.NextIssueExcluding(skip)
	if err != nil {
		return nil, err
	}

	if issue == nil {
		return &OnceResult{
			Processed: false,
			Message:   "No spawnable issues in queue",
		}, nil
	}

	skill, err := InferSkill(issue.IssueType)
	if err != nil {
		return nil, fmt.Errorf("failed to infer skill: %w", err)
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

	// Spawn the work
	if err := d.spawnFunc(issue.ID); err != nil {
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

	skill, err := InferSkill(issue.IssueType)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to infer skill: %w", err)
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

	// Spawn the work
	if err := d.spawnFunc(issue.ID); err != nil {
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
