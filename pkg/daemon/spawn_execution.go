package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// spawnIssue executes the spawn pipeline for a single issue: dedup gates,
// pool slot acquisition, beads status update, and the actual spawn call.
// Contains rollback logic for spawn failures and auto-completion for
// orphaned "Phase: Complete but not closed" issues.
func (d *Daemon) spawnIssue(issue *Issue, skill string, inferredModel string) (*OnceResult, *Slot, error) {
	// Run the dedup pipeline: all pre-spawn gates and advisory checks.
	pipeline := d.buildSpawnPipeline()
	pipelineResult := pipeline.Run(issue)

	if !pipelineResult.Allowed {
		if d.Config.Verbose {
			fmt.Printf("  DEBUG: Skipping %s (rejected by %s: %s)\n", issue.ID, pipelineResult.RejectedBy, pipelineResult.RejectionMessage)
		}
		return &OnceResult{
			Processed: false,
			Issue:     issue,
			Skill:     skill,
			Model:     inferredModel,
			Message:   fmt.Sprintf("%s - skipping to prevent duplicate", pipelineResult.RejectionMessage),
		}, nil, nil
	}

	// Log advisory warnings
	for _, advisory := range pipelineResult.Advisories {
		if d.Config.Verbose {
			fmt.Printf("  ADVISORY [%s]: %s\n", advisory.Name, advisory.Warning)
		}
	}

	// If pool is configured, acquire a slot first
	var slot *Slot
	if d.Pool != nil {
		slot = d.Pool.TryAcquire()
		if slot == nil {
			logDaemonGateDecision("concurrency", "block", skill, issue.ID,
				fmt.Sprintf("At capacity: %d/%d slots occupied", d.Pool.Active(), d.Pool.MaxWorkers()))
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
	// Resolve status updater: use project-specific variant when project dir
	// is known (cross-project issues or local-project with registry).
	// This avoids FindSocketPath("") which breaks when CWD is wrong (launchd).
	statusUpdater := d.StatusUpdater
	if statusUpdater == nil {
		statusUpdater = &defaultIssueUpdater{}
	}
	statusProjectDir := issue.ProjectDir
	if statusProjectDir == "" && d.ProjectRegistry != nil {
		statusProjectDir = d.ProjectRegistry.CurrentDir()
	}
	if statusProjectDir != "" {
		if _, isDefault := statusUpdater.(*defaultIssueUpdater); isDefault {
			statusUpdater = issueUpdaterFunc(func(beadsID, status string) error {
				return UpdateBeadsStatusForProject(beadsID, status, statusProjectDir)
			})
		}
	}
	if err := statusUpdater.UpdateStatus(issue.ID, "in_progress"); err != nil {
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

	// Resolve account from project group (if groups are configured)
	account := d.resolveAccountForProject(workdir)

	// Spawn the work with inferred model and optional workdir
	spawner := d.Spawner
	if spawner == nil {
		spawner = &defaultSpawner{}
	}
	if err := spawner.SpawnWork(issue.ID, skill, inferredModel, workdir, account); err != nil {
		// Check if this is a "Phase: Complete but not closed" error.
		// This happens with cross-repo issues where the agent completed work
		// but the issue was never closed (e.g., orphaned cross-project issues).
		// Instead of rolling back to "open" and retrying every cycle, attempt
		// auto-completion to close the issue permanently.
		if strings.Contains(err.Error(), "Phase: Complete but is not closed") {
			if d.AutoCompleter != nil {
				completeErr := d.AutoCompleter.Complete(issue.ID, workdir)
				if completeErr == nil {
					// Auto-completion succeeded — issue is now closed.
					// Clean up spawn tracking state.
					if d.SpawnedIssues != nil {
						d.SpawnedIssues.Unmark(issue.ID)
					}
					if d.Pool != nil && slot != nil {
						d.Pool.Release(slot)
					}
					return &OnceResult{
						Processed: false,
						Issue:     issue,
						Skill:     skill,
						Model:     inferredModel,
						Message:   fmt.Sprintf("Auto-completed %s (Phase: Complete but not closed)", issue.ID),
					}, nil, nil
				}
				// Auto-completion failed — fall through to normal error handling
				fmt.Fprintf(os.Stderr, "Warning: auto-complete failed for Phase:Complete issue %s, skipping: %v\n", issue.ID, completeErr)
			}
		}

		// On spawn failure, roll back beads status to open
		// CRITICAL: If rollback fails, return immediately. Rollback failure indicates
		// database issues (connectivity, beads daemon unavailability, etc.) that need
		// immediate attention. Continuing would leave the issue in an inconsistent state
		// (marked in_progress but spawn failed), blocking future spawns and orphaning the issue.
		if rollbackErr := UpdateBeadsStatusForProject(issue.ID, "open", statusProjectDir); rollbackErr != nil {
			// Log as ERROR (not warning) - this is a critical failure
			fmt.Fprintf(os.Stderr, "ERROR: Failed to rollback status for %s after spawn failure: %v\n", issue.ID, rollbackErr)
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

	// POST-SPAWN VERIFICATION: Verify workspace was actually created.
	// Catches phantom spawns where orch work exits 0 but no workspace directory
	// exists (e.g., silent failures in workspace setup, cross-project path issues).
	// Without this check, the daemon logs "Spawned" and marks the issue in_progress
	// but no agent is actually running — creating a phantom agent.
	if d.WorkspaceVerifier != nil {
		verifyDir := workdir
		if verifyDir == "" && d.ProjectRegistry != nil {
			verifyDir = d.ProjectRegistry.CurrentDir()
		}
		if verifyDir != "" && !d.WorkspaceVerifier.Exists(issue.ID, verifyDir) {
			// Phantom spawn detected — rollback to allow retry.
			// Use the resolved statusUpdater (same one used for initial in_progress update)
			// so cross-project and mock status updaters are used consistently.
			if rollbackErr := statusUpdater.UpdateStatus(issue.ID, "open"); rollbackErr != nil {
				fmt.Fprintf(os.Stderr, "ERROR: Failed to rollback status for %s after phantom spawn: %v\n", issue.ID, rollbackErr)
				return &OnceResult{
					Processed: false,
					Issue:     issue,
					Skill:     skill,
					Model:     inferredModel,
					Error:     fmt.Errorf("phantom spawn (no workspace) and rollback failed: %v", rollbackErr),
					Message:   fmt.Sprintf("CRITICAL: phantom spawn and rollback failed for %s", issue.ID),
				}, nil, nil
			}
			if d.SpawnedIssues != nil {
				d.SpawnedIssues.Unmark(issue.ID)
			}
			if d.Pool != nil && slot != nil {
				d.Pool.Release(slot)
			}
			return &OnceResult{
				Processed: false,
				Issue:     issue,
				Skill:     skill,
				Model:     inferredModel,
				Error:     fmt.Errorf("phantom spawn: orch work exited 0 but no workspace found for %s", issue.ID),
				Message:   fmt.Sprintf("Phantom spawn: no workspace created for %s — rolled back to open", issue.ID),
			}, nil, nil
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
		Model:     inferredModel,
		Message:   fmt.Sprintf("Spawned work on %s", issue.ID),
	}, slot, nil
}

// buildSpawnPipeline constructs the dedup pipeline from the daemon's current state.
// This replaces the inline 6-layer dedup gauntlet that was previously in spawnIssue().
// Gate order matches the original execution order for behavioral equivalence.
func (d *Daemon) buildSpawnPipeline() *SpawnPipeline {
	// Build fresh status gate with appropriate status functions
	freshStatusGate := &FreshStatusGate{}
	if d.Issues != nil {
		freshStatusGate.GetStatusFunc = d.Issues.GetIssueStatus
		freshStatusGate.GetStatusForProjectFunc = GetBeadsIssueStatusForProject
	}

	// Build title dedup gate with project-aware lookup when registry is available.
	// This avoids FindSocketPath("") which returns wrong socket from launchd.
	titleDedupGate := &TitleDedupBeadsGate{}
	if d.ProjectRegistry != nil {
		currentDir := d.ProjectRegistry.CurrentDir()
		titleDedupGate.FindFunc = func(title string) *Issue {
			return FindInProgressByTitleForProject(title, currentDir)
		}
	}

	// Build commit dedup gate: checks if referenced beads IDs have git commits.
	// Uses d.CommitChecker when set (production); nil skips the gate (tests).
	// GetIssueTypeFunc enables cross-type reference filtering: a task referencing
	// a completed investigation is follow-up work, not duplication.
	commitDedupGate := &CommitDedupGate{
		HasCommitsFunc:    d.CommitChecker,
		GetIssueTypeFunc:  GetBeadsIssueType,
		GetIssueTitleFunc: GetBeadsIssueTitle,
	}

	// Build gate list with required gates
	gates := []SpawnGate{
		&SpawnTrackerGate{Tracker: d.SpawnedIssues},     // L1: Spawn cache (ID)
		&SessionDedupGate{},                              // L2: Session/tmux existence
		&TitleDedupMemoryGate{Tracker: d.SpawnedIssues}, // L3: Title dedup (in-memory)
		titleDedupGate,                                   // L4: Title dedup (beads DB)
		freshStatusGate,                                  // L5: Fresh status re-check
		commitDedupGate,                                  // L6: Referenced issue commit check
	}

	// Add keyword dedup gate when spawn tracker is available
	if d.SpawnedIssues != nil {
		gates = append(gates, &KeywordDedupGate{ // L7: Keyword overlap dedup
			FindOverlapFunc: func(title, selfID string) (bool, string) {
				return FindKeywordOverlap(d.SpawnedIssues, title, selfID)
			},
		})
	}

	return &SpawnPipeline{
		Gates: gates,
		AdvisoryChecks: []AdvisoryCheck{
			&SpawnCountAdvisory{Tracker: d.SpawnedIssues, Threshold: 3},
		},
		Verbose: d.Config.Verbose,
	}
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

// resolveAccountForProject determines the account to use for a project directory
// based on group configuration. Returns empty string (use default account) when:
// - No group config is loaded
// - Project directory is empty (local project)
// - Project is not in any group
// - Matching group has no account set
func (d *Daemon) resolveAccountForProject(projectDir string) string {
	if d.GroupConfig == nil || projectDir == "" {
		return ""
	}
	return d.GroupConfig.AccountForProjectDir(projectDir, d.KBProjects)
}

// BuildKBProjectsMap builds a name->path map from the ProjectRegistry,
// suitable for group membership resolution. Uses filepath.Base(dir) as the name.
func BuildKBProjectsMap(registry *ProjectRegistry) map[string]string {
	if registry == nil {
		return nil
	}
	m := make(map[string]string)
	for _, proj := range registry.Projects() {
		name := filepath.Base(proj.Dir)
		m[name] = proj.Dir
	}
	return m
}
