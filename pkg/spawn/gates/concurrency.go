package gates

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

// DefaultMaxAgents is the default maximum number of concurrent agents.
const DefaultMaxAgents = 5

// GetMaxAgents returns the effective maximum agents limit.
// Priority: flagValue (if >= 0) > ORCH_MAX_AGENTS env var > DefaultMaxAgents constant.
// A negative flagValue means "not set" (use env var or default).
// Returns 0 to disable the limit (unlimited).
func GetMaxAgents(flagValue int) int {
	// If flag was explicitly set (non-negative), use it directly.
	// 0 = unlimited (disabled), >0 = specific limit.
	// Negative values mean "not set" (sentinel from CLI flag default of -1).
	if flagValue >= 0 {
		return flagValue
	}

	// Check environment variable
	if envVal := os.Getenv("ORCH_MAX_AGENTS"); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil && val >= 0 {
			return val
		}
		// Invalid value - fall through to default
		fmt.Fprintf(os.Stderr, "Warning: invalid ORCH_MAX_AGENTS value '%s', using default %d\n", envVal, DefaultMaxAgents)
	}

	return DefaultMaxAgents
}

// CheckConcurrency checks if spawning a new agent would exceed the concurrency limit.
// serverURL is the OpenCode server URL. maxAgentsFlag is the --max-agents flag value
// (-1 = use default/env, 0 = unlimited, >0 = specific limit).
// extractBeadsID extracts a beads ID from a session title string.
// Returns nil if spawning is allowed, or an error if at the limit.
func CheckConcurrency(serverURL string, maxAgentsFlag int, extractBeadsID func(string) string) error {
	maxAgents := GetMaxAgents(maxAgentsFlag)

	// Limit disabled (0 means unlimited)
	if maxAgents == 0 {
		return nil
	}

	// Ensure OpenCode is running before checking
	if err := EnsureOpenCodeRunning(serverURL); err != nil {
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

	for _, s := range sessions {
		// Only count sessions with parseable beadsID (orch-spawned agents)
		beadsID := extractBeadsID(s.Title)
		if beadsID == "" {
			continue // not an orch-spawned agent
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
	}

	// Phase 2: Add tmux-based agents (Claude CLI backend)
	// These agents run in tmux windows WITHOUT OpenCode sessions.
	// Without this, tmux agents are invisible to the concurrency check.
	seenBeadsIDs := make(map[string]bool)
	for _, sd := range sessionList {
		seenBeadsIDs[sd.beadsID] = true
	}
	tmuxAgents := daemon.CountActiveTmuxAgents()
	for beadsID := range tmuxAgents {
		if seenBeadsIDs[beadsID] {
			continue // Already counted from OpenCode session
		}
		// Tmux agents are "running" by definition (window exists = process active)
		sessionList = append(sessionList, sessionData{
			beadsID:   beadsID,
			updatedAt: now, // Window exists = recently active
			status:    "running",
		})
		beadsIDs = append(beadsIDs, beadsID)
	}

	// Phase 3: Batch check which beads issues are closed
	// This prevents counting agents whose work is already complete
	// (issue closed) but whose OpenCode session is still lingering
	closedIssues := daemon.GetClosedIssuesBatch(beadsIDs)

	// Phase 4: Count active agents, excluding closed issues
	activeCount := 0
	for _, sd := range sessionList {
		// Skip sessions whose beads issues are closed
		if closedIssues[sd.beadsID] {
			continue
		}

		// Get phase from beads comments
		phase := ""
		if isComplete, _ := verify.IsPhaseComplete(sd.beadsID, ""); isComplete {
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

// EnsureOpenCodeRunning checks if OpenCode is reachable, and starts it if not.
// Returns nil if OpenCode is running (or was successfully started), error otherwise.
func EnsureOpenCodeRunning(serverURL string) error {
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
