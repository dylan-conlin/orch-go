package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// runReviewSingle displays detailed review information for a single agent.
func runReviewSingle(beadsID, workdir string) error {
	// Try to find workspace from current directory
	currentDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Resolve project directory using shared helper
	projectResult, err := resolveProjectDir(workdir, "", currentDir)
	if err != nil {
		return err
	}
	projectDir := projectResult.ProjectDir

	// Set beads.DefaultDir for cross-project operations
	projectResult.SetBeadsDefaultDir()

	// Find workspace by beads ID (searches SPAWN_CONTEXT.md, not just directory name)
	workspacePath, _ := findWorkspaceByBeadsID(projectDir, beadsID)

	// Get review data
	review, err := verify.GetAgentReview(beadsID, workspacePath, projectDir)
	if err != nil {
		return fmt.Errorf("failed to get agent review: %w", err)
	}

	// Derive skill from workspace name if available
	if workspacePath != "" {
		review.Skill = extractSkillFromTitle(filepath.Base(workspacePath))
	}

	// Check if this is a light tier agent
	if workspacePath != "" {
		review.IsLightTier = isLightTierWorkspace(workspacePath)
	}

	// Display the review
	fmt.Print(verify.FormatAgentReview(review))

	// Print next steps
	fmt.Println("---")
	if review.Status == "Phase: Complete" {
		// Light tier agents are ready without SYNTHESIS.md
		if review.SynthesisExists || review.IsLightTier {
			fmt.Printf("Ready to complete: orch complete %s\n", beadsID)
		} else {
			fmt.Println("Missing: SYNTHESIS.md - agent should create this before completing")
			fmt.Printf("\nTo force completion: orch complete %s --force\n", beadsID)
		}
	} else {
		if !review.SynthesisExists && !review.IsLightTier {
			fmt.Println("Missing: SYNTHESIS.md - agent should create this before completing")
		}
		fmt.Println("Missing: Phase: Complete - agent should report via bd comment")
		fmt.Printf("\nTo force completion: orch complete %s --force\n", beadsID)
	}

	return nil
}

// printNoneReady displays messaging when no agents are ready to complete
// and lists agents needing manual review with their failure reasons.
func printNoneReady(needsReview []CompletionInfo) {
	fmt.Println("\nNo agents ready to complete (need Phase: Complete and valid beads ID)")
	if len(needsReview) > 0 {
		fmt.Println("\nAgents needing manual review:")
		for _, c := range needsReview {
			fmt.Printf("  - %s: %s\n", c.WorkspaceID, reviewFailureReason(c))
		}
	}
}

// reviewFailureReason returns a human-readable reason why a completion needs review.
func reviewFailureReason(c CompletionInfo) string {
	if c.BeadsID == "" {
		return "missing beads ID"
	}
	if c.VerifyError != "" {
		return c.VerifyError
	}
	return "verification failed"
}

// printReviewDoneSummary displays the final summary of the review done operation.
func printReviewDoneSummary(completed, total int, errors []string, needsReview []CompletionInfo) {
	fmt.Printf("\n---\n")
	fmt.Printf("Completed: %d/%d agents\n", completed, total)

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nErrors (%d):\n", len(errors))
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
	}

	if len(needsReview) > 0 {
		fmt.Printf("\nAgents needing manual review (%d):\n", len(needsReview))
		for _, c := range needsReview {
			reason := "missing beads ID"
			if c.BeadsID != "" {
				reason = "verification failed"
			}
			fmt.Printf("  - %s: %s\n", c.WorkspaceID, reason)
		}
	}
}

// printSynthesisCard displays a condensed Synthesis Card for an agent.
// Shows the D.E.K.N. sections (Delta, Evidence, Knowledge, Next) in a compact format.
func printSynthesisCard(s *verify.Synthesis) {
	indent := "         "

	// TLDR is always shown if available
	if s.TLDR != "" {
		// Truncate TLDR if too long (single line display)
		tldr := s.TLDR
		if len(tldr) > 100 {
			tldr = tldr[:97] + "..."
		}
		// Replace newlines with spaces for single-line display
		tldr = strings.ReplaceAll(tldr, "\n", " ")
		fmt.Printf("%sTLDR:  %s\n", indent, tldr)
	}

	// Outcome and Recommendation (condensed line)
	if s.Outcome != "" || s.Recommendation != "" {
		var meta []string
		if s.Outcome != "" {
			meta = append(meta, fmt.Sprintf("outcome=%s", s.Outcome))
		}
		if s.Recommendation != "" {
			meta = append(meta, fmt.Sprintf("rec=%s", s.Recommendation))
		}
		fmt.Printf("%sStatus: %s\n", indent, strings.Join(meta, ", "))
	}

	// Delta summary (files changed, commits)
	if s.Delta != "" {
		deltaSummary := summarizeDelta(s.Delta)
		if deltaSummary != "" {
			fmt.Printf("%sDelta: %s\n", indent, deltaSummary)
		}
	}

	// Next Actions
	if len(s.NextActions) > 0 {
		fmt.Printf("%sNext:\n", indent)
		// Show at most 3 actions to keep it condensed
		maxActions := 3
		for i, action := range s.NextActions {
			if i >= maxActions {
				fmt.Printf("%s  ... +%d more\n", indent, len(s.NextActions)-maxActions)
				break
			}
			// Truncate long actions
			if len(action) > 80 {
				action = action[:77] + "..."
			}
			fmt.Printf("%s  %s\n", indent, action)
		}
	}
}

// summarizeDelta creates a one-line summary of the Delta section.
// Extracts file counts and commit info.
func summarizeDelta(delta string) string {
	var parts []string

	// Count files created
	createdCount := strings.Count(delta, "### Files Created")
	if createdCount > 0 {
		// Count bullet points in the section
		fileCount := countBulletPoints(delta, "### Files Created")
		if fileCount > 0 {
			parts = append(parts, fmt.Sprintf("%d files created", fileCount))
		}
	}

	// Count files modified
	modifiedCount := strings.Count(delta, "### Files Modified")
	if modifiedCount > 0 {
		fileCount := countBulletPoints(delta, "### Files Modified")
		if fileCount > 0 {
			parts = append(parts, fmt.Sprintf("%d files modified", fileCount))
		}
	}

	// Count commits
	commitsCount := strings.Count(delta, "### Commits")
	if commitsCount > 0 {
		commitCount := countBulletPoints(delta, "### Commits")
		if commitCount > 0 {
			parts = append(parts, fmt.Sprintf("%d commits", commitCount))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, ", ")
}

// countBulletPoints counts bullet points (-) after a section header.
func countBulletPoints(content, sectionHeader string) int {
	idx := strings.Index(content, sectionHeader)
	if idx == -1 {
		return 0
	}

	// Find content after header
	afterHeader := content[idx+len(sectionHeader):]

	// Find end (next ### or end of content)
	endIdx := strings.Index(afterHeader, "\n###")
	if endIdx == -1 {
		endIdx = len(afterHeader)
	}

	section := afterHeader[:endIdx]

	// Count lines starting with -
	count := 0
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			count++
		}
	}

	return count
}
