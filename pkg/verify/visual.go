// Package verify provides verification helpers for agent completion.
package verify

import (
	"os/exec"
	"regexp"
	"strings"
)

// Skills that require visual verification when modifying web/ files.
// Only skills that are explicitly about UI work should require visual verification.
// Non-UI skills (architects, investigations, debugging) may incidentally modify web/
// files as part of broader work - these shouldn't require visual verification.
var skillsRequiringVisualVerification = map[string]bool{
	"feature-impl": true, // UI features need visual verification
	// Note: We don't include all possible UI skills - the default is permissive.
	// If a skill is not in this list and modifies web/ files, we assume it's incidental.
}

// Skills that are explicitly excluded from visual verification requirements.
// These skills are known to work on web/ files incidentally (not as primary UI work).
var skillsExcludedFromVisualVerification = map[string]bool{
	"architect":            true, // Design work may touch web/ files
	"investigation":        true, // Research may examine/modify web/ files
	"systematic-debugging": true, // Debugging may touch web/ files
	"research":             true, // Research doesn't do UI work
	"codebase-audit":       true, // Audits may touch any files
	"reliability-testing":  true, // Testing may touch any files
	"design-session":       true, // Design sessions don't do UI implementation
	"issue-creation":       true, // Issue creation doesn't do UI work
	"writing-skills":       true, // Skill writing may touch web/ examples
}

// IsSkillRequiringVisualVerification determines if a skill requires visual verification
// for web/ file changes.
//
// The logic is:
// 1. If skill is explicitly excluded (architect, investigation, etc.) -> false
// 2. If skill is explicitly included (feature-impl) -> true
// 3. If skill is unknown -> false (permissive default to avoid false positives)
//
// This approach prevents false positives from architects/investigations that modify
// web/ files incidentally as part of broader work.
func IsSkillRequiringVisualVerification(skillName string) bool {
	// Empty skill name means we couldn't determine the skill - be permissive
	if skillName == "" {
		return false
	}

	// Normalize skill name to lowercase for comparison
	skillName = strings.ToLower(skillName)

	// Check explicit exclusions first
	if skillsExcludedFromVisualVerification[skillName] {
		return false
	}

	// Check explicit inclusions
	if skillsRequiringVisualVerification[skillName] {
		return true
	}

	// Unknown skill - be permissive to avoid false positives
	return false
}

// VisualVerificationResult represents the result of checking for visual verification evidence.
type VisualVerificationResult struct {
	Passed          bool     // Whether verification passed
	HasWebChanges   bool     // Whether web/ files were changed
	HasEvidence     bool     // Whether visual verification evidence was found
	HasHumanApproval bool    // Whether human/orchestrator explicitly approved
	NeedsApproval   bool     // Whether human approval is required but missing
	Errors          []string // Error messages
	Warnings        []string // Warning messages
	Evidence        []string // Evidence found (for debugging)
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

// humanApprovalPatterns defines patterns that indicate explicit human/orchestrator approval.
// These patterns must come from a human orchestrator, not from the agent itself.
// The patterns are designed to be unlikely to be accidentally used by agents.
var humanApprovalPatterns = []*regexp.Regexp{
	// Explicit approval markers (orchestrator uses these)
	regexp.MustCompile(`(?i)✅\s*APPROVED`),
	regexp.MustCompile(`(?i)UI\s*APPROVED`),
	regexp.MustCompile(`(?i)VISUAL\s*APPROVED`),
	regexp.MustCompile(`(?i)human_approved:\s*true`),
	regexp.MustCompile(`(?i)orchestrator_approved:\s*true`),
	// "I approve" style (first person indicates human)
	regexp.MustCompile(`(?i)I\s+approve\s+(the\s+)?(UI|visual|changes)`),
	regexp.MustCompile(`(?i)LGTM.*UI`),
	regexp.MustCompile(`(?i)UI.*LGTM`),
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

// HasHumanApproval checks beads comments for explicit human/orchestrator approval.
// Returns true if any comment contains an explicit approval marker.
// These markers are designed to be used by human orchestrators, not agents.
func HasHumanApproval(comments []Comment) (bool, []string) {
	var approvals []string

	for _, comment := range comments {
		for _, pattern := range humanApprovalPatterns {
			if pattern.MatchString(comment.Text) {
				matches := pattern.FindString(comment.Text)
				if matches != "" {
					approvals = append(approvals, matches)
				}
			}
		}
	}

	return len(approvals) > 0, approvals
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
// This is a gate that blocks completion if web/ files were modified without visual verification evidence
// AND explicit human approval.
//
// The verification passes if:
// 1. No web/ files were modified in recent commits, OR
// 2. The skill is not a UI-focused skill (architect, investigation, debugging, etc.), OR
// 3. Visual verification evidence is found AND human approval is present
//
// This skill-aware approach prevents false positives from non-UI skills that incidentally
// modify web/ files as part of broader work. Only feature-impl (and similar UI-focused skills)
// require visual verification for web/ changes.
//
// Evidence includes:
// - Screenshots mentioned (screenshot, captured image)
// - Visual verification mentioned (visually verified, UI verified)
// - Browser testing mentioned (playwright, browser_take_screenshot, tested in browser)
//
// Human Approval includes:
// - ✅ APPROVED marker
// - UI APPROVED / VISUAL APPROVED
// - human_approved: true
// - orchestrator_approved: true
// - "I approve the UI/visual/changes"
func VerifyVisualVerification(beadsID, workspacePath, projectDir string) VisualVerificationResult {
	result := VisualVerificationResult{Passed: true}

	// Check if web/ files were modified
	result.HasWebChanges = HasWebChangesInRecentCommits(projectDir)

	// No web changes = no verification needed
	if !result.HasWebChanges {
		return result
	}

	// Check skill type - only UI-focused skills require visual verification
	skillName, _ := ExtractSkillNameFromSpawnContext(workspacePath)
	if !IsSkillRequiringVisualVerification(skillName) {
		// Non-UI skill modifying web/ files - this is incidental, not UI work
		// Skip visual verification requirement
		result.Warnings = append(result.Warnings,
			"web/ files modified by non-UI skill ("+skillName+") - visual verification not required")
		return result
	}

	// UI-focused skill (feature-impl) - need visual verification evidence AND human approval

	// Check beads comments for evidence and approval
	comments, err := GetComments(beadsID)
	if err != nil {
		result.Warnings = append(result.Warnings, "failed to get beads comments: "+err.Error())
	} else {
		// Check for visual verification evidence
		hasEvidence, evidence := HasVisualVerificationEvidence(comments)
		if hasEvidence {
			result.HasEvidence = true
			result.Evidence = append(result.Evidence, evidence...)
		}

		// Check for human approval
		hasApproval, approvals := HasHumanApproval(comments)
		if hasApproval {
			result.HasHumanApproval = true
			result.Evidence = append(result.Evidence, approvals...)
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

	// Determine what's missing
	if !result.HasEvidence {
		result.Passed = false
		result.Errors = append(result.Errors,
			"web/ files modified but no visual verification evidence found",
			"Agent must capture screenshot or mention visual verification in beads comment",
			"Example: bd comment <id> \"Visual verification: screenshot captured showing [description]\"",
		)
	} else if !result.HasHumanApproval {
		// Evidence exists but needs human approval
		result.Passed = false
		result.NeedsApproval = true
		result.Errors = append(result.Errors,
			"web/ files modified - visual evidence found but requires human approval",
			"Use: orch complete <id> --approve   OR",
			"Add approval comment: bd comment <id> \"✅ APPROVED\"",
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
