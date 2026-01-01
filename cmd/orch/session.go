// Package main provides the CLI entry point for orch-go.
//
// This file implements the `orch session` command for orchestrator session management.
// An orchestrator session is a "focus block" - a time-bounded period with a goal.
// This is distinct from `orch sessions` (plural) which searches OpenCode session history.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/sessions"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

// ============================================================================
// Session Command - Orchestrator session management
// ============================================================================

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage orchestrator sessions (focus blocks)",
	Long: `Manage orchestrator sessions (focus blocks).

An orchestrator session represents a focused work period with a specific goal.
Unlike worker sessions (spawn→complete), orchestrator sessions are composite:
they may include multiple agent spawns and have a strategic goal.

Session state is stored at ~/.orch/session.json.

Subcommands:
  start   - Start a new session with a goal
  status  - Show current session status
  end     - End the current session

Examples:
  orch session start "Ship snap MVP"     # Start session with goal
  orch session status                    # Show current session
  orch session end                       # End current session`,
}

var sessionStartCmd = &cobra.Command{
	Use:   "start [goal]",
	Short: "Start a new orchestrator session",
	Long: `Start a new orchestrator session with an optional goal.

Creates a new session with a unique ID and sets the focus to the given goal.
The session tracks when it started, its goal, and spawned agents.

If a goal is provided, it will also set the focus using 'orch focus'.

Examples:
  orch session start                           # Start without explicit goal
  orch session start "Ship snap MVP"           # Start with goal (also sets focus)
  orch session start "Fix auth bugs" --issue proj-123  # Start with focus and issue`,
	RunE: func(cmd *cobra.Command, args []string) error {
		goal := ""
		if len(args) > 0 {
			goal = strings.Join(args, " ")
		}
		return runSessionStart(goal)
	},
}

var sessionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current session status",
	Long: `Show the current orchestrator session status.

Displays session ID, duration, goal, and spawned agents.

Examples:
  orch session status     # Show current session`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionStatus()
	},
}

var sessionEndCmd = &cobra.Command{
	Use:   "end",
	Short: "End the current session",
	Long: `End the current orchestrator session.

Generates a handoff document with D.E.K.N. synthesis sections, saves it to
the session directory, and clears session state.

If there are in-progress agents, you'll be warned and can choose to continue
or abort.

Use --skip-reflection with --reason to skip handoff generation. This implements
the 'Gate Over Remind' principle - you must provide a reason for skipping.

Examples:
  orch session end                                              # End with handoff generation
  orch session end --skip-reflection --reason "quick context switch"  # Skip with reason
  orch session end --force                                      # Skip in-progress agent warning`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate --skip-reflection requires --reason
		if sessionEndSkipReflection && sessionEndSkipReason == "" {
			return fmt.Errorf("--skip-reflection requires --reason flag with explanation\n\nExample: orch session end --skip-reflection --reason \"quick context switch\"")
		}
		
		// Handle deprecated --no-handoff by treating it as --skip-reflection
		if sessionEndNoHandoff {
			sessionEndSkipReflection = true
			if sessionEndSkipReason == "" {
				sessionEndSkipReason = "deprecated --no-handoff flag used"
			}
		}
		
		return runSessionEnd()
	},
}

var (
	// Session start flags
	sessionStartIssue string

	// Session end flags
	sessionEndNoHandoff       bool // Deprecated: use --skip-reflection instead
	sessionEndForce           bool
	sessionEndSkipReflection  bool
	sessionEndSkipReason      string
)

func init() {
	sessionStartCmd.Flags().StringVar(&sessionStartIssue, "issue", "", "Beads issue ID to associate with focus")

	// Deprecated: --no-handoff is replaced by --skip-reflection
	sessionEndCmd.Flags().BoolVar(&sessionEndNoHandoff, "no-handoff", false, "DEPRECATED: Use --skip-reflection instead")
	sessionEndCmd.Flags().MarkDeprecated("no-handoff", "use --skip-reflection --reason instead")
	
	sessionEndCmd.Flags().BoolVar(&sessionEndForce, "force", false, "Skip in-progress agent warning")
	sessionEndCmd.Flags().BoolVar(&sessionEndSkipReflection, "skip-reflection", false, "Skip reflection prompts (requires --reason)")
	sessionEndCmd.Flags().StringVar(&sessionEndSkipReason, "reason", "", "Reason for skipping reflection (required with --skip-reflection)")

	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStatusCmd)
	sessionCmd.AddCommand(sessionEndCmd)

	// Register with root command
	rootCmd.AddCommand(sessionCmd)
}

func runSessionStart(goal string) error {
	// Create orchestrator session store
	store, err := sessions.NewOrchestratorStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	// Check if session already exists
	existing := store.Get()
	if existing != nil {
		fmt.Printf("Note: Ending previous session: %s (%s)\n",
			existing.ID,
			formatDuration(time.Since(existing.Started)))
	}

	// Start new session
	session, err := store.Start(goal)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	// If goal provided, also set focus
	if goal != "" {
		focusStore, err := focus.New("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to initialize focus store: %v\n", err)
		} else {
			f := &focus.Focus{
				Goal:    goal,
				BeadsID: sessionStartIssue,
			}
			if err := focusStore.Set(f); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to set focus: %v\n", err)
			} else {
				// Update session with focus ID
				if err := store.SetFocusID(goal); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to link focus to session: %v\n", err)
				}
			}
		}
	}

	// Log the session start
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.orchestrator.started",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"session_id": session.ID,
			"goal":       goal,
			"beads_id":   sessionStartIssue,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Output confirmation
	fmt.Printf("Session started:\n")
	fmt.Printf("  ID:      %s\n", session.ID)
	fmt.Printf("  Started: %s\n", session.Started.Format("2006-01-02 15:04:05"))
	if goal != "" {
		fmt.Printf("  Goal:    %s\n", goal)
	}
	if sessionStartIssue != "" {
		fmt.Printf("  Issue:   %s\n", sessionStartIssue)
	}

	return nil
}

func runSessionStatus() error {
	store, err := sessions.NewOrchestratorStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	session := store.Get()
	if session == nil {
		fmt.Println("No active session")
		fmt.Println("\nStart a session with: orch session start \"your goal\"")
		return nil
	}

	// Calculate duration
	duration := time.Since(session.Started)

	// Header with session ID
	fmt.Printf("Session: %s\n", session.ID)
	fmt.Printf("Started: %s (%s ago)\n",
		session.Started.Format("2006-01-02 15:04"),
		formatDuration(duration))

	// Goal
	if session.Goal != "" {
		fmt.Printf("Goal: %s\n", session.Goal)
	}

	// Spawns this session
	fmt.Println()
	if len(session.Spawns) > 0 {
		fmt.Printf("Spawns this session: %d\n", len(session.Spawns))
		for _, spawnRecord := range session.Spawns {
			// Get spawn status from beads
			status := getSpawnStatus(spawnRecord.BeadsID)
			fmt.Printf("  - %s (%s)\n", spawnRecord.BeadsID, status)
		}
	} else {
		fmt.Println("Spawns this session: 0")
	}

	// Active agents from OpenCode
	activeCount := countActiveAgents()
	fmt.Println()
	fmt.Printf("Active agents: %d\n", activeCount)
	if activeCount > 0 {
		fmt.Println("  Run `orch status` for details.")
	}

	return nil
}

// getSpawnStatus returns the status of a spawn (complete, in_progress, etc.)
// by checking beads comments for Phase: Complete.
func getSpawnStatus(beadsID string) string {
	// Check if phase complete via beads
	isComplete, _ := verify.IsPhaseComplete(beadsID)
	if isComplete {
		return "complete"
	}
	return "in_progress"
}

// countActiveAgents returns the count of active agents from OpenCode.
// An agent is considered active if:
// 1. Updated within last 30 minutes
// 2. Has a parseable beadsID (is orch-spawned)
// 3. Has not reported Phase: Complete
func countActiveAgents() int {
	client := opencode.NewClient(serverURL)
	sessions, err := client.ListSessions("")
	if err != nil {
		return 0
	}

	now := time.Now()
	staleThreshold := 30 * time.Minute
	activeCount := 0

	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		idleTime := now.Sub(updatedAt)
		if idleTime >= staleThreshold {
			continue // stale session
		}
		// Only count sessions with parseable beadsID (orch-spawned agents)
		beadsID := extractBeadsIDFromTitle(s.Title)
		if beadsID == "" {
			continue // not an orch-spawned agent
		}
		// Exclude completed agents
		if isComplete, _ := verify.IsPhaseComplete(beadsID); isComplete {
			continue
		}
		activeCount++
	}

	return activeCount
}

func runSessionEnd() error {
	store, err := sessions.NewOrchestratorStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	session := store.Get()
	if session == nil {
		fmt.Println("No active session to end")
		return nil
	}

	// Calculate duration
	duration := time.Since(session.Started)

	// Print session header
	fmt.Printf("Session %s ending...\n", session.ID)
	fmt.Printf("Duration: %s\n", formatDuration(duration))
	if session.Goal != "" {
		fmt.Printf("Goal: %s\n", session.Goal)
	}

	// Count spawns by status
	completeCount, inProgressCount := countSpawnsByStatus(session.Spawns)
	fmt.Printf("Spawns: %d (%d complete, %d in_progress)\n",
		len(session.Spawns), completeCount, inProgressCount)

	// Warn about in-progress agents unless --force
	if inProgressCount > 0 && !sessionEndForce {
		fmt.Println()
		fmt.Printf("⚠️  %d agent(s) still in progress:\n", inProgressCount)
		for _, spawn := range session.Spawns {
			if getSpawnStatus(spawn.BeadsID) == "in_progress" {
				fmt.Printf("   - %s (%s)\n", spawn.BeadsID, spawn.Skill)
			}
		}
		fmt.Println()
		fmt.Print("Continue ending session? (y/n): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Session end cancelled.")
			return nil
		}
	}

	// Handle --skip-reflection flag (also covers deprecated --no-handoff)
	if sessionEndSkipReflection {
		return endSessionWithSkippedReflection(store, session, duration, sessionEndSkipReason)
	}

	// Generate handoff
	fmt.Println()
	fmt.Println("Generating handoff document...")

	handoffData, err := gatherHandoffData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to gather handoff data: %v\n", err)
		fmt.Println("Continuing without handoff...")
		return endSessionWithoutHandoff(store, session, duration)
	}

	// Prompt for D.E.K.N. synthesis sections (Knowledge and Next)
	fmt.Println()
	fmt.Println("D.E.K.N. Synthesis (required for handoff):")
	fmt.Println("─────────────────────────────────────────────")

	reader := bufio.NewReader(os.Stdin)

	// Knowledge prompt
	fmt.Println()
	fmt.Println("Knowledge (What was learned this session?):")
	fmt.Println("  Patterns discovered, insights gained, or lessons learned.")
	fmt.Print("  > ")
	knowledge, _ := reader.ReadString('\n')
	knowledge = strings.TrimSpace(knowledge)
	if knowledge == "" {
		fmt.Println("  (skipped)")
	}
	handoffData.DEKN.Knowledge = knowledge

	// Next prompt
	fmt.Println()
	fmt.Println("Next (Recommended actions for next session?):")
	fmt.Println("  What should the next session prioritize?")
	fmt.Print("  > ")
	next, _ := reader.ReadString('\n')
	next = strings.TrimSpace(next)
	if next == "" {
		fmt.Println("  (skipped)")
	}
	handoffData.DEKN.Next = next

	// Friction prompt
	fmt.Println()
	fmt.Println("Friction (What was harder than it should have been?):")
	fmt.Println("  Tool issues, context gaps, process friction.")
	fmt.Print("  > ")
	friction, _ := reader.ReadString('\n')
	friction = strings.TrimSpace(friction)
	if friction == "" {
		fmt.Println("  (skipped)")
	}
	handoffData.DEKN.Friction = friction

	// System Reaction prompt
	fmt.Println()
	fmt.Println("System Reaction (Does this suggest improvements?):")
	fmt.Println("  Skill update, CLAUDE.md update, new tooling?")
	fmt.Print("  > ")
	reaction, _ := reader.ReadString('\n')
	reaction = strings.TrimSpace(reaction)
	if reaction == "" {
		fmt.Println("  (skipped)")
	}
	handoffData.DEKN.SystemReaction = reaction

	// Generate markdown
	markdown, err := generateHandoffMarkdown(handoffData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to generate handoff markdown: %v\n", err)
		return endSessionWithoutHandoff(store, session, duration)
	}

	// Save to session directory
	sessionDir := getSessionDirectory(session.Started)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create session directory: %v\n", err)
	} else {
		handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
		if err := os.WriteFile(handoffPath, []byte(markdown), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write handoff file: %v\n", err)
		} else {
			fmt.Println()
			fmt.Printf("Handoff saved to: %s\n", handoffPath)
		}
	}

	// End the session
	_, err = store.End()
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	// Log the session end
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.orchestrator.ended",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"session_id":      session.ID,
			"goal":            session.Goal,
			"duration":        duration.Seconds(),
			"spawns":          len(session.Spawns),
			"spawns_complete": completeCount,
			"handoff_saved":   true,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Println()
	fmt.Println("Session ended.")

	return nil
}

// countSpawnsByStatus counts spawns by their completion status.
func countSpawnsByStatus(spawns []sessions.SpawnRecord) (complete, inProgress int) {
	for _, spawn := range spawns {
		if getSpawnStatus(spawn.BeadsID) == "complete" {
			complete++
		} else {
			inProgress++
		}
	}
	return
}

// endSessionWithSkippedReflection ends the session without generating a handoff document,
// logging the skip reason for pattern detection.
func endSessionWithSkippedReflection(store *sessions.OrchestratorStore, session *sessions.OrchestratorSession, duration time.Duration, reason string) error {
	logger := events.NewLogger(events.DefaultLogPath())

	// Log the reflection skip event for pattern detection
	skipEvent := events.Event{
		Type:      "session.orchestrator.reflection_skipped",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"session_id": session.ID,
			"reason":     reason,
		},
	}
	if err := logger.Log(skipEvent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log reflection skip event: %v\n", err)
	}

	_, err := store.End()
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	completeCount, _ := countSpawnsByStatus(session.Spawns)

	// Log the session end
	endEvent := events.Event{
		Type:      "session.orchestrator.ended",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"session_id":         session.ID,
			"goal":               session.Goal,
			"duration":           duration.Seconds(),
			"spawns":             len(session.Spawns),
			"spawns_complete":    completeCount,
			"handoff_saved":      false,
			"reflection_skipped": true,
			"skip_reason":        reason,
		},
	}
	if err := logger.Log(endEvent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Println()
	fmt.Printf("Session ended (reflection skipped: %s).\n", reason)
	fmt.Println("Tip: Run 'orch handoff' to generate a handoff document for the next session")

	return nil
}

// endSessionWithoutHandoff ends the session without generating a handoff document.
// This is called when handoff generation fails mid-process.
func endSessionWithoutHandoff(store *sessions.OrchestratorStore, session *sessions.OrchestratorSession, duration time.Duration) error {
	_, err := store.End()
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	completeCount, _ := countSpawnsByStatus(session.Spawns)

	// Log the session end
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.orchestrator.ended",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"session_id":      session.ID,
			"goal":            session.Goal,
			"duration":        duration.Seconds(),
			"spawns":          len(session.Spawns),
			"spawns_complete": completeCount,
			"handoff_saved":   false,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Println()
	fmt.Println("Session ended (no handoff generated).")
	fmt.Println("Tip: Run 'orch handoff' to generate a handoff document for the next session")

	return nil
}

// getSessionDirectory returns the session directory path for the given time.
// Format: ~/.orch/session/YYYY-MM-DD/
func getSessionDirectory(t time.Time) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "session", t.Format("2006-01-02"))
}

// Note: formatDuration is defined in wait.go and shared across commands
