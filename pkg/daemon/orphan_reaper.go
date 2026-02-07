// orphan_reaper.go contains periodic orphan process detection and cleanup.
package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/process"
)

// OrphanReapResult contains the result of an orphan reap operation.
type OrphanReapResult struct {
	// Found is the number of orphan processes detected.
	Found int
	// Killed is the number of orphan processes successfully terminated.
	Killed int
	// Error is set if the operation encountered an error.
	Error error
	// Message is a human-readable summary.
	Message string
}

// ReapOrphanProcesses finds and kills bun agent processes that are not associated
// with any active OpenCode session. This prevents resource leaks from agents whose
// sessions have ended but whose bun processes remain running.
//
// The method queries the OpenCode API for active sessions, then uses
// process.FindOrphanProcesses to identify bun processes not matching any session.
// Orphan processes are terminated via SIGTERM (with SIGKILL fallback).
//
// Returns nil if orphan reaping is not due (based on interval tracking).
func (d *Daemon) ReapOrphanProcesses() *OrphanReapResult {
	if !d.Config.OrphanReapEnabled {
		return nil
	}

	// Check if enough time has passed since last reap
	if !d.lastOrphanReap.IsZero() && time.Since(d.lastOrphanReap) < d.Config.OrphanReapInterval {
		return nil
	}

	d.lastOrphanReap = time.Now()

	// Get active session titles from OpenCode API
	activeTitles, err := getActiveSessionTitles(d.Config.CleanupServerURL)
	if err != nil {
		return &OrphanReapResult{
			Error:   fmt.Errorf("failed to get active sessions: %w", err),
			Message: fmt.Sprintf("Failed to get active sessions: %v", err),
		}
	}

	// Find orphan processes
	orphans, err := process.FindOrphanProcesses(activeTitles)
	if err != nil {
		return &OrphanReapResult{
			Error:   fmt.Errorf("failed to find orphan processes: %w", err),
			Message: fmt.Sprintf("Failed to find orphan processes: %v", err),
		}
	}

	if len(orphans) == 0 {
		return &OrphanReapResult{
			Message: "No orphan processes found",
		}
	}

	// Kill orphan processes
	killed := 0
	for _, orphan := range orphans {
		if process.Terminate(orphan.PID, "bun (orphan)") {
			killed++
			if d.Config.Verbose {
				name := orphan.WorkspaceName
				if name == "" {
					name = "(unknown)"
				}
				beadsInfo := ""
				if orphan.BeadsID != "" {
					beadsInfo = fmt.Sprintf(" [%s]", orphan.BeadsID)
				}
				fmt.Printf("  Orphan reaper: killed PID %d (%s%s)\n", orphan.PID, name, beadsInfo)
			}
		}
	}

	return &OrphanReapResult{
		Found:   len(orphans),
		Killed:  killed,
		Message: fmt.Sprintf("Reaped %d/%d orphan processes", killed, len(orphans)),
	}
}

// getActiveSessionTitles queries the OpenCode API and returns a set of active
// session titles (workspace names and beads IDs) for orphan detection.
func getActiveSessionTitles(serverURL string) (map[string]bool, error) {
	if serverURL == "" {
		serverURL = "http://127.0.0.1:4096"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(serverURL + "/session")
	if err != nil {
		return nil, fmt.Errorf("failed to query OpenCode sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenCode sessions API returned status %d", resp.StatusCode)
	}

	var sessions []struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions response: %w", err)
	}

	activeTitles := make(map[string]bool)
	for _, s := range sessions {
		title := s.Title
		if title == "" {
			continue
		}
		activeTitles[title] = true
		// Also extract workspace name from title (format: "workspace-name [beads-id]")
		if idx := strings.Index(title, " ["); idx != -1 {
			activeTitles[strings.TrimSpace(title[:idx])] = true
		}
	}

	return activeTitles, nil
}
