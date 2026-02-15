// Package orch provides orchestration-level utilities for agent management.
// This includes completion backlog detection and related metrics.
package orch

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
	"golang.org/x/term"
)

// AgentInfo represents the minimal agent information needed for completion backlog detection.
// This is populated from serve_agents.go's agent data structures.
type AgentInfo struct {
	BeadsID         string    // Beads issue ID (e.g., "orch-go-e6o")
	SessionID       string    // OpenCode session ID
	Phase           string    // Current phase from beads comments (e.g., "Planning", "Complete")
	PhaseReportedAt time.Time // Timestamp when the phase was reported
	Status          string    // Agent status (e.g., "active", "idle", "dead", "completed")
}

// DetectCompletionBacklog checks for agents that have reported Phase: Complete
// but haven't been closed by orch complete for longer than the threshold duration.
//
// This is used to detect completion backlog - agents that are done but waiting for
// orchestrator action. These agents should be surfaced to the orchestrator for review.
//
// The threshold is typically 10 minutes, based on the coaching metrics design:
// "Detect agents at Phase:Complete for >10 minutes without orch complete being run."
//
// Parameters:
//   - agents: slice of AgentInfo structs containing agent phase and timing information
//   - threshold: duration after which a completed agent is considered backlogged
//
// Returns:
//   - slice of beads IDs for agents in completion backlog
//
// Example usage:
//
//	agents := []orch.AgentInfo{
//	    {BeadsID: "orch-go-abc", Phase: "Complete", PhaseReportedAt: time.Now().Add(-15 * time.Minute)},
//	    {BeadsID: "orch-go-xyz", Phase: "Planning", PhaseReportedAt: time.Now().Add(-5 * time.Minute)},
//	}
//	backlog := orch.DetectCompletionBacklog(agents, 10 * time.Minute)
//	// backlog = ["orch-go-abc"]
func DetectCompletionBacklog(agents []AgentInfo, threshold time.Duration) []string {
	now := time.Now()
	var backlog []string
	for _, a := range agents {
		// Skip agents not at Phase: Complete
		if !strings.EqualFold(a.Phase, "Complete") {
			continue
		}
		// Skip agents already closed by orch complete
		if a.Status == "completed" {
			continue
		}
		// Skip agents with zero PhaseReportedAt (no timestamp available)
		if a.PhaseReportedAt.IsZero() {
			continue
		}
		// Check if agent has been in Complete phase longer than threshold
		if now.Sub(a.PhaseReportedAt) > threshold {
			backlog = append(backlog, a.BeadsID)
		}
	}
	return backlog
}

// RunExplainBackGate executes the explain-back verification gate for agent completion.
// This gate requires the human orchestrator to explain what was built and why in their own words,
// creating an unfakeable verification that ensures human comprehension beyond agent self-reporting.
//
// The gate is skipped for:
//   - Orchestrator sessions (they have handoffs instead)
//   - Untracked agents (no beads issue to comment on)
//   - When skipExplainBack is true (with a required reason)
//
// When the gate is active, it:
//   - Checks if stdin is a terminal (required for interactive prompts)
//   - Prompts the human for a structured explanation (what, why, verification)
//   - Returns the explanation for the caller to store
//
// This is extracted from cmd/orch/complete_cmd.go to pkg/orch for reusability
// and separation of concerns (orchestration logic vs command handling).
//
// Parameters:
//   - beadsID: The beads issue ID being completed
//   - forced: Whether --force was used (skips the gate entirely)
//   - skipExplainBack: Whether --skip-explain-back was set
//   - skipReason: The reason for skipping (required if skipExplainBack is true)
//   - isOrchestratorSession: Whether this is an orchestrator session
//   - isUntracked: Whether this is an untracked agent
//   - stdin: Input reader for prompts
//   - stdout: Output writer for prompts
//
// Returns:
//   - *verify.ExplainBackResult: The captured explanation (nil if gate was skipped)
//   - error: Error if verification fails or terminal check fails
func RunExplainBackGate(
	beadsID string,
	forced bool,
	skipExplainBack bool,
	skipReason string,
	isOrchestratorSession bool,
	isUntracked bool,
	stdin io.Reader,
	stdout io.Writer,
) (*verify.ExplainBackResult, error) {
	// Skip for orchestrator sessions (they have handoffs)
	if isOrchestratorSession {
		return nil, nil
	}

	// Skip for untracked agents (no beads issue)
	if isUntracked || beadsID == "" {
		return nil, nil
	}

	// Skip if --force was used
	if forced {
		fmt.Fprintln(stdout, "Skipping explain-back verification (--force)")
		return nil, nil
	}

	// Skip if --skip-explain-back was set
	if skipExplainBack {
		fmt.Fprintf(stdout, "⚠️  Bypassing explain-back gate (reason: %s)\n", skipReason)
		return nil, nil
	}

	// Check if stdin is a terminal for interactive prompting
	// This is required because explain-back needs conversational input
	if stdinFile, ok := stdin.(interface{ Fd() uintptr }); ok {
		if !term.IsTerminal(int(stdinFile.Fd())) {
			fmt.Fprintln(stdout, "⚠️  Explain-back gate requires terminal interaction")
			fmt.Fprintln(stdout, "Use --skip-explain-back --skip-reason \"...\" to bypass in non-interactive mode")
			return nil, fmt.Errorf("explain-back gate requires terminal")
		}
	} else {
		// If we can't determine terminal status (e.g., in tests with bytes.Buffer),
		// proceed anyway - the PromptExplainBack call will fail naturally if stdin doesn't work
	}

	// Prompt for explanation
	explainResult, err := verify.PromptExplainBack(stdin, stdout)
	if err != nil {
		return nil, fmt.Errorf("explain-back verification failed: %w", err)
	}

	return explainResult, nil
}
