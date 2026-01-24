// Package main provides the frontier command for showing decidability state.
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/frontier"
	"github.com/dylan-conlin/orch-go/pkg/registry"
	"github.com/spf13/cobra"
)

var (
	frontierJSON bool
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
  orch frontier           # Show decidability state
  orch frontier --json    # Output as JSON for scripting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFrontier()
	},
}

func init() {
	frontierCmd.Flags().BoolVar(&frontierJSON, "json", false, "Output as JSON for scripting")
}

// FrontierOutput represents the full frontier output for JSON serialization.
type FrontierOutput struct {
	Warnings    []string        `json:"warnings,omitempty"`
	Ready       []FrontierIssue `json:"ready"`
	ReadyTotal  int             `json:"ready_total"`
	Blocked     []BlockedOutput `json:"blocked"`
	BlockedTotal int            `json:"blocked_total"`
	Active      []ActiveOutput  `json:"active"`
	ActiveTotal int             `json:"active_total"`
	Stuck       []ActiveOutput  `json:"stuck"`
	StuckTotal  int             `json:"stuck_total"`
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
	// Calculate frontier state from beads
	state, err := frontier.CalculateFrontier()
	if err != nil {
		return fmt.Errorf("failed to calculate frontier: %w", err)
	}

	// Get active agents from registry and split into active vs stuck
	activeAgents, stuckAgents := getActiveAndStuckAgents()

	if frontierJSON {
		return printFrontierJSON(state, activeAgents, stuckAgents)
	}

	printFrontierText(state, activeAgents, stuckAgents)
	return nil
}

// getActiveAndStuckAgents fetches agents from the registry and splits them into
// active (< 2h) and stuck (>= 2h) categories.
func getActiveAndStuckAgents() (active, stuck []ActiveOutput) {
	reg, err := registry.New("")
	if err != nil {
		return nil, nil
	}

	agents := reg.ListActive()
	for _, agent := range agents {
		output := agentToOutput(agent)
		if output.Duration >= stuckThreshold {
			stuck = append(stuck, output)
		} else {
			active = append(active, output)
		}
	}

	return active, stuck
}

// agentToOutput converts a registry agent to an ActiveOutput.
func agentToOutput(agent *registry.Agent) ActiveOutput {
	var duration time.Duration
	if agent.SpawnedAt != "" {
		if spawnedAt, err := time.Parse(registry.TimeFormat, agent.SpawnedAt); err == nil {
			duration = time.Since(spawnedAt)
		}
	}

	id := agent.BeadsID
	if id == "" {
		id = agent.ID
	}

	return ActiveOutput{
		BeadsID:  id,
		Title:    "", // Title not stored in registry, would require bd lookup
		Runtime:  formatDuration(duration),
		Duration: duration,
		Skill:    agent.Skill,
	}
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
			title := truncateTitle(issue.Title, 50)
			fmt.Printf("   %s  %s\n", issue.ID, title)
		}
		if len(state.Ready) > maxDisplayItems {
			fmt.Printf("   ... and %d more\n", len(state.Ready)-maxDisplayItems)
		}
	}
	fmt.Println()

	// BLOCKED - sorted by leverage, show ID + title
	fmt.Printf("BLOCKED (%d)\n", len(state.Blocked))
	if len(state.Blocked) == 0 {
		fmt.Println("   (none)")
	} else {
		displayCount := min(len(state.Blocked), maxDisplayItems)
		for i := 0; i < displayCount; i++ {
			bi := state.Blocked[i]
			title := truncateTitle(bi.Issue.Title, 40)
			leverage := frontier.FormatLeverage(bi)
			if leverage != "" {
				fmt.Printf("   %s  %s → %s\n", bi.Issue.ID, title, leverage)
			} else {
				fmt.Printf("   %s  %s\n", bi.Issue.ID, title)
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

// truncateTitle truncates a title to maxLen characters, adding "..." if truncated.
func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen-3] + "..."
}

