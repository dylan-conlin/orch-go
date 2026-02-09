// Package main provides the CLI entry point for orch-go.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

var (
	reconcileJSON     bool
	reconcileFix      bool
	reconcileFixMode  string
	reconcileFixAll   bool
	reconcileProject  string
	reconcileMinAge   int
	reconcileMaxShown int
)

type ZombieIssue struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Project          string    `json:"project"`
	Status           string    `json:"status"`
	Priority         int       `json:"priority"`
	Labels           []string  `json:"labels"`
	UpdatedAt        time.Time `json:"updated_at"`
	AgeSinceUpdate   string    `json:"age_since_update"`
	HoursSinceUpdate float64   `json:"hours_since_update"`
	HasWorkspace     bool      `json:"has_workspace"`
	WorkspacePath    string    `json:"workspace_path,omitempty"`
	LastPhase        string    `json:"last_phase,omitempty"`
}

var reconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Detect and fix zombie in_progress issues",
	Long: `Detect in_progress issues that have no active agent and offer to fix them.

Examples:
  orch reconcile                      # Show zombie issues
  orch reconcile --json               # Output as JSON
  orch reconcile --fix                # Interactive fix mode`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReconcile()
	},
}

func init() {
	reconcileCmd.Flags().BoolVar(&reconcileJSON, "json", false, "Output as JSON")
	reconcileCmd.Flags().BoolVar(&reconcileFix, "fix", false, "Apply fixes")
	reconcileCmd.Flags().StringVar(&reconcileFixMode, "mode", "reset", "Fix mode: reset or close")
	reconcileCmd.Flags().BoolVar(&reconcileFixAll, "all", false, "Fix all without prompting")
	reconcileCmd.Flags().StringVarP(&reconcileProject, "project", "p", "", "Filter by project")
	reconcileCmd.Flags().IntVar(&reconcileMinAge, "min-age", 1, "Minimum hours since update")
	reconcileCmd.Flags().IntVar(&reconcileMaxShown, "limit", 50, "Maximum zombies to show")
	rootCmd.AddCommand(reconcileCmd)
}

func runReconcile() error {
	projectDir, _ := currentProjectDir()
	zombies, err := findZombieIssues(projectDir)
	if err != nil {
		return fmt.Errorf("failed to find zombie issues: %w", err)
	}

	if reconcileProject != "" {
		filtered := make([]ZombieIssue, 0)
		for _, z := range zombies {
			if z.Project == reconcileProject {
				filtered = append(filtered, z)
			}
		}
		zombies = filtered
	}

	if reconcileMinAge > 0 {
		filtered := make([]ZombieIssue, 0)
		for _, z := range zombies {
			if z.HoursSinceUpdate >= float64(reconcileMinAge) {
				filtered = append(filtered, z)
			}
		}
		zombies = filtered
	}

	sort.Slice(zombies, func(i, j int) bool {
		return zombies[i].HoursSinceUpdate > zombies[j].HoursSinceUpdate
	})

	if reconcileMaxShown > 0 && len(zombies) > reconcileMaxShown {
		zombies = zombies[:reconcileMaxShown]
	}

	if reconcileJSON {
		data, _ := json.MarshalIndent(zombies, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(zombies) == 0 {
		fmt.Println("No zombie issues found")
		return nil
	}

	fmt.Printf("Found %d zombie issue(s):\n\n", len(zombies))
	for i, z := range zombies {
		fmt.Printf("  %d. [P%d] %s: %s\n", i+1, z.Priority, z.ID, z.Title)
		fmt.Printf("     Age: %s\n\n", z.AgeSinceUpdate)
	}

	if !reconcileFix {
		fmt.Println("Run with --fix to repair")
		return nil
	}
	return runReconcileFix(zombies)
}

func findZombieIssues(projectDir string) ([]ZombieIssue, error) {
	inProgress := "in_progress"
	var issues []beads.Issue
	err := withBeadsClient("", func(client *beads.Client) error {
		var rpcErr error
		issues, rpcErr = client.List(&beads.ListArgs{Status: inProgress})
		return rpcErr
	}, beads.WithAutoReconnect(3))
	if err != nil {
		return findZombieIssuesFallback(projectDir)
	}
	return filterZombies(projectDir, issues)
}

func findZombieIssuesFallback(projectDir string) ([]ZombieIssue, error) {
	issues, err := beads.FallbackList("in_progress")
	if err != nil {
		return nil, err
	}
	return filterZombies(projectDir, issues)
}

func filterZombies(projectDir string, issues []beads.Issue) ([]ZombieIssue, error) {
	return filterZombiesWithClient(opencode.NewClient(serverURL), projectDir, issues)
}

func filterZombiesWithClient(ocClient opencode.ClientInterface, projectDir string, issues []beads.Issue) ([]ZombieIssue, error) {
	now := time.Now()
	var zombies []ZombieIssue

	var activeSessions []opencode.Session
	seenSessionIDs := make(map[string]bool)

	if projectDir != "" {
		dirSessions, _ := ocClient.ListSessions(projectDir)
		for _, s := range dirSessions {
			if !seenSessionIDs[s.ID] {
				seenSessionIDs[s.ID] = true
				activeSessions = append(activeSessions, s)
			}
		}
	}
	globalSessions, _ := ocClient.ListSessions("")
	for _, s := range globalSessions {
		if !seenSessionIDs[s.ID] {
			seenSessionIDs[s.ID] = true
			activeSessions = append(activeSessions, s)
		}
	}

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

	for _, issue := range issues {
		if activeBeadsIDs[issue.ID] {
			continue
		}
		var updatedAt time.Time
		if issue.UpdatedAt != "" {
			t, _ := time.Parse(time.RFC3339, issue.UpdatedAt)
			updatedAt = t
		}
		if updatedAt.IsZero() {
			updatedAt = now
		}
		hoursSinceUpdate := now.Sub(updatedAt).Hours()
		liveness := state.GetLiveness(issue.ID, serverURL, projectDir)
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

func getLastPhase(beadsID string) string {
	var comments []beads.Comment
	err := withBeadsClient("", func(client *beads.Client) error {
		var rpcErr error
		comments, rpcErr = client.Comments(beadsID)
		return rpcErr
	}, beads.WithAutoReconnect(1))
	if err != nil {
		return ""
	}

	for i := len(comments) - 1; i >= 0; i-- {
		if strings.HasPrefix(comments[i].Text, "Phase:") {
			return strings.TrimPrefix(comments[i].Text, "Phase: ")
		}
	}
	return ""
}

func runReconcileFix(zombies []ZombieIssue) error {
	reader := bufio.NewReader(os.Stdin)
	logger := events.NewLogger(events.DefaultLogPath())
	var successCount, failCount int
	for _, z := range zombies {
		var mode string
		if reconcileFixAll {
			mode = reconcileFixMode
		} else {
			fmt.Printf("Fix %s? [r]eset [c]lose [s]kip: ", z.ID)
			input, _ := reader.ReadString('\n')
			switch strings.TrimSpace(strings.ToLower(input)) {
			case "r", "reset", "":
				mode = "reset"
			case "c", "close":
				mode = "close"
			default:
				// Skip
				continue
			}
		}

		if err := applyFix(z, mode, ""); err != nil {
			fmt.Printf("  ✗ Failed to %s %s: %v\n", mode, z.ID, err)
			failCount++
		} else {
			action := "Reset"
			if mode == "close" {
				action = "Closed"
				// Emit agent.completed event for closed zombies so stats capture these completions
				event := events.Event{
					Type:      "agent.completed",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"beads_id":           z.ID,
						"reason":             "zombie_reconciled",
						"source":             "reconcile",
						"project":            z.Project,
						"last_phase":         z.LastPhase,
						"hours_since_update": z.HoursSinceUpdate,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: failed to log event: %v\n", err)
				}
			}
			fmt.Printf("  ✓ %s %s\n", action, z.ID)
			successCount++
		}
	}

	if successCount > 0 || failCount > 0 {
		fmt.Printf("\nReconcile complete: %d succeeded, %d failed\n", successCount, failCount)
	}
	return nil
}

func applyFix(z ZombieIssue, mode, reason string) error {
	// For zombie reconciliation, always use --force because zombies
	// are inherently incomplete (no agent working on them, likely no
	// "Phase: Complete" comment). Using CLI fallback with --force is
	// the most reliable approach.
	return applyFixFallback(z, mode, reason)
}

func applyFixFallback(z ZombieIssue, mode, reason string) error {
	switch mode {
	case "reset":
		return beads.FallbackUpdate(z.ID, "open")
	case "close":
		// Use --force for zombie reconciliation since zombies typically
		// don't have "Phase: Complete" comments (they were abandoned)
		return forceCloseIssue(z.ID, "Zombie reconciled")
	}
	return nil
}

// forceCloseIssue closes an issue with --force flag to bypass "Phase: Complete" check.
// This is specifically for zombie reconciliation where issues were abandoned.
func forceCloseIssue(id, reason string) error {
	args := []string{"close", id, "--force"}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	// Use "bd" command - ResolveBdPath should be called at startup
	args = append([]string{"--quiet"}, args...)
	cmd := exec.Command("bd", args...)
	cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd close failed: %w: %s", err, string(output))
	}
	// Also check output for error patterns since bd close may return 0 on soft errors
	outputStr := string(output)
	if strings.Contains(outputStr, "Error:") || strings.Contains(outputStr, "error:") {
		return fmt.Errorf("bd close failed: %s", strings.TrimSpace(outputStr))
	}
	return nil
}
