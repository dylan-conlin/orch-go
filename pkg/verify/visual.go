// Package verify provides verification helpers for agent completion.
package verify

import (
	"os/exec"
	"regexp"
	"strings"
)

// VisualVerificationResult represents the result of checking for visual verification evidence.
type VisualVerificationResult struct {
	Passed        bool     // Whether verification passed
	HasWebChanges bool     // Whether web/ files were changed
	HasEvidence   bool     // Whether visual verification evidence was found
	Errors        []string // Error messages
	Warnings      []string // Warning messages
	Evidence      []string // Evidence found (for debugging)
}

// visualEvidencePatterns defines patterns that indicate visual verification was performed.
// These patterns are checked against beads comments.
var visualEvidencePatterns = []*regexp.Regexp{
	// Screenshot mentions
	regexp.MustCompile(`(?i)screenshot`),
	regexp.MustCompile(`(?i)screen\s*shot`),
	regexp.MustCompile(`(?i)captured.*image`),
	regexp.MustCompile(`(?i)image.*captured`),
	// Visual verification mentions
	regexp.MustCompile(`(?i)visual\s*verif`),
	regexp.MustCompile(`(?i)visually\s*verif`),
	regexp.MustCompile(`(?i)browser\s*verif`),
	regexp.MustCompile(`(?i)ui\s*verif`),
	// Playwright/browser tool mentions
	regexp.MustCompile(`(?i)playwright`),
	regexp.MustCompile(`(?i)browser_take_screenshot`),
	regexp.MustCompile(`(?i)browser_navigate`),
	// Smoke test with UI context
	regexp.MustCompile(`(?i)smoke\s*test.*ui`),
	regexp.MustCompile(`(?i)ui.*smoke\s*test`),
	// "Verified in browser" style comments
	regexp.MustCompile(`(?i)verified.*browser`),
	regexp.MustCompile(`(?i)browser.*verified`),
	regexp.MustCompile(`(?i)checked.*browser`),
	regexp.MustCompile(`(?i)tested.*browser`),
}

// HasWebChangesInRecentCommits checks if any of the last 5 commits contain changes
// to web/ files (Svelte, TypeScript, CSS, etc.).
func HasWebChangesInRecentCommits(projectDir string) bool {
	// Get changed files from last 5 commits
	cmd := exec.Command("git", "diff", "--name-only", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// If git command fails (e.g., not enough commits), try last 1 commit
		cmd = exec.Command("git", "diff", "--name-only", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return false
		}
	}

	return hasWebChangesInFiles(string(output))
}

// hasWebChangesInFiles checks if any files in the output are web/ files.
// This is extracted for testing.
func hasWebChangesInFiles(gitOutput string) bool {
	lines := strings.Split(gitOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if IsWebFile(line) {
			return true
		}
	}
	return false
}

// IsWebFile returns true if the file path is a web-related file.
// Matches files in web/ directory with web file extensions.
func IsWebFile(filePath string) bool {
	// Must be in web/ directory
	if !strings.HasPrefix(filePath, "web/") {
		return false
	}

	// Check for web file extensions
	webExtensions := []string{
		".svelte", ".ts", ".tsx", ".js", ".jsx",
		".css", ".scss", ".html", ".vue",
	}

	for _, ext := range webExtensions {
		if strings.HasSuffix(filePath, ext) {
			return true
		}
	}

	return false
}

// HasVisualVerificationEvidence checks beads comments for evidence of visual verification.
// Returns true if any comment mentions screenshots, visual verification, or browser testing.
func HasVisualVerificationEvidence(comments []Comment) (bool, []string) {
	var evidence []string

	for _, comment := range comments {
		for _, pattern := range visualEvidencePatterns {
			if pattern.MatchString(comment.Text) {
				// Extract a snippet around the match for evidence
				matches := pattern.FindString(comment.Text)
				if matches != "" {
					evidence = append(evidence, matches)
				}
			}
		}
	}

	return len(evidence) > 0, evidence
}

// HasVisualVerificationInSynthesis checks SYNTHESIS.md for visual verification evidence.
// Looks in the Evidence section for screenshot/visual verification mentions.
func HasVisualVerificationInSynthesis(workspacePath string) (bool, []string) {
	if workspacePath == "" {
		return false, nil
	}

	synthesis, err := ParseSynthesis(workspacePath)
	if err != nil {
		return false, nil
	}

	var evidence []string

	// Check Evidence section
	for _, pattern := range visualEvidencePatterns {
		if pattern.MatchString(synthesis.Evidence) {
			matches := pattern.FindString(synthesis.Evidence)
			if matches != "" {
				evidence = append(evidence, "Evidence: "+matches)
			}
		}
	}

	// Also check TLDR
	for _, pattern := range visualEvidencePatterns {
		if pattern.MatchString(synthesis.TLDR) {
			matches := pattern.FindString(synthesis.TLDR)
			if matches != "" {
				evidence = append(evidence, "TLDR: "+matches)
			}
		}
	}

	return len(evidence) > 0, evidence
}

// VerifyVisualVerification checks if visual verification was performed for web/ changes.
// This is a gate that blocks completion if web/ files were modified without visual verification evidence.
//
// The verification passes if:
// 1. No web/ files were modified in recent commits, OR
// 2. Visual verification evidence is found in beads comments or SYNTHESIS.md
//
// Evidence includes:
// - Screenshots mentioned (screenshot, captured image)
// - Visual verification mentioned (visually verified, UI verified)
// - Browser testing mentioned (playwright, browser_take_screenshot, tested in browser)
func VerifyVisualVerification(beadsID, workspacePath, projectDir string) VisualVerificationResult {
	result := VisualVerificationResult{Passed: true}

	// Check if web/ files were modified
	result.HasWebChanges = HasWebChangesInRecentCommits(projectDir)

	// No web changes = no verification needed
	if !result.HasWebChanges {
		return result
	}

	// Web changes detected - need visual verification evidence

	// Check beads comments for evidence
	comments, err := GetComments(beadsID)
	if err != nil {
		result.Warnings = append(result.Warnings, "failed to get beads comments: "+err.Error())
	} else {
		hasEvidence, evidence := HasVisualVerificationEvidence(comments)
		if hasEvidence {
			result.HasEvidence = true
			result.Evidence = append(result.Evidence, evidence...)
		}
	}

	// Check SYNTHESIS.md for evidence
	if workspacePath != "" {
		hasEvidence, evidence := HasVisualVerificationInSynthesis(workspacePath)
		if hasEvidence {
			result.HasEvidence = true
			result.Evidence = append(result.Evidence, evidence...)
		}
	}

	// If web changes but no evidence, fail verification
	if !result.HasEvidence {
		result.Passed = false
		result.Errors = append(result.Errors,
			"web/ files modified but no visual verification evidence found",
			"Agent must capture screenshot or mention visual verification in beads comment",
			"Example: bd comment <id> \"Visual verification: screenshot captured showing [description]\"",
		)
	}

	return result
}

// VerifyVisualVerificationForCompletion is a convenience function for use in orch complete.
// Returns nil if no verification is needed (no web changes) or if verification passes.
func VerifyVisualVerificationForCompletion(beadsID, workspacePath, projectDir string) *VisualVerificationResult {
	result := VerifyVisualVerification(beadsID, workspacePath, projectDir)

	// Return nil if no web changes - no action needed
	if !result.HasWebChanges {
		return nil
	}

	return &result
}
