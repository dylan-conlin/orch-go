// Package main provides the CLI entry point for orch-go.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/tree"
	// spawn import moved to session_handoff.go
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
	validateJSON       bool
)

func init() {
	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStatusCmd)
	sessionCmd.AddCommand(sessionEndCmd)
	sessionCmd.AddCommand(sessionResumeCmd)
	sessionCmd.AddCommand(sessionMigrateCmd)
	sessionCmd.AddCommand(sessionValidateCmd)
	sessionCmd.AddCommand(sessionLabelCmd)

	// Add --json flag to status command
	sessionStatusCmd.Flags().BoolVar(&sessionJSON, "json", false, "Output as JSON")

	// Add flags for resume command
	sessionResumeCmd.Flags().BoolVar(&resumeForInjection, "for-injection", false, "Output condensed format for hook injection")
	sessionResumeCmd.Flags().BoolVar(&resumeCheck, "check", false, "Check if handoff exists (exit code only)")

	// Add --json flag for validate command
	sessionValidateCmd.Flags().BoolVar(&validateJSON, "json", false, "Output as JSON")

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

	// Get current working directory (project directory)
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %w", err)
	}

	// Generate session name in format {project}-{count}
	sessionName, err := session.GenerateSessionName(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to generate session name: %v\n", err)
		// Fall back to timestamp-based name
		sessionName = fmt.Sprintf("session-%s", time.Now().Format("20060102-1504"))
	}

	// Rename tmux window to match session name (auto-naming pattern)
	if err := tmux.RenameCurrentWindow(sessionName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to rename tmux window: %v\n", err)
		// Continue anyway - window renaming is nice-to-have
	}

	// Capture the window name AFTER renaming - this is the name used for session directories
	// We store this in the session so that session end can archive to the correct directory
	// even if called from a different tmux window
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		// Fall back to session name if we can't get window name
		windowName = sessionName
	}

	// Create active session handoff in project-specific location
	// This replaces the global ~/.orch workspace with project/.orch/session/{sessionName}/active/
	handoffPath, err := createActiveSessionHandoff(goal, sessionName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create active session handoff: %v\n", err)
		// Continue anyway - handoff is nice-to-have for interactive sessions
	}

	// Derive workspace path from handoff path (handoff is in the workspace directory)
	// If handoff creation failed, workspace path will be empty string
	workspacePath := ""
	if handoffPath != "" {
		workspacePath = filepath.Dir(handoffPath)
	}

	// Start the session with the captured window name and workspace path
	if err := store.Start(goal, windowName, workspacePath); err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	// Progressive Session Capture: Prompt for TLDR and Where We Started
	// Part of decision 2026-01-14 - capture context when it's freshest
	if handoffPath != "" {
		responses, err := promptForStartSections()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to collect start section responses: %v\n", err)
			// Continue anyway - we can still end the session and prompt for these later
		} else if len(responses) > 0 {
			if err := updateHandoffWithStartResponses(handoffPath, responses); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to update handoff with responses: %v\n", err)
			} else {
				fmt.Println("   ✅ Initial context captured in SESSION_HANDOFF.md")
			}
		}
	}

	// Log the session start
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"goal":         goal,
		"was_active":   wasActive,
		"started_at":   time.Now().Format(session.TimeFormat),
		"session_name": sessionName,
	}
	if handoffPath != "" {
		eventData["handoff_path"] = handoffPath
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
	fmt.Printf("  Name:       %s\n", sessionName)
	fmt.Printf("  Start time: %s\n", time.Now().Format("15:04"))
	if handoffPath != "" {
		fmt.Printf("  Handoff:    %s\n", handoffPath)
	}

	// Surface reflection suggestions for high-count synthesis opportunities
	// This proactively surfaces consolidation needs that accumulated since last reflection
	surfaceReflectSuggestions()

	// Surface focus guidance - group ready issues into thematic threads
	// Part of Capture at Context principle - surface context when it matters
	surfaceFocusGuidance()

	// Surface tree summary - show work view with health smells
	// Gives cluster awareness and health smell triage without extra commands
	surfaceTreeSummary()

	return nil
}

// SynthesisWarningThreshold is the minimum count of investigations to show a warning.
// Matches SynthesisIssueThreshold in kb-cli to maintain consistency.
const SynthesisWarningThreshold = 10

// SuggestionFreshnessHours is the maximum age of suggestions to consider fresh.
// Suggestions older than this are considered stale and won't be shown.
const SuggestionFreshnessHours = 24

// InvestigationPromotionThreshold is the count above which session end will warn.
// Gates accumulation of promotion candidates that need triage.
const InvestigationPromotionThreshold = 5

// surfaceFocusGuidance loads ready issues and displays them grouped into thematic threads.
// This helps orchestrators orient at session start: "Here are your active threads. What's nagging you?"
// Part of Capture at Context principle.
func surfaceFocusGuidance() {
	guidance, err := focus.GenerateFocusGuidance()
	if err != nil {
		// Failed to load issues - silently skip (not critical)
		return
	}

	if guidance.TotalIssues == 0 {
		// No ready issues - brief message only
		fmt.Println("\n📋 No ready issues found")
		return
	}

	// Display formatted guidance
	fmt.Print(focus.FormatFocusGuidance(guidance))
}

// surfaceTreeSummary displays orch tree summary with work view and health smells.
// This gives orchestrators cluster awareness and health smell triage at session start.
func surfaceTreeSummary() {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		// Failed to get cwd - silently skip (not critical)
		return
	}

	// Find .kb/ directory
	kbDir := filepath.Join(cwd, ".kb")
	if _, err := os.Stat(kbDir); os.IsNotExist(err) {
		// No .kb/ directory - silently skip (not all projects have knowledge bases)
		return
	}

	// Build tree with work view, smells only, and compact format
	opts := tree.TreeOptions{
		Depth:      2,
		Format:     "text",
		WorkView:   true,
		SmellsOnly: true,
		Compact:    true,
	}

	issues, err := tree.BuildWorkTree(kbDir, cwd, opts)
	if err != nil {
		// Failed to build tree - silently skip (not critical)
		return
	}

	output, err := tree.RenderWorkView(issues, opts)
	if err != nil {
		// Failed to render - silently skip (not critical)
		return
	}

	// Display output with a blank line before it
	fmt.Println()
	fmt.Print(output)
}

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

// InvestigationPromotionItem represents a single investigation promotion candidate.
type InvestigationPromotionItem struct {
	File       string `json:"file"`
	Title      string `json:"title"`
	AgeDays    int    `json:"age_days"`
	Suggestion string `json:"suggestion"`
}

// InvestigationPromotionResult holds the JSON output from kb reflect --type investigation-promotion.
type InvestigationPromotionResult struct {
	InvestigationPromotion []InvestigationPromotionItem `json:"investigation_promotion"`
}

// checkInvestigationPromotions runs kb reflect --type investigation-promotion --format json
// and returns the count of promotion candidates. Returns 0 and logs warning on error.
func checkInvestigationPromotions() int {
	cmd := exec.Command("kb", "reflect", "--type", "investigation-promotion", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		// kb reflect may not be available or may fail - not critical, just skip
		return 0
	}

	var result InvestigationPromotionResult
	if err := json.Unmarshal(output, &result); err != nil {
		// Parse error - skip silently
		return 0
	}

	return len(result.InvestigationPromotion)
}

// gateInvestigationPromotions checks for accumulated investigation promotion candidates
// and prompts user to triage if above threshold. Returns error if user aborts.
// This is a gate at session end to prevent accumulation of promotion candidates.
func gateInvestigationPromotions() error {
	count := checkInvestigationPromotions()
	if count <= InvestigationPromotionThreshold {
		return nil // Below threshold, proceed
	}

	fmt.Println()
	fmt.Printf("⚠️  INVESTIGATION PROMOTION BACKLOG\n")
	fmt.Printf("   %d investigations need promotion review (threshold: %d)\n", count, InvestigationPromotionThreshold)
	fmt.Printf("   Run 'kb reflect --type investigation-promotion' to triage.\n")
	fmt.Println()

	// Prompt user to confirm proceeding
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("   Continue ending session anyway? (y/N): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println("   Session end aborted. Please triage investigation promotions first.")
		return fmt.Errorf("session end aborted: investigation promotion backlog needs triage")
	}

	return nil
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
	Active        bool                      `json:"active"`
	Goal          string                    `json:"goal,omitempty"`
	StartedAt     string                    `json:"started_at,omitempty"`
	Duration      string                    `json:"duration,omitempty"`
	WorkspacePath string                    `json:"workspace_path,omitempty"`
	Spawns        []session.SpawnStatus     `json:"spawns,omitempty"`
	Counts        *SpawnCounts              `json:"counts,omitempty"`
	Checkpoint    *session.CheckpointStatus `json:"checkpoint,omitempty"`
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
		output.WorkspacePath = sess.WorkspacePath

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
	fmt.Printf("  Goal:      %s\n", output.Goal)
	fmt.Printf("  Duration:  %s", output.Duration)

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

	if output.WorkspacePath != "" {
		fmt.Printf("  Workspace: %s\n", output.WorkspacePath)
	}

	fmt.Printf("  Spawns:    %d total", output.Counts.Total)
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

	// Gate: Check for accumulated investigation promotion candidates
	// This prevents backlog accumulation by prompting triage before session end
	if err := gateInvestigationPromotions(); err != nil {
		return err
	}

	// Get session info before ending - IMPORTANT: Get the session object to access WindowName
	// which was captured at session start. This is used for archiving, NOT GetCurrentWindowName().
	sess := store.Get()
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

	// Get project directory
	projectDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get project directory: %v\n", err)
	} else {
		// Use the stored window name from session start, NOT GetCurrentWindowName()
		// This ensures we archive to the correct directory even if called from a different window
		windowName := sess.WindowName
		if windowName == "" {
			// Fallback for sessions created before WindowName was added
			windowName, err = tmux.GetCurrentWindowName()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to get window name: %v\n", err)
				windowName = "default"
			}
		}

		// Complete and archive the session handoff
		// This validates unfilled sections, prompts for completion, then archives
		if err := completeAndArchiveHandoff(projectDir, windowName); err != nil {
			// Only warn - not all sessions will have active handoffs (pre-active-pattern sessions)
			fmt.Fprintf(os.Stderr, "Warning: failed to complete/archive session handoff: %v\n", err)
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

// ============================================================================
// Session Validate Command
// ============================================================================

var sessionValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Show unfilled handoff sections without ending session",
	Long: `Validate SESSION_HANDOFF.md quality by showing unfilled sections.

This command checks the active session handoff for placeholder patterns
and displays which sections still need to be filled. Unlike 'session end',
it does NOT prompt for input or archive the handoff.

Use cases:
- Check handoff quality mid-session
- Debug validation logic
- Verify handoff is ready before ending session

The command looks for the active session handoff in:
  .orch/session/{window-name}/active/SESSION_HANDOFF.md

If no active handoff exists, it reports that state.

Examples:
  orch session validate          # Human-readable output
  orch session validate --json   # Machine-readable output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionValidate()
	},
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

// ============================================================================
// Session Label Command
// ============================================================================

var sessionLabelCmd = &cobra.Command{
	Use:   "label [name]",
	Short: "Set a human-readable label for the current OpenCode session",
	Long: `Set a human-readable label for the current OpenCode session.

This label will be used in the dashboard timeline view to identify
the session instead of showing the raw session ID (ses_xxxxx).

The label is stored in the .orch workspace for the current project
and is associated with the OpenCode session ID from the environment.

Examples:
  orch session label "verifiability design review"
  orch session label "bug fixes"
  orch session label "dashboard timeline feature"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		label := strings.Join(args, " ")
		return runSessionLabel(label)
	},
}

func runSessionLabel(label string) error {
	// Get current OpenCode session ID from environment
	sessionID := os.Getenv("CLAUDE_SESSION_ID")
	if sessionID == "" {
		return fmt.Errorf("no OpenCode session detected (CLAUDE_SESSION_ID not set)")
	}

	// Get current working directory (project directory)
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %w", err)
	}

	// Ensure .orch directory exists
	orchDir := filepath.Join(projectDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		return fmt.Errorf("failed to create .orch directory: %w", err)
	}

	// Store session label in .orch/session_labels.json
	labelsFile := filepath.Join(orchDir, "session_labels.json")

	// Load existing labels
	labels := make(map[string]string)
	if data, err := os.ReadFile(labelsFile); err == nil {
		json.Unmarshal(data, &labels)
	}

	// Add/update label for this session
	labels[sessionID] = label

	// Write back to file
	data, err := json.MarshalIndent(labels, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode labels: %w", err)
	}

	if err := os.WriteFile(labelsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write labels file: %w", err)
	}

	// Log the session label event
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.labeled",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"session_id": sessionID,
			"label":      label,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("✅ Session labeled: %s\n", label)
	fmt.Printf("   Session ID: %s\n", sessionID)

	return nil
}

// ============================================================================
