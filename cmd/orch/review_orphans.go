// Package main provides the orphans pass for orch review.
// This surfaces closed architect designs that never got implementation follow-up,
// closing the gap where designs "fell through the cracks."
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	orphansCreateFollowUp bool
)

// OrphanItem represents a closed architect issue with no implementation follow-up.
type OrphanItem struct {
	ID            string
	Title         string
	Age           string // Human-readable age since closure
	ClosedAt      time.Time
	SynthesisTLDR string // From SYNTHESIS.md if workspace exists
}

var reviewOrphansCmd = &cobra.Command{
	Use:   "orphans",
	Short: "Surface architect designs with no implementation follow-up",
	Long: `Find closed architect issues that never generated implementation work.

Queries all closed issues with skill=architect, then checks if any issue
references them via title pattern "(from architect <id>)" or dependency edges.

Orphans are designs that completed but no one acted on — they fell through the cracks.

Use --create-follow-up to retroactively create implementation issues from
each orphan's synthesis (same logic as the auto-create in orch complete).

Examples:
  orch review orphans                   # List orphaned architect designs
  orch review orphans --create-follow-up  # Create implementation issues for each`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReviewOrphans(orphansCreateFollowUp)
	},
}

func init() {
	reviewOrphansCmd.Flags().BoolVar(&orphansCreateFollowUp, "create-follow-up", false, "Create implementation issues for each orphan (from synthesis)")
	reviewCmd.AddCommand(reviewOrphansCmd)
}

// orphanItemFromIssue converts a beads Issue to an OrphanItem.
func orphanItemFromIssue(issue beads.Issue) OrphanItem {
	item := OrphanItem{
		ID:    issue.ID,
		Title: issue.Title,
	}

	// Parse closed time for age calculation
	if issue.ClosedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, issue.ClosedAt); err == nil {
			item.ClosedAt = t
			item.Age = humanAge(time.Since(t))
		} else if t, err := time.Parse(time.RFC3339, issue.ClosedAt); err == nil {
			item.ClosedAt = t
			item.Age = humanAge(time.Since(t))
		}
	}

	// Fallback: use created_at if closed_at is empty
	if item.Age == "" && issue.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, issue.CreatedAt); err == nil {
			item.Age = humanAge(time.Since(t))
		} else if t, err := time.Parse(time.RFC3339, issue.CreatedAt); err == nil {
			item.Age = humanAge(time.Since(t))
		}
	}

	return item
}

// hasImplementationFollowUp checks if any issue references the given architect ID.
// Looks for the title pattern "(from architect <id>)" which is created by
// maybeAutoCreateImplementationIssue in complete_architect.go.
func hasImplementationFollowUp(architectID string, allIssues []beads.Issue) bool {
	pattern := strings.ToLower(fmt.Sprintf("from architect %s", architectID))
	for _, issue := range allIssues {
		if strings.Contains(strings.ToLower(issue.Title), pattern) {
			return true
		}
	}
	return false
}

// formatOrphansList formats orphan items for display.
func formatOrphansList(items []OrphanItem) string {
	if len(items) == 0 {
		return "No orphaned architect designs found.\n"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n## Orphaned Architect Designs (%d found)\n\n", len(items)))

	for _, item := range items {
		ageStr := ""
		if item.Age != "" {
			ageStr = fmt.Sprintf(" (%s ago)", item.Age)
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", item.ID, ageStr))

		// Truncate long titles
		title := item.Title
		if len(title) > 90 {
			title = title[:87] + "..."
		}
		b.WriteString(fmt.Sprintf("      %s\n", title))

		if item.SynthesisTLDR != "" {
			tldr := item.SynthesisTLDR
			if len(tldr) > 100 {
				tldr = tldr[:97] + "..."
			}
			b.WriteString(fmt.Sprintf("      TLDR: %s\n", tldr))
		}

		b.WriteString("      → No implementation issues found\n\n")
	}

	return b.String()
}

// isArchitectBeadsIssue returns true if a beads.Issue is an architect issue.
// Checks for skill:architect label or "architect:" in the title.
func isArchitectBeadsIssue(issue beads.Issue) bool {
	for _, label := range issue.Labels {
		if label == "skill:architect" {
			return true
		}
	}
	return strings.Contains(strings.ToLower(issue.Title), "architect:")
}

// getClosedArchitectIssues fetches all closed issues identified as architect work.
// Uses beads queries with server-side filtering when available (RPC client),
// with post-filtering as fallback for CLI client which doesn't support Labels/Title.
func getClosedArchitectIssues() ([]beads.Issue, error) {
	client, cleanup, err := getBeadsClient()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Query 1: closed issues with skill:architect label
	labelIssues, err := client.List(&beads.ListArgs{
		Status: "closed",
		Labels: []string{"skill:architect"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list closed architect issues: %w", err)
	}

	// Query 2: closed issues with "architect:" in title (for issues without label)
	titleIssues, err := client.List(&beads.ListArgs{
		Status: "closed",
		Title:  "architect:",
	})
	if err != nil {
		// Non-fatal: we still have label results
		titleIssues = nil
	}

	// Merge and deduplicate
	seen := make(map[string]bool)
	var merged []beads.Issue
	for _, issue := range labelIssues {
		seen[issue.ID] = true
		merged = append(merged, issue)
	}
	for _, issue := range titleIssues {
		if !seen[issue.ID] {
			merged = append(merged, issue)
		}
	}

	// Post-filter: CLI client doesn't support Labels/Title filtering,
	// so results may include non-architect issues. Filter them here.
	var filtered []beads.Issue
	for _, issue := range merged {
		if isArchitectBeadsIssue(issue) {
			filtered = append(filtered, issue)
		}
	}

	return filtered, nil
}

// getAllIssuesForReferenceCheck fetches issues to check for
// title-based references to architect issues.
// Uses targeted title query when available (RPC), post-filters for CLI fallback.
func getAllIssuesForReferenceCheck() ([]beads.Issue, error) {
	client, cleanup, err := getBeadsClient()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Fetch issues that contain "from architect" in the title.
	// This is more targeted than fetching ALL issues.
	issues, err := client.List(&beads.ListArgs{
		Title: "from architect",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list reference issues: %w", err)
	}

	// Post-filter: CLI client doesn't support Title filtering.
	var filtered []beads.Issue
	for _, issue := range issues {
		if strings.Contains(strings.ToLower(issue.Title), "from architect") {
			filtered = append(filtered, issue)
		}
	}

	return filtered, nil
}

// enrichWithSynthesisTLDR tries to find the workspace for an architect issue
// and extract the SYNTHESIS.md TLDR.
func enrichWithSynthesisTLDR(item *OrphanItem) {
	projectDir, err := os.Getwd()
	if err != nil {
		return
	}

	// Search both active and archived workspaces
	for _, subdir := range []string{"", "archived"} {
		workspaceDir := filepath.Join(projectDir, ".orch", "workspace", subdir)
		entries, err := os.ReadDir(workspaceDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			dirPath := filepath.Join(workspaceDir, entry.Name())

			// Check if this workspace's beads ID matches
			beadsID := extractBeadsIDFromWorkspace(dirPath)
			if beadsID != item.ID {
				continue
			}

			// Found the workspace - try to parse synthesis
			synthesis, err := verify.ParseSynthesis(dirPath)
			if err == nil && synthesis != nil && synthesis.TLDR != "" {
				item.SynthesisTLDR = synthesis.TLDR
			}
			return
		}
	}
}

// runReviewOrphans implements the orphans review pass.
func runReviewOrphans(createFollowUp bool) error {
	// Step 1: Get all closed architect issues
	architectIssues, err := getClosedArchitectIssues()
	if err != nil {
		return err
	}

	if len(architectIssues) == 0 {
		fmt.Println("No closed architect issues found.")
		return nil
	}

	// Step 2: Get all issues that reference architects (by title pattern)
	referenceIssues, err := getAllIssuesForReferenceCheck()
	if err != nil {
		// Non-fatal: show all as potential orphans with a warning
		fmt.Fprintf(os.Stderr, "Warning: could not check for reference issues: %v\n", err)
		referenceIssues = nil
	}

	// Step 3: Filter to orphans (no follow-up found)
	var orphans []OrphanItem
	for _, issue := range architectIssues {
		if !hasImplementationFollowUp(issue.ID, referenceIssues) {
			item := orphanItemFromIssue(issue)
			enrichWithSynthesisTLDR(&item)
			orphans = append(orphans, item)
		}
	}

	// Step 4: Display results
	fmt.Print(formatOrphansList(orphans))

	if len(orphans) == 0 {
		return nil
	}

	fmt.Printf("Total: %d closed architect designs, %d orphaned\n",
		len(architectIssues), len(orphans))

	// Step 5: Optionally create follow-up issues
	if createFollowUp {
		return createOrphanFollowUps(orphans)
	}

	fmt.Printf("\nTo create implementation issues: orch review orphans --create-follow-up\n")
	return nil
}

// createOrphanFollowUps creates implementation issues for each orphan
// using the same logic as complete_architect.go's auto-create.
func createOrphanFollowUps(orphans []OrphanItem) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	created := 0
	skipped := 0

	for _, orphan := range orphans {
		// Find workspace to get synthesis
		workspacePath, _ := findWorkspaceByBeadsID(projectDir, orphan.ID)

		// Also check archived workspaces
		if workspacePath == "" {
			archivedDir := filepath.Join(projectDir, ".orch", "workspace", "archived")
			entries, err := os.ReadDir(archivedDir)
			if err == nil {
				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}
					dirPath := filepath.Join(archivedDir, entry.Name())
					beadsID := extractBeadsIDFromWorkspace(dirPath)
					if beadsID == orphan.ID {
						workspacePath = dirPath
						break
					}
				}
			}
		}

		if workspacePath == "" {
			fmt.Printf("  %s: skipped (no workspace found)\n", orphan.ID)
			skipped++
			continue
		}

		// Use the same auto-create logic from complete_architect.go
		issueID := maybeAutoCreateImplementationIssue("architect", orphan.ID, workspacePath)
		if issueID != "" {
			created++
		} else {
			fmt.Printf("  %s: skipped (synthesis not actionable or missing)\n", orphan.ID)
			skipped++
		}
	}

	fmt.Printf("\n---\n")
	fmt.Printf("Created: %d implementation issues, Skipped: %d\n", created, skipped)

	return nil
}
