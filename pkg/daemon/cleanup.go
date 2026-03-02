package daemon

import (
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

const cleanupMaxIdleTime = 30 * time.Minute

func defaultCleanup(config Config) (int, string, error) {
	closed, err := cleanStaleTmuxWindows(config.CleanupServerURL, config.CleanupPreserveOrchestrator)
	if err != nil {
		return 0, fmt.Sprintf("Cleanup failed: %v", err), err
	}
	if closed == 0 {
		return 0, "No stale tmux windows found", nil
	}
	return closed, fmt.Sprintf("Closed %d stale tmux windows", closed), nil
}

// beadsStatusFunc is a function that returns the beads issue status for a given ID.
// Extracted for testability.
type beadsStatusFunc func(beadsID string) (string, error)

// isWindowStale determines whether a tmux window should be cleaned up.
// A window is stale if:
//  1. Its beads ID is not active in any OpenCode session, AND
//  2. Its beads issue status is NOT in_progress or open (i.e., it's closed/completed)
//
// If beads status cannot be determined, the window is kept alive (fail-safe).
// This protects Claude CLI workers which run in tmux without OpenCode sessions.
func isWindowStale(beadsID string, activeBeadsIDs map[string]bool, getStatus beadsStatusFunc) bool {
	// Skip if active in OpenCode (headless/tmux backend)
	if activeBeadsIDs[beadsID] {
		return false
	}

	// Check beads issue status (protects Claude CLI workers).
	// Claude CLI workers run in tmux without OpenCode sessions, so they
	// would always appear "stale" if we only checked OpenCode. Querying
	// beads status ensures we don't kill workers that are still in_progress.
	// Fail-safe: if we can't determine status, keep the window alive.
	status, err := getStatus(beadsID)
	if err != nil {
		// Can't determine status — keep window alive (fail-safe)
		return false
	}
	if status == "in_progress" || status == "open" {
		// Issue is still active — agent is working
		return false
	}

	return true
}

func cleanStaleTmuxWindows(serverURL string, preserveOrchestrator bool) (int, error) {
	client := opencode.NewClient(serverURL)
	now := time.Now()

	// Source 1: OpenCode sessions (headless/tmux backend)
	activeBeadsIDs := make(map[string]bool)
	sessions, err := client.ListSessions("")
	if err == nil {
		for _, s := range sessions {
			updatedAt := time.Unix(s.Time.Updated/1000, 0)
			if now.Sub(updatedAt) <= cleanupMaxIdleTime {
				beadsID := extractBeadsIDFromTitle(s.Title)
				if beadsID != "" {
					activeBeadsIDs[beadsID] = true
				}
			}
		}
	}
	// If OpenCode is unavailable, activeBeadsIDs is empty — that's OK because
	// isWindowStale also checks beads issue status (protects Claude CLI workers).

	workersSessions, _ := tmux.ListWorkersSessions()
	var staleWindows []tmux.WindowInfo
	for _, sessionName := range workersSessions {
		if preserveOrchestrator && (sessionName == tmux.OrchestratorSessionName || sessionName == tmux.MetaOrchestratorSessionName) {
			continue
		}

		windows, err := tmux.ListWindows(sessionName)
		if err != nil {
			continue
		}

		for _, w := range windows {
			if w.Name == "servers" || w.Name == "zsh" {
				continue
			}

			beadsID := extractBeadsIDFromWindowName(w.Name)
			if beadsID == "" {
				continue
			}

			if isWindowStale(beadsID, activeBeadsIDs, GetBeadsIssueStatus) {
				staleWindows = append(staleWindows, w)
			}
		}
	}

	closed := 0
	for _, w := range staleWindows {
		if err := tmux.KillWindow(w.Target); err != nil {
			continue
		}
		closed++
	}

	return closed, nil
}

func extractBeadsIDFromTitle(title string) string {
	start := strings.LastIndex(title, "[")
	end := strings.LastIndex(title, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(title[start+1 : end])
}

func extractBeadsIDFromWindowName(name string) string {
	start := strings.LastIndex(name, "[")
	end := strings.LastIndex(name, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(name[start+1 : end])
}
