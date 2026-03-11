// Package verify provides verification helpers for agent completion.
// This file implements the plan hydration advisory gate for architect completions.
// When an architect produces a multi-phase plan (.kb/plans/) without creating
// beads issues for each phase ("hydrating" the plan), this gate emits a warning
// suggesting orch plan hydrate. This is advisory only — it does not block completion.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GatePlanHydration is the gate name for plan hydration advisory checks.
const GatePlanHydration = "plan_hydration"

// regexPhaseHeading matches ### Phase headings in plan files.
// Matches both "### Phase 1: Name" and "### Name" (unnumbered) under a ## Phases section.
var regexPhaseHeading = regexp.MustCompile(`(?m)^### .+`)

// PlanHydrationResult contains the advisory outcome of the plan hydration check.
type PlanHydrationResult struct {
	Warnings []string // Advisory warnings about unhydrated plans
}

// CheckPlanHydration checks if an architect agent produced multi-phase plan files
// without hydrating them into beads issues. Returns nil if no advisory is needed.
//
// This is an advisory gate (warnings only, never blocks completion).
// It only activates for architect skill completions.
func CheckPlanHydration(skillName, workspacePath, projectDir string) *PlanHydrationResult {
	if strings.ToLower(skillName) != "architect" {
		return nil
	}

	if projectDir == "" {
		return nil
	}

	// Find plan files in .kb/plans/
	plansDir := filepath.Join(projectDir, ".kb", "plans")
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return nil // No plans directory or unreadable — nothing to check
	}

	var warnings []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(plansDir, entry.Name()))
		if err != nil {
			continue
		}

		phases := CountPlanPhases(string(content))
		if phases > 1 {
			warnings = append(warnings,
				fmt.Sprintf("Plan %s has %d phases but may not be hydrated into beads issues. Consider: orch plan hydrate .kb/plans/%s",
					entry.Name(), phases, entry.Name()))
		}
	}

	if len(warnings) == 0 {
		return nil
	}

	return &PlanHydrationResult{
		Warnings: warnings,
	}
}

// CountPlanPhases counts the number of phase subsections under a ## Phases section.
// Returns 0 if no ## Phases section is found.
func CountPlanPhases(content string) int {
	// Find the ## Phases section
	phasesIdx := strings.Index(content, "## Phases")
	if phasesIdx == -1 {
		return 0
	}

	// Extract content from ## Phases to the next ## heading or end of file
	afterPhases := content[phasesIdx+len("## Phases"):]

	// Find the next ## heading (not ###) that ends the Phases section
	nextH2 := -1
	lines := strings.Split(afterPhases, "\n")
	charCount := 0
	for i, line := range lines {
		if i > 0 && strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			nextH2 = charCount
			break
		}
		charCount += len(line) + 1 // +1 for newline
	}

	var phasesSection string
	if nextH2 > 0 {
		phasesSection = afterPhases[:nextH2]
	} else {
		phasesSection = afterPhases
	}

	// Count ### headings within the Phases section
	matches := regexPhaseHeading.FindAllString(phasesSection, -1)
	return len(matches)
}
