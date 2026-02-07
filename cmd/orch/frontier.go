// Package main provides the frontier command for showing decidability state.
package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/frontier"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

var (
	frontierJSON    bool
	frontierWorkdir string
)

const (
	// maxDisplayItems is the maximum number of items to show per section
	maxDisplayItems = 8
	// stuckThreshold is the duration after which an agent is considered stuck
	stuckThreshold = 2 * time.Hour
)

var frontierCmd = &cobra.Command{
	Use:   "frontier",
	Short: "Show decidability state - what's ready, blocked, and active",
	Long: `Show the current decidability frontier in readable format.

Output is grouped by "who needs to act":
- READY TO RELEASE: Issues ready to work on (no blockers)
- BLOCKED: Issues that are blocked, sorted by leverage (what would unblock the most)
- ACTIVE: Agents currently working on issues

Examples:
  orch frontier                           # Show decidability state
  orch frontier --json                    # Output as JSON for scripting
  orch frontier --workdir ~/projects/foo  # Show frontier for another project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFrontier()
	},
}

func init() {
	frontierCmd.Flags().BoolVar(&frontierJSON, "json", false, "Output as JSON for scripting")
	frontierCmd.Flags().StringVar(&frontierWorkdir, "workdir", "", "Target project directory for cross-project frontier view")
	frontierCmd.Flags().StringVar(&frontierWorkdir, "project", "", "Alias for --workdir")
	frontierCmd.Flags().MarkHidden("project")
}

// FrontierOutput represents the full frontier output for JSON serialization.
type FrontierOutput struct {
	Warnings     []string        `json:"warnings,omitempty"`
	Ready        []FrontierIssue `json:"ready"`
	ReadyTotal   int             `json:"ready_total"`
	Blocked      []BlockedOutput `json:"blocked"`
	BlockedTotal int             `json:"blocked_total"`
	Active       []ActiveOutput  `json:"active"`
	ActiveTotal  int             `json:"active_total"`
	Stuck        []ActiveOutput  `json:"stuck"`
	StuckTotal   int             `json:"stuck_total"`
}

// FrontierIssue represents an issue in the frontier output.
type FrontierIssue struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	IssueType string `json:"issue_type"`
	Priority  int    `json:"priority"`
}

// BlockedOutput represents a blocked issue with leverage info.
type BlockedOutput struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	IssueType     string   `json:"issue_type"`
	Priority      int      `json:"priority"`
	BlockedBy     []string `json:"blocked_by,omitempty"`
	WouldUnblock  []string `json:"would_unblock,omitempty"`
	TotalLeverage int      `json:"total_leverage"`
}

// ActiveOutput represents an active agent.
type ActiveOutput struct {
	BeadsID  string        `json:"beads_id"`
	Title    string        `json:"title,omitempty"`
	Runtime  string        `json:"runtime"`
	Duration time.Duration `json:"-"` // For sorting, not serialized
	Skill    string        `json:"skill,omitempty"`
}

func runFrontier() error {
	// Resolve project directory for beads operations
	currentDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	projectResult, err := resolveProjectDir(frontierWorkdir, "", currentDir)
	if err != nil {
		return err
	}

	// Set beads.DefaultDir for cross-project operations
	projectResult.SetBeadsDefaultDir()

	// Log if using explicit workdir
	if projectResult.Source == "workdir" {
		fmt.Printf("Project: %s\n\n", projectResult.ProjectDir)
	}

	// Calculate frontier state from beads
	state, err := frontier.CalculateFrontier()
	if err != nil {
		return fmt.Errorf("failed to calculate frontier: %w", err)
	}

	// Get active agents from live sources and split into active vs stuck
	activeAgents, stuckAgents := getActiveAndStuckAgents()

	// Filter out agents whose beads issues are closed
	// This prevents stale OpenCode sessions from showing as "stuck"
	activeAgents = filterOpenIssueAgents(activeAgents)
	stuckAgents = filterOpenIssueAgents(stuckAgents)

	if frontierJSON {
		return printFrontierJSON(state, activeAgents, stuckAgents)
	}

	printFrontierText(state, activeAgents, stuckAgents)
	return nil
}

// getActiveAndStuckAgents discovers agents from tmux windows and OpenCode sessions,
// then splits them into active (< 2h) and stuck (>= 2h) categories.
// This uses authoritative sources (live runtime state) for liveness detection.
func getActiveAndStuckAgents() ([]ActiveOutput, []ActiveOutput) {
	return getActiveAndStuckAgentsWithClient(opencode.NewClient(serverURL))
}

func getActiveAndStuckAgentsWithClient(client opencode.ClientInterface) (active, stuck []ActiveOutput) {
	// Initialize as empty slices (not nil) to ensure JSON encodes as [] not null
	active = []ActiveOutput{}
	stuck = []ActiveOutput{}
	now := time.Now()
	seenBeadsIDs := make(map[string]bool)

	// Phase 1: Discover agents from tmux windows (claude mode agents)
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		windows, _ := tmux.ListWindows(sessionName)
		for _, w := range windows {
			// Skip known non-agent windows
			if w.Name == "servers" || w.Name == "zsh" {
				continue
			}

			beadsID := extractBeadsIDFromWindowName(w.Name)
			if beadsID == "" {
				continue
			}

			// Skip if already seen (duplicate window)
			if seenBeadsIDs[beadsID] {
				continue
			}

			// For tmux agents, we don't have spawn time easily accessible
			// Default to 0 duration (won't be marked as stuck)
			// This is acceptable since tmux agents are visible and monitored
			output := ActiveOutput{
				BeadsID: beadsID,
				Runtime: "tmux",
				Skill:   extractSkillFromWindowName(w.Name),
			}

			active = append(active, output)
			seenBeadsIDs[beadsID] = true
		}
	}

	// Phase 2: Discover agents from OpenCode sessions
	// Use 3h window to catch stuck agents (beyond 2h threshold)
	sessions, err := client.ListSessions("")
	if err != nil {
		return active, stuck
	}

	const maxAge = 3 * time.Hour
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) > maxAge {
			continue
		}

		beadsID := extractBeadsIDFromTitle(s.Title)
		if beadsID == "" {
			continue
		}

		// Skip if already tracked via tmux
		if seenBeadsIDs[beadsID] {
			continue
		}

		createdAt := time.Unix(s.Time.Created/1000, 0)
		duration := now.Sub(createdAt)

		output := ActiveOutput{
			BeadsID:  beadsID,
			Runtime:  formatDuration(duration),
			Duration: duration,
			Skill:    extractSkillFromTitle(s.Title),
		}

		if duration >= stuckThreshold {
			stuck = append(stuck, output)
		} else {
			active = append(active, output)
		}
		seenBeadsIDs[beadsID] = true
	}

	return active, stuck
}

func printFrontierJSON(state *frontier.FrontierState, active, stuck []ActiveOutput) error {
	output := FrontierOutput{
		Ready:        make([]FrontierIssue, 0, len(state.Ready)),
		ReadyTotal:   len(state.Ready),
		Blocked:      make([]BlockedOutput, 0, len(state.Blocked)),
		BlockedTotal: len(state.Blocked),
		Active:       active,
		ActiveTotal:  len(active),
		Stuck:        stuck,
		StuckTotal:   len(stuck),
	}

	// Add health warnings
	if len(stuck) > 0 {
		output.Warnings = append(output.Warnings, fmt.Sprintf("%d stuck agents (> 2h) - run 'orch clean --stale' to clean up", len(stuck)))
	}

	for _, issue := range state.Ready {
		output.Ready = append(output.Ready, FrontierIssue{
			ID:        issue.ID,
			Title:     issue.Title,
			IssueType: issue.IssueType,
			Priority:  issue.Priority,
		})
	}

	for _, bi := range state.Blocked {
		output.Blocked = append(output.Blocked, BlockedOutput{
			ID:            bi.Issue.ID,
			Title:         bi.Issue.Title,
			IssueType:     bi.Issue.IssueType,
			Priority:      bi.Issue.Priority,
			BlockedBy:     bi.Issue.BlockedBy,
			WouldUnblock:  bi.WouldUnblock,
			TotalLeverage: bi.TotalLeverage,
		})
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func printFrontierText(state *frontier.FrontierState, active, stuck []ActiveOutput) {
	// Health warnings at top
	if len(stuck) > 0 {
		fmt.Println("⚠️  HEALTH WARNINGS")
		fmt.Printf("   %d stuck agents (> 2h) - run 'orch clean --stale' to clean up\n", len(stuck))
		fmt.Println()
	}

	// READY TO RELEASE - show ID + title
	fmt.Printf("READY TO RELEASE (%d)\n", len(state.Ready))
	if len(state.Ready) == 0 {
		fmt.Println("   (none)")
	} else {
		displayCount := min(len(state.Ready), maxDisplayItems)
		for i := 0; i < displayCount; i++ {
			issue := state.Ready[i]
			title := truncateTitle(issue.Title, 45)
			fmt.Printf("   [%s] %s  %s\n", issue.IssueType, issue.ID, title)
		}
		if len(state.Ready) > maxDisplayItems {
			fmt.Printf("   ... and %d more\n", len(state.Ready)-maxDisplayItems)
		}
	}
	fmt.Println()

	// BLOCKED - sorted by leverage, show ID + title + blockers
	fmt.Printf("BLOCKED (%d)\n", len(state.Blocked))
	if len(state.Blocked) == 0 {
		fmt.Println("   (none)")
	} else {
		displayCount := min(len(state.Blocked), maxDisplayItems)
		for i := 0; i < displayCount; i++ {
			bi := state.Blocked[i]
			title := truncateTitle(bi.Issue.Title, 35)
			blockers := formatBlockers(bi.Issue.BlockedBy)
			fmt.Printf("   %s  %s\n", bi.Issue.ID, title)
			if blockers != "" {
				fmt.Printf("      ← %s\n", blockers)
			}
		}
		if len(state.Blocked) > maxDisplayItems {
			fmt.Printf("   ... and %d more\n", len(state.Blocked)-maxDisplayItems)
		}
	}
	fmt.Println()

	// ACTIVE agents (< 2h)
	fmt.Printf("ACTIVE (%d)\n", len(active))
	if len(active) == 0 {
		fmt.Println("   (none)")
	} else {
		displayCount := min(len(active), maxDisplayItems)
		for i := 0; i < displayCount; i++ {
			agent := active[i]
			skillInfo := ""
			if agent.Skill != "" {
				skillInfo = fmt.Sprintf(" (%s)", agent.Skill)
			}
			fmt.Printf("   %s [%s]%s\n", agent.BeadsID, agent.Runtime, skillInfo)
		}
		if len(active) > maxDisplayItems {
			fmt.Printf("   ... and %d more\n", len(active)-maxDisplayItems)
		}
	}
	fmt.Println()

	// STUCK agents (>= 2h)
	if len(stuck) > 0 {
		fmt.Printf("STUCK (> 2h) (%d)\n", len(stuck))
		displayCount := min(len(stuck), maxDisplayItems)
		for i := 0; i < displayCount; i++ {
			agent := stuck[i]
			fmt.Printf("   %s [%s]\n", agent.BeadsID, agent.Runtime)
		}
		if len(stuck) > maxDisplayItems {
			fmt.Printf("   ... and %d more\n", len(stuck)-maxDisplayItems)
		}
	}
}

// filterOpenIssueAgents filters out agents whose beads issues are closed.
// This prevents stale OpenCode sessions from appearing in the frontier output.
func filterOpenIssueAgents(agents []ActiveOutput) []ActiveOutput {
	if len(agents) == 0 {
		return agents
	}

	// Collect all beads IDs
	beadsIDs := make([]string, 0, len(agents))
	for _, agent := range agents {
		if agent.BeadsID != "" {
			beadsIDs = append(beadsIDs, agent.BeadsID)
		}
	}

	if len(beadsIDs) == 0 {
		return agents
	}

	// Get status for all beads IDs in one call
	closedIssues := getClosedIssueIDs(beadsIDs)

	// Filter out agents with closed issues
	filtered := make([]ActiveOutput, 0, len(agents))
	for _, agent := range agents {
		if !closedIssues[agent.BeadsID] {
			filtered = append(filtered, agent)
		}
	}

	return filtered
}

// getClosedIssueIDs checks which of the given beads IDs are for closed issues.
// Returns a map of beads ID -> true for closed issues.
func getClosedIssueIDs(beadsIDs []string) map[string]bool {
	closed := make(map[string]bool)

	if len(beadsIDs) == 0 {
		return closed
	}

	// Build command: bd show <id1> <id2> ... --json
	args := append([]string{"--sandbox", "show"}, beadsIDs...)
	args = append(args, "--json")

	cmd := exec.Command("bd", args...)
	output, err := cmd.Output()
	if err != nil {
		// On error, assume all are open (fail open for visibility)
		return closed
	}

	// Parse response - bd show returns an array of issues
	var issues []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(output, &issues); err != nil {
		return closed
	}

	// Mark closed issues
	for _, issue := range issues {
		if issue.Status == "closed" {
			closed[issue.ID] = true
		}
	}

	return closed
}

// truncateTitle truncates a title to maxLen characters, adding "..." if truncated.
func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen-3] + "..."
}

// formatBlockers returns a human-readable description of what's blocking an issue.
func formatBlockers(blockedBy []string) string {
	if len(blockedBy) == 0 {
		return ""
	}

	if len(blockedBy) == 1 {
		return fmt.Sprintf("blocked by: %s", blockedBy[0])
	}

	if len(blockedBy) <= 3 {
		return fmt.Sprintf("blocked by: %s", strings.Join(blockedBy, ", "))
	}

	return fmt.Sprintf("blocked by: %s (+%d more)",
		strings.Join(blockedBy[:2], ", "),
		len(blockedBy)-2)
}
