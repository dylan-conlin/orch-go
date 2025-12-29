// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// Comment is an alias for beads.Comment for compatibility.
type Comment = beads.Comment

// Issue represents a beads issue with comments.
type Issue struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	IssueType   string    `json:"issue_type"`
	CloseReason string    `json:"close_reason,omitempty"`
	Comments    []Comment `json:"comments"`
}

// PhaseStatus represents the current phase of an agent.
type PhaseStatus struct {
	Phase         string // Current phase (e.g., "Complete", "Implementing", "Planning")
	Summary       string // Optional summary from the phase comment
	Found         bool   // Whether a Phase: comment was found
	IsBlocked     bool   // Whether a BLOCKED comment was found
	BlockedReason string // The reason from the BLOCKED comment
}

// GetComments retrieves comments for a beads issue.
// It uses the beads RPC client when available, falling back to the bd CLI.
func GetComments(beadsID string) ([]Comment, error) {
	return GetCommentsWithDir(beadsID, "")
}

// GetCommentsWithDir retrieves comments for a beads issue from a specific project directory.
// This is used for cross-project agent visibility where the beads issue is in a different
// project than the current working directory.
// If projectDir is empty, uses the current working directory.
func GetCommentsWithDir(beadsID, projectDir string) ([]Comment, error) {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		comments, err := client.Comments(beadsID)
		if err == nil {
			return comments, nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI with project directory
	return FallbackCommentsWithDir(beadsID, projectDir)
}

// FallbackCommentsWithDir retrieves comments via the beads CLIClient in a specific directory.
func FallbackCommentsWithDir(beadsID, projectDir string) ([]Comment, error) {
	opts := []beads.CLIOption{}
	if projectDir != "" {
		opts = append(opts, beads.WithWorkDir(projectDir))
	}
	client := beads.NewCLIClient(opts...)
	return client.Comments(beadsID)
}

// ParsePhaseFromComments extracts the latest Phase status from comments.
// Looks for comments matching "Phase: <phase> - <summary>" pattern.
// Also detects BLOCKED: comments and sets IsBlocked accordingly.
// If a Phase update occurs after a BLOCKED comment, the blocked status is cleared
// (agent resumed work after being unblocked).
func ParsePhaseFromComments(comments []Comment) PhaseStatus {
	// Pattern: "Phase: <phase>" optionally followed by " - <summary>"
	phasePattern := regexp.MustCompile(`(?i)Phase:\s*(\w+)(?:\s*[-–—]\s*(.*))?`)
	// Pattern: "BLOCKED:" followed by reason
	blockedPattern := regexp.MustCompile(`(?i)BLOCKED:\s*(.*)`)

	var latestPhase PhaseStatus
	latestBlockedIdx := -1 // Track index of latest BLOCKED comment
	latestPhaseIdx := -1   // Track index of latest Phase comment

	for idx, comment := range comments {
		// Check for Phase: pattern
		matches := phasePattern.FindStringSubmatch(comment.Text)
		if len(matches) >= 2 {
			latestPhase.Phase = matches[1]
			latestPhase.Found = true
			if len(matches) >= 3 && matches[2] != "" {
				latestPhase.Summary = strings.TrimSpace(matches[2])
			}
			latestPhaseIdx = idx
		}

		// Check for BLOCKED: pattern
		blockedMatches := blockedPattern.FindStringSubmatch(comment.Text)
		if len(blockedMatches) >= 2 {
			latestPhase.IsBlocked = true
			latestPhase.BlockedReason = strings.TrimSpace(blockedMatches[1])
			latestBlockedIdx = idx
		}
	}

	// If there was a Phase update after the BLOCKED comment, clear the blocked status
	// (agent reported progress after being blocked, so it's no longer blocked)
	if latestPhase.IsBlocked && latestPhaseIdx > latestBlockedIdx {
		latestPhase.IsBlocked = false
		latestPhase.BlockedReason = ""
	}

	return latestPhase
}

// GetPhaseStatus retrieves the current phase status for a beads issue.
func GetPhaseStatus(beadsID string) (PhaseStatus, error) {
	comments, err := GetComments(beadsID)
	if err != nil {
		return PhaseStatus{}, err
	}

	return ParsePhaseFromComments(comments), nil
}

// IsPhaseComplete returns true if the agent has reported "Phase: Complete".
func IsPhaseComplete(beadsID string) (bool, error) {
	status, err := GetPhaseStatus(beadsID)
	if err != nil {
		return false, err
	}

	if !status.Found {
		return false, nil
	}

	return strings.EqualFold(status.Phase, "Complete"), nil
}

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
	// Try explicit recommendation field
	recPattern := regexp.MustCompile(`(?m)\*\*Recommendation:\*\*\s*(\w+)`)
	matches := recPattern.FindStringSubmatch(nextSection)
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
	numberedPattern := regexp.MustCompile(`^\d+\.`)

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
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || numberedPattern.MatchString(line) {
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
// 4. Visual verification for web/ changes
// 5. Test execution evidence for code changes (blocks when code modified without test output)
// 6. Build verification for Go projects (blocks when Go files modified but build fails)
// 7. Git commit verification for code-producing skills (blocks when no commits since spawn)
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

	// Verify git commits exist for code-producing skills
	// This gates completion when a code-producing skill (feature-impl, systematic-debugging)
	// reports Phase: Complete but has no git commits since spawn time
	gitCommitResult := VerifyGitCommitsForCompletion(workspacePath, projectDir)
	if gitCommitResult != nil {
		if !gitCommitResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, gitCommitResult.Errors...)
		}
		result.Warnings = append(result.Warnings, gitCommitResult.Warnings...)
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

// CloseIssue closes a beads issue with the given reason.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
func CloseIssue(beadsID, reason string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.CloseIssue(beadsID, reason); err == nil {
			return nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackClose(beadsID, reason)
}

// UpdateIssueStatus updates the status of a beads issue.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
func UpdateIssueStatus(beadsID, status string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		statusPtr := &status
		_, err := client.Update(&beads.UpdateArgs{
			ID:     beadsID,
			Status: statusPtr,
		})
		if err == nil {
			return nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackUpdate(beadsID, status)
}

// RemoveTriageReadyLabel removes the triage:ready label from a beads issue.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
// This should be called after UpdateIssueStatus transitions the issue to in_progress.
func RemoveTriageReadyLabel(beadsID string) error {
	const triageReadyLabel = "triage:ready"

	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		err := client.RemoveLabel(beadsID, triageReadyLabel)
		if err == nil {
			return nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackRemoveLabel(beadsID, triageReadyLabel)
}

// GetIssue retrieves issue details from beads.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
func GetIssue(beadsID string) (*Issue, error) {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		issue, err := client.Show(beadsID)
		if err == nil {
			// Convert beads.Issue to verify.Issue
			return &Issue{
				ID:          issue.ID,
				Title:       issue.Title,
				Description: issue.Description,
				Status:      issue.Status,
				IssueType:   issue.IssueType,
				CloseReason: issue.CloseReason,
				// Comments are not populated via Show() - use GetComments() if needed
			}, nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	issue, err := beads.FallbackShow(beadsID)
	if err != nil {
		return nil, err
	}

	return &Issue{
		ID:          issue.ID,
		Title:       issue.Title,
		Description: issue.Description,
		Status:      issue.Status,
		IssueType:   issue.IssueType,
		CloseReason: issue.CloseReason,
	}, nil
}

// GetIssuesBatch retrieves multiple issues efficiently.
// Returns a map from beadsID to Issue. Missing/invalid IDs are silently skipped.
// Uses List() to get all issues in one call, then filters by requested IDs.
// This is much faster than individual Show() calls (1 call vs N calls).
func GetIssuesBatch(beadsIDs []string) (map[string]*Issue, error) {
	if len(beadsIDs) == 0 {
		return make(map[string]*Issue), nil
	}

	// Build a set of requested IDs for O(1) lookup
	requestedIDs := make(map[string]bool, len(beadsIDs))
	for _, id := range beadsIDs {
		requestedIDs[id] = true
	}

	result := make(map[string]*Issue, len(beadsIDs))

	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))

		// Fetch ALL issues in one call, then filter
		allIssues, err := client.List(nil)
		if err == nil {
			for i := range allIssues {
				if requestedIDs[allIssues[i].ID] {
					result[allIssues[i].ID] = &Issue{
						ID:          allIssues[i].ID,
						Title:       allIssues[i].Title,
						Description: allIssues[i].Description,
						Status:      allIssues[i].Status,
						IssueType:   allIssues[i].IssueType,
						CloseReason: allIssues[i].CloseReason,
					}
				}
			}
			return result, nil
		}
		// Fall through to CLI if RPC failed
	}

	// Fallback to CLI - List all issues in one call
	allIssues, err := beads.FallbackList("")
	if err != nil {
		return result, nil // Return empty on error (don't fail the whole request)
	}

	for i := range allIssues {
		if requestedIDs[allIssues[i].ID] {
			result[allIssues[i].ID] = &Issue{
				ID:          allIssues[i].ID,
				Title:       allIssues[i].Title,
				Description: allIssues[i].Description,
				Status:      allIssues[i].Status,
				IssueType:   allIssues[i].IssueType,
				CloseReason: allIssues[i].CloseReason,
			}
		}
	}

	return result, nil
}

// ListOpenIssues retrieves all open issues in a single call.
// Returns a map from beadsID to Issue.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
func ListOpenIssues() (map[string]*Issue, error) {
	result := make(map[string]*Issue)

	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))

		// List all issues via RPC
		issues, err := client.List(nil)
		if err == nil {
			// Filter to open/in_progress/blocked statuses
			for i := range issues {
				status := strings.ToLower(issues[i].Status)
				if status == "open" || status == "in_progress" || status == "blocked" {
					result[issues[i].ID] = &Issue{
						ID:          issues[i].ID,
						Title:       issues[i].Title,
						Description: issues[i].Description,
						Status:      issues[i].Status,
						IssueType:   issues[i].IssueType,
						CloseReason: issues[i].CloseReason,
					}
				}
			}
			return result, nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	issues, err := beads.FallbackList("")
	if err != nil {
		return nil, err
	}

	// Filter to open/in_progress/blocked statuses
	for i := range issues {
		status := strings.ToLower(issues[i].Status)
		if status == "open" || status == "in_progress" || status == "blocked" {
			result[issues[i].ID] = &Issue{
				ID:          issues[i].ID,
				Title:       issues[i].Title,
				Description: issues[i].Description,
				Status:      issues[i].Status,
				IssueType:   issues[i].IssueType,
				CloseReason: issues[i].CloseReason,
			}
		}
	}

	return result, nil
}

// GetCommentsBatch fetches comments for multiple issues sequentially.
// Returns a map from beadsID to comments. Errors are silently skipped.
// Uses the beads RPC client with auto-reconnect, so no concurrency control needed.
func GetCommentsBatch(beadsIDs []string) map[string][]Comment {
	if len(beadsIDs) == 0 {
		return make(map[string][]Comment)
	}

	commentMap := make(map[string][]Comment, len(beadsIDs))

	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))

		// Fetch comments sequentially via RPC
		for _, beadsID := range beadsIDs {
			comments, err := client.Comments(beadsID)
			if err == nil {
				commentMap[beadsID] = comments
			}
		}
		if len(commentMap) > 0 {
			return commentMap
		}
		// Fall through to CLI if no results
	}

	// Fallback to CLI for each issue
	for _, beadsID := range beadsIDs {
		comments, err := beads.FallbackComments(beadsID)
		if err == nil {
			commentMap[beadsID] = comments
		}
	}

	return commentMap
}

// GetCommentsBatchWithProjectDirs fetches comments for multiple issues in parallel.
// The projectDirs map should contain beadsID -> projectDir mappings.
// For beads IDs not in projectDirs, the current working directory is used.
// Returns a map from beadsID to comments. Errors are silently skipped.
// This is used for cross-project agent visibility where agents may be from different projects.
// Uses goroutines with semaphore to parallelize fetching (much faster than sequential).
func GetCommentsBatchWithProjectDirs(beadsIDs []string, projectDirs map[string]string) map[string][]Comment {
	if len(beadsIDs) == 0 {
		return make(map[string][]Comment)
	}

	// Use mutex-protected map for thread-safe writes
	var mu sync.Mutex
	commentMap := make(map[string][]Comment, len(beadsIDs))

	// Group beads IDs by project directory for efficient RPC client reuse
	byProjectDir := make(map[string][]string)
	for _, beadsID := range beadsIDs {
		dir := projectDirs[beadsID]
		byProjectDir[dir] = append(byProjectDir[dir], beadsID)
	}

	// Limit concurrent RPC calls to avoid overwhelming the server
	const maxConcurrent = 20
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup

	// Process each project directory group in parallel
	for projectDir, ids := range byProjectDir {
		// Try RPC client first
		socketPath, err := beads.FindSocketPath(projectDir)
		if err == nil {
			client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))

			// Fetch comments in parallel via RPC
			for _, beadsID := range ids {
				wg.Add(1)
				go func(id string, c *beads.Client) {
					defer wg.Done()
					sem <- struct{}{}        // Acquire semaphore
					defer func() { <-sem }() // Release semaphore

					comments, err := c.Comments(id)
					if err == nil {
						mu.Lock()
						commentMap[id] = comments
						mu.Unlock()
					}
				}(beadsID, client)
			}
		} else {
			// Fallback to CLI for this project dir in parallel
			for _, beadsID := range ids {
				wg.Add(1)
				go func(id string, dir string) {
					defer wg.Done()
					sem <- struct{}{}        // Acquire semaphore
					defer func() { <-sem }() // Release semaphore

					comments, err := FallbackCommentsWithDir(id, dir)
					if err == nil {
						mu.Lock()
						commentMap[id] = comments
						mu.Unlock()
					}
				}(beadsID, projectDir)
			}
		}
	}

	wg.Wait()
	return commentMap
}

// HasBeadsComment checks if a beads issue has any comments.
// Returns true if at least one comment exists, false otherwise.
// This is used to detect sessions that spawn but never execute (no Phase report).
func HasBeadsComment(beadsID string) (bool, error) {
	comments, err := GetComments(beadsID)
	if err != nil {
		return false, err
	}
	return len(comments) > 0, nil
}

// CommentCheckResult contains the result of checking for beads comments.
type CommentCheckResult struct {
	HasComments     bool   // Whether any comments exist
	CommentCount    int    // Number of comments
	FirstCommentAge int    // Age of first comment in seconds (0 if no comments)
	HasPhase        bool   // Whether a Phase: comment was found
	LatestPhase     string // The latest phase reported (empty if none)
}

// CheckCommentsWithAge checks if a beads issue has comments and returns detailed info.
// This is used for detecting sessions that may have failed to start.
func CheckCommentsWithAge(beadsID string) (*CommentCheckResult, error) {
	comments, err := GetComments(beadsID)
	if err != nil {
		return nil, err
	}

	result := &CommentCheckResult{
		HasComments:  len(comments) > 0,
		CommentCount: len(comments),
	}

	if len(comments) > 0 {
		// Parse phase from comments
		phaseStatus := ParsePhaseFromComments(comments)
		result.HasPhase = phaseStatus.Found
		result.LatestPhase = phaseStatus.Phase
	}

	return result, nil
}

// WaitForFirstCommentResult contains the result of waiting for a first comment.
type WaitForFirstCommentResult struct {
	Found     bool          // Whether a comment was found within timeout
	Timeout   bool          // Whether the wait timed out
	WaitedFor time.Duration // How long we waited before giving up or finding comment
	HasPhase  bool          // Whether a Phase: comment was found
	Phase     string        // The phase if found
	Error     error         // Any error during polling
}

// WaitForFirstComment polls for the first beads comment with a timeout.
// This is used after spawning to detect if the session actually started.
// Returns when either:
// - A comment is found (success)
// - Timeout is reached (possible failed-to-start)
// - An error occurs (unexpected failure)
//
// pollInterval controls how frequently to check (default: 5s)
// timeout controls how long to wait total (default: 60s)
func WaitForFirstComment(beadsID string, timeout, pollInterval time.Duration) *WaitForFirstCommentResult {
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	if pollInterval == 0 {
		pollInterval = 5 * time.Second
	}

	startTime := time.Now()
	deadline := startTime.Add(timeout)

	for time.Now().Before(deadline) {
		comments, err := GetComments(beadsID)
		if err != nil {
			// On error, sleep and retry (daemon might be restarting)
			time.Sleep(pollInterval)
			continue
		}

		if len(comments) > 0 {
			// Found comment(s)
			phaseStatus := ParsePhaseFromComments(comments)
			return &WaitForFirstCommentResult{
				Found:     true,
				WaitedFor: time.Since(startTime),
				HasPhase:  phaseStatus.Found,
				Phase:     phaseStatus.Phase,
			}
		}

		// No comments yet, sleep before next poll
		time.Sleep(pollInterval)
	}

	// Timeout reached without finding comments
	return &WaitForFirstCommentResult{
		Timeout:   true,
		WaitedFor: time.Since(startTime),
	}
}
