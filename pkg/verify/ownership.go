// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// GateOwnershipReconciliation is Gate 15: at completion, verify all tracked dirty
// files since baseline are either committed or belong to an allowed artifact class.
const GateOwnershipReconciliation = "ownership_reconciliation"

// ArtifactClass defines ownership and tracking policy for a category of files.
type ArtifactClass struct {
	Name              string // Human-readable class name
	RequiresOwnership bool   // Whether dirty files in this class must be owned by an open issue
}

// OwnershipResult represents the result of the ownership reconciliation gate.
type OwnershipResult struct {
	Passed       bool     // Whether verification passed
	Errors       []string // Error messages (blocking)
	Warnings     []string // Warning messages (non-blocking)
	UnownedFiles []string // Tracked dirty files that are unowned and require ownership
	AllowedFiles []string // Tracked dirty files in allowed-residue classes
	PreBaseline  []string // Dirty files that existed before this agent's baseline
}

// ClassifyArtifact determines the artifact class for a file path.
// Unknown files default to "source" (requires ownership).
func ClassifyArtifact(path string) ArtifactClass {
	switch {
	// Local state — always dirty, never requires ownership
	case strings.HasPrefix(path, ".beads/"):
		return ArtifactClass{Name: "local-state", RequiresOwnership: false}

	// Generated workspace — ephemeral per-session
	case strings.HasPrefix(path, ".orch/workspace/"):
		return ArtifactClass{Name: "generated-workspace", RequiresOwnership: false}

	// Experiment results — archived separately
	case strings.Contains(path, "experiments/") && strings.Contains(path, "/results/"):
		return ArtifactClass{Name: "experiment-results", RequiresOwnership: false}

	// Knowledge backlog — committed in batches, allowed residue
	case strings.HasPrefix(path, ".kb/"):
		return ArtifactClass{Name: "knowledge-backlog", RequiresOwnership: false}

	// Skill compilation stats — auto-generated, allowed residue
	case strings.HasPrefix(path, "skills/") && strings.HasSuffix(path, "stats.json"):
		return ArtifactClass{Name: "skill-stats", RequiresOwnership: false}

	// Docs — markdown files in tracked locations
	case strings.HasSuffix(path, ".md"):
		return ArtifactClass{Name: "docs", RequiresOwnership: true}

	// Default: source — requires ownership
	default:
		return ArtifactClass{Name: "source", RequiresOwnership: true}
	}
}

// VerifyOwnershipReconciliation checks that no tracked dirty files remain from this
// agent's work that are unowned. This is Gate 15 in the completion pipeline.
//
// Algorithm:
//  1. Get git baseline from AGENT_MANIFEST.json
//  2. Compute tracked dirty files (uncommitted changes to tracked files)
//  3. For each dirty file, check if it was dirty BEFORE the agent's baseline
//  4. For post-baseline dirty files, classify by artifact class
//  5. Files in allowed-residue classes pass; source/docs files fail
//
// Returns nil if the gate is not applicable (no workspace, no baseline, no dirty files).
func VerifyOwnershipReconciliation(workspacePath, projectDir string) *OwnershipResult {
	if workspacePath == "" || projectDir == "" {
		return nil
	}

	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	baseline := manifest.GitBaseline
	if baseline == "" {
		return nil
	}

	// Get all tracked dirty files (staged + unstaged modifications to tracked files)
	dirtyFiles := getTrackedDirtyFiles(projectDir)
	if len(dirtyFiles) == 0 {
		return nil
	}

	// Get files that were dirty BEFORE the baseline (not this agent's responsibility)
	preBaselineDirty := getPreBaselineDirtyFiles(projectDir, baseline)
	preBaselineSet := make(map[string]bool)
	for _, f := range preBaselineDirty {
		preBaselineSet[f] = true
	}

	result := &OwnershipResult{
		Passed: true,
	}

	for _, file := range dirtyFiles {
		// Skip pre-baseline dirt
		if preBaselineSet[file] {
			result.PreBaseline = append(result.PreBaseline, file)
			continue
		}

		class := ClassifyArtifact(file)
		if !class.RequiresOwnership {
			result.AllowedFiles = append(result.AllowedFiles, file)
			continue
		}

		// This is a post-baseline, ownership-requiring dirty file
		result.UnownedFiles = append(result.UnownedFiles, file)
	}

	if len(result.UnownedFiles) > 0 {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("ownership reconciliation: %d tracked file(s) modified since baseline are uncommitted", len(result.UnownedFiles)))
		for _, f := range result.UnownedFiles {
			result.Errors = append(result.Errors, fmt.Sprintf("  uncommitted: %s", f))
		}
		result.Errors = append(result.Errors, "commit these files or transfer ownership to another issue before completing")
	}

	if len(result.AllowedFiles) > 0 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("ownership reconciliation: %d dirty file(s) in allowed-residue classes (not blocking)", len(result.AllowedFiles)))
	}

	return result
}

// getTrackedDirtyFiles returns all tracked files with uncommitted changes.
// This includes both staged and unstaged modifications, but NOT untracked files.
func getTrackedDirtyFiles(projectDir string) []string {
	// git diff --name-only HEAD shows all tracked files that differ from HEAD
	// (both staged and unstaged changes)
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Also get staged-only changes (new files that are git add'd but not committed)
	cmdStaged := exec.Command("git", "diff", "--name-only", "--cached")
	cmdStaged.Dir = projectDir
	outStaged, _ := cmdStaged.Output()

	seen := make(map[string]bool)
	var files []string

	for _, output := range [][]byte{out, outStaged} {
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !seen[line] {
				seen[line] = true
				files = append(files, line)
			}
		}
	}

	return files
}

// getPreBaselineDirtyFiles determines which files were already dirty before the
// agent's baseline commit.
//
// A file is pre-baseline dirt if:
// - It is dirty in the working tree (differs from HEAD)
// - AND its content at HEAD is identical to its content at baseline
//   (no post-baseline commits changed it)
// - AND it's not staged (staged files are the agent's work-in-progress)
//
// If a file was modified by a post-baseline commit but is STILL dirty,
// that's the agent's responsibility (they committed partial work).
func getPreBaselineDirtyFiles(projectDir, baseline string) []string {
	// Get files that differ between baseline and HEAD (files touched by post-baseline commits)
	cmd := exec.Command("git", "diff", "--name-only", baseline, "HEAD")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		// If baseline is invalid/gc'd, conservatively treat all dirt as post-baseline
		return nil
	}

	postBaselineCommitted := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			postBaselineCommitted[line] = true
		}
	}

	// Get files that are staged (these are the agent's work-in-progress, not pre-baseline)
	cmdStaged := exec.Command("git", "diff", "--name-only", "--cached")
	cmdStaged.Dir = projectDir
	outStaged, _ := cmdStaged.Output()
	stagedFiles := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(outStaged)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			stagedFiles[line] = true
		}
	}

	// Get all dirty tracked files (unstaged only — differs from HEAD in working tree)
	cmdDirty := exec.Command("git", "diff", "--name-only")
	cmdDirty.Dir = projectDir
	outDirty, err := cmdDirty.Output()
	if err != nil {
		return nil
	}

	// Files that are dirty (working tree != HEAD) AND NOT changed by post-baseline
	// commits AND NOT staged are pre-baseline dirt
	var preBaseline []string
	for _, line := range strings.Split(strings.TrimSpace(string(outDirty)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !postBaselineCommitted[line] && !stagedFiles[line] {
			preBaseline = append(preBaseline, line)
		}
	}

	return preBaseline
}
