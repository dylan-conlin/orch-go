// history.go - Show agent history with skill usage analytics
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	// History command flags
	historyDays       int
	historyJSONOutput bool
	historyProject    string

	// Pre-compiled regex patterns for history.go
	regexSkillGuidance = regexp.MustCompile(`## SKILL GUIDANCE \(([a-z0-9-]+)\)`)
	regexSkillField    = regexp.MustCompile(`(?:^|\n)(?:\*\*|\-\s+)?[Ss]kill:(?:\*\*)?\s*([a-z0-9-]+)`)
	regexSkillUsing    = regexp.MustCompile(`(?i)Using\s+([a-z0-9-]+)\s+skill:`)
	regexPhaseField    = regexp.MustCompile(`(?m)^\*\*Phase:\*\*\s*(\w+)`)
	regexStartedField  = regexp.MustCompile(`(?m)^\*\*Started:\*\*\s*(\d{4}-\d{2}-\d{2})`)
	regexLastUpdated   = regexp.MustCompile(`(?m)^\*\*Last Updated:\*\*\s*(\d{4}-\d{2}-\d{2})`)
)

var historyCmd = &cobra.Command{
	Use:    "history",
	Short:  "Show agent history with skill usage analytics",
	Hidden: true,
	Long: `Show agent history with skill usage analytics.

Analyzes workspace files to extract:
- Skill usage patterns
- Success rates by skill
- Adoption metrics

Examples:
  orch history                  # Show last 30 days
  orch history --days 7         # Show last 7 days
  orch history --json           # Output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHistory()
	},
}

func init() {
	historyCmd.Flags().IntVar(&historyDays, "days", 30, "Number of days to analyze")
	historyCmd.Flags().BoolVar(&historyJSONOutput, "json", false, "Output as JSON")
	historyCmd.Flags().StringVar(&historyProject, "project", "", "Project directory (default: current dir)")

	rootCmd.AddCommand(historyCmd)
}

// SkillUsage represents usage of a skill in a workspace.
type SkillUsage struct {
	SkillName     string     `json:"skill_name"`
	WorkspaceName string     `json:"workspace_name"`
	WorkspacePath string     `json:"workspace_path"`
	Phase         string     `json:"phase,omitempty"`
	Started       *time.Time `json:"started,omitempty"`
	Completed     *time.Time `json:"completed,omitempty"`
	Success       bool       `json:"success"`
}

// SkillStats represents aggregated statistics for a skill.
type SkillStats struct {
	SkillName      string   `json:"name"`
	TotalUses      int      `json:"total_uses"`
	SuccessfulUses int      `json:"successful_uses"`
	FailedUses     int      `json:"failed_uses"`
	SuccessRate    float64  `json:"success_rate"`
	Workspaces     []string `json:"workspaces"`
}

// SkillAnalytics represents complete skill usage analytics.
type SkillAnalytics struct {
	TotalWorkspaces         int          `json:"total_workspaces"`
	WorkspacesWithSkills    int          `json:"workspaces_with_skills"`
	WorkspacesWithoutSkills int          `json:"workspaces_without_skills"`
	SkillAdoptionRate       float64      `json:"skill_adoption_rate"`
	DateRange               *DateRange   `json:"date_range,omitempty"`
	Skills                  []SkillStats `json:"skills"`
}

// DateRange represents a date range.
type DateRange struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

func runHistory() error {
	// Determine project directory
	projectDir := historyProject
	if projectDir == "" {
		var err error
		projectDir, err = currentProjectDir()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Analyze skill usage
	analytics := analyzeSkillUsage(projectDir, historyDays)

	if historyJSONOutput {
		data, err := json.MarshalIndent(analytics, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Format human-readable output
	output := formatSkillAnalytics(analytics)
	fmt.Println(output)
	return nil
}

func analyzeSkillUsage(projectDir string, days int) *SkillAnalytics {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")

	// Scan for skill usages
	usages := scanWorkspacesForSkills(workspaceDir, days)

	// Aggregate statistics
	statsBySkill := aggregateSkillStats(usages)

	// Count total workspaces
	totalWorkspaces := 0
	if entries, err := os.ReadDir(workspaceDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				totalWorkspaces++
			}
		}
	}

	workspacesWithSkills := len(usages)
	workspacesWithoutSkills := totalWorkspaces - workspacesWithSkills

	// Calculate adoption rate
	adoptionRate := 0.0
	if totalWorkspaces > 0 {
		adoptionRate = float64(workspacesWithSkills) / float64(totalWorkspaces) * 100
	}

	// Calculate date range
	var dateRange *DateRange
	var minDate, maxDate time.Time
	for _, usage := range usages {
		if usage.Started != nil {
			if minDate.IsZero() || usage.Started.Before(minDate) {
				minDate = *usage.Started
			}
			if maxDate.IsZero() || usage.Started.After(maxDate) {
				maxDate = *usage.Started
			}
		}
	}
	if !minDate.IsZero() {
		dateRange = &DateRange{
			Start: minDate.Format("2006-01-02"),
			End:   maxDate.Format("2006-01-02"),
		}
	}

	// Convert stats map to sorted slice
	var skillStats []SkillStats
	for _, stats := range statsBySkill {
		skillStats = append(skillStats, stats)
	}
	sort.Slice(skillStats, func(i, j int) bool {
		return skillStats[i].TotalUses > skillStats[j].TotalUses
	})

	return &SkillAnalytics{
		TotalWorkspaces:         totalWorkspaces,
		WorkspacesWithSkills:    workspacesWithSkills,
		WorkspacesWithoutSkills: workspacesWithoutSkills,
		SkillAdoptionRate:       adoptionRate,
		DateRange:               dateRange,
		Skills:                  skillStats,
	}
}

func scanWorkspacesForSkills(workspaceDir string, days int) []SkillUsage {
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		return nil
	}

	var usages []SkillUsage
	cutoffDate := time.Now().AddDate(0, 0, -days)

	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		workspacePath := filepath.Join(workspaceDir, entry.Name())
		usage := extractSkillFromWorkspace(workspacePath, entry.Name())

		if usage != nil {
			// Apply date filter
			if usage.Started != nil && usage.Started.Before(cutoffDate) {
				continue
			}
			usages = append(usages, *usage)
		}
	}

	return usages
}

func extractSkillFromWorkspace(workspacePath, workspaceName string) *SkillUsage {
	// Try both WORKSPACE.md (legacy) and SPAWN_CONTEXT.md
	var content []byte
	var err error

	spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	workspaceFilePath := filepath.Join(workspacePath, "WORKSPACE.md")

	if _, err = os.Stat(spawnContextPath); err == nil {
		content, err = os.ReadFile(spawnContextPath)
	} else if _, err = os.Stat(workspaceFilePath); err == nil {
		content, err = os.ReadFile(workspaceFilePath)
	} else {
		return nil
	}

	if err != nil {
		return nil
	}

	contentStr := string(content)

	// Extract skill name - multiple patterns
	var skillName string

	// Pattern 1: "## SKILL GUIDANCE (skill-name)"
	if match := regexSkillGuidance.FindStringSubmatch(contentStr); match != nil {
		skillName = match[1]
	}

	// Pattern 2: "**Skill:**", "- Skill:", "Skill:"
	if skillName == "" {
		if match := regexSkillField.FindStringSubmatch(contentStr); match != nil {
			skillName = match[1]
		}
	}

	// Pattern 3: "Using skill-name skill:"
	if skillName == "" {
		if match := regexSkillUsing.FindStringSubmatch(contentStr); match != nil {
			skillName = match[1]
		}
	}

	if skillName == "" {
		return nil
	}

	// Extract phase
	var phase string
	if match := regexPhaseField.FindStringSubmatch(contentStr); match != nil {
		phase = match[1]
	}

	// Extract started date
	var started *time.Time
	if match := regexStartedField.FindStringSubmatch(contentStr); match != nil {
		if t, err := time.Parse("2006-01-02", match[1]); err == nil {
			started = &t
		}
	}

	// Check success (Phase: Complete)
	success := strings.EqualFold(phase, "complete")

	// Extract completion date
	var completed *time.Time
	if success {
		if match := regexLastUpdated.FindStringSubmatch(contentStr); match != nil {
			if t, err := time.Parse("2006-01-02", match[1]); err == nil {
				completed = &t
			}
		}
	}

	return &SkillUsage{
		SkillName:     skillName,
		WorkspaceName: workspaceName,
		WorkspacePath: workspacePath,
		Phase:         phase,
		Started:       started,
		Completed:     completed,
		Success:       success,
	}
}

func aggregateSkillStats(usages []SkillUsage) map[string]SkillStats {
	statsMap := make(map[string]SkillStats)

	for _, usage := range usages {
		stats, ok := statsMap[usage.SkillName]
		if !ok {
			stats = SkillStats{SkillName: usage.SkillName}
		}

		stats.TotalUses++
		stats.Workspaces = append(stats.Workspaces, usage.WorkspaceName)

		if usage.Success {
			stats.SuccessfulUses++
		} else {
			stats.FailedUses++
		}

		if stats.TotalUses > 0 {
			stats.SuccessRate = float64(stats.SuccessfulUses) / float64(stats.TotalUses) * 100
		}

		statsMap[usage.SkillName] = stats
	}

	return statsMap
}

func formatSkillAnalytics(analytics *SkillAnalytics) string {
	var lines []string

	// Header
	lines = append(lines, "")
	lines = append(lines, strings.Repeat("=", 70))
	lines = append(lines, "📊 SPAWNED AGENT SKILL USAGE (Workspace Markers)")
	lines = append(lines, strings.Repeat("=", 70))
	lines = append(lines, "")
	lines = append(lines, "💡 Context: These are spawned agents created via 'orch spawn {skill}'.")
	lines = append(lines, "   Skills are embedded in SPAWN_CONTEXT.md (not invoked via Skill tool).")
	lines = append(lines, "")
	lines = append(lines, "   Note: Low markers may indicate agents not documenting skills in workspaces,")
	lines = append(lines, "   NOT that skills aren't being used. See --include-transcripts for interactive")
	lines = append(lines, "   Skill tool usage.")

	// Date range
	if analytics.DateRange != nil {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("📅 Date Range: %s to %s", analytics.DateRange.Start, analytics.DateRange.End))
	}

	// Overall statistics
	lines = append(lines, "")
	lines = append(lines, "📈 Overall Statistics:")
	lines = append(lines, fmt.Sprintf("  Total workspaces: %d", analytics.TotalWorkspaces))
	lines = append(lines, fmt.Sprintf("  Workspaces with skill markers: %d", analytics.WorkspacesWithSkills))
	lines = append(lines, fmt.Sprintf("  Workspaces without skill markers: %d", analytics.WorkspacesWithoutSkills))
	lines = append(lines, fmt.Sprintf("  Skill marker rate: %.1f%% (documentation, not usage)", analytics.SkillAdoptionRate))

	// Skill breakdown
	if len(analytics.Skills) > 0 {
		lines = append(lines, "")
		lines = append(lines, "🎯 Skill Usage Breakdown:")

		for _, stats := range analytics.Skills {
			successRateStr := "N/A"
			if stats.TotalUses > 0 {
				successRateStr = fmt.Sprintf("%.0f%% success", stats.SuccessRate)
			}

			usesStr := "use"
			if stats.TotalUses != 1 {
				usesStr = "uses"
			}
			lines = append(lines, fmt.Sprintf("  • %s: %d %s (%s)",
				stats.SkillName, stats.TotalUses, usesStr, successRateStr))

			// Show workspace list if not too many
			if len(stats.Workspaces) <= 5 {
				for _, workspace := range stats.Workspaces {
					lines = append(lines, fmt.Sprintf("      - %s", workspace))
				}
			}
		}
	} else {
		lines = append(lines, "")
		lines = append(lines, "⚠️  No skill usage found in the specified time period.")
	}

	// Pattern gaps section
	if analytics.WorkspacesWithoutSkills > 0 {
		lines = append(lines, "")
		lines = append(lines, "⚡ Documentation Gap (Not Usage Gap):")
		lines = append(lines, fmt.Sprintf("  %d workspace(s) without skill markers in WORKSPACE.md", analytics.WorkspacesWithoutSkills))
		lines = append(lines, "  This indicates missing documentation, not that skills weren't used.")
		lines = append(lines, "  Most spawned agents have skills embedded but don't document in workspace.")
	}

	lines = append(lines, "")
	lines = append(lines, strings.Repeat("=", 70))

	return strings.Join(lines, "\n")
}
