// Package daemon provides autonomous overnight processing capabilities.
// This file wires pkg/agent lifecycle orphan detection and recovery into the daemon loop.
// It uses the LifecycleManager to detect orphaned agents (in_progress with no live execution)
// and applies ForceComplete or ForceAbandon transitions based on their last phase.
package daemon

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
)

// LandedArtifactFlagger is an optional interface that LifecycleManager implementations
// can support to flag orphaned agents that crashed with committed work.
// This adds a beads comment and label so the agent shows up in `orch review`.
type LandedArtifactFlagger interface {
	FlagLandedArtifacts(agent agent.AgentRef) error
}

// LifecycleOrphanRecoveryResult contains the results of lifecycle-based orphan recovery.
type LifecycleOrphanRecoveryResult struct {
	// Scanned is the number of agents examined.
	Scanned int

	// ForceCompleted is the number of orphans that were force-completed (Phase: Complete).
	ForceCompleted int

	// ForceAbandoned is the number of orphans that were force-abandoned (reset to open).
	ForceAbandoned int

	// FlaggedForReview is the number of orphans flagged as crashed-with-artifacts.
	FlaggedForReview int

	// Skipped is the number of orphans skipped (e.g., due to errors).
	Skipped int

	// Error is set if detection itself failed.
	Error error

	// Message is a human-readable summary.
	Message string

	// Elapsed is how long the recovery took.
	Elapsed time.Duration
}

// Snapshot converts a LifecycleOrphanRecoveryResult to a dashboard-ready snapshot.
func (r *LifecycleOrphanRecoveryResult) Snapshot() OrphanDetectionSnapshot {
	return OrphanDetectionSnapshot{
		ResetCount:   r.ForceCompleted + r.ForceAbandoned,
		SkippedCount: r.Skipped + r.FlaggedForReview,
		LastCheck:    time.Now(),
	}
}

// RunLifecycleOrphanRecovery detects orphaned agents using the pkg/agent lifecycle
// and applies ForceComplete or ForceAbandon transitions.
//
// This augments the existing RunPeriodicOrphanDetection by using the orch:agent label
// for discovery (more precise than beads status alone) and properly cleaning up
// labels, assignees, and workspaces via the lifecycle manager.
//
// Orphans with Phase: Complete → ForceComplete (close the issue)
// Orphans without Phase: Complete → ForceAbandon (reset to open for respawn)
func RunLifecycleOrphanRecovery(lm agent.LifecycleManager, projectDirs []string, threshold time.Duration, verbose bool) *LifecycleOrphanRecoveryResult {
	start := time.Now()

	result, err := lm.DetectOrphans(projectDirs, threshold)
	if err != nil {
		return &LifecycleOrphanRecoveryResult{
			Error:   err,
			Message: fmt.Sprintf("Lifecycle orphan detection failed: %v", err),
			Elapsed: time.Since(start),
		}
	}

	if len(result.Orphans) == 0 {
		return &LifecycleOrphanRecoveryResult{
			Scanned: result.Scanned,
			Message: fmt.Sprintf("Lifecycle orphan scan: %d agents checked, no orphans", result.Scanned),
			Elapsed: time.Since(start),
		}
	}

	forceCompleted := 0
	forceAbandoned := 0
	flaggedForReview := 0
	skipped := 0

	for _, orphan := range result.Orphans {
		if orphan.Agent.BeadsID == "" {
			skipped++
			continue
		}

		if isPhaseCompleteStr(orphan.LastPhase) {
			// Phase: Complete orphans → close the issue
			event, err := lm.ForceComplete(orphan.Agent, "GC: orphaned agent with Phase: Complete")
			if err != nil || !event.Success {
				if verbose {
					errMsg := "unknown"
					if err != nil {
						errMsg = err.Error()
					}
					fmt.Printf("  Lifecycle orphan: failed to force-complete %s: %s\n",
						orphan.Agent.BeadsID, errMsg)
				}
				skipped++
				continue
			}
			forceCompleted++
			if verbose {
				fmt.Printf("  Lifecycle orphan: force-completed %s (Phase: Complete, stale %v)\n",
					orphan.Agent.BeadsID, orphan.StaleFor.Round(time.Minute))
			}
		} else if orphan.HasLandedArtifacts {
			// Crashed with work: agent committed artifacts but died before Phase: Complete.
			// Flag for human review instead of abandoning (which would lose context).
			if flagger, ok := lm.(LandedArtifactFlagger); ok {
				if err := flagger.FlagLandedArtifacts(orphan.Agent); err != nil {
					if verbose {
						fmt.Printf("  Lifecycle orphan: failed to flag landed artifacts for %s: %v\n",
							orphan.Agent.BeadsID, err)
					}
					skipped++
					continue
				}
			}
			flaggedForReview++
			if verbose {
				fmt.Printf("  Lifecycle orphan: flagged for review %s (crashed with artifacts, phase=%s, stale %v)\n",
					orphan.Agent.BeadsID, orphan.LastPhase, orphan.StaleFor.Round(time.Minute))
			}
		} else {
			// Non-complete orphans with no artifacts → abandon for respawn
			event, err := lm.ForceAbandon(orphan.Agent)
			if err != nil || !event.Success {
				if verbose {
					errMsg := "unknown"
					if err != nil {
						errMsg = err.Error()
					}
					fmt.Printf("  Lifecycle orphan: failed to force-abandon %s: %s\n",
						orphan.Agent.BeadsID, errMsg)
				}
				skipped++
				continue
			}
			forceAbandoned++
			if verbose {
				fmt.Printf("  Lifecycle orphan: force-abandoned %s (phase=%s, stale %v)\n",
					orphan.Agent.BeadsID, orphan.LastPhase, orphan.StaleFor.Round(time.Minute))
			}
		}
	}

	return &LifecycleOrphanRecoveryResult{
		Scanned:          result.Scanned,
		ForceCompleted:   forceCompleted,
		ForceAbandoned:   forceAbandoned,
		FlaggedForReview: flaggedForReview,
		Skipped:          skipped,
		Message: fmt.Sprintf("Lifecycle orphan recovery: %d force-completed, %d force-abandoned, %d flagged-for-review, %d skipped (of %d scanned)",
			forceCompleted, forceAbandoned, flaggedForReview, skipped, result.Scanned),
		Elapsed: time.Since(start),
	}
}

// isPhaseCompleteStr checks if a phase string indicates completion.
func isPhaseCompleteStr(phase string) bool {
	return phase == "Complete" || phase == "complete" || phase == "COMPLETE"
}
