// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Pre-compiled regex patterns for check.go
var (
	regexRecommendation  = regexp.MustCompile(`(?m)\*\*Recommendation:\*\*\s*(\w+)`)
	regexNumberedPattern = regexp.MustCompile(`^\d+\.`)
)

// VerificationResult represents the result of a completion verification.
type VerificationResult struct {
	Passed   bool     // Whether all checks passed
	Errors   []string // Errors that prevent completion
	Warnings []string // Warnings that don't block completion
	Phase    PhaseStatus
}

// Synthesis represents the content of a SYNTHESIS.md file using the D.E.K.N. structure.
// D.E.K.N. = Delta (what changed), Evidence (what was observed), Knowledge (what was learned), Next (what should happen)
type Synthesis struct {
	// Header fields
	Agent    string // Agent workspace name
	Issue    string // Beads issue ID
	Duration string // Session duration
	Outcome  string // success, partial, blocked, etc.

	// Core D.E.K.N. sections
	TLDR      string // One-sentence summary
	Delta     string // What changed (files created/modified, commits)
	Evidence  string // What was observed (tests run, verification)
	Knowledge string // What was learned (artifacts, decisions, constraints)
	Next      string // What should happen (recommendation, follow-up)

	// Unexplored Questions section (for self-reflection)
	UnexploredQuestions string   // Questions that emerged during session
	AreasToExplore      []string // Areas worth exploring further
	Uncertainties       []string // What remains unclear

	// Parsed fields for easy access
	Recommendation string   // Extracted from Next section (close, continue, escalate)
	NextActions    []string // Follow-up items
}

// ParseSynthesis extracts key information from a SYNTHESIS.md file.
// Supports both the full D.E.K.N. format and simpler formats with just TLDR and Next Actions.
func ParseSynthesis(workspacePath string) (*Synthesis, error) {
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	data, err := os.ReadFile(synthesisPath)
	if err != nil {
		return nil, err
	}
	content := string(data)

	s := &Synthesis{}

	// Parse header fields
	s.Agent = extractHeaderField(content, "Agent")
	s.Issue = extractHeaderField(content, "Issue")
	s.Duration = extractHeaderField(content, "Duration")
	s.Outcome = extractHeaderField(content, "Outcome")

	// Parse TLDR section
	s.TLDR = extractSection(content, "TLDR")

	// Parse D.E.K.N. sections
	// Delta can be "## Delta" or "## Delta (What Changed)"
	s.Delta = extractSectionWithVariant(content, "Delta", "Delta (What Changed)")

	// Evidence can be "## Evidence" or "## Evidence (What Was Observed)"
	s.Evidence = extractSectionWithVariant(content, "Evidence", "Evidence (What Was Observed)")

	// Knowledge can be "## Knowledge" or "## Knowledge (What Was Learned)"
	s.Knowledge = extractSectionWithVariant(content, "Knowledge", "Knowledge (What Was Learned)")

	// Next can be "## Next", "## Next (What Should Happen)", or "## Next Actions"
	s.Next = extractSectionWithVariant(content, "Next", "Next (What Should Happen)")

	// Extract recommendation from Next section
	s.Recommendation = extractRecommendation(s.Next)

	// Parse Next Actions (follow-up items)
	s.NextActions = extractNextActions(content)

	// Parse Unexplored Questions section
	unexploredSection := extractSection(content, "Unexplored Questions")
	if unexploredSection != "" {
		s.UnexploredQuestions = unexploredSection
		s.AreasToExplore = extractBoldSubsection(unexploredSection, "Areas worth exploring further")
		s.Uncertainties = extractBoldSubsection(unexploredSection, "What remains unclear")
	}

	return s, nil
}

// extractHeaderField extracts a header field like "**Field:** value"
func extractHeaderField(content, field string) string {
	pattern := regexp.MustCompile(`(?m)\*\*` + regexp.QuoteMeta(field) + `:\*\*\s*(.+)$`)
	matches := pattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractSection extracts content from a markdown section.
// Handles sections that end at the next ## heading or end of file.
func extractSection(content, sectionName string) string {
	// Match section header (with optional parenthetical)
	// Use \n## to match next section, but be careful to capture multi-line content
	pattern := regexp.MustCompile(`(?s)## ` + regexp.QuoteMeta(sectionName) + `(?:\s*\([^)]*\))?\s*\n(.*?)(?:\n---\n|\n## |\z)`)
	matches := pattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractSectionWithVariant tries multiple section name variants.
func extractSectionWithVariant(content, name1, name2 string) string {
	result := extractSection(content, name1)
	if result == "" {
		result = extractSection(content, name2)
	}
	return result
}

// extractRecommendation extracts the recommendation from the Next section.
// Looks for patterns like "**Recommendation:** close" or just "close" on its own line.
func extractRecommendation(nextSection string) string {
	matches := regexRecommendation.FindStringSubmatch(nextSection)
	if len(matches) >= 2 {
		return strings.ToLower(strings.TrimSpace(matches[1]))
	}
	return ""
}

// extractNextActions extracts follow-up action items from various sections.
func extractNextActions(content string) []string {
	var actions []string

	// Try "## Next Actions" section first
	actionsSection := extractSection(content, "Next Actions")
	if actionsSection != "" {
		actions = append(actions, parseActionItems(actionsSection)...)
	}

	// Also look for follow-up work in Next section
	nextSection := extractSectionWithVariant(content, "Next", "Next (What Should Happen)")
	if nextSection != "" {
		// Look for follow-up subsections with various naming conventions:
		// - "### Follow-up Work" or "### Follow-up Work Identified"
		// - "### Spawn Follow-up" or "### If Spawn Follow-up"
		followUpPatterns := []string{
			`(?s)### Follow-up Work[^\n]*\n(.*?)(?:\n###|\n---|\z)`,
			`(?s)### (?:If )?Spawn Follow-up[^\n]*\n(.*?)(?:\n###|\n---|\z)`,
		}
		for _, pattern := range followUpPatterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(nextSection)
			if len(matches) >= 2 {
				actions = append(actions, parseActionItems(matches[1])...)
			}
		}
	}

	return actions
}

// parseActionItems extracts list items (- item, * item, or 1. item format).
// Note: Uses "* " (asterisk+space) to distinguish bullet points from markdown bold (**text**).
// Note: Only matches non-indented lines to avoid capturing continuation/metadata lines
// that are indented under a parent item.
func parseActionItems(section string) []string {
	var items []string
	lines := strings.Split(section, "\n")

	for _, line := range lines {
		// Skip indented lines - they're continuation/metadata, not separate items
		// Check for indentation BEFORE trimming
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Match bullet points: "- item" or "* item" (with space after marker)
		// Using "* " to avoid matching markdown bold syntax like "**Skill:**"
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || regexNumberedPattern.MatchString(line) {
			items = append(items, line)
		}
	}
	return items
}

// extractBoldSubsection extracts list items from a subsection that starts with **bold header:**
// For example: **Areas worth exploring further:** followed by bullet points.
func extractBoldSubsection(content, header string) []string {
	var items []string

	// Find the bold header and extract content until the next bold header or end
	pattern := regexp.MustCompile(`(?s)\*\*` + regexp.QuoteMeta(header) + `:\*\*\s*\n(.*?)(?:\n\*\*|\n---|\z)`)
	matches := pattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return items
	}

	subsection := matches[1]
	lines := strings.Split(subsection, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Only extract bullet point items
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			items = append(items, line)
		}
	}

	return items
}

// VerifySynthesis checks if SYNTHESIS.md exists and is not empty.
func VerifySynthesis(workspacePath string) (bool, error) {
	if workspacePath == "" {
		return false, nil
	}
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	info, err := os.Stat(synthesisPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Size() > 0, nil
}

// VerifyCompletion checks if an agent is ready for completion.
// Returns a VerificationResult with any errors or warnings.
// Uses VerifyCompletionWithTier with an empty tier (reads from workspace).
func VerifyCompletion(beadsID string, workspacePath string) (VerificationResult, error) {
	return VerifyCompletionWithTier(beadsID, workspacePath, "")
}

// VerifyCompletionFull checks if an agent is ready for completion including skill constraints
// and phase gates. This extends VerifyCompletion with:
// 1. Constraint verification from SPAWN_CONTEXT.md (file patterns must match)
// 2. Phase gate verification (required phases must be reported via beads comments)
// 3. Skill output verification from skill.yaml outputs.required section
//
// The projectDir is used to verify that constraint patterns match actual files.
func VerifyCompletionFull(beadsID, workspacePath, projectDir, tier string) (VerificationResult, error) {
	// First run standard verification
	result, err := VerifyCompletionWithTier(beadsID, workspacePath, tier)
	if err != nil {
		return result, err
	}

	// If standard verification failed, no need to check constraints
	if !result.Passed {
		return result, nil
	}

	// Skip constraint verification if no workspace
	if workspacePath == "" {
		return result, nil
	}

	// Skip constraint verification if no project dir
	if projectDir == "" {
		return result, nil
	}

	// Verify skill constraints from SPAWN_CONTEXT.md
	constraintResult, err := VerifyConstraintsForCompletion(workspacePath, projectDir)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify constraints: %v", err))
		// Continue to phase gate verification even if constraints failed to parse
	} else {
		// Merge constraint results
		if !constraintResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, constraintResult.Errors...)
		}
		result.Warnings = append(result.Warnings, constraintResult.Warnings...)
	}

	// Verify phase gates from SPAWN_CONTEXT.md
	// This checks that required phases were reported in beads comments
	phaseGateResult, err := VerifyPhaseGatesForCompletion(workspacePath, beadsID)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify phase gates: %v", err))
	} else if !phaseGateResult.Passed {
		result.Passed = false
		result.Errors = append(result.Errors, phaseGateResult.Errors...)
	}

	// Verify skill outputs from skill.yaml outputs.required section
	// This is the "skillc verify" integration - checks that required skill outputs exist
	skillOutputResult, err := VerifySkillOutputsForCompletion(workspacePath, projectDir)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify skill outputs: %v", err))
	} else if skillOutputResult != nil {
		// Only add results if skill had outputs.required defined
		if !skillOutputResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, skillOutputResult.Errors...)
		}
		result.Warnings = append(result.Warnings, skillOutputResult.Warnings...)
	}

	// Verify visual verification for web/ changes
	// This gates completion when web files are modified without visual verification evidence
	visualResult := VerifyVisualVerificationForCompletion(beadsID, workspacePath, projectDir)
	if visualResult != nil {
		if !visualResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, visualResult.Errors...)
		}
		result.Warnings = append(result.Warnings, visualResult.Warnings...)
	}

	// Verify test execution evidence for code changes
	// This gates completion when code files are modified without test execution evidence
	testEvidenceResult := VerifyTestEvidenceForCompletion(beadsID, workspacePath, projectDir)
	if testEvidenceResult != nil {
		if !testEvidenceResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, testEvidenceResult.Errors...)
		}
		result.Warnings = append(result.Warnings, testEvidenceResult.Warnings...)
	}

	// Verify git diff against SYNTHESIS claims
	// This detects false positives where agent claims to modify files but didn't
	gitDiffResult := VerifyGitDiffForCompletion(workspacePath, projectDir)
	if gitDiffResult != nil {
		if !gitDiffResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, gitDiffResult.Errors...)
		}
		result.Warnings = append(result.Warnings, gitDiffResult.Warnings...)
	}

	// Verify build for Go projects
	// This gates completion when Go files are modified but the project doesn't build
	buildResult := VerifyBuildForCompletion(workspacePath, projectDir)
	if buildResult != nil {
		if !buildResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, buildResult.Errors...)
		}
		result.Warnings = append(result.Warnings, buildResult.Warnings...)
	}

	return result, nil
}

// VerifyCompletionWithTier checks if an agent is ready for completion.
// The tier parameter specifies the spawn tier ("light" or "full").
// If tier is empty, it will be read from the workspace's .tier file.
// Light tier spawns skip the SYNTHESIS.md requirement.
// Returns a VerificationResult with any errors or warnings.
func VerifyCompletionWithTier(beadsID string, workspacePath string, tier string) (VerificationResult, error) {
	result := VerificationResult{
		Passed: true,
	}

	// Get phase status
	status, err := GetPhaseStatus(beadsID)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to get phase status: %v", err))
		return result, nil
	}

	result.Phase = status

	// Check if Phase: Complete was reported
	if !status.Found {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("agent has not reported any Phase status for %s", beadsID))
		return result, nil
	}

	if !strings.EqualFold(status.Phase, "Complete") {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("agent phase is '%s', not 'Complete' (beads: %s)", status.Phase, beadsID))
		return result, nil
	}

	// Determine tier if not provided
	if tier == "" && workspacePath != "" {
		tier = ReadTierFromWorkspace(workspacePath)
	}

	// Check for SYNTHESIS.md (only for full tier)
	if workspacePath != "" && tier != "light" {
		ok, err := VerifySynthesis(workspacePath)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify SYNTHESIS.md: %v", err))
		} else if !ok {
			result.Passed = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("SYNTHESIS.md is missing or empty in workspace: %s", workspacePath))
		}
	}

	return result, nil
}

// ReadTierFromWorkspace reads the spawn tier from the workspace's .tier file.
// Returns "full" as the conservative default if the file doesn't exist.
func ReadTierFromWorkspace(workspacePath string) string {
	tierFile := filepath.Join(workspacePath, ".tier")
	data, err := os.ReadFile(tierFile)
	if err != nil {
		return "full" // Conservative default
	}
	tier := strings.TrimSpace(string(data))
	if tier == "" {
		return "full"
	}
	return tier
}
