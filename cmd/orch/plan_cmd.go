// Package main provides the plan command for coordination plan management.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/spf13/cobra"
)

// PlanFile represents a parsed .kb/plans/ artifact.
type PlanFile struct {
	Title        string
	Date         string
	Status       string // active, completed, superseded
	Owner        string
	Filename     string
	Projects     []string
	SupersededBy string
	Phases       []PlanPhase
}

// PlanPhase represents a phase within a plan.
type PlanPhase struct {
	Name      string
	Goal      string
	DependsOn string
	BeadsIDs  []string
}

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

		plans, err := scanPlansDir(plansDir)
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
			plan := findPlanBySlug(plans, slug)
			if plan == nil {
				return fmt.Errorf("no plan matching %q found", slug)
			}

			// Query beads for issue statuses
			statusMap := queryBeadsStatuses(plan)
			fmt.Print(formatPlanShow(plan, statusMap))
			return nil
		}

		// Show filtered plans
		filtered := plans
		if !planShowAll {
			filtered = filterPlansByStatus(plans, "active")
		}

		if len(filtered) == 0 {
			fmt.Println("No active plans. Use --all to see completed/superseded plans.")
			return nil
		}

		for i, plan := range filtered {
			if i > 0 {
				fmt.Println()
			}
			statusMap := queryBeadsStatuses(&plan)
			fmt.Print(formatPlanShow(&plan, statusMap))
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

		plans, err := scanPlansDir(plansDir)
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

// scanPlansDir reads all .md files from the plans directory and parses them.
func scanPlansDir(dir string) ([]PlanFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var plans []PlanFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		plan := parsePlanContent(string(content), entry.Name())
		plans = append(plans, plan)
	}

	return plans, nil
}

// parsePlanContent extracts metadata and phases from a plan markdown file.
func parsePlanContent(content, filename string) PlanFile {
	plan := PlanFile{
		Filename: filename,
	}

	lines := strings.Split(content, "\n")

	var currentPhase *PlanPhase
	inPhases := false
	statusFound := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Title: "# Plan: Title Here" or "# Coordination Plan: Title Here"
		if strings.HasPrefix(trimmed, "# Plan: ") {
			plan.Title = strings.TrimPrefix(trimmed, "# Plan: ")
			continue
		}
		if strings.HasPrefix(trimmed, "# Coordination Plan: ") {
			plan.Title = strings.TrimPrefix(trimmed, "# Coordination Plan: ")
			continue
		}

		// Metadata fields (only parse plan-level Status before Phases section)
		if strings.HasPrefix(trimmed, "**Date:**") && !inPhases {
			plan.Date = extractMetaValue(trimmed, "**Date:**")
			continue
		}
		if strings.HasPrefix(trimmed, "**Status:**") && !inPhases && !statusFound {
			plan.Status = extractMetaValue(trimmed, "**Status:**")
			statusFound = true
			continue
		}
		if strings.HasPrefix(trimmed, "**Owner:**") {
			plan.Owner = extractMetaValue(trimmed, "**Owner:**")
			continue
		}
		if strings.HasPrefix(trimmed, "**Projects:**") {
			val := extractMetaValue(trimmed, "**Projects:**")
			if val != "" {
				for _, p := range strings.Split(val, ",") {
					p = strings.TrimSpace(p)
					if p != "" {
						plan.Projects = append(plan.Projects, p)
					}
				}
			}
			continue
		}
		if strings.HasPrefix(trimmed, "**Superseded-By:**") {
			plan.SupersededBy = extractMetaValue(trimmed, "**Superseded-By:**")
			continue
		}

		// Phases section
		if trimmed == "## Phases" {
			inPhases = true
			continue
		}
		// End of phases section (next ## heading)
		if inPhases && strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			if currentPhase != nil {
				plan.Phases = append(plan.Phases, *currentPhase)
				currentPhase = nil
			}
			inPhases = false
			continue
		}

		if !inPhases {
			continue
		}

		// Phase heading: "### Phase N: Name"
		if strings.HasPrefix(trimmed, "### Phase ") {
			if currentPhase != nil {
				plan.Phases = append(plan.Phases, *currentPhase)
			}
			// Extract name after "### Phase N: "
			name := trimmed
			if idx := strings.Index(trimmed, ": "); idx >= 0 {
				name = trimmed[idx+2:]
			}
			currentPhase = &PlanPhase{Name: name}
			continue
		}

		if currentPhase == nil {
			continue
		}

		// Phase metadata
		if strings.HasPrefix(trimmed, "**Goal:**") {
			currentPhase.Goal = extractMetaValue(trimmed, "**Goal:**")
		}
		if strings.HasPrefix(trimmed, "**Depends on:**") {
			currentPhase.DependsOn = extractMetaValue(trimmed, "**Depends on:**")
		}
		if strings.HasPrefix(trimmed, "**Beads:**") {
			currentPhase.BeadsIDs = parseBeadsLine(trimmed)
		}
	}

	// Don't forget the last phase
	if currentPhase != nil {
		plan.Phases = append(plan.Phases, *currentPhase)
	}

	return plan
}

// extractMetaValue extracts the value from a "**Key:** value" line.
func extractMetaValue(line, prefix string) string {
	val := strings.TrimPrefix(line, prefix)
	return strings.TrimSpace(val)
}

// parseBeadsLine extracts beads IDs from a "**Beads:** id1, id2" line.
func parseBeadsLine(line string) []string {
	val := extractMetaValue(line, "**Beads:**")
	if val == "" || val == "none" {
		return nil
	}

	var ids []string
	for _, id := range strings.Split(val, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// filterPlansByStatus returns plans matching the given status.
func filterPlansByStatus(plans []PlanFile, status string) []PlanFile {
	var result []PlanFile
	for _, p := range plans {
		if p.Status == status {
			result = append(result, p)
		}
	}
	return result
}

// findPlanBySlug finds a plan whose filename contains the slug.
func findPlanBySlug(plans []PlanFile, slug string) *PlanFile {
	for i, p := range plans {
		if strings.Contains(p.Filename, slug) {
			return &plans[i]
		}
	}
	return nil
}

// collectAllBeadsIDs gathers all beads IDs from all phases of a plan.
func collectAllBeadsIDs(plan *PlanFile) []string {
	var ids []string
	for _, phase := range plan.Phases {
		ids = append(ids, phase.BeadsIDs...)
	}
	return ids
}

// queryBeadsStatuses queries beads for the status of all referenced issues.
// Returns a map of beads ID -> issue status. Returns nil on error.
func queryBeadsStatuses(plan *PlanFile) map[string]string {
	ids := collectAllBeadsIDs(plan)
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
func formatPlanShow(plan *PlanFile, statusMap map[string]string) string {
	var b strings.Builder

	// Header
	fmt.Fprintf(&b, "Plan: %s\n", plan.Title)
	fmt.Fprintf(&b, "Status: %s\n", plan.Status)
	fmt.Fprintf(&b, "Date: %s\n", plan.Date)
	if plan.Owner != "" {
		fmt.Fprintf(&b, "Owner: %s\n", plan.Owner)
	}
	if len(plan.Projects) > 0 {
		fmt.Fprintf(&b, "Projects: %s\n", strings.Join(plan.Projects, ", "))
	}
	if plan.SupersededBy != "" {
		fmt.Fprintf(&b, "Superseded-By: %s\n", plan.SupersededBy)
	}
	fmt.Fprintf(&b, "File: .kb/plans/%s\n", plan.Filename)

	if len(plan.Phases) == 0 {
		return b.String()
	}

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Phases:")

	for i, phase := range plan.Phases {
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
func formatPlanStatus(plans []PlanFile) string {
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

	for _, plan := range plans {
		statusIcon := planStatusIcon(plan.Status)
		phaseCount := len(plan.Phases)
		fmt.Fprintf(&b, "  %s %s (%s, %d phases)\n", statusIcon, plan.Title, plan.Status, phaseCount)
		fmt.Fprintf(&b, "    File: .kb/plans/%s\n", plan.Filename)
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
