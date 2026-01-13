// Package main provides the CLI entry point for orch-go.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

// ============================================================================
// Session Command - Manage orchestrator work sessions
// ============================================================================

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage orchestrator work sessions",
	Long: `Manage orchestrator work sessions.

A session represents a focused work period with:
- A goal (north star priority)
- Start time
- Tracked spawns during the session

Session status derives agent state at query time via actual liveness checks,
not stored state. This prevents stale tracking.

Examples:
  orch session start "Ship snap MVP"    # Start a new session
  orch session status                   # Show current session status
  orch session end                      # End the current session`,
}

var (
	sessionJSON        bool
	resumeForInjection bool
	resumeCheck        bool
)

func init() {
	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStatusCmd)
	sessionCmd.AddCommand(sessionEndCmd)
	sessionCmd.AddCommand(sessionResumeCmd)
	sessionCmd.AddCommand(sessionMigrateCmd)

	// Add --json flag to status command
	sessionStatusCmd.Flags().BoolVar(&sessionJSON, "json", false, "Output as JSON")

	// Add flags for resume command
	sessionResumeCmd.Flags().BoolVar(&resumeForInjection, "for-injection", false, "Output condensed format for hook injection")
	sessionResumeCmd.Flags().BoolVar(&resumeCheck, "check", false, "Check if handoff exists (exit code only)")

	rootCmd.AddCommand(sessionCmd)
}

// ============================================================================
// Session Start Command
// ============================================================================

var sessionStartCmd = &cobra.Command{
	Use:   "start [goal]",
	Short: "Start a new orchestrator work session",
	Long: `Start a new orchestrator work session with a focus goal.

The session tracks:
- Your focus goal
- When the session started
- Agents spawned during the session

If a session is already active, it will be replaced.

Examples:
  orch session start "Ship snap MVP"
  orch session start "Fix auth bugs"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		goal := strings.Join(args, " ")
		return runSessionStart(goal)
	},
}

func runSessionStart(goal string) error {
	store, err := session.New("")
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Check if session was already active
	wasActive := store.IsActive()

	if err := store.Start(goal); err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	// Create session workspace with SESSION_HANDOFF.md
	workspacePath, err := createSessionWorkspace(goal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create session workspace: %v\n", err)
		// Continue anyway - workspace is nice-to-have for interactive sessions
	}

	// Log the session start
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"goal":       goal,
		"was_active": wasActive,
		"started_at": time.Now().Format(session.TimeFormat),
	}
	if workspacePath != "" {
		eventData["workspace_path"] = workspacePath
	}
	event := events.Event{
		Type:      "session.started",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	if wasActive {
		fmt.Println("Previous session ended.")
	}
	fmt.Printf("Session started: %s\n", goal)
	fmt.Printf("  Start time: %s\n", time.Now().Format("15:04"))
	if workspacePath != "" {
		fmt.Printf("  Workspace:  %s\n", workspacePath)
	}

	// Surface reflection suggestions for high-count synthesis opportunities
	// This proactively surfaces consolidation needs that accumulated since last reflection
	surfaceReflectSuggestions()

	return nil
}

// createSessionWorkspace creates a workspace directory for interactive orchestrator sessions.
// This provides parity with spawned orchestrators by pre-creating SESSION_HANDOFF.md.
// Workspace is created in ~/.orch/session/{date}/ to match existing session directory structure.
func createSessionWorkspace(goal string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Use date-based directory for session workspace (matches existing structure)
	dateStr := time.Now().Format("2006-01-02")
	workspacePath := filepath.Join(home, ".orch", "session", dateStr)

	// Create workspace directory
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Generate workspace name for SESSION_HANDOFF.md
	// Interactive sessions use "interactive-" prefix + date + time suffix
	timeStr := time.Now().Format("150405")
	workspaceName := fmt.Sprintf("interactive-%s-%s", dateStr, timeStr)
	startTime := time.Now().Format("2006-01-02 15:04")

	// Use "Interactive session" as default goal if empty
	sessionGoal := goal
	if sessionGoal == "" {
		sessionGoal = "Interactive session"
	}

	// Generate pre-filled SESSION_HANDOFF.md content
	content, err := spawn.GeneratePreFilledSessionHandoff(workspaceName, sessionGoal, startTime)
	if err != nil {
		return "", fmt.Errorf("failed to generate session handoff: %w", err)
	}

	// Write SESSION_HANDOFF.md
	handoffPath := filepath.Join(workspacePath, "SESSION_HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write session handoff: %w", err)
	}

	return workspacePath, nil
}

// SynthesisWarningThreshold is the minimum count of investigations to show a warning.
// Matches SynthesisIssueThreshold in kb-cli to maintain consistency.
const SynthesisWarningThreshold = 10

// SuggestionFreshnessHours is the maximum age of suggestions to consider fresh.
// Suggestions older than this are considered stale and won't be shown.
const SuggestionFreshnessHours = 24

// surfaceReflectSuggestions loads and displays synthesis warnings from reflect-suggestions.json.
// This proactively surfaces consolidation needs at session start so orchestrators are aware
// of accumulated investigation clusters that need synthesis into guides.
func surfaceReflectSuggestions() {
	suggestions, err := daemon.LoadSuggestions()
	if err != nil || suggestions == nil {
		// No suggestions file or failed to load - silently skip
		return
	}

	// Check freshness - skip stale suggestions
	if time.Since(suggestions.Timestamp).Hours() > SuggestionFreshnessHours {
		return
	}

	// Filter to high-count synthesis opportunities
	var highCount []daemon.SynthesisSuggestion
	for _, s := range suggestions.Synthesis {
		if s.Count >= SynthesisWarningThreshold {
			highCount = append(highCount, s)
		}
	}

	if len(highCount) == 0 {
		return
	}

	// Display synthesis warnings
	fmt.Println()
	fmt.Println("📚 SYNTHESIS OPPORTUNITIES")
	fmt.Printf("   %d topics have accumulated %d+ investigations:\n", len(highCount), SynthesisWarningThreshold)

	// Show top 5 topics
	maxShow := 5
	if len(highCount) < maxShow {
		maxShow = len(highCount)
	}
	for i := 0; i < maxShow; i++ {
		s := highCount[i]
		fmt.Printf("   • %s: %d investigations → kb create guide \"%s\"\n", s.Topic, s.Count, s.Topic)
	}
	if len(highCount) > maxShow {
		fmt.Printf("   ... and %d more topics\n", len(highCount)-maxShow)
	}
	fmt.Printf("   Run 'kb reflect --type synthesis' for details.\n")
}

// ============================================================================
// Session Status Command
// ============================================================================

var sessionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current session status with spawn reconciliation",
	Long: `Show current session status including spawns with reconciled states.

Spawn states are derived at query time by checking:
- OpenCode session liveness
- tmux window existence
- Beads issue status

This ensures accurate state rather than trusting potentially stale stored data.

States:
  active    - Agent is running (has live OpenCode session or tmux window)
  completed - Agent finished (beads issue closed, no live session)
  phantom   - Agent lost (beads issue open, but no live session)

Examples:
  orch session status         # Show status
  orch session status --json  # Output as JSON for scripting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionStatus()
	},
}

// SessionStatusOutput is the JSON output format for session status.
type SessionStatusOutput struct {
	Active     bool                      `json:"active"`
	Goal       string                    `json:"goal,omitempty"`
	StartedAt  string                    `json:"started_at,omitempty"`
	Duration   string                    `json:"duration,omitempty"`
	Spawns     []session.SpawnStatus     `json:"spawns,omitempty"`
	Counts     *SpawnCounts              `json:"counts,omitempty"`
	Checkpoint *session.CheckpointStatus `json:"checkpoint,omitempty"`
}

// SpawnCounts summarizes spawn states.
type SpawnCounts struct {
	Total     int `json:"total"`
	Active    int `json:"active"`
	Completed int `json:"completed"`
	Phantom   int `json:"phantom"`
}

func runSessionStatus() error {
	store, err := session.New("")
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	sess := store.Get()

	// Build output
	output := SessionStatusOutput{
		Active: sess != nil,
	}

	if sess != nil {
		output.Goal = sess.Goal
		output.StartedAt = sess.StartedAt.Format(session.TimeFormat)
		output.Duration = formatSessionDuration(store.Duration())

		// Get spawn statuses with reconciliation
		statuses := store.GetSpawnStatuses(serverURL)
		output.Spawns = statuses

		// Count by state
		counts := &SpawnCounts{Total: len(statuses)}
		for _, s := range statuses {
			switch s.State {
			case "active":
				counts.Active++
			case "completed":
				counts.Completed++
			case "phantom":
				counts.Phantom++
			}
		}
		output.Counts = counts

		// Get checkpoint status using orchestrator thresholds
		// (orch session is for orchestrator sessions, not agent sessions)
		output.Checkpoint = store.GetCheckpointStatusWithType(session.SessionTypeOrchestrator)
	}

	// JSON output
	if sessionJSON {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Human-readable output
	if !output.Active {
		fmt.Println("No active session")
		fmt.Println("\nStart a session with: orch session start \"your goal\"")
		return nil
	}

	fmt.Printf("Session active:\n")
	fmt.Printf("  Goal:     %s\n", output.Goal)
	fmt.Printf("  Duration: %s", output.Duration)

	// Show checkpoint status inline with duration
	if output.Checkpoint != nil {
		switch output.Checkpoint.Level {
		case "exceeded":
			fmt.Printf(" ⛔")
		case "strong":
			fmt.Printf(" 🔴")
		case "warning":
			fmt.Printf(" 🟡")
		}
	}
	fmt.Println()

	fmt.Printf("  Spawns:   %d total", output.Counts.Total)
	if output.Counts.Active > 0 {
		fmt.Printf(" (%d active", output.Counts.Active)
		if output.Counts.Completed > 0 {
			fmt.Printf(", %d completed", output.Counts.Completed)
		}
		if output.Counts.Phantom > 0 {
			fmt.Printf(", %d phantom", output.Counts.Phantom)
		}
		fmt.Printf(")")
	}
	fmt.Println()

	// Show spawn details if any
	if len(output.Spawns) > 0 {
		fmt.Println("\nSpawns:")
		for _, spawn := range output.Spawns {
			stateIcon := stateToIcon(spawn.State)
			age := formatSessionDuration(time.Since(spawn.SpawnedAt))
			fmt.Printf("  %s %s (%s) - %s ago\n", stateIcon, spawn.BeadsID, spawn.Skill, age)
		}
	}

	// Show checkpoint warning if applicable
	if output.Checkpoint != nil && output.Checkpoint.Level != "ok" {
		fmt.Println()
		switch output.Checkpoint.Level {
		case "exceeded":
			fmt.Printf("⛔ CHECKPOINT EXCEEDED: %s\n", output.Checkpoint.Message)
			fmt.Println("   Session has run too long. Quality may be degraded.")
			fmt.Println("   Action: Run 'orch session end' and start fresh.")
		case "strong":
			fmt.Printf("🔴 CHECKPOINT STRONGLY RECOMMENDED: %s\n", output.Checkpoint.Message)
			fmt.Printf("   Time until max: %s\n", formatSessionDuration(output.Checkpoint.NextThreshold))
			fmt.Println("   Action: Write SESSION_HANDOFF.md, consider ending session.")
		case "warning":
			fmt.Printf("🟡 CHECKPOINT SUGGESTED: %s\n", output.Checkpoint.Message)
			fmt.Printf("   Time until strong warning: %s\n", formatSessionDuration(output.Checkpoint.NextThreshold))
			fmt.Println("   Action: Assess progress, write interim handoff if needed.")
		}
	}

	return nil
}

// stateToIcon converts state to a visual indicator.
func stateToIcon(state string) string {
	switch state {
	case "active":
		return "🟢"
	case "completed":
		return "✅"
	case "phantom":
		return "👻"
	default:
		return "❓"
	}
}

// formatSessionDuration formats a duration in a human-readable way for session display.
func formatSessionDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

// ============================================================================
// Session End Command
// ============================================================================

var sessionEndCmd = &cobra.Command{
	Use:   "end",
	Short: "End the current session",
	Long: `End the current orchestrator work session.

This clears the session state. Use before:
- Taking a break
- Handing off to another orchestrator
- Changing focus to a different goal

The session summary is logged for posterity.

Examples:
  orch session end`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionEnd()
	},
}

func runSessionEnd() error {
	store, err := session.New("")
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	if !store.IsActive() {
		fmt.Println("No active session to end")
		return nil
	}

	// Get session info before ending
	duration := store.Duration()
	spawnCount := store.SpawnCount()

	// Get spawn statuses for final summary
	statuses := store.GetSpawnStatuses(serverURL)
	activeCount := 0
	for _, s := range statuses {
		if s.State == "active" {
			activeCount++
		}
	}

	// Create project-specific session directory and handoff
	// This creates .orch/session/{timestamp}/ and updates latest symlink
	projectDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get project directory: %v\n", err)
	} else {
		if err := createSessionHandoffDirectory(projectDir, store.Get()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create session handoff directory: %v\n", err)
		}
	}

	// End the session
	ended, err := store.End()
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	// Log the session end
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.ended",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"goal":          ended.Goal,
			"started_at":    ended.StartedAt.Format(session.TimeFormat),
			"duration":      duration.String(),
			"spawn_count":   spawnCount,
			"active_at_end": activeCount,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("Session ended: %s\n", ended.Goal)
	fmt.Printf("  Duration:  %s\n", formatSessionDuration(duration))
	fmt.Printf("  Spawns:    %d total\n", spawnCount)

	if activeCount > 0 {
		fmt.Printf("\n⚠️  %d agent(s) still active. Use 'orch status' to monitor.\n", activeCount)
	}

	// Show checkpoint advice based on session duration using orchestrator thresholds
	orchThresholds := session.DefaultOrchestratorThresholds()
	if duration >= orchThresholds.Max {
		fmt.Printf("\n⛔ Session exceeded %s checkpoint max.\n", formatSessionDuration(orchThresholds.Max))
		fmt.Println("   Consider shorter sessions to maintain quality.")
	} else if duration >= orchThresholds.Strong {
		fmt.Printf("\n🟡 Session was %s+. Good to hand off, but review quality of late work.\n", formatSessionDuration(orchThresholds.Strong))
	}

	return nil
}

// ============================================================================
// Session Resume Command
// ============================================================================

var sessionResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume orchestrator session by injecting prior handoff",
	Long: `Resume an orchestrator session by discovering and displaying the most recent SESSION_HANDOFF.md.

This command walks up the directory tree to find .orch/session/latest/SESSION_HANDOFF.md
and displays it in the format appropriate for the use case.

Modes:
  Default (interactive):  Display formatted handoff for manual review
  --for-injection:        Output condensed format for hook injection (no decorations)
  --check:                Just check if handoff exists (exit code 0 if yes, 1 if no)

Discovery:
  1. Starts from current directory
  2. Walks up directory tree looking for .orch/session/latest symlink
  3. Reads SESSION_HANDOFF.md from the symlink target
  4. Fails gracefully if no handoff found (valid for fresh sessions)

Examples:
  orch session resume                  # Interactive display
  orch session resume --for-injection  # For hooks (condensed format)
  orch session resume --check          # Check existence only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionResume()
	},
}

func runSessionResume() error {
	// Discover handoff by walking up directory tree
	handoffPath, err := discoverSessionHandoff()
	if err != nil {
		if resumeCheck {
			// Exit code 1 for --check mode when handoff not found
			os.Exit(1)
		}
		return err
	}

	if resumeCheck {
		// Exit code 0 for --check mode when handoff exists
		os.Exit(0)
	}

	// Read the handoff content
	content, err := os.ReadFile(handoffPath)
	if err != nil {
		return fmt.Errorf("failed to read handoff: %w", err)
	}

	// Output based on mode
	if resumeForInjection {
		// Condensed format for hooks (just the content, no decorations)
		fmt.Print(string(content))
	} else {
		// Interactive format with metadata
		fmt.Printf("📋 Session Handoff\n")
		fmt.Printf("   Source: %s\n", handoffPath)
		fmt.Println()
		fmt.Print(string(content))
	}

	return nil
}

// discoverSessionHandoff walks up the directory tree to find .orch/session/{window-name}/latest/SESSION_HANDOFF.md.
// Returns the full path to the handoff file, or an error if not found.
// Window-scoping prevents concurrent orchestrator sessions from clobbering each other's context.
func discoverSessionHandoff() (string, error) {
	// Get current tmux window name (or "default" if not in tmux)
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		return "", fmt.Errorf("failed to get tmux window name: %w", err)
	}

	// Start from current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up the directory tree
	dir := currentDir
	for {
		// Check for .orch/session/{window-name}/latest symlink
		latestPath := filepath.Join(dir, ".orch", "session", windowName, "latest")
		if stat, err := os.Lstat(latestPath); err == nil {
			// latest exists - check if it's a symlink or directory
			var sessionDir string
			if stat.Mode()&os.ModeSymlink != 0 {
				// It's a symlink - resolve it
				target, err := os.Readlink(latestPath)
				if err != nil {
					return "", fmt.Errorf("failed to read latest symlink: %w", err)
				}
				// If target is relative, resolve it relative to .orch/session/{window-name}/
				if !filepath.IsAbs(target) {
					sessionDir = filepath.Join(dir, ".orch", "session", windowName, target)
				} else {
					sessionDir = target
				}
			} else {
				// It's a directory (not a symlink)
				sessionDir = latestPath
			}

			// Check for SESSION_HANDOFF.md in the session directory
			handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
			if _, err := os.Stat(handoffPath); err == nil {
				return handoffPath, nil
			}
		}

		// BACKWARD COMPATIBILITY: Check for old non-window-scoped structure
		// This fallback enables session resume for handoffs created before window-scoping was added
		legacyLatestPath := filepath.Join(dir, ".orch", "session", "latest")
		if stat, err := os.Lstat(legacyLatestPath); err == nil {
			var sessionDir string
			if stat.Mode()&os.ModeSymlink != 0 {
				// It's a symlink - resolve it
				target, err := os.Readlink(legacyLatestPath)
				if err != nil {
					return "", fmt.Errorf("failed to read legacy latest symlink: %w", err)
				}
				// If target is relative, resolve it relative to .orch/session/
				if !filepath.IsAbs(target) {
					sessionDir = filepath.Join(dir, ".orch", "session", target)
				} else {
					sessionDir = target
				}
			} else {
				// It's a directory (not a symlink)
				sessionDir = legacyLatestPath
			}

			// Check for SESSION_HANDOFF.md in the legacy session directory
			handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
			if _, err := os.Stat(handoffPath); err == nil {
				// Found legacy handoff - emit warning about migration
				fmt.Fprintf(os.Stderr, "⚠️  Using legacy session handoff structure.\n")
				fmt.Fprintf(os.Stderr, "   Run 'orch session migrate' to update to window-scoped structure.\n")
				fmt.Fprintf(os.Stderr, "   (This prevents concurrent orchestrators from clobbering each other's context)\n\n")
				return handoffPath, nil
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	// Enhanced error message showing both paths checked
	windowScopedPath := fmt.Sprintf(".orch/session/%s/latest/SESSION_HANDOFF.md", windowName)
	legacyPath := ".orch/session/latest/SESSION_HANDOFF.md"
	return "", fmt.Errorf("no session handoff found for window %q\nChecked:\n  - Window-scoped: %s\n  - Legacy: %s", windowName, windowScopedPath, legacyPath)
}

// createSessionHandoffDirectory creates a timestamped session directory with SESSION_HANDOFF.md
// and updates the latest symlink to point to it.
// Session handoffs are scoped by tmux window name to prevent concurrent orchestrators from clobbering each other.
func createSessionHandoffDirectory(projectDir string, sess *session.Session) error {
	if sess == nil {
		return fmt.Errorf("no active session")
	}

	// Get current tmux window name (or "default" if not in tmux)
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		return fmt.Errorf("failed to get tmux window name: %w", err)
	}

	// Create timestamped directory name (YYYY-MM-DD-HHMM format)
	timestamp := time.Now().Format("2006-01-02-1504")
	// Structure: .orch/session/{window-name}/{timestamp}/
	sessionDir := filepath.Join(projectDir, ".orch", "session", windowName, timestamp)

	// Create the session directory (with parent window-scoped directory)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Generate SESSION_HANDOFF.md content using the same template as spawned orchestrators
	// For now, create a basic handoff - TODO: enhance with reflection prompts
	handoffContent := fmt.Sprintf(`# Session Handoff

**Session Goal:** %s
**Started:** %s
**Duration:** %s

---

## Summary

[Orchestrator fills this in during session end]

---

## What Was Accomplished

[Key achievements and completions from this session]

---

## Active Work

[Agents still running or issues in progress]

---

## Pending Work

[Ready work that wasn't tackled]

---

## Recommendations

[What should the next session focus on?]

---

## Context for Next Session

[Important context, decisions made, patterns discovered]
`,
		sess.Goal,
		sess.StartedAt.Format("2006-01-02 15:04"),
		time.Since(sess.StartedAt).String(),
	)

	// Write SESSION_HANDOFF.md
	handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte(handoffContent), 0644); err != nil {
		return fmt.Errorf("failed to write SESSION_HANDOFF.md: %w", err)
	}

	// Update window-scoped latest symlink
	// Structure: .orch/session/{window-name}/latest -> {timestamp}
	latestSymlink := filepath.Join(projectDir, ".orch", "session", windowName, "latest")

	// Remove existing symlink if present
	_ = os.Remove(latestSymlink)

	// Create new symlink (relative path to avoid absolute path issues)
	if err := os.Symlink(timestamp, latestSymlink); err != nil {
		return fmt.Errorf("failed to create latest symlink: %w", err)
	}

	fmt.Printf("\n📋 Session handoff created: %s\n", handoffPath)
	fmt.Printf("   Latest symlink updated: .orch/session/%s/latest -> %s\n", windowName, timestamp)

	return nil
}

// ============================================================================
// Session Migrate Command - Migrate legacy handoffs to window-scoped structure
// ============================================================================

var sessionMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate legacy session handoffs to window-scoped structure",
	Long: `Migrate legacy session handoffs to window-scoped structure.

Before window-scoping was added, session handoffs were stored in:
  .orch/session/{timestamp}/SESSION_HANDOFF.md

After window-scoping, they're stored in:
  .orch/session/{window-name}/{timestamp}/SESSION_HANDOFF.md

This command migrates old handoffs to the new structure.

Examples:
  orch session migrate              # Migrate to current window
  orch session migrate --all        # Show migration status for all windows`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionMigrate()
	},
}

func runSessionMigrate() error {
	// Get current directory to find .orch/session
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find project root by walking up to .orch directory
	projectDir := currentDir
	for {
		sessionDir := filepath.Join(projectDir, ".orch", "session")
		if _, err := os.Stat(sessionDir); err == nil {
			break
		}
		parent := filepath.Dir(projectDir)
		if parent == projectDir {
			return fmt.Errorf("no .orch/session directory found (not in an orch-managed project)")
		}
		projectDir = parent
	}

	sessionBaseDir := filepath.Join(projectDir, ".orch", "session")

	// Get current window name for migration target
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		return fmt.Errorf("failed to get window name: %w", err)
	}

	// Check for legacy handoffs (non-window-scoped directories)
	entries, err := os.ReadDir(sessionBaseDir)
	if err != nil {
		return fmt.Errorf("failed to read session directory: %w", err)
	}

	var legacyDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Legacy directories are timestamp format: YYYY-MM-DD-HHMM
		// Window-scoped directories are names (e.g., "default", "pw", "og-feat-...")
		name := entry.Name()
		// Check if it looks like a timestamp (starts with digit)
		if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
			legacyDirs = append(legacyDirs, name)
		}
	}

	if len(legacyDirs) == 0 {
		fmt.Println("✅ No legacy handoffs found - already using window-scoped structure")
		return nil
	}

	// Show what will be migrated
	fmt.Printf("Found %d legacy handoff(s) to migrate:\n\n", len(legacyDirs))
	for _, dir := range legacyDirs {
		handoffPath := filepath.Join(sessionBaseDir, dir, "SESSION_HANDOFF.md")
		if _, err := os.Stat(handoffPath); err == nil {
			fmt.Printf("  • %s → .orch/session/%s/%s\n", dir, windowName, dir)
		}
	}

	fmt.Printf("\nMigrate to window-scoped structure for window %q? (y/N): ", windowName)
	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("Migration cancelled")
		return nil
	}

	// Perform migration
	windowScopedDir := filepath.Join(sessionBaseDir, windowName)
	if err := os.MkdirAll(windowScopedDir, 0755); err != nil {
		return fmt.Errorf("failed to create window-scoped directory: %w", err)
	}

	migratedCount := 0
	for _, dir := range legacyDirs {
		sourcePath := filepath.Join(sessionBaseDir, dir)
		destPath := filepath.Join(windowScopedDir, dir)

		// Check if handoff exists
		handoffPath := filepath.Join(sourcePath, "SESSION_HANDOFF.md")
		if _, err := os.Stat(handoffPath); err != nil {
			// Skip directories without handoffs
			continue
		}

		// Move the directory
		if err := os.Rename(sourcePath, destPath); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to migrate %s: %v\n", dir, err)
			continue
		}
		migratedCount++
	}

	// Update latest symlink to point to most recent migrated handoff
	if migratedCount > 0 {
		// Find most recent timestamp directory
		var latestTimestamp string
		entries, _ := os.ReadDir(windowScopedDir)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if name > latestTimestamp && len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
				latestTimestamp = name
			}
		}

		if latestTimestamp != "" {
			latestSymlink := filepath.Join(windowScopedDir, "latest")
			_ = os.Remove(latestSymlink) // Remove old symlink if exists
			if err := os.Symlink(latestTimestamp, latestSymlink); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Failed to update latest symlink: %v\n", err)
			}
		}
	}

	// Remove legacy latest symlink at root level
	legacyLatest := filepath.Join(sessionBaseDir, "latest")
	if _, err := os.Lstat(legacyLatest); err == nil {
		if err := os.Remove(legacyLatest); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to remove legacy latest symlink: %v\n", err)
		}
	}

	fmt.Printf("\n✅ Successfully migrated %d handoff(s) to window-scoped structure\n", migratedCount)
	fmt.Printf("   Window: %s\n", windowName)
	fmt.Printf("   Location: .orch/session/%s/\n", windowName)

	return nil
}
