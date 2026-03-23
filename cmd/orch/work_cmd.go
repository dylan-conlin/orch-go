// Package main provides the work command for daemon-driven agent spawning.
// This file contains:
// - work command definition and flags
// - skill/browser tool inference from beads issues
// - beads label loading
package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Work command flags
	workInline bool   // Run inline (blocking) with TUI
	workSkill  string // Pre-inferred skill from daemon (overrides local inference)

	// spawnOrientationFrame holds separate context from the task title.
	// Set by runWork from the beads issue description, rendered as
	// ORIENTATION_FRAME: section in SPAWN_CONTEXT.md (separate from TASK:).
	spawnOrientationFrame string

	// spawnIssueType holds the beads issue type (feature, bug, task, etc.).
	// Set by runWork from the beads issue, used for review tier inference.
	spawnIssueType string
)

var workCmd = &cobra.Command{
	Use:   "work [beads-id]",
	Short: "Start work on a beads issue with skill inference",
	Long: `Start work on a beads issue by inferring the skill from the issue type.

The skill is automatically determined from the issue type:
  - bug         → systematic-debugging
  - feature     → feature-impl
  - task        → feature-impl
  - investigation → investigation

The issue description becomes the task prompt for the spawned agent.

By default, spawns in a tmux window (visible, interruptible).
Use --inline to run in the current terminal (blocking with TUI).

Examples:
  orch-go work proj-123           # Start work in tmux window (default)
  orch-go work proj-123 --inline  # Start work inline (blocking TUI)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		spawnModeSet = false
		spawnValidationSet = false
		return runWork(serverURL, beadsID, workInline)
	},
}

func init() {
	workCmd.Flags().BoolVar(&workInline, "inline", false, "Run inline (blocking) with TUI")
	workCmd.Flags().StringVar(&workSkill, "skill", "", "Pre-inferred skill from daemon (skips local inference)")
	workCmd.Flags().StringVar(&spawnModel, "model", "", "Model alias (opus, sonnet) or provider/model format")
	workCmd.Flags().StringVar(&spawnWorkdir, "workdir", "", "Target project directory (for cross-project work)")
	workCmd.Flags().StringVar(&spawnAccount, "account", "", "Account name for Claude CLI spawns (e.g., 'work', 'personal')")
}

// InferSkillFromIssueType maps issue types to appropriate skills.
// Returns an error for types that cannot be spawned (e.g., epic) or unknown types.
//
// Bug handling: Defaults to "architect" (understand before fixing) rather than
// "systematic-debugging". This implements the "Premise Before Solution" principle -
// most bugs reported as vague symptoms need understanding before patching.
// Use explicit skill:systematic-debugging label for isolated bugs with clear cause.
func InferSkillFromIssueType(issueType string) (string, error) {
	switch issueType {
	case "bug":
		return "systematic-debugging", nil
	case "feature":
		return "feature-impl", nil
	case "task":
		return "feature-impl", nil
	case "investigation":
		return "investigation", nil
	case "epic":
		return "", fmt.Errorf("cannot spawn work on epic issues - epics are decomposed into sub-issues")
	case "":
		return "", fmt.Errorf("issue type is empty")
	default:
		return "", fmt.Errorf("unknown issue type: %s", issueType)
	}
}

// inferSkillFromBeadsIssue infers skill from a beads issue using labels, title, then type.
func inferSkillFromBeadsIssue(issue *beads.Issue) string {
	// Check for skill:* labels first
	for _, label := range issue.Labels {
		if strings.HasPrefix(label, "skill:") {
			return strings.TrimPrefix(label, "skill:")
		}
	}

	// Check for title patterns (e.g., synthesis issues)
	if strings.HasPrefix(issue.Title, "Synthesize ") && strings.Contains(issue.Title, " investigations") {
		return "kb-reflect"
	}

	// Fall back to type-based inference
	skill, err := InferSkillFromIssueType(issue.IssueType)
	if err != nil {
		return "feature-impl" // Default fallback
	}
	return skill
}

// inferBrowserToolFromBeadsIssue extracts browser tool requirements from issue labels.
// Returns "playwright-cli" if needs:playwright label is found, or empty string otherwise.
//
// This allows daemon-spawned agents to automatically get browser automation context
// (playwright-cli) when working on UI/CSS fixes that require visual verification.
func inferBrowserToolFromBeadsIssue(issue *beads.Issue) string {
	for _, label := range issue.Labels {
		if strings.HasPrefix(label, "needs:") {
			need := strings.TrimPrefix(label, "needs:")
			switch need {
			case "playwright":
				return "playwright-cli" // Triggers playwright-cli context injection
			}
		}
	}
	return ""
}

func loadBeadsLabels(beadsID, projectDir string) []string {
	if beadsID == "" {
		return nil
	}
	socketPath, connErr := beads.FindSocketPath(projectDir)
	if connErr != nil {
		return nil
	}
	beadsClient := beads.NewClient(socketPath)
	if err := beadsClient.Connect(); err != nil {
		return nil
	}
	defer beadsClient.Close()
	beadsIssue, showErr := beadsClient.Show(beadsID)
	if showErr != nil {
		return nil
	}
	return beadsIssue.Labels
}

func runWork(serverURL, beadsID string, inline bool) error {
	// For cross-project work (--workdir set), resolve the target project directory
	// so verify/beads operations use the correct .beads/ database.
	var workProjectDir string
	if spawnWorkdir != "" {
		absWorkdir, err := filepath.Abs(spawnWorkdir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir: %w", err)
		}
		workProjectDir = absWorkdir
	}

	// Get issue details from verify (for description)
	issue, err := verify.GetIssue(beadsID, workProjectDir)
	if err != nil {
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	// Use daemon-provided skill if available (--skill flag).
	// This ensures the daemon's label-aware inference (skill:* labels > title > type)
	// is preserved through the spawn chain, avoiding re-inference failures when
	// beads connection fails for cross-project issues.
	var skillName string
	var browserTool string
	if workSkill != "" {
		skillName = workSkill
	}

	// Infer skill and browser tool from issue (labels, title pattern, then type)
	// Use beads.Issue which has Labels for full skill/browser tool inference
	socketPath, connErr := beads.FindSocketPath(workProjectDir)
	if connErr == nil {
		beadsClient := beads.NewClient(socketPath)
		if connErr := beadsClient.Connect(); connErr == nil {
			defer beadsClient.Close()
			beadsIssue, showErr := beadsClient.Show(beadsID)
			if showErr == nil {
				if skillName == "" {
					skillName = inferSkillFromBeadsIssue(beadsIssue)
				}
				browserTool = inferBrowserToolFromBeadsIssue(beadsIssue)
			}
		}
	}
	// Fall back to type-only inference if beads fails and no daemon skill
	if skillName == "" {
		skillName, err = InferSkillFromIssueType(issue.IssueType)
		if err != nil {
			return fmt.Errorf("cannot work on issue %s: %w", beadsID, err)
		}
	}

	// Use issue title as the TASK (concise, drives workspace name slug).
	// Issue description goes into a separate ORIENTATION_FRAME section in SPAWN_CONTEXT.md.
	// This prevents long descriptions from polluting workspace names (e.g., "orientation-frame-...").
	task := issue.Title
	spawnOrientationFrame = issue.Description

	// Also check for FRAME beads comments — these contain strategic context added by the
	// orchestrator after issue creation. If the issue description is empty but a FRAME
	// comment exists, use it as the orientation frame. If both exist, append the FRAME
	// comment for richer context.
	if frame := spawn.ExtractFrameFromBeadsComments(beadsID); frame != "" {
		if spawnOrientationFrame == "" {
			spawnOrientationFrame = frame
		} else if !strings.Contains(spawnOrientationFrame, frame) {
			// Append FRAME comment if it adds new information beyond the description
			spawnOrientationFrame = spawnOrientationFrame + "\n\n**Orchestrator Frame:** " + frame
		}
	}

	// Set the spawnIssue flag so runSpawnWithSkillInternal uses the existing issue
	spawnIssue = beadsID
	spawnIssueType = issue.IssueType

	// NOTE: Do NOT load user config default_model into spawnModel here.
	// spawnModel maps to CLI.Model in the resolve pipeline (highest priority).
	// User config default_model is already handled at correct precedence in
	// pkg/spawn/resolve.go:resolveModel() via ResolveInput.UserConfig.

	fmt.Printf("Starting work on: %s\n", beadsID)
	fmt.Printf("  Title:  %s\n", issue.Title)
	fmt.Printf("  Type:   %s\n", issue.IssueType)
	fmt.Printf("  Skill:  %s\n", skillName)
	if spawnModel != "" {
		fmt.Printf("  Model:  %s\n", spawnModel)
	}
	if browserTool != "" {
		fmt.Printf("  Browser: %s\n", browserTool)
	}

	// Work command is daemon-driven (issue already created and triaged)
	// Pass daemonDriven=true to skip triage bypass check
	return runSpawnWithSkillInternal(serverURL, skillName, task, inline, true, false, false, true)
}
