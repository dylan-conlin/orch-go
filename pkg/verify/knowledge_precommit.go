package verify

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// StagedKnowledgeResult represents the result of checking staged investigation files
// for model coupling.
type StagedKnowledgeResult struct {
	Passed      bool     // Whether all new investigations are coupled to a model
	OrphanFiles []string // New investigation files without model coupling
}

// modelFieldPattern matches **Model:** followed by a non-empty value.
// modelFieldPattern matches **Model:** followed by a non-empty value on the same line.
var modelFieldPattern = regexp.MustCompile(`(?m)^\*\*Model:\*\*[ \t]+\S`)

// orphanFieldPattern matches **Orphan:** acknowledged (explicit opt-out).
var orphanFieldPattern = regexp.MustCompile(`(?m)^\*\*Orphan:\*\*\s+acknowledged`)

// CheckStagedKnowledge checks newly staged .kb/investigations/ files for model coupling.
// New investigation files must contain either:
//   - **Model:** <name> field (structurally coupled to a model)
//   - **Orphan:** acknowledged field (explicit opt-out)
//   - A probe file also staged in .kb/models/*/probes/ (structural coupling via directory)
//
// Returns nil if projectDir is empty.
func CheckStagedKnowledge(projectDir string) *StagedKnowledgeResult {
	if projectDir == "" {
		return nil
	}

	result := &StagedKnowledgeResult{Passed: true}

	// Get list of NEW staged files (Added only — not modified or copied)
	cmd := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=A")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return result
	}

	stagedFiles := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Check if any probe files are also staged (indicates structural coupling)
	hasProbeStaged := false
	var newInvestigations []string

	for _, file := range stagedFiles {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		if strings.HasPrefix(file, ".kb/models/") && strings.Contains(file, "/probes/") {
			hasProbeStaged = true
		}

		if strings.HasPrefix(file, ".kb/investigations/") && strings.HasSuffix(file, ".md") {
			newInvestigations = append(newInvestigations, file)
		}
	}

	// If a probe is staged, all investigations pass (structural coupling exists)
	if hasProbeStaged {
		return result
	}

	// Check each new investigation for model coupling
	for _, file := range newInvestigations {
		content, err := readStagedFile(projectDir, file)
		if err != nil {
			continue
		}

		if modelFieldPattern.Match(content) || orphanFieldPattern.Match(content) {
			continue
		}

		result.Passed = false
		result.OrphanFiles = append(result.OrphanFiles, file)
	}

	return result
}

// FormatStagedKnowledgeError formats orphan investigation files into a human-readable error.
func FormatStagedKnowledgeError(result *StagedKnowledgeResult) string {
	if result == nil || result.Passed {
		return ""
	}

	var b strings.Builder
	b.WriteString("BLOCKED: knowledge gate — new investigations without model coupling:\n")
	for _, f := range result.OrphanFiles {
		fmt.Fprintf(&b, "  %s\n", f)
	}
	b.WriteString("\nAdd one of:\n")
	b.WriteString("  **Model:** <model-name>    (link to .kb/models/<name>)\n")
	b.WriteString("  **Orphan:** acknowledged   (explicit opt-out)\n")
	b.WriteString("\nOverride: FORCE_ORPHAN=1 git commit ...\n")
	return b.String()
}

// readStagedFile reads the staged (index) version of a file.
func readStagedFile(projectDir, file string) ([]byte, error) {
	cmd := exec.Command("git", "show", ":"+file)
	cmd.Dir = projectDir
	return cmd.Output()
}
