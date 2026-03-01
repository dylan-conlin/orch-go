// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
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

	// lastReflect tracks when reflection was last run for periodic reflection.
	lastReflect time.Time

	// lastModelDriftReflect tracks when model drift reflection was last run.
	lastModelDriftReflect time.Time

	// lastKnowledgeHealth tracks when knowledge health was last checked.
	lastKnowledgeHealth time.Time

	// lastCleanup tracks when session cleanup was last run for periodic cleanup.
	lastCleanup time.Time

	// lastRecovery tracks when recovery was last run for periodic recovery.
	lastRecovery time.Time

	// lastOrphanDetection tracks when orphan detection was last run.
	lastOrphanDetection time.Time

	// lastPhaseTimeout tracks when phase timeout detection was last run.
	lastPhaseTimeout time.Time

	// resumeAttempts tracks when we last attempted to resume each agent (by beads ID).
	// Prevents infinite resume loops by rate-limiting to 1 attempt per hour per agent.
	resumeAttempts map[string]time.Time

	// VerificationTracker tracks completions since last human verification and manages
	// pause state when threshold is reached. This enforces the verifiability-first
	// constraint by pausing autonomous operation after N completions without human review.
	VerificationTracker *VerificationTracker

	// SpawnFailureTracker tracks spawn failures to surface them in health metrics.
	// This prevents silent failure when UpdateBeadsStatus persistently fails.
	SpawnFailureTracker *SpawnFailureTracker

	// CompletionFailureTracker tracks completion processing failures to surface them in health metrics.
	// This prevents silent failure when CompletionOnce persistently fails (e.g., beads database issues).
	CompletionFailureTracker *CompletionFailureTracker

	// Issues queries beads issues for the spawn pipeline.
	Issues IssueQuerier
	// ProjectRegistry resolves issue ID prefixes to project directories.
	// When set, the daemon passes --workdir to orch work for cross-project issues.
	ProjectRegistry *ProjectRegistry

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
	// Cleaner cleans up stale sessions.
	Cleaner SessionCleaner
	// ActiveCounter counts active agents for pool reconciliation.
	ActiveCounter ActiveCounter
	// Agents discovers agents for orphan detection and recovery.
	Agents AgentDiscoverer
	// StatusUpdater updates beads issue status.
	StatusUpdater IssueUpdater
}

// New creates a new Daemon instance with default configuration.
func New() *Daemon {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new Daemon instance with the given configuration.
func NewWithConfig(config Config) *Daemon {
	d := &Daemon{
		Config:                  config,
		SpawnedIssues:           NewSpawnedIssueTracker(),
		resumeAttempts:          make(map[string]time.Time),
		VerificationTracker:     NewVerificationTracker(config.VerificationPauseThreshold),
		SpawnFailureTracker:     NewSpawnFailureTracker(),
		CompletionFailureTracker: NewCompletionFailureTracker(),
		Issues:                  &defaultIssueQuerier{},
		Spawner:                 &defaultSpawner{},
		Completions:             &defaultCompletionFinder{},
		Reflector:               &defaultReflector{},
		ModelDrift:              modeldrift.NewDefaultStore(),
		KnowledgeHealth:         &defaultKnowledgeHealthService{},
		Cleaner:                 &defaultSessionCleaner{},
		ActiveCounter:           &defaultActiveCounter{},
		Agents:                  &defaultAgentDiscoverer{},
		StatusUpdater:           &defaultIssueUpdater{},
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
		Config:              config,
		Pool:                pool,
		SpawnedIssues:       NewSpawnedIssueTracker(),
		resumeAttempts:      make(map[string]time.Time),
		VerificationTracker: NewVerificationTracker(config.VerificationPauseThreshold),
		SpawnFailureTracker: NewSpawnFailureTracker(),
		Issues:              &defaultIssueQuerier{},
		Spawner:             &defaultSpawner{},
		Completions:         &defaultCompletionFinder{},
		Reflector:           &defaultReflector{},
		ModelDrift:          modeldrift.NewDefaultStore(),
		KnowledgeHealth:     &defaultKnowledgeHealthService{},
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

// resolveIssueQuerier returns the effective IssueQuerier.
// If Issues is set, returns it directly.
// If only ProjectRegistry is set (no custom Issues), wraps it into a defaultIssueQuerier.
func (d *Daemon) resolveIssueQuerier() IssueQuerier {
	if d.Issues != nil {
		// If it's the default querier, update its registry pointer lazily
		if dq, ok := d.Issues.(*defaultIssueQuerier); ok {
			dq.registry = d.ProjectRegistry
		}
		return d.Issues
	}
	return &defaultIssueQuerier{registry: d.ProjectRegistry}
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
	issues, err := d.resolveIssueQuerier().ListReadyIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	if d.Config.Verbose {
		fmt.Printf("  DEBUG: Found %d open issues\n", len(issues))
	}

	// Expand triage:ready epics by including their children.
	// This allows "label the epic" to mean "process the entire epic".
	issues, epicChildIDs, err := d.expandTriageReadyEpics(issues)
	if err != nil {
		return nil, fmt.Errorf("failed to expand epics: %w", err)
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
		// Skip issues that have been recently spawned but status not yet updated.
		// This prevents the race condition where the daemon spawns duplicate agents
		// because beads status update hasn't propagated yet.
		if d.SpawnedIssues != nil && d.SpawnedIssues.IsSpawned(issue.ID) {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (recently spawned, awaiting status update)\n", issue.ID)
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
func (d *Daemon) expandTriageReadyEpics(issues []Issue) ([]Issue, map[string]bool, error) {
	epicChildIDs := make(map[string]bool)

	// If no label filter is set, no expansion needed
	if d.Config.Label == "" {
		return issues, epicChildIDs, nil
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
		return issues, epicChildIDs, nil
	}

	// Expand each epic by fetching its children
	querier := d.resolveIssueQuerier()
	for _, epicID := range epicsToExpand {
		children, err := querier.ListEpicChildren(epicID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list children of epic %s: %w", epicID, err)
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

	return issues, epicChildIDs, nil
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
	// If the issue targets a file >1500 lines, spawn an extraction agent first
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
	// If the issue targets a hotspot area (any type, not just bloat-size >1500)
	// and the skill is an implementation skill (feature-impl, systematic-debugging),
	// escalate to architect for architectural review before implementation.
	// This only applies when extraction didn't already happen (extraction handles the most critical case).
	architectEscalated := false
	if !extractionSpawned && d.HotspotChecker != nil {
		escalation := CheckArchitectEscalation(issue, skill, d.HotspotChecker, d.PriorArchitectFinder)
		if escalation != nil {
			if d.Config.Verbose {
				fmt.Printf("  Architect escalation: %s targets hotspot %s (%s, score=%d)\n",
					issue.ID, escalation.HotspotFile, escalation.HotspotType, escalation.HotspotScore)
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

func (d *Daemon) spawnIssue(issue *Issue, skill string, inferredModel string) (*OnceResult, *Slot, error) {
	// Session-level dedup: Check if there's an existing OpenCode session OR tmux window.
	// This prevents duplicate spawns when:
	// 1. SpawnedIssueTracker TTL expires (5min/6h) but agent is still running
	// 2. Status update to "in_progress" failed silently
	// 3. Multiple daemon instances try to spawn the same issue
	// 4. Claude CLI backend: agent runs in tmux without OpenCode session
	if HasExistingSessionForBeadsID(issue.ID) {
		if d.Config.Verbose {
			fmt.Printf("  DEBUG: Skipping %s (existing session or tmux window found)\n", issue.ID)
		}
		return &OnceResult{
			Processed: false,
			Issue:     issue,
			Skill:     skill,
			Model:     inferredModel,
			Message:   fmt.Sprintf("Existing session/window found for %s - skipping to prevent duplicate", issue.ID),
		}, nil, nil
	}

	// CONTENT-AWARE DEDUP: Check if another issue with the same title is already in_progress.
	// This catches the case where multiple beads issues are created with identical content
	// (different IDs, same title). Without this, the daemon would spawn each one because
	// all other dedup layers are keyed on issue ID.
	//
	// Layer 1 (in-memory): Check SpawnedIssueTracker title index for recently-spawned titles.
	// This is fast and catches duplicates within the same daemon instance.
	if d.SpawnedIssues != nil {
		if spawned, dupID := d.SpawnedIssues.IsTitleSpawned(issue.Title); spawned && dupID != issue.ID {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (title matches recently spawned %s)\n", issue.ID, dupID)
			}
			return &OnceResult{
				Processed: false,
				Issue:     issue,
				Skill:     skill,
				Model:     inferredModel,
				Message:   fmt.Sprintf("Skipping %s - title matches recently spawned %s", issue.ID, dupID),
			}, nil, nil
		}
	}
	// Layer 2 (persistent): Check beads database for in_progress issues with same title.
	// This survives daemon restarts and catches duplicates across daemon instances.
	if dup := FindInProgressByTitle(issue.Title); dup != nil && dup.ID != issue.ID {
		if d.Config.Verbose {
			fmt.Printf("  DEBUG: Skipping %s (content duplicate of in_progress %s)\n", issue.ID, dup.ID)
		}
		return &OnceResult{
			Processed: false,
			Issue:     issue,
			Skill:     skill,
			Model:     inferredModel,
			Message:   fmt.Sprintf("Skipping %s - duplicate of in_progress issue %s with same title", issue.ID, dup.ID),
		}, nil, nil
	}

	// FRESH STATUS CHECK: Re-fetch the issue's current status from beads.
	// This catches the TOCTOU race where another daemon process marked this issue
	// as in_progress between our ListReadyIssues() call and now. Without this check,
	// UpdateBeadsStatus is idempotent (setting in_progress when already in_progress
	// succeeds), so two concurrent daemon processes can both spawn the same issue.
	// This was the root cause of the Feb 15 2026 orch-go-09cc duplicate spawn incident.
	if d.Issues != nil {
		var currentStatus string
		var statusErr error
		if issue.ProjectDir != "" {
			// Cross-project issue: check status in the target project
			currentStatus, statusErr = GetBeadsIssueStatusForProject(issue.ID, issue.ProjectDir)
		} else {
			currentStatus, statusErr = d.Issues.GetIssueStatus(issue.ID)
		}
		if statusErr == nil {
			if currentStatus != "open" {
				if d.Config.Verbose {
					fmt.Printf("  DEBUG: Skipping %s (status is %s, expected open)\n", issue.ID, currentStatus)
				}
				return &OnceResult{
					Processed: false,
					Issue:     issue,
					Skill:     skill,
					Model:     inferredModel,
					Message:   fmt.Sprintf("Issue %s is already %s - skipping to prevent duplicate", issue.ID, currentStatus),
				}, nil, nil
			}
		}
		// On error, continue to spawn (fail-open to avoid blocking work).
		// The UpdateBeadsStatus call below provides the persistent dedup gate.
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
				Model:     inferredModel,
				Message:   "At capacity - no slots available",
			}, nil, nil
		}
		slot.BeadsID = issue.ID
	}

	// PRIMARY DEDUP: Update beads status to in_progress BEFORE spawning.
	// This makes the beads database (source of truth) immediately reflect that
	// the issue is being worked on. This prevents duplicate spawns even if:
	// - SpawnedIssueTracker TTL expires (6 hours)
	// - Daemon restarts (in-memory tracker lost)
	// - Multiple daemon instances poll simultaneously
	// The status update happens synchronously before spawn to ensure immediate visibility.
	//
	// CRITICAL: If status update fails, we MUST NOT spawn. Spawning without persistent
	// tracking leads to duplicate spawns when SpawnedIssueTracker TTL expires or daemon restarts.
	// Fail-fast here prevents the Feb 14 2026 incident where 10 duplicate spawns occurred
	// because UpdateBeadsStatus was failing silently.
	// Resolve status updater: for cross-project issues with the default updater,
	// use the project-specific variant.
	statusUpdater := d.StatusUpdater
	if statusUpdater == nil {
		statusUpdater = &defaultIssueUpdater{}
	}
	if issue.ProjectDir != "" {
		if _, isDefault := statusUpdater.(*defaultIssueUpdater); isDefault {
			statusUpdater = issueUpdaterFunc(func(beadsID, status string) error {
				return UpdateBeadsStatusForProject(beadsID, status, issue.ProjectDir)
			})
		}
	}
	if err := statusUpdater.UpdateStatus(issue.ID, "in_progress"); err != nil {
		// Track failure for health card visibility
		if d.SpawnFailureTracker != nil {
			d.SpawnFailureTracker.RecordFailure(fmt.Sprintf("UpdateBeadsStatus failed: %v", err))
		}
		// Release slot on status update failure
		if d.Pool != nil && slot != nil {
			d.Pool.Release(slot)
		}
		return &OnceResult{
			Processed: false,
			Issue:     issue,
			Skill:     skill,
			Model:     inferredModel,
			Error:     fmt.Errorf("failed to mark issue as in_progress: %w", err),
			Message:   fmt.Sprintf("Failed to update beads status for %s - skipping spawn to prevent duplicates", issue.ID),
		}, nil, nil
	}

	// SECONDARY DEDUP: Mark issue as spawned in memory (with title for content dedup).
	// This catches the race window between beads update and subprocess spawn completion.
	// Title tracking prevents duplicate content spawns within the same daemon instance.
	if d.SpawnedIssues != nil {
		d.SpawnedIssues.MarkSpawnedWithTitle(issue.ID, issue.Title)
	}

	// Use project directory from issue (set during multi-project polling)
	workdir := issue.ProjectDir

	// Spawn the work with inferred model and optional workdir
	spawner := d.Spawner
	if spawner == nil {
		spawner = &defaultSpawner{}
	}
	if err := spawner.SpawnWork(issue.ID, inferredModel, workdir); err != nil {
		// Track spawn failure for health card visibility
		if d.SpawnFailureTracker != nil {
			d.SpawnFailureTracker.RecordFailure(fmt.Sprintf("Spawn failed: %v", err))
		}
		// On spawn failure, roll back beads status to open
		// CRITICAL: If rollback fails, return immediately. Rollback failure indicates
		// database issues (connectivity, beads daemon unavailability, etc.) that need
		// immediate attention. Continuing would leave the issue in an inconsistent state
		// (marked in_progress but spawn failed), blocking future spawns and orphaning the issue.
		if rollbackErr := UpdateBeadsStatusForProject(issue.ID, "open", issue.ProjectDir); rollbackErr != nil {
			// Log as ERROR (not warning) - this is a critical failure
			fmt.Fprintf(os.Stderr, "ERROR: Failed to rollback status for %s after spawn failure: %v\n", issue.ID, rollbackErr)
			// Track rollback failure for health metrics
			if d.SpawnFailureTracker != nil {
				d.SpawnFailureTracker.RecordFailure(fmt.Sprintf("Rollback failed for %s: %v", issue.ID, rollbackErr))
			}
			// Return rollback error immediately - don't continue cleanup
			// The rollback error is more critical than the spawn error
			return &OnceResult{
				Processed: false,
				Issue:     issue,
				Skill:     skill,
				Model:     inferredModel,
				Error:     fmt.Errorf("spawn failed (%w) and rollback failed: %v - issue may be orphaned", err, rollbackErr),
				Message:   fmt.Sprintf("CRITICAL: spawn failed and status rollback failed for %s - issue may be orphaned", issue.ID),
			}, nil, nil
		}
		// Unmark from tracker so issue can be retried
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
			Model:     inferredModel,
			Error:     err,
			Message:   fmt.Sprintf("Failed to spawn: %v", err),
		}, nil, nil
	}

	// Record successful spawn for rate limiting
	if d.RateLimiter != nil {
		d.RateLimiter.RecordSpawn()
	}

	// Record successful spawn for failure tracking
	if d.SpawnFailureTracker != nil {
		d.SpawnFailureTracker.RecordSuccess()
	}

	return &OnceResult{
		Processed: true,
		Issue:     issue,
		Skill:     skill,
		Model:     inferredModel,
		Message:   fmt.Sprintf("Spawned work on %s", issue.ID),
	}, slot, nil
}

// issueUpdaterFunc adapts a function to the IssueUpdater interface.
// Used for cross-project status updates that need a different target directory.
type issueUpdaterFunc func(beadsID, status string) error

func (f issueUpdaterFunc) UpdateStatus(beadsID, status string) error {
	return f(beadsID, status)
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
