// Package daemon provides autonomous overnight processing capabilities.
// ooda.go structures the daemon's poll cycle into explicit OODA phases:
// Sense → Orient → Decide → Act. Each phase is a named method on Daemon.
// The Act phase feeds back into Sense on the next cycle.
package daemon

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// --- SENSE phase: gather raw signals from the environment ---

// SenseResult holds raw signals collected from the environment.
// This is the input to the Orient phase.
type SenseResult struct {
	// GateSignal is the result of pre-spawn compliance gates.
	GateSignal SpawnGateSignal
	// Issues is the raw list of ready issues from the queue.
	Issues []Issue
	// IssueErr is non-nil if the issue query failed.
	IssueErr error
}

// Sense gathers raw signals: checks compliance gates and polls the issue queue.
// Pure data collection — no decisions, no side effects beyond querying.
func (d *Daemon) Sense(skip map[string]bool) SenseResult {
	result := SenseResult{}

	// Compliance gates (verification pause, completion health, rate limit)
	result.GateSignal = d.CheckPreSpawnGates()

	// Poll the ready queue — even if gates block, we collect the data
	// so Orient can still analyze the work graph for status reporting.
	issues, err := d.resolveIssueQuerier().ListReadyIssues()
	result.Issues = issues
	result.IssueErr = err

	if d.Config.Verbose && err == nil {
		fmt.Printf("  DEBUG: Sense: found %d open issues\n", len(issues))
	}

	return result
}

// --- ORIENT phase: analyze and contextualize raw signals ---

// OrientResult holds analyzed, prioritized data ready for decision-making.
type OrientResult struct {
	// Sense is the raw signal data from the Sense phase.
	Sense SenseResult
	// PrioritizedIssues is the issue list after epic expansion, focus boost,
	// allocation scoring, and project interleaving.
	PrioritizedIssues []Issue
	// EpicChildIDs tracks which issues are epic children (for label exemption).
	EpicChildIDs map[string]bool
	// ChannelHealthWarnings flags skills where rework=0 alongside high
	// completion volume — absent negative signal should not be treated as positive.
	ChannelHealthWarnings []ChannelHealthWarning
	// ThinIssueIDs lists issue IDs with empty descriptions.
	// Advisory-only: agents spawned from these rely on title-only orientation.
	ThinIssueIDs []string
	// OrientErr is non-nil if prioritization failed.
	OrientErr error
}

// Orient analyzes raw signals: prioritizes issues, applies coordination logic.
// Transforms Sense data into a form that supports decision-making.
// Does not mutate state — produces an OrientResult for the Decide phase.
func (d *Daemon) Orient(sense SenseResult) OrientResult {
	result := OrientResult{
		Sense:        sense,
		EpicChildIDs: make(map[string]bool),
	}

	if sense.IssueErr != nil {
		result.OrientErr = sense.IssueErr
		return result
	}

	// Prioritize: epic expansion, focus boost, allocation scoring, interleaving
	prioritized, epicChildIDs, err := d.PrioritizeIssues(sense.Issues)
	if err != nil {
		result.OrientErr = err
		return result
	}

	result.PrioritizedIssues = prioritized
	result.EpicChildIDs = epicChildIDs

	// Check for silent feedback channels (rework=0 with high completions)
	result.ChannelHealthWarnings = CheckChannelHealth(d.Learning)

	// Detect thin issues (empty description) for observability
	result.ThinIssueIDs = DetectThinIssues(prioritized)

	return result
}

// DetectThinIssues returns IDs of issues with empty descriptions.
// These issues will produce agents with title-only orientation.
func DetectThinIssues(issues []Issue) []string {
	var thin []string
	for _, issue := range issues {
		if strings.TrimSpace(issue.Description) == "" {
			thin = append(thin, issue.ID)
		}
	}
	return thin
}

// LogThinIssueAdvisories logs and emits events for thin issues detected in Orient.
// Advisory-only: no blocking, purely observational for telemetry.
func LogThinIssueAdvisories(orient OrientResult, verbose bool) {
	if len(orient.ThinIssueIDs) == 0 {
		return
	}

	// Build a lookup for titles
	titleByID := make(map[string]string)
	for _, issue := range orient.PrioritizedIssues {
		titleByID[issue.ID] = issue.Title
	}

	logger := events.NewDefaultLogger()
	for _, id := range orient.ThinIssueIDs {
		title := titleByID[id]
		if verbose {
			fmt.Printf("  ADVISORY: thin issue %s has no description — agent will rely on title-only orientation\n", id)
		}
		_ = logger.LogThinIssueDetected(events.ThinIssueDetectedData{
			IssueID: id,
			Title:   title,
		})
	}
}

// --- DECIDE phase: select the next action based on oriented data ---

// SpawnDecision describes what the daemon should do this cycle.
type SpawnDecision struct {
	// ShouldSpawn is true if an issue was selected for spawning.
	ShouldSpawn bool
	// Blocked is true if compliance gates prevent spawning.
	Blocked bool
	// BlockReason explains why spawning is blocked (empty if not blocked).
	BlockReason string
	// Issue is the selected issue to spawn (nil if nothing to spawn).
	Issue *Issue
	// Skill is the inferred skill for the selected issue.
	Skill string
	// Model is the inferred model alias for the selected issue.
	Model string
	// ModelRouteReason explains why this model was chosen.
	ModelRouteReason string
	// Route contains routing metadata (extraction, architect escalation).
	Route SkillRoute
}

// Decide selects the next action: which issue to spawn, with what skill/model,
// after applying compliance filters and routing logic.
// Pure decision — no side effects. Returns a SpawnDecision for the Act phase.
func (d *Daemon) Decide(orient OrientResult, skip map[string]bool) SpawnDecision {
	decision := SpawnDecision{}

	// If gates blocked, propagate
	if !orient.Sense.GateSignal.Allowed {
		decision.Blocked = true
		decision.BlockReason = orient.Sense.GateSignal.Reason
		return decision
	}

	// If orient failed, nothing to decide
	if orient.OrientErr != nil {
		decision.Blocked = true
		decision.BlockReason = orient.OrientErr.Error()
		return decision
	}

	// Build sibling validator that checks beads for ghost issues.
	// Caches results within this Decide call to avoid repeated queries.
	siblingCache := make(map[string]bool)
	siblingExists := func(id string) bool {
		if cached, ok := siblingCache[id]; ok {
			return cached
		}
		_, err := d.resolveIssueQuerier().GetIssueStatus(id)
		exists := err == nil
		siblingCache[id] = exists
		if !exists && d.Config.Verbose {
			fmt.Printf("  DEBUG: Decide: ghost sibling %s not found in beads, ignoring\n", id)
		}
		return exists
	}

	// Filter each issue through coordination and compliance checks, select first passing
	var selected *Issue
	for _, issue := range orient.PrioritizedIssues {
		// Coordination: defer test issues when implementation siblings are pending (epic children only)
		if shouldDefer, reason := ShouldDeferTestIssue(issue, orient.PrioritizedIssues, siblingExists, orient.EpicChildIDs); shouldDefer {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Decide: deferring %s (%s)\n", issue.ID, reason)
			}
			continue
		}

		filter := d.CheckIssueCompliance(issue, skip, orient.EpicChildIDs)
		if !filter.Passed {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Decide: skipping %s (%s)\n", issue.ID, filter.Reason)
			}
			continue
		}
		if d.Config.Verbose {
			fmt.Printf("  DEBUG: Decide: selected %s (type=%s, labels=%v)\n", issue.ID, issue.IssueType, issue.Labels)
		}
		issueCopy := issue
		selected = &issueCopy
		break
	}

	if selected == nil {
		decision.BlockReason = "No spawnable issues in queue"
		return decision
	}

	// Infer skill and model
	skill, err := InferSkillFromIssue(selected)
	if err != nil {
		decision.Blocked = true
		decision.BlockReason = fmt.Sprintf("failed to infer skill: %v", err)
		return decision
	}

	modelRoute := RouteModel(skill, selected)

	// Apply daemon model routing config override if configured.
	// Config takes precedence over hardcoded RouteModel inference.
	if d.Config.ModelRouting != nil && d.Config.ModelRouting.IsConfigured() {
		configRoute := d.Config.ModelRouting.Resolve(skill, modelRoute.Model)
		if configRoute.Source != "none" {
			modelRoute.Model = configRoute.EffectiveModel
			modelRoute.Reason = configRoute.Reason
		}
	}

	// Route through hotspot extraction and architect escalation
	route, err := d.RouteIssueForSpawn(selected, skill, modelRoute.Model, modelRoute.Reason)
	if err != nil {
		decision.Blocked = true
		decision.BlockReason = err.Error()
		return decision
	}

	// Apply routing: replace issue/skill/model if extraction spawned
	if route.ExtractionSpawned {
		selected = route.ReplacementIssue
	}

	decision.ShouldSpawn = true
	decision.Issue = selected
	decision.Skill = route.Skill
	decision.Model = route.Model
	decision.ModelRouteReason = route.ModelRouteReason
	decision.Route = route

	return decision
}

// --- ACT phase: execute the decision ---

// Act executes a spawn decision. Returns the same OnceResult type for
// backward compatibility with the existing daemon loop.
func (d *Daemon) Act(decision SpawnDecision) (*OnceResult, error) {
	if !decision.ShouldSpawn {
		return &OnceResult{
			Processed: false,
			Message:   decision.BlockReason,
		}, nil
	}

	// Spawn the issue
	result, _, err := d.spawnIssue(decision.Issue, decision.Skill, decision.Model)
	if result != nil {
		result.ModelRouteReason = decision.ModelRouteReason
		if decision.Route.ExtractionSpawned {
			result.ExtractionSpawned = true
			result.OriginalIssueID = decision.Route.OriginalIssueID
		}
		if decision.Route.ArchitectEscalated {
			result.ArchitectEscalated = true
		}
		if decision.Route.ArchitectEscalationDetail != nil {
			result.ArchitectEscalationDetail = decision.Route.ArchitectEscalationDetail
		}
	}
	return result, err
}
