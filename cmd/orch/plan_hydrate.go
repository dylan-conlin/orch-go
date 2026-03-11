// Package main provides the plan hydrate command for creating beads issues from plan phases.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/spf13/cobra"
)

var (
	hydrateDryRun   bool
	hydrateParentID string // optional parent beads ID for context
)

var planHydrateCmd = &cobra.Command{
	Use:   "hydrate <slug>",
	Short: "Create beads issues from plan phases with dependency wiring",
	Long: `Hydrate a coordination plan by creating beads issues for each phase.

For each unhydrated phase:
1. Creates a beads issue with title, goal, and plan context
2. Adds inter-phase dependencies (Phase 2 blocks on Phase 1, etc.)
3. Writes the beads IDs back into the plan file

Idempotent: phases that already have **Beads:** populated are skipped.

Labels: triage:ready (for daemon pickup) + plan:<slug> (for plan-level queries)

Examples:
  orch plan hydrate gate-signal-vs-noise
  orch plan hydrate gate-signal-vs-noise --dry-run
  orch plan hydrate gate-signal-vs-noise --parent orch-go-865v3`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		projectDir, _ := os.Getwd()
		plansDir := filepath.Join(projectDir, ".kb", "plans")

		plans, err := scanPlansDir(plansDir)
		if err != nil {
			return fmt.Errorf("failed to scan plans: %w", err)
		}

		plan := findPlanBySlug(plans, slug)
		if plan == nil {
			return fmt.Errorf("no plan matching %q found in .kb/plans/", slug)
		}

		planPath := filepath.Join(plansDir, plan.Filename)
		return hydratePlan(plan, planPath)
	},
}

func init() {
	planCmd.AddCommand(planHydrateCmd)
	planHydrateCmd.Flags().BoolVar(&hydrateDryRun, "dry-run", false, "Show what would be created without creating")
	planHydrateCmd.Flags().StringVar(&hydrateParentID, "parent", "", "Parent beads ID for context in descriptions")
}

// hydratePlan creates beads issues for unhydrated phases and writes IDs back.
func hydratePlan(plan *PlanFile, planPath string) error {
	toHydrate := phasesNeedingHydration(*plan)
	if len(toHydrate) == 0 {
		fmt.Println("All phases already hydrated. Nothing to do.")
		return nil
	}

	planSlug := extractSlugFromFilename(plan.Filename)

	fmt.Printf("Plan: %s\n", plan.Title)
	fmt.Printf("Phases to hydrate: %d of %d\n\n", len(toHydrate), len(plan.Phases))

	if hydrateDryRun {
		return hydrateDryRunOutput(plan, toHydrate, planSlug)
	}

	// Create issues for each unhydrated phase
	phaseIDs := make(map[int]string) // phase index -> beads ID
	// Also collect existing IDs for dependency wiring
	allPhaseIDs := make(map[int]string)
	for i, phase := range plan.Phases {
		if len(phase.BeadsIDs) > 0 {
			allPhaseIDs[i] = phase.BeadsIDs[0] // use first ID for dep wiring
		}
	}

	labels := []string{"triage:ready", "plan:" + planSlug}

	for _, idx := range toHydrate {
		phase := plan.Phases[idx]
		phaseNum := idx + 1

		title := buildPhaseTitle(plan.Title, phaseNum, phase.Name)
		description := buildPhaseDescription(phase, plan.Title, hydrateParentID)

		issue, err := beads.FallbackCreate(title, description, "task", 2, labels, "")
		if err != nil {
			return fmt.Errorf("failed to create issue for Phase %d: %w", phaseNum, err)
		}

		phaseIDs[idx] = issue.ID
		allPhaseIDs[idx] = issue.ID
		fmt.Printf("  Created Phase %d: %s → %s\n", phaseNum, phase.Name, issue.ID)
	}

	// Add inter-phase dependencies
	depCount := 0
	for _, idx := range toHydrate {
		phase := plan.Phases[idx]
		depIndices := parseDependsOn(phase.DependsOn)

		for _, depIdx := range depIndices {
			depID, ok := allPhaseIDs[depIdx]
			if !ok {
				fmt.Fprintf(os.Stderr, "  Warning: Phase %d depends on Phase %d but no beads ID found\n", idx+1, depIdx+1)
				continue
			}
			thisID := allPhaseIDs[idx]
			// thisID blocks on depID (depID must complete before thisID can start)
			err := beads.FallbackDepAdd(thisID, depID, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: failed to add dependency %s → %s: %v\n", thisID, depID, err)
				continue
			}
			depCount++
			fmt.Printf("  Dependency: Phase %d (%s) blocks on Phase %d (%s)\n", idx+1, thisID, depIdx+1, depID)
		}
	}

	// Write beads IDs back into plan file
	if err := updatePlanWithBeadsIDs(planPath, phaseIDs); err != nil {
		return fmt.Errorf("failed to update plan file: %w", err)
	}

	fmt.Printf("\nHydration complete: %d issues created, %d dependencies wired\n", len(phaseIDs), depCount)
	fmt.Printf("Plan updated: %s\n", planPath)
	return nil
}

// hydrateDryRunOutput shows what would be created without creating anything.
func hydrateDryRunOutput(plan *PlanFile, toHydrate []int, planSlug string) error {
	fmt.Println("[DRY RUN] Would create:")
	for _, idx := range toHydrate {
		phase := plan.Phases[idx]
		phaseNum := idx + 1
		title := buildPhaseTitle(plan.Title, phaseNum, phase.Name)
		fmt.Printf("  Phase %d: %s\n", phaseNum, title)
		fmt.Printf("    Labels: triage:ready, plan:%s\n", planSlug)

		deps := parseDependsOn(phase.DependsOn)
		if len(deps) > 0 {
			depStrs := make([]string, len(deps))
			for i, d := range deps {
				depStrs[i] = fmt.Sprintf("Phase %d", d+1)
			}
			fmt.Printf("    Dependencies: blocks on %s\n", strings.Join(depStrs, ", "))
		}
	}
	return nil
}

// phasesNeedingHydration returns indices of phases without beads IDs.
func phasesNeedingHydration(plan PlanFile) []int {
	var indices []int
	for i, phase := range plan.Phases {
		if len(phase.BeadsIDs) == 0 {
			indices = append(indices, i)
		}
	}
	return indices
}

// parseDependsOn extracts 0-indexed phase numbers from a "Depends on" field.
// Handles: "Nothing", "none", "Phase 1", "Phase 1 (extra text)", "Phases 1-3", "Phase 1, Phase 3"
func parseDependsOn(dep string) []int {
	dep = strings.TrimSpace(dep)
	lower := strings.ToLower(dep)

	if lower == "" || lower == "nothing" || lower == "none" || strings.HasPrefix(lower, "nothing") {
		return nil
	}

	var result []int

	// Match "Phases N-M" range pattern (only first range)
	rangeRe := regexp.MustCompile(`(?i)phases?\s+(\d+)\s*-\s*(\d+)`)
	if m := rangeRe.FindStringSubmatch(dep); len(m) == 3 {
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		for i := start; i <= end; i++ {
			result = append(result, i-1) // convert to 0-indexed
		}
		return result
	}

	// Match individual "Phase N" references
	phaseRe := regexp.MustCompile(`(?i)phase\s+(\d+)`)
	matches := phaseRe.FindAllStringSubmatch(dep, -1)
	for _, m := range matches {
		n, _ := strconv.Atoi(m[1])
		result = append(result, n-1) // convert to 0-indexed
	}

	return result
}

// extractSlugFromFilename extracts the slug from a plan filename.
// "2026-03-11-gate-signal-vs-noise.md" → "gate-signal-vs-noise"
func extractSlugFromFilename(filename string) string {
	// Remove .md extension
	name := strings.TrimSuffix(filename, ".md")

	// Remove date prefix (YYYY-MM-DD-)
	dateRe := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)
	name = dateRe.ReplaceAllString(name, "")

	return name
}

// buildPhaseTitle creates the beads issue title for a plan phase.
func buildPhaseTitle(planTitle string, phaseNum int, phaseName string) string {
	return fmt.Sprintf("Plan: %s — Phase %d: %s", planTitle, phaseNum, phaseName)
}

// buildPhaseDescription creates the beads issue description for a plan phase.
func buildPhaseDescription(phase PlanPhase, planTitle, parentID string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("From plan: %s\n\n", planTitle))
	if parentID != "" {
		b.WriteString(fmt.Sprintf("Parent architect: %s\n\n", parentID))
	}

	if phase.Goal != "" {
		b.WriteString("## Goal\n")
		b.WriteString(phase.Goal)
		b.WriteString("\n\n")
	}

	if phase.DependsOn != "" && strings.ToLower(phase.DependsOn) != "nothing" && strings.ToLower(phase.DependsOn) != "none" {
		b.WriteString(fmt.Sprintf("**Depends on:** %s\n", phase.DependsOn))
	}

	return b.String()
}

// updatePlanWithBeadsIDs writes beads IDs into the plan file's **Beads:** fields.
// Only updates phases whose index is in phaseIDs. Preserves all other content.
func updatePlanWithBeadsIDs(planPath string, phaseIDs map[int]string) error {
	content, err := os.ReadFile(planPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	phaseIndex := -1
	inPhases := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "## Phases" {
			inPhases = true
			continue
		}
		if inPhases && strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			inPhases = false
			continue
		}

		if !inPhases {
			continue
		}

		if strings.HasPrefix(trimmed, "### Phase ") {
			phaseIndex++
			continue
		}

		// Update **Beads:** line for phases that need hydration
		if strings.HasPrefix(trimmed, "**Beads:**") {
			id, ok := phaseIDs[phaseIndex]
			if ok {
				// Preserve leading whitespace
				leading := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				lines[i] = leading + "**Beads:** " + id
			}
		}
	}

	return os.WriteFile(planPath, []byte(strings.Join(lines, "\n")), 0o644)
}
