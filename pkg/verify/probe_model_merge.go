// Package verify provides verification helpers for agent completion.
// This file implements the probe-to-model merge gate: probes with "contradicts"
// or "extends" verdicts must show evidence that the parent model.md was updated.
package verify

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// GateProbeModelMerge is the gate name for probe-to-model merge verification.
const GateProbeModelMerge = "probe_model_merge"

// ProbeModelMergeResult holds the result of verifying probe-to-model merge.
type ProbeModelMergeResult struct {
	Passed       bool
	Errors       []string
	Warnings     []string
	GatesFailed  []string
	UnmergedProbes []ProbeVerdict // Probes with contradicts/extends that lack model updates
}

// CheckProbeModelMerge verifies that probes with "contradicts" or "extends" verdicts
// have corresponding model.md updates in the git diff since spawn time.
// Returns nil if no probes require model updates.
func CheckProbeModelMerge(workspacePath, projectDir string) *ProbeModelMergeResult {
	if workspacePath == "" || projectDir == "" {
		return nil
	}

	// Find probes produced during this session
	probes := FindProbesForWorkspace(workspacePath, projectDir)
	if len(probes) == 0 {
		return nil
	}

	// Filter to probes that require model updates
	var actionableProbes []ProbeVerdict
	for _, p := range probes {
		if p.Verdict == "contradicts" || p.Verdict == "extends" {
			actionableProbes = append(actionableProbes, p)
		}
	}

	if len(actionableProbes) == 0 {
		return nil // Only "confirms" verdicts — no model update required
	}

	// Get git diff files since spawn time to check for model.md updates
	modifiedFiles, err := getModifiedModelFiles(workspacePath, projectDir)
	if err != nil {
		// Can't verify — warn but don't block
		return &ProbeModelMergeResult{
			Passed:   true,
			Warnings: []string{fmt.Sprintf("could not check model file updates: %v", err)},
		}
	}

	// Build set of modified model files for O(1) lookup
	modifiedModelSet := make(map[string]bool)
	for _, f := range modifiedFiles {
		// Normalize: .kb/models/foo/model.md → foo
		if isModelFile(f) {
			modelName := extractModelNameFromPath(f)
			if modelName != "" {
				modifiedModelSet[modelName] = true
			}
		}
	}

	// Check each actionable probe for a corresponding model update
	result := &ProbeModelMergeResult{Passed: true}
	for _, probe := range actionableProbes {
		if !modifiedModelSet[probe.ModelName] {
			result.UnmergedProbes = append(result.UnmergedProbes, probe)
		}
	}

	if len(result.UnmergedProbes) > 0 {
		result.Passed = false
		result.GatesFailed = append(result.GatesFailed, GateProbeModelMerge)

		var details []string
		for _, p := range result.UnmergedProbes {
			details = append(details, fmt.Sprintf("  - %s verdict '%s' on model '%s': %s",
				filepath.Base(p.ProbePath), p.Verdict, p.ModelName, p.Details))
		}

		result.Errors = append(result.Errors,
			fmt.Sprintf("Probe-to-model merge required: %d probe(s) with '%s'/'%s' verdicts have no model.md update:",
				len(result.UnmergedProbes), "contradicts", "extends"))
		result.Errors = append(result.Errors, details...)
		result.Errors = append(result.Errors,
			"Agent must merge probe findings into .kb/models/{model}/model.md before completion")
	}

	return result
}

// getModifiedModelFiles returns files modified since spawn time, filtered to .kb/models/ paths.
func getModifiedModelFiles(workspacePath, projectDir string) ([]string, error) {
	// Read spawn time and baseline from workspace
	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	spawnTime := manifest.ParseSpawnTime()
	baseline := manifest.GitBaseline

	// Cross-repo check: discard baseline if from different repo
	if baseline != "" && manifest.ProjectDir != "" && filepath.Clean(manifest.ProjectDir) != filepath.Clean(projectDir) {
		baseline = ""
	}

	if spawnTime.IsZero() && baseline == "" {
		return nil, fmt.Errorf("spawn time and git baseline unavailable")
	}

	// Get all modified files
	allFiles, err := GetGitDiffFiles(projectDir, spawnTime, baseline)
	if err != nil {
		return nil, err
	}

	// Also check uncommitted changes (staged + unstaged)
	uncommitted, err := getUncommittedFiles(projectDir)
	if err == nil {
		allFiles = append(allFiles, uncommitted...)
	}

	// Filter to .kb/models/ files
	var modelFiles []string
	seen := make(map[string]bool)
	for _, f := range allFiles {
		normalized := NormalizePath(f)
		if strings.Contains(normalized, ".kb/models/") && !seen[normalized] {
			modelFiles = append(modelFiles, normalized)
			seen[normalized] = true
		}
	}

	return modelFiles, nil
}

// getUncommittedFiles returns files with uncommitted changes (staged + unstaged).
func getUncommittedFiles(projectDir string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Also get staged changes
	cmdStaged := exec.Command("git", "diff", "--name-only", "--cached")
	cmdStaged.Dir = projectDir
	stagedOutput, err := cmdStaged.Output()
	if err == nil {
		output = append(output, stagedOutput...)
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// isModelFile returns true if the path is a model.md file (not a probe).
func isModelFile(path string) bool {
	normalized := NormalizePath(path)
	// Must be in .kb/models/*/model.md pattern
	if !strings.Contains(normalized, ".kb/models/") {
		return false
	}
	// Must end with model.md
	return strings.HasSuffix(normalized, "/model.md")
}

// extractModelNameFromPath extracts the model name from a .kb/models/{name}/model.md path.
func extractModelNameFromPath(path string) string {
	normalized := NormalizePath(path)
	// Pattern: .kb/models/{name}/model.md
	parts := strings.Split(normalized, "/")
	for i, part := range parts {
		if part == "models" && i+2 < len(parts) && parts[i+2] == "model.md" {
			return parts[i+1]
		}
	}
	return ""
}

// FormatProbeModelMergeFailure formats the probe-model merge gate failure for display.
func FormatProbeModelMergeFailure(result *ProbeModelMergeResult) string {
	if result == nil || result.Passed {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n--- Probe-to-Model Merge Gate ---\n")
	for _, p := range result.UnmergedProbes {
		icon := verdictIcon(p.Verdict)
		sb.WriteString(fmt.Sprintf("%s  Probe: %s\n", icon, filepath.Base(p.ProbePath)))
		sb.WriteString(fmt.Sprintf("   Model: %s (not updated)\n", p.ModelName))
		sb.WriteString(fmt.Sprintf("   Verdict: %s — %s\n", p.Verdict, p.Details))
		sb.WriteString("\n")
	}
	sb.WriteString("Fix: Merge probe findings into .kb/models/{model}/model.md and commit\n")
	sb.WriteString("---------------------------------\n")
	return sb.String()
}

