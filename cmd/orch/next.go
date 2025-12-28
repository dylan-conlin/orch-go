// Package main provides the next command for synthesized work recommendations.
package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

// ============================================================================
// Enhanced Next Command - Synthesize prioritized work recommendations
// ============================================================================

// RecommendationType categorizes the type of recommendation.
type RecommendationType string

const (
	// RecommendationBlocker indicates a critical blocker requiring immediate attention.
	RecommendationBlocker RecommendationType = "BLOCKER"

	// RecommendationFocus indicates focus-aligned ready work.
	RecommendationFocus RecommendationType = "FOCUS"

	// RecommendationMaintenance indicates high-value maintenance (recurring patterns).
	RecommendationMaintenance RecommendationType = "MAINTENANCE"

	// RecommendationBacklog indicates strategic backlog items.
	RecommendationBacklog RecommendationType = "BACKLOG"
)

// Recommendation represents a single prioritized work recommendation.
type Recommendation struct {
	Type        RecommendationType `json:"type"`
	Priority    int                `json:"priority"` // 1-5, where 1 is highest
	BeadsID     string             `json:"beads_id,omitempty"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Reason      string             `json:"reason"`    // Why this is recommended
	Command     string             `json:"command"`   // Suggested command to execute
	FocusMatch  bool               `json:"focus_match"` // True if aligned with current focus
}

// NextOutput represents the complete next command output.
type NextOutput struct {
	Focus           string            `json:"focus,omitempty"`
	FocusIssue      string            `json:"focus_issue,omitempty"`
	TotalReady      int               `json:"total_ready"`
	BlockerCount    int               `json:"blocker_count"`
	Recommendations []Recommendation  `json:"recommendations"`
	GeneratedAt     time.Time         `json:"generated_at"`
}

var (
	nextSynthJSON    bool
	nextSynthVerbose bool
	nextSynthLimit   int
)

// newNextSynthCmd creates the enhanced next command.
// Note: This replaces the basic next command in focus.go.
func newNextSynthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next",
		Short: "Synthesize prioritized work recommendations",
		Long: `Combine multiple signals into ranked 'what to work on next' recommendations.

Synthesizes these signals into prioritized recommendations:
  1. [BLOCKER]     Critical blockers (patterns showing active failures)
  2. [FOCUS]       Focus-aligned ready work
  3. [MAINTENANCE] High-value maintenance (recurring patterns worth fixing)
  4. [BACKLOG]     Strategic backlog items

Signals analyzed:
  - bd ready:      Available work with no blockers
  - orch patterns: Behavioral patterns (retries, gaps, failures)
  - orch focus:    Current north star alignment
  - Issue status:  Priority and recency

Examples:
  orch next              # Show prioritized recommendations
  orch next --json       # Output as JSON for scripting
  orch next --verbose    # Show detailed reasoning
  orch next --limit 10   # Show top 10 recommendations`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNextSynth()
		},
	}

	cmd.Flags().BoolVar(&nextSynthJSON, "json", false, "Output as JSON for scripting")
	cmd.Flags().BoolVarP(&nextSynthVerbose, "verbose", "v", false, "Show detailed reasoning")
	cmd.Flags().IntVarP(&nextSynthLimit, "limit", "n", 5, "Maximum recommendations to show")

	return cmd
}

func init() {
	// Replace the old nextCmd registration with the enhanced version
	// The old nextCmd is defined in focus.go but we override it here
	rootCmd.RemoveCommand(nextCmd)
	rootCmd.AddCommand(newNextSynthCmd())
}

func runNextSynth() error {
	output := NextOutput{
		Recommendations: []Recommendation{},
		GeneratedAt:     time.Now(),
	}

	// 1. Get current focus
	focusStore, err := focus.New("")
	if err == nil {
		if f := focusStore.Get(); f != nil {
			output.Focus = f.Goal
			output.FocusIssue = f.BeadsID
		}
	}

	// 2. Collect blockers from patterns
	blockers := collectBlockerRecommendations()
	output.Recommendations = append(output.Recommendations, blockers...)
	output.BlockerCount = len(blockers)

	// 3. Collect ready work from beads
	readyWork := collectReadyWorkRecommendations(output.Focus, output.FocusIssue)
	output.Recommendations = append(output.Recommendations, readyWork...)
	output.TotalReady = len(readyWork)

	// 4. Collect maintenance recommendations from patterns
	maintenance := collectMaintenanceRecommendations()
	output.Recommendations = append(output.Recommendations, maintenance...)

	// 5. Sort recommendations by priority
	sortRecommendations(output.Recommendations)

	// 6. Limit results
	if nextSynthLimit > 0 && len(output.Recommendations) > nextSynthLimit {
		output.Recommendations = output.Recommendations[:nextSynthLimit]
	}

	// Output
	if nextSynthJSON {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Human-readable output
	printNextOutput(output)
	return nil
}

// collectBlockerRecommendations finds blockers from patterns (failures, retries).
func collectBlockerRecommendations() []Recommendation {
	recommendations := []Recommendation{}

	// Get retry/failure patterns
	retryStats, err := verify.GetAllRetryPatterns()
	if err != nil || len(retryStats) == 0 {
		return recommendations
	}

	// Collect beads IDs for batch status check
	beadsIDs := make([]string, 0, len(retryStats))
	for _, stats := range retryStats {
		beadsIDs = append(beadsIDs, stats.BeadsID)
	}

	// Batch-fetch issue statuses to filter out closed issues
	issueMap, _ := verify.GetIssuesBatch(beadsIDs)

	for _, stats := range retryStats {
		// Skip closed issues
		if issue, ok := issueMap[stats.BeadsID]; ok {
			status := strings.ToLower(issue.Status)
			if status == "closed" || status == "deferred" || status == "tombstone" {
				continue
			}
		}

		// Only include critical failures as blockers
		if stats.IsPersistentFailure() {
			rec := Recommendation{
				Type:        RecommendationBlocker,
				Priority:    1,
				BeadsID:     stats.BeadsID,
				Title:       fmt.Sprintf("Persistent failure: %s", stats.BeadsID),
				Description: fmt.Sprintf("Failed %dx without success (%d abandoned)", stats.SpawnCount, stats.AbandonedCount),
				Reason:      "Issue has failed repeatedly - needs systematic debugging",
				Command:     fmt.Sprintf("orch spawn systematic-debugging --issue %s \"Debug persistent failure\"", stats.BeadsID),
			}

			// Add skill context if available
			if len(stats.Skills) > 0 {
				rec.Reason += fmt.Sprintf(" (tried: %s)", strings.Join(stats.Skills, ", "))
			}

			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

// collectReadyWorkRecommendations gets ready work from beads, prioritized by focus alignment.
func collectReadyWorkRecommendations(focusGoal, focusIssue string) []Recommendation {
	recommendations := []Recommendation{}

	// Try RPC client first, fall back to CLI
	var issues []beads.Issue
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()
			issues, _ = client.Ready(nil)
		}
	}
	if len(issues) == 0 {
		issues, _ = beads.FallbackReady()
	}

	if len(issues) == 0 {
		return recommendations
	}

	// Get retry patterns to annotate issues with retry history
	retryStats, _ := verify.GetAllRetryPatterns()
	retryMap := make(map[string]*verify.FixAttemptStats)
	for _, stats := range retryStats {
		retryMap[stats.BeadsID] = stats
	}

	for _, issue := range issues {
		rec := Recommendation{
			BeadsID:     issue.ID,
			Title:       issue.Title,
			Priority:    issue.Priority,
			Description: truncateDescription(issue.Description, 80),
		}

		// Determine type based on focus alignment
		if focusIssue != "" && issue.ID == focusIssue {
			rec.Type = RecommendationFocus
			rec.Priority = 1 // Focused issue gets highest priority
			rec.Reason = "Directly aligned with current focus"
			rec.FocusMatch = true
		} else if focusGoal != "" && matchesFocusGoal(issue, focusGoal) {
			rec.Type = RecommendationFocus
			rec.Reason = "Related to focus: " + focusGoal
			rec.FocusMatch = true
		} else {
			rec.Type = RecommendationBacklog
			rec.Reason = fmt.Sprintf("P%d ready work", issue.Priority)
		}

		// Generate command based on issue type
		skill := inferSkillFromIssue(issue)
		rec.Command = fmt.Sprintf("orch spawn %s --issue %s \"%s\"", skill, issue.ID, truncateDescription(issue.Title, 40))

		// Check for retry history and add warning
		if stats, ok := retryMap[issue.ID]; ok && stats.IsRetryPattern() {
			rec.Reason += fmt.Sprintf(" [WARNING: %d prior attempts]", stats.SpawnCount)
		}

		recommendations = append(recommendations, rec)
	}

	return recommendations
}

// collectMaintenanceRecommendations finds patterns worth addressing proactively.
func collectMaintenanceRecommendations() []Recommendation {
	recommendations := []Recommendation{}

	// Get recurring gaps from gap tracker
	tracker, err := spawn.LoadTracker()
	if err != nil {
		return recommendations
	}

	suggestions := tracker.FindRecurringGaps()
	for _, s := range suggestions {
		if s.Priority != "high" {
			continue // Only include high-priority gaps as maintenance
		}

		rec := Recommendation{
			Type:        RecommendationMaintenance,
			Priority:    3,
			Title:       fmt.Sprintf("Recurring gap: %q", s.Query),
			Description: fmt.Sprintf("Query has returned limited results %d times", s.Count),
			Reason:      s.Suggestion,
			Command:     s.Command,
		}

		recommendations = append(recommendations, rec)
	}

	return recommendations
}

// sortRecommendations sorts by type priority and then by issue priority.
func sortRecommendations(recs []Recommendation) {
	typePriority := map[RecommendationType]int{
		RecommendationBlocker:     0,
		RecommendationFocus:       1,
		RecommendationMaintenance: 2,
		RecommendationBacklog:     3,
	}

	sort.Slice(recs, func(i, j int) bool {
		// First by type priority
		if typePriority[recs[i].Type] != typePriority[recs[j].Type] {
			return typePriority[recs[i].Type] < typePriority[recs[j].Type]
		}
		// Then by issue priority (lower number = higher priority)
		return recs[i].Priority < recs[j].Priority
	})
}

// printNextOutput prints the recommendations in human-readable format.
func printNextOutput(output NextOutput) {
	if len(output.Recommendations) == 0 {
		fmt.Println("No work recommendations.")
		fmt.Println("\nTo get started:")
		fmt.Println("  - Run 'bd ready' to see available issues")
		fmt.Println("  - Run 'orch focus \"goal\"' to set a priority")
		return
	}

	// Header with focus context
	if output.Focus != "" {
		fmt.Printf("\nRECOMMENDED NEXT ACTIONS (aligned with focus: '%s')\n", output.Focus)
	} else {
		fmt.Println("\nRECOMMENDED NEXT ACTIONS")
	}
	fmt.Println(strings.Repeat("─", 70))

	// Print recommendations with type indicators
	for i, rec := range output.Recommendations {
		typeIcon := getTypeIcon(rec.Type)
		
		// Main recommendation line
		if rec.BeadsID != "" {
			fmt.Printf("%d. [%s] %s - %s\n", i+1, rec.Type, rec.BeadsID, rec.Title)
		} else {
			fmt.Printf("%d. [%s] %s\n", i+1, rec.Type, rec.Title)
		}

		// Show reason (always) and description (if verbose)
		fmt.Printf("   %s %s\n", typeIcon, rec.Reason)
		
		if nextSynthVerbose && rec.Description != "" {
			fmt.Printf("      %s\n", rec.Description)
		}

		// Show command suggestion
		if rec.Command != "" {
			fmt.Printf("   → %s\n", rec.Command)
		}

		fmt.Println()
	}

	// Summary footer
	fmt.Println(strings.Repeat("─", 70))
	var parts []string
	if output.BlockerCount > 0 {
		parts = append(parts, fmt.Sprintf("%d blockers", output.BlockerCount))
	}
	if output.TotalReady > 0 {
		parts = append(parts, fmt.Sprintf("%d ready", output.TotalReady))
	}
	if len(parts) > 0 {
		fmt.Printf("Summary: %s\n", strings.Join(parts, ", "))
	}
}

// getTypeIcon returns an icon for the recommendation type.
func getTypeIcon(t RecommendationType) string {
	switch t {
	case RecommendationBlocker:
		return "🚨"
	case RecommendationFocus:
		return "🎯"
	case RecommendationMaintenance:
		return "🔧"
	case RecommendationBacklog:
		return "📋"
	default:
		return "•"
	}
}

// matchesFocusGoal checks if an issue title/description relates to the focus goal.
func matchesFocusGoal(issue beads.Issue, focusGoal string) bool {
	// Simple keyword matching - could be enhanced with more sophisticated matching
	goalLower := strings.ToLower(focusGoal)
	titleLower := strings.ToLower(issue.Title)
	descLower := strings.ToLower(issue.Description)
	
	// Extract key words from goal (skip common words)
	keywords := extractKeywords(goalLower)
	
	for _, kw := range keywords {
		if strings.Contains(titleLower, kw) || strings.Contains(descLower, kw) {
			return true
		}
	}
	
	return false
}

// extractKeywords extracts meaningful keywords from a string.
func extractKeywords(s string) []string {
	// Skip common words
	skip := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"to": true, "for": true, "of": true, "in": true, "on": true,
		"with": true, "as": true, "by": true, "at": true, "from": true,
	}
	
	words := strings.Fields(s)
	keywords := make([]string, 0, len(words))
	
	for _, word := range words {
		word = strings.Trim(word, ".,!?\"'()[]")
		if len(word) > 2 && !skip[word] {
			keywords = append(keywords, word)
		}
	}
	
	return keywords
}

// inferSkillFromIssue infers the appropriate skill for an issue.
func inferSkillFromIssue(issue beads.Issue) string {
	switch issue.IssueType {
	case "bug":
		return "systematic-debugging"
	case "feature":
		return "feature-impl"
	case "task":
		return "feature-impl"
	case "investigation":
		return "investigation"
	default:
		return "feature-impl"
	}
}

// truncateDescription truncates a description to maxLen characters.
func truncateDescription(s string, maxLen int) string {
	// Remove newlines for display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
