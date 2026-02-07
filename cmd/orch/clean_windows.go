// Package main provides tmux window cleanup for the clean command.
// Extracted from clean_cmd.go for per-concern file organization.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// cleanPhantomWindows finds and closes tmux windows that are phantoms
// (have a beads ID in the window name but no active OpenCode session).
func cleanPhantomWindows(serverURL string, dryRun bool, preserveOrchestrator bool) (int, error) {
	return cleanPhantomWindowsWithClient(opencode.NewClient(serverURL), dryRun, preserveOrchestrator)
}

func cleanPhantomWindowsWithClient(client opencode.ClientInterface, dryRun bool, preserveOrchestrator bool) (int, error) {
	now := time.Now()
	const maxIdleTime = 30 * time.Minute

	fmt.Println("\nScanning for phantom tmux windows...")

	sessions, err := client.ListSessions("")
	if err != nil {
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	activeBeadsIDs := make(map[string]bool)
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) <= maxIdleTime {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID != "" {
				activeBeadsIDs[beadsID] = true
			}
		}
	}

	fmt.Printf("  Found %d active OpenCode sessions\n", len(activeBeadsIDs))

	var phantomWindows []struct {
		window      *tmux.WindowInfo
		sessionName string
		beadsID     string
	}

	skippedOrch := 0
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		if preserveOrchestrator && (sessionName == tmux.OrchestratorSessionName || sessionName == tmux.MetaOrchestratorSessionName) {
			skippedOrch++
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
				windowCopy := w
				phantomWindows = append(phantomWindows, struct {
					window      *tmux.WindowInfo
					sessionName string
					beadsID     string
				}{&windowCopy, sessionName, beadsID})
			}
		}
	}

	if skippedOrch > 0 {
		fmt.Printf("  Skipped %d orchestrator sessions (--preserve-orchestrator)\n", skippedOrch)
	}

	if len(phantomWindows) == 0 {
		fmt.Println("  No phantom windows found")
		return 0, nil
	}

	fmt.Printf("  Found %d phantom windows:\n", len(phantomWindows))

	closed := 0
	for _, pw := range phantomWindows {
		if dryRun {
			fmt.Printf("    [DRY-RUN] Would close: %s:%s\n", pw.sessionName, pw.window.Name)
			closed++
			continue
		}

		if err := tmux.KillWindow(pw.window.Target); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to close %s: %v\n", pw.window.Name, err)
			continue
		}

		fmt.Printf("    Closed: %s:%s\n", pw.sessionName, pw.window.Name)
		closed++
	}

	return closed, nil
}
