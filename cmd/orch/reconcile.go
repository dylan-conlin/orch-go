// Package main provides the CLI entry point for orch-go.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

var (
	// Reconcile command flags
	reconcileJSON     bool
	reconcileFix      bool   // Apply fixes non-interactively
	reconcileFixMode  string // "reset" or "close"
	reconcileFixAll   bool   // Fix all zombies without prompting
	reconcileProject  string // Filter by project
	reconcileMinAge   int    // Minimum age in hours to consider zombie
	reconcileMaxShown int    // Maximum number of zombies to show
)

// ZombieIssue represents an in_progress issue with no active agent.
type ZombieIssue struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Project         string    `json:"project"`
	Status          string    `json:"status"`
	Priority        int       `json:"priority"`
	Labels          []string  `json:"labels"`
	UpdatedAt       time.Time `json:"updated_at"`
	AgeSinceUpdate  string    `json:"age_since_update"`
	HoursSinceUpdate float64  `json:"hours_since_update"`
	HasWorkspace    bool      `json:"has_workspace"`
	WorkspacePath   string    `json:"workspace_path,omitempty"`
	LastPhase       string    `json:"last_phase,omitempty"`
}

var reconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Detect and fix zombie in_progress issues",
	Long: `Detect in_progress issues that have no active agent and offer to fix them.

A "zombie" issue is one with status=in_progress but no corresponding:
  - Active OpenCode session (updated within 30 minutes)
  - Active tmux window

This can happen when:
  - Agent crashed or was killed without completing
  - Session timed out without state transition
  - 'orch complete' was never run after agent finished

The reconcile command cross-references beads issues against OpenCode sessions
and tmux windows to identify orphaned work.

Actions:
  reset  - Set status back to 'open' (default - allows re-spawning)
  close  - Close the issue (for work that should be abandoned)

Examples:
  orch reconcile                      # Show zombie issues
  orch reconcile --json               # Output as JSON
  orch reconcile --fix                # Interactive fix mode
  orch reconcile --fix --all          # Fix all without prompting
  orch reconcile --fix --mode close   # Close all zombies
  orch reconcile --min-age 24         # Only issues stale for 24+ hours
  orch reconcile -p orch-go           # Filter by project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReconcile()
	},
}

func init() {
	reconcileCmd.Flags().BoolVar(&reconcileJSON, "json", false, "Output as JSON")
	reconcileCmd.Flags().BoolVar(&reconcileFix, "fix", false, "Apply fixes (interactive by default)")
	reconcileCmd.Flags().StringVar(&reconcileFixMode, "mode", "reset", "Fix mode: 'reset' (set to open) or 'close'")
	reconcileCmd.Flags().BoolVar(&reconcileFixAll, "all", false, "Fix all zombies without prompting")
	reconcileCmd.Flags().StringVarP(&reconcileProject, "project", "p", "", "Filter by project")
	reconcileCmd.Flags().IntVar(&reconcileMinAge, "min-age", 1, "Minimum hours since last update to consider zombie")
	reconcileCmd.Flags().IntVar(&reconcileMaxShown, "limit", 50, "Maximum number of zombies to show")

	rootCmd.AddCommand(reconcileCmd)
}

func runReconcile() error {
	projectDir, _ := os.Getwd()

	// 1. Get all in_progress issues from beads
	zombies, err := findZombieIssues(projectDir)
	if err != nil {
		return fmt.Errorf("failed to find zombie issues: %w", err)
	}

	// Apply project filter
	if reconcileProject != "" {
		filtered := make([]ZombieIssue, 0)
		for _, z := range zombies {
			if z.Project == reconcileProject {
				filtered = append(filtered, z)
			}
		}
		zombies = filtered
	}

	// Apply minimum age filter
	if reconcileMinAge > 0 {
		filtered := make([]ZombieIssue, 0)
		for _, z := range zombies {
			if z.HoursSinceUpdate >= float64(reconcileMinAge) {
				filtered = append(filtered, z)
			}
		}
		zombies = filtered
	}

	// Sort by age (oldest first)
	sort.Slice(zombies, func(i, j int) bool {
		return zombies[i].HoursSinceUpdate > zombies[j].HoursSinceUpdate
	})

	// Apply limit
	if reconcileMaxShown > 0 && len(zombies) > reconcileMaxShown {
		zombies = zombies[:reconcileMaxShown]
	}

	// JSON output mode
	if reconcileJSON {
		data, err := json.MarshalIndent(zombies, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// No zombies found
	if len(zombies) == 0 {
		fmt.Println("✨ No zombie issues found - all in_progress issues have active agents")
		return nil
	}

	// Display zombies
	fmt.Printf("🧟 Found %d zombie issue(s) (in_progress with no active agent):\n\n", len(zombies))

	for i, z := range zombies {
		priorityStr := fmt.Sprintf("P%d", z.Priority)
		if z.Priority == 0 {
			priorityStr = "P?"
		}

		labelsStr := ""
		if len(z.Labels) > 0 {
			labelsStr = " [" + strings.Join(z.Labels, ", ") + "]"
		}

		workspaceStr := ""
		if z.HasWorkspace {
			workspaceStr = " 📁"
		}

		phaseStr := ""
		if z.LastPhase != "" {
			phaseStr = fmt.Sprintf(" (Last: %s)", z.LastPhase)
		}

		fmt.Printf("  %d. [%s] %s: %s%s%s%s\n", i+1, priorityStr, z.ID, z.Title, labelsStr, workspaceStr, phaseStr)
		fmt.Printf("     Age: %s (last updated: %s)\n\n", z.AgeSinceUpdate, z.UpdatedAt.Format("2006-01-02 15:04"))
	}

	// If not in fix mode, show hint
	if !reconcileFix {
		fmt.Println("Run with --fix to repair these issues, or --json for scripting")
		return nil
	}

	// Fix mode
	return runReconcileFix(zombies)
}

// findZombieIssues finds in_progress issues with no active agent.
func findZombieIssues(projectDir string) ([]ZombieIssue, error) {
	// Get beads client
	socketPath, err := beads.FindSocketPath("")
	if err != nil {
		// Try fallback to CLI
		return findZombieIssuesFallback(projectDir)
	}

	client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
	if err := client.Connect(); err != nil {
		return findZombieIssuesFallback(projectDir)
	}
	defer client.Close()

	// List all in_progress issues
	inProgress := "in_progress"
	issues, err := client.List(&beads.ListArgs{
		Status: inProgress,
	})
	if err != nil {
		return findZombieIssuesFallback(projectDir)
	}

	return filterZombies(projectDir, issues)
}

// findZombieIssuesFallback uses bd CLI as fallback.
func findZombieIssuesFallback(projectDir string) ([]ZombieIssue, error) {
	issues, err := beads.FallbackList("in_progress")
	if err != nil {
		return nil, fmt.Errorf("failed to list in_progress issues: %w", err)
	}
	return filterZombies(projectDir, issues)
}

// filterZombies filters issues to find those without active agents.
func filterZombies(projectDir string, issues []beads.Issue) ([]ZombieIssue, error) {
	now := time.Now()
	zombies := make([]ZombieIssue, 0)

	// Get OpenCode client
	ocClient := opencode.NewClient(serverURL)

	// Get all active OpenCode sessions
	var activeSessions []opencode.Session
	seenSessionIDs := make(map[string]bool)

	// Query current project directory first
	if projectDir != "" {
		dirSessions, err := ocClient.ListSessions(projectDir)
		if err == nil {
			for _, s := range dirSessions {
				if !seenSessionIDs[s.ID] {
					seenSessionIDs[s.ID] = true
					activeSessions = append(activeSessions, s)
				}
			}
		}
	}

	// Also query global sessions
	globalSessions, _ := ocClient.ListSessions("")
	for _, s := range globalSessions {
		if !seenSessionIDs[s.ID] {
			seenSessionIDs[s.ID] = true
			activeSessions = append(activeSessions, s)
		}
	}

	// Build map of beads ID -> active session
	const maxIdleTime = 30 * time.Minute
	activeBeadsIDs := make(map[string]bool)

	for _, s := range activeSessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) <= maxIdleTime {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID != "" {
				activeBeadsIDs[beadsID] = true
			}
		}
	}

	// Get all tmux worker windows
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		windows, _ := tmux.ListWindows(sessionName)
		for _, w := range windows {
			beadsID := extractBeadsIDFromWindowName(w.Name)
			if beadsID != "" {
				activeBeadsIDs[beadsID] = true
			}
		}
	}

	// Check each in_progress issue
	for _, issue := range issues {
		// Skip if there's an active agent
		if activeBeadsIDs[issue.ID] {
			continue
		}

		// Parse updated_at
		var updatedAt time.Time
		if issue.UpdatedAt != "" {
			t, err := time.Parse(time.RFC3339, issue.UpdatedAt)
			if err != nil {
				t, _ = time.Parse("2006-01-02T15:04:05", issue.UpdatedAt)
			}
			updatedAt = t
		}
		if updatedAt.IsZero() {
			updatedAt = now // Fallback to now if no timestamp
		}

		hoursSinceUpdate := now.Sub(updatedAt).Hours()

		// Check for workspace
		liveness := state.GetLiveness(issue.ID, serverURL, projectDir)

		// Extract last phase from comments (if available)
		lastPhase := getLastPhase(issue.ID)

		zombie := ZombieIssue{
			ID:               issue.ID,
			Title:            issue.Title,
			Project:          extractProjectFromBeadsID(issue.ID),
			Status:           issue.Status,
			Priority:         issue.Priority,
			Labels:           issue.Labels,
			UpdatedAt:        updatedAt,
			AgeSinceUpdate:   formatDuration(now.Sub(updatedAt)),
			HoursSinceUpdate: hoursSinceUpdate,
			HasWorkspace:     liveness.WorkspaceExists,
			WorkspacePath:    liveness.WorkspacePath,
			LastPhase:        lastPhase,
		}

		zombies = append(zombies, zombie)
	}

	return zombies, nil
}

// getLastPhase extracts the last phase from beads comments.
func getLastPhase(beadsID string) string {
	// Try to get comments
	socketPath, err := beads.FindSocketPath("")
	if err != nil {
		return ""
	}

	client := beads.NewClient(socketPath, beads.WithAutoReconnect(1))
	if err := client.Connect(); err != nil {
		return ""
	}
	defer client.Close()

	comments, err := client.Comments(beadsID)
	if err != nil {
		return ""
	}

	// Find the last "Phase:" comment
	for i := len(comments) - 1; i >= 0; i-- {
		text := comments[i].Text
		if strings.HasPrefix(text, "Phase:") {
			// Extract just the phase name
			parts := strings.SplitN(text, " - ", 2)
			if len(parts) > 0 {
				phase := strings.TrimPrefix(parts[0], "Phase: ")
				phase = strings.TrimPrefix(phase, "Phase:")
				return strings.TrimSpace(phase)
			}
			return text
		}
	}

	return ""
}

// runReconcileFix applies fixes to zombie issues.
func runReconcileFix(zombies []ZombieIssue) error {
	if len(zombies) == 0 {
		return nil
	}

	reader := bufio.NewReader(os.Stdin)

	for _, z := range zombies {
		if reconcileFixAll {
			// Apply fix without prompting
			if err := applyFix(z, reconcileFixMode, ""); err != nil {
				fmt.Printf("  ❌ Failed to fix %s: %v\n", z.ID, err)
			} else {
				fmt.Printf("  ✅ Fixed %s (%s)\n", z.ID, reconcileFixMode)
			}
			continue
		}

		// Interactive prompt
		fmt.Printf("Fix %s: %s?\n", z.ID, z.Title)
		fmt.Printf("  [r]eset to open  [c]lose  [s]kip  [a]ll-reset  [q]uit: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "r", "reset", "":
			if err := applyFix(z, "reset", ""); err != nil {
				fmt.Printf("  ❌ Failed to reset: %v\n", err)
			} else {
				fmt.Printf("  ✅ Reset to open\n")
			}
		case "c", "close":
			fmt.Print("  Reason (optional): ")
			reason, _ := reader.ReadString('\n')
			reason = strings.TrimSpace(reason)
			if err := applyFix(z, "close", reason); err != nil {
				fmt.Printf("  ❌ Failed to close: %v\n", err)
			} else {
				fmt.Printf("  ✅ Closed\n")
			}
		case "s", "skip":
			fmt.Printf("  ⏭️  Skipped\n")
		case "a", "all":
			// Reset this one and all remaining
			if err := applyFix(z, "reset", ""); err != nil {
				fmt.Printf("  ❌ Failed to reset: %v\n", err)
			} else {
				fmt.Printf("  ✅ Reset to open\n")
			}
			// Set flag to skip prompts for remaining
			reconcileFixAll = true
		case "q", "quit":
			fmt.Println("  Aborted")
			return nil
		default:
			fmt.Printf("  ⏭️  Skipped (unknown input)\n")
		}
		fmt.Println()
	}

	return nil
}

// applyFix applies a fix to a zombie issue.
func applyFix(z ZombieIssue, mode, reason string) error {
	socketPath, err := beads.FindSocketPath("")
	if err != nil {
		return applyFixFallback(z, mode, reason)
	}

	client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
	if err := client.Connect(); err != nil {
		return applyFixFallback(z, mode, reason)
	}
	defer client.Close()

	switch mode {
	case "reset":
		// Update status to open
		status := "open"
		_, err := client.Update(&beads.UpdateArgs{
			ID:     z.ID,
			Status: &status,
		})
		return err
	case "close":
		if reason == "" {
			reason = "Zombie issue reconciled - no active agent found"
		}
		return client.CloseIssue(z.ID, reason)
	default:
		return fmt.Errorf("unknown fix mode: %s", mode)
	}
}

// applyFixFallback uses bd CLI as fallback.
func applyFixFallback(z ZombieIssue, mode, reason string) error {
	switch mode {
	case "reset":
		return beads.FallbackUpdate(z.ID, "open")
	case "close":
		if reason == "" {
			reason = "Zombie issue reconciled - no active agent found"
		}
		return beads.FallbackClose(z.ID, reason)
	default:
		return fmt.Errorf("unknown fix mode: %s", mode)
	}
}
