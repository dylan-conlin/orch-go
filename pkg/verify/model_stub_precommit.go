package verify

import (
	"fmt"
	"os/exec"
	"strings"
)

// Model template placeholder patterns that indicate an unfilled stub.
// These bracket-enclosed patterns come from kb create model's embedded template
// and should never appear in a completed model.
var modelStubPlaceholders = []string{
	"[What phenomenon or pattern does this model describe?",
	"[Concise claim statement]",
	"[How to test this claim]",
	"[Scope item 1]",
	"[Exclusion 1]",
	"[Question that further investigation could answer]",
	"[Question about model boundaries or edge cases]",
}

// StagedModelStubResult represents the result of checking staged model files
// for unfilled template placeholders.
type StagedModelStubResult struct {
	Passed    bool
	StubFiles []ModelStubInfo
}

// ModelStubInfo contains info about a staged model file that has placeholders.
type ModelStubInfo struct {
	Path         string   // File path relative to repo root
	Placeholders []string // Which placeholder patterns were found
}

// CheckStagedModelStubs checks newly staged .kb/models/*/model.md files for
// unfilled template placeholders. This prevents committing model scaffolds
// created by `kb create model` without filling in the content.
//
// Returns nil if projectDir is empty.
func CheckStagedModelStubs(projectDir string) *StagedModelStubResult {
	if projectDir == "" {
		return nil
	}

	result := &StagedModelStubResult{Passed: true}

	// Get staged files (both new and modified — a stub could be committed either way)
	cmd := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=ACM")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return result
	}

	stagedFiles := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, file := range stagedFiles {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		// Only check model.md files in .kb/models/
		if !isModelStubCandidate(file) {
			continue
		}

		content, err := readStagedFile(projectDir, file)
		if err != nil {
			continue
		}

		found := findPlaceholders(string(content))
		if len(found) > 0 {
			result.Passed = false
			result.StubFiles = append(result.StubFiles, ModelStubInfo{
				Path:         file,
				Placeholders: found,
			})
		}
	}

	return result
}

// isModelStubCandidate returns true if the path matches .kb/models/*/model.md or
// .kb/global/models/*/model.md. Unlike isModelFile in probe_model_merge.go,
// this also matches .kb/global/models/ paths.
func isModelStubCandidate(path string) bool {
	if !strings.HasSuffix(path, "/model.md") {
		return false
	}
	return strings.HasPrefix(path, ".kb/models/") || strings.HasPrefix(path, ".kb/global/models/")
}

// findPlaceholders checks content for model template placeholder patterns.
func findPlaceholders(content string) []string {
	var found []string
	for _, placeholder := range modelStubPlaceholders {
		if strings.Contains(content, placeholder) {
			found = append(found, placeholder)
		}
	}
	return found
}

// FormatStagedModelStubError formats stub model files into a human-readable error.
func FormatStagedModelStubError(result *StagedModelStubResult) string {
	if result == nil || result.Passed {
		return ""
	}

	var b strings.Builder
	b.WriteString("BLOCKED: model-stub gate — staged model files with unfilled template placeholders:\n")
	for _, stub := range result.StubFiles {
		fmt.Fprintf(&b, "  %s\n", stub.Path)
		for _, p := range stub.Placeholders {
			fmt.Fprintf(&b, "    - %s\n", p)
		}
	}
	b.WriteString("\nFill in the template placeholders before committing.\n")
	b.WriteString("Models created with `kb create model` need content before they're useful.\n")
	b.WriteString("\nOverride: FORCE_MODEL_STUB=1 git commit ...\n")
	return b.String()
}
