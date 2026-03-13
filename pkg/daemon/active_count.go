// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// DefaultActiveCount returns the number of active agents by querying OpenCode API.
// Counts only recently-active sessions (updated within the last 30 minutes) to avoid
// counting stale sessions that persist indefinitely in OpenCode.
// Excludes sessions whose beads issues are already closed (completed agents).
func DefaultActiveCount() int {
	// Use OpenCode API to count active sessions
	// The default server URL is used; this works because the daemon runs
	// on the same machine as OpenCode server.
	serverURL := os.Getenv("OPENCODE_URL")
	if serverURL == "" {
		serverURL = "http://127.0.0.1:4096"
	}

	// Make HTTP request to list sessions
	resp, err := http.Get(serverURL + "/session")
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var sessions []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Time  struct {
			Updated int64 `json:"updated"` // Unix timestamp in milliseconds
		} `json:"time"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return 0
	}

	// Only count sessions that have been active recently.
	// OpenCode sessions persist indefinitely (including old test sessions),
	// so we filter to sessions updated within the last 30 minutes.
	// This matches the same threshold used in orch status for agent matching.
	const maxIdleTime = 30 * time.Minute
	now := time.Now()

	// Collect beads IDs for batch lookup
	var recentBeadsIDs []string
	beadsIDToSession := make(map[string]bool)
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) > maxIdleTime {
			continue
		}

		// Extract beads ID from title (format: "workspace-name [beads-id]")
		beadsID := extractBeadsIDFromSessionTitle(s.Title)
		if beadsID == "" {
			continue
		}

		recentBeadsIDs = append(recentBeadsIDs, beadsID)
		beadsIDToSession[beadsID] = true
	}

	// If no recent sessions, return early
	if len(recentBeadsIDs) == 0 {
		return 0
	}

	// Batch fetch issue status to check if closed
	// This prevents counting completed agents (beads issue closed but session still exists)
	closedIssues := GetClosedIssuesBatch(recentBeadsIDs)

	// Count sessions with open issues only
	activeCount := 0
	for beadsID := range beadsIDToSession {
		if closedIssues[beadsID] {
			// Issue is closed, don't count this session
			continue
		}
		activeCount++
	}

	return activeCount
}

// GetClosedIssuesBatch checks which beads IDs have closed or done issues.
// Returns a map of beadsID -> true for issues that should NOT count as active.
// An issue is considered "not active" if:
//   - Its status is "closed"
//   - It has a daemon:verification-failed label (verification exhausted, deferred for human review)
//   - It has a daemon:ready-review label (verification passed, waiting for orchestrator review)
//
// Uses beads RPC daemon for efficiency, falls back to CLI if needed.
// Exported for use by checkConcurrencyLimit in spawn_cmd.go.
func GetClosedIssuesBatch(beadsIDs []string) map[string]bool {
	closed := make(map[string]bool)
	if len(beadsIDs) == 0 {
		return closed
	}

	// Try beads RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(2))
		if err := client.Connect(); err == nil {
			defer client.Close()
			// Check each issue status and labels
			for _, id := range beadsIDs {
				issue, err := client.Show(id)
				if err != nil {
					// Issue not found in local project beads — treat as not active.
					// This handles cross-project beads IDs (e.g., skillc-cb3 queried
					// against orch-go beads) and deleted issues.
					closed[id] = true
					continue
				}
				if isIssueDone(issue.Status, issue.Labels) {
					closed[id] = true
				}
			}
			return closed
		}
	}

	// Fallback to CLI for each issue
	for _, id := range beadsIDs {
		issue, err := beads.FallbackShow(id, "")
		if err != nil {
			// Issue not found — treat as not active (same as RPC path above).
			closed[id] = true
			continue
		}
		if isIssueDone(issue.Status, issue.Labels) {
			closed[id] = true
		}
	}

	return closed
}

// isIssueDone returns true if an issue should not count as an active agent.
// Checks both status (closed) and labels (daemon:verification-failed, daemon:ready-review).
func isIssueDone(status string, labels []string) bool {
	if strings.EqualFold(status, "closed") {
		return true
	}
	for _, label := range labels {
		if label == LabelVerificationFailed || label == LabelReadyReview {
			return true
		}
	}
	return false
}

// extractBeadsIDFromSessionTitle extracts beads ID from an OpenCode session title.
// Session titles follow format: "workspace-name [beads-id]" (e.g., "og-feat-add-feature-24dec [orch-go-3anf]")
func extractBeadsIDFromSessionTitle(title string) string {
	// Look for "[beads-id]" pattern at the end
	if start := strings.LastIndex(title, "["); start != -1 {
		if end := strings.LastIndex(title, "]"); end != -1 && end > start {
			return strings.TrimSpace(title[start+1 : end])
		}
	}
	return ""
}

// CountActiveTmuxAgents returns beads IDs of orch-managed agents running in tmux windows.
// This complements DefaultActiveCount() which only counts OpenCode sessions.
// Claude CLI backend agents run in tmux windows WITHOUT OpenCode sessions,
// making them invisible to DefaultActiveCount().
//
// Only counts windows where the pane has an active (non-shell) process.
// Windows where the agent process has exited (leaving only an idle shell) are
// excluded to prevent ghost slots in the worker pool.
//
// If projectName is non-empty, only scans the workers-{projectName} session
// to avoid cross-project inflation (e.g., skillc agents inflating orch-go count).
// If empty, scans all workers-* sessions (legacy behavior).
//
// Always scans orchestrator and meta-orchestrator sessions (they are global).
func CountActiveTmuxAgents(projectName string) map[string]bool {
	activeBeadsIDs := make(map[string]bool)

	// Gather tmux sessions to search
	var sessions []string
	if projectName != "" {
		// Scoped mode: only scan this project's worker session
		projectSession := tmux.GetWorkersSessionName(projectName)
		if tmux.SessionExists(projectSession) {
			sessions = append(sessions, projectSession)
		}
	} else {
		// Legacy mode: scan all worker sessions
		var err error
		sessions, err = tmux.ListWorkersSessions()
		if err != nil {
			// Fail-open: if tmux isn't running, return empty
			return activeBeadsIDs
		}
	}

	// Also search orchestrator and meta-orchestrator sessions (global)
	if tmux.SessionExists(tmux.OrchestratorSessionName) {
		sessions = append(sessions, tmux.OrchestratorSessionName)
	}
	if tmux.SessionExists(tmux.MetaOrchestratorSessionName) {
		sessions = append(sessions, tmux.MetaOrchestratorSessionName)
	}

	// List windows in each session and extract beads IDs
	for _, sessionName := range sessions {
		windows, err := tmux.ListWindows(sessionName)
		if err != nil {
			continue // Skip sessions that fail
		}
		for _, w := range windows {
			beadsID := extractBeadsIDFromWindowName(w.Name)
			if beadsID == "" {
				continue
			}
			// Only count windows with active processes.
			// Tmux windows persist after agent process exits, leaving an idle shell.
			// Without this check, dead windows inflate the active count and create
			// ghost slots that block all spawning.
			if !tmux.IsPaneActive(w.ID) {
				continue
			}
			activeBeadsIDs[beadsID] = true
		}
	}

	return activeBeadsIDs
}

// BeadsActiveCount returns the number of active orch-managed agents by querying
// beads for in_progress issues with the orch:agent label. This is the capacity
// source (replacing infrastructure scanning) — beads is the authoritative state
// machine for agent lifecycle, so querying it directly eliminates ghost slot bugs
// caused by tmux window scanning (dead panes, child windows, cross-project inflation).
//
// Issues with daemon:verification-failed or daemon:ready-review labels are excluded
// (they represent completed agents awaiting review, not active capacity consumers).
//
// Uses RPC client first, falls back to bd CLI.
func BeadsActiveCount() int {
	const orchAgentLabel = "orch:agent"

	// Try RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(2))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issues, err := client.List(&beads.ListArgs{
				Status:    "in_progress",
				LabelsAny: []string{orchAgentLabel},
				Limit:     beads.IntPtr(0),
			})
			if err == nil {
				count := 0
				for _, issue := range issues {
					if !isIssueDone(issue.Status, issue.Labels) {
						count++
					}
				}
				return count
			}
		}
	}

	// Fallback: bd CLI
	issues, err := beads.FallbackListWithLabel(orchAgentLabel, "")
	if err != nil {
		return 0 // Fail-open: if beads is unreachable, report 0
	}

	count := 0
	for _, issue := range issues {
		if issue.Status != "in_progress" {
			continue
		}
		if !isBeadsIssueDone(issue.Labels) {
			count++
		}
	}
	return count
}

// isBeadsIssueDone checks if a beads issue's labels indicate it should not count
// as active capacity. Used by BeadsActiveCount for the CLI fallback path where
// we have beads.Issue (not daemon.Issue).
func isBeadsIssueDone(labels []string) bool {
	for _, label := range labels {
		if label == LabelVerificationFailed || label == LabelReadyReview {
			return true
		}
	}
	return false
}

// DiscoverLiveAgents returns the set of beads IDs with live infrastructure
// (OpenCode sessions or tmux windows with active processes), deduplicated.
// This is NOT a capacity source — use BeadsActiveCount() for capacity decisions.
//
// Use this for liveness checks and orphan detection: comparing live infrastructure
// against beads state to find agents that are running but shouldn't be (orphans)
// or should be running but aren't (stalled).
//
// Tmux scanning is scoped to the current project (derived from cwd)
// to prevent cross-project agents from appearing.
//
// Agents whose beads issues are closed are excluded from results.
func DiscoverLiveAgents() map[string]bool {
	activeBeadsIDs := make(map[string]bool)

	// Derive project name from cwd for scoped tmux scanning.
	// The daemon and spawn commands always run from the project directory.
	projectName := ""
	if wd, err := os.Getwd(); err == nil {
		projectName = filepath.Base(wd)
	}

	// Source 1: OpenCode sessions (headless backend)
	serverURL := os.Getenv("OPENCODE_URL")
	if serverURL == "" {
		serverURL = "http://127.0.0.1:4096"
	}
	resp, err := http.Get(serverURL + "/session")
	if err == nil {
		defer resp.Body.Close()
		var sessions []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Time  struct {
				Updated int64 `json:"updated"`
			} `json:"time"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&sessions); err == nil {
			const maxIdleTime = 30 * time.Minute
			now := time.Now()
			for _, s := range sessions {
				updatedAt := time.Unix(s.Time.Updated/1000, 0)
				if now.Sub(updatedAt) > maxIdleTime {
					continue
				}
				beadsID := extractBeadsIDFromSessionTitle(s.Title)
				if beadsID == "" {
					continue
				}
				activeBeadsIDs[beadsID] = true
			}
		}
	}

	// Source 2: Tmux windows (Claude CLI backend)
	// Scoped to current project to prevent cross-project inflation.
	tmuxAgents := CountActiveTmuxAgents(projectName)
	for beadsID := range tmuxAgents {
		activeBeadsIDs[beadsID] = true // Deduplicated by map key
	}

	// If no agents found from either source, return empty
	if len(activeBeadsIDs) == 0 {
		return activeBeadsIDs
	}

	// Exclude agents whose beads issues are closed
	var allIDs []string
	for id := range activeBeadsIDs {
		allIDs = append(allIDs, id)
	}
	closedIssues := GetClosedIssuesBatch(allIDs)

	for id := range activeBeadsIDs {
		if closedIssues[id] {
			delete(activeBeadsIDs, id)
		}
	}

	return activeBeadsIDs
}
