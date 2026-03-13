// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
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
	GroupConfig *group.Config
	// KBProjects maps project name -> absolute path for group membership resolution.
	// Built from ProjectRegistry at daemon startup, used by GroupConfig.AccountForProjectDir.
	KBProjects map[string]string

	// FocusGoal is the current focus goal text (from ~/.orch/focus.json).
	// When set, issues from projects matching this goal get a priority boost.
	FocusGoal string
	// FocusBoostAmount is how many priority levels to boost (default: 1).
	// E.g., with boost=1, a P2 issue becomes effectively P1.
	FocusBoostAmount int
	// ProjectDirNames maps project prefixes to directory basenames for focus matching.
	// Built from ProjectRegistry at daemon startup.
	ProjectDirNames map[string]string
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
	d := &Daemon{
		Config:                   config,
		Scheduler:                NewSchedulerFromConfig(config),
		SpawnedIssues:            spawnTracker,
		resumeAttempts:           make(map[string]time.Time),
		VerificationTracker:      NewVerificationTracker(config.VerificationPauseThreshold),
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
	// Initialize invariant checker if enabled
	if config.InvariantCheckEnabled && config.InvariantViolationThreshold > 0 {
		d.InvariantChecker = NewInvariantChecker(config.InvariantViolationThreshold, config.MaxAgents)
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
// This allows the daemon to skip issues that failed to spawn (e.g., due to failure
// report gate) and continue processing other issues in the queue.
//
// The skip map should contain issue IDs that should be skipped this cycle.
// If a worker pool is configured, it acquires a slot before spawning.
// If a rate limiter is configured, it checks the hourly limit before spawning.
func (d *Daemon) OnceExcluding(skip map[string]bool) (*OnceResult, error) {
	// Check verification pause BEFORE any other checks (including rate limit).
	// This enforces the verifiability-first constraint: daemon pauses after N
	// auto-completions without human verification.
	if d.VerificationTracker != nil && d.VerificationTracker.IsPaused() {
		status := d.VerificationTracker.Status()
		return &OnceResult{
			Processed: false,
			Message: fmt.Sprintf("Paused for human verification (%d/%d auto-completions). Resume with: orch daemon resume",
				status.CompletionsSinceVerification, status.Threshold),
		}, nil
	}

	// Check completion processing health BEFORE spawning.
	// If completion processing has failed 3+ times consecutively, pause spawning.
	// This prevents orphaning completed agents when completion processing is broken.
	const completionFailureThreshold = 3
	if d.CompletionFailureTracker != nil {
		consecutiveFailures := d.CompletionFailureTracker.ConsecutiveFailures()
		if consecutiveFailures >= completionFailureThreshold {
			lastFailureTime, lastFailureReason := d.CompletionFailureTracker.LastFailure()
			return &OnceResult{
				Processed: false,
				Message: fmt.Sprintf("Paused: completion processing has failed %d consecutive times (last: %v at %s). Fix completion processing before spawning more agents.",
					consecutiveFailures, lastFailureReason, lastFailureTime.Format("15:04:05")),
			}, nil
		}
	}

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

	skill, err := InferSkillFromIssue(issue)
	if err != nil {
		return nil, fmt.Errorf("failed to infer skill: %w", err)
	}

	// Infer model from skill type
	inferredModel := InferModelFromSkill(skill)

	// Check for critical hotspots requiring pre-extraction.
	// If the issue targets a file >800 lines, spawn an extraction agent first
	// and block the original issue until extraction completes.
	extractionSpawned := false
	originalIssueID := ""
	if d.HotspotChecker != nil {
		extraction := CheckExtractionNeeded(issue, d.HotspotChecker)
		if extraction != nil && extraction.Needed {
			extractionID, err := d.resolveIssueQuerier().CreateExtractionIssue(extraction.ExtractionTask, issue.ID)
			if err != nil {
				// Extraction gate is non-negotiable: if setup fails, skip the issue
				// and return error (fail-fast). Do not proceed with normal spawn.
				if d.Config.Verbose {
					fmt.Printf("  Extraction setup failed for %s: %v (skipping issue)\n", issue.ID, err)
				}
				return &OnceResult{
					Processed: false,
					Message:   fmt.Sprintf("Extraction setup failed for %s: %v (issue skipped, will retry on next poll)", issue.ID, err),
				}, nil
			}

			if d.Config.Verbose {
				fmt.Printf("  Auto-extraction: created %s blocking %s for %s (%d lines)\n",
					extractionID, issue.ID, extraction.CriticalFile, extraction.Hotspot.Score)
			}
			// Replace issue and skill with extraction work.
			// The original issue now has a blocking dependency and will be
			// picked up on a future poll cycle after extraction completes.
			originalIssueID = issue.ID
			issue = &Issue{
				ID:        extractionID,
				Title:     extraction.ExtractionTask,
				IssueType: "task",
				Priority:  1,
			}
			skill = "feature-impl"
			inferredModel = InferModelFromSkill(skill)
			extractionSpawned = true
		}
	}

	// Layer 2: Architect escalation for hotspot areas.
	// If the issue targets a hotspot area (any type, not just bloat-size >800)
	// and the skill is an implementation skill (feature-impl, systematic-debugging),
	// escalate to architect for architectural review before implementation.
	// This only applies when extraction didn't already happen (extraction handles the most critical case).
	architectEscalated := false
	var escalationDetail *ArchitectEscalation
	if !extractionSpawned && d.HotspotChecker != nil {
		escalationDetail = CheckArchitectEscalation(issue, skill, d.HotspotChecker, d.PriorArchitectFinder)
		if escalationDetail != nil && escalationDetail.Escalated {
			if d.Config.Verbose {
				fmt.Printf("  Architect escalation: %s targets hotspot %s (%s, score=%d)\n",
					issue.ID, escalationDetail.HotspotFile, escalationDetail.HotspotType, escalationDetail.HotspotScore)
			}
			skill = "architect"
			inferredModel = InferModelFromSkill(skill)
			architectEscalated = true
		}
	}

	// Session-level dedup: Check if there's an existing OpenCode session for this issue.
	// This prevents duplicate spawns when:
	// 1. SpawnedIssueTracker TTL expires (5min/6h) but agent is still running
	// 2. Status update to "in_progress" failed silently
	// 3. Multiple daemon instances try to spawn the same issue
	result, _, err := d.spawnIssue(issue, skill, inferredModel)
	if result != nil {
		if extractionSpawned {
			result.ExtractionSpawned = true
			result.OriginalIssueID = originalIssueID
		}
		if architectEscalated {
			result.ArchitectEscalated = true
		}
		if escalationDetail != nil {
			result.ArchitectEscalationDetail = escalationDetail
		}
	}
	return result, err
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
