// Package main provides the plan command for coordination plan management.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/plan"
	"github.com/spf13/cobra"
)

// Type aliases for backward compatibility with plan_hydrate.go and other files.
type PlanFile = plan.File
type PlanPhase = plan.Phase

var (
	planShowAll bool // Show all plans (not just active)
	planJSON    bool // JSON output
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Coordination plan management",
	Long: `Manage coordination plans in .kb/plans/.

Plans persist strategic narrative alongside beads' graph structure,
capturing phasing rationale, blocking logic, and cross-project awareness.

Examples:
  orch plan show                  # Show active plans with beads status
  orch plan show my-plan          # Show specific plan by slug
  orch plan status                # Summary of all plans
  orch plan create my-slug        # Create new plan via kb create plan`,
}

var planShowCmd = &cobra.Command{
	Use:   "show [slug]",
	Short: "Display plans with beads status overlay",
	Long: `Show active coordination plans with live beads issue status.

Without arguments, shows all active plans. With a slug argument,
shows the specific plan in detail.

Examples:
  orch plan show                  # All active plans
  orch plan show --all            # All plans (including completed/superseded)
  orch plan show toolshed-pw      # Specific plan by slug match`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, _ := os.Getwd()
		plansDir := filepath.Join(projectDir, ".kb", "plans")

		plans, err := plan.ScanDir(plansDir)
		if err != nil {
			return fmt.Errorf("failed to scan plans: %w", err)
		}

		if len(plans) == 0 {
			fmt.Println("No plans found in .kb/plans/")
			fmt.Println("Create one: orch plan create <slug>")
			return nil
		}

		// If slug argument provided, find and show that specific plan
		if len(args) > 0 {
			slug := args[0]
			p := plan.FindBySlug(plans, slug)
			if p == nil {
				return fmt.Errorf("no plan matching %q found", slug)
			}

			// Query beads for issue statuses
			statusMap := queryBeadsStatuses(p)
			fmt.Print(formatPlanShow(p, statusMap))
			return nil
		}

		// Show filtered plans
		filtered := plans
		if !planShowAll {
			filtered = plan.FilterByStatus(plans, "active")
		}

		if len(filtered) == 0 {
			fmt.Println("No active plans. Use --all to see completed/superseded plans.")
			return nil
		}

		for i, p := range filtered {
			if i > 0 {
				fmt.Println()
			}
			statusMap := queryBeadsStatuses(&p)
			fmt.Print(formatPlanShow(&p, statusMap))
		}
		return nil
	},
}

var planStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Summary of all plans with progress",
	Long: `Show a summary view of all plans with their status and phase counts.

Examples:
  orch plan status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, _ := os.Getwd()
		plansDir := filepath.Join(projectDir, ".kb", "plans")

		plans, err := plan.ScanDir(plansDir)
		if err != nil {
			return fmt.Errorf("failed to scan plans: %w", err)
		}

		if len(plans) == 0 {
			fmt.Println("No plans found in .kb/plans/")
			return nil
		}

		fmt.Print(formatPlanStatus(plans))
		return nil
	},
}

var planCreateCmd = &cobra.Command{
	Use:   "create <slug>",
	Short: "Create a new coordination plan",
	Long: `Create a new plan artifact in .kb/plans/ using kb create plan.

Examples:
  orch plan create toolshed-pw-integration
  orch plan create auth-migration`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]

		// Delegate to kb create plan
		kbCmd := exec.Command("kb", "create", "plan", slug)
		kbCmd.Stdout = os.Stdout
		kbCmd.Stderr = os.Stderr
		kbCmd.Stdin = os.Stdin

		if err := kbCmd.Run(); err != nil {
			return fmt.Errorf("kb create plan failed: %w", err)
		}

		return nil
	},
}

func init() {
	planCmd.AddCommand(planShowCmd)
	planCmd.AddCommand(planStatusCmd)
	planCmd.AddCommand(planCreateCmd)

	planShowCmd.Flags().BoolVar(&planShowAll, "all", false, "Show all plans (including completed/superseded)")
}

// Package-level aliases for plan package functions.
// Used by tests and by plan_hydrate.go to avoid plan/plan naming collisions.
var (
	scanPlansDir            = plan.ScanDir
	findPlanBySlug          = plan.FindBySlug
	filterPlansByStatus     = plan.FilterByStatus
	collectAllBeadsIDs      = plan.CollectAllBeadsIDs
	parsePlanContent        = plan.ParseContent
	parseBeadsLine          = plan.ParseBeadsLine
	extractSlugFromFilename = plan.ExtractSlugFromFilename
	parseDependsOn          = plan.ParseDependsOn
)

// queryBeadsStatuses queries beads for the status of all referenced issues.
// Returns a map of beads ID -> issue status. Returns nil on error.
func queryBeadsStatuses(p *plan.File) map[string]string {
	ids := plan.CollectAllBeadsIDs(p)
	if len(ids) == 0 {
		return nil
	}

	client := beads.NewCLIClient()
	statusMap := make(map[string]string)

	for _, id := range ids {
		issue, err := client.Show(id)
		if err != nil {
			statusMap[id] = "unknown"
			continue
		}
		statusMap[id] = issue.Status
	}

	return statusMap
}

// formatPlanShow formats a single plan for display with optional beads status overlay.
func formatPlanShow(p *plan.File, statusMap map[string]string) string {
	var b strings.Builder

	// Header
	fmt.Fprintf(&b, "Plan: %s\n", p.Title)
	fmt.Fprintf(&b, "Status: %s\n", p.Status)
	fmt.Fprintf(&b, "Date: %s\n", p.Date)
	if p.Owner != "" {
		fmt.Fprintf(&b, "Owner: %s\n", p.Owner)
	}
	if len(p.Projects) > 0 {
		fmt.Fprintf(&b, "Projects: %s\n", strings.Join(p.Projects, ", "))
	}
	if p.SupersededBy != "" {
		fmt.Fprintf(&b, "Superseded-By: %s\n", p.SupersededBy)
	}
	fmt.Fprintf(&b, "File: .kb/plans/%s\n", p.Filename)

	if len(p.Phases) == 0 {
		return b.String()
	}

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Phases:")

	for i, phase := range p.Phases {
		phaseNum := i + 1

		// Compute phase status from beads issues
		phaseStatus := computePhaseStatus(phase.BeadsIDs, statusMap)
		statusIcon := phaseStatusIcon(phaseStatus)

		fmt.Fprintf(&b, "  %s Phase %d: %s", statusIcon, phaseNum, phase.Name)
		if phase.DependsOn != "" && phase.DependsOn != "none" {
			fmt.Fprintf(&b, " (depends: %s)", phase.DependsOn)
		}
		fmt.Fprintln(&b)

		// Show individual beads issues with status
		if len(phase.BeadsIDs) > 0 && statusMap != nil {
			for _, id := range phase.BeadsIDs {
				status := statusMap[id]
				icon := issueStatusIcon(status)
				fmt.Fprintf(&b, "    %s %s (%s)\n", icon, id, status)
			}
		} else if len(phase.BeadsIDs) > 0 {
			fmt.Fprintf(&b, "    Beads: %s\n", strings.Join(phase.BeadsIDs, ", "))
		}
	}

	return b.String()
}

// computePhaseStatus determines the overall phase status from its beads issues.
func computePhaseStatus(beadsIDs []string, statusMap map[string]string) string {
	if len(beadsIDs) == 0 || statusMap == nil {
		return "no-issues"
	}

	allClosed := true
	anyInProgress := false
	anyOpen := false

	for _, id := range beadsIDs {
		status := statusMap[id]
		switch status {
		case "closed":
			// ok
		case "in_progress":
			allClosed = false
			anyInProgress = true
		default:
			allClosed = false
			anyOpen = true
		}
	}

	if allClosed {
		return "complete"
	}
	if anyInProgress {
		return "in-progress"
	}
	if anyOpen {
		return "ready"
	}
	return "unknown"
}

// phaseStatusIcon returns an icon for the phase status.
func phaseStatusIcon(status string) string {
	switch status {
	case "complete":
		return "[x]"
	case "in-progress":
		return "[~]"
	case "ready":
		return "[ ]"
	default:
		return "[ ]"
	}
}

// issueStatusIcon returns an icon for a beads issue status.
func issueStatusIcon(status string) string {
	switch status {
	case "closed":
		return "[x]"
	case "in_progress":
		return "[~]"
	case "open":
		return "[ ]"
	default:
		return "[?]"
	}
}

// formatPlanStatus formats a summary view of all plans.
func formatPlanStatus(plans []plan.File) string {
	var b strings.Builder

	// Count by status
	counts := map[string]int{}
	for _, p := range plans {
		counts[p.Status]++
	}

	fmt.Fprintln(&b, "Plans Summary")
	fmt.Fprintln(&b, strings.Repeat("-", 40))

	if n := counts["active"]; n > 0 {
		fmt.Fprintf(&b, "Active:     %d\n", n)
	}
	if n := counts["completed"]; n > 0 {
		fmt.Fprintf(&b, "Completed:  %d\n", n)
	}
	if n := counts["superseded"]; n > 0 {
		fmt.Fprintf(&b, "Superseded: %d\n", n)
	}
	if n := counts["draft"]; n > 0 {
		fmt.Fprintf(&b, "Draft:      %d\n", n)
	}
	fmt.Fprintln(&b)

	for _, p := range plans {
		statusIcon := planStatusIcon(p.Status)
		phaseCount := len(p.Phases)
		fmt.Fprintf(&b, "  %s %s (%s, %d phases)\n", statusIcon, p.Title, p.Status, phaseCount)
		fmt.Fprintf(&b, "    File: .kb/plans/%s\n", p.Filename)
	}

	return b.String()
}

// planStatusIcon returns an icon for a plan status.
func planStatusIcon(status string) string {
	switch status {
	case "active":
		return "[~]"
	case "completed":
		return "[x]"
	case "superseded":
		return "[-]"
	case "draft":
		return "[ ]"
	default:
		return "[?]"
	}
}
