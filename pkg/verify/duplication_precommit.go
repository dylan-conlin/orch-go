package verify

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/dupdetect"
)

// DuplicationPrecommitResult represents the result of checking staged files for duplication.
type DuplicationPrecommitResult struct {
	Passed   bool              // Always true — duplication is advisory only
	Warnings []dupdetect.DupPair // Duplicate pairs involving staged files
}

// CheckStagedDuplication scans staged Go files for functions that are near-clones
// of existing functions in the same project. This is advisory only — it warns
// but does not block the commit.
//
// Returns nil if projectDir is empty or no Go files are staged.
func CheckStagedDuplication(projectDir string) *DuplicationPrecommitResult {
	if projectDir == "" {
		return nil
	}

	result := &DuplicationPrecommitResult{Passed: true}

	// Get staged Go files (added, copied, modified — not deleted)
	cmd := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=ACM")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return result
	}

	var goFiles []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, ".go") && !strings.HasSuffix(line, "_test.go") {
			goFiles = append(goFiles, line)
		}
	}

	if len(goFiles) == 0 {
		return result
	}

	d := dupdetect.NewDetector()
	d.MinBodyLines = 10
	d.Threshold = 0.85

	pairs, err := d.CheckModifiedFilesProject(projectDir, goFiles)
	if err != nil || len(pairs) == 0 {
		return result
	}

	result.Warnings = pairs
	return result
}

// FormatStagedDuplicationWarning formats duplication results for pre-commit output.
func FormatStagedDuplicationWarning(result *DuplicationPrecommitResult) string {
	if result == nil || len(result.Warnings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("WARNING: duplication detected in staged files:\n")
	for _, pair := range result.Warnings {
		fmt.Fprintf(&sb, "  %.0f%% similar: %s (%s:%d) ↔ %s (%s:%d)\n",
			pair.Similarity*100,
			pair.FuncA.Name, pair.FuncA.File, pair.FuncA.StartLine,
			pair.FuncB.Name, pair.FuncB.File, pair.FuncB.StartLine,
		)
	}
	sb.WriteString("\nConsider extracting shared logic. Run: orch dupdetect")
	return sb.String()
}
