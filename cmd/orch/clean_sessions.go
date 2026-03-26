// Package main provides tmux session/window cleanup functions for the clean command.
// Extracted from clean_cmd.go for cohesion (stale tmux window detection and cleanup).
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/execution"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// PaneProcessChecker determines if a tmux window's pane has an active (non-shell) process.
// This is used as a last-resort safety check before classifying a window as stale.
type PaneProcessChecker interface {
	HasActiveProcess(windowID string) bool
}

// DefaultPaneProcessChecker delegates to tmux.IsPaneActive for process liveness detection.
// Uses two signals: pane_current_command and child process detection.
// See tmux.IsPaneActive for implementation details.
type DefaultPaneProcessChecker struct{}

// HasActiveProcess checks if a tmux window has an active agent process.
// Delegates to tmux.IsPaneActive which uses pane_current_command + child process detection.
func (c *DefaultPaneProcessChecker) HasActiveProcess(windowID string) bool {
	return tmux.IsPaneActive(windowID)
}

// staleTmuxWindow represents a tmux window identified as stale.
type staleTmuxWindow struct {
	window      *tmux.WindowInfo
	sessionName string
	beadsID     string
}

// classifyTmuxWindows identifies tmux windows that are stale (no active OpenCode session,
// no open beads issue, and no running process). This protects daemon-spawned Claude CLI
// agents that have tmux windows but no corresponding OpenCode sessions.
//
// The process checker is the last-resort safety net: even if OpenCode has zero sessions
// (e.g., server restarted) and the beads issue is closed, a window with a running agent
// process (claude, opencode, etc.) will be protected from cleanup.
func classifyTmuxWindows(
	windows []tmux.WindowInfo,
	sessionName string,
	activeBeadsIDs map[string]bool,
	openIssues map[string]*verify.Issue,
	processChecker PaneProcessChecker,
) (stale []staleTmuxWindow, protected int) {
	for _, w := range windows {
		// Skip known non-agent windows
		if w.Name == "servers" || w.Name == "zsh" {
			continue
		}

		beadsID := extractBeadsIDFromWindowName(w.Name)
		if beadsID == "" {
			continue
		}

		// Has an active OpenCode session? Not stale.
		if activeBeadsIDs[beadsID] {
			continue
		}

		// Has an open beads issue? Protected (likely Claude CLI daemon spawn).
		if _, isOpen := openIssues[beadsID]; isOpen {
			protected++
			continue
		}

		// Has an active process running in the pane? Protected.
		// This catches Claude CLI agents that have no OpenCode session and whose
		// beads issue may be closed, but are still actively running.
		if processChecker != nil && processChecker.HasActiveProcess(w.ID) {
			protected++
			continue
		}

		// Window is truly stale (no OpenCode session AND beads issue not open AND no active process)
		windowCopy := w
		stale = append(stale, staleTmuxWindow{&windowCopy, sessionName, beadsID})
	}
	return stale, protected
}

// cleanStaleTmuxWindows finds and closes tmux windows with no active OpenCode session backing them.
// Windows with open beads issues are protected (handles Claude CLI daemon spawns).
// If preserveOrchestrator is true, windows in orchestrator/meta-orchestrator sessions are skipped.
// Returns the number of windows closed and any error encountered.
func cleanStaleTmuxWindows(serverURL string, projectDir string, dryRun bool, preserveOrchestrator bool) (int, error) {
	client := execution.NewOpenCodeAdapter(serverURL)
	now := time.Now()
	const maxIdleTime = 30 * time.Minute

	fmt.Println("\nScanning for stale tmux windows...")

	// Get all OpenCode sessions and build a map of recently active beads IDs
	sessions, err := client.ListSessions(context.Background(), "")
	if err != nil {
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	activeBeadsIDs := make(map[string]bool)
	for _, s := range sessions {
		updatedAt := s.Updated
		if now.Sub(updatedAt) <= maxIdleTime {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID != "" {
				activeBeadsIDs[beadsID] = true
			}
		}
	}

	fmt.Printf("  Found %d active OpenCode sessions\n", len(activeBeadsIDs))

	// Get open beads issues to protect active agents without OpenCode sessions
	// (e.g., Claude CLI daemon spawns)
	openIssues, err := verify.ListOpenIssuesWithDir(projectDir)
	if err != nil {
		return 0, fmt.Errorf("failed to check beads issues (needed to protect active agents): %w", err)
	}

	// Scan all workers sessions and classify windows
	var allStale []staleTmuxWindow
	totalProtected := 0

	processChecker := &DefaultPaneProcessChecker{}

	skippedOrch := 0
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		// Skip orchestrator and meta-orchestrator sessions entirely
		if preserveOrchestrator && (sessionName == tmux.OrchestratorSessionName || sessionName == tmux.MetaOrchestratorSessionName) {
			skippedOrch++
			continue
		}

		windows, err := tmux.ListWindows(sessionName)
		if err != nil {
			continue
		}

		stale, protected := classifyTmuxWindows(windows, sessionName, activeBeadsIDs, openIssues, processChecker)
		allStale = append(allStale, stale...)
		totalProtected += protected
	}

	if skippedOrch > 0 {
		fmt.Printf("  Skipped %d orchestrator sessions (--preserve-orchestrator)\n", skippedOrch)
	}

	if totalProtected > 0 {
		fmt.Printf("  Protected %d windows with open beads issues (no OpenCode session)\n", totalProtected)
	}

	if len(allStale) == 0 {
		fmt.Println("  No stale tmux windows found")
		return 0, nil
	}

	fmt.Printf("  Found %d stale tmux windows:\n", len(allStale))

	// Close stale windows
	closed := 0
	for _, pw := range allStale {
		if dryRun {
			fmt.Printf("    [DRY-RUN] Would close: %s:%s\n", pw.sessionName, pw.window.Name)
			closed++
			continue
		}

		if err := tmux.KillWindowByID(pw.window.ID); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to close %s (%s): %v\n", pw.window.Name, pw.window.ID, err)
			continue
		}

		fmt.Printf("    Closed: %s:%s (%s)\n", pw.sessionName, pw.window.Name, pw.window.ID)
		closed++
	}

	return closed, nil
}
