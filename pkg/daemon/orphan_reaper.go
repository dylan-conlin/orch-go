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
	// LedgerSwept is the number of stale entries removed from the process ledger.
	LedgerSwept int
	// Error is set if the operation encountered an error.
	Error error
	// Message is a human-readable summary.
	Message string
}

// ReapOrphanProcesses finds and kills agent processes that are no longer associated
// with any active OpenCode session. Uses a two-tier approach:
//
// Tier 1 (primary): Ledger-backed ownership verification — reconciles the process
// ledger against active session IDs and kills stale entries.
//
// Tier 2 (fallback): Title-string matching via process.FindOrphanProcesses — catches
// processes spawned before the ledger existed or not recorded in the ledger.
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

	// Get active sessions from OpenCode API (both IDs and titles)
	activeIDs, activeTitles, err := getActiveSessionInfo(d.Config.CleanupServerURL)
	if err != nil {
		return &OrphanReapResult{
			Error:   fmt.Errorf("failed to get active sessions: %w", err),
			Message: fmt.Sprintf("Failed to get active sessions: %v", err),
		}
	}

	result := &OrphanReapResult{}

	// Tier 1: Ledger-backed sweep
	ledger := process.NewLedger(process.DefaultLedgerPath())
	sweepResult := ledger.SweepWithKill(activeIDs)
	result.LedgerSwept = sweepResult.StaleRemoved
	result.Killed += sweepResult.Killed
	result.Found += sweepResult.StaleRemoved

	// Tier 2: Title-based fallback for processes not in the ledger
	orphans, err := process.FindOrphanProcesses(activeTitles)
	if err != nil {
		// Non-fatal: tier 1 may have already handled most cases
		if d.Config.Verbose {
			fmt.Printf("  Orphan reaper: title-based fallback failed: %v\n", err)
		}
	} else {
		for _, orphan := range orphans {
			if process.Terminate(orphan.PID, "bun (orphan, title-fallback)") {
				result.Killed++
				result.Found++
				if d.Config.Verbose {
					name := orphan.WorkspaceName
					if name == "" {
						name = "(unknown)"
					}
					beadsInfo := ""
					if orphan.BeadsID != "" {
						beadsInfo = fmt.Sprintf(" [%s]", orphan.BeadsID)
					}
					fmt.Printf("  Orphan reaper: killed PID %d (%s%s) via title-fallback\n", orphan.PID, name, beadsInfo)
				}
			}
		}
	}

	if result.Found == 0 {
		result.Message = "No orphan processes found"
	} else {
		result.Message = fmt.Sprintf("Reaped %d/%d orphan processes (ledger: %d swept)", result.Killed, result.Found, result.LedgerSwept)
	}

	return result
}

// getActiveSessionInfo queries the OpenCode API and returns both:
//   - activeIDs: a set of session IDs (for ledger-backed verification)
//   - activeTitles: a set of session titles and workspace names (for title-based fallback)
func getActiveSessionInfo(serverURL string) (activeIDs map[string]bool, activeTitles map[string]bool, err error) {
	if serverURL == "" {
		serverURL = "http://127.0.0.1:4096"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(serverURL + "/session")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query OpenCode sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("OpenCode sessions API returned status %d", resp.StatusCode)
	}

	var sessions []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, nil, fmt.Errorf("failed to decode sessions response: %w", err)
	}

	activeIDs = make(map[string]bool, len(sessions))
	activeTitles = make(map[string]bool, len(sessions)*2)
	for _, s := range sessions {
		if s.ID != "" {
			activeIDs[s.ID] = true
		}
		if s.Title != "" {
			activeTitles[s.Title] = true
			// Also extract workspace name from title (format: "workspace-name [beads-id]")
			if idx := strings.Index(s.Title, " ["); idx != -1 {
				activeTitles[strings.TrimSpace(s.Title[:idx])] = true
			}
		}
	}

	return activeIDs, activeTitles, nil
}

// getActiveSessionTitles is a convenience wrapper that returns only titles.
// Kept for backward compatibility with tests.
func getActiveSessionTitles(serverURL string) (map[string]bool, error) {
	_, titles, err := getActiveSessionInfo(serverURL)
	return titles, err
}
