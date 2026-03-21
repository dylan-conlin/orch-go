// Package main provides the reject command for quality rejection of agent work.
// This is a 1-step negative feedback verb that closes the broken feedback loop
// in the completion pipeline (0 quality rejections in 1,113 completions).
package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/identity"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	rejectCategory string
	rejectWorkdir  string
)

var rejectCmd = &cobra.Command{
	Use:   "reject <beads-id> <reason>",
	Short: "Reject agent work quality and reopen issue",
	Long: `Reject an agent's completed work due to quality issues.

This is a 1-step negative feedback verb matching the friction level of 'orch complete'.
The beads issue is reopened, tagged with 'rejected', and made ready for reassignment.

An agent.rejected event is emitted so the daemon learning loop gains negative signal
(skill failure rates become real instead of showing 100% success).

Categories:
  quality   - Work doesn't meet standards (default)
  scope     - Work addresses wrong scope or missed requirements
  approach  - Fundamentally wrong approach, needs rethinking
  stale     - Work is outdated or superseded

Examples:
  orch reject orch-go-abc12 "Tests don't cover edge cases"
  orch reject orch-go-abc12 "Wrong API design" --category approach
  orch reject kb-cli-xyz "Outdated after refactor" --category stale --workdir ~/projects/kb-cli`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReject(args[0], args[1], rejectCategory, rejectWorkdir)
	},
}

func init() {
	rejectCmd.Flags().StringVar(&rejectCategory, "category", "quality", "Rejection category: quality, scope, approach, stale")
	rejectCmd.Flags().StringVar(&rejectWorkdir, "workdir", "", "Target project directory (for cross-project rejection)")
}

func runReject(beadsID, reason, category, workdir string) error {
	// Validate category
	validCategories := map[string]bool{"quality": true, "scope": true, "approach": true, "stale": true}
	if !validCategories[category] {
		return fmt.Errorf("invalid category %q: must be one of quality, scope, approach, stale", category)
	}

	// --- Phase 1: Resolve project directory ---
	projectDir, err := identity.ResolveProject(beadsID, workdir)
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// --- Phase 2: Validate beads issue ---
	issue, err := verify.GetIssue(beadsID, projectDir)
	if err != nil {
		return fmt.Errorf("failed to get beads issue %s: %w", beadsID, err)
	}

	if issue.Status != "closed" {
		return fmt.Errorf("issue %s is not closed (status: %s) — reject is for completed work", beadsID, issue.Status)
	}

	// Check for existing rejected label (dedup)
	for _, label := range issue.Labels {
		if label == "rejected" {
			return fmt.Errorf("issue %s is already rejected", beadsID)
		}
	}

	// --- Phase 3: Extract skill/model from workspace (best effort) ---
	var originalSkill, originalModel string
	wsPath, _ := findWorkspaceByBeadsID(projectDir, beadsID)
	if wsPath != "" {
		manifest := spawn.ReadAgentManifestWithFallback(wsPath)
		originalSkill = manifest.Skill
		originalModel = manifest.Model
	}

	// --- Phase 4: Reopen issue ---
	reopenCmd := exec.Command("bd", "reopen", beadsID, "--reason", fmt.Sprintf("Rejected: %s", reason))
	reopenCmd.Dir = projectDir
	reopenCmd.Stderr = os.Stderr
	if err := reopenCmd.Run(); err != nil {
		return fmt.Errorf("failed to reopen issue %s: %w", beadsID, err)
	}
	fmt.Printf("Reopened issue: %s\n", beadsID)

	// --- Phase 5: Add rejection comment ---
	commentText := fmt.Sprintf("Rejected (%s): %s", category, reason)
	commentCmd := exec.Command("bd", "comments", "add", beadsID, commentText)
	commentCmd.Dir = projectDir
	commentCmd.Stderr = os.Stderr
	if err := commentCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add rejection comment: %v\n", err)
	}

	// --- Phase 6: Add labels ---
	labelCmd := exec.Command("bd", "label", "add", beadsID, "rejected")
	labelCmd.Dir = projectDir
	labelCmd.Stderr = os.Stderr
	if err := labelCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add rejected label: %v\n", err)
	}

	triageCmd := exec.Command("bd", "label", "add", beadsID, "triage:ready")
	triageCmd.Dir = projectDir
	triageCmd.Stderr = os.Stderr
	if err := triageCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add triage:ready label: %v\n", err)
	}

	// --- Phase 7: Emit agent.rejected event ---
	logger := events.NewLogger(events.DefaultLogPath())
	if err := logger.LogAgentRejected(events.AgentRejectedData{
		BeadsID:       beadsID,
		Reason:        reason,
		Category:      category,
		OriginalSkill: originalSkill,
		OriginalModel: originalModel,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log rejection event: %v\n", err)
	}

	// --- Phase 8: Summary ---
	fmt.Printf("Rejected agent work: %s\n", beadsID)
	fmt.Printf("  Category: %s\n", category)
	fmt.Printf("  Reason: %s\n", reason)
	if originalSkill != "" {
		fmt.Printf("  Skill: %s\n", originalSkill)
	}
	fmt.Printf("  Use 'orch work %s' to reassign\n", beadsID)

	return nil
}

