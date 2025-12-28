// synthesis.go - Synthesize recent activity from git, beads, and investigations
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/spf13/cobra"
)

var (
	// Synthesis command flags
	synthesisDays       int
	synthesisJSONOutput bool
	synthesisProject    string
)

var synthesisCmd = &cobra.Command{
	Use:   "synthesis",
	Short: "Synthesize recent activity from git, beads, and investigations",
	Long: `Synthesize recent activity from multiple sources:
- Git commits (last N days)
- Beads issues closed (last N days)
- Investigation TLDRs (last N days)

Output is grouped by category:
- Bugs Fixed
- Features Added
- Key Learnings
- Refactoring

Examples:
  orch synthesis                  # Show last 7 days
  orch synthesis --days 14        # Show last 14 days
  orch synthesis --json           # Output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSynthesis()
	},
}

func init() {
	synthesisCmd.Flags().IntVar(&synthesisDays, "days", 7, "Number of days to look back")
	synthesisCmd.Flags().BoolVar(&synthesisJSONOutput, "json", false, "Output as JSON")
	synthesisCmd.Flags().StringVar(&synthesisProject, "project", "", "Project directory (default: current dir)")

	rootCmd.AddCommand(synthesisCmd)
}

// SynthesisItem represents a single activity item.
type SynthesisItem struct {
	Source  string `json:"source"` // "git", "beads", or "investigation"
	Message string `json:"message,omitempty"`
	ID      string `json:"id,omitempty"`
	Title   string `json:"title,omitempty"`
	Path    string `json:"path,omitempty"`
	TLDR    string `json:"tldr,omitempty"`
}

// SynthesisData represents the complete synthesis output.
type SynthesisData struct {
	Bugs        []SynthesisItem `json:"bugs"`
	Features    []SynthesisItem `json:"features"`
	Learnings   []SynthesisItem `json:"learnings"`
	Refactoring []SynthesisItem `json:"refactoring"`
	Other       []SynthesisItem `json:"other"`
	Metadata    struct {
		ProjectName        string `json:"project_name"`
		Days               int    `json:"days"`
		StartDate          string `json:"start_date"`
		EndDate            string `json:"end_date"`
		CommitCount        int    `json:"commit_count"`
		IssueCount         int    `json:"issue_count"`
		InvestigationCount int    `json:"investigation_count"`
	} `json:"metadata"`
}

func runSynthesis() error {
	// Determine project directory
	projectDir := synthesisProject
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Synthesize activity
	data := synthesizeActivity(projectDir, synthesisDays)

	if synthesisJSONOutput {
		output, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Format human-readable output
	output := formatSynthesisOutput(data)
	fmt.Println(output)
	return nil
}

func synthesizeActivity(projectDir string, days int) SynthesisData {
	data := SynthesisData{
		Bugs:        []SynthesisItem{},
		Features:    []SynthesisItem{},
		Learnings:   []SynthesisItem{},
		Refactoring: []SynthesisItem{},
		Other:       []SynthesisItem{},
	}

	// Get git commits
	commits := getGitCommits(projectDir, days)
	for _, commit := range commits {
		category := categorizeCommit(commit)
		item := SynthesisItem{
			Source:  "git",
			Message: commit,
		}
		switch category {
		case "bugs":
			data.Bugs = append(data.Bugs, item)
		case "features":
			data.Features = append(data.Features, item)
		case "refactoring":
			data.Refactoring = append(data.Refactoring, item)
		case "learnings":
			data.Learnings = append(data.Learnings, item)
		default:
			data.Other = append(data.Other, item)
		}
	}

	// Get closed beads issues
	issues := getClosedIssues(projectDir, days)
	for _, issue := range issues {
		category := categorizeIssue(issue)
		item := SynthesisItem{
			Source: "beads",
			ID:     issue.ID,
			Title:  issue.Title,
		}
		switch category {
		case "bugs":
			data.Bugs = append(data.Bugs, item)
		case "features":
			data.Features = append(data.Features, item)
		default:
			data.Other = append(data.Other, item)
		}
	}

	// Get investigation TLDRs
	investigations := findRecentInvestigations(projectDir, days)
	for _, inv := range investigations {
		tldr := extractTLDR(inv)
		if tldr != "" {
			data.Learnings = append(data.Learnings, SynthesisItem{
				Source: "investigation",
				Path:   filepath.Base(inv),
				TLDR:   tldr,
			})
		}
	}

	// Set metadata
	now := time.Now()
	start := now.AddDate(0, 0, -days)
	data.Metadata.ProjectName = filepath.Base(projectDir)
	data.Metadata.Days = days
	data.Metadata.StartDate = start.Format("Jan 02")
	data.Metadata.EndDate = now.Format("Jan 02")
	data.Metadata.CommitCount = len(commits)
	data.Metadata.IssueCount = len(issues)
	data.Metadata.InvestigationCount = len(investigations)

	return data
}

func getGitCommits(projectDir string, days int) []string {
	cmd := exec.Command("git", "log", "--oneline", fmt.Sprintf("--since=%d.days.ago", days))
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var commits []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove hash prefix
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			commits = append(commits, parts[1])
		} else {
			commits = append(commits, line)
		}
	}
	return commits
}

func getClosedIssues(projectDir string, days int) []beads.Issue {
	// Try RPC client
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()
			issues, err := client.List(&beads.ListArgs{Status: "closed"})
			if err == nil {
				return filterIssuesByDate(issues, days)
			}
		}
	}

	// Fall back to CLI
	issues, err := beads.FallbackList("closed")
	if err != nil {
		return nil
	}
	return filterIssuesByDate(issues, days)
}

func filterIssuesByDate(issues []beads.Issue, days int) []beads.Issue {
	cutoff := time.Now().AddDate(0, 0, -days)
	var filtered []beads.Issue

	for _, issue := range issues {
		if issue.ClosedAt == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, issue.ClosedAt)
		if err != nil {
			// Try other formats
			t, err = time.Parse("2006-01-02T15:04:05", issue.ClosedAt)
		}
		if err == nil && t.After(cutoff) {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func findRecentInvestigations(projectDir string, days int) []string {
	kbDir := filepath.Join(projectDir, ".kb", "investigations")
	if _, err := os.Stat(kbDir); os.IsNotExist(err) {
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	var investigations []string

	// Search for markdown files
	patterns := []string{"*.md", "*/*.md"}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(kbDir, pattern))
		for _, match := range matches {
			filename := filepath.Base(match)
			// Check date prefix
			if len(filename) >= 10 && filename[:10] >= cutoff {
				investigations = append(investigations, match)
			}
		}
	}

	return investigations
}

func extractTLDR(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	// Match **TLDR:** followed by content
	re := regexp.MustCompile(`\*\*TLDR:\*\*\s*(.+?)(?:\n---|\z)`)
	match := re.FindSubmatch(content)
	if match != nil {
		tldr := strings.TrimSpace(string(match[1]))
		// Clean up multiline
		tldr = strings.Join(strings.Fields(tldr), " ")
		return tldr
	}
	return ""
}

func categorizeCommit(message string) string {
	lower := strings.ToLower(message)
	if strings.HasPrefix(lower, "fix:") || strings.HasPrefix(lower, "fix(") {
		return "bugs"
	} else if strings.HasPrefix(lower, "feat:") || strings.HasPrefix(lower, "feat(") {
		return "features"
	} else if strings.HasPrefix(lower, "refactor:") || strings.HasPrefix(lower, "refactor(") {
		return "refactoring"
	} else if strings.HasPrefix(lower, "docs:") || strings.HasPrefix(lower, "docs(") {
		return "learnings"
	}
	return "other"
}

func categorizeIssue(issue beads.Issue) string {
	switch strings.ToLower(issue.IssueType) {
	case "bug":
		return "bugs"
	case "feature":
		return "features"
	default:
		return "other"
	}
}

// extractKeyPoint extracts the key point from a commit message, removing prefix.
func extractKeyPoint(message string, maxLength int) string {
	if message == "" {
		return ""
	}

	// Remove conventional commit prefix
	re := regexp.MustCompile(`^(fix|feat|refactor|docs|test|chore|style|perf|ci|build|revert)(\([^)]+\))?\s*:\s*`)
	stripped := re.ReplaceAllString(message, "")
	if stripped == "" {
		stripped = message
	}

	// Capitalize first letter
	if len(stripped) > 0 {
		stripped = strings.ToUpper(string(stripped[0])) + stripped[1:]
	}

	// Truncate if too long
	if maxLength > 0 && len(stripped) > maxLength {
		stripped = stripped[:maxLength-3] + "..."
	}

	return stripped
}

func formatSynthesisOutput(data SynthesisData) string {
	const boxWidth = 60
	const maxItems = 5

	var lines []string

	// Box header
	content := fmt.Sprintf("  %s  │  %s-%s  │  %d days",
		data.Metadata.ProjectName,
		data.Metadata.StartDate,
		data.Metadata.EndDate,
		data.Metadata.Days)
	width := boxWidth
	if len(content)+4 > width {
		width = len(content) + 4
	}

	lines = append(lines, "╭"+strings.Repeat("─", width-2)+"╮")
	lines = append(lines, "│"+content+strings.Repeat(" ", width-2-len(content))+"│")
	lines = append(lines, "╰"+strings.Repeat("─", width-2)+"╯")

	// Summary line
	var parts []string
	if data.Metadata.CommitCount > 0 {
		parts = append(parts, fmt.Sprintf("%d commit%s", data.Metadata.CommitCount, pluralize(data.Metadata.CommitCount)))
	}
	if data.Metadata.IssueCount > 0 {
		parts = append(parts, fmt.Sprintf("%d issue%s closed", data.Metadata.IssueCount, pluralize(data.Metadata.IssueCount)))
	}
	if data.Metadata.InvestigationCount > 0 {
		parts = append(parts, fmt.Sprintf("%d investigation%s", data.Metadata.InvestigationCount, pluralize(data.Metadata.InvestigationCount)))
	}
	if len(parts) > 0 {
		lines = append(lines, "  "+strings.Join(parts, " · "))
		lines = append(lines, "")
	}

	// Sections
	sections := []struct {
		title string
		items []SynthesisItem
	}{
		{"Bugs Fixed", data.Bugs},
		{"Features Added", data.Features},
		{"Key Learnings", data.Learnings},
		{"Refactoring", data.Refactoring},
	}

	hasContent := false
	for _, section := range sections {
		if len(section.items) == 0 {
			continue
		}
		hasContent = true

		// Section header
		header := fmt.Sprintf("═══ %s (%d) ", section.title, len(section.items))
		padding := strings.Repeat("═", max(0, boxWidth-len(header)))
		lines = append(lines, header+padding)

		// Items
		count := len(section.items)
		if count > maxItems {
			count = maxItems
		}
		for _, item := range section.items[:count] {
			var text string
			switch item.Source {
			case "git":
				text = extractKeyPoint(item.Message, 55)
			case "beads":
				text = extractKeyPoint(item.Title, 55)
			case "investigation":
				text = item.TLDR
				if len(text) > 50 {
					text = text[:47] + "..."
				}
			}
			lines = append(lines, "  • "+text)
		}

		if len(section.items) > maxItems {
			lines = append(lines, fmt.Sprintf("  ... and %d more", len(section.items)-maxItems))
		}
		lines = append(lines, "")
	}

	if !hasContent {
		return "No activity found in the specified timeframe."
	}

	// Remove trailing empty line
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
