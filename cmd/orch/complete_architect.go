// Package main provides auto-creation of implementation issues when architect agents complete.
// When an architect agent's SYNTHESIS.md recommends action (not "close"), this creates
// a triage:ready implementation issue with inferred skill, closing the architect→implement gap.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// regexNumberedPrefix matches "1. ", "2. ", etc. at the start of a string.
var regexNumberedPrefix = regexp.MustCompile(`^\d+\.\s+`)

// maybeAutoCreateImplementationIssue checks if a completed architect agent's synthesis
// recommends action, and if so, creates a triage:ready implementation issue.
// Idempotent: if an implementation issue already exists for this architect, returns
// the existing issue pattern without creating a duplicate.
// Returns the created/existing issue ID or empty string if no issue was created.
func maybeAutoCreateImplementationIssue(skillName, beadsID, workspacePath string) string {
	// Only for architect skill
	if skillName != "architect" {
		return ""
	}

	// Parse synthesis
	if workspacePath == "" {
		return ""
	}
	synthesis, err := verify.ParseSynthesis(workspacePath)
	if err != nil || synthesis == nil {
		return ""
	}

	// Check if recommendation is actionable
	if !verify.IsActionableArchitectRecommendation(synthesis.Recommendation) {
		return ""
	}

	// Detect multi-phase structure from SYNTHESIS.md
	var phases []verify.PhaseInfo
	if synthData, readErr := os.ReadFile(filepath.Join(workspacePath, "SYNTHESIS.md")); readErr == nil {
		phases = verify.ExtractPhases(string(synthData))
	}

	if len(phases) >= 2 {
		return createMultiPhaseIssues(phases, synthesis, beadsID, workspacePath)
	}

	// Single-phase: existing behavior
	// Check if implementation issue already exists (idempotency)
	if exists, err := verify.HasImplementationFollowUp(beadsID, ""); err == nil && exists {
		fmt.Printf("Implementation issue already exists for architect %s (skipping auto-create)\n", beadsID)
		return beadsID // Return non-empty to signal issue exists
	}

	// Gather knowledge enrichment (best-effort, 3s timeout)
	projectDir, _ := os.Getwd()
	kbContext := gatherArchitectKBContext(synthesis, projectDir)
	targetFiles := extractTargetFiles(synthesis)

	// Build the implementation issue
	title := buildImplementationTitle(synthesis, beadsID)
	description := buildImplementationDescription(synthesis, beadsID, kbContext, targetFiles)
	skill := inferImplementationSkill(synthesis)

	// Labels: triage:ready for daemon pickup + skill hint
	labels := []string{"triage:ready"}
	if skill != "" {
		labels = append(labels, "skill:"+skill)
	}

	// Create the issue
	issue, err := beads.FallbackCreate(title, description, "task", 2, labels, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to auto-create implementation issue: %v\n", err)
		return ""
	}

	fmt.Printf("\n┌─────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│  AUTO-CREATED IMPLEMENTATION ISSUE                          │\n")
	fmt.Printf("├─────────────────────────────────────────────────────────────┤\n")
	fmt.Printf("│  Issue: %-50s │\n", issue.ID)
	fmt.Printf("│  Skill: %-50s │\n", skill)
	fmt.Printf("│  From:  %-50s │\n", beadsID+" (architect)")
	fmt.Printf("└─────────────────────────────────────────────────────────────┘\n")

	return issue.ID
}

// createMultiPhaseIssues creates one implementation issue per phase in a multi-phase
// architect design. Returns comma-separated issue IDs or empty string on failure.
func createMultiPhaseIssues(phases []verify.PhaseInfo, synthesis *verify.Synthesis, beadsID, workspacePath string) string {
	// Check how many issues already exist (idempotency)
	existingCount, err := verify.CountImplementationFollowUps(beadsID, "")
	if err == nil && existingCount >= len(phases) {
		fmt.Printf("Implementation issues already exist for architect %s (%d/%d, skipping)\n",
			beadsID, existingCount, len(phases))
		return beadsID
	}

	// Gather knowledge enrichment once for all phases (best-effort, 3s timeout)
	projectDir, _ := os.Getwd()
	kbContext := gatherArchitectKBContext(synthesis, projectDir)

	var createdIDs []string
	for _, phase := range phases {
		title := buildArchitectPhaseTitle(phase, beadsID)
		description := buildArchitectPhaseDescription(phase, synthesis, beadsID, kbContext)
		skill := inferImplementationSkill(&verify.Synthesis{
			TLDR: phase.Title,
			Next: phase.Description,
		})

		labels := []string{"triage:ready"}
		if skill != "" {
			labels = append(labels, "skill:"+skill)
		}

		issue, err := beads.FallbackCreate(title, description, "task", 2, labels, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create phase %d issue: %v\n", phase.Number, err)
			continue
		}
		createdIDs = append(createdIDs, issue.ID)
	}

	if len(createdIDs) > 0 {
		fmt.Printf("\n┌─────────────────────────────────────────────────────────────┐\n")
		fmt.Printf("│  AUTO-CREATED IMPLEMENTATION ISSUES (multi-phase)           │\n")
		fmt.Printf("├─────────────────────────────────────────────────────────────┤\n")
		for i, id := range createdIDs {
			if i < len(phases) {
				fmt.Printf("│  Phase %d: %-49s │\n", phases[i].Number, id)
			}
		}
		fmt.Printf("│  From:    %-49s │\n", beadsID+" (architect)")
		fmt.Printf("└─────────────────────────────────────────────────────────────┘\n")
		return strings.Join(createdIDs, ",")
	}
	return ""
}

// buildArchitectPhaseTitle creates a title for a single phase implementation issue.
func buildArchitectPhaseTitle(phase verify.PhaseInfo, beadsID string) string {
	suffix := fmt.Sprintf(" (from architect %s)", beadsID)
	if phase.Title != "" {
		return fmt.Sprintf("Phase %d: %s%s", phase.Number, phase.Title, suffix)
	}
	return fmt.Sprintf("Phase %d implementation%s", phase.Number, suffix)
}

// buildArchitectPhaseDescription creates a description for a single phase implementation issue,
// including the architect's overall summary and phase-specific content.
func buildArchitectPhaseDescription(phase verify.PhaseInfo, synthesis *verify.Synthesis, beadsID, kbContext string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Auto-created from architect review %s (Phase %d of %s).\n\n",
		beadsID, phase.Number, "multi-phase design"))

	if synthesis.TLDR != "" {
		b.WriteString("## Architect Summary\n")
		b.WriteString(synthesis.TLDR)
		b.WriteString("\n\n")
	}

	b.WriteString(fmt.Sprintf("## Phase %d: %s\n", phase.Number, phase.Title))
	if phase.Description != "" {
		b.WriteString(phase.Description)
		b.WriteString("\n\n")
	}

	// Extract target files from phase description
	phaseSynth := &verify.Synthesis{
		Delta:       phase.Description,
		NextActions: []string{phase.Description},
		Next:        phase.Description,
	}
	targetFiles := extractTargetFiles(phaseSynth)
	if len(targetFiles) > 0 {
		b.WriteString("## Target Files\n")
		for _, f := range targetFiles {
			b.WriteString("- `")
			b.WriteString(f)
			b.WriteString("`\n")
		}
		b.WriteString("\n")
	}

	if kbContext != "" {
		b.WriteString("## Relevant Knowledge\n")
		b.WriteString(kbContext)
		b.WriteString("\n")
	}

	return b.String()
}

// isActionableRecommendation wraps the exported verify function for local use.
// Kept for backward compatibility with review_orphans.go and tests.
func isActionableRecommendation(recommendation string) bool {
	return verify.IsActionableArchitectRecommendation(recommendation)
}

// inferImplementationSkill determines the appropriate skill for the follow-up
// implementation based on synthesis content.
func inferImplementationSkill(synthesis *verify.Synthesis) string {
	// Combine relevant text for keyword analysis
	text := strings.ToLower(synthesis.TLDR + " " + synthesis.Next + " " + strings.Join(synthesis.NextActions, " "))

	// Debug/fix signals → systematic-debugging
	debugKeywords := []string{"fix", "debug", "bug", "crash", "error", "broken", "failing"}
	for _, kw := range debugKeywords {
		if strings.Contains(text, kw) {
			return "systematic-debugging"
		}
	}

	// Investigation signals → investigation
	investigationKeywords := []string{"investigate", "analyze", "understand", "explore", "root cause"}
	for _, kw := range investigationKeywords {
		if strings.Contains(text, kw) {
			return "investigation"
		}
	}

	// Default: feature-impl covers implement, refactor, add, create, extract, etc.
	return "feature-impl"
}

// buildImplementationTitle creates a concise title for the implementation issue.
// Uses the first next action if available, otherwise falls back to TLDR.
func buildImplementationTitle(synthesis *verify.Synthesis, beadsID string) string {
	suffix := fmt.Sprintf(" (from architect %s)", beadsID)

	if len(synthesis.NextActions) > 0 {
		action := cleanActionItem(synthesis.NextActions[0])
		return action + suffix
	}

	if synthesis.TLDR != "" {
		return "Implement: " + synthesis.TLDR + suffix
	}

	return "Implementation follow-up" + suffix
}

// buildImplementationDescription creates a detailed description for the implementation issue,
// including context from the architect's synthesis, relevant knowledge, and target files.
func buildImplementationDescription(synthesis *verify.Synthesis, beadsID, kbContext string, targetFiles []string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Auto-created from architect review %s.\n\n", beadsID))

	if synthesis.TLDR != "" {
		b.WriteString("## Architect Summary\n")
		b.WriteString(synthesis.TLDR)
		b.WriteString("\n\n")
	}

	if len(synthesis.NextActions) > 0 {
		b.WriteString("## Recommended Actions\n")
		for _, action := range synthesis.NextActions {
			b.WriteString(action)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if synthesis.Next != "" {
		b.WriteString("## Architect Next Section\n")
		b.WriteString(synthesis.Next)
		b.WriteString("\n\n")
	}

	if len(targetFiles) > 0 {
		b.WriteString("## Target Files\n")
		for _, f := range targetFiles {
			b.WriteString("- `")
			b.WriteString(f)
			b.WriteString("`\n")
		}
		b.WriteString("\n")
	}

	if kbContext != "" {
		b.WriteString("## Relevant Knowledge\n")
		b.WriteString(kbContext)
		b.WriteString("\n")
	}

	return b.String()
}

// regexFilePath matches Go/project file paths like pkg/foo/bar.go, cmd/orch/thing.go, web/src/file.svelte
var regexFilePath = regexp.MustCompile(`(?:^|[\s,:(])([a-zA-Z][\w\-./]*\.(?:go|ts|tsx|js|svelte|yaml|md|sql|sh))`)

// extractTargetFiles pulls file paths from the synthesis Delta and NextActions fields.
func extractTargetFiles(synthesis *verify.Synthesis) []string {
	seen := make(map[string]bool)
	var files []string

	addFiles := func(text string) {
		matches := regexFilePath.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			path := m[1]
			if !seen[path] {
				seen[path] = true
				files = append(files, path)
			}
		}
	}

	addFiles(synthesis.Delta)
	for _, action := range synthesis.NextActions {
		addFiles(action)
	}
	addFiles(synthesis.Next)

	if len(files) == 0 {
		return nil
	}
	return files
}

// gatherArchitectKBContext runs kb context with keywords extracted from the architect's
// synthesis to provide environmental knowledge (constraints, decisions) for the implementing agent.
// Uses a 3s timeout and degrades gracefully on failure.
func gatherArchitectKBContext(synthesis *verify.Synthesis, projectDir string) string {
	text := synthesis.TLDR + " " + strings.Join(synthesis.NextActions, " ")
	keywords := spawn.ExtractKeywords(text, 5)
	if keywords == "" {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kb", "context", keywords)
	if projectDir != "" {
		cmd.Dir = projectDir
	}

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	result := strings.TrimSpace(string(output))
	if result == "" || strings.Contains(result, "No results found") {
		return ""
	}

	return result
}

// cleanActionItem strips bullet/numbered prefixes from an action item string.
func cleanActionItem(item string) string {
	item = strings.TrimSpace(item)
	item = strings.TrimPrefix(item, "- ")
	item = strings.TrimPrefix(item, "* ")
	item = regexNumberedPrefix.ReplaceAllString(item, "")
	return item
}
