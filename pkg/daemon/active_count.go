// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// DefaultActiveCount returns the number of active agents by querying OpenCode API.
// Counts only recently-active sessions (updated within the last 30 minutes) to avoid
// counting stale sessions that persist indefinitely in OpenCode.
// Excludes untracked agents (spawned with --no-track) which have "-untracked-" in their beads ID.
// Excludes sessions whose beads issues are already closed (completed agents).
func DefaultActiveCount() int {
	// Use OpenCode API to count active sessions
	// The default server URL is used; this works because the daemon runs
	// on the same machine as OpenCode server.
	serverURL := os.Getenv("OPENCODE_URL")
	if serverURL == "" {
		serverURL = "http://localhost:4096"
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
		// Skip untracked agents which have "-untracked-" in their beads ID.
		// These are ad-hoc spawns that shouldn't count against daemon capacity.
		beadsID := extractBeadsIDFromSessionTitle(s.Title)
		if beadsID == "" || isUntrackedBeadsID(beadsID) {
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
	closedIssues := getClosedIssuesBatch(recentBeadsIDs)

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

// getClosedIssuesBatch checks which beads IDs have closed issues.
// Returns a map of beadsID -> true for closed issues.
// Uses beads RPC daemon for efficiency, falls back to CLI if needed.
func getClosedIssuesBatch(beadsIDs []string) map[string]bool {
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
			// Check each issue status
			for _, id := range beadsIDs {
				issue, err := client.Show(id)
				if err != nil {
					// If we can't find the issue, assume it's not running
					// (might have been deleted or never existed)
					continue
				}
				if strings.EqualFold(issue.Status, "closed") {
					closed[id] = true
				}
			}
			return closed
		}
	}

	// Fallback to CLI for each issue
	for _, id := range beadsIDs {
		issue, err := beads.FallbackShow(id)
		if err != nil {
			continue
		}
		if strings.EqualFold(issue.Status, "closed") {
			closed[id] = true
		}
	}

	return closed
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

// isUntrackedBeadsID returns true if the beads ID indicates an untracked agent.
// Untracked agents are spawned with --no-track and have IDs like "project-untracked-1766695797".
func isUntrackedBeadsID(beadsID string) bool {
	return strings.Contains(beadsID, "-untracked-")
}
