// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// MaxPatchesBeforeArchitectReview is the threshold after which architect review is required.
	// After the 3rd patch to a decision, the 4th patch is blocked until architect reviews.
	MaxPatchesBeforeArchitectReview = 3
)

// VerifyDecisionPatchCount checks if investigations are accumulating patches to the same decision.
// After N patches (default: 3), gates completion and requires architect review before more patches.
// This prevents "launchd-style" patch accumulation where tactical fixes pile up without strategic review.
//
// Detection logic:
// 1. Reads SYNTHESIS.md to find decision references (e.g., ".kb/decisions/2026-01-09-foo.md")
// 2. For each decision referenced, counts existing investigations mentioning that decision
// 3. If count >= MaxPatchesBeforeArchitectReview, blocks completion with architect review prompt
//
// Returns nil if no decision patches detected or count below threshold.
// Returns VerificationResult with Passed=false if threshold exceeded.
func VerifyDecisionPatchCount(workspacePath, projectDir string) *VerificationResult {
	if workspacePath == "" || projectDir == "" {
		return nil
	}

	// Read SYNTHESIS.md to find decision references
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	synthesisBytes, err := os.ReadFile(synthesisPath)
	if err != nil {
		// If no SYNTHESIS.md, can't detect decision patches - skip check
		return nil
	}
	synthesisContent := string(synthesisBytes)

	// Find decision file references in SYNTHESIS.md
	// Pattern: .kb/decisions/YYYY-MM-DD-*.md or full paths
	decisionRefs := findDecisionReferences(synthesisContent)
	if len(decisionRefs) == 0 {
		// No decisions referenced, nothing to check
		return nil
	}

	// For each decision, count patches in .kb/investigations/
	kbDir := filepath.Join(projectDir, ".kb")
	investigationsDir := filepath.Join(kbDir, "investigations")

	result := &VerificationResult{
		Passed:      true,
		Errors:      []string{},
		Warnings:    []string{},
		GatesFailed: []string{},
	}

	for _, decisionPath := range decisionRefs {
		// Normalize decision path (handle both relative and absolute paths)
		normalizedPath := normalizeDecisionPath(decisionPath, projectDir)

		// Count existing patches to this decision
		patchCount, err := countPatchesToDecision(investigationsDir, normalizedPath)
		if err != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Warning: Failed to count patches to decision %s: %v", normalizedPath, err))
			continue
		}

		// Check if threshold exceeded (>=3 existing patches means this would be the 4th)
		if patchCount >= MaxPatchesBeforeArchitectReview {
			result.Passed = false
			result.Errors = append(result.Errors,
				fmt.Sprintf(
					"⚠️ Decision patch limit reached: %d patches already address %s\n"+
						"   After %d tactical fixes, strategic review is required.\n"+
						"   Action required: Spawn architect to review decision before allowing more patches:\n"+
						"   → orch spawn architect \"Review %s after %d patches\"",
					patchCount,
					filepath.Base(normalizedPath),
					MaxPatchesBeforeArchitectReview,
					filepath.Base(normalizedPath),
					patchCount,
				))
			result.GatesFailed = append(result.GatesFailed, "decision_patch_limit")
		} else if patchCount >= 1 {
			// Warning on 2nd patch and beyond (any accumulation is notable)
			patchesUntilLimit := MaxPatchesBeforeArchitectReview - patchCount
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("⚠️ Warning: %d patches to %s (%d more before architect review required)",
					patchCount, filepath.Base(normalizedPath), patchesUntilLimit))
		}
	}

	// Return nil if no errors or warnings (nothing to report)
	if result.Passed && len(result.Warnings) == 0 {
		return nil
	}

	return result
}

// findDecisionReferences extracts decision file paths from SYNTHESIS.md content.
// Looks for patterns like:
//   - .kb/decisions/YYYY-MM-DD-*.md
//   - /full/path/.kb/decisions/YYYY-MM-DD-*.md
func findDecisionReferences(content string) []string {
	// Regex to match decision file paths
	// Matches: .kb/decisions/YYYY-MM-DD-*.md or /path/.kb/decisions/YYYY-MM-DD-*.md
	decisionPattern := regexp.MustCompile(`(?:^|[\s(])([^\s()]*\.kb/decisions/\d{4}-\d{2}-\d{2}-[a-z0-9-]+\.md)`)
	matches := decisionPattern.FindAllStringSubmatch(content, -1)

	seen := make(map[string]bool)
	var refs []string
	for _, match := range matches {
		if len(match) > 1 {
			path := match[1]
			if !seen[path] {
				seen[path] = true
				refs = append(refs, path)
			}
		}
	}
	return refs
}

// normalizeDecisionPath converts decision paths to consistent format for comparison.
// Handles:
//   - Relative paths: .kb/decisions/foo.md -> absolute path
//   - Absolute paths: /full/path/.kb/decisions/foo.md -> keep as-is
//   - Extracts just the filename if we can't resolve to full path
func normalizeDecisionPath(path, projectDir string) string {
	// If already absolute, use as-is
	if filepath.IsAbs(path) {
		return path
	}

	// If relative, join with project dir
	fullPath := filepath.Join(projectDir, path)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath
	}

	// Fallback: use just the filename for matching
	return filepath.Base(path)
}

// countPatchesToDecision counts how many investigations in .kb/investigations/
// reference the given decision file.
// Uses ripgrep (rg) for fast searching if available, otherwise falls back to grep.
func countPatchesToDecision(investigationsDir, decisionPath string) (int, error) {
	if _, err := os.Stat(investigationsDir); os.IsNotExist(err) {
		return 0, nil
	}

	// Extract decision filename for searching
	decisionFile := filepath.Base(decisionPath)

	// Try ripgrep first (faster)
	cmd := exec.Command("rg", "-l", decisionFile, investigationsDir)
	output, err := cmd.Output()
	if err != nil {
		// Fall back to grep if rg not available
		cmd = exec.Command("grep", "-rl", decisionFile, investigationsDir)
		output, err = cmd.Output()
		if err != nil {
			// If both fail, return 0 (can't count, but don't block)
			return 0, nil
		}
	}

	// Count lines in output (each line is one matching file)
	if len(output) == 0 {
		return 0, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	return len(lines), nil
}

// DecisionWithoutBlocks represents a decision that was referenced but lacks blocks: frontmatter.
type DecisionWithoutBlocks struct {
	Path     string // Path to the decision file
	Filename string // Just the filename (for display)
}

// FindDecisionsWithoutBlocksFrontmatter analyzes SYNTHESIS.md and returns decisions
// that are referenced but lack blocks: frontmatter.
// This surfaces opportunities to add blocks: keywords during completion review.
func FindDecisionsWithoutBlocksFrontmatter(workspacePath, projectDir string) ([]DecisionWithoutBlocks, error) {
	if workspacePath == "" || projectDir == "" {
		return nil, nil
	}

	// Read SYNTHESIS.md
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	synthesisBytes, err := os.ReadFile(synthesisPath)
	if err != nil {
		// No SYNTHESIS.md, nothing to check
		return nil, nil
	}

	// Find decision references
	decisionRefs := findDecisionReferences(string(synthesisBytes))
	if len(decisionRefs) == 0 {
		return nil, nil
	}

	var results []DecisionWithoutBlocks

	// Check each referenced decision
	for _, decisionPath := range decisionRefs {
		normalizedPath := normalizeDecisionPath(decisionPath, projectDir)

		// Check if decision file exists
		var fullPath string
		if filepath.IsAbs(normalizedPath) {
			fullPath = normalizedPath
		} else {
			// Try to find it in .kb/decisions/
			fullPath = filepath.Join(projectDir, ".kb", "decisions", normalizedPath)
		}

		// Check if decision has blocks: frontmatter
		hasBlocks, err := hasBlocksFrontmatter(fullPath)
		if err != nil {
			// If we can't read the file, skip it (might not exist yet)
			continue
		}

		if !hasBlocks {
			results = append(results, DecisionWithoutBlocks{
				Path:     fullPath,
				Filename: filepath.Base(fullPath),
			})
		}
	}

	return results, nil
}

// hasBlocksFrontmatter checks if a decision file has blocks: frontmatter.
// Returns true if blocks: is present, false otherwise.
func hasBlocksFrontmatter(decisionPath string) (bool, error) {
	content, err := os.ReadFile(decisionPath)
	if err != nil {
		return false, err
	}

	// Look for YAML frontmatter with blocks: field
	// Pattern: ---\n...blocks:\n...---
	lines := strings.Split(string(content), "\n")

	inFrontmatter := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for frontmatter start
		if i == 0 && trimmed == "---" {
			inFrontmatter = true
			continue
		}

		// Check for frontmatter end
		if inFrontmatter && trimmed == "---" {
			// Reached end of frontmatter without finding blocks:
			return false, nil
		}

		// Check for blocks: field
		if inFrontmatter && strings.HasPrefix(trimmed, "blocks:") {
			return true, nil
		}
	}

	// No frontmatter or no blocks: field found
	return false, nil
}
