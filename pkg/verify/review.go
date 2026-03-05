// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Pre-compiled regex patterns for review.go
var (
	regexFilesChanged = regexp.MustCompile(`(\d+) files? changed`)
	regexInsertion    = regexp.MustCompile(`(\d+) insertion`)
	regexDeletion     = regexp.MustCompile(`(\d+) deletion`)
)

// AgentReview contains comprehensive information about an agent's work for review.
type AgentReview struct {
	// Agent identification
	WorkspaceName string
	BeadsID       string
	Skill         string
	ProjectDir    string

	// Status
	Phase   string
	Summary string
	Status  string // Phase: Complete status

	// Synthesis (from SYNTHESIS.md)
	TLDR           string
	Outcome        string
	Recommendation string

	// Unexplored Questions (from SYNTHESIS.md)
	UnexploredQuestions string   // Raw section content
	AreasToExplore      []string // Areas worth exploring further
	Uncertainties       []string // What remains unclear

	// Delta (from git)
	FilesCreated  int
	FilesModified int
	Commits       []CommitInfo

	// Beads comments
	Comments []Comment

	// Artifacts
	SynthesisExists    bool
	InvestigationPath  string
	InvestigationFound bool

	// Light tier indicator
	IsLightTier bool // True if spawned as light tier (no SYNTHESIS.md by design)

	// Test results (if available)
	TestOutput string
}

// CommitInfo represents basic commit information.
type CommitInfo struct {
	Hash    string
	Message string
	Author  string
	Date    time.Time
	Stats   string // +X/-Y format
}

// GetAgentReview retrieves comprehensive review information for an agent.
func GetAgentReview(beadsID, workspacePath, projectDir string) (*AgentReview, error) {
	review := &AgentReview{
		BeadsID:    beadsID,
		ProjectDir: projectDir,
	}

	// Extract workspace name from path
	if workspacePath != "" {
		review.WorkspaceName = filepath.Base(workspacePath)
	}

	// Get phase status from beads comments
	comments, err := GetComments(beadsID, "")
	if err == nil {
		review.Comments = comments
		phaseStatus := ParsePhaseFromComments(comments)
		review.Phase = phaseStatus.Phase
		review.Summary = phaseStatus.Summary
		if strings.EqualFold(phaseStatus.Phase, "Complete") {
			review.Status = "Phase: Complete"
		}
	}

	// Parse synthesis if available
	if workspacePath != "" {
		synthesis, err := ParseSynthesis(workspacePath)
		if err == nil {
			review.SynthesisExists = true
			review.TLDR = synthesis.TLDR
			review.Outcome = synthesis.Outcome
			review.Recommendation = synthesis.Recommendation
			review.UnexploredQuestions = synthesis.UnexploredQuestions
			review.AreasToExplore = synthesis.AreasToExplore
			review.Uncertainties = synthesis.Uncertainties
		} else {
			// Check if file exists but couldn't parse
			synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
			if _, statErr := os.Stat(synthesisPath); statErr == nil {
				review.SynthesisExists = true
			}
		}

		// Look for investigation file
		review.InvestigationPath, review.InvestigationFound = findInvestigationFile(workspacePath)
	}

	// Get git commits for this workspace (if we can determine the timeframe)
	if projectDir != "" {
		commits, filesCreated, filesModified := getGitDelta(projectDir, workspacePath)
		review.Commits = commits
		review.FilesCreated = filesCreated
		review.FilesModified = filesModified
	}

	return review, nil
}

// findInvestigationFile looks for investigation files in the .kb/investigations directory.
func findInvestigationFile(workspacePath string) (string, bool) {
	// Get the project dir from workspace path
	// workspacePath is like: /path/to/project/.orch/workspace/{name}
	// We want: /path/to/project/.kb/investigations/
	projectDir := filepath.Dir(filepath.Dir(filepath.Dir(workspacePath)))
	investigationsDir := filepath.Join(projectDir, ".kb", "investigations")

	// Extract workspace name for matching
	workspaceName := filepath.Base(workspacePath)

	// Look for files that match the workspace name (approximately)
	entries, err := os.ReadDir(investigationsDir)
	if err != nil {
		return "", false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Look for investigation files matching patterns like:
		// 2025-12-21-inv-{keywords}.md
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		// Check if any significant keywords from workspace name are in the file name
		// Workspace names are like: og-feat-implement-orch-complete-21dec
		keywords := extractKeywords(workspaceName)
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(name), keyword) {
				return filepath.Join(investigationsDir, name), true
			}
		}
	}

	return "", false
}

// extractKeywords extracts meaningful keywords from a workspace name.
func extractKeywords(workspaceName string) []string {
	// Split by hyphens
	parts := strings.Split(workspaceName, "-")
	var keywords []string

	// Skip common prefixes like "og", "feat", "fix", etc.
	skipWords := map[string]bool{
		"og": true, "feat": true, "fix": true, "debug": true, "inv": true,
		"21dec": true, "20dec": true, "19dec": true, "18dec": true,
	}

	for _, part := range parts {
		part = strings.ToLower(part)
		if len(part) < 3 {
			continue
		}
		if skipWords[part] {
			continue
		}
		keywords = append(keywords, part)
	}

	return keywords
}

// getGitDelta retrieves git commit information and file changes.
func getGitDelta(projectDir, workspacePath string) ([]CommitInfo, int, int) {
	var commits []CommitInfo
	filesCreated := 0
	filesModified := 0

	// Try to get commits from the last 24 hours (reasonable window for agent work)
	// Format: hash|message|author|date
	cmd := exec.Command("git", "log", "--since=24 hours ago", "--format=%h|%s|%an|%ai", "--stat", "-10")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return commits, filesCreated, filesModified
	}

	// Parse git log output
	lines := strings.Split(string(output), "\n")
	var currentCommit *CommitInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this is a commit header line (contains |)
		if strings.Count(line, "|") >= 3 {
			if currentCommit != nil {
				commits = append(commits, *currentCommit)
			}
			parts := strings.SplitN(line, "|", 4)
			if len(parts) >= 4 {
				currentCommit = &CommitInfo{
					Hash:    parts[0],
					Message: parts[1],
					Author:  parts[2],
				}
				if t, err := time.Parse("2006-01-02 15:04:05 -0700", parts[3]); err == nil {
					currentCommit.Date = t
				}
			}
		} else if currentCommit != nil && strings.Contains(line, "file") && strings.Contains(line, "changed") {
			// This is the summary line like "1 file changed, 10 insertions(+)"
			currentCommit.Stats = line

			// Parse file changes
			if matches := regexFilesChanged.FindStringSubmatch(line); len(matches) > 1 {
				fmt.Sscanf(matches[1], "%d", &filesModified)
			}

			// Check for insertions/deletions for rough file count
			if regexInsertion.MatchString(line) && !regexDeletion.MatchString(line) {
				// Likely new files
				filesCreated++
			}
		}
	}

	if currentCommit != nil {
		commits = append(commits, *currentCommit)
	}

	return commits, filesCreated, filesModified
}

// FormatAgentReview formats the agent review for display.
func FormatAgentReview(review *AgentReview) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("AGENT REVIEW: %s\n", review.WorkspaceName))
	sb.WriteString(strings.Repeat("─", 60))
	sb.WriteString("\n\n")

	// Basic info
	sb.WriteString(fmt.Sprintf("Beads:  %s\n", review.BeadsID))
	if review.Skill != "" {
		sb.WriteString(fmt.Sprintf("Skill:  %s\n", review.Skill))
	}
	if review.Status != "" {
		sb.WriteString(fmt.Sprintf("Status: %s\n", review.Status))
	}
	sb.WriteString("\n")

	// TLDR section
	if review.TLDR != "" {
		sb.WriteString("TLDR:\n")
		// Wrap TLDR text for readability
		tldr := wrapText(review.TLDR, 70)
		for _, line := range strings.Split(tldr, "\n") {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
		sb.WriteString("\n")
	}

	// Delta section
	sb.WriteString("DELTA:\n")
	if review.FilesCreated > 0 || review.FilesModified > 0 {
		sb.WriteString(fmt.Sprintf("  Files:   +%d created, %d modified\n", review.FilesCreated, review.FilesModified))
	}
	if len(review.Commits) > 0 {
		commitHashes := make([]string, 0, len(review.Commits))
		for _, c := range review.Commits {
			commitHashes = append(commitHashes, c.Hash)
		}
		sb.WriteString(fmt.Sprintf("  Commits: %s\n", strings.Join(commitHashes, ", ")))
	}
	if review.FilesCreated == 0 && review.FilesModified == 0 && len(review.Commits) == 0 {
		sb.WriteString("  (no changes detected)\n")
	}
	sb.WriteString("\n")

	// Beads comments section (last 5)
	if len(review.Comments) > 0 {
		sb.WriteString("BEADS COMMENTS:\n")
		start := 0
		if len(review.Comments) > 5 {
			start = len(review.Comments) - 5
		}
		for _, c := range review.Comments[start:] {
			// Truncate long comments
			text := c.Text
			if len(text) > 80 {
				text = text[:77] + "..."
			}
			sb.WriteString(fmt.Sprintf("  • %s\n", text))
		}
		sb.WriteString("\n")
	}

	// Artifacts section
	sb.WriteString("ARTIFACTS:\n")
	if review.SynthesisExists {
		outcomeStr := ""
		if review.Outcome != "" {
			outcomeStr = fmt.Sprintf(" (%s", review.Outcome)
			if review.Recommendation != "" {
				outcomeStr += fmt.Sprintf(", rec=%s", review.Recommendation)
			}
			outcomeStr += ")"
		}
		sb.WriteString(fmt.Sprintf("  • SYNTHESIS.md%s\n", outcomeStr))
	} else if review.IsLightTier {
		sb.WriteString("  • SYNTHESIS.md (not required - light tier)\n")
	} else {
		sb.WriteString("  • SYNTHESIS.md (missing)\n")
	}
	if review.InvestigationFound {
		sb.WriteString(fmt.Sprintf("  • %s\n", filepath.Base(review.InvestigationPath)))
	}
	sb.WriteString("\n")

	// Unexplored Questions section
	hasUnexplored := len(review.AreasToExplore) > 0 || len(review.Uncertainties) > 0 || review.UnexploredQuestions != ""
	if hasUnexplored {
		sb.WriteString("UNEXPLORED QUESTIONS:\n")

		// Show areas to explore
		if len(review.AreasToExplore) > 0 {
			sb.WriteString("  Areas to explore:\n")
			for _, area := range review.AreasToExplore {
				// Clean up the bullet point prefix
				area = strings.TrimPrefix(area, "- ")
				area = strings.TrimPrefix(area, "* ")
				if len(area) > 70 {
					area = area[:67] + "..."
				}
				sb.WriteString(fmt.Sprintf("    • %s\n", area))
			}
		}

		// Show uncertainties
		if len(review.Uncertainties) > 0 {
			sb.WriteString("  What remains unclear:\n")
			for _, uncertainty := range review.Uncertainties {
				// Clean up the bullet point prefix
				uncertainty = strings.TrimPrefix(uncertainty, "- ")
				uncertainty = strings.TrimPrefix(uncertainty, "* ")
				if len(uncertainty) > 70 {
					uncertainty = uncertainty[:67] + "..."
				}
				sb.WriteString(fmt.Sprintf("    • %s\n", uncertainty))
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// wrapText wraps text at the specified width.
func wrapText(text string, width int) string {
	// Replace newlines with spaces for uniform wrapping
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.TrimSpace(text)

	if len(text) <= width {
		return text
	}

	var lines []string
	for len(text) > width {
		// Find last space before width
		idx := strings.LastIndex(text[:width], " ")
		if idx <= 0 {
			idx = width
		}
		lines = append(lines, text[:idx])
		text = strings.TrimSpace(text[idx:])
	}
	if len(text) > 0 {
		lines = append(lines, text)
	}

	return strings.Join(lines, "\n")
}
