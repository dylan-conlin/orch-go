// Package daemon provides autonomous overnight processing capabilities.
// compliance.go contains compliance signal producers — gates, enforcement,
// and verification checks that determine whether operations are allowed.
// Compliance signals are consumed by coordination logic (coordination.go).
package daemon

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// SpawnGateSignal is the result of pre-spawn compliance gates.
// These gates must pass before any issue selection or spawning.
type SpawnGateSignal struct {
	// Allowed is true if all gates pass.
	Allowed bool
	// Reason explains why spawning was blocked (empty if Allowed).
	Reason string
}

// CheckPreSpawnGates runs all compliance gates that must pass before spawning.
// Checks: verification pause, completion health, rate limit.
// Returns immediately on first failure (short-circuit).
func (d *Daemon) CheckPreSpawnGates() SpawnGateSignal {
	// Gate 1: Verification pause — daemon pauses after N auto-completions
	// without human verification. This enforces verifiability-first constraint.
	if d.VerificationTracker != nil && d.VerificationTracker.IsPaused() {
		status := d.VerificationTracker.Status()
		return SpawnGateSignal{
			Allowed: false,
			Reason: fmt.Sprintf("Paused for human verification (%d/%d auto-completions). Resume with: orch daemon resume",
				status.CompletionsSinceVerification, status.Threshold),
		}
	}

	// Gate 2: Completion health — if completion processing is broken,
	// pause spawning to prevent orphaning completed agents.
	const completionFailureThreshold = 3
	if d.CompletionFailureTracker != nil {
		consecutiveFailures := d.CompletionFailureTracker.ConsecutiveFailures()
		if consecutiveFailures >= completionFailureThreshold {
			lastFailureTime, lastFailureReason := d.CompletionFailureTracker.LastFailure()
			return SpawnGateSignal{
				Allowed: false,
				Reason: fmt.Sprintf("Paused: completion processing has failed %d consecutive times (last: %v at %s). Fix completion processing before spawning more agents.",
					consecutiveFailures, lastFailureReason, lastFailureTime.Format("15:04:05")),
			}
		}
	}

	// Gate 3: Comprehension queue — pause spawning when too much uncomprehended work exists.
	if d.ComprehensionQuerier != nil {
		allowed, count, threshold := CheckComprehensionThrottle(d.ComprehensionQuerier, d.Config.ComprehensionThreshold)
		if !allowed {
			reason := fmt.Sprintf("Comprehension queue full: %d/%d pending items. Run 'orch comprehension' to review.", count, threshold)
			logDaemonGateDecision("comprehension", "block", "", "", reason)
			return SpawnGateSignal{
				Allowed: false,
				Reason:  reason,
			}
		}
	}

	// Gate 4: Rate limit — hourly spawn cap.
	if d.RateLimiter != nil {
		canSpawn, count, msg := d.RateLimiter.CanSpawn()
		if !canSpawn {
			if d.Config.Verbose {
				fmt.Printf("  Rate limited: %s\n", msg)
			}
			reason := fmt.Sprintf("Rate limited: %d/%d spawns in the last hour", count, d.RateLimiter.MaxPerHour)
			logDaemonGateDecision("ratelimit", "block", "", "", reason)
			return SpawnGateSignal{
				Allowed: false,
				Reason:  reason,
			}
		}
	}

	return SpawnGateSignal{Allowed: true}
}

// logDaemonGateDecision emits a spawn.gate_decision event for daemon-level enforcement.
// This makes daemon gates (concurrency, ratelimit, governance) visible to harness audit,
// which only counts events in events.jsonl.
func logDaemonGateDecision(gateName, decision, skill, beadsID, reason string) {
	logger := events.NewLogger(events.DefaultLogPath())
	_ = logger.LogGateDecision(events.GateDecisionData{
		GateName: gateName,
		Decision: decision,
		Skill:    skill,
		BeadsID:  beadsID,
		Reason:   reason,
	})
}

// IssueFilterResult indicates whether an issue passes compliance filters.
type IssueFilterResult struct {
	// Passed is true if the issue passes all compliance filters.
	Passed bool
	// Reason explains why the issue was filtered (empty if Passed).
	Reason string
}

// CheckIssueCompliance checks whether a single issue passes all compliance
// filters that determine spawnability. These are enforcement checks, not
// coordination decisions (prioritization, routing happen elsewhere).
//
// Checks: skip set, spawn dedup, spawnable type, blocked status, in_progress
// status, completion labels, label match, blocking dependencies.
func (d *Daemon) CheckIssueCompliance(issue Issue, skip map[string]bool, epicChildIDs map[string]bool) IssueFilterResult {
	// Skip issues in the skip set (failed to spawn this cycle)
	if skip != nil && skip[issue.ID] {
		return IssueFilterResult{Passed: false, Reason: "failed to spawn this cycle"}
	}

	// Skip issues recently spawned but status not yet updated (dedup)
	if d.SpawnedIssues != nil && d.SpawnedIssues.IsSpawned(issue.ID) {
		return IssueFilterResult{Passed: false, Reason: "recently spawned, awaiting status update"}
	}

	// Skip issues with missing type
	if issue.IssueType == "" {
		return IssueFilterResult{Passed: false, Reason: "missing type (required for skill inference)"}
	}

	// Skip non-spawnable types
	if !IsSpawnableType(issue.IssueType) {
		// Epics with the required label get a helpful message explaining children will be processed
		if issue.IssueType == "epic" && d.issueMatchesLabel(issue) {
			return IssueFilterResult{Passed: false, Reason: "type 'epic' not spawnable (children will be processed instead)"}
		}
		return IssueFilterResult{Passed: false, Reason: fmt.Sprintf("type '%s' not spawnable (must be bug/feature/task/investigation/question)", issue.IssueType)}
	}

	// Skip blocked issues
	if issue.Status == "blocked" {
		return IssueFilterResult{Passed: false, Reason: "blocked"}
	}

	// Skip in_progress issues (already being worked on)
	if issue.Status == "in_progress" {
		return IssueFilterResult{Passed: false, Reason: "already in_progress"}
	}

	// Skip issues already processed by the completion loop
	if issue.HasAnyLabel(LabelReadyReview, LabelVerificationFailed) {
		return IssueFilterResult{Passed: false, Reason: "has daemon completion label"}
	}

	// Skip issues without required label (if filter is set)
	// Epic children are exempt — they inherit triage-ready from parent.
	if !d.issueMatchesLabel(issue) {
		if _, isEpicChild := epicChildIDs[issue.ID]; !isEpicChild {
			return IssueFilterResult{
				Passed: false,
				Reason: fmt.Sprintf("missing label %s, has %v", d.Config.Label, issue.Labels),
			}
		}
		// Epic child — proceed even without label
	}

	// Skip issues with blocking dependencies
	blockers, err := beads.CheckBlockingDependencies(issue.ID)
	if err != nil {
		// Continue checking — don't skip issue because dependency check failed
		if d.Config.Verbose {
			fmt.Printf("  DEBUG: Warning: could not check dependencies for %s: %v\n", issue.ID, err)
		}
	} else if len(blockers) > 0 {
		var blockerIDs []string
		for _, b := range blockers {
			blockerIDs = append(blockerIDs, fmt.Sprintf("%s (%s)", b.ID, b.Status))
		}
		return IssueFilterResult{
			Passed: false,
			Reason: fmt.Sprintf("blocked by dependencies: %s", strings.Join(blockerIDs, ", ")),
		}
	}

	return IssueFilterResult{Passed: true}
}

// CompletionVerifySignal contains the result of completion compliance checks:
// verification and escalation determination.
type CompletionVerifySignal struct {
	// Passed is true if verification passed and escalation allows processing.
	Passed bool
	// Verification is the raw verification result.
	Verification verify.VerificationResult
	// Escalation is the determined escalation level.
	Escalation verify.EscalationLevel
	// Error describes why compliance failed (nil if Passed).
	Error error
}

// VerifyCompletionCompliance runs compliance verification on a completed agent.
// This produces signals about whether the completion meets quality/escalation
// requirements. The routing decision (auto-complete vs label vs block) is made
// by coordination logic.
func VerifyCompletionCompliance(agent CompletedAgent, config CompletionConfig) CompletionVerifySignal {
	signal := CompletionVerifySignal{}

	// Use agent's project dir for cross-project operations, fall back to config
	effectiveProjectDir := agent.ProjectDir
	if effectiveProjectDir == "" {
		effectiveProjectDir = config.ProjectDir
	}

	// Determine tier from workspace if available
	tier := ""
	if agent.WorkspacePath != "" {
		tier = verify.ReadTierFromWorkspace(agent.WorkspacePath)
	}

	// Pre-fetch comments using the correct project directory
	comments, err := verify.GetComments(agent.BeadsID, effectiveProjectDir)
	if err != nil {
		signal.Error = fmt.Errorf("failed to fetch comments for %s (dir=%s): %w", agent.BeadsID, effectiveProjectDir, err)
		signal.Escalation = verify.EscalationFailed
		return signal
	}

	// Run full verification with pre-fetched comments
	verificationResult, err := verify.VerifyCompletionFullWithComments(
		agent.BeadsID,
		agent.WorkspacePath,
		effectiveProjectDir,
		tier,
		config.ServerURL,
		comments,
	)
	if err != nil {
		signal.Error = fmt.Errorf("verification failed: %w", err)
		signal.Verification = verificationResult
		signal.Escalation = verify.EscalationFailed
		return signal
	}

	signal.Verification = verificationResult

	// Parse synthesis for escalation signals
	var synthesis *verify.Synthesis
	if agent.WorkspacePath != "" {
		synthesis, _ = verify.ParseSynthesis(agent.WorkspacePath)
	}

	// Determine escalation level
	signal.Escalation = verify.DetermineEscalationFromCompletion(
		verificationResult,
		synthesis,
		agent.BeadsID,
		agent.WorkspacePath,
		effectiveProjectDir,
	)

	// Check if verification passed
	if !verificationResult.Passed {
		signal.Error = fmt.Errorf("verification failed: %s", strings.Join(verificationResult.Errors, "; "))
		return signal
	}

	// Check if escalation allows auto-completion
	if !signal.Escalation.ShouldAutoComplete() {
		reason := verify.ExplainEscalation(verify.EscalationInput{
			VerificationPassed:  verificationResult.Passed,
			VerificationErrors:  verificationResult.Errors,
			NeedsVisualApproval: signal.Escalation == verify.EscalationBlock,
		})
		signal.Error = fmt.Errorf("requires human review: %s", reason.Reason)
		return signal
	}

	signal.Passed = true
	return signal
}
