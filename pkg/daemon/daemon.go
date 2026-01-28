// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// EventLogger is an interface for logging deduplication events.
// Implemented by events.Logger.
type EventLogger interface {
	LogDedupBlocked(data interface{}) error
}

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

	// CrossProject enables polling across all kb-registered projects.
	// When enabled, the daemon iterates over all projects from `kb projects list`
	// and processes issues from each project. Global capacity pool is shared.
	CrossProject bool

	// Backend specifies the spawn backend: "opencode", "docker", or "claude".
	// This affects how active agents are counted for concurrency control.
	// "docker" counts running Docker containers, others query OpenCode API.
	Backend string

	// ReflectEnabled controls whether periodic reflection analysis is enabled.
	// When enabled, the daemon will run kb reflect periodically.
	ReflectEnabled bool

	// ReflectInterval is how often to run kb reflect (0 = disabled).
	// Default is 1 hour.
	ReflectInterval time.Duration

	// ReflectCreateIssues controls whether reflection creates beads issues
	// for synthesis opportunities (topics with 10+ investigations).
	ReflectCreateIssues bool

	// CleanupEnabled controls whether periodic session cleanup is enabled.
	// When enabled, the daemon will run session cleanup periodically.
	CleanupEnabled bool

	// CleanupInterval is how often to run session cleanup (0 = disabled).
	// Default is 6 hours.
	CleanupInterval time.Duration

	// CleanupAgeDays is the age threshold in days for session cleanup.
	// Sessions older than this will be deleted. Default is 7 days.
	CleanupAgeDays int

	// CleanupPreserveOrchestrator if true, skips orchestrator sessions.
	// Default is true to avoid disrupting orchestrator sessions.
	CleanupPreserveOrchestrator bool

	// CleanupServerURL is the OpenCode server URL for cleanup operations.
	// Defaults to http://127.0.0.1:4096.
	CleanupServerURL string

	// RecoveryEnabled controls whether stuck agent recovery is enabled.
	// When enabled, the daemon will detect idle agents and attempt auto-resume.
	RecoveryEnabled bool

	// RecoveryInterval is how often to check for stuck agents (0 = disabled).
	// Default is 5 minutes.
	RecoveryInterval time.Duration

	// RecoveryIdleThreshold is how long an agent must be idle before recovery.
	// Default is 10 minutes.
	RecoveryIdleThreshold time.Duration

	// RecoveryRateLimit is minimum time between resume attempts per agent.
	// Default is 1 hour to prevent infinite loops.
	RecoveryRateLimit time.Duration

	// ServerRecoveryEnabled controls whether server restart recovery is enabled.
	// When enabled, the daemon will detect orphaned sessions after server restart
	// and resume them with recovery-specific context.
	ServerRecoveryEnabled bool

	// ServerRecoveryStabilizationDelay is how long to wait after daemon start
	// before running server recovery. This allows the OpenCode server to fully
	// initialize before we query it.
	// Default is 30 seconds.
	ServerRecoveryStabilizationDelay time.Duration

	// ServerRecoveryResumeDelay is the delay between resuming each orphaned session.
	// This prevents overwhelming the server with simultaneous resumes.
	// Default is 10 seconds.
	ServerRecoveryResumeDelay time.Duration

	// ServerRecoveryRateLimit is minimum time between recovery attempts per agent.
	// Default is 1 hour to prevent infinite loops.
	ServerRecoveryRateLimit time.Duration

	// MaxResumeAttempts is the maximum number of resume attempts before escalating
	// to 'Needs Human Decision'. After this many failed attempts, the agent is
	// marked as needing manual intervention.
	// Default is 3 attempts.
	MaxResumeAttempts int

	// AutoAbandonAfterHours is how long an agent can be dead with no progress
	// before being automatically abandoned. Set to 0 to disable auto-abandon.
	// Default is 24 hours.
	AutoAbandonAfterHours int
}

// DefaultConfig returns sensible defaults for daemon configuration.
func DefaultConfig() Config {
	return Config{
		PollInterval:                     time.Minute,
		MaxAgents:                        3,
		MaxSpawnsPerHour:                 20, // Prevents runaway spawning
		Label:                            "triage:ready",
		SpawnDelay:                       10 * time.Second,
		DryRun:                           false,
		Verbose:                          false,
		ReflectEnabled:                   true,
		ReflectInterval:                  time.Hour, // Hourly by default
		ReflectCreateIssues:              true,
		CleanupEnabled:                   true,
		CleanupInterval:                  6 * time.Hour, // Every 6 hours by default
		CleanupAgeDays:                   7,             // 7 days threshold
		CleanupPreserveOrchestrator:      true,          // Preserve orchestrator sessions
		CleanupServerURL:                 "http://127.0.0.1:4096",
		RecoveryEnabled:                  true,
		RecoveryInterval:                 5 * time.Minute,  // Check every 5 minutes
		RecoveryIdleThreshold:            10 * time.Minute, // Idle >10min triggers recovery
		RecoveryRateLimit:                time.Hour,        // 1 resume per agent per hour
		ServerRecoveryEnabled:            true,
		ServerRecoveryStabilizationDelay: 30 * time.Second, // Wait 30s for server stability
		ServerRecoveryResumeDelay:        10 * time.Second, // 10s between each resume
		ServerRecoveryRateLimit:          time.Hour,        // 1 recovery per agent per hour
		MaxResumeAttempts:                3,                // Escalate after 3 failed attempts
		AutoAbandonAfterHours:            24,               // Auto-abandon after 24h dead
	}
}

// RejectedIssue captures why an issue was rejected for spawning.
type RejectedIssue struct {
	Issue  Issue  // The rejected issue
	Reason string // Human-readable rejection reason
}

// PreviewResult contains the result of a preview operation.
type PreviewResult struct {
	Issue           *Issue
	Skill           string
	Message         string
	RateLimited     bool             // True if rate limit would prevent spawning
	RateStatus      string           // Rate limit status message (e.g., "5/20 spawns in last hour")
	HotspotWarnings []HotspotWarning // Warnings about hotspot areas this issue may touch
	RejectedIssues  []RejectedIssue  // Issues that were rejected with reasons
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

	// SpawnedIssues tracks issue IDs that have been spawned but may not yet
	// have their beads status updated to in_progress. This prevents the race
	// condition where the daemon spawns duplicate agents for the same issue
	// because the status update hasn't propagated yet.
	SpawnedIssues *SpawnedIssueTracker

	// EventLogger is used to log deduplication events for telemetry.
	// If nil, events are not logged.
	EventLogger EventLogger

	// lastReflect tracks when reflection was last run for periodic reflection.
	lastReflect time.Time

	// lastCleanup tracks when session cleanup was last run for periodic cleanup.
	lastCleanup time.Time

	// lastRecovery tracks when recovery was last run for periodic recovery.
	lastRecovery time.Time

	// resumeAttempts tracks when we last attempted to resume each agent (by beads ID).
	// Prevents infinite resume loops by rate-limiting to 1 attempt per hour per agent.
	resumeAttempts map[string]time.Time

	// resumeAttemptCounts tracks how many times we've attempted to resume each agent.
	// Used for escalation after N failed attempts.
	resumeAttemptCounts map[string]int

	// serverRecoveryState tracks state for server restart recovery.
	// Used to determine when server recovery should run (once per daemon start).
	serverRecoveryState *ServerRecoveryState

	// listIssuesFunc is used for testing - allows mocking bd list
	listIssuesFunc func() ([]Issue, error)
	// spawnFunc is used for testing - allows mocking orch work
	spawnFunc func(beadsID string) error
	// activeCountFunc is used for testing - allows mocking active agent count
	// Deprecated: Use Pool for concurrency control instead.
	activeCountFunc func() int
	// listCompletedAgentsFunc is used for testing - allows mocking completed agents list
	listCompletedAgentsFunc func(CompletionConfig) ([]CompletedAgent, error)
	// reflectFunc is used for testing - allows mocking kb reflect
	reflectFunc func(createIssues bool) (*ReflectResult, error)
	// listEpicChildrenFunc is used for testing - allows mocking ListEpicChildren
	listEpicChildrenFunc func(epicID string) ([]Issue, error)
	// listProjectsFunc is used for testing - allows mocking kb projects list
	listProjectsFunc func() ([]Project, error)
	// listIssuesForProjectFunc is used for testing - allows mocking ListReadyIssuesForProject
	listIssuesForProjectFunc func(projectPath string) ([]Issue, error)
	// spawnForProjectFunc is used for testing - allows mocking SpawnWorkForProject
	spawnForProjectFunc func(beadsID, projectPath string) error
}

// New creates a new Daemon instance with default configuration.
func New() *Daemon {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new Daemon instance with the given configuration.
func NewWithConfig(config Config) *Daemon {
	// Select active count function based on backend
	activeCount := DefaultActiveCount
	if config.Backend == "docker" {
		activeCount = DockerActiveCount
	}

	d := &Daemon{
		Config:                   config,
		SpawnedIssues:            NewSpawnedIssueTracker(),
		EventLogger:              nil, // Set via SetEventLogger() to avoid circular deps
		resumeAttempts:           make(map[string]time.Time),
		resumeAttemptCounts:      make(map[string]int),
		serverRecoveryState:      NewServerRecoveryState(),
		listIssuesFunc:           ListReadyIssues,
		spawnFunc:                SpawnWork,
		activeCountFunc:          activeCount,
		reflectFunc:              DefaultRunReflection,
		listEpicChildrenFunc:     ListEpicChildren,
		listProjectsFunc:         ListProjects,
		listIssuesForProjectFunc: ListReadyIssuesForProject,
		spawnForProjectFunc:      SpawnWorkForProject,
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

// SetEventLogger sets the event logger for telemetry.
// This is separate from the constructor to avoid circular dependencies
// with the events package.
func (d *Daemon) SetEventLogger(logger EventLogger) {
	d.EventLogger = logger
}

// NewWithPool creates a new Daemon instance with an explicit worker pool.
// This is useful for sharing a pool across daemon instances or for testing.
func NewWithPool(config Config, pool *WorkerPool) *Daemon {
	// Select active count function based on backend
	activeCount := DefaultActiveCount
	if config.Backend == "docker" {
		activeCount = DockerActiveCount
	}

	d := &Daemon{
		Config:                   config,
		Pool:                     pool,
		SpawnedIssues:            NewSpawnedIssueTracker(),
		EventLogger:              nil, // Set via SetEventLogger() to avoid circular deps
		resumeAttempts:           make(map[string]time.Time),
		serverRecoveryState:      NewServerRecoveryState(),
		listIssuesFunc:           ListReadyIssues,
		spawnFunc:                SpawnWork,
		activeCountFunc:          activeCount,
		reflectFunc:              DefaultRunReflection,
		listProjectsFunc:         ListProjects,
		listIssuesForProjectFunc: ListReadyIssuesForProject,
		spawnForProjectFunc:      SpawnWorkForProject,
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
//
// Epic child expansion: When an epic has the required label (e.g., triage:ready),
// its children are automatically included in the spawn queue even if they don't
// have the label themselves. This implements the user mental model that labeling
// an epic means "process this entire epic".
func (d *Daemon) NextIssueExcluding(skip map[string]bool) (*Issue, error) {
	issues, err := d.listIssuesFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	if d.Config.Verbose {
		fmt.Printf("  DEBUG: Found %d open issues\n", len(issues))
	}

	// Expand triage:ready epics by including their children.
	// This allows "label the epic" to mean "process the entire epic".
	issues, epicChildIDs := d.expandTriageReadyEpics(issues)

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
		// Skip issues that have been recently spawned but status not yet updated.
		// This prevents the race condition where the daemon spawns duplicate agents
		// because beads status update hasn't propagated yet.
		if d.SpawnedIssues != nil && d.SpawnedIssues.IsSpawned(issue.ID) {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (recently spawned, awaiting status update)\n", issue.ID)
			}
			// Emit telemetry event when SpawnedIssueTracker blocks spawn
			if d.EventLogger != nil {
				_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
					"beads_id":    issue.ID,
					"dedup_layer": "spawned_tracker",
					"reason":      "Issue recently spawned, awaiting status update (6h TTL)",
				})
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
		// BUT: Children of triage:ready epics are exempt from this check
		// (they inherit triage-ready status from their parent)
		if d.Config.Label != "" && !issue.HasLabel(d.Config.Label) {
			// Check if this issue is a child of a triage:ready epic
			if _, isEpicChild := epicChildIDs[issue.ID]; !isEpicChild {
				if d.Config.Verbose {
					fmt.Printf("  DEBUG: Skipping %s (missing label %s, has %v)\n", issue.ID, d.Config.Label, issue.Labels)
				}
				continue
			}
			// Epic child - proceed even without label
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Including %s (epic child, inherits triage status from parent)\n", issue.ID)
			}
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

// expandTriageReadyEpics finds epics with the required label and includes their children.
// Returns the expanded issue list and a map of issue IDs that are epic children
// (for label exemption in NextIssueExcluding).
func (d *Daemon) expandTriageReadyEpics(issues []Issue) ([]Issue, map[string]bool) {
	epicChildIDs := make(map[string]bool)

	// If no label filter is set, no expansion needed
	if d.Config.Label == "" {
		return issues, epicChildIDs
	}

	// Find epics with the required label
	var epicsToExpand []string
	existingIDs := make(map[string]bool)
	for _, issue := range issues {
		existingIDs[issue.ID] = true
		if issue.IssueType == "epic" && issue.HasLabel(d.Config.Label) {
			epicsToExpand = append(epicsToExpand, issue.ID)
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Found triage:ready epic %s, will include children\n", issue.ID)
			}
		}
	}

	// No epics to expand
	if len(epicsToExpand) == 0 {
		return issues, epicChildIDs
	}

	// Expand each epic by fetching its children
	listChildren := d.listEpicChildrenFunc
	if listChildren == nil {
		listChildren = ListEpicChildren
	}
	for _, epicID := range epicsToExpand {
		children, err := listChildren(epicID)
		if err != nil {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Warning: could not list children of epic %s: %v\n", epicID, err)
			}
			continue
		}

		for _, child := range children {
			// Skip closed children - they shouldn't be spawned
			if child.Status == "closed" {
				if d.Config.Verbose {
					fmt.Printf("  DEBUG: Skipping closed epic child %s (from parent %s)\n", child.ID, epicID)
				}
				continue
			}
			// Only add if not already in the list
			if !existingIDs[child.ID] {
				issues = append(issues, child)
				existingIDs[child.ID] = true
				epicChildIDs[child.ID] = true
				if d.Config.Verbose {
					fmt.Printf("  DEBUG: Added epic child %s (from parent %s)\n", child.ID, epicID)
				}
			} else {
				// Already in list, but mark as epic child for label exemption
				epicChildIDs[child.ID] = true
			}
		}
	}

	return issues, epicChildIDs
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

// Preview shows what would be processed next without actually processing.
// It also collects all rejected issues with their rejection reasons.
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
			// Still collect rejected issues even if rate limited
		}
	}

	// Get all issues and categorize them
	issues, err := d.listIssuesFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// Expand triage:ready epics by including their children
	issues, epicChildIDs := d.expandTriageReadyEpics(issues)

	// Sort by priority (lower number = higher priority)
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Priority < issues[j].Priority
	})

	var spawnable *Issue
	for _, issue := range issues {
		// Check each rejection reason in order and collect all rejected issues
		reason := d.checkRejectionReasonWithEpicChildren(issue, epicChildIDs)
		if reason != "" {
			result.RejectedIssues = append(result.RejectedIssues, RejectedIssue{
				Issue:  issue,
				Reason: reason,
			})
			continue
		}

		// Found a spawnable issue - take the first one (highest priority)
		if spawnable == nil {
			issueCopy := issue
			spawnable = &issueCopy
		}
	}

	// If rate limited, we still collected rejected issues but can't spawn
	if result.RateLimited {
		return result, nil
	}

	if spawnable == nil {
		result.Message = "No spawnable issues in queue"
		return result, nil
	}

	skill, err := InferSkillFromIssue(spawnable)
	if err != nil {
		return nil, fmt.Errorf("failed to infer skill: %w", err)
	}

	result.Issue = spawnable
	result.Skill = skill

	// Check for hotspot warnings if checker is configured
	if d.HotspotChecker != nil {
		result.HotspotWarnings = CheckHotspotsForIssue(spawnable, d.HotspotChecker)
	}

	return result, nil
}

// checkRejectionReason checks if an issue should be rejected and returns the reason.
// Returns empty string if the issue is spawnable.
// This is the legacy version that doesn't consider epic children.
func (d *Daemon) checkRejectionReason(issue Issue) string {
	return d.checkRejectionReasonWithEpicChildren(issue, nil)
}

// checkRejectionReasonWithEpicChildren checks if an issue should be rejected and returns the reason.
// The epicChildIDs map contains IDs of issues that are children of triage:ready epics.
// These children are exempt from the label requirement check.
// Returns empty string if the issue is spawnable.
func (d *Daemon) checkRejectionReasonWithEpicChildren(issue Issue, epicChildIDs map[string]bool) string {
	// Check for empty/missing type first (the main problem case from the bug report)
	if issue.IssueType == "" {
		return "missing type (required for skill inference)"
	}

	// Check for non-spawnable type
	// Note: Epics with triage:ready are not spawnable themselves, but their children are.
	// The message is informative to explain why epics are rejected.
	if !IsSpawnableType(issue.IssueType) {
		if issue.IssueType == "epic" && issue.HasLabel(d.Config.Label) {
			return fmt.Sprintf("type 'epic' not spawnable (children will be processed instead)")
		}
		return fmt.Sprintf("type '%s' not spawnable (must be bug/feature/task/investigation)", issue.IssueType)
	}

	// Check for blocked status
	if issue.Status == "blocked" {
		return "status is blocked"
	}

	// Check for in_progress status
	if issue.Status == "in_progress" {
		return "status is in_progress (already being worked on)"
	}

	// Check for missing required label
	// Epic children are exempt from this check - they inherit triage status from parent
	if d.Config.Label != "" && !issue.HasLabel(d.Config.Label) {
		if epicChildIDs == nil || !epicChildIDs[issue.ID] {
			return fmt.Sprintf("missing label '%s'", d.Config.Label)
		}
		// Epic child - exempt from label requirement
	}

	// Check for blocking dependencies
	blockers, err := beads.CheckBlockingDependencies(issue.ID)
	if err == nil && len(blockers) > 0 {
		var blockerIDs []string
		for _, b := range blockers {
			blockerIDs = append(blockerIDs, fmt.Sprintf("%s (%s)", b.ID, b.Status))
		}
		return fmt.Sprintf("blocked by dependencies: %s", strings.Join(blockerIDs, ", "))
	}

	return "" // Spawnable
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

// FormatRejectedIssues formats rejected issues for display.
func FormatRejectedIssues(rejected []RejectedIssue) string {
	if len(rejected) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\nRejected issues:\n")
	for _, r := range rejected {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", r.Issue.ID, r.Reason))
	}
	return sb.String()
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

		// Session-level dedup: Check if there's an existing OpenCode session for this issue.
		// This prevents duplicate spawns when:
		// 1. SpawnedIssueTracker TTL expires (5min/6h) but agent is still running
		// 2. Status update to "in_progress" failed silently
		// 3. Multiple daemon instances try to spawn the same issue
		if HasExistingSessionForBeadsID(issue.ID) {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (existing OpenCode session found)\n", issue.ID)
			}
			// Emit telemetry event when session dedup blocks spawn
			if d.EventLogger != nil {
				_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
					"beads_id":    issue.ID,
					"dedup_layer": "session_dedup",
					"reason":      "Existing OpenCode session found via API check",
				})
			}
			extendedSkip[issue.ID] = true
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s: existing session", issue.ID))
			continue
		}

		// Pre-spawn completion check: Skip issues where an agent has already reported
		// Phase: Complete but the orchestrator hasn't closed the issue yet.
		// This prevents respawning completed work when:
		// 1. SpawnedIssueTracker TTL expires
		// 2. OpenCode session was deleted (manual cleanup, server restart)
		// 3. Beads status is still "open" because orch complete hasn't run
		if hasComplete, _ := HasPhaseComplete(issue.ID); hasComplete {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (Phase: Complete already reported)\n", issue.ID)
			}
			// Emit telemetry event when Phase:Complete blocks spawn
			if d.EventLogger != nil {
				_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
					"beads_id":    issue.ID,
					"dedup_layer": "phase_complete",
					"reason":      "Phase: Complete comment found in beads issue",
				})
			}
			extendedSkip[issue.ID] = true
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s: Phase: Complete", issue.ID))
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

	// Mark issue as spawned BEFORE calling spawnFunc to prevent race condition.
	// This prevents duplicate spawns if daemon polls again before beads status updates.
	if d.SpawnedIssues != nil {
		d.SpawnedIssues.MarkSpawned(issue.ID)
	}

	// Spawn the work
	if err := d.spawnFunc(issue.ID); err != nil {
		// Unmark on spawn failure so issue can be retried
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

	// Session-level dedup: Check if there's an existing OpenCode session for this issue.
	// This prevents duplicate spawns when:
	// 1. SpawnedIssueTracker TTL expires but agent is still running
	// 2. Status update to "in_progress" failed silently
	// 3. Multiple daemon instances try to spawn the same issue
	if HasExistingSessionForBeadsID(issue.ID) {
		if d.Config.Verbose {
			fmt.Printf("  DEBUG: Skipping %s (existing OpenCode session found)\n", issue.ID)
		}
		// Emit telemetry event when session dedup blocks spawn
		if d.EventLogger != nil {
			_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
				"beads_id":    issue.ID,
				"dedup_layer": "session_dedup",
				"reason":      "Existing OpenCode session found via API check",
			})
		}
		return &OnceResult{
			Processed: false,
			Issue:     issue,
			Skill:     skill,
			Message:   fmt.Sprintf("Existing session found for %s - skipping to prevent duplicate", issue.ID),
		}, nil, nil
	}

	// Pre-spawn completion check: Skip issues where an agent has already reported
	// Phase: Complete but the orchestrator hasn't closed the issue yet.
	if hasComplete, _ := HasPhaseComplete(issue.ID); hasComplete {
		if d.Config.Verbose {
			fmt.Printf("  DEBUG: Skipping %s (Phase: Complete already reported)\n", issue.ID)
		}
		// Emit telemetry event when Phase:Complete blocks spawn
		if d.EventLogger != nil {
			_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
				"beads_id":    issue.ID,
				"dedup_layer": "phase_complete",
				"reason":      "Phase: Complete comment found in beads issue",
			})
		}
		return &OnceResult{
			Processed: false,
			Issue:     issue,
			Skill:     skill,
			Message:   fmt.Sprintf("Skipping %s: already completed (Phase: Complete found in comments)", issue.ID),
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

	// Mark issue as spawned BEFORE calling spawnFunc to prevent race condition.
	// This prevents duplicate spawns if daemon polls again before beads status updates.
	if d.SpawnedIssues != nil {
		d.SpawnedIssues.MarkSpawned(issue.ID)
	}

	// Spawn the work
	if err := d.spawnFunc(issue.ID); err != nil {
		// Unmark on spawn failure so issue can be retried
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

// ShouldRunReflection returns true if periodic reflection should run.
// This checks if reflection is enabled and enough time has elapsed since the last run.
func (d *Daemon) ShouldRunReflection() bool {
	if !d.Config.ReflectEnabled || d.Config.ReflectInterval <= 0 {
		return false
	}
	// Run immediately if we've never run before
	if d.lastReflect.IsZero() {
		return true
	}
	return time.Since(d.lastReflect) >= d.Config.ReflectInterval
}

// RunPeriodicReflection runs the periodic reflection analysis if due.
// Returns the result if reflection was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicReflection() *ReflectResult {
	if !d.ShouldRunReflection() {
		return nil
	}

	result, err := d.reflectFunc(d.Config.ReflectCreateIssues)
	if err != nil {
		return &ReflectResult{
			Error:   err,
			Message: fmt.Sprintf("Reflection failed: %v", err),
		}
	}

	// Update last reflect time on success
	d.lastReflect = time.Now()

	return result
}

// LastReflectTime returns when reflection was last run.
// Returns zero time if reflection has never run.
func (d *Daemon) LastReflectTime() time.Time {
	return d.lastReflect
}

// NextReflectTime returns when the next reflection is scheduled.
// Returns zero time if reflection is disabled.
func (d *Daemon) NextReflectTime() time.Time {
	if !d.Config.ReflectEnabled || d.Config.ReflectInterval <= 0 {
		return time.Time{}
	}
	if d.lastReflect.IsZero() {
		return time.Now() // Due immediately
	}
	return d.lastReflect.Add(d.Config.ReflectInterval)
}

// ShouldRunCleanup returns true if periodic session cleanup should run.
// This checks if cleanup is enabled and enough time has elapsed since the last run.
func (d *Daemon) ShouldRunCleanup() bool {
	if !d.Config.CleanupEnabled || d.Config.CleanupInterval <= 0 {
		return false
	}
	// Run immediately if we've never run before
	if d.lastCleanup.IsZero() {
		return true
	}
	return time.Since(d.lastCleanup) >= d.Config.CleanupInterval
}

// CleanupResult contains the result of a cleanup operation.
type CleanupResult struct {
	Deleted int
	Error   error
	Message string
}

// RunPeriodicCleanup runs the periodic session cleanup if due.
// Returns the result if cleanup was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicCleanup() *CleanupResult {
	if !d.ShouldRunCleanup() {
		return nil
	}

	// Import cleanup package functions via helper
	deleted, err := runSessionCleanup(d.Config.CleanupServerURL, d.Config.CleanupAgeDays, d.Config.CleanupPreserveOrchestrator)
	if err != nil {
		return &CleanupResult{
			Deleted: 0,
			Error:   err,
			Message: fmt.Sprintf("Session cleanup failed: %v", err),
		}
	}

	// Update last cleanup time on success
	d.lastCleanup = time.Now()

	return &CleanupResult{
		Deleted: deleted,
		Error:   nil,
		Message: fmt.Sprintf("Deleted %d stale sessions (age >%d days)", deleted, d.Config.CleanupAgeDays),
	}
}

// LastCleanupTime returns when cleanup was last run.
// Returns zero time if cleanup has never run.
func (d *Daemon) LastCleanupTime() time.Time {
	return d.lastCleanup
}

// NextCleanupTime returns when the next cleanup is scheduled.
// Returns zero time if cleanup is disabled.
func (d *Daemon) NextCleanupTime() time.Time {
	if !d.Config.CleanupEnabled || d.Config.CleanupInterval <= 0 {
		return time.Time{}
	}
	if d.lastCleanup.IsZero() {
		return time.Now() // Due immediately
	}
	return d.lastCleanup.Add(d.Config.CleanupInterval)
}

// ShouldRunRecovery returns true if periodic recovery should run.
// This checks if recovery is enabled and enough time has elapsed since the last run.
func (d *Daemon) ShouldRunRecovery() bool {
	if !d.Config.RecoveryEnabled || d.Config.RecoveryInterval <= 0 {
		return false
	}
	// Run immediately if we've never run before
	if d.lastRecovery.IsZero() {
		return true
	}
	return time.Since(d.lastRecovery) >= d.Config.RecoveryInterval
}

// RecoveryResult contains the result of a recovery operation.
type RecoveryResult struct {
	ResumedCount   int
	SkippedCount   int
	EscalatedCount int // Agents escalated to needs human decision
	AbandonedCount int // Agents auto-abandoned after timeout
	Error          error
	Message        string
}

// RunPeriodicRecovery runs the periodic stuck agent recovery if due.
// Returns the result if recovery was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicRecovery() *RecoveryResult {
	if !d.ShouldRunRecovery() {
		return nil
	}

	// Get list of active agents via registry
	agents, err := GetActiveAgents()
	if err != nil {
		return &RecoveryResult{
			ResumedCount:   0,
			SkippedCount:   0,
			EscalatedCount: 0,
			AbandonedCount: 0,
			Error:          err,
			Message:        fmt.Sprintf("Recovery failed to list agents: %v", err),
		}
	}

	resumed := 0
	skipped := 0
	escalated := 0
	abandoned := 0
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

		// Auto-abandon: If agent has been dead for X hours with no progress, auto-abandon
		if d.Config.AutoAbandonAfterHours > 0 {
			abandonThreshold := time.Duration(d.Config.AutoAbandonAfterHours) * time.Hour
			if idleTime >= abandonThreshold {
				if d.Config.Verbose {
					fmt.Printf("  Auto-abandoning %s (dead for %v, threshold: %v)\n",
						agent.BeadsID, idleTime.Round(time.Minute), abandonThreshold)
				}
				// Close the issue with auto-abandon reason
				reason := fmt.Sprintf("Auto-abandoned: No progress for %v (threshold: %v)",
					idleTime.Round(time.Minute), abandonThreshold)
				if err := verify.CloseIssue(agent.BeadsID, reason); err != nil {
					if d.Config.Verbose {
						fmt.Printf("  Failed to auto-abandon %s: %v\n", agent.BeadsID, err)
					}
				} else {
					abandoned++
					// Clear tracking for this agent
					delete(d.resumeAttempts, agent.BeadsID)
					delete(d.resumeAttemptCounts, agent.BeadsID)
				}
				continue
			}
		}

		// Escalation: If agent has failed resume N times, escalate to needs human decision
		attemptCount := d.resumeAttemptCounts[agent.BeadsID]
		if d.Config.MaxResumeAttempts > 0 && attemptCount >= d.Config.MaxResumeAttempts {
			if d.Config.Verbose {
				fmt.Printf("  Escalating %s to 'Needs Human Decision' (attempts: %d, threshold: %d)\n",
					agent.BeadsID, attemptCount, d.Config.MaxResumeAttempts)
			}
			// Add needs:human label for escalation
			if err := addNeedsHumanLabel(agent.BeadsID); err != nil {
				if d.Config.Verbose {
					fmt.Printf("  Failed to escalate %s: %v\n", agent.BeadsID, err)
				}
			} else {
				escalated++
				// Reset attempt count after escalation
				delete(d.resumeAttemptCounts, agent.BeadsID)
			}
			// Skip resume attempt after escalation
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
			fmt.Printf("  Attempting recovery for %s (idle for %v, attempt %d)\n",
				agent.BeadsID, idleTime.Round(time.Minute), attemptCount+1)
		}

		// Increment attempt count BEFORE attempting resume
		d.resumeAttemptCounts[agent.BeadsID] = attemptCount + 1

		if err := ResumeAgentByBeadsID(agent.BeadsID); err != nil {
			if d.Config.Verbose {
				fmt.Printf("  Failed to resume %s: %v\n", agent.BeadsID, err)
			}
			// Record failed attempt time (for rate limiting)
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
	d.lastRecovery = time.Now()

	message := fmt.Sprintf("Recovery attempted: %d resumed, %d skipped", resumed, skipped)
	if escalated > 0 {
		message += fmt.Sprintf(", %d escalated", escalated)
	}
	if abandoned > 0 {
		message += fmt.Sprintf(", %d abandoned", abandoned)
	}

	return &RecoveryResult{
		ResumedCount:   resumed,
		SkippedCount:   skipped,
		EscalatedCount: escalated,
		AbandonedCount: abandoned,
		Error:          nil,
		Message:        message,
	}
}

// LastRecoveryTime returns when recovery was last run.
// Returns zero time if recovery has never run.
func (d *Daemon) LastRecoveryTime() time.Time {
	return d.lastRecovery
}

// NextRecoveryTime returns when the next recovery is scheduled.
// Returns zero time if recovery is disabled.
func (d *Daemon) NextRecoveryTime() time.Time {
	if !d.Config.RecoveryEnabled || d.Config.RecoveryInterval <= 0 {
		return time.Time{}
	}
	if d.lastRecovery.IsZero() {
		return time.Now() // Due immediately
	}
	return d.lastRecovery.Add(d.Config.RecoveryInterval)
}

// addNeedsHumanLabel adds the needs:human label to a beads issue.
// This label indicates that the agent requires human intervention.
// Uses the beads RPC client with auto-reconnect when available, falling back to CLI.
func addNeedsHumanLabel(beadsID string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if beads.DefaultDir != "" {
			opts = append(opts, beads.WithCwd(beads.DefaultDir))
		}
		client := beads.NewClient(socketPath, opts...)
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			err := client.AddLabel(beadsID, "needs:human")
			if err == nil {
				return nil
			}
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackAddLabel(beadsID, "needs:human")
}

// ShouldRunServerRecovery returns true if server restart recovery should run.
// This runs once after daemon startup, after the stabilization delay has passed.
func (d *Daemon) ShouldRunServerRecovery() bool {
	if d.Config.Verbose {
		fmt.Printf("[DEBUG] ShouldRunServerRecovery: ServerRecoveryEnabled=%v\n", d.Config.ServerRecoveryEnabled)
	}
	if !d.Config.ServerRecoveryEnabled {
		if d.Config.Verbose {
			fmt.Printf("[DEBUG] ShouldRunServerRecovery: returning false - ServerRecoveryEnabled is false\n")
		}
		return false
	}
	if d.serverRecoveryState == nil {
		if d.Config.Verbose {
			fmt.Printf("[DEBUG] ShouldRunServerRecovery: returning false - serverRecoveryState is nil\n")
		}
		return false
	}
	result := d.serverRecoveryState.ShouldRunServerRecovery(d.Config.ServerRecoveryStabilizationDelay)
	if d.Config.Verbose {
		fmt.Printf("[DEBUG] ShouldRunServerRecovery: stabilizationDelay=%v, result=%v\n",
			d.Config.ServerRecoveryStabilizationDelay, result)
	}
	return result
}

// RunServerRecovery runs server restart recovery if due.
// This detects orphaned sessions (sessions that exist on disk but aren't in OpenCode's
// in-memory state) and resumes them with recovery-specific context.
//
// Unlike RunPeriodicRecovery which handles individual stuck agents, this handles
// the bulk recovery scenario after a server restart where ALL in-memory sessions
// are lost simultaneously.
//
// Returns the result if recovery was run, or nil if it wasn't due.
func (d *Daemon) RunServerRecovery() *ServerRecoveryResult {
	if !d.ShouldRunServerRecovery() {
		return nil
	}

	// Mark that we've run recovery (regardless of outcome)
	d.serverRecoveryState.MarkRecoveryRun()

	serverURL := d.Config.CleanupServerURL
	if serverURL == "" {
		serverURL = "http://127.0.0.1:4096"
	}

	// Find orphaned sessions
	orphaned, err := FindOrphanedSessions(serverURL)
	if err != nil {
		return &ServerRecoveryResult{
			Error:   err,
			Message: fmt.Sprintf("Server recovery failed to find orphaned sessions: %v", err),
		}
	}

	if len(orphaned) == 0 {
		return &ServerRecoveryResult{
			OrphanedCount: 0,
			Message:       "Server recovery: no orphaned sessions found",
		}
	}

	// Resume orphaned sessions with staggered delay
	resumed := 0
	skipped := 0

	for i, orphan := range orphaned {
		// Check rate limit for this specific agent
		if d.serverRecoveryState.WasRecentlyRecovered(orphan.BeadsID, d.Config.ServerRecoveryRateLimit) {
			if d.Config.Verbose {
				fmt.Printf("  Skipping %s: already recovered recently (rate limit)\n", orphan.BeadsID)
			}
			skipped++
			continue
		}

		// Add delay between resumes (except for the first one)
		if i > 0 && d.Config.ServerRecoveryResumeDelay > 0 {
			time.Sleep(d.Config.ServerRecoveryResumeDelay)
		}

		if d.Config.Verbose {
			fmt.Printf("  Resuming orphaned session %s (phase=%s)\n", orphan.BeadsID, orphan.Phase)
		}

		if err := ResumeOrphanedAgent(orphan, serverURL); err != nil {
			if d.Config.Verbose {
				fmt.Printf("  Failed to resume %s: %v\n", orphan.BeadsID, err)
			}
			// Still mark as attempted to avoid retry storm
			d.serverRecoveryState.MarkRecovered(orphan.BeadsID)
			skipped++
			continue
		}

		// Mark as successfully recovered
		d.serverRecoveryState.MarkRecovered(orphan.BeadsID)
		resumed++

		if d.Config.Verbose {
			fmt.Printf("  Resumed %s successfully\n", orphan.BeadsID)
		}
	}

	return &ServerRecoveryResult{
		ResumedCount:  resumed,
		SkippedCount:  skipped,
		OrphanedCount: len(orphaned),
		Message:       fmt.Sprintf("Server recovery: %d orphaned found, %d resumed, %d skipped", len(orphaned), resumed, skipped),
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

// CrossProjectIssue represents an issue with its associated project context.
// Used for cross-project polling where issues need to track their source project.
type CrossProjectIssue struct {
	Issue   Issue
	Project Project
}

// CrossProjectOnceResult contains the result of processing one cross-project issue.
type CrossProjectOnceResult struct {
	Processed   bool
	Issue       *Issue
	Project     *Project
	Skill       string
	Message     string
	Error       error
	ProjectName string // Convenience field for logging: "[project-name]"
}

// ListCrossProjectIssues returns all triage:ready issues across all kb-registered projects.
// Issues are sorted by priority (0 = highest priority).
// Errors in individual projects are logged but don't stop processing of other projects.
func (d *Daemon) ListCrossProjectIssues() ([]CrossProjectIssue, error) {
	projects, err := d.listProjectsFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	var allIssues []CrossProjectIssue

	for _, project := range projects {
		issues, err := d.listIssuesForProjectFunc(project.Path)
		if err != nil {
			// Log error but continue to next project (per acceptance criteria)
			if d.Config.Verbose {
				fmt.Printf("  [%s] Failed to list issues: %v\n", project.Name, err)
			}
			continue
		}

		for _, issue := range issues {
			allIssues = append(allIssues, CrossProjectIssue{
				Issue:   issue,
				Project: project,
			})
		}
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(allIssues, func(i, j int) bool {
		return allIssues[i].Issue.Priority < allIssues[j].Issue.Priority
	})

	return allIssues, nil
}

// CrossProjectOnce processes a single issue from any kb-registered project.
// If cross-project mode is not enabled in config, this behaves like Once().
// Returns a result indicating what was processed and from which project.
//
// Key behaviors:
// - Iterates over all kb-registered projects
// - Respects global capacity limit (shared across all projects)
// - Error in one project doesn't block other projects
// - Includes project name in result for logging visibility
func (d *Daemon) CrossProjectOnce() (*CrossProjectOnceResult, error) {
	return d.CrossProjectOnceExcluding(nil)
}

// CrossProjectOnceExcluding processes a single issue from any kb-registered project,
// excluding any issues in the skip set. The skip map keys should be "projectPath:issueID".
func (d *Daemon) CrossProjectOnceExcluding(skip map[string]bool) (*CrossProjectOnceResult, error) {
	// Check rate limit first (before fetching issues)
	if d.RateLimiter != nil {
		canSpawn, count, msg := d.RateLimiter.CanSpawn()
		if !canSpawn {
			if d.Config.Verbose {
				fmt.Printf("  Rate limited: %s\n", msg)
			}
			return &CrossProjectOnceResult{
				Processed: false,
				Message:   fmt.Sprintf("Rate limited: %d/%d spawns in the last hour", count, d.RateLimiter.MaxPerHour),
			}, nil
		}
	}

	// Get all projects
	projects, err := d.listProjectsFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		return &CrossProjectOnceResult{
			Processed: false,
			Message:   "No kb-registered projects found",
		}, nil
	}

	// Collect all issues across projects
	var allIssues []CrossProjectIssue

	for _, project := range projects {
		issues, err := d.listIssuesForProjectFunc(project.Path)
		if err != nil {
			// Log error but continue to next project (per acceptance criteria)
			if d.Config.Verbose {
				fmt.Printf("  [%s] Failed to list issues: %v\n", project.Name, err)
			}
			continue
		}

		// Track skip reasons per project for summary logging (reduces log verbosity)
		var skipCounts struct {
			failedSpawn   int
			recentSpawn   int
			typeNotSpawn  int
			statusBlocked int
			missingLabel  int
		}
		spawnable := 0

		for _, issue := range issues {
			// Skip issues in the skip set
			skipKey := fmt.Sprintf("%s:%s", project.Path, issue.ID)
			if skip != nil && skip[skipKey] {
				skipCounts.failedSpawn++
				continue
			}

			// Skip issues that have been recently spawned
			if d.SpawnedIssues != nil && d.SpawnedIssues.IsSpawned(issue.ID) {
				skipCounts.recentSpawn++
				// Emit telemetry event when SpawnedIssueTracker blocks spawn
				if d.EventLogger != nil {
					_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
						"beads_id":    issue.ID,
						"dedup_layer": "spawned_tracker",
						"reason":      "Issue recently spawned, awaiting status update (6h TTL)",
					})
				}
				continue
			}

			// Skip non-spawnable types
			if !IsSpawnableType(issue.IssueType) {
				skipCounts.typeNotSpawn++
				continue
			}

			// Skip blocked or in_progress issues
			if issue.Status == "blocked" || issue.Status == "in_progress" {
				skipCounts.statusBlocked++
				continue
			}

			// Skip issues without required label (if filter is set)
			if d.Config.Label != "" && !issue.HasLabel(d.Config.Label) {
				skipCounts.missingLabel++
				continue
			}

			spawnable++
			allIssues = append(allIssues, CrossProjectIssue{
				Issue:   issue,
				Project: project,
			})
		}

		// Log skip summary per project (much less verbose than per-issue)
		if d.Config.Verbose {
			totalSkipped := skipCounts.failedSpawn + skipCounts.recentSpawn +
				skipCounts.typeNotSpawn + skipCounts.statusBlocked + skipCounts.missingLabel
			if totalSkipped > 0 || spawnable > 0 {
				var parts []string
				if spawnable > 0 {
					parts = append(parts, fmt.Sprintf("%d spawnable", spawnable))
				}
				if skipCounts.missingLabel > 0 {
					parts = append(parts, fmt.Sprintf("%d missing label", skipCounts.missingLabel))
				}
				if skipCounts.statusBlocked > 0 {
					parts = append(parts, fmt.Sprintf("%d blocked/in_progress", skipCounts.statusBlocked))
				}
				if skipCounts.typeNotSpawn > 0 {
					parts = append(parts, fmt.Sprintf("%d non-spawnable type", skipCounts.typeNotSpawn))
				}
				if skipCounts.recentSpawn > 0 {
					parts = append(parts, fmt.Sprintf("%d recently spawned", skipCounts.recentSpawn))
				}
				if skipCounts.failedSpawn > 0 {
					parts = append(parts, fmt.Sprintf("%d failed this cycle", skipCounts.failedSpawn))
				}
				fmt.Printf("  [%s] %s\n", project.Name, strings.Join(parts, ", "))
			}
		}
	}

	if len(allIssues) == 0 {
		return &CrossProjectOnceResult{
			Processed: false,
			Message:   "No spawnable issues in any project",
		}, nil
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(allIssues, func(i, j int) bool {
		return allIssues[i].Issue.Priority < allIssues[j].Issue.Priority
	})

	// Try each issue in priority order until one passes session/completion checks.
	// This fixes the bug where the daemon stops looking if the highest-priority
	// issue has an existing session or Phase: Complete.
	var selected *CrossProjectIssue
	var skill string
	var skippedReasons []string

	for i := range allIssues {
		candidate := &allIssues[i]

		// Infer skill for this candidate
		candidateSkill, err := InferSkillFromIssue(&candidate.Issue)
		if err != nil {
			if d.Config.Verbose {
				fmt.Printf("  [%s] Skipping %s (failed to infer skill: %v)\n",
					candidate.Project.Name, candidate.Issue.ID, err)
			}
			skippedReasons = append(skippedReasons,
				fmt.Sprintf("%s: failed to infer skill", candidate.Issue.ID))
			continue
		}

		// Session-level dedup: Check if there's an existing OpenCode session for this issue
		if HasExistingSessionForBeadsID(candidate.Issue.ID) {
			if d.Config.Verbose {
				fmt.Printf("  [%s] Skipping %s (existing OpenCode session found)\n",
					candidate.Project.Name, candidate.Issue.ID)
			}
			// Emit telemetry event when session dedup blocks spawn
			if d.EventLogger != nil {
				_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
					"beads_id":    candidate.Issue.ID,
					"dedup_layer": "session_dedup",
					"reason":      "Existing OpenCode session found via API check",
				})
			}
			skippedReasons = append(skippedReasons,
				fmt.Sprintf("%s: existing session", candidate.Issue.ID))
			continue
		}

		// Pre-spawn completion check: Skip issues where an agent has already reported
		// Phase: Complete but the orchestrator hasn't closed the issue yet.
		// Use project path for correct beads socket lookup in cross-project mode.
		if hasComplete, _ := HasPhaseCompleteForProject(candidate.Issue.ID, candidate.Project.Path); hasComplete {
			if d.Config.Verbose {
				fmt.Printf("  [%s] Skipping %s (Phase: Complete already reported)\n",
					candidate.Project.Name, candidate.Issue.ID)
			}
			// Emit telemetry event when Phase:Complete blocks spawn
			if d.EventLogger != nil {
				_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
					"beads_id":    candidate.Issue.ID,
					"dedup_layer": "phase_complete",
					"reason":      "Phase: Complete comment found in beads issue",
				})
			}
			skippedReasons = append(skippedReasons,
				fmt.Sprintf("%s: Phase: Complete", candidate.Issue.ID))
			continue
		}

		// This candidate passes all checks
		selected = candidate
		skill = candidateSkill
		break
	}

	// If no issue passed the checks, report what was skipped
	if selected == nil {
		msg := "No spawnable issues (all skipped due to existing sessions or Phase: Complete)"
		if len(skippedReasons) > 0 && d.Config.Verbose {
			msg = fmt.Sprintf("Skipped %d issues: %v", len(skippedReasons), skippedReasons)
		}
		return &CrossProjectOnceResult{
			Processed: false,
			Message:   msg,
		}, nil
	}

	// If pool is configured, acquire a slot first
	var slot *Slot
	if d.Pool != nil {
		slot = d.Pool.TryAcquire()
		if slot == nil {
			return &CrossProjectOnceResult{
				Processed:   false,
				Issue:       &selected.Issue,
				Project:     &selected.Project,
				Skill:       skill,
				ProjectName: selected.Project.Name,
				Message:     "At capacity - no slots available",
			}, nil
		}
		slot.BeadsID = selected.Issue.ID
	}

	// Mark issue as spawned BEFORE calling spawnFunc
	if d.SpawnedIssues != nil {
		d.SpawnedIssues.MarkSpawned(selected.Issue.ID)
	}

	// Spawn the work with project context
	if err := d.spawnForProjectFunc(selected.Issue.ID, selected.Project.Path); err != nil {
		// Unmark on spawn failure
		if d.SpawnedIssues != nil {
			d.SpawnedIssues.Unmark(selected.Issue.ID)
		}
		// Release slot on spawn failure
		if d.Pool != nil && slot != nil {
			d.Pool.Release(slot)
		}
		return &CrossProjectOnceResult{
			Processed:   false,
			Issue:       &selected.Issue,
			Project:     &selected.Project,
			Skill:       skill,
			ProjectName: selected.Project.Name,
			Error:       err,
			Message:     fmt.Sprintf("[%s] Failed to spawn: %v", selected.Project.Name, err),
		}, nil
	}

	// Record successful spawn for rate limiting
	if d.RateLimiter != nil {
		d.RateLimiter.RecordSpawn()
	}

	return &CrossProjectOnceResult{
		Processed:   true,
		Issue:       &selected.Issue,
		Project:     &selected.Project,
		Skill:       skill,
		ProjectName: selected.Project.Name,
		Message:     fmt.Sprintf("[%s] Spawned work on %s", selected.Project.Name, selected.Issue.ID),
	}, nil
}

// CrossProjectPreview shows what would be processed next without actually processing.
// Returns issues from all kb-registered projects, sorted by priority.
func (d *Daemon) CrossProjectPreview() (*CrossProjectPreviewResult, error) {
	result := &CrossProjectPreviewResult{}

	// Check rate limit status
	if d.RateLimiter != nil {
		canSpawn, count, msg := d.RateLimiter.CanSpawn()
		result.RateLimited = !canSpawn
		if d.RateLimiter.MaxPerHour > 0 {
			result.RateStatus = fmt.Sprintf("%d/%d spawns in last hour", count, d.RateLimiter.MaxPerHour)
		}
		if !canSpawn {
			result.Message = msg
		}
	}

	// Get all projects
	projects, err := d.listProjectsFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	result.Projects = projects

	if len(projects) == 0 {
		result.Message = "No kb-registered projects found"
		return result, nil
	}

	// Collect spawnable and rejected issues from all projects
	for _, project := range projects {
		issues, err := d.listIssuesForProjectFunc(project.Path)
		if err != nil {
			result.ProjectErrors = append(result.ProjectErrors, ProjectError{
				Project: project,
				Error:   err,
			})
			continue
		}

		for _, issue := range issues {
			reason := d.checkRejectionReason(issue)
			if reason != "" {
				result.RejectedIssues = append(result.RejectedIssues, CrossProjectRejected{
					Issue:   issue,
					Project: project,
					Reason:  reason,
				})
				continue
			}

			result.SpawnableIssues = append(result.SpawnableIssues, CrossProjectIssue{
				Issue:   issue,
				Project: project,
			})
		}
	}

	// Sort spawnable by priority
	sort.Slice(result.SpawnableIssues, func(i, j int) bool {
		return result.SpawnableIssues[i].Issue.Priority < result.SpawnableIssues[j].Issue.Priority
	})

	// Select the first spawnable issue (if any) for preview
	if len(result.SpawnableIssues) > 0 {
		first := result.SpawnableIssues[0]
		result.NextIssue = &first.Issue
		result.NextProject = &first.Project

		skill, err := InferSkillFromIssue(&first.Issue)
		if err == nil {
			result.Skill = skill
		}

		// Check for hotspot warnings if checker is configured
		if d.HotspotChecker != nil {
			result.HotspotWarnings = CheckHotspotsForIssue(&first.Issue, d.HotspotChecker)
		}
	} else if result.Message == "" {
		result.Message = "No spawnable issues in any project"
	}

	return result, nil
}

// CrossProjectPreviewResult contains the result of a cross-project preview operation.
type CrossProjectPreviewResult struct {
	NextIssue       *Issue
	NextProject     *Project
	Skill           string
	Message         string
	RateLimited     bool
	RateStatus      string
	HotspotWarnings []HotspotWarning
	Projects        []Project
	SpawnableIssues []CrossProjectIssue
	RejectedIssues  []CrossProjectRejected
	ProjectErrors   []ProjectError
}

// CrossProjectRejected captures a rejected issue with its project context.
type CrossProjectRejected struct {
	Issue   Issue
	Project Project
	Reason  string
}

// ProjectError captures an error that occurred while processing a project.
type ProjectError struct {
	Project Project
	Error   error
}

// FormatCrossProjectPreview formats cross-project preview results for display.
func FormatCrossProjectPreview(result *CrossProjectPreviewResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Projects scanned: %d\n", len(result.Projects)))

	if result.RateLimited {
		sb.WriteString(fmt.Sprintf("Rate limited: %s\n", result.Message))
	}

	if len(result.ProjectErrors) > 0 {
		sb.WriteString("\nProject errors:\n")
		for _, pe := range result.ProjectErrors {
			sb.WriteString(fmt.Sprintf("  [%s] %v\n", pe.Project.Name, pe.Error))
		}
	}

	if result.NextIssue != nil && result.NextProject != nil {
		sb.WriteString("\nNext to spawn:\n")
		sb.WriteString(fmt.Sprintf("  Project:  %s\n", result.NextProject.Name))
		sb.WriteString(FormatPreview(result.NextIssue))
		sb.WriteString(fmt.Sprintf("\nInferred skill: %s\n", result.Skill))
	} else {
		sb.WriteString(fmt.Sprintf("\n%s\n", result.Message))
	}

	if len(result.SpawnableIssues) > 1 {
		sb.WriteString(fmt.Sprintf("\nOther spawnable issues: %d\n", len(result.SpawnableIssues)-1))
		for i, cpi := range result.SpawnableIssues[1:] {
			if i >= 5 {
				sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(result.SpawnableIssues)-6))
				break
			}
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", cpi.Project.Name, cpi.Issue.ID, cpi.Issue.Title))
		}
	}

	if len(result.RejectedIssues) > 0 {
		sb.WriteString(fmt.Sprintf("\nRejected issues: %d\n", len(result.RejectedIssues)))
		for i, cpr := range result.RejectedIssues {
			if i >= 10 {
				sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(result.RejectedIssues)-10))
				break
			}
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", cpr.Project.Name, cpr.Issue.ID, cpr.Reason))
		}
	}

	return sb.String()
}
