// Package main provides the batch-complete command for bulk-closing already-reviewed agents.
// This runs only Tier 1 (skill-aware core) gates on each agent and closes all that pass.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	batchCompleteAll    bool
	batchCompleteDryRun bool
)

var batchCompleteCmd = &cobra.Command{
	Use:   "batch-complete [beads-id...]",
	Short: "Bulk-complete multiple agents with core gates only",
	Long: `Complete multiple agents in batch mode, running only Tier 1 (core) gates.

Universal core gates (always run, cannot be skipped):
  - phase_complete: Agent reported "Phase: Complete"
  - commit_evidence: Agent branch has commits (prevents ghost completions)
  - synthesis: SYNTHESIS.md exists (prevents broken handoffs)

Code core gates (only for code-producing skills):
  - test_evidence: Test execution evidence (prevents shipping untested code)
  - git_diff: Diff matches SYNTHESIS claims (prevents fiction in handoffs)

Quality gates (skipped in batch mode):
  - build, model_connection, verification_spec, visual_verification,
    constraint, phase_gate, skill_output, decision_patch_limit,
    dashboard_health, handoff_content

Use --all to discover and complete all agents that reported Phase: Complete.
Use --dry-run to preview what would be completed without making changes.
Use skip-set/skip-list/skip-clear to control issues excluded from batch runs.

Examples:
  orch batch-complete orch-go-abc1 orch-go-def2 orch-go-ghi3
  orch batch-complete --all
  orch batch-complete --all --dry-run
  orch skip-set orch-go-stuck1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !batchCompleteAll && len(args) == 0 {
			return fmt.Errorf("provide beads IDs or use --all to discover completable agents")
		}

		currentDir, err := currentProjectDir()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Discover or validate agents
		var agents []batchAgent
		if batchCompleteAll {
			agents = discoverCompletableAgents(currentDir)
			if len(agents) == 0 {
				fmt.Println("No agents with Phase: Complete found")
				return nil
			}
		} else {
			for _, id := range args {
				resolved, err := resolveShortBeadsID(id)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to resolve %s: %v\n", id, err)
					continue
				}
				workspace, name := findWorkspaceByBeadsID(currentDir, resolved)
				agents = append(agents, batchAgent{
					BeadsID:       resolved,
					WorkspacePath: workspace,
					AgentName:     name,
				})
			}
		}

		skipSet, err := readBatchSkipSet(currentDir)
		if err != nil {
			return fmt.Errorf("failed to read batch skip list: %w", err)
		}
		if len(skipSet) > 0 {
			fmt.Printf("Loaded %d skipped issue(s) from skip list\n\n", len(skipSet))
		}

		fmt.Printf("Batch completing %d agent(s) (skill-aware core gates only)\n\n", len(agents))

		if batchCompleteDryRun {
			fmt.Println("DRY RUN - no changes will be made")
			for _, agent := range agents {
				if _, isSkipped := skipSet[agent.BeadsID]; isSkipped {
					fmt.Printf("  Would skip: %s", agent.BeadsID)
					if agent.AgentName != "" {
						fmt.Printf(" (%s)", agent.AgentName)
					}
					fmt.Println(" [skip list]")
					continue
				}

				fmt.Printf("  Would complete: %s", agent.BeadsID)
				if agent.AgentName != "" {
					fmt.Printf(" (%s)", agent.AgentName)
				}
				fmt.Println()
			}
			return nil
		}

		// Process each agent
		var passed, failed, skipped int
		for _, agent := range agents {
			if _, isSkipped := skipSet[agent.BeadsID]; isSkipped {
				label := agent.BeadsID
				if agent.AgentName != "" {
					label = fmt.Sprintf("%s (%s)", agent.BeadsID, agent.AgentName)
				}
				fmt.Printf("  SKIPPED: %s [skip list]\n", label)
				skipped++
				continue
			}

			result := batchCompleteOne(agent)
			switch result {
			case "passed":
				passed++
			case "failed":
				failed++
			case "skipped":
				skipped++
			}
		}

		// Summary
		fmt.Printf("\nBatch complete summary: %d passed, %d failed, %d skipped\n", passed, failed, skipped)
		return nil
	},
}

func init() {
	batchCompleteCmd.Flags().BoolVar(&batchCompleteAll, "all", false, "Discover and complete all agents with Phase: Complete")
	batchCompleteCmd.Flags().BoolVar(&batchCompleteDryRun, "dry-run", false, "Preview what would be completed without making changes")
}

// batchAgent represents an agent to be batch-completed.
type batchAgent struct {
	BeadsID       string
	WorkspacePath string
	AgentName     string
}

// discoverCompletableAgents finds all workspaces with Phase: Complete.
func discoverCompletableAgents(projectDir string) []batchAgent {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil
	}

	var agents []batchAgent
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "archived" {
			continue
		}

		dirPath := filepath.Join(workspaceDir, entry.Name())

		// Skip orchestrator workspaces
		if isOrchestratorWorkspace(dirPath) {
			continue
		}

		// Read beads ID
		beadsIDPath := filepath.Join(dirPath, ".beads_id")
		content, err := os.ReadFile(beadsIDPath)
		if err != nil {
			continue
		}
		beadsID := strings.TrimSpace(string(content))
		if beadsID == "" || isUntrackedBeadsID(beadsID) {
			continue
		}

		// Check Phase: Complete
		complete, err := verify.IsPhaseComplete(beadsID)
		if err != nil || !complete {
			continue
		}

		agents = append(agents, batchAgent{
			BeadsID:       beadsID,
			WorkspacePath: dirPath,
			AgentName:     entry.Name(),
		})
	}

	return agents
}

// batchCompleteOne completes a single agent in batch mode.
// Returns "passed", "failed", or "skipped".
func batchCompleteOne(agent batchAgent) string {
	label := agent.BeadsID
	if agent.AgentName != "" {
		label = fmt.Sprintf("%s (%s)", agent.BeadsID, agent.AgentName)
	}

	// Set batch mode globals for runComplete
	prev := completeBatch
	prevNoChangelog := completeNoChangelogCheck

	completeBatch = true
	completeNoChangelogCheck = true

	defer func() {
		completeBatch = prev
		completeNoChangelogCheck = prevNoChangelog
	}()

	err := runComplete(agent.BeadsID, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  FAILED: %s - %v\n", label, err)
		return "failed"
	}

	fmt.Printf("  PASSED: %s\n", label)
	return "passed"
}
