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

func cleanStaleTmuxWindows(serverURL string, preserveOrchestrator bool) (int, error) {
	client := opencode.NewClient(serverURL)
	now := time.Now()

	sessions, err := client.ListSessions("")
	if err != nil {
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	activeBeadsIDs := make(map[string]bool)
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) <= cleanupMaxIdleTime {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID != "" {
				activeBeadsIDs[beadsID] = true
			}
		}
	}

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

			if !activeBeadsIDs[beadsID] {
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
