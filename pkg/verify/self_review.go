// Package verify provides verification helpers for agent completion.
//
// self_review.go implements automated self-review checks that run at orch complete time.
// These checks replace behavioral instructions that agents previously carried in their
// SPAWN_CONTEXT, reducing cross-cutting token weight across all worker skills.
//
// Automatable checks (moved here from skill self-review phases):
//   - Debug statements in changed files (console.log, fmt.Print, debugger, etc.)
//   - Commit message format (conventional commits)
//   - Placeholder/demo data patterns in changed production files
//   - Orphaned new Go files (added but not imported anywhere)
package verify

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// SelfReviewResult represents the result of automated self-review checks.
type SelfReviewResult struct {
	Passed   bool     // Whether all checks passed
	Errors   []string // Blocking errors
	Warnings []string // Non-blocking warnings
}

// SelfReviewCheckResult represents the result of a single self-review check.
type SelfReviewCheckResult struct {
	Name     string   // Check name (e.g., "debug_statements")
	Passed   bool     // Whether this check passed
	Findings []string // Specific findings (file:line or description)
}

// VerifySelfReviewForCompletion runs automated self-review checks on changed files.
// Returns nil if no checks are applicable (no recent changes).
func VerifySelfReviewForCompletion(workspacePath, projectDir string) *SelfReviewResult {
	if projectDir == "" {
		return nil
	}

	changedFiles := getRecentlyChangedFiles(projectDir)
	if len(changedFiles) == 0 {
		return nil
	}

	result := &SelfReviewResult{Passed: true}

	// Run each check and collect results
	checks := []SelfReviewCheckResult{
		checkDebugStatements(projectDir, changedFiles),
		checkCommitMessages(projectDir),
		checkPlaceholderData(projectDir, changedFiles),
		checkOrphanedGoFiles(projectDir, changedFiles),
	}

	for _, check := range checks {
		if !check.Passed {
			result.Passed = false
			for _, finding := range check.Findings {
				result.Errors = append(result.Errors,
					fmt.Sprintf("self-review/%s: %s", check.Name, finding))
			}
		}
	}

	return result
}

// getRecentlyChangedFiles returns files changed in recent commits (last 5).
func getRecentlyChangedFiles(projectDir string) []string {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// Fewer commits available — try HEAD~1
		cmd = exec.Command("git", "diff", "--name-only", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return nil
		}
	}

	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

// debugPatterns are patterns that indicate leftover debug statements.
// Each entry has a pattern and the file extensions it applies to.
var debugPatterns = []struct {
	Pattern    *regexp.Regexp
	Extensions []string // Empty means all files
	Label      string   // Human-readable description
}{
	{
		Pattern:    regexp.MustCompile(`\bconsole\.(log|debug|warn|error|info|trace)\b`),
		Extensions: []string{".js", ".jsx", ".ts", ".tsx", ".svelte"},
		Label:      "console.log/debug statement",
	},
	{
		Pattern:    regexp.MustCompile(`\bdebugger\b`),
		Extensions: []string{".js", ".jsx", ".ts", ".tsx", ".svelte"},
		Label:      "debugger statement",
	},
	{
		Pattern:    regexp.MustCompile(`\bfmt\.Print(ln|f)?\b`),
		Extensions: []string{".go"},
		Label:      "fmt.Print debug statement",
	},
	{
		Pattern:    regexp.MustCompile(`\bprint\s*\(`),
		Extensions: []string{".py"},
		Label:      "print() debug statement",
	},
	{
		Pattern:    regexp.MustCompile(`\bpdb\.set_trace\b`),
		Extensions: []string{".py"},
		Label:      "pdb.set_trace() debugger",
	},
}

// checkDebugStatements scans changed production files for leftover debug statements.
// Skips test files and known non-production paths.
func checkDebugStatements(projectDir string, changedFiles []string) SelfReviewCheckResult {
	result := SelfReviewCheckResult{Name: "debug_statements", Passed: true}

	prodFiles := filterProductionFiles(changedFiles)
	if len(prodFiles) == 0 {
		return result
	}

	for _, file := range prodFiles {
		ext := filepath.Ext(file)
		for _, dp := range debugPatterns {
			if !matchesExtension(ext, dp.Extensions) {
				continue
			}

			// Use git show to get the file content at HEAD
			findings := grepFileAtHEAD(projectDir, file, dp.Pattern)
			for _, lineNum := range findings {
				result.Passed = false
				result.Findings = append(result.Findings,
					fmt.Sprintf("%s at %s:%d", dp.Label, file, lineNum))
			}
		}
	}

	return result
}

// conventionalCommitPattern matches conventional commit format: type(scope): description
// or type: description
var conventionalCommitPattern = regexp.MustCompile(
	`^(feat|fix|refactor|test|docs|chore|style|perf|ci|build|revert)(\([^)]+\))?(!)?:\s+\S`)

// wipCommitPattern matches WIP/temp/placeholder commit messages.
var wipCommitPattern = regexp.MustCompile(`(?i)^(wip|temp|tmp|fixup|squash|xxx|todo)\b`)

// checkCommitMessages verifies recent commits follow conventional format.
func checkCommitMessages(projectDir string) SelfReviewCheckResult {
	result := SelfReviewCheckResult{Name: "commit_format", Passed: true}

	cmd := exec.Command("git", "log", "--oneline", "--format=%s", "-5")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return result // Can't check, pass by default
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for WIP commits (blocking)
		if wipCommitPattern.MatchString(line) {
			result.Passed = false
			result.Findings = append(result.Findings,
				fmt.Sprintf("WIP/temp commit found: %q", truncateString(line, 60)))
			continue
		}

		// Check conventional format (warning only - not blocking)
		// Many projects don't strictly enforce this, so we only block on WIP
	}

	return result
}

// placeholderPatterns are patterns that indicate demo/placeholder data in production code.
var placeholderPatterns = []struct {
	Pattern *regexp.Regexp
	Label   string
}{
	{regexp.MustCompile(`(?i)\bjohn\s+doe\b`), "placeholder name 'John Doe'"},
	{regexp.MustCompile(`(?i)\bjane\s+(doe|smith)\b`), "placeholder name"},
	{regexp.MustCompile(`(?i)\btest\s+user\b`), "placeholder 'Test User'"},
	{regexp.MustCompile(`(?i)\blorem\s+ipsum\b`), "lorem ipsum placeholder text"},
	{regexp.MustCompile(`\btest@example\.com\b`), "placeholder email"},
	{regexp.MustCompile(`\b555-\d{4}\b`), "placeholder phone number"},
}

// checkPlaceholderData scans changed production files for demo/placeholder data.
func checkPlaceholderData(projectDir string, changedFiles []string) SelfReviewCheckResult {
	result := SelfReviewCheckResult{Name: "placeholder_data", Passed: true}

	prodFiles := filterProductionFiles(changedFiles)
	if len(prodFiles) == 0 {
		return result
	}

	for _, file := range prodFiles {
		for _, pp := range placeholderPatterns {
			findings := grepFileAtHEAD(projectDir, file, pp.Pattern)
			for _, lineNum := range findings {
				result.Passed = false
				result.Findings = append(result.Findings,
					fmt.Sprintf("%s at %s:%d", pp.Label, file, lineNum))
			}
		}
	}

	return result
}

// checkOrphanedGoFiles checks if newly added .go files are imported somewhere.
// Only checks Go files because Go has a reliable import mechanism to verify.
func checkOrphanedGoFiles(projectDir string, changedFiles []string) SelfReviewCheckResult {
	result := SelfReviewCheckResult{Name: "orphaned_files", Passed: true}

	// Get newly added files (not just modified)
	newFiles := getNewlyAddedFiles(projectDir)
	if len(newFiles) == 0 {
		return result
	}

	// Filter to new Go files (excluding tests and main packages)
	var newGoFiles []string
	for _, f := range newFiles {
		if strings.HasSuffix(f, ".go") &&
			!strings.HasSuffix(f, "_test.go") &&
			!strings.Contains(f, "/testdata/") {
			newGoFiles = append(newGoFiles, f)
		}
	}

	if len(newGoFiles) == 0 {
		return result
	}

	for _, file := range newGoFiles {
		// Extract package directory to determine import path
		dir := filepath.Dir(file)
		if dir == "." || dir == "" {
			continue // Root-level files, can't easily check
		}

		// Skip cmd/ directories (main packages aren't imported)
		if strings.HasPrefix(dir, "cmd/") || strings.HasPrefix(dir, "cmd\\") {
			continue
		}

		// Check if any other Go file imports this package
		// The import path would contain the directory
		pkgName := filepath.Base(dir)
		if !isPackageImported(projectDir, pkgName, dir) {
			result.Passed = false
			result.Findings = append(result.Findings,
				fmt.Sprintf("new file %s in package %q — package not imported anywhere", file, pkgName))
		}
	}

	return result
}

// getNewlyAddedFiles returns files that were added (not just modified) in recent commits.
func getNewlyAddedFiles(projectDir string) []string {
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=A", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("git", "diff", "--name-only", "--diff-filter=A", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return nil
		}
	}

	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

// isPackageImported checks if a Go package is imported anywhere in the project.
// Uses grep for the package directory name in import statements.
func isPackageImported(projectDir, pkgName, pkgDir string) bool {
	// Search for import of this package name/path
	cmd := exec.Command("grep", "-r", "--include=*.go", "-l", fmt.Sprintf(`"%s"`, pkgDir))
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		return true
	}

	// Also try searching by package base name (handles partial paths)
	cmd = exec.Command("grep", "-r", "--include=*.go", "-l", fmt.Sprintf(`".*/%s"`, pkgName))
	cmd.Dir = projectDir
	output, err = cmd.Output()
	return err == nil && len(strings.TrimSpace(string(output))) > 0
}

// filterProductionFiles filters out test files, fixtures, and non-production paths.
func filterProductionFiles(files []string) []string {
	var prod []string
	for _, f := range files {
		if isProductionFile(f) {
			prod = append(prod, f)
		}
	}
	return prod
}

// isProductionFile returns true if the file is a production file (not test/fixture/doc).
func isProductionFile(path string) bool {
	// Skip test files
	if strings.HasSuffix(path, "_test.go") ||
		strings.HasSuffix(path, ".test.ts") ||
		strings.HasSuffix(path, ".test.tsx") ||
		strings.HasSuffix(path, ".test.js") ||
		strings.HasSuffix(path, ".test.jsx") ||
		strings.HasSuffix(path, ".spec.ts") ||
		strings.HasSuffix(path, ".spec.tsx") ||
		strings.HasSuffix(path, ".spec.js") ||
		strings.HasSuffix(path, ".spec.jsx") {
		return false
	}

	// Skip test directories
	lowerPath := strings.ToLower(path)
	testDirs := []string{
		"/test/", "/tests/", "/__tests__/", "/testdata/",
		"/fixtures/", "/mocks/", "/__mocks__/",
		"/stories/", "/.storybook/",
	}
	for _, dir := range testDirs {
		if strings.Contains(lowerPath, dir) {
			return false
		}
	}

	// Skip non-code files
	nonCodeExts := []string{".md", ".txt", ".yaml", ".yml", ".json", ".toml", ".xml",
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
		".lock", ".sum"}
	ext := filepath.Ext(path)
	for _, ncExt := range nonCodeExts {
		if ext == ncExt {
			return false
		}
	}

	// Skip skill files and docs
	if strings.Contains(path, "skills/") && strings.HasSuffix(path, ".md") {
		return false
	}
	if strings.HasPrefix(path, ".kb/") || strings.HasPrefix(path, ".beads/") {
		return false
	}

	return true
}

// matchesExtension checks if ext matches any of the given extensions.
// Empty extensions list means match all files.
func matchesExtension(ext string, extensions []string) bool {
	if len(extensions) == 0 {
		return true
	}
	for _, e := range extensions {
		if ext == e {
			return true
		}
	}
	return false
}

// grepFileAtHEAD searches a file at HEAD for a pattern and returns matching line numbers.
func grepFileAtHEAD(projectDir, file string, pattern *regexp.Regexp) []int {
	cmd := exec.Command("git", "show", fmt.Sprintf("HEAD:%s", file))
	cmd.Dir = projectDir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil
	}

	var lineNums []int
	for i, line := range strings.Split(out.String(), "\n") {
		if pattern.MatchString(line) {
			lineNums = append(lineNums, i+1)
		}
	}
	return lineNums
}

// truncateString truncates a string to maxLen, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
