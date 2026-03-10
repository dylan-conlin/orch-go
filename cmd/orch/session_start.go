package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/control"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/tree"
	"github.com/spf13/cobra"
)

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
	// Auto-lock control plane at session start to ensure agents spawned
	// during this session can't modify settings.json or enforcement hooks.
	if n, err := control.EnsureLocked(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to lock control plane: %v\n", err)
	} else if n > 0 {
		fmt.Fprintf(os.Stderr, "Control plane: locked %d unlocked files\n", n)
	}

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
