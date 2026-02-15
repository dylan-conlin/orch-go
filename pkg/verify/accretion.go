// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// AccretionResult represents the result of verifying file size accretion.
type AccretionResult struct {
	Passed   bool     // Whether verification passed (no hard gate violations)
	Errors   []string // Error messages (blocking) - files >1500 lines with >50 line additions
	Warnings []string // Warning messages (non-blocking) - files >800 lines with >50 line additions

	// Detailed file information
	FilesChecked    []FileAccretionInfo // All files that were checked
	RiskFiles       []FileAccretionInfo // Files that triggered warnings or errors
	ExtractionDelta bool                // True if net delta is negative (extraction work)
}

// FileAccretionInfo contains accretion details for a single file.
type FileAccretionInfo struct {
	Path         string // File path
	CurrentLines int    // Current total line count
	LinesAdded   int    // Lines added in this change
	LinesRemoved int    // Lines removed in this change
	NetDelta     int    // Net change (added - removed)
	IsRisk       bool   // Whether this file exceeds thresholds
	Severity     string // "critical" (>1500), "warning" (>800), or "ok"
}

// Accretion thresholds (matches hotspot analysis in cmd/orch/hotspot.go).
const (
	AccretionWarningThreshold  = 800  // Files >800 lines trigger warnings
	AccretionCriticalThreshold = 1500 // Files >1500 lines trigger errors
	AccretionDeltaThreshold    = 50   // Net additions >50 lines trigger checks
)

// VerifyAccretionForCompletion checks if the agent added significant lines to already-large files.
// This gate prevents "accretion gravity" - the pattern where 25 agents each add small features
// to the same file, growing it from 800 to 2,000+ lines.
//
// Gate logic:
// - Files >1,500 lines with +50 net lines → ERROR (blocks completion)
// - Files >800 lines with +50 net lines → WARNING (non-blocking)
// - Net negative delta (extraction work) → PASS regardless of file size
// - Skip for orchestrator tier (orchestrators don't write code)
//
// Returns nil if verification cannot run (missing workspace/projectDir).
func VerifyAccretionForCompletion(workspacePath, projectDir string) *AccretionResult {
	if workspacePath == "" || projectDir == "" {
		return nil
	}

	result := &AccretionResult{
		Passed: true,
	}

	// Get list of changed files with line counts from git diff --numstat
	changes, err := getGitDiffWithLineCounts(projectDir)
	if err != nil {
		// If we can't get git diff, we can't verify - treat as pass
		// (Don't block completion on git errors)
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("could not verify accretion (git diff failed): %v", err))
		return result
	}

	if len(changes) == 0 {
		// No changes - nothing to verify
		return result
	}

	// Calculate total net delta across all files
	totalNetDelta := 0
	for _, change := range changes {
		totalNetDelta += change.NetDelta
	}

	// If net delta is negative, this is extraction work - pass regardless of file sizes
	if totalNetDelta < 0 {
		result.ExtractionDelta = true
		result.Warnings = append(result.Warnings,
			"accretion gate auto-passed: net negative delta (extraction work)")
		return result
	}

	// Check each changed file for accretion
	for _, change := range changes {
		result.FilesChecked = append(result.FilesChecked, change)

		// Skip files that are shrinking (negative delta)
		if change.NetDelta < 0 {
			continue
		}

		// Skip files that added <50 net lines (small changes)
		if change.NetDelta < AccretionDeltaThreshold {
			continue
		}

		// Check if file is in accretion risk zone
		if change.CurrentLines > AccretionCriticalThreshold {
			// CRITICAL: >1,500 lines + adding >50 lines → hard gate (error)
			change.IsRisk = true
			change.Severity = "critical"
			result.RiskFiles = append(result.RiskFiles, change)
			result.Passed = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("CRITICAL accretion: %s is %d lines (+%d added, %d total). Files >%d lines require extraction before additions. See `orch hotspot` and `.kb/guides/code-extraction-patterns.md`",
					change.Path, change.CurrentLines-change.NetDelta, change.NetDelta, change.CurrentLines,
					AccretionCriticalThreshold))
		} else if change.CurrentLines > AccretionWarningThreshold {
			// WARNING: >800 lines + adding >50 lines → soft signal (warning)
			change.IsRisk = true
			change.Severity = "warning"
			result.RiskFiles = append(result.RiskFiles, change)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Accretion warning: %s is %d lines (+%d added, %d total). Consider extraction before further growth. See `orch hotspot`",
					change.Path, change.CurrentLines-change.NetDelta, change.NetDelta, change.CurrentLines))
		}
	}

	return result
}

// getGitDiffWithLineCounts returns a list of changed files with their line counts.
// Uses `git diff --numstat HEAD` to get added/removed counts,
// then uses `wc -l` to get current total line counts.
func getGitDiffWithLineCounts(projectDir string) ([]FileAccretionInfo, error) {
	// Run git diff --numstat HEAD to get changes
	cmd := exec.Command("git", "diff", "--numstat", "HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	var changes []FileAccretionInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse numstat format: added<TAB>removed<TAB>filepath
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		filePath := parts[2]

		// Skip if not a source file (only check code files, not generated/vendor/etc)
		if !isSourceFile(filePath) {
			continue
		}

		// Parse line counts (may be "-" for binary files)
		added := 0
		removed := 0
		if parts[0] != "-" {
			n, err := strconv.Atoi(parts[0])
			if err == nil {
				added = n
			}
		}
		if parts[1] != "-" {
			n, err := strconv.Atoi(parts[1])
			if err == nil {
				removed = n
			}
		}

		// Get current total line count for this file
		currentLines, err := getFileLineCount(projectDir, filePath)
		if err != nil {
			// If we can't count lines, skip this file
			continue
		}

		changes = append(changes, FileAccretionInfo{
			Path:         filePath,
			CurrentLines: currentLines,
			LinesAdded:   added,
			LinesRemoved: removed,
			NetDelta:     added - removed,
			IsRisk:       false,
			Severity:     "ok",
		})
	}

	return changes, nil
}

// getFileLineCount returns the number of lines in a file using wc -l.
func getFileLineCount(projectDir, filePath string) (int, error) {
	fullPath := filepath.Join(projectDir, filePath)
	cmd := exec.Command("wc", "-l", fullPath)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// Parse wc output: "  123 filepath"
	parts := strings.Fields(string(output))
	if len(parts) < 1 {
		return 0, fmt.Errorf("unexpected wc output: %s", output)
	}

	count, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("failed to parse line count: %w", err)
	}

	return count, nil
}

// isSourceFile checks if a file is a source code file that should be checked for accretion.
// Excludes: vendor/, node_modules/, generated files, binaries, etc.
func isSourceFile(path string) bool {
	// Normalize path separators for consistent matching
	normalizedPath := filepath.ToSlash(path)

	// Exclude vendor and dependencies
	// Check for these directories anywhere in the path (start, middle, or as a segment)
	if strings.Contains(normalizedPath, "vendor/") ||
		strings.Contains(normalizedPath, "node_modules/") ||
		strings.Contains(normalizedPath, "dist/") ||
		strings.Contains(normalizedPath, "build/") {
		return false
	}

	// Exclude generated files
	if strings.HasSuffix(path, ".gen.go") ||
		strings.HasSuffix(path, ".gen.ts") ||
		strings.HasSuffix(path, ".pb.go") ||
		strings.HasSuffix(path, "_gen.go") {
		return false
	}

	// Only check source code files
	ext := filepath.Ext(path)
	sourceExts := map[string]bool{
		".go":     true,
		".ts":     true,
		".tsx":    true,
		".js":     true,
		".jsx":    true,
		".py":     true,
		".rb":     true,
		".java":   true,
		".c":      true,
		".cpp":    true,
		".h":      true,
		".cs":     true,
		".svelte": true,
		".vue":    true,
	}

	return sourceExts[ext]
}
