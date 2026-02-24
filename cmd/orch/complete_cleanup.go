package main

import (
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// cleanupTmuxWindow finds and kills the tmux window associated with an agent.
// For orchestrator sessions, searches by workspace name.
// For regular agents, searches by beads ID (or identifier as fallback).
// This is idempotent - if no window is found, it's a no-op.
func cleanupTmuxWindow(isOrchestratorSession bool, agentName, beadsID, identifier string) {
	var window *tmux.WindowInfo
	var tmuxSessionName string
	var findErr error

	if isOrchestratorSession {
		window, tmuxSessionName, findErr = tmux.FindWindowByWorkspaceNameAllSessions(agentName)
	} else {
		var windowSearchID string
		if beadsID != "" {
			windowSearchID = beadsID
		} else {
			windowSearchID = identifier
		}
		window, tmuxSessionName, findErr = tmux.FindWindowByBeadsIDAllSessions(windowSearchID)
	}

	if findErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to search for tmux window: %v\n", findErr)
		return
	}

	if window != nil {
		if err := tmux.KillWindow(window.Target); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close tmux window %s: %v\n", window.Target, err)
		} else {
			fmt.Printf("Closed tmux window: %s:%s\n", tmuxSessionName, window.Name)
		}
	}
}
