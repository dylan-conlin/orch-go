// daemon_lifecycle.go contains daemon configuration, types, and constructors.
package daemon

import (
	"time"

	daemonsort "github.com/dylan-conlin/orch-go/pkg/daemon/sort"
	"github.com/dylan-conlin/orch-go/pkg/frontier"
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

	// GracePeriod is the delay before spawning a newly-seen triage:ready issue.
	// Allows orchestrator time to adjust labels/model before daemon grabs the issue.
	GracePeriod time.Duration

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
	// for supported kb reflect types (currently synthesis + defect-class).
	ReflectCreateIssues bool

	// PolishEnabled controls whether idle-time polish mode is enabled.
	// When enabled, the daemon runs low-priority self-improvement audits
	// when no triage:ready issues are available to spawn.
	PolishEnabled bool

	// PolishInterval is how often to run polish audits (0 = disabled).
	// Default is 30 minutes.
	PolishInterval time.Duration

	// PolishMaxIssuesPerCycle caps how many polish issues can be created in
	// a single poll cycle.
	// Default is 3.
	PolishMaxIssuesPerCycle int

	// PolishMaxIssuesPerDay caps how many polish issues can be created in
	// a UTC day window.
	// Default is 10.
	PolishMaxIssuesPerDay int

	// CleanupEnabled controls whether periodic cleanup is enabled.
	// When enabled, the daemon will run cleanup operations periodically.
	CleanupEnabled bool

	// CleanupInterval is how often to run cleanup (0 = disabled).
	// Default is 30 minutes.
	CleanupInterval time.Duration

	// CleanupSessions if true, cleans stale OpenCode sessions.
	// Default is true.
	CleanupSessions bool

	// CleanupSessionsAgeDays is the age threshold in days for session cleanup.
	// Sessions older than this will be deleted. Default is 7 days.
	CleanupSessionsAgeDays int

	// CleanupWorkspaces if true, archives stale completed workspaces.
	// Default is true.
	CleanupWorkspaces bool

	// CleanupWorkspacesAgeDays is the age threshold in days for workspace cleanup.
	// Workspaces older than this will be archived. Default is 7 days.
	CleanupWorkspacesAgeDays int

	// CleanupInvestigations if true, archives empty investigation files.
	// Default is true.
	CleanupInvestigations bool

	// CleanupPreserveOrchestrator if true, skips orchestrator sessions and workspaces.
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

	// SpawnFactualQuestions controls whether the daemon should spawn investigations
	// for factual questions (questions with subtype:factual label).
	// When enabled, queries 'bd ready --type question --label subtype:factual'
	// and spawns investigation skill for matching questions.
	// Default is false (opt-in feature).
	SpawnFactualQuestions bool

	// DeadSessionDetectionEnabled controls whether dead session detection is enabled.
	// When enabled, the daemon will periodically check for in_progress issues with
	// no active session and no Phase: Complete comment, marking them as failed.
	DeadSessionDetectionEnabled bool

	// DeadSessionDetectionInterval is how often to check for dead sessions (0 = disabled).
	// Default is 10 minutes.
	DeadSessionDetectionInterval time.Duration

	// MaxDeadSessionRetries is the maximum number of times a dead session can be
	// reset to open before escalating to needs:human. Derived from DEAD SESSION comment count.
	// Default is 2 (escalate after dying twice).
	MaxDeadSessionRetries int

	// OrphanReapEnabled controls whether periodic orphan process reaping is enabled.
	// When enabled, the daemon periodically scans for bun agent processes that are
	// not associated with any active OpenCode session and terminates them.
	OrphanReapEnabled bool

	// OrphanReapInterval is how often to scan for and kill orphan processes.
	// Default is 5 minutes.
	OrphanReapInterval time.Duration

	// SortMode selects the sort strategy for issue prioritization.
	// Available modes: "priority" (default), "unblock".
	// See pkg/daemon/sort/ for strategy details.
	SortMode string

	// DashboardWatchdogEnabled controls whether dashboard health monitoring is enabled.
	// When enabled, the daemon periodically checks if the dashboard API service
	// (orch serve on 3348) is responding and automatically restarts it via `orch-dashboard restart`.
	DashboardWatchdogEnabled bool

	// DashboardWatchdogInterval is how often to check dashboard health.
	// Default is 30 seconds (matches daemon poll interval for responsive detection).
	DashboardWatchdogInterval time.Duration

	// DashboardWatchdogFailuresBeforeRestart is how many consecutive health check
	// failures are required before triggering a restart. This prevents flapping
	// on transient network issues.
	// Default is 2 (restart after ~1 minute of consecutive failures at 30s interval).
	DashboardWatchdogFailuresBeforeRestart int

	// DashboardWatchdogRestartCooldown is the minimum time between restart attempts.
	// Prevents infinite restart loops when the underlying issue persists.
	// Default is 5 minutes.
	DashboardWatchdogRestartCooldown time.Duration
}

// DefaultConfig returns sensible defaults for daemon configuration.
func DefaultConfig() Config {
	return Config{
		PollInterval:                           time.Minute,
		MaxAgents:                              3,
		MaxSpawnsPerHour:                       20, // Prevents runaway spawning
		Label:                                  "triage:ready",
		SpawnDelay:                             10 * time.Second,
		DryRun:                                 false,
		Verbose:                                false,
		ReflectEnabled:                         true,
		ReflectInterval:                        time.Hour, // Hourly by default
		ReflectCreateIssues:                    true,
		PolishEnabled:                          true,
		PolishInterval:                         30 * time.Minute,
		PolishMaxIssuesPerCycle:                3,
		PolishMaxIssuesPerDay:                  10,
		CleanupEnabled:                         true,
		CleanupInterval:                        30 * time.Minute, // Every 30 minutes by default
		CleanupSessions:                        true,             // Clean sessions by default
		CleanupSessionsAgeDays:                 7,                // 7 days threshold for sessions
		CleanupWorkspaces:                      true,             // Archive stale workspaces by default
		CleanupWorkspacesAgeDays:               7,                // 7 days threshold for workspaces
		CleanupInvestigations:                  true,             // Archive empty investigations by default
		CleanupPreserveOrchestrator:            true,             // Preserve orchestrator sessions
		CleanupServerURL:                       "http://127.0.0.1:4096",
		RecoveryEnabled:                        true,
		RecoveryInterval:                       5 * time.Minute,  // Check every 5 minutes
		RecoveryIdleThreshold:                  10 * time.Minute, // Idle >10min triggers recovery
		RecoveryRateLimit:                      time.Hour,        // 1 resume per agent per hour
		ServerRecoveryEnabled:                  true,
		ServerRecoveryStabilizationDelay:       30 * time.Second,             // Wait 30s for server stability
		ServerRecoveryResumeDelay:              10 * time.Second,             // 10s between each resume
		ServerRecoveryRateLimit:                time.Hour,                    // 1 recovery per agent per hour
		MaxResumeAttempts:                      3,                            // Escalate after 3 failed attempts
		AutoAbandonAfterHours:                  24,                           // Auto-abandon after 24h dead
		SpawnFactualQuestions:                  false,                        // Opt-in feature
		DeadSessionDetectionEnabled:            true,                         // Enabled by default
		DeadSessionDetectionInterval:           10 * time.Minute,             // Check every 10 minutes
		MaxDeadSessionRetries:                  DefaultMaxDeadSessionRetries, // Escalate after N dead sessions
		GracePeriod:                            30 * time.Second,             // 30s grace period for triage corrections
		OrphanReapEnabled:                      true,                         // Enabled by default
		OrphanReapInterval:                     5 * time.Minute,              // Check every 5 minutes
		DashboardWatchdogEnabled:               true,                         // Enabled by default
		DashboardWatchdogInterval:              30 * time.Second,             // Check every 30s
		DashboardWatchdogFailuresBeforeRestart: 2,                            // Restart after 2 consecutive failures (~1min)
		DashboardWatchdogRestartCooldown:       5 * time.Minute,              // 5min between restarts
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

	// SortStrategy is the active sort strategy for issue prioritization.
	// Initialized from Config.SortMode. Defaults to PriorityStrategy.
	SortStrategy daemonsort.Strategy

	// CachedFrontier holds the most recently computed frontier state.
	// Updated once per poll cycle and shared with sort strategies via SortContext.
	// May be nil if frontier computation fails or hasn't run yet.
	CachedFrontier *frontier.FrontierState

	// Pool is the worker pool for concurrency control.
	// If set, it is used instead of activeCountFunc.
	Pool *WorkerPool

	// RateLimiter tracks spawn history for hourly rate limiting.
	RateLimiter *RateLimiter

	// HotspotChecker checks for hotspot areas before spawning.
	// If set, Preview will include hotspot warnings.
	HotspotChecker HotspotChecker

	// ProcessedCache provides unified deduplication for spawned issues.
	// Consolidates three fragmented dedup mechanisms:
	// 1. Persistent cache (survives daemon restart)
	// 2. Session dedup (checks OpenCode sessions)
	// 3. Phase Complete check (checks beads comments)
	ProcessedCache *ProcessedIssueCache

	// firstSeen tracks when each issue was first observed for grace period support.
	firstSeen map[string]time.Time

	// SpawnedIssues is deprecated - use ProcessedCache instead.
	// Kept for backward compatibility during migration.
	SpawnedIssues *SpawnedIssueTracker

	// EventLogger is used to log deduplication events for telemetry.
	// If nil, events are not logged.
	EventLogger EventLogger

	// lastReflect tracks when reflection was last run for periodic reflection.
	lastReflect time.Time

	// lastPolish tracks when polish mode was last run.
	lastPolish time.Time

	// polishWindowStart is the start time of the current UTC day window for
	// polish issue daily caps.
	polishWindowStart time.Time

	// polishCreatedToday counts issues created by polish mode in the current
	// UTC day window.
	polishCreatedToday int

	// lastCleanup tracks when session cleanup was last run for periodic cleanup.
	lastCleanup time.Time

	// lastRecovery tracks when recovery was last run for periodic recovery.
	lastRecovery time.Time

	// lastDeadSessionDetection tracks when dead session detection was last run.
	lastDeadSessionDetection time.Time

	// lastOrphanReap tracks when orphan process reaping was last run.
	lastOrphanReap time.Time

	// lastDashboardCheck tracks when dashboard health was last checked.
	lastDashboardCheck time.Time

	// lastDashboardRestart tracks when dashboard was last restarted (for cooldown).
	lastDashboardRestart time.Time

	// dashboardConsecutiveFailures counts consecutive health check failures.
	// Reset to 0 when services are healthy.
	dashboardConsecutiveFailures int

	// restartDashboardFunc is the function that performs the actual restart.
	// Defaults to restartDashboard() which runs `orch-dashboard restart`.
	// Can be overridden for testing.
	restartDashboardFunc func() error

	// resumeAttempts tracks when we last attempted to resume each agent (by beads ID).
	// Prevents infinite resume loops by rate-limiting to 1 attempt per hour per agent.
	resumeAttempts map[string]time.Time

	// resumeAttemptCounts tracks how many times we've attempted to resume each agent.
	// Used for escalation after N failed attempts.
	resumeAttemptCounts map[string]int

	// serverRecoveryState tracks state for server restart recovery.
	// Used to determine when server recovery should run (once per daemon start).
	serverRecoveryState *ServerRecoveryState

	// hasSessionFunc is used for testing - allows mocking session dedup check.
	// Defaults to HasExistingSessionForBeadsID when nil.
	hasSessionFunc func(beadsID string) bool
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
	// collectPolishCandidatesFunc is used for testing - allows mocking polish audits
	collectPolishCandidatesFunc func(projectDir string) ([]PolishIssueSpec, error)
	// createPolishIssueFunc is used for testing - allows mocking polish issue creation
	createPolishIssueFunc func(spec PolishIssueSpec) (string, error)
	// listAllIssuesFunc is used for testing - allows mocking list of active issues
	listAllIssuesFunc func() ([]Issue, error)
	// listEpicChildrenFunc is used for testing - allows mocking ListEpicChildren
	listEpicChildrenFunc func(epicID string) ([]Issue, error)
	// listProjectsFunc is used for testing - allows mocking kb projects list
	listProjectsFunc func() ([]Project, error)
	// listIssuesForProjectFunc is used for testing - allows mocking ListReadyIssuesForProject
	listIssuesForProjectFunc func(projectPath string) ([]Issue, error)
	// blockersFunc is used for testing - allows mocking dependency checks
	blockersFunc func(issueID, projectPath string) ([]string, error)
	// spawnForProjectFunc is used for testing - allows mocking SpawnWorkForProject
	spawnForProjectFunc func(beadsID, projectPath string) error
	// closedIssuesBatchFunc is used for testing - allows mocking closed-issue lookups
	closedIssuesBatchFunc func(beadsIDs []string) map[string]bool
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

	// Initialize sort strategy from config
	sortStrategy, err := daemonsort.Get(config.SortMode)
	if err != nil {
		// Fall back to priority sort if mode is invalid
		sortStrategy = &daemonsort.PriorityStrategy{}
	}

	d := &Daemon{
		Config:                   config,
		SortStrategy:             sortStrategy,
		SpawnedIssues:            NewSpawnedIssueTracker(DefaultSpawnedIssueTrackerMaxEntries, DefaultSpawnedIssueTrackerTTL),
		EventLogger:              nil, // Set via SetEventLogger() to avoid circular deps
		resumeAttempts:           make(map[string]time.Time),
		resumeAttemptCounts:      make(map[string]int),
		serverRecoveryState:      NewServerRecoveryState(),
		restartDashboardFunc:     restartDashboard,
		listIssuesFunc:           ListReadyIssues,
		spawnFunc:                SpawnWork,
		activeCountFunc:          activeCount,
		reflectFunc:              DefaultRunReflection,
		listEpicChildrenFunc:     ListEpicChildren,
		listProjectsFunc:         ListProjects,
		listIssuesForProjectFunc: ListReadyIssuesForProject,
		spawnForProjectFunc:      SpawnWorkForProject,
		closedIssuesBatchFunc:    GetClosedIssuesBatch,
	}
	d.collectPolishCandidatesFunc = d.collectPolishCandidates
	d.createPolishIssueFunc = d.createPolishIssue
	d.listAllIssuesFunc = ListOpenAndInProgressIssues
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

	// Initialize sort strategy from config
	sortStrategy, err := daemonsort.Get(config.SortMode)
	if err != nil {
		sortStrategy = &daemonsort.PriorityStrategy{}
	}

	d := &Daemon{
		Config:                   config,
		SortStrategy:             sortStrategy,
		Pool:                     pool,
		SpawnedIssues:            NewSpawnedIssueTracker(DefaultSpawnedIssueTrackerMaxEntries, DefaultSpawnedIssueTrackerTTL),
		EventLogger:              nil, // Set via SetEventLogger() to avoid circular deps
		resumeAttempts:           make(map[string]time.Time),
		serverRecoveryState:      NewServerRecoveryState(),
		restartDashboardFunc:     restartDashboard,
		listIssuesFunc:           ListReadyIssues,
		spawnFunc:                SpawnWork,
		activeCountFunc:          activeCount,
		reflectFunc:              DefaultRunReflection,
		listProjectsFunc:         ListProjects,
		listIssuesForProjectFunc: ListReadyIssuesForProject,
		spawnForProjectFunc:      SpawnWorkForProject,
		closedIssuesBatchFunc:    GetClosedIssuesBatch,
	}
	d.collectPolishCandidatesFunc = d.collectPolishCandidates
	d.createPolishIssueFunc = d.createPolishIssue
	d.listAllIssuesFunc = ListOpenAndInProgressIssues
	// Initialize rate limiter if MaxSpawnsPerHour is set
	if config.MaxSpawnsPerHour > 0 {
		d.RateLimiter = NewRateLimiter(config.MaxSpawnsPerHour)
	}
	return d
}
