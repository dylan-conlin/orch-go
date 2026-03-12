// Package daemon provides autonomous overnight processing capabilities.
// This file implements stalled agent diagnostic classification based on 6 failure
// modes identified in the 2026-02-28 audit of 1655 archived workspaces.
//
// Failure modes (ordered by frequency from audit):
//  1. SYNTHESIS Compliance Gap — agent completed but no SYNTHESIS.md
//  2. Silent Failure — agent never reported any phase
//  3. Model Protocol Incompatibility — non-Anthropic model can't follow protocol
//  4. QUESTION Deadlock — agent needs input, no delivery mechanism
//  5. Prior Art Confusion — agent discovers overlapping prior work, stalls
//  6. Concurrency Ceiling — agent blocked by resource limits
//
// See: .kb/investigations/2026-02-28-audit-stalled-agent-failure-patterns.md
package daemon

import (
	"fmt"
	"strings"
	"time"
)

// FailureMode identifies the category of agent stall.
type FailureMode string

const (
	FailureModeNone                FailureMode = "none"
	FailureModeSynthesisGap        FailureMode = "synthesis_gap"
	FailureModeSilentFailure       FailureMode = "silent_failure"
	FailureModeModelIncompatibility FailureMode = "model_incompatibility"
	FailureModeQuestionDeadlock    FailureMode = "question_deadlock"
	FailureModePriorArtConfusion   FailureMode = "prior_art_confusion"
	FailureModeConcurrencyCeiling  FailureMode = "concurrency_ceiling"
	FailureModePhaseStall          FailureMode = "phase_stall"
)

// Severity indicates how urgent the failure mode is.
type Severity string

const (
	SeverityNone       Severity = "none"
	SeverityAdvisory   Severity = "advisory"   // informational, no action needed
	SeverityActionable Severity = "actionable"  // can be resolved automatically or with simple intervention
	SeverityCritical   Severity = "critical"    // requires immediate attention
)

// ActionType identifies a recommended response to a failure mode.
type ActionType string

const (
	ActionNotifyUser      ActionType = "notify_user"       // surface to user via notification
	ActionRespawnWithModel ActionType = "respawn_with_model" // abandon and respawn with a different model
	ActionWaitForSlot     ActionType = "wait_for_slot"      // wait for concurrency slot to open
	ActionResumeAgent     ActionType = "resume_agent"       // send resume prompt
	ActionAbandon         ActionType = "abandon"            // mark agent as abandoned
	ActionEnforceSynthesis ActionType = "enforce_synthesis"  // reject completion without SYNTHESIS
	ActionInjectPriorArt  ActionType = "inject_prior_art"   // inject prior completion info
)

// RecommendedAction is a specific action the orchestrator can take.
type RecommendedAction struct {
	Action      ActionType
	Description string
	// Metadata holds action-specific parameters (e.g., model to respawn with).
	Metadata map[string]string
}

// DiagnosticAgent holds the data needed to classify an agent's failure mode.
// This is the input to the diagnostic classifier — callers populate it from
// beads, workspace, and session data.
type DiagnosticAgent struct {
	BeadsID        string
	Title          string
	Phase          string
	Model          string
	Skill          string
	HasSession     bool      // whether a live session (OpenCode or tmux) exists
	HasSynthesis   bool      // whether SYNTHESIS.md exists in workspace
	IsFullTier     bool      // whether this is a full-tier (not light) spawn
	HasPriorAgents bool      // whether prior agents worked on same/similar issue
	UpdatedAt      time.Time // last phase report timestamp
}

// DiagnosticResult is the classification output for a single agent.
type DiagnosticResult struct {
	BeadsID            string
	Mode               FailureMode
	Description        string
	Severity           Severity
	IdleDuration       time.Duration
	RecommendedActions []RecommendedAction
}

// String returns a human-readable summary.
func (r DiagnosticResult) String() string {
	if r.Mode == FailureModeNone {
		return fmt.Sprintf("%s: healthy", r.BeadsID)
	}
	return fmt.Sprintf("%s: %s (%s) — %s [idle %v]",
		r.BeadsID, r.Mode, r.Severity, r.Description, r.IdleDuration.Round(time.Minute))
}

// DiagnosticReport is the aggregate output from diagnosing multiple agents.
type DiagnosticReport struct {
	TotalAgents  int
	HealthyCount int
	FailingCount int
	// ByMode groups failing agents by failure mode.
	ByMode map[FailureMode][]DiagnosticResult
	// All contains every result (including healthy).
	All []DiagnosticResult
}

// Thresholds for classification.
const (
	// silentFailureThreshold is how long an agent can go without reporting
	// any phase before being classified as a silent failure.
	silentFailureThreshold = 30 * time.Minute

	// phaseStallThreshold is how long an agent can stay in a non-terminal
	// phase before being classified as stalled.
	phaseStallThreshold = 30 * time.Minute
)

// ClassifyFailureMode determines which of the 6 audit failure modes applies
// to the given agent. Returns FailureModeNone if the agent is healthy.
func ClassifyFailureMode(agent DiagnosticAgent) DiagnosticResult {
	now := time.Now()
	idle := now.Sub(agent.UpdatedAt)
	phaseName := extractPhaseName(agent.Phase)

	result := DiagnosticResult{
		BeadsID:      agent.BeadsID,
		IdleDuration: idle,
	}

	// 1. Check for completed agents first
	if strings.EqualFold(phaseName, "complete") {
		return classifyCompleted(agent, result)
	}

	// 2. QUESTION deadlock — agent explicitly asked for input
	if strings.EqualFold(phaseName, "question") {
		result.Mode = FailureModeQuestionDeadlock
		result.Severity = SeverityActionable
		result.Description = fmt.Sprintf("Agent waiting for answer: %s", extractPhaseDetail(agent.Phase))
		result.RecommendedActions = []RecommendedAction{
			{Action: ActionNotifyUser, Description: "Surface question to user for response"},
			{Action: ActionResumeAgent, Description: "Send answer via orch send"},
		}
		return result
	}

	// 3. BLOCKED — concurrency ceiling or resource constraint
	if strings.EqualFold(phaseName, "blocked") {
		result.Mode = FailureModeConcurrencyCeiling
		result.Severity = SeverityActionable
		result.Description = fmt.Sprintf("Agent blocked: %s", extractPhaseDetail(agent.Phase))
		result.RecommendedActions = []RecommendedAction{
			{Action: ActionWaitForSlot, Description: "Wait for concurrency slot to open"},
			{Action: ActionNotifyUser, Description: "Escalate to orchestrator for triage"},
		}
		return result
	}

	// 4. Silent failure — no phase ever reported
	if agent.Phase == "" {
		if idle < silentFailureThreshold {
			// Too early to classify — agent may still be starting up
			result.Mode = FailureModeNone
			result.Severity = SeverityNone
			return result
		}
		result.Mode = FailureModeSilentFailure
		result.Severity = SeverityCritical
		result.Description = "Agent never reported any phase — possible startup crash or model incompatibility"
		result.RecommendedActions = []RecommendedAction{
			{Action: ActionAbandon, Description: "Abandon and respawn if issue still relevant"},
		}
		if isNonAnthropicModel(agent.Model) {
			result.Description += " (non-Anthropic model — likely protocol incompatibility)"
			result.RecommendedActions = append(result.RecommendedActions, RecommendedAction{
				Action:      ActionRespawnWithModel,
				Description: "Respawn with Anthropic model",
				Metadata:    map[string]string{"model": "anthropic/claude-opus-4-5"},
			})
		}
		return result
	}

	// 5. Agent has a non-terminal phase and is stalled — classify the stall
	if idle < phaseStallThreshold {
		// Not yet stalled
		result.Mode = FailureModeNone
		result.Severity = SeverityNone
		return result
	}

	// 6. Prior art confusion — stalled in Exploration with prior agents
	if agent.HasPriorAgents && strings.EqualFold(phaseName, "exploration") {
		result.Mode = FailureModePriorArtConfusion
		result.Severity = SeverityActionable
		result.Description = "Agent stalled in Exploration due to overlapping prior work"
		result.RecommendedActions = []RecommendedAction{
			{Action: ActionInjectPriorArt, Description: "Inject prior completion info and resume"},
			{Action: ActionResumeAgent, Description: "Send scope clarification prompt"},
		}
		return result
	}

	// 7. Model protocol incompatibility — non-Anthropic model stalled in protocol-heavy phase
	if isNonAnthropicModel(agent.Model) {
		result.Mode = FailureModeModelIncompatibility
		result.Severity = SeverityActionable
		result.Description = fmt.Sprintf("Non-Anthropic model (%s) stalled in %s — 67-87%% stall rate for protocol-heavy skills",
			agent.Model, phaseName)
		result.RecommendedActions = []RecommendedAction{
			{Action: ActionRespawnWithModel, Description: "Abandon and respawn with Anthropic model",
				Metadata: map[string]string{"model": "anthropic/claude-opus-4-5"}},
			{Action: ActionAbandon, Description: "Abandon agent"},
		}
		return result
	}

	// 8. Generic phase stall — Anthropic model stuck in a non-terminal phase
	result.Mode = FailureModePhaseStall
	result.Severity = SeverityActionable
	result.Description = fmt.Sprintf("Agent stalled in %s for %v", phaseName, idle.Round(time.Minute))
	result.RecommendedActions = []RecommendedAction{
		{Action: ActionResumeAgent, Description: "Send resume prompt to continue work"},
		{Action: ActionAbandon, Description: "Abandon if unrecoverable"},
	}
	return result
}

// classifyCompleted handles agents that reported Phase: Complete.
func classifyCompleted(agent DiagnosticAgent, result DiagnosticResult) DiagnosticResult {
	// Full-tier agent completed without SYNTHESIS = compliance gap
	if agent.IsFullTier && !agent.HasSynthesis {
		result.Mode = FailureModeSynthesisGap
		result.Severity = SeverityAdvisory
		result.Description = "Agent reported Phase: Complete but SYNTHESIS.md is missing"
		result.RecommendedActions = []RecommendedAction{
			{Action: ActionEnforceSynthesis, Description: "Reject completion — require SYNTHESIS.md"},
		}
		return result
	}

	// Completed successfully
	result.Mode = FailureModeNone
	result.Severity = SeverityNone
	return result
}

// RunDiagnostics classifies all provided agents and returns an aggregate report.
func RunDiagnostics(agents []DiagnosticAgent) DiagnosticReport {
	report := DiagnosticReport{
		TotalAgents: len(agents),
		ByMode:      make(map[FailureMode][]DiagnosticResult),
	}

	for _, agent := range agents {
		result := ClassifyFailureMode(agent)
		report.All = append(report.All, result)

		if result.Mode == FailureModeNone {
			report.HealthyCount++
		} else {
			report.FailingCount++
			report.ByMode[result.Mode] = append(report.ByMode[result.Mode], result)
		}
	}

	return report
}

// extractPhaseName returns the phase name before the " - " detail separator.
// e.g., "QUESTION - Should we use JWT?" → "QUESTION"
func extractPhaseName(phase string) string {
	if idx := strings.Index(phase, " - "); idx >= 0 {
		return strings.TrimSpace(phase[:idx])
	}
	return strings.TrimSpace(phase)
}

// extractPhaseDetail returns the detail text after the " - " separator.
// e.g., "QUESTION - Should we use JWT?" → "Should we use JWT?"
func extractPhaseDetail(phase string) string {
	if idx := strings.Index(phase, " - "); idx >= 0 {
		return strings.TrimSpace(phase[idx+3:])
	}
	return ""
}

// isNonAnthropicModel returns true if the model is not an Anthropic model.
// Empty/unknown models are treated as potentially Anthropic (benefit of the doubt).
func isNonAnthropicModel(model string) bool {
	if model == "" {
		return false
	}
	return !strings.HasPrefix(strings.ToLower(model), "anthropic/")
}

// isProtocolHeavySkill returns true if the skill requires substantial
// multi-step protocol compliance (phase reporting, SYNTHESIS, etc.).
func isProtocolHeavySkill(skill string) bool {
	switch strings.ToLower(skill) {
	case "architect", "investigation", "feature-impl", "systematic-debugging":
		return true
	default:
		return false
	}
}
