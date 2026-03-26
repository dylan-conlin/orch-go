// Package daemonconfig provides daemon configuration defaults and types.
package daemonconfig

import "time"

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

	// ReflectEnabled controls whether periodic reflection analysis is enabled.
	// When enabled, the daemon will run kb reflect periodically.
	ReflectEnabled bool

	// ReflectInterval is how often to run kb reflect (0 = disabled).
	// Default is 1 hour.
	ReflectInterval time.Duration

	// ReflectCreateIssues controls whether reflection creates beads issues
	// for synthesis opportunities (topics with 10+ investigations).
	ReflectCreateIssues bool

	// ReflectOpenEnabled controls whether reflection creates issues for open
	// investigation actions (Next: items older than 3 days).
	ReflectOpenEnabled bool



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

	// CleanupArchivedTTLDays is the TTL in days for archived workspace expiry.
	// Archived workspaces older than this are deleted. Default is 30 days.
	CleanupArchivedTTLDays int

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

	// VerificationPauseThreshold is the maximum number of agents that can be marked
	// ready-for-review before pausing for human verification. When the daemon marks
	// this many issues as ready-for-review without human verification (manual orch complete),
	// it will pause spawning until Dylan explicitly resumes. Set to 0 to disable (no pause).
	// Default is 3.
	VerificationPauseThreshold int

	// OrphanDetectionEnabled controls whether periodic orphan detection is enabled.
	// When enabled, the daemon detects in_progress issues with no active agent
	// (no OpenCode session, no tmux window) and resets them to open for respawning.
	OrphanDetectionEnabled bool

	// OrphanDetectionInterval is how often to check for orphaned issues (0 = disabled).
	// Default is 30 minutes.
	OrphanDetectionInterval time.Duration

	// OrphanAgeThreshold is how long an issue must be in_progress with no agent
	// before it's considered orphaned and reset to open. Default is 1 hour.
	OrphanAgeThreshold time.Duration

	// PhaseTimeoutEnabled controls whether periodic phase timeout detection is enabled.
	// When enabled, the daemon detects agents that have an active session but haven't
	// reported a new phase comment within PhaseTimeoutThreshold. These agents are
	// surfaced as "unresponsive" in orch status and daemon logs.
	PhaseTimeoutEnabled bool

	// PhaseTimeoutInterval is how often to check for unresponsive agents (0 = disabled).
	// Default is 5 minutes.
	PhaseTimeoutInterval time.Duration

	// PhaseTimeoutThreshold is how long an agent can go without a phase update
	// before being flagged as unresponsive. Default is 30 minutes.
	PhaseTimeoutThreshold time.Duration

	// AgreementCheckEnabled controls whether periodic agreement checking is enabled.
	// When enabled, the daemon runs kb agreements check periodically and auto-creates
	// beads issues for failing error-severity agreements (with label-based dedup).
	AgreementCheckEnabled bool

	// AgreementCheckInterval is how often to run agreement checks (0 = disabled).
	// Default is 30 minutes.
	AgreementCheckInterval time.Duration

	// InvariantCheckEnabled controls whether daemon self-check invariants run each poll cycle.
	// When enabled, the daemon validates assumptions about its state (active count range,
	// verification counter bounds, completion agent validity) and pauses after repeated violations.
	InvariantCheckEnabled bool

	// InvariantViolationThreshold is the number of consecutive poll cycles with invariant
	// violations before the daemon pauses. Default is 3. Set to 0 to disable.
	InvariantViolationThreshold int

	// BeadsHealthEnabled controls whether periodic beads health snapshot collection is enabled.
	// When enabled, the daemon collects health metrics (open/blocked/stale issues, bloated files,
	// fix:feat ratio) and appends them to the health snapshot store for trend analysis.
	BeadsHealthEnabled bool

	// BeadsHealthInterval is how often to collect beads health snapshots (0 = disabled).
	// Default is 1 hour.
	BeadsHealthInterval time.Duration

	// ArtifactSyncEnabled controls whether periodic artifact sync checking is enabled.
	// When enabled, the daemon analyzes drift events from ~/.orch/artifact-drift.jsonl
	// against ARTIFACT_MANIFEST.yaml and creates beads issues for drifted artifacts.
	ArtifactSyncEnabled bool

	// ArtifactSyncInterval is how often to check for artifact drift (0 = disabled).
	ArtifactSyncInterval time.Duration

	// ArtifactSyncProjectDir is the project directory containing ARTIFACT_MANIFEST.yaml.
	// Defaults to current working directory if empty.
	ArtifactSyncProjectDir string

	// ArtifactSyncAutoSpawn controls whether the daemon auto-spawns a sync agent
	// when drift exceeds the threshold. When false, only creates beads issues.
	ArtifactSyncAutoSpawn bool

	// ArtifactSyncAutoSpawnThreshold is the minimum number of drifted artifact entries
	// needed to auto-spawn a sync agent. Default is 3.
	ArtifactSyncAutoSpawnThreshold int

	// ArtifactSyncCLAUDEMDLineBudget is the maximum number of lines CLAUDE.md should
	// contain. When over budget, sync agents are instructed to remove lowest-relevance
	// content before adding new content. Default is 300.
	ArtifactSyncCLAUDEMDLineBudget int

	// RegistryRefreshEnabled controls whether the daemon periodically refreshes
	// its project registry. When enabled, new projects added to kb or groups.yaml
	// are picked up without requiring a daemon restart.
	RegistryRefreshEnabled bool

	// RegistryRefreshInterval is how often to rebuild the project registry.
	// Default is 5 minutes.
	RegistryRefreshInterval time.Duration

	// Compliance holds per-spawn compliance level configuration.
	// When nil/zero-value, defaults to ComplianceStrict (current behavior).
	Compliance ComplianceConfig

	// ModelRouting holds per-spawn model routing configuration.
	// When nil, the hardcoded skillModelMapping in skill_inference.go is used.
	ModelRouting *ModelRoutingConfig

	// VerificationFailedEscalationEnabled controls whether the daemon periodically
	// escalates verification-failed agents to triage:review for human attention.
	// Issues labeled daemon:verification-failed sit in_progress indefinitely after
	// exhausting their retry budget — this task adds triage:review after a timeout.
	VerificationFailedEscalationEnabled bool

	// VerificationFailedEscalationInterval is how often to scan for verification-failed issues.
	// Default is 30 minutes.
	VerificationFailedEscalationInterval time.Duration

	// VerificationFailedEscalationTimeout is how long a verification-failed issue must
	// exist before being escalated to triage:review. Default is 1 hour.
	VerificationFailedEscalationTimeout time.Duration

	// LightweightCleanupEnabled controls whether the daemon periodically closes stale
	// tier:lightweight issues (created by --no-track spawns, including exploration children).
	// These issues are ephemeral by design and should be closed when their parent completes
	// or they've been idle too long.
	LightweightCleanupEnabled bool

	// LightweightCleanupInterval is how often to scan for stale lightweight issues.
	// Default is 30 minutes.
	LightweightCleanupInterval time.Duration

	// LightweightCleanupTimeout is how long a tier:lightweight issue can be in_progress
	LightweightCleanupTimeout time.Duration


	// CapacityPollEnabled controls whether the daemon periodically polls account capacity
	// and writes the result to ~/.orch/capacity-cache.json for orch status to read.
	CapacityPollEnabled bool

	// CapacityPollInterval is how often to poll account capacity from the Anthropic API.
	CapacityPollInterval time.Duration

	// AuditSelectEnabled controls whether the daemon periodically selects
	// random completed issues for quality audit (deep review).
	AuditSelectEnabled bool

	// AuditSelectInterval is how often to run audit selection (default: 168h / 7 days).
	AuditSelectInterval time.Duration

	// AuditSelectCount is the number of issues to select per audit cycle (default: 2).
	AuditSelectCount int

	// AuditAutoCompleteWeight is the fraction of selections drawn from the
	// auto-completed pool (0.0–1.0). The remainder comes from all completions.
	// Default: 0.6 (60% auto-completed, 40% any completion).
	AuditAutoCompleteWeight float64

	// ComprehensionThreshold is the maximum number of comprehension:unread items
	// before the daemon pauses spawning. The daemon adds this label after
	// auto-completing agents; orch complete transitions it to comprehension:processed.
	// Default: 5. Set to 0 to disable comprehension throttle.
	ComprehensionThreshold int
}

// DefaultConfig returns sensible defaults for daemon configuration.
func DefaultConfig() Config {
	return Config{
		PollInterval:                   15 * time.Second, // Faster polling for responsive dashboard updates
		MaxAgents:                      5,
		MaxSpawnsPerHour:               20, // Prevents runaway spawning
		Label:                          "triage:ready",
		SpawnDelay:                     3 * time.Second, // Reduced from 10s - dedup cache prevents duplicates
		DryRun:                         false,
		Verbose:                        false,
		ReflectEnabled:                 true,
		ReflectInterval:                time.Hour, // Hourly by default
		ReflectCreateIssues:            true,
		ReflectOpenEnabled:             true,
		CleanupEnabled:                 true,
		CleanupInterval:                6 * time.Hour, // Every 6 hours by default
		CleanupAgeDays:                 7,             // 7 days threshold
		CleanupPreserveOrchestrator:    true,          // Preserve orchestrator sessions
		CleanupServerURL:               "http://127.0.0.1:4096",
		CleanupArchivedTTLDays:         30, // 30-day TTL for archived workspace expiry
		RecoveryEnabled:                true,
		RecoveryInterval:               5 * time.Minute,  // Check every 5 minutes
		RecoveryIdleThreshold:          10 * time.Minute, // Idle >10min triggers recovery
		RecoveryRateLimit:              time.Hour,        // 1 resume per agent per hour
		VerificationPauseThreshold:     5, // Pause after 5 unique auto-completions
		OrphanDetectionEnabled:         true,
		OrphanDetectionInterval:        30 * time.Minute, // Check every 30 minutes
		OrphanAgeThreshold:             time.Hour,        // 1 hour before considering orphaned
		PhaseTimeoutEnabled:            true,
		PhaseTimeoutInterval:           5 * time.Minute,  // Check every 5 minutes
		PhaseTimeoutThreshold:          30 * time.Minute, // Flag after 30 minutes without phase update
		AgreementCheckEnabled:          true,
		AgreementCheckInterval:         30 * time.Minute, // Check every 30 minutes
		InvariantCheckEnabled:          true,
		InvariantViolationThreshold:    3, // Pause after 3 consecutive violation cycles
		BeadsHealthEnabled:             true,
		BeadsHealthInterval:            time.Hour, // Every hour
		ArtifactSyncEnabled:            true,
		ArtifactSyncInterval:           24 * time.Hour, // Daily cadence
		ArtifactSyncAutoSpawn:          false,          // Issues only by default
		ArtifactSyncAutoSpawnThreshold: 3,              // 3+ entries triggers auto-spawn
		ArtifactSyncCLAUDEMDLineBudget: 300,            // CLAUDE.md line budget
		RegistryRefreshEnabled:         true,
		RegistryRefreshInterval:        5 * time.Minute, // Refresh every 5 minutes
		VerificationFailedEscalationEnabled:      true,
		VerificationFailedEscalationInterval:     30 * time.Minute, // Check every 30 minutes
		VerificationFailedEscalationTimeout:      time.Hour,        // 1h before escalating to triage:review
		LightweightCleanupEnabled:                true,
		LightweightCleanupInterval:               30 * time.Minute, // Check every 30 minutes
		LightweightCleanupTimeout:                2 * time.Hour,    // 2h before auto-closing
		CapacityPollEnabled:                     true,
		CapacityPollInterval:                    5 * time.Minute, // Poll every 5 minutes
		AuditSelectEnabled:                      true,
		AuditSelectInterval:                     168 * time.Hour, // Weekly
		AuditSelectCount:                        2,               // 2 issues per cycle
		AuditAutoCompleteWeight:                 0.6,             // 60% from auto-completed pool
		ComprehensionThreshold:                  5,               // Pause after 5 uncomprehended items
	}
}
