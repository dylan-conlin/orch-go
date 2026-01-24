// Package main provides the frontier command for showing decidability state.
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/frontier"
	"github.com/dylan-conlin/orch-go/pkg/registry"
	"github.com/spf13/cobra"
)

var (
	frontierJSON bool
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
	Ready   []FrontierIssue `json:"ready"`
	Blocked []BlockedOutput `json:"blocked"`
	Active  []ActiveOutput  `json:"active"`
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
	BeadsID string `json:"beads_id"`
	Runtime string `json:"runtime"`
	Skill   string `json:"skill,omitempty"`
}

func runFrontier() error {
	// Calculate frontier state from beads
	state, err := frontier.CalculateFrontier()
	if err != nil {
		return fmt.Errorf("failed to calculate frontier: %w", err)
	}

	// Get active agents from registry
	activeAgents := getActiveAgentsForFrontier()

	if frontierJSON {
		return printFrontierJSON(state, activeAgents)
	}

	printFrontierText(state, activeAgents)
	return nil
}

// getActiveAgentsForFrontier fetches active agents from the registry.
func getActiveAgentsForFrontier() []*registry.Agent {
	reg, err := registry.New("")
	if err != nil {
		return nil
	}
	return reg.ListActive()
}

func printFrontierJSON(state *frontier.FrontierState, activeAgents []*registry.Agent) error {
	output := FrontierOutput{
		Ready:   make([]FrontierIssue, 0, len(state.Ready)),
		Blocked: make([]BlockedOutput, 0, len(state.Blocked)),
		Active:  make([]ActiveOutput, 0, len(activeAgents)),
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

	for _, agent := range activeAgents {
		runtime := formatAgentRuntime(agent)
		output.Active = append(output.Active, ActiveOutput{
			BeadsID: agent.BeadsID,
			Runtime: runtime,
			Skill:   agent.Skill,
		})
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func printFrontierText(state *frontier.FrontierState, activeAgents []*registry.Agent) {
	// READY TO RELEASE
	fmt.Printf("READY TO RELEASE (%d)\n", len(state.Ready))
	if len(state.Ready) == 0 {
		fmt.Println("   (none)")
	} else {
		// Show as comma-separated list for compactness
		ids := make([]string, 0, len(state.Ready))
		for _, issue := range state.Ready {
			ids = append(ids, issue.ID)
		}
		// Print in rows of ~4 IDs to fit terminal width
		printCompactList(ids, 4, "   ")
	}
	fmt.Println()

	// BLOCKED - sorted by leverage
	fmt.Printf("BLOCKED - would unblock most first (%d)\n", len(state.Blocked))
	if len(state.Blocked) == 0 {
		fmt.Println("   (none)")
	} else {
		for _, bi := range state.Blocked {
			leverage := frontier.FormatLeverage(bi)
			if leverage != "" {
				fmt.Printf("   %s → %s\n", bi.Issue.ID, leverage)
			} else {
				fmt.Printf("   %s (no leverage)\n", bi.Issue.ID)
			}
		}
	}
	fmt.Println()

	// ACTIVE agents
	fmt.Printf("ACTIVE (%d)\n", len(activeAgents))
	if len(activeAgents) == 0 {
		fmt.Println("   (none)")
	} else {
		// Show agents with runtime
		for _, agent := range activeAgents {
			runtime := formatAgentRuntime(agent)
			if agent.BeadsID != "" {
				fmt.Printf("   %s [%s]\n", agent.BeadsID, runtime)
			} else {
				fmt.Printf("   %s [%s]\n", agent.ID, runtime)
			}
		}
	}
}

// printCompactList prints IDs in rows of `perRow` items.
func printCompactList(ids []string, perRow int, indent string) {
	for i := 0; i < len(ids); i += perRow {
		end := i + perRow
		if end > len(ids) {
			end = len(ids)
		}
		fmt.Printf("%s%s\n", indent, strings.Join(ids[i:end], ", "))
	}
}

// formatAgentRuntime formats the runtime of an agent.
func formatAgentRuntime(agent *registry.Agent) string {
	if agent.SpawnedAt == "" {
		return "unknown"
	}

	spawnedAt, err := time.Parse(registry.TimeFormat, agent.SpawnedAt)
	if err != nil {
		return "unknown"
	}

	duration := time.Since(spawnedAt)
	return formatDuration(duration)
}
