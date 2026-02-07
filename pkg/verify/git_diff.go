// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// GitDiffResult represents the result of verifying git diff against SYNTHESIS claims.
type GitDiffResult struct {
	Passed          bool     // Whether verification passed
	Errors          []string // Error messages (blocking)
	Warnings        []string // Warning messages (non-blocking)
	ClaimedFiles    []string // Files claimed in SYNTHESIS.md Delta section
	ActualFiles     []string // Files in actual git diff since spawn
	MissingFromDiff []string // Files claimed but not in diff
	ExtraInDiff     []string // Files in diff but not claimed (info only)
}

// ParseDeltaFiles extracts file paths from the SYNTHESIS.md Delta section.
// Looks for patterns like:
// - "- `path/to/file`" (backtick-quoted)
// - "- path/to/file" (unquoted on a bullet line)
// - Within "### Files Modified" or "### Files Created" subsections
func ParseDeltaFiles(synthesis *Synthesis) []string {
	if synthesis == nil || synthesis.Delta == "" {
		return nil
	}

	var files []string
	seen := make(map[string]bool)

	// Pattern to extract file paths - handles:
	// 1. Backtick-quoted paths: `path/to/file`
	// 2. Bold paths: **path/to/file**
	// 3. Paths with extensions after bullet: - path/to/file.go
	// 4. Paths in parentheses: (path/to/file.go)
	patterns := []string{
		// Backtick-quoted paths
		"`([^`]+\\.[a-zA-Z0-9]+)`",
		// Bold paths with extension
		"\\*\\*([^*]+\\.[a-zA-Z0-9]+)\\*\\*",
		// Path after bullet point with extension (handles "- filepath" pattern)
		"^\\s*[-*]\\s+([^\\s`*]+\\.[a-zA-Z0-9]+)",
		// Path in parentheses
		"\\(([^)]+\\.[a-zA-Z0-9]+)\\)",
	}

	for _, p := range patterns {
		re := regexp.MustCompile(p)
		matches := re.FindAllStringSubmatch(synthesis.Delta, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				path := strings.TrimSpace(match[1])
				// Skip if it's obviously not a file path
				if isLikelyFilePath(path) && !seen[path] {
					files = append(files, path)
					seen[path] = true
				}
			}
		}
	}

	return files
}

// knownFileExtensions contains common file extensions that indicate a real file path.
// This is used to distinguish file paths from event type names like "session.created".
var knownFileExtensions = map[string]bool{
	// Code
	"go": true, "py": true, "js": true, "ts": true, "tsx": true, "jsx": true,
	"rs": true, "rb": true, "java": true, "c": true, "cpp": true, "h": true,
	"cs": true, "php": true, "swift": true, "kt": true, "scala": true,
	"sh": true, "bash": true, "zsh": true, "fish": true,
	// Web
	"html": true, "css": true, "scss": true, "sass": true, "less": true,
	"svelte": true, "vue": true, "astro": true,
	// Data/Config
	"json": true, "yaml": true, "yml": true, "toml": true, "xml": true,
	"csv": true, "tsv": true, "ini": true, "env": true, "conf": true,
	// Documentation
	"md": true, "txt": true, "rst": true, "adoc": true,
	// Build/DevOps
	"dockerfile": true, "makefile": true, "mod": true, "sum": true,
	"lock": true, "plist": true,
	// Data stores
	"db": true, "sqlite": true, "sql": true,
	// Other
	"log": true, "tmp": true, "bak": true, "pdf": true, "png": true,
	"jpg": true, "jpeg": true, "gif": true, "svg": true, "ico": true,
}

// isLikelyFilePath checks if a string looks like a file path.
// Uses heuristics to distinguish real file paths from:
// - Event type names (session.created, agent.spawned)
// - Version numbers (v0.33.2)
// - URLs
// - Sentences
func isLikelyFilePath(s string) bool {
	if s == "" {
		return false
	}

	// Skip URLs
	if strings.Contains(s, "://") {
		return false
	}

	// Skip if it contains spaces (likely a sentence fragment)
	if strings.Contains(s, " ") {
		return false
	}

	// Skip if it contains = (likely an assignment like hasCodeChanges=true)
	if strings.Contains(s, "=") {
		return false
	}

	// Skip if it's too long (likely a description)
	if len(s) > 200 {
		return false
	}

	// Skip common non-file patterns
	skipPatterns := []string{
		"e.g.",
		"i.e.",
		"etc.",
	}
	for _, skip := range skipPatterns {
		if strings.Contains(s, skip) {
			return false
		}
	}

	// Handle dotfiles (files starting with .) - always valid file paths
	// Examples: .gitignore, .env, .beads/beads.db
	if strings.HasPrefix(s, ".") {
		// If it's just a dotfile with no further extension, it's valid (.gitignore, .env)
		// If it has a path separator, it's valid (.beads/beads.db)
		// If it has a known extension after the dot prefix, it's valid (.env.local)
		return true
	}

	// Must have a file extension for non-dotfiles
	if !strings.Contains(s, ".") {
		return false
	}

	// Extract the extension (last part after the last dot)
	lastDot := strings.LastIndex(s, ".")
	if lastDot == -1 || lastDot == len(s)-1 {
		return false
	}
	ext := strings.ToLower(s[lastDot+1:])

	// Check for version number pattern (v0.33.2, 1.2.3)
	// Version numbers typically have multiple dots with numeric segments
	if isVersionNumber(s) {
		return false
	}

	// Must have a known file extension
	return knownFileExtensions[ext]
}

// isVersionNumber checks if a string looks like a version number.
// Examples: v0.33.2, 1.2.3, v1.0.0
func isVersionNumber(s string) bool {
	// Strip leading 'v' if present
	if strings.HasPrefix(s, "v") {
		s = s[1:]
	}

	// Check if it looks like a version (all dots separate numeric or alphanumeric segments)
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return false
	}

	// Version numbers typically have 2-4 segments
	if len(parts) > 4 {
		return false
	}

	// Each segment should be primarily numeric (allow alpha suffixes like "0-beta")
	numericSegments := 0
	for _, part := range parts {
		// Check if segment starts with a digit
		if len(part) > 0 && part[0] >= '0' && part[0] <= '9' {
			numericSegments++
		}
	}

	// If all or almost all segments are numeric, it's likely a version
	return numericSegments >= len(parts)-1
}

// GetGitDiffFiles returns the list of files changed since the given time.
// Uses `git diff --name-only` to get modified files.
// If since is zero, returns files changed vs HEAD (uncommitted changes).
func GetGitDiffFiles(projectDir string, since time.Time) ([]string, error) {
	var cmd *exec.Cmd

	if since.IsZero() {
		// Get uncommitted changes
		cmd = exec.Command("git", "diff", "--name-only", "HEAD")
	} else {
		// Get all files changed since the spawn time
		// We need to find commits since spawn time and get their changed files
		sinceStr := since.Format(time.RFC3339)

		// Use git log to find all changed files since spawn time
		cmd = exec.Command("git", "log", "--name-only", "--pretty=format:", "--since="+sinceStr)
	}

	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run git: %w", err)
	}

	// Parse output - one file per line
	var files []string
	seen := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !seen[line] {
			files = append(files, line)
			seen[line] = true
		}
	}

	return files, nil
}

// NormalizePath normalizes a file path for comparison.
// Removes leading ./ or / and converts to forward slashes.
func NormalizePath(path string) string {
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "/")
	path = strings.ReplaceAll(path, "\\", "/")
	return path
}

// VerifyGitDiff compares claimed file changes in SYNTHESIS against actual git diff.
// Returns a result indicating whether claimed files exist in the actual diff.
//
// The verification:
// - Passes if all claimed files are present in the actual git diff
// - Fails if any claimed file is NOT in the git diff (false positive detection)
// - Provides warnings for extra files in diff not claimed (acceptable - agent may under-report)
func VerifyGitDiff(workspacePath, projectDir string) GitDiffResult {
	result := GitDiffResult{Passed: true}

	// Parse SYNTHESIS.md
	synthesis, err := ParseSynthesis(workspacePath)
	if err != nil {
		// No SYNTHESIS.md or parse error - skip verification
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("could not parse SYNTHESIS.md: %v - skipping git diff verification", err))
		return result
	}

	// Extract claimed files from Delta section
	result.ClaimedFiles = ParseDeltaFiles(synthesis)

	// If no files claimed, nothing to verify
	if len(result.ClaimedFiles) == 0 {
		result.Warnings = append(result.Warnings,
			"no files found in SYNTHESIS.md Delta section - skipping git diff verification")
		return result
	}

	// Get spawn time from workspace
	spawnTime := spawn.ReadSpawnTime(workspacePath)

	// Get actual git diff files
	actualFiles, err := GetGitDiffFiles(projectDir, spawnTime)
	if err != nil {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("failed to get git diff: %v - skipping verification", err))
		return result
	}
	result.ActualFiles = actualFiles

	// Build a set of actual files for O(1) lookup
	actualSet := make(map[string]bool)
	for _, f := range actualFiles {
		actualSet[NormalizePath(f)] = true
	}

	// Check each claimed file
	for _, claimed := range result.ClaimedFiles {
		normalized := NormalizePath(claimed)
		if !actualSet[normalized] {
			result.MissingFromDiff = append(result.MissingFromDiff, claimed)
		}
	}

	// Check for extra files in diff (informational)
	claimedSet := make(map[string]bool)
	for _, f := range result.ClaimedFiles {
		claimedSet[NormalizePath(f)] = true
	}
	for _, actual := range actualFiles {
		normalized := NormalizePath(actual)
		if !claimedSet[normalized] {
			result.ExtraInDiff = append(result.ExtraInDiff, actual)
		}
	}

	// Fail if claimed files are missing from diff
	if len(result.MissingFromDiff) > 0 {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("SYNTHESIS.md claims %d file(s) not in git diff:", len(result.MissingFromDiff)))
		for _, f := range result.MissingFromDiff {
			result.Errors = append(result.Errors, fmt.Sprintf("  - %s", f))
		}
		result.Errors = append(result.Errors,
			"Agent claimed to modify files that have no git changes - possible false positive")
	}

	// Add informational warning about extra files
	if len(result.ExtraInDiff) > 0 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("%d file(s) in git diff but not claimed in SYNTHESIS (under-reporting is acceptable)",
				len(result.ExtraInDiff)))
	}

	return result
}

// VerifyGitDiffForCompletion is a convenience function for use in VerifyCompletionFull.
// Returns nil if verification should be skipped (no SYNTHESIS.md, no claimed files).
// Returns blocking result if claimed files are not in git diff.
func VerifyGitDiffForCompletion(workspacePath, projectDir string) *GitDiffResult {
	// Skip if no workspace
	if workspacePath == "" || projectDir == "" {
		return nil
	}

	result := VerifyGitDiff(workspacePath, projectDir)

	// Return nil if there was nothing to verify (no claims)
	if len(result.ClaimedFiles) == 0 {
		return nil
	}

	return &result
}
