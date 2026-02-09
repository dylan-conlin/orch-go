package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

const (
	backlogRuleEmptyDescription    = "empty-description"
	backlogRulePlaceholderDesc     = "placeholder-description"
	backlogRuleMetadataTruncated   = "metadata-truncated-title"
	backlogRulePlaceholderTitle    = "placeholder-title"
	maxBacklogIssuePrintNonVerbose = 20
)

var (
	legacySpawnTitlePattern = regexp.MustCompile(`^\[[^\]]+\]\s+[a-z0-9][a-z0-9_-]*:\s+`)
	placeholderTokens       = map[string]struct{}{
		"tbd":         {},
		"todo":        {},
		"wip":         {},
		"fixme":       {},
		"placeholder": {},
		"unknown":     {},
		"n/a":         {},
		"na":          {},
		"none":        {},
	}
)

type BacklogHygieneViolation struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

type BacklogHygieneIssue struct {
	ID         string                    `json:"id"`
	Title      string                    `json:"title"`
	Status     string                    `json:"status"`
	Violations []BacklogHygieneViolation `json:"violations"`
}

type BacklogHygieneReport struct {
	Healthy      bool                  `json:"healthy"`
	CheckedCount int                   `json:"checked_count"`
	IssueCount   int                   `json:"issue_count"`
	RuleCounts   map[string]int        `json:"rule_counts"`
	Issues       []BacklogHygieneIssue `json:"issues"`
}

func runBacklogHygieneCheck() error {
	fmt.Println("orch doctor --backlog")
	fmt.Println("Checking recurring backlog issue hygiene...")
	fmt.Println()

	issues, err := listBacklogIssues()
	if err != nil {
		return fmt.Errorf("backlog hygiene check failed: %w", err)
	}

	report := evaluateBacklogHygiene(issues)
	printBacklogHygieneReport(report)
	return nil
}

func listBacklogIssues() ([]beads.Issue, error) {
	projectDir, err := currentProjectDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine project directory: %w", err)
	}

	previousDefaultDir := beads.DefaultDir
	beads.DefaultDir = projectDir
	defer func() {
		beads.DefaultDir = previousDefaultDir
	}()

	var issues []beads.Issue
	err = withBeadsFallback(projectDir, func(client *beads.Client) error {
		var rpcErr error
		issues, rpcErr = client.List(&beads.ListArgs{Limit: 0})
		return rpcErr
	}, func() error {
		var fallbackErr error
		issues, fallbackErr = beads.FallbackList("")
		return fallbackErr
	})
	if err != nil {
		return nil, err
	}

	return issues, nil
}

func evaluateBacklogHygiene(issues []beads.Issue) *BacklogHygieneReport {
	report := &BacklogHygieneReport{
		Healthy:    true,
		RuleCounts: make(map[string]int),
		Issues:     make([]BacklogHygieneIssue, 0),
	}

	for _, issue := range issues {
		if !isBacklogStatus(issue.Status) {
			continue
		}

		report.CheckedCount++
		violations := detectBacklogHygieneViolations(issue)
		if len(violations) == 0 {
			continue
		}

		report.Healthy = false
		report.IssueCount++
		for _, violation := range violations {
			report.RuleCounts[violation.Rule]++
		}

		report.Issues = append(report.Issues, BacklogHygieneIssue{
			ID:         issue.ID,
			Title:      issue.Title,
			Status:     issue.Status,
			Violations: violations,
		})
	}

	sort.Slice(report.Issues, func(i, j int) bool {
		return report.Issues[i].ID < report.Issues[j].ID
	})

	return report
}

func isBacklogStatus(status string) bool {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if normalized == "" {
		return true
	}

	return normalized != "closed" && normalized != "tombstone"
}

func detectBacklogHygieneViolations(issue beads.Issue) []BacklogHygieneViolation {
	violations := make([]BacklogHygieneViolation, 0, 4)
	title := strings.TrimSpace(issue.Title)
	description := strings.TrimSpace(issue.Description)

	if description == "" {
		violations = append(violations, BacklogHygieneViolation{
			Rule:    backlogRuleEmptyDescription,
			Message: "description is empty",
		})
	} else if isPlaceholderText(description) {
		violations = append(violations, BacklogHygieneViolation{
			Rule:    backlogRulePlaceholderDesc,
			Message: "description is placeholder text",
		})
	}

	if hasMetadataTruncatedTitle(title) {
		violations = append(violations, BacklogHygieneViolation{
			Rule:    backlogRuleMetadataTruncated,
			Message: "title matches metadata-prefixed truncated anti-pattern",
		})
	}

	if isPlaceholderText(title) {
		violations = append(violations, BacklogHygieneViolation{
			Rule:    backlogRulePlaceholderTitle,
			Message: "title is placeholder text",
		})
	}

	return violations
}

func hasMetadataTruncatedTitle(title string) bool {
	normalized := strings.TrimSpace(title)
	if !legacySpawnTitlePattern.MatchString(strings.ToLower(normalized)) {
		return false
	}

	return strings.HasSuffix(normalized, "...") || strings.HasSuffix(normalized, "…")
}

func isPlaceholderText(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.Trim(normalized, " .:-_")
	if normalized == "" {
		return false
	}

	_, ok := placeholderTokens[normalized]
	return ok
}

func printBacklogHygieneReport(report *BacklogHygieneReport) {
	fmt.Printf("Issues checked: %d\n", report.CheckedCount)
	fmt.Printf("Issues with hygiene defects: %d\n", report.IssueCount)

	if report.Healthy {
		fmt.Println()
		fmt.Println("✓ Backlog hygiene check passed")
		return
	}

	if len(report.RuleCounts) > 0 {
		rules := make([]string, 0, len(report.RuleCounts))
		for rule := range report.RuleCounts {
			rules = append(rules, rule)
		}
		sort.Strings(rules)

		fmt.Println()
		fmt.Println("Rule counts:")
		for _, rule := range rules {
			fmt.Printf("  - %s: %d\n", rule, report.RuleCounts[rule])
		}
	}

	limit := len(report.Issues)
	if !doctorVerbose && limit > maxBacklogIssuePrintNonVerbose {
		limit = maxBacklogIssuePrintNonVerbose
	}

	fmt.Println()
	fmt.Println("✗ Hygiene defects:")
	for i := 0; i < limit; i++ {
		issue := report.Issues[i]
		fmt.Printf("  - %s (%s)\n", issue.ID, issue.Status)
		if doctorVerbose {
			fmt.Printf("      Title: %s\n", issue.Title)
		}
		for _, violation := range issue.Violations {
			fmt.Printf("      • %s\n", violation.Message)
		}
		fmt.Printf("      Action: bd update %s --title \"...\" --description \"...\"\n", issue.ID)
	}

	if limit < len(report.Issues) {
		fmt.Printf("\n  ... %d more issues (use --verbose for full list)\n", len(report.Issues)-limit)
	}
}
