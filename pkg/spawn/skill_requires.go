// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// RequiresContext represents the context requirements declared by a skill.
// This is parsed from the <!-- SKILL-REQUIRES --> block embedded by skillc.
type RequiresContext struct {
	// KBContext indicates whether to run kb context with task keywords.
	// When true, prior knowledge (constraints, decisions, investigations) is gathered.
	KBContext bool

	// BeadsIssue indicates whether to include beads issue details when spawned with --issue.
	// When true, issue title, description, and notes are included in context.
	BeadsIssue bool

	// PriorWork contains glob patterns for .kb/ files to load.
	// Matched files' TLDRs/summaries are included in context.
	PriorWork []string
}

// ParseSkillRequires extracts RequiresContext from skill content.
// Looks for the <!-- SKILL-REQUIRES --> block embedded by skillc.
//
// Expected format:
//
//	<!-- SKILL-REQUIRES -->
//	<!-- kb-context: true -->
//	<!-- beads-issue: true -->
//	<!-- prior-work: .kb/investigations/* -->
//	<!-- prior-work: .kb/decisions/* -->
//	<!-- /SKILL-REQUIRES -->
func ParseSkillRequires(skillContent string) *RequiresContext {
	if skillContent == "" {
		return nil
	}

	// Find the SKILL-REQUIRES block
	startMarker := "<!-- SKILL-REQUIRES -->"
	endMarker := "<!-- /SKILL-REQUIRES -->"

	startIdx := strings.Index(skillContent, startMarker)
	if startIdx == -1 {
		return nil
	}

	endIdx := strings.Index(skillContent[startIdx:], endMarker)
	if endIdx == -1 {
		return nil
	}

	// Extract the block content
	blockContent := skillContent[startIdx+len(startMarker) : startIdx+endIdx]

	result := &RequiresContext{}

	// Parse each line for requirements
	lines := strings.Split(blockContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse HTML comment format: <!-- key: value -->
		if !strings.HasPrefix(line, "<!--") || !strings.HasSuffix(line, "-->") {
			continue
		}

		// Extract content between <!-- and -->
		content := strings.TrimPrefix(line, "<!--")
		content = strings.TrimSuffix(content, "-->")
		content = strings.TrimSpace(content)

		// Parse key: value format
		parts := strings.SplitN(content, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "kb-context":
			result.KBContext = parseBool(value)
		case "beads-issue":
			result.BeadsIssue = parseBool(value)
		case "prior-work":
			if value != "" {
				result.PriorWork = append(result.PriorWork, value)
			}
		}
	}

	// If nothing was found in the block, return nil
	if !result.KBContext && !result.BeadsIssue && len(result.PriorWork) == 0 {
		return nil
	}

	return result
}

// parseBool parses a boolean value from string.
// Accepts "true", "yes", "1" as true; everything else as false.
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "yes" || s == "1"
}

// GatherRequiredContext gathers context based on skill requirements.
// Returns a formatted string suitable for inclusion in SPAWN_CONTEXT.md.
func GatherRequiredContext(requires *RequiresContext, task, beadsID, projectDir string, stalenessMeta *StalenessEventMeta) string {
	if requires == nil {
		return ""
	}

	var sections []string

	// Gather kb-context if required
	if requires.KBContext {
		kbContext := gatherKBContext(task, projectDir, stalenessMeta)
		if kbContext != "" {
			sections = append(sections, kbContext)
		}
	}

	// Gather beads-issue context if required and beadsID is provided
	if requires.BeadsIssue && beadsID != "" {
		issueContext := gatherBeadsIssueContext(beadsID)
		if issueContext != "" {
			sections = append(sections, issueContext)
		}
	}

	// Gather prior-work files if patterns are specified
	if len(requires.PriorWork) > 0 && projectDir != "" {
		priorWorkContext := gatherPriorWorkContext(requires.PriorWork, projectDir)
		if priorWorkContext != "" {
			sections = append(sections, priorWorkContext)
		}
	}

	if len(sections) == 0 {
		return ""
	}

	return strings.Join(sections, "\n\n")
}

// gatherKBContext runs kb context query with task keywords.
// Returns formatted context string or empty string if no matches.
// Also performs gap analysis and includes gap summary in the context if significant.
func gatherKBContext(task, projectDir string, stalenessMeta *StalenessEventMeta) string {
	// Extract keywords from task description
	keywords := ExtractKeywords(task, 3)
	if keywords == "" {
		// Perform gap analysis even when no keywords extracted
		gapAnalysis := AnalyzeGaps(nil, task, projectDir)
		if gapAnalysis.ShouldWarnAboutGaps() {
			fmt.Fprintf(os.Stderr, "\n%s\n", gapAnalysis.FormatGapWarning())
		}
		return ""
	}

	// Run kb context check (use projectDir for cross-project group resolution)
	result, err := RunKBContextCheckForDir(keywords, projectDir)
	if err != nil || result == nil || !result.HasMatches {
		// Try with broader search (single keyword)
		broadKeywords := ExtractKeywords(task, 1)
		if broadKeywords != "" && broadKeywords != keywords {
			result, err = RunKBContextCheckForDir(broadKeywords, projectDir)
		}
	}

	// Perform gap analysis
	gapAnalysis := AnalyzeGaps(result, keywords, projectDir)
	if gapAnalysis.ShouldWarnAboutGaps() {
		fmt.Fprintf(os.Stderr, "\n%s\n", gapAnalysis.FormatGapWarning())
	}

	if result == nil || !result.HasMatches {
		return ""
	}

	// Format context with optional gap summary
	if projectDir == "" {
		projectDir = "."
	}
	formatResult := FormatContextForSpawnWithLimitAndMeta(result, MaxKBContextChars, projectDir, stalenessMeta)
	contextContent := formatResult.Content
	if gapSummary := gapAnalysis.FormatGapSummary(); gapSummary != "" {
		contextContent = gapSummary + "\n\n" + contextContent
	}

	return contextContent
}

// beadsIssue represents minimal issue data from beads.
// This is a local type to avoid import cycles with verify package.
type beadsIssue struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IssueType   string `json:"type"`
	Status      string `json:"status"`
}

// beadsComment represents a comment on an issue.
type beadsComment struct {
	Text string `json:"text"`
}

// gatherBeadsIssueContext fetches beads issue details and formats them.
// Returns formatted issue context or empty string if issue not found.
// FRAME comments (prefixed with "FRAME:") are shown prominently and untruncated
// since they contain strategic context from the orchestrator.
func gatherBeadsIssueContext(beadsID string) string {
	if beadsID == "" {
		return ""
	}

	issue, err := getBeadsIssue(beadsID)
	if err != nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## BEADS ISSUE CONTEXT\n\n")
	sb.WriteString(fmt.Sprintf("**Issue:** %s\n", beadsID))
	sb.WriteString(fmt.Sprintf("**Title:** %s\n", issue.Title))
	if issue.IssueType != "" {
		sb.WriteString(fmt.Sprintf("**Type:** %s\n", issue.IssueType))
	}
	if issue.Description != "" {
		sb.WriteString(fmt.Sprintf("\n**Description:**\n%s\n", issue.Description))
	}

	// Get any notes/comments on the issue
	comments, err := getBeadsComments(beadsID)
	if err == nil && len(comments) > 0 {
		// Extract FRAME comments separately — show prominently and untruncated
		var frameComments []string
		var regularComments []beadsComment
		for _, comment := range comments {
			text := strings.TrimSpace(comment.Text)
			if strings.HasPrefix(text, "FRAME:") {
				frame := strings.TrimSpace(strings.TrimPrefix(text, "FRAME:"))
				if frame != "" {
					frameComments = append(frameComments, frame)
				}
			} else {
				regularComments = append(regularComments, comment)
			}
		}

		// Show FRAME comments first, prominently
		if len(frameComments) > 0 {
			sb.WriteString("\n**Strategic Frame:**\n")
			for _, frame := range frameComments {
				sb.WriteString(fmt.Sprintf("%s\n", frame))
			}
		}

		// Show last 5 regular comments (most recent context)
		if len(regularComments) > 0 {
			sb.WriteString("\n**Comments:**\n")
			startIdx := 0
			if len(regularComments) > 5 {
				startIdx = len(regularComments) - 5
			}
			for _, comment := range regularComments[startIdx:] {
				// Truncate long comments
				text := comment.Text
				if len(text) > 200 {
					text = text[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("- %s\n", text))
			}
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

// getBeadsIssue retrieves issue details from beads.
// It uses the beads RPC client when available, falling back to the bd CLI.
func getBeadsIssue(beadsID string) (*beadsIssue, error) {
	// Try RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()

			issue, err := client.Show(beadsID)
			if err == nil {
				return &beadsIssue{
					ID:          issue.ID,
					Title:       issue.Title,
					Description: issue.Description,
					IssueType:   issue.IssueType,
					Status:      issue.Status,
				}, nil
			}
			// Fall through to CLI fallback on RPC error
		}
	}

	// Fallback to CLI
	issue, err := beads.FallbackShow(beadsID, "")
	if err != nil {
		return nil, err
	}

	return &beadsIssue{
		ID:          issue.ID,
		Title:       issue.Title,
		Description: issue.Description,
		IssueType:   issue.IssueType,
		Status:      issue.Status,
	}, nil
}

// getBeadsComments retrieves comments for a beads issue.
// It uses the beads RPC client when available, falling back to the bd CLI.
func getBeadsComments(beadsID string) ([]beadsComment, error) {
	// Try RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()

			comments, err := client.Comments(beadsID)
			if err == nil {
				result := make([]beadsComment, len(comments))
				for i, c := range comments {
					result[i] = beadsComment{Text: c.Text}
				}
				return result, nil
			}
			// Fall through to CLI fallback on RPC error
		}
	}

	// Fallback to CLI
	comments, err := beads.FallbackComments(beadsID, "")
	if err != nil {
		return nil, err
	}

	result := make([]beadsComment, len(comments))
	for i, c := range comments {
		result[i] = beadsComment{Text: c.Text}
	}
	return result, nil
}

// ExtractFrameFromBeadsComments retrieves the most recent FRAME annotation from beads comments.
// FRAME comments (prefixed with "FRAME:") contain strategic context added by the orchestrator.
// Returns empty string if no frame is found or comments cannot be retrieved.
// This is used during spawn to include orchestrator framing in OrientationFrame,
// ensuring spawned agents see strategic context without needing to run `bd show`.
func ExtractFrameFromBeadsComments(beadsID string) string {
	if beadsID == "" {
		return ""
	}

	comments, err := getBeadsComments(beadsID)
	if err != nil || len(comments) == 0 {
		return ""
	}

	// Scan from newest to oldest for FRAME comment
	for i := len(comments) - 1; i >= 0; i-- {
		text := strings.TrimSpace(comments[i].Text)
		if strings.HasPrefix(text, "FRAME:") {
			return strings.TrimSpace(strings.TrimPrefix(text, "FRAME:"))
		}
	}

	return ""
}

// gatherPriorWorkContext loads files matching patterns and extracts TLDRs.
// Returns formatted prior work context or empty string if no matches.
func gatherPriorWorkContext(patterns []string, projectDir string) string {
	if len(patterns) == 0 || projectDir == "" {
		return ""
	}

	var files []priorWorkFile
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		// Make pattern relative to project dir
		fullPattern := filepath.Join(projectDir, pattern)

		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			if seen[match] {
				continue
			}
			seen[match] = true

			// Only load markdown files
			if !strings.HasSuffix(match, ".md") {
				continue
			}

			// Read file and extract TLDR
			content, err := os.ReadFile(match)
			if err != nil {
				continue
			}

			tldr := extractTLDR(string(content))
			if tldr == "" {
				continue
			}

			// Get relative path for display
			relPath, err := filepath.Rel(projectDir, match)
			if err != nil {
				relPath = match
			}

			files = append(files, priorWorkFile{
				Path: relPath,
				TLDR: tldr,
			})
		}

		// Limit to 10 files total to prevent context flood
		if len(files) >= 10 {
			break
		}
	}

	if len(files) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## PRIOR WORK (from skill requirements)\n\n")
	sb.WriteString("**Relevant files from .kb/ directory:**\n\n")

	for _, f := range files {
		sb.WriteString(fmt.Sprintf("### %s\n", f.Path))
		sb.WriteString(fmt.Sprintf("%s\n\n", f.TLDR))
	}

	return sb.String()
}

// priorWorkFile represents a file with its extracted TLDR.
type priorWorkFile struct {
	Path string
	TLDR string
}

// extractTLDR extracts the TLDR or summary section from markdown content.
// Looks for common patterns:
// - ## TLDR / ## Summary / ## Summary (D.E.K.N.)
// - **TLDR:** / **Summary:**
// - Delta: (from D.E.K.N. structure)
func extractTLDR(content string) string {
	lines := strings.Split(content, "\n")

	var inTLDR bool
	var tldrLines []string
	var sectionDepth int

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for TLDR/Summary section headers
		if strings.HasPrefix(trimmed, "## TLDR") ||
			strings.HasPrefix(trimmed, "## Summary") ||
			trimmed == "**TLDR:**" ||
			trimmed == "**Summary:**" {
			inTLDR = true
			sectionDepth = 2
			continue
		}

		// Check for D.E.K.N. Delta line (first part of summary)
		if strings.HasPrefix(trimmed, "**Delta:**") {
			// Extract just the delta line
			delta := strings.TrimPrefix(trimmed, "**Delta:**")
			delta = strings.TrimSpace(delta)
			if delta != "" {
				return delta
			}
			continue
		}

		if inTLDR {
			// End of section on next header of same or higher level
			if strings.HasPrefix(trimmed, "## ") && sectionDepth == 2 {
				break
			}
			if strings.HasPrefix(trimmed, "---") {
				break
			}

			// Skip empty lines at the start
			if len(tldrLines) == 0 && trimmed == "" {
				continue
			}

			tldrLines = append(tldrLines, trimmed)

			// Limit to first 3 non-empty lines (keep it concise)
			nonEmpty := 0
			for _, l := range tldrLines {
				if l != "" {
					nonEmpty++
				}
			}
			if nonEmpty >= 3 {
				break
			}
		}
	}

	// Fall back to first non-header paragraph if no TLDR found
	if len(tldrLines) == 0 {
		return extractFirstParagraph(content)
	}

	return strings.TrimSpace(strings.Join(tldrLines, "\n"))
}

// extractFirstParagraph extracts the first meaningful paragraph from content.
// Skips headers, front matter, and empty lines.
func extractFirstParagraph(content string) string {
	lines := strings.Split(content, "\n")

	var inFrontMatter bool
	var paragraphLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip front matter
		if trimmed == "---" {
			inFrontMatter = !inFrontMatter
			continue
		}
		if inFrontMatter {
			continue
		}

		// Skip headers
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Skip HTML comments
		if strings.HasPrefix(trimmed, "<!--") {
			continue
		}

		// Skip empty lines before content starts
		if len(paragraphLines) == 0 && trimmed == "" {
			continue
		}

		// End paragraph on empty line after content
		if len(paragraphLines) > 0 && trimmed == "" {
			break
		}

		paragraphLines = append(paragraphLines, trimmed)

		// Limit to 2 lines
		if len(paragraphLines) >= 2 {
			break
		}
	}

	result := strings.TrimSpace(strings.Join(paragraphLines, " "))
	// Truncate if too long
	if len(result) > 300 {
		result = result[:300] + "..."
	}
	return result
}

// HasRequirements returns true if the skill declares any context requirements.
func (r *RequiresContext) HasRequirements() bool {
	return r != nil && (r.KBContext || r.BeadsIssue || len(r.PriorWork) > 0)
}

// String returns a human-readable description of the requirements.
func (r *RequiresContext) String() string {
	if r == nil {
		return "none"
	}

	var parts []string
	if r.KBContext {
		parts = append(parts, "kb-context")
	}
	if r.BeadsIssue {
		parts = append(parts, "beads-issue")
	}
	if len(r.PriorWork) > 0 {
		parts = append(parts, fmt.Sprintf("prior-work(%d patterns)", len(r.PriorWork)))
	}

	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// regexTLDR is a regex to match common TLDR header patterns.
var regexTLDR = regexp.MustCompile(`(?i)^#+\s*(tldr|summary)\s*$`)
