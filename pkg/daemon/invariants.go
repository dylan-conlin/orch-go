// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// InvariantViolation represents a single invariant check failure.
type InvariantViolation struct {
	// Name identifies which invariant was violated.
	Name string
	// Message describes the violation with diagnostic detail.
	Message string
	// Severity is "warning" or "critical".
	Severity string
	// Timestamp is when the violation was detected.
	Timestamp time.Time
}

// InvariantCheckResult contains the outcome of running all invariant checks.
type InvariantCheckResult struct {
	// Violations contains all detected violations.
	Violations []InvariantViolation
	// CheckedAt is when the checks were run.
	CheckedAt time.Time
	// Error is set if the check itself failed to run (infrastructure issue).
	// When set, the result should be treated as fail-open (skip, don't pause).
	Error error
}

// HasViolations returns true if any violations were detected.
func (r *InvariantCheckResult) HasViolations() bool {
	return len(r.Violations) > 0
}

// CriticalCount returns the number of critical violations.
func (r *InvariantCheckResult) CriticalCount() int {
	count := 0
	for _, v := range r.Violations {
		if v.Severity == "critical" {
			count++
		}
	}
	return count
}

// InvariantChecker runs self-check invariants on daemon state each poll cycle.
// It tracks violation counts across cycles and triggers pause when the
// threshold is exceeded.
type InvariantChecker struct {
	mu sync.Mutex

	// threshold is the number of violations before triggering pause.
	// Default: 3. Zero disables the checker.
	threshold int

	// violationCount tracks consecutive cycles with violations.
	violationCount int

	// lastViolations stores the most recent violations for diagnostics.
	lastViolations []InvariantViolation

	// isPaused indicates the checker has triggered a pause.
	isPaused bool

	// maxAgents is the configured concurrency cap.
	maxAgents int
}

// NewInvariantChecker creates a new InvariantChecker.
// threshold is the number of violation cycles before pause (0 = disabled).
// maxAgents is the configured concurrency cap for range checks.
func NewInvariantChecker(threshold, maxAgents int) *InvariantChecker {
	return &InvariantChecker{
		threshold: threshold,
		maxAgents: maxAgents,
	}
}

// InvariantInput contains the daemon state snapshot needed for invariant checks.
// All fields are pre-collected by the caller to keep checks pure and fast.
type InvariantInput struct {
	// ActiveCount is the current number of active agents (from pool).
	ActiveCount int
	// MaxAgents is the configured concurrency cap.
	MaxAgents int
	// PoolActiveCount is the pool's internal count (may differ from ActiveCount
	// if reconciliation hasn't run yet).
	PoolActiveCount int

	// VerificationCount is the number of unverified completions.
	VerificationCount int
	// VerificationThreshold is the configured threshold.
	VerificationThreshold int

	// CompletedAgents are agents that reported Phase: Complete.
	// Used to check ProjectDir validity for cross-project agents.
	CompletedAgents []CompletedAgent
}

// Check runs all invariant assertions against the provided state snapshot.
// Returns a result with any violations found. If the check itself cannot run
// (e.g., nil input), returns a result with Error set (fail-open).
func (ic *InvariantChecker) Check(input *InvariantInput) *InvariantCheckResult {
	if ic.threshold == 0 {
		return &InvariantCheckResult{CheckedAt: time.Now()}
	}

	if input == nil {
		return &InvariantCheckResult{
			CheckedAt: time.Now(),
			Error:     fmt.Errorf("invariant check: nil input (fail-open)"),
		}
	}

	now := time.Now()
	result := &InvariantCheckResult{CheckedAt: now}

	// Invariant 1: Active agent count must be in valid range [0, 2*MaxAgents]
	// A count > 2x cap indicates a reconciliation bug or ghost agents.
	if input.MaxAgents > 0 {
		upperBound := input.MaxAgents * 2
		if input.ActiveCount < 0 {
			result.Violations = append(result.Violations, InvariantViolation{
				Name:      "active-count-negative",
				Message:   fmt.Sprintf("active agent count is negative: %d", input.ActiveCount),
				Severity:  "critical",
				Timestamp: now,
			})
		}
		if input.ActiveCount > upperBound {
			result.Violations = append(result.Violations, InvariantViolation{
				Name:      "active-count-exceeds-cap",
				Message:   fmt.Sprintf("active agent count %d exceeds 2x cap (%d max, %d upper bound)", input.ActiveCount, input.MaxAgents, upperBound),
				Severity:  "critical",
				Timestamp: now,
			})
		}
	}

	// Invariant 2: Verification counter must not exceed threshold
	// If it does, the pause mechanism failed.
	if input.VerificationThreshold > 0 && input.VerificationCount > input.VerificationThreshold {
		result.Violations = append(result.Violations, InvariantViolation{
			Name:     "verification-counter-overflow",
			Message:  fmt.Sprintf("verification counter %d exceeds threshold %d (pause mechanism may have failed)", input.VerificationCount, input.VerificationThreshold),
			Severity: "critical",
			Timestamp: now,
		})
	}

	// Invariant 3: Completed agents must have valid ProjectDir for cross-project agents
	// An empty ProjectDir on a cross-project agent means completion processing
	// will run bd commands against the wrong directory.
	for _, agent := range input.CompletedAgents {
		if isCrossProjectAgent(agent) && agent.ProjectDir == "" {
			result.Violations = append(result.Violations, InvariantViolation{
				Name:      "completion-missing-project-dir",
				Message:   fmt.Sprintf("completed agent %s has no ProjectDir (cross-project completion will fail)", agent.BeadsID),
				Severity:  "warning",
				Timestamp: now,
			})
		}
	}

	// Invariant 4: Completed agents must have valid beads IDs (not synthetic/untracked)
	for _, agent := range input.CompletedAgents {
		if isUntrackedOrSyntheticBeadsID(agent.BeadsID) {
			result.Violations = append(result.Violations, InvariantViolation{
				Name:      "completion-synthetic-beads-id",
				Message:   fmt.Sprintf("completed agent has synthetic beads ID: %s", agent.BeadsID),
				Severity:  "warning",
				Timestamp: now,
			})
		}
	}

	// Record result
	ic.recordResult(result)

	return result
}

// recordResult updates internal state based on check outcome.
func (ic *InvariantChecker) recordResult(result *InvariantCheckResult) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if result.Error != nil {
		// Fail-open: infrastructure error, don't count as violation
		return
	}

	if result.HasViolations() {
		ic.violationCount++
		ic.lastViolations = result.Violations
		if ic.threshold > 0 && ic.violationCount >= ic.threshold {
			ic.isPaused = true
		}
	} else {
		// Clean cycle resets the counter
		ic.violationCount = 0
		ic.lastViolations = nil
	}
}

// IsPaused returns true if the checker has triggered a pause due to
// repeated invariant violations.
func (ic *InvariantChecker) IsPaused() bool {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	return ic.isPaused
}

// ViolationCount returns the number of consecutive violation cycles.
func (ic *InvariantChecker) ViolationCount() int {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	return ic.violationCount
}

// LastViolations returns the most recent violations for diagnostics.
func (ic *InvariantChecker) LastViolations() []InvariantViolation {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	cp := make([]InvariantViolation, len(ic.lastViolations))
	copy(cp, ic.lastViolations)
	return cp
}

// Resume resets the violation counter and unpauses.
func (ic *InvariantChecker) Resume() {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.violationCount = 0
	ic.lastViolations = nil
	ic.isPaused = false
}

// isCrossProjectAgent returns true if the agent appears to be from a different project.
// Cross-project agents have beads IDs with a prefix that doesn't match the current project.
func isCrossProjectAgent(agent CompletedAgent) bool {
	// If workspace path contains a different project directory, it's cross-project.
	// Simple heuristic: agents with WorkspacePath set but empty ProjectDir are suspicious.
	// The actual cross-project detection is done by the completion scanner
	// which sets ProjectDir for agents found in other projects.
	// Here we check: if the agent was found (has a workspace), it should have ProjectDir.
	return agent.WorkspacePath != ""
}

// isUntrackedOrSyntheticBeadsID returns true if the beads ID looks synthetic.
// Synthetic IDs are generated for workspaces without proper beads tracking.
func isUntrackedOrSyntheticBeadsID(beadsID string) bool {
	if beadsID == "" {
		return true
	}
	// Pattern: *-untracked-* (generated by workspace scanner for untracked agents)
	if strings.Contains(beadsID, "-untracked-") {
		return true
	}
	return false
}
