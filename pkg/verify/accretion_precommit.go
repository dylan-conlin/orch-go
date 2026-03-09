package verify

import (
	"fmt"
	"os/exec"
	"strings"
)

// StagedAccretionResult represents the result of checking staged files for accretion.
type StagedAccretionResult struct {
	Passed       bool                // Whether all staged files are within threshold
	BlockedFiles []StagedFileInfo    // Files that exceed the critical threshold
}

// StagedFileInfo contains info about a staged file that exceeded the threshold.
type StagedFileInfo struct {
	Path  string // File path relative to repo root
	Lines int    // Line count in the staged version
}

// CheckStagedAccretion checks staged files for accretion violations.
// Any source file staged with >1500 lines (CRITICAL threshold) blocks the commit.
// This is Layer 0 of the accretion enforcement system — catches bloat at commit time.
//
// Returns nil if projectDir is empty.
func CheckStagedAccretion(projectDir string) *StagedAccretionResult {
	if projectDir == "" {
		return nil
	}

	result := &StagedAccretionResult{Passed: true}

	// Get list of staged files (added, copied, modified — not deleted)
	cmd := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=ACM")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// Can't check — don't block
		return result
	}

	stagedFiles := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, file := range stagedFiles {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		// Only check source files (same filter as completion gate)
		if !isSourceFile(file) {
			continue
		}

		// Count lines in the staged version (index, not working tree)
		lines, err := countStagedFileLines(projectDir, file)
		if err != nil {
			// Can't count — skip this file
			continue
		}

		if lines > AccretionCriticalThreshold {
			result.Passed = false
			result.BlockedFiles = append(result.BlockedFiles, StagedFileInfo{
				Path:  file,
				Lines: lines,
			})
		}
	}

	return result
}

// FormatStagedAccretionError formats the blocked files into a human-readable error message.
func FormatStagedAccretionError(result *StagedAccretionResult) string {
	if result == nil || result.Passed {
		return ""
	}

	var b strings.Builder
	b.WriteString("BLOCKED: accretion gate — files exceed 1500-line CRITICAL threshold:\n")
	for _, f := range result.BlockedFiles {
		fmt.Fprintf(&b, "  %s (%d lines)\n", f.Path, f.Lines)
	}
	b.WriteString("\nExtract before adding more code. See: orch hotspot")
	b.WriteString("\nOverride: FORCE_ACCRETION=1 git commit ...")
	return b.String()
}

// countStagedFileLines counts lines in the staged (index) version of a file.
func countStagedFileLines(projectDir, file string) (int, error) {
	// git show :file reads the staged version from the index
	cmd := exec.Command("git", "show", ":"+file)
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("git show failed for %s: %w", file, err)
	}

	// Count newlines
	content := string(output)
	if content == "" {
		return 0, nil
	}

	lines := strings.Count(content, "\n")
	// If file doesn't end with newline, add 1 for last line
	if !strings.HasSuffix(content, "\n") {
		lines++
	}
	return lines, nil
}
