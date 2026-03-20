// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/group"
	"github.com/dylan-conlin/orch-go/pkg/modeldrift"
)

// Config holds configuration for the daemon.
type Config = daemonconfig.Config

// DefaultConfig returns sensible defaults for daemon configuration.
func DefaultConfig() Config {
	return daemonconfig.DefaultConfig()
}

// OnceResult contains the result of processing one issue.
type OnceResult struct {
	Processed bool
	Issue     *Issue
	Skill     string
	Model     string // Inferred model alias (e.g., "opus", "sonnet")
	Message   string
	Error     error

	// ExtractionSpawned indicates that an extraction agent was spawned instead of the original issue.
	// The original issue was given a blocking dependency on the extraction issue.
	ExtractionSpawned bool
	// OriginalIssueID is set when ExtractionSpawned is true, containing the ID of the
	// original issue that will be spawned after extraction completes.
	OriginalIssueID string

	// ArchitectEscalated indicates that the skill was escalated from an implementation skill
	// (feature-impl or systematic-debugging) to architect because the issue targets a hotspot area.
	// This implements Layer 2 of hotspot enforcement (daemon-level skill routing).
	ArchitectEscalated bool
	// ArchitectEscalationDetail contains the full escalation decision when a hotspot match was found.
	// Non-nil whenever an implementation skill targets a hotspot area, regardless of whether
	// escalation actually happened (PriorArchitectRef may have prevented it).
	ArchitectEscalationDetail *ArchitectEscalation
}

// Daemon manages autonomous issue processing.
type Daemon struct {
	// Config holds the daemon configuration.
	Config Config

	// Pool is the worker pool for concurrency control.
	Pool *WorkerPool

	// RateLimiter tracks spawn history for hourly rate limiting.
	RateLimiter *RateLimiter

	// HotspotChecker checks for hotspot areas before spawning.
	// If set, Preview will include hotspot warnings.
	HotspotChecker HotspotChecker

	// PriorArchitectFinder searches for closed architect issues covering given files.
	// If set and returns a match, daemon skips architect escalation for that area.
	PriorArchitectFinder PriorArchitectFinder

	// SpawnedIssues tracks issue IDs that have been spawned but may not yet
	// have their beads status updated to in_progress. This prevents the race
	// condition where the daemon spawns duplicate agents for the same issue
	// because the status update hasn't propagated yet.
	SpawnedIssues *SpawnedIssueTracker

	// Scheduler manages timing for all periodic maintenance tasks.
	// Replaces individual last* fields with a unified scheduler.
	Scheduler *PeriodicScheduler

	// questionNotified tracks which agents have been notified about QUESTION phase.
	// Prevents duplicate notifications. Cleaned when agent leaves QUESTION phase.
	questionNotified map[string]time.Time

	// resumeAttempts tracks when we last attempted to resume each agent (by beads ID).
	// Prevents infinite resume loops by rate-limiting to 1 attempt per hour per agent.
	resumeAttempts map[string]time.Time

	// VerificationTracker tracks completions since last human verification and manages
	// pause state when threshold is reached. This enforces the verifiability-first
	// constraint by pausing autonomous operation after N completions without human review.
	VerificationTracker *VerificationTracker

	// CompletionFailureTracker tracks completion processing failures to surface them in health metrics.
	// This prevents silent failure when CompletionOnce persistently fails (e.g., beads database issues).
	CompletionFailureTracker *CompletionFailureTracker

	// VerificationRetryTracker tracks per-agent verification failures across poll cycles.
	// After exhausting the retry budget (3 attempts for local, 1 for cross-project),
	// the agent is labeled daemon:verification-failed and excluded from future scans.
	// This prevents the infinite retry loop on verification failures.
	VerificationRetryTracker *VerificationRetryTracker

	// Issues queries beads issues for the spawn pipeline.
	Issues IssueQuerier
	// ProjectRegistry resolves issue ID prefixes to project directories.
	// When set, the daemon passes --workdir to orch work for cross-project issues.
	ProjectRegistry *ProjectRegistry

	// AutoCompleter runs the full orch complete pipeline for auto-tier agents.
	// When set and review tier is "auto", the daemon shells out to orch complete
	// instead of just labeling the issue for orchestrator review.
	AutoCompleter AutoCompleter

	// Spawner spawns agent work.
	Spawner Spawner
	// Completions finds completed agents.
	Completions CompletionFinder
	// Reflector runs knowledge reflection.
	Reflector Reflector
	// ModelDrift provides I/O for model drift analysis.
	ModelDrift modeldrift.Store
	// KnowledgeHealth provides knowledge health operations.
	KnowledgeHealth KnowledgeHealthService
	// AgreementCheck provides agreement checking operations.
	AgreementCheck AgreementCheckService
	// Cleaner cleans up stale sessions.
	Cleaner SessionCleaner
	// ActiveCounter counts active agents for pool reconciliation.
	ActiveCounter ActiveCounter
	// Agents discovers agents for orphan detection and recovery.
	Agents AgentDiscoverer
	// StatusUpdater updates beads issue status.
	StatusUpdater IssueUpdater

	// BeadsHealth provides beads health snapshot collection and storage.
	BeadsHealth BeadsHealthService

	// FrictionAccumulator scans completed agents for friction and stores results.
	FrictionAccumulator FrictionAccumulatorService

	// ArtifactSync provides periodic artifact drift analysis and issue creation.
	ArtifactSync ArtifactSyncService

	// SynthesisAutoCreate provides periodic auto-creation of synthesis issues
	// for investigation clusters lacking model directories.
	SynthesisAutoCreate SynthesisAutoCreateService

	// ProactiveExtraction provides periodic scanning for files approaching the
	// critical 1500-line threshold. Creates architect issues at 1200 lines.
	ProactiveExtraction ProactiveExtractionService

	// TriggerScan provides periodic pattern detection that creates beads issues
	// for recurring bugs, orphaned investigations, stale threads, etc.
	TriggerScan TriggerScanService

	// DetectorOutcomes provides I/O for computing per-detector resolution rates.
	// When set, RunPeriodicTriggerScan adjusts budgets based on detector performance.
	DetectorOutcomes DetectorOutcomeService

	// TriggerDetectors holds the registered pattern detectors for the trigger scan system.
	// Constructed at daemon startup, used by RunPeriodicTriggerScan.
	TriggerDetectors []PatternDetector

	// TriggerExpiry provides periodic expiry of stale daemon:trigger issues.
	TriggerExpiry TriggerExpiryService

	// Digest provides periodic scanning of .kb/ artifacts and production
	// of consumable digest products in ~/.orch/digest/.
	Digest DigestService
	// DigestDir is the directory where digest product files are stored.
	DigestDir string
	// DigestStatePath is the path to the digest scan state file.
	DigestStatePath string

	// CompletionDedupTracker prevents re-processing the same Phase: Complete
	// across poll cycles. Defense-in-depth for when daemon:ready-review label
	// fails to persist (beads flakiness, label removed externally).
	CompletionDedupTracker *CompletionDedupTracker

	// BeadsCircuitBreaker tracks consecutive bd command failures and provides
	// exponential backoff to prevent lock cascade when beads is unhealthy.
	BeadsCircuitBreaker *BeadsCircuitBreaker

	// InvariantChecker runs self-check invariants each poll cycle to catch
	// scope-expansion bugs (e.g., ghost agents, counter overflow, missing ProjectDir).
	// Pauses daemon after configurable threshold of consecutive violation cycles.
	InvariantChecker *InvariantChecker

	// GroupConfig holds groups.yaml for account routing per project group.
	// When set, the daemon resolves the account to use before spawning
	// based on which group the target project belongs to.
	// Refreshed during periodic registry refresh to pick up new group members.
	GroupConfig *group.Config
	// KBProjects maps project name -> absolute path for group membership resolution.
	// Built from ProjectRegistry, used by GroupConfig.AccountForProjectDir.
	// Refreshed during periodic registry refresh to include newly discovered projects.
	KBProjects map[string]string
	// GroupFilter is the --group flag value from daemon startup.
	// When set, periodic registry refresh reapplies the group filter after rebuilding
	// so new group members are discovered without requiring daemon restart.
	GroupFilter string

	// Learning holds aggregated per-skill metrics from events.jsonl.
	// When set, PrioritizeIssues uses skill-aware scoring instead of pure priority sort.
	// Computed via events.ComputeLearning() at daemon startup or refresh.
	Learning *events.LearningStore

	// PlanStatusQuerier queries beads for plan phase issue statuses.
	// When nil, uses defaultPlanStatusQuerier (shells out to bd).
	PlanStatusQuerier PlanStatusQuerier

	// FocusGoal is the current focus goal text (from ~/.orch/focus.json).
	// When set, issues from projects matching this goal get a priority boost.
	FocusGoal string
	// FocusBoostAmount is how many priority levels to boost (default: 1).
	// E.g., with boost=1, a P2 issue becomes effectively P1.
	FocusBoostAmount int
	// ProjectDirNames maps project prefixes to directory basenames for focus matching.
	// Built from ProjectRegistry at daemon startup.
	ProjectDirNames map[string]string

	// ClaimProbeService generates probe issues for stale/unconfirmed claims.
	ClaimProbeService ClaimProbeService

	// TensionClusterService creates architect issues for tension clusters.
	TensionClusterService TensionClusterService
}

// New creates a new Daemon instance with default configuration.
func New() *Daemon {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new Daemon instance with the given configuration.
func NewWithConfig(config Config) *Daemon {
	// Use disk-backed spawn tracker to survive daemon restarts.
	// Falls back to in-memory if path resolution fails.
	spawnTracker := NewSpawnedIssueTracker()
	if cachePath := DefaultSpawnCachePath(); cachePath != "" {
		spawnTracker = NewSpawnedIssueTrackerWithFile(cachePath)
	}
	// Derive thresholds from compliance level
	verificationThreshold := daemonconfig.DeriveVerificationThreshold(config.Compliance.Default)
	invariantThreshold := daemonconfig.DeriveInvariantThreshold(config.Compliance.Default)

	d := &Daemon{
		Config:                   config,
		Scheduler:                NewSchedulerFromConfig(config),
		SpawnedIssues:            spawnTracker,
		resumeAttempts:           make(map[string]time.Time),
		VerificationTracker:      NewVerificationTracker(verificationThreshold),
		CompletionFailureTracker: NewCompletionFailureTracker(),
		VerificationRetryTracker: NewVerificationRetryTracker(),
		Issues:                   &defaultIssueQuerier{},
		Spawner:                  &defaultSpawner{},
		Completions:              &defaultCompletionFinder{},
		Reflector:                &defaultReflector{},
		ModelDrift:               modeldrift.NewDefaultStore(),
		KnowledgeHealth:          &defaultKnowledgeHealthService{},
		AgreementCheck:           &defaultAgreementCheckService{},
		Cleaner:                  &defaultSessionCleaner{},
		ActiveCounter:            &defaultActiveCounter{},
		Agents:                   &defaultAgentDiscoverer{},
		StatusUpdater:            &defaultIssueUpdater{},
		CompletionDedupTracker:   NewCompletionDedupTracker(),
		BeadsCircuitBreaker:      NewBeadsCircuitBreaker(),
		ArtifactSync:             &defaultArtifactSyncService{},
		SynthesisAutoCreate:      &defaultSynthesisAutoCreateService{},
	}
	// Initialize worker pool if MaxAgents is set
	if config.MaxAgents > 0 {
		d.Pool = NewWorkerPool(config.MaxAgents)
	}
	// Initialize invariant checker if enabled (threshold from compliance level)
	if config.InvariantCheckEnabled && invariantThreshold > 0 {
		d.InvariantChecker = NewInvariantChecker(invariantThreshold, config.MaxAgents)
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
	spawnTracker := NewSpawnedIssueTracker()
	if cachePath := DefaultSpawnCachePath(); cachePath != "" {
		spawnTracker = NewSpawnedIssueTrackerWithFile(cachePath)
	}
	d := &Daemon{
		Config:              config,
		Pool:                pool,
		Scheduler:           NewSchedulerFromConfig(config),
		SpawnedIssues:       spawnTracker,
		resumeAttempts:      make(map[string]time.Time),
		VerificationTracker: NewVerificationTracker(config.VerificationPauseThreshold),
		Issues:              &defaultIssueQuerier{},
		Spawner:             &defaultSpawner{},
		Completions:         &defaultCompletionFinder{},
		Reflector:           &defaultReflector{},
		ModelDrift:          modeldrift.NewDefaultStore(),
		KnowledgeHealth:     &defaultKnowledgeHealthService{},
		AgreementCheck:      &defaultAgreementCheckService{},
		Cleaner:             &defaultSessionCleaner{},
		ActiveCounter:       &defaultActiveCounter{},
		Agents:              &defaultAgentDiscoverer{},
		StatusUpdater:       &defaultIssueUpdater{},
	}
	// Initialize rate limiter if MaxSpawnsPerHour is set
	if config.MaxSpawnsPerHour > 0 {
		d.RateLimiter = NewRateLimiter(config.MaxSpawnsPerHour)
	}
	return d
}

// Issue selection methods are in issue_selection.go:
//   resolveIssueQuerier, issueMatchesLabel, NextIssue, NextIssueExcluding, expandTriageReadyEpics

// Once processes a single issue from the queue and returns.
// If a worker pool is configured, it acquires a slot before spawning.
// Note: The slot is NOT automatically released when the agent completes.
// Use OnceWithSlot() for explicit slot management, or ReleaseSlot() manually.
func (d *Daemon) Once() (*OnceResult, error) {
	return d.OnceExcluding(nil)
}

// OnceExcluding processes a single issue from the queue, excluding skipped issues.
// Structured as an OODA loop: Sense → Orient → Decide → Act.
// See ooda.go for the individual phase implementations.
//
// The skip map should contain issue IDs that should be skipped this cycle.
// If a worker pool is configured, it acquires a slot before spawning.
// If a rate limiter is configured, it checks the hourly limit before spawning.
func (d *Daemon) OnceExcluding(skip map[string]bool) (*OnceResult, error) {
	// SENSE: gather raw signals (gates + issue queue)
	sense := d.Sense(skip)

	// ORIENT: prioritize and contextualize
	orient := d.Orient(sense)

	// DECIDE: select issue, infer skill/model, apply routing
	decision := d.Decide(orient, skip)

	// ACT: execute the spawn decision
	return d.Act(decision)
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

	// Infer model from skill type
	inferredModel := InferModelFromSkill(skill)

	return d.spawnIssue(issue, skill, inferredModel)
}

// Spawn execution methods are in spawn_execution.go:
//   spawnIssue, buildSpawnPipeline, issueUpdaterFunc, ReleaseSlot

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
