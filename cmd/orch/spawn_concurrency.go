// Package main provides concurrency management for spawn commands.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// getMaxAgents returns the effective maximum agents limit.
// Priority: --max-agents flag > ORCH_MAX_AGENTS env var > DefaultMaxAgents constant.
// Returns 0 if limit is explicitly disabled (flag set to 0 or env var set to "0").
func getMaxAgents() int {
	// If flag was explicitly set (not the sentinel -1), use it
	// This allows --max-agents 0 to mean "unlimited"
	if spawnMaxAgents != -1 {
		return spawnMaxAgents
	}

	// Check environment variable
	if envVal := os.Getenv("ORCH_MAX_AGENTS"); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil {
			return val
		}
		// Invalid value - fall through to default
		fmt.Fprintf(os.Stderr, "Warning: invalid ORCH_MAX_AGENTS value '%s', using default %d\n", envVal, DefaultMaxAgents)
	}

	return DefaultMaxAgents
}

// ensureOpenCodeRunning checks if OpenCode is reachable, and starts it if not.
// Returns nil if OpenCode is running (or was successfully started), error otherwise.
func ensureOpenCodeRunning() error {
	client := opencode.NewClient(serverURL)
	_, err := client.ListSessions("")
	if err == nil {
		return nil // Already running
	}

	// Check if it's a connection error (not running)
	if !strings.Contains(err.Error(), "connection refused") {
		return nil // Some other error, let it proceed
	}

	fmt.Fprintf(os.Stderr, "OpenCode not running, starting it...\n")

	// Start OpenCode server in background, fully detached via shell
	// This ensures the process survives even if the parent is killed
	// Set ORCH_WORKER=1 so agents spawned by this server know they are orch-managed workers
	cmd := exec.Command("sh", "-c", "ORCH_WORKER=1 opencode serve --port 4096 </dev/null >/dev/null 2>&1 &")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start OpenCode: %w", err)
	}

	// Wait for it to be ready (poll for up to 10 seconds)
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		_, err := client.ListSessions("")
		if err == nil {
			fmt.Fprintf(os.Stderr, "OpenCode started successfully\n")
			return nil
		}
	}

	return fmt.Errorf("OpenCode started but not responding after 10s")
}

// checkConcurrencyLimit checks if spawning a new agent would exceed the concurrency limit.
// Returns nil if spawning is allowed, or an error if at the limit.
func checkConcurrencyLimit() error {
	maxAgents := getMaxAgents()

	// Limit disabled (0 means unlimited)
	if maxAgents == 0 {
		return nil
	}

	// Ensure OpenCode is running before checking
	if err := ensureOpenCodeRunning(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		return nil // Allow spawn to proceed, it will fail later with better error
	}

	// Check active count via OpenCode API
	client := opencode.NewClient(serverURL)
	sessions, err := client.ListSessions("")
	if err != nil {
		// If we can't check, log a warning but allow the spawn
		fmt.Fprintf(os.Stderr, "Warning: could not check agent limit (API error): %v\n", err)
		return nil
	}

	// Filter to only count active ORCH-SPAWNED sessions using two-threshold logic:
	// Uses agent.IsActiveForConcurrency with aggressive 1h threshold
	// This prevents ghost agents from blocking new spawns
	now := time.Now()
	activeThreshold := 10 * time.Minute // Threshold for determining "running" vs "idle"

	// Phase 1: Collect all beads IDs and session data for batch processing
	type sessionData struct {
		beadsID   string
		updatedAt time.Time
		status    string
	}
	var sessionList []sessionData
	var beadsIDs []string
	beadsIDToProjectDir := make(map[string]string)

	for _, s := range sessions {
		// Only count sessions with parseable beadsID (orch-spawned agents)
		beadsID := extractBeadsIDFromTitle(s.Title)
		if beadsID == "" {
			continue // not an orch-spawned agent
		}

		// Skip untracked agents (spawned with --no-track).
		// These have beads IDs like "project-untracked-1766695797" which don't exist
		// in the beads database, so they would incorrectly fail the "is closed" check
		// and count against concurrency limit. Untracked spawns are ad-hoc work
		// that shouldn't block tracked agent spawns.
		if isUntrackedBeadsID(beadsID) {
			continue
		}

		// Determine status based on recent activity
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		status := "idle"
		if now.Sub(updatedAt) < activeThreshold {
			status = "running"
		}

		sessionList = append(sessionList, sessionData{
			beadsID:   beadsID,
			updatedAt: updatedAt,
			status:    status,
		})
		beadsIDs = append(beadsIDs, beadsID)

		// Use session.Directory for cross-project resolution.
		// Skip "/" as it's not a valid project directory.
		if s.Directory != "" && s.Directory != "/" {
			beadsIDToProjectDir[beadsID] = s.Directory
		}
	}

	// Phase 2: Batch check which beads issues are closed using session.Directory
	// This prevents counting agents whose work is already complete
	// (issue closed) but whose OpenCode session is still lingering.
	// Using session.Directory avoids cross-project lookup failures that would
	// otherwise be suppressed as "issue not found" errors.
	closedIssues := daemon.GetClosedIssuesBatchWithProjectDirs(beadsIDs, beadsIDToProjectDir)

	// Phase 3: Count active agents, excluding closed issues
	activeCount := 0
	for _, sd := range sessionList {
		// Skip sessions whose beads issues are closed
		if closedIssues[sd.beadsID] {
			continue
		}

		// Get phase from beads comments
		phase := ""
		if isComplete, _ := verify.IsPhaseComplete(sd.beadsID); isComplete {
			phase = "Complete"
		}

		// Use IsActiveForConcurrency to determine if this agent counts
		if agent.IsActiveForConcurrency(sd.status, sd.updatedAt, phase) {
			activeCount++
		}
	}

	if activeCount >= maxAgents {
		return fmt.Errorf("concurrency limit reached: %d active agents (max %d). Use 'orch status' to see active agents, 'orch complete' to finish agents, or --max-agents to increase limit", activeCount, maxAgents)
	}

	return nil
}
