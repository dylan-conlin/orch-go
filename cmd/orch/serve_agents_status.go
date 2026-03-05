package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/coaching"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	orchpkg "github.com/dylan-conlin/orch-go/pkg/orch"
	"github.com/dylan-conlin/orch-go/pkg/port"
)

// getProjectAPIPort returns the allocated API port for the current project.
// Returns 0 if no allocation exists or on error.
func getProjectAPIPort() int {
	projectDir, err := os.Getwd()
	if err != nil {
		return 0
	}
	projectName := filepath.Base(projectDir)

	registry, err := port.New("")
	if err != nil {
		return 0
	}

	alloc := registry.Find(projectName, "api")
	if alloc == nil {
		return 0
	}

	return alloc.Port
}

// checkWorkspaceSynthesis checks if a workspace has a non-empty SYNTHESIS.md file.
// This is used to detect completion for untracked agents (--no-track) where
// there's no beads issue to check Phase: Complete.
func checkWorkspaceSynthesis(workspacePath string) bool {
	if workspacePath == "" {
		return false
	}
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	info, err := os.Stat(synthesisPath)
	if err != nil {
		return false
	}
	// SYNTHESIS.md must exist and be non-empty
	return info.Size() > 0
}

// determineAgentStatus implements the Priority Cascade model for agent status.
// This is the SINGLE CANONICAL source of truth for determining agent status.
// All consumers (dashboard API, CLI status, work-graph) must use this function.
//
// This function receives the RAW status from pkg/discovery.JoinWithReasonCodes
// (active, idle, retrying, completed, dead, unknown) and maps it to the final
// dashboard status vocabulary. No other function should perform status mapping.
//
// Priority order (highest to lowest):
//  1. Beads issue closed -> "completed" (orchestrator verified completion)
//  2. Phase: Complete AND session not alive -> "awaiting-cleanup" (agent done, needs orch complete)
//  3. Phase: Complete -> "completed" (agent declared done, still has active session)
//  4. SYNTHESIS.md exists AND session not alive -> "awaiting-cleanup" (artifact proves completion)
//  5. SYNTHESIS.md exists -> "completed" (artifact proves completion)
//  6. Raw status mapping (final fallback):
//     - "active" -> "active"
//     - "retrying" -> "active"
//     - "idle" -> "idle" (session running but not busy, can still be interacted with)
//     - "unknown" -> "dead" (unreachable/unresolvable agents shown as dead)
//     - "completed" -> "completed"
//     - "dead" -> "dead"
//
// The "awaiting-cleanup" status distinguishes completed-but-orphaned agents from crashed agents.
// This helps orchestrators prioritize: awaiting-cleanup needs orch complete, dead needs investigation.
//
// See .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md for design rationale.
// See .kb/investigations/2026-01-08-inv-handle-multiple-agents-same-beads.md for awaiting-cleanup addition.
func determineAgentStatus(issueClosed bool, phaseComplete bool, workspacePath string, sessionStatus string) string {
	// Priority 1: Beads issue closed -> completed (orchestrator verified completion)
	if issueClosed {
		return "completed"
	}

	hasSynthesis := checkWorkspaceSynthesis(workspacePath)
	// Session is "not alive" if it's dead or unknown -- the session process
	// is gone or unreachable. "idle" is NOT included: idle sessions are still
	// running and can be interacted with (e.g., orch send).
	isNotAlive := sessionStatus == "dead" || sessionStatus == "unknown"

	// Priority 2: Phase: Complete reported AND session not alive -> awaiting-cleanup
	// Agent finished work and reported completion, but orchestrator hasn't run orch complete.
	// This is NOT an error state - the agent did its job, just needs cleanup.
	if phaseComplete && isNotAlive {
		return "awaiting-cleanup"
	}

	// Priority 3: Phase: Complete reported (session still active) -> completed
	if phaseComplete {
		return "completed"
	}

	// Priority 4: SYNTHESIS.md exists AND session not alive -> awaiting-cleanup
	// Agent wrote synthesis artifact (proof of completion) but session is not alive.
	// Similar to Phase: Complete case - needs cleanup, not investigation.
	if hasSynthesis && isNotAlive {
		return "awaiting-cleanup"
	}

	// Priority 5: SYNTHESIS.md exists (session still active) -> completed
	if hasSynthesis {
		return "completed"
	}

	// Priority 6: Raw status mapping (canonical mapping from query engine vocabulary
	// to dashboard vocabulary). This is the ONLY place this mapping happens.
	switch sessionStatus {
	case "active":
		return "active"
	case "retrying":
		return "active"
	case "completed":
		return "completed"
	case "idle":
		// Idle sessions are still running but not busy. Keep as "idle"
		// so dashboard can distinguish between idle (reachable) and dead (gone).
		return "idle"
	case "unknown":
		// Unreachable or unresolvable sessions shown as dead.
		return "dead"
	case "dead":
		return "dead"
	default:
		return "dead"
	}
}

// extractLastActivityFromMessages extracts the last meaningful activity from messages.
// It looks for the most recent assistant message and extracts a summary of what
// the agent is doing (tool use, text generation, etc.).
// Returns nil if no activity can be extracted.
func extractLastActivityFromMessages(messages []opencode.Message) *opencode.LastActivity {
	if len(messages) == 0 {
		return nil
	}

	// Find the last assistant message (most relevant for activity)
	var lastAssistantMsg *opencode.Message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Info.Role == "assistant" {
			lastAssistantMsg = &messages[i]
			break
		}
	}

	if lastAssistantMsg == nil {
		return nil
	}

	// Extract activity from message parts
	// Priority: tool invocation > text > reasoning
	var activityText string
	for _, part := range lastAssistantMsg.Parts {
		switch part.Type {
		case "tool-invocation", "tool":
			// Tool use is the most informative activity
			activityText = "Using tool"
			if part.Text != "" {
				// Truncate tool text for display
				toolText := part.Text
				if len(toolText) > 40 {
					toolText = toolText[:40] + "..."
				}
				activityText = "Using tool: " + toolText
			}
		case "text":
			if part.Text != "" && activityText == "" {
				// Truncate long text
				text := part.Text
				if len(text) > 80 {
					// Find last space before 80 chars
					cutoff := 77
					for i := cutoff; i > 0; i-- {
						if text[i] == ' ' {
							cutoff = i
							break
						}
					}
					text = text[:cutoff] + "..."
				}
				activityText = text
			}
		case "reasoning":
			if activityText == "" {
				activityText = "Thinking..."
			}
		}
	}

	if activityText == "" {
		return nil
	}

	// Use message completion time if available, otherwise created time
	timestamp := lastAssistantMsg.Info.Time.Completed
	if timestamp == 0 {
		timestamp = lastAssistantMsg.Info.Time.Created
	}

	return &opencode.LastActivity{
		Text:      activityText,
		Timestamp: timestamp,
	}
}

// completionBacklogLastEmit tracks the last time we wrote completion_backlog metrics.
// Rate-limits metric writes to at most once per 5 minutes to avoid spamming the
// metrics file from dashboard polls (every 30s).
var completionBacklogLastEmit time.Time

// globalStallTracker tracks token progress for stall detection.
// Agents that are running but making no token progress for N minutes are flagged as stalled.
// This catches agents stuck in infinite loops, crashed during tool execution, etc.
var globalStallTracker = daemon.NewStallTracker(3 * time.Minute)

// emitCompletionBacklogMetrics detects agents stuck in Phase: Complete and writes
// completion_backlog metrics to coaching-metrics.jsonl.
// Rate-limited: writes at most once per 5 minutes.
func emitCompletionBacklogMetrics(agents []AgentAPIResponse, phaseReportedAtMap map[string]time.Time) {
	// Rate limit: only emit once per 5 minutes
	if time.Since(completionBacklogLastEmit) < 5*time.Minute {
		return
	}

	// Build AgentInfo slice for detection
	var agentInfos []orchpkg.AgentInfo
	for _, a := range agents {
		if a.BeadsID == "" {
			continue
		}
		reportedAt, ok := phaseReportedAtMap[a.BeadsID]
		if !ok {
			continue
		}
		agentInfos = append(agentInfos, orchpkg.AgentInfo{
			BeadsID:         a.BeadsID,
			SessionID:       a.SessionID,
			Phase:           a.Phase,
			PhaseReportedAt: reportedAt,
			Status:          a.Status,
		})
	}

	backlog := orchpkg.DetectCompletionBacklog(agentInfos, 10*time.Minute)
	if len(backlog) == 0 {
		return
	}

	// Build a lookup for session IDs
	sessionMap := make(map[string]string)
	for _, a := range agents {
		if a.BeadsID != "" && a.SessionID != "" {
			sessionMap[a.BeadsID] = a.SessionID
		}
	}

	metricsPath := coaching.DefaultMetricsPath()
	if metricsPath == "" {
		return
	}

	for _, beadsID := range backlog {
		details := map[string]interface{}{
			"beads_id": beadsID,
		}
		if sid, ok := sessionMap[beadsID]; ok {
			details["session_id"] = sid
		}
		if reportedAt, ok := phaseReportedAtMap[beadsID]; ok {
			details["completed_at"] = reportedAt.Format(time.RFC3339)
			details["wait_minutes"] = int(time.Since(reportedAt).Minutes())
		}

		m := coaching.Metric{
			Timestamp: time.Now().Format(time.RFC3339),
			SessionID: sessionMap[beadsID],
			Type:      "completion_backlog",
			Value:     float64(len(backlog)),
			Details:   details,
		}
		// Best-effort write; don't fail the API response on metric write errors
		_ = coaching.WriteMetric(metricsPath, m)
	}

	completionBacklogLastEmit = time.Now()
}
