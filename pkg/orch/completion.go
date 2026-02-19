// Package orch provides orchestration-level utilities for agent management.
// This includes completion backlog detection and related metrics.
package orch

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/checkpoint"
	"github.com/dylan-conlin/orch-go/pkg/verify"
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

// addBeadsComment adds a comment to a beads issue.
// This is used internally by the explain-back gate to store explanations.
// It tries RPC first, then falls back to CLI if RPC fails.
func addBeadsComment(beadsID, comment string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		// Use "orchestrator" as the author for explain-back comments
		err := client.AddComment(beadsID, "orchestrator", comment)
		if err == nil {
			return nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackAddComment(beadsID, comment)
}

// getOrientationFrame retrieves the most recent FRAME annotation from beads comments.
// Returns empty string if no frame is found or comments cannot be retrieved.
func getOrientationFrame(beadsID string) string {
	if beadsID == "" {
		return ""
	}

	var comments []beads.Comment
	// Try RPC client first with auto-reconnect
	if socketPath, err := beads.FindSocketPath(""); err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			if result, err := client.Comments(beadsID); err == nil {
				comments = result
			}
		}
	}

	// Fallback to CLI if RPC failed or returned nothing
	if comments == nil {
		if result, err := beads.FallbackComments(beadsID); err == nil {
			comments = result
		}
	}

	if len(comments) == 0 {
		return ""
	}

	for i := len(comments) - 1; i >= 0; i-- {
		text := strings.TrimSpace(comments[i].Text)
		if strings.HasPrefix(text, "FRAME:") {
			return strings.TrimSpace(strings.TrimPrefix(text, "FRAME:"))
		}
	}

	return ""
}

// RunExplainBackGate executes the explain-back verification gate for agent completion.
// This gate requires the orchestrator to provide an explanation of what was built via
// the --explain flag, creating a verification that ensures human comprehension beyond
// agent self-reporting.
//
// The gate is skipped for:
//   - Orchestrator sessions (they have handoffs instead)
//   - Untracked agents (no beads issue to comment on)
//   - When skipExplainBack is true (with a required reason)
//
// When the gate is active, it:
//   - Gates on non-empty explanation text (from --explain flag)
//   - Formats and stores the explanation as a beads comment
//
// The conversational quality check (is the explanation sufficient?) stays with the
// AI orchestrator. The CLI's job is: accept explanation, store it, gate on non-empty.
//
// Parameters:
//   - beadsID: The beads issue ID being completed
//   - forced: Whether --force was used (skips the gate entirely)
//   - skipExplainBack: Whether --skip-explain-back was set
//   - skipReason: The reason for skipping (required if skipExplainBack is true)
//   - isOrchestratorSession: Whether this is an orchestrator session
//   - isUntracked: Whether this is an untracked agent
//   - explanation: The explanation text from --explain flag
//   - verified: Whether --verified flag was set (records gate2 in checkpoint)
//   - stdout: Output writer for status messages
//
// Returns:
//   - error: Error if explanation is missing or storage fails
func RunExplainBackGate(
	beadsID string,
	forced bool,
	skipExplainBack bool,
	skipReason string,
	isOrchestratorSession bool,
	isUntracked bool,
	explanation string,
	verified bool,
	stdout io.Writer,
) error {
	// Skip for orchestrator sessions (they have handoffs)
	if isOrchestratorSession {
		return nil
	}

	// Skip for untracked agents (no beads issue)
	if isUntracked || beadsID == "" {
		return nil
	}

	// Skip if --force was used
	if forced {
		fmt.Fprintln(stdout, "Skipping explain-back verification (--force)")
		return nil
	}

	// Skip if --skip-explain-back was set
	if skipExplainBack {
		fmt.Fprintf(stdout, "⚠️  Bypassing explain-back gate (reason: %s)\n", skipReason)
		return nil
	}

	// Gate on non-empty explanation
	if explanation == "" {
		fmt.Fprintln(stdout, "❌ Explain-back gate: --explain flag is required")
		fmt.Fprintln(stdout, "")
		if frame := getOrientationFrame(beadsID); frame != "" {
			fmt.Fprintf(stdout, "Spawn frame: %s\n\n", frame)
		}
		fmt.Fprintln(stdout, "The orchestrator must provide an explanation of what was built:")
		fmt.Fprintln(stdout, "  orch complete <id> --explain 'Built X because Y, verified by Z'")
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Or bypass with:")
		fmt.Fprintln(stdout, "  --skip-explain-back --skip-reason \"...\"")
		return fmt.Errorf("explain-back gate: --explain is required")
	}

	// Format and validate the explanation
	explainResult, err := verify.FormatExplainBack(explanation)
	if err != nil {
		return fmt.Errorf("explain-back verification failed: %w", err)
	}

	// Store explanation as beads comment
	if err := addBeadsComment(beadsID, explainResult.FullExplanation); err != nil {
		fmt.Fprintf(stdout, "Warning: failed to save explanation to beads: %v\n", err)
		// Continue anyway - the explanation was captured even if storage failed
	} else {
		fmt.Fprintln(stdout, "Explanation saved to beads issue")
	}

	// Write checkpoint entry (Phase 1: Comprehension gate)
	// This creates a persistent record that the orchestrator has verified comprehension.
	// The checkpoint file is used by orch complete to gate Tier 1 work.
	cp := checkpoint.Checkpoint{
		BeadsID:       beadsID,
		Deliverable:   "completion", // Could be enhanced to track specific deliverables
		Gate1Complete: true,         // Comprehension gate (explain-back)
		Gate2Complete: verified,     // Behavioral gate (via --verified flag)
		Timestamp:     time.Now(),
		ExplainText:   explanation,
	}

	if err := checkpoint.WriteCheckpoint(cp); err != nil {
		fmt.Fprintf(stdout, "Warning: failed to write verification checkpoint: %v\n", err)
		// Continue anyway - the beads comment was saved, checkpoint is supplementary
	} else {
		fmt.Fprintln(stdout, "Verification checkpoint written")
	}

	return nil
}

// RecordGate2Checkpoint writes a checkpoint entry recording gate2 (behavioral verification).
// This is used when gate1 was already recorded in a previous run and only gate2 needs to be added.
func RecordGate2Checkpoint(beadsID string, stdout io.Writer) error {
	cp := checkpoint.Checkpoint{
		BeadsID:       beadsID,
		Deliverable:   "completion",
		Gate1Complete: true, // Gate1 must have passed before gate2
		Gate2Complete: true, // Behavioral gate
		Timestamp:     time.Now(),
	}

	if err := checkpoint.WriteCheckpoint(cp); err != nil {
		fmt.Fprintf(stdout, "Warning: failed to write gate2 checkpoint: %v\n", err)
		return err
	}
	fmt.Fprintln(stdout, "✓ Behavioral verification (gate2) checkpoint recorded")
	return nil
}
