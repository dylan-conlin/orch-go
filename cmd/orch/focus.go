// Package main provides the CLI entry point for orch-go.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/spf13/cobra"
)

// ============================================================================
// Focus Command - Set/Get/Clear north star priority
// ============================================================================

var focusCmd = &cobra.Command{
	Use:   "focus [goal]",
	Short: "Set or view the current north star priority",
	Long: `Set or view the current north star priority for multi-project work.

When called without arguments, displays the current focus.
When called with a goal, sets it as the new focus.

The focus helps orchestrators stay aligned with priorities and avoid drift.

Examples:
  orch-go focus                           # View current focus
  orch-go focus "Ship snap MVP"           # Set new focus
  orch-go focus "Fix auth bugs" --issue proj-123  # Set focus with beads issue
  orch-go focus clear                     # Clear current focus`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return runFocusGet()
		}
		goal := strings.Join(args, " ")
		if goal == "clear" {
			return runFocusClear()
		}
		return runFocusSet(goal)
	},
}

var (
	focusIssue string
	focusJSON  bool
)

var focusClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the current focus",
	Long:  "Clear the current focus. Use when changing priorities or ending a focused session.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFocusClear()
	},
}

func init() {
	focusCmd.AddCommand(focusClearCmd)
	focusCmd.Flags().StringVar(&focusIssue, "issue", "", "Beads issue ID to associate with focus")
	focusCmd.Flags().BoolVar(&focusJSON, "json", false, "Output in JSON format")
}

func runFocusGet() error {
	store, err := focus.New("")
	if err != nil {
		return fmt.Errorf("failed to load focus: %w", err)
	}

	f := store.Get()
	if f == nil {
		if focusJSON {
			fmt.Println("{}")
			return nil
		}
		fmt.Println("No focus set")
		fmt.Println("\nSet a focus with: orch-go focus \"your goal\"")
		return nil
	}

	if focusJSON {
		data, err := json.MarshalIndent(f, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal focus: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Parse and format the time
	setAt := f.SetAt
	if t, err := time.Parse(focus.TimeFormat, f.SetAt); err == nil {
		setAt = t.Format("2006-01-02 15:04:05")
	}

	fmt.Printf("Current focus:\n")
	fmt.Printf("  Goal:    %s\n", f.Goal)
	if f.BeadsID != "" {
		fmt.Printf("  Issue:   %s\n", f.BeadsID)
	}
	fmt.Printf("  Set at:  %s\n", setAt)

	return nil
}

func runFocusSet(goal string) error {
	store, err := focus.New("")
	if err != nil {
		return fmt.Errorf("failed to load focus: %w", err)
	}

	f := &focus.Focus{
		Goal:    goal,
		BeadsID: focusIssue,
	}

	if err := store.Set(f); err != nil {
		return fmt.Errorf("failed to set focus: %w", err)
	}

	// Log the focus change
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "focus.set",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"goal":     goal,
			"beads_id": focusIssue,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("Focus set: %s\n", goal)
	if focusIssue != "" {
		fmt.Printf("  Issue: %s\n", focusIssue)
	}

	return nil
}

func runFocusClear() error {
	store, err := focus.New("")
	if err != nil {
		return fmt.Errorf("failed to load focus: %w", err)
	}

	// Check if there's a focus to clear
	f := store.Get()
	if f == nil {
		fmt.Println("No focus to clear")
		return nil
	}

	if err := store.Clear(); err != nil {
		return fmt.Errorf("failed to clear focus: %w", err)
	}

	// Log the focus clear
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "focus.cleared",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"previous_goal": f.Goal,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Println("Focus cleared")
	return nil
}

// ============================================================================
// Drift Command - Check if active work aligns with focus
// ============================================================================

var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Check if current work aligns with focus",
	Long: `Check if current work aligns with the north star focus.

Queries tracked agents via beads and groups by skill, showing task titles
and phases. Compares against focus to detect drift.

Examples:
  orch-go drift         # Alignment analysis
  orch-go drift --json  # Output in JSON format`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDrift()
	},
}

var driftJSON bool

func init() {
	driftCmd.Flags().BoolVar(&driftJSON, "json", false, "Output in JSON format")
}

// DriftAnalysis is the rich output type for the drift command.
type DriftAnalysis struct {
	Goal           string            `json:"goal,omitempty"`
	FocusedIssue   string            `json:"focused_issue,omitempty"`
	IsDrifting     bool              `json:"is_drifting"`
	Verdict        string            `json:"verdict"` // "on-track", "drifting", "unverified", "no-focus"
	Reason         string            `json:"reason"`
	Groups         []DriftSkillGroup `json:"groups,omitempty"`
	AgentCount     int               `json:"agent_count"`
	UntrackedCount int               `json:"untracked_count"`
}

// DriftSkillGroup groups agents by skill for display.
type DriftSkillGroup struct {
	Skill  string       `json:"skill"`
	Agents []DriftAgent `json:"agents"`
}

// DriftAgent is a compact representation of an agent for drift output.
type DriftAgent struct {
	BeadsID string `json:"beads_id"`
	Title   string `json:"title"`
	Phase   string `json:"phase,omitempty"`
	Status  string `json:"status"`
}

func runDrift() error {
	store, err := focus.New("")
	if err != nil {
		return fmt.Errorf("failed to load focus: %w", err)
	}

	// Get current project directory
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Query tracked agents (beads-first, with workspace manifests and liveness)
	projectDirs := uniqueProjectDirs(append([]string{projectDir}, getKBProjectsFn()...))
	trackedAgents, err := queryTrackedAgents(projectDirs)
	if err != nil {
		return fmt.Errorf("failed to query tracked agents: %w", err)
	}

	// Count untracked sessions for context
	untrackedCount := countUntrackedSessions(projectDir)

	// Build ActiveWork for drift check
	activeWork := make([]focus.ActiveWork, 0, len(trackedAgents))
	for _, agent := range trackedAgents {
		activeWork = append(activeWork, focus.ActiveWork{
			BeadsID: agent.BeadsID,
			Title:   agent.Title,
		})
	}

	// Check drift against focus
	driftResult := store.CheckDrift(activeWork)

	// Build analysis
	analysis := buildDriftAnalysis(driftResult, trackedAgents, untrackedCount)

	if driftJSON {
		data, err := json.MarshalIndent(analysis, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	printDriftAnalysis(analysis)
	return nil
}

// buildDriftAnalysis creates a rich analysis from tracked agents and drift result.
func buildDriftAnalysis(driftResult focus.DriftResult, agents []AgentStatus, untrackedCount int) DriftAnalysis {
	analysis := DriftAnalysis{
		Goal:           driftResult.Goal,
		FocusedIssue:   driftResult.FocusedIssue,
		IsDrifting:     driftResult.IsDrifting,
		Verdict:        driftResult.Verdict,
		Reason:         driftResult.Reason,
		AgentCount:     len(agents),
		UntrackedCount: untrackedCount,
	}

	// Group agents by skill
	skillMap := make(map[string][]DriftAgent)
	for _, agent := range agents {
		skill := agent.Skill
		if skill == "" {
			skill = "(unknown)"
		}
		da := DriftAgent{
			BeadsID: agent.BeadsID,
			Title:   agent.Title,
			Phase:   agent.Phase,
			Status:  agent.Status,
		}
		skillMap[skill] = append(skillMap[skill], da)
	}

	// Sort skill groups by count (descending), then name
	type kv struct {
		skill  string
		agents []DriftAgent
	}
	var sorted []kv
	for skill, agents := range skillMap {
		sorted = append(sorted, kv{skill, agents})
	}
	// Sort: largest groups first, then alphabetically
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if len(sorted[j].agents) > len(sorted[i].agents) ||
				(len(sorted[j].agents) == len(sorted[i].agents) && sorted[j].skill < sorted[i].skill) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	for _, kv := range sorted {
		analysis.Groups = append(analysis.Groups, DriftSkillGroup{
			Skill:  kv.skill,
			Agents: kv.agents,
		})
	}

	return analysis
}

// printDriftAnalysis formats the analysis for terminal output.
func printDriftAnalysis(a DriftAnalysis) {
	// No focus set
	if a.Verdict == "no-focus" {
		fmt.Println("No focus set - nothing to drift from")
		if a.AgentCount > 0 {
			fmt.Printf("\n%d tracked agents active (set focus to check alignment)\n", a.AgentCount)
		}
		fmt.Println("\nSet a focus with: orch-go focus \"your goal\"")
		return
	}

	// Header with verdict
	switch a.Verdict {
	case "drifting":
		fmt.Println("DRIFT DETECTED")
	case "unverified":
		fmt.Println("ALIGNMENT UNVERIFIED (goal-only focus)")
	default:
		fmt.Println("ON TRACK")
	}
	fmt.Printf("  Focus: %s\n", truncate(a.Goal, 100))
	if a.FocusedIssue != "" {
		fmt.Printf("  Target: %s\n", a.FocusedIssue)
	}

	// No active work
	if a.AgentCount == 0 {
		fmt.Println("\n  (no tracked agents)")
		if a.UntrackedCount > 0 {
			fmt.Printf("  %d untracked sessions (use 'orch sessions' to view)\n", a.UntrackedCount)
		}
		return
	}

	// Agent groups by skill
	fmt.Printf("\nActive Work (%d tracked):\n", a.AgentCount)
	for _, group := range a.Groups {
		fmt.Printf("\n  %s (%d):\n", group.Skill, len(group.Agents))
		for _, agent := range group.Agents {
			phase := agent.Phase
			if phase == "" {
				phase = agent.Status
			}
			// Truncate phase to just the first part (before " - ")
			if idx := strings.Index(phase, " - "); idx > 0 {
				phase = phase[:idx]
			}
			fmt.Printf("    %-16s  %-50s  %s\n", agent.BeadsID, truncate(agent.Title, 50), phase)
		}
	}

	// Footer
	if a.UntrackedCount > 0 {
		fmt.Printf("\n  + %d untracked sessions (use 'orch sessions' to view)\n", a.UntrackedCount)
	}
}

// countUntrackedSessions counts OpenCode sessions that don't map to tracked beads issues.
func countUntrackedSessions(projectDir string) int {
	client := opencode.NewClient(opencode.DefaultServerURL)
	sessions, err := client.ListSessions(projectDir)
	if err != nil {
		return 0
	}

	count := 0
	for _, s := range sessions {
		if extractBeadsIDFromTitle(s.Title) == "" {
			count++
		}
	}
	return count
}

// getActiveWork returns active work items from OpenCode sessions.
// Extracts beads IDs from session titles and returns them as ActiveWork.
func getActiveWork() []focus.ActiveWork {
	// Get current directory for project context
	projectDir, err := os.Getwd()
	if err != nil {
		return nil
	}

	// Use default OpenCode server URL
	client := opencode.NewClient("http://127.0.0.1:4096")
	sessions, err := client.ListSessions(projectDir)
	if err != nil {
		return nil
	}

	var work []focus.ActiveWork
	for _, session := range sessions {
		// Extract beads ID from session title (format: "workspace [beads-id]")
		beadsID := extractBeadsIDFromTitle(session.Title)
		if beadsID != "" {
			work = append(work, focus.ActiveWork{BeadsID: beadsID})
		}
	}

	return work
}

// ============================================================================
// Next Command - Suggest next action based on current state
// ============================================================================

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Suggest next action based on current focus and state",
	Long: `Suggest the next action based on current focus and active work.

Recommendations include:
  - set-focus:  No focus set, suggest setting one
  - start-work: Focus set but no active work, suggest starting
  - continue:   Already working on focused issue
  - refocus:    Working on something else, suggest switching

Examples:
  orch-go next         # Get suggestion
  orch-go next --json  # Output in JSON format`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNext()
	},
}

var nextJSON bool

func init() {
	nextCmd.Flags().BoolVar(&nextJSON, "json", false, "Output in JSON format")
}

func runNext() error {
	store, err := focus.New("")
	if err != nil {
		return fmt.Errorf("failed to load focus: %w", err)
	}

	// Get active work from OpenCode sessions
	activeWork := getActiveWork()

	// Get ready issues from beads for additional context
	readyIssues := getReadyIssues()

	suggestion := store.SuggestNext(activeWork)

	if nextJSON {
		data, err := json.MarshalIndent(suggestion, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal suggestion: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Format output based on action type
	switch suggestion.Action {
	case "set-focus":
		fmt.Printf("📋 %s\n", suggestion.Description)
		if len(readyIssues) > 0 {
			fmt.Printf("\nReady issues:\n")
			for _, issue := range readyIssues[:min(5, len(readyIssues))] {
				fmt.Printf("  - %s\n", issue)
			}
		}

	case "start-work":
		fmt.Printf("🚀 %s\n", suggestion.Description)
		if suggestion.BeadsID != "" {
			fmt.Printf("\nStart with: orch-go work %s\n", suggestion.BeadsID)
		}

	case "continue":
		fmt.Printf("✅ %s\n", suggestion.Description)
		if suggestion.Goal != "" {
			fmt.Printf("   Goal: %s\n", suggestion.Goal)
		}

	case "refocus":
		fmt.Printf("🔄 %s\n", suggestion.Description)
		if suggestion.BeadsID != "" {
			fmt.Printf("\nSwitch with: orch-go work %s\n", suggestion.BeadsID)
		}
		fmt.Printf("Or clear focus: orch-go focus clear\n")

	default:
		fmt.Printf("%s: %s\n", suggestion.Action, suggestion.Description)
	}

	return nil
}

// getReadyIssues returns beads issues that are ready for work.
// It uses the beads RPC client when available, falling back to the bd CLI.
func getReadyIssues() []string {
	var issues []string

	// Try RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()

			readyIssues, err := client.Ready(nil)
			if err == nil {
				for _, issue := range readyIssues {
					issues = append(issues, issue.ID)
				}
				return issues
			}
			// Fall through to CLI fallback on RPC error
		}
	}

	// Fallback to CLI
	readyIssues, err := beads.FallbackReady()
	if err != nil {
		return nil
	}

	for _, issue := range readyIssues {
		issues = append(issues, issue.ID)
	}

	return issues
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
