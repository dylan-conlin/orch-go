// Package cleanup provides utilities for cleaning up stale OpenCode sessions.
package cleanup

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// CleanStaleSessionsOptions configures the session cleanup behavior.
type CleanStaleSessionsOptions struct {
	// ServerURL is the OpenCode server URL
	ServerURL string
	// StaleDays is the number of days after which a session is considered stale
	StaleDays int
	// DryRun if true, only reports what would be deleted without actually deleting
	DryRun bool
	// PreserveOrchestrator if true, skips orchestrator sessions
	PreserveOrchestrator bool
	// Quiet if true, suppresses progress output (for daemon use)
	Quiet bool
}

// CleanStaleSessions deletes OpenCode sessions older than the specified number of days.
// It skips sessions that are currently active (processing or recently updated).
// If preserveOrchestrator is true, sessions associated with orchestrator workspaces are skipped.
// Returns the number of sessions deleted and any error encountered.
func CleanStaleSessions(opts CleanStaleSessionsOptions) (int, error) {
	if !opts.Quiet {
		fmt.Printf("\nScanning for stale OpenCode sessions (older than %d days)...\n", opts.StaleDays)
	}

	client := opencode.NewClient(opts.ServerURL)

	// Get all in-memory sessions (without x-opencode-directory header)
	sessions, err := client.ListSessions("")
	if err != nil {
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	if !opts.Quiet {
		fmt.Printf("  Found %d total sessions\n", len(sessions))
	}

	// Calculate the cutoff time
	cutoff := time.Now().AddDate(0, 0, -opts.StaleDays)
	cutoffMs := cutoff.UnixMilli()

	// Find stale sessions (not updated since cutoff)
	// Skip sessions that are actively processing
	var staleSessions []opencode.Session
	var skippedActive int

	for _, session := range sessions {
		// Skip recently updated sessions (within cutoff period)
		if session.Time.Updated > cutoffMs {
			continue
		}

		// Skip sessions that are currently processing
		if client.IsSessionProcessing(session.ID) {
			skippedActive++
			continue
		}

		staleSessions = append(staleSessions, session)
	}

	if !opts.Quiet && skippedActive > 0 {
		fmt.Printf("  Skipped %d active sessions (currently processing)\n", skippedActive)
	}

	if len(staleSessions) == 0 {
		if !opts.Quiet {
			fmt.Println("  No stale sessions found")
		}
		return 0, nil
	}

	if !opts.Quiet {
		fmt.Printf("  Found %d stale sessions:\n", len(staleSessions))
	}

	// Delete stale sessions
	deleted := 0
	skippedOrch := 0
	for _, session := range staleSessions {
		title := session.Title
		if title == "" {
			title = "(untitled)"
		}

		// Skip orchestrator sessions if --preserve-orchestrator is set
		if opts.PreserveOrchestrator && IsOrchestratorSessionTitle(title) {
			skippedOrch++
			continue
		}

		updatedAt := time.Unix(session.Time.Updated/1000, 0)
		age := time.Since(updatedAt).Hours() / 24

		if opts.DryRun {
			if !opts.Quiet {
				fmt.Printf("    [DRY-RUN] Would delete: %s (%s) - %.0f days old\n", session.ID[:12], title, age)
			}
			deleted++
			continue
		}

		if err := client.DeleteSession(session.ID); err != nil {
			if !opts.Quiet {
				fmt.Fprintf(os.Stderr, "    Warning: failed to delete %s: %v\n", session.ID[:12], err)
			}
			continue
		}

		if !opts.Quiet {
			fmt.Printf("    Deleted: %s (%s) - %.0f days old\n", session.ID[:12], title, age)
		}
		deleted++
	}

	if !opts.Quiet && skippedOrch > 0 {
		fmt.Printf("  Skipped %d orchestrator sessions (--preserve-orchestrator)\n", skippedOrch)
	}

	return deleted, nil
}

// IsOrchestratorSessionTitle checks if a session title indicates an orchestrator session.
// This is used when we don't have workspace files (e.g., orphaned sessions).
func IsOrchestratorSessionTitle(title string) bool {
	titleLower := strings.ToLower(title)
	// Check for orchestrator patterns in title
	if strings.Contains(titleLower, "orchestrator") ||
		strings.Contains(titleLower, "meta-orch") ||
		strings.HasPrefix(titleLower, "meta-") ||
		strings.Contains(titleLower, "-orch-") {
		return true
	}
	return false
}
