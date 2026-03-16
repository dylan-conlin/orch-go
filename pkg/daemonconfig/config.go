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

	// ReflectModelDriftEnabled controls whether model drift reflection is enabled.
	// When enabled, the daemon will scan staleness events and create model maintenance issues.
	ReflectModelDriftEnabled bool

	// ReflectModelDriftInterval is how often to run model drift reflection (0 = disabled).
	// Default is 4 hours.
	ReflectModelDriftInterval time.Duration

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

	// KnowledgeHealthEnabled controls whether periodic knowledge health checks are enabled.
	// When enabled, the daemon counts active kb quick entries during idle cycles
	// and flags accumulation without promotion.
	KnowledgeHealthEnabled bool

	// KnowledgeHealthInterval is how often to run the knowledge health check (0 = disabled).
	// Default is 2 hours.
	KnowledgeHealthInterval time.Duration

	// KnowledgeHealthThreshold is the number of active quick entries that triggers
	// a triage:review issue for knowledge maintenance. Default is 50.
	KnowledgeHealthThreshold int

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

	// FrictionAccumulationEnabled controls whether periodic friction accumulation is enabled.
	// When enabled, the daemon scans recently-closed agents' beads comments for friction
	// reports and stores them in ~/.orch/friction.jsonl for pattern analysis.
	FrictionAccumulationEnabled bool

	// FrictionAccumulationInterval is how often to scan for friction items (0 = disabled).
	// Default is 1 hour.
	FrictionAccumulationInterval time.Duration

	// ArtifactSyncEnabled controls whether periodic artifact sync checking is enabled.
	// When enabled, the daemon analyzes drift events from ~/.orch/artifact-drift.jsonl
	// against ARTIFACT_MANIFEST.yaml and creates beads issues for drifted artifacts.
	ArtifactSyncEnabled bool

	// ArtifactSyncInterval is how often to check for artifact drift (0 = disabled).
	// Default is 24 hours (daily cadence).
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

	// RegistryRefreshEnabled controls whether the daemon periodically refreshes
	// its project registry. When enabled, new projects added to kb or groups.yaml
	// are picked up without requiring a daemon restart.
	RegistryRefreshEnabled bool

	// RegistryRefreshInterval is how often to rebuild the project registry.
	// Default is 5 minutes.
	RegistryRefreshInterval time.Duration

	// SynthesisAutoCreateEnabled controls whether the daemon auto-creates beads
	// issues for investigation clusters that lack a corresponding model directory.
	// When enabled, clusters detected by kb reflect with 5+ investigations (configurable)
	// and no .kb/models/{topic}/ directory will get a triage:ready issue created.
	SynthesisAutoCreateEnabled bool

	// SynthesisAutoCreateInterval is how often to check for synthesis opportunities.
	// Default is 2 hours. Runs after reflection to use fresh synthesis data.
	SynthesisAutoCreateInterval time.Duration

	// SynthesisAutoCreateThreshold is the minimum number of investigations in a cluster
	// before auto-creating a synthesis issue. Default is 5.
	SynthesisAutoCreateThreshold int

	// Compliance holds per-spawn compliance level configuration.
	// When nil/zero-value, defaults to ComplianceStrict (current behavior).
	Compliance ComplianceConfig

	// LearningRefreshEnabled controls whether the daemon periodically
	// recomputes learning metrics and auto-adjusts compliance levels.
	LearningRefreshEnabled bool

	// LearningRefreshInterval is how often to recompute learning metrics
	// and evaluate compliance auto-downgrades. Default is 1 hour.
	LearningRefreshInterval time.Duration

	// PlanStalenessEnabled controls whether periodic plan staleness detection is enabled.
	// When enabled, the daemon scans active plans in .kb/plans/ and detects:
	// - Unhydrated plans (active but no beads issues)
	// - Phase advancement stalls (completed phases with unstarted successors)
	// - No-progress plans (hydrated but no phases in progress or complete)
	PlanStalenessEnabled bool

	// PlanStalenessInterval is how often to check for stale plans (0 = disabled).
	// Default is 30 minutes.
	PlanStalenessInterval time.Duration

	// ProactiveExtractionEnabled controls whether periodic proactive extraction scanning is enabled.
	// When enabled, the daemon scans source files and creates architect issues for files
	// crossing 1200 lines (before they hit the 1500-line critical threshold that blocks spawning).
	ProactiveExtractionEnabled bool

	// ProactiveExtractionInterval is how often to scan for files approaching critical size.
	// Default is 6 hours.
	ProactiveExtractionInterval time.Duration

	// TriggerScanEnabled controls whether periodic pattern detection trigger scanning is enabled.
	// When enabled, the daemon runs pattern detectors that surface recurring bugs,
	// orphaned investigations, stale threads, etc. as beads issues.
	TriggerScanEnabled bool

	// TriggerScanInterval is how often to run the trigger scan (0 = disabled).
	// Default is 1 hour.
	TriggerScanInterval time.Duration

	// TriggerBudgetMax is the maximum number of open daemon:trigger issues allowed.
	// Prevents creation/removal asymmetry from bloating the issue queue.
	// Default is 10.
	TriggerBudgetMax int

	// TriggerExpiryEnabled controls whether periodic trigger expiry is enabled.
	// When enabled, the daemon auto-closes daemon:trigger issues not acted on
	// within TriggerExpiryMaxAge, addressing creation/removal asymmetry.
	TriggerExpiryEnabled bool

	// TriggerExpiryInterval is how often to check for expired trigger issues.
	// Default is 24 hours.
	TriggerExpiryInterval time.Duration

	// TriggerExpiryMaxAge is the maximum age for daemon:trigger issues before
	// they are auto-closed. Issues older than this are expired with the
	// daemon:expired label. Default is 14 days.
	TriggerExpiryMaxAge time.Duration

	// DigestEnabled controls whether the periodic digest producer is enabled.
	DigestEnabled bool

	// DigestInterval is how often to scan for artifact changes and produce digests.
	DigestInterval time.Duration

	// InvestigationOrphanEnabled controls whether periodic investigation orphan surfacing is enabled.
	// When enabled, the daemon surfaces investigations that have been in_progress for longer than
	// InvestigationOrphanThreshold without completion, creating closure pressure via notifications.
	InvestigationOrphanEnabled bool

	// InvestigationOrphanInterval is how often to check for orphaned investigations (0 = disabled).
	// Default is 1 hour.
	InvestigationOrphanInterval time.Duration

	// InvestigationOrphanThreshold is how long an investigation can be in_progress without
	// completion before being flagged as orphaned. Default is 48 hours.
	InvestigationOrphanThreshold time.Duration
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
		ReflectModelDriftEnabled:       true,
		ReflectModelDriftInterval:      4 * time.Hour,
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
		VerificationPauseThreshold:     5,                // Pause after 5 unique auto-completions
		KnowledgeHealthEnabled:         true,
		KnowledgeHealthInterval:        2 * time.Hour, // Every 2 hours
		KnowledgeHealthThreshold:       50,            // Flag when 50+ active entries
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
		FrictionAccumulationEnabled:    true,
		FrictionAccumulationInterval:   time.Hour, // Every hour
		ArtifactSyncEnabled:            true,
		ArtifactSyncInterval:           24 * time.Hour, // Daily cadence
		ArtifactSyncAutoSpawn:          false,          // Issues only by default
		ArtifactSyncAutoSpawnThreshold: 3,              // 3+ entries triggers auto-spawn
		RegistryRefreshEnabled:         true,
		RegistryRefreshInterval:        5 * time.Minute, // Refresh every 5 minutes
		SynthesisAutoCreateEnabled:     true,
		SynthesisAutoCreateInterval:    2 * time.Hour, // Every 2 hours (after reflection)
		SynthesisAutoCreateThreshold:   5,             // 5+ investigations triggers auto-create
		LearningRefreshEnabled:         true,
		LearningRefreshInterval:        time.Hour, // Hourly learning refresh + compliance auto-adjust
		PlanStalenessEnabled:           true,
		PlanStalenessInterval:          30 * time.Minute, // Check every 30 minutes
		ProactiveExtractionEnabled:     true,
		ProactiveExtractionInterval:    6 * time.Hour, // Every 6 hours
		TriggerScanEnabled:             true,
		TriggerScanInterval:            time.Hour, // Hourly trigger scan
		TriggerBudgetMax:               10,        // Max 10 open trigger issues
		TriggerExpiryEnabled:           true,
		TriggerExpiryInterval:          24 * time.Hour,      // Daily expiry check
		TriggerExpiryMaxAge:            14 * 24 * time.Hour, // 14-day TTL for trigger issues
		DigestEnabled:                  true,
		DigestInterval:                 30 * time.Minute,
		InvestigationOrphanEnabled:     true,
		InvestigationOrphanInterval:    time.Hour,      // Hourly check
		InvestigationOrphanThreshold:   48 * time.Hour, // 48h before flagging
	}
}
