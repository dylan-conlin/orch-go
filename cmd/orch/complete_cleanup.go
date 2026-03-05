package main

import (
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// cleanupTmuxWindow finds and kills ALL tmux windows associated with an agent.
// For orchestrator sessions, searches by workspace name.
// For regular agents, searches by beads ID (or identifier as fallback).
// This is idempotent - if no windows are found, it's a no-op.
func cleanupTmuxWindow(isOrchestratorSession bool, agentName, beadsID, identifier string) {
	var matches []tmux.WindowMatch
	var findErr error

	if isOrchestratorSession {
		matches, findErr = tmux.FindAllWindowsByWorkspaceNameAllSessions(agentName)
	} else {
		var windowSearchID string
		if beadsID != "" {
			windowSearchID = beadsID
		} else {
			windowSearchID = identifier
		}
		matches, findErr = tmux.FindAllWindowsByBeadsIDAllSessions(windowSearchID)
	}

	if findErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to search for tmux windows: %v\n", findErr)
		return
	}

	for _, m := range matches {
		if err := tmux.KillWindowByID(m.Window.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close tmux window %s (%s): %v\n", m.Window.ID, m.Window.Name, err)
		} else {
			fmt.Printf("Closed tmux window: %s:%s (%s)\n", m.SessionName, m.Window.Name, m.Window.ID)
		}
	}
}
