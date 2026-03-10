package verify

import (
	"fmt"
	"os/exec"
	"strings"
)

// Pre-commit specific thresholds (from harness-engineering model, Layer 0).
// These are warning-only — they don't block the commit.
const (
	PrecommitWarningThreshold800 = 800 // >800 lines with significant growth → warn
	PrecommitWarningDelta800     = 30  // ≥30 net lines added triggers warning at 800+
	PrecommitWarningThreshold600 = 600 // >600 lines with significant growth → warn
	PrecommitWarningDelta600     = 50  // ≥50 net lines added triggers warning at 600+
)

// StagedAccretionResult represents the result of checking staged files for accretion.
type StagedAccretionResult struct {
	Passed       bool              // Whether all staged files are within threshold (false = hard block)
	BlockedFiles []StagedFileInfo  // Files that exceed the critical threshold (1500)
	WarningFiles []StagedFileInfo  // Files that exceed warning thresholds (800/600)
}

// StagedFileInfo contains info about a staged file that exceeded a threshold.
type StagedFileInfo struct {
	Path      string // File path relative to repo root
	Lines     int    // Line count in the staged version
	NetDelta  int    // Net lines added (staged - HEAD), 0 if unknown
	Threshold int    // Which threshold was exceeded
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
		stagedLines, err := countStagedFileLines(projectDir, file)
		if err != nil {
			// Can't count — skip this file
			continue
		}

		// Calculate net delta (staged - HEAD)
		headLines, _ := countHeadFileLines(projectDir, file) // 0 for new files
		netDelta := stagedLines - headLines

		// Hard block: >1500 lines
		if stagedLines > AccretionCriticalThreshold {
			result.Passed = false
			result.BlockedFiles = append(result.BlockedFiles, StagedFileInfo{
				Path:      file,
				Lines:     stagedLines,
				NetDelta:  netDelta,
				Threshold: AccretionCriticalThreshold,
			})
			continue
		}

		// Warning: >800 lines with ≥30 net lines added
		if stagedLines > PrecommitWarningThreshold800 && netDelta >= PrecommitWarningDelta800 {
			result.WarningFiles = append(result.WarningFiles, StagedFileInfo{
				Path:      file,
				Lines:     stagedLines,
				NetDelta:  netDelta,
				Threshold: PrecommitWarningThreshold800,
			})
			continue
		}

		// Warning: >600 lines with ≥50 net lines added
		if stagedLines > PrecommitWarningThreshold600 && netDelta >= PrecommitWarningDelta600 {
			result.WarningFiles = append(result.WarningFiles, StagedFileInfo{
				Path:      file,
				Lines:     stagedLines,
				NetDelta:  netDelta,
				Threshold: PrecommitWarningThreshold600,
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

// FormatStagedAccretionWarnings formats warning files into a human-readable warning message.
func FormatStagedAccretionWarnings(result *StagedAccretionResult) string {
	if result == nil || len(result.WarningFiles) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("WARNING: accretion approaching — files growing toward thresholds:\n")
	for _, f := range result.WarningFiles {
		fmt.Fprintf(&b, "  %s (%d lines, +%d net, threshold %d)\n", f.Path, f.Lines, f.NetDelta, f.Threshold)
	}
	b.WriteString("\nConsider extraction before further growth. See: orch hotspot")
	return b.String()
}

// countHeadFileLines counts lines in the HEAD version of a file.
// Returns 0 for new files (not in HEAD).
func countHeadFileLines(projectDir, file string) (int, error) {
	cmd := exec.Command("git", "show", "HEAD:"+file)
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// File doesn't exist in HEAD (new file) — return 0
		return 0, nil
	}

	content := string(output)
	if content == "" {
		return 0, nil
	}

	lines := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		lines++
	}
	return lines, nil
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
