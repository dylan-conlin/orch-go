package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/friction"
	"github.com/spf13/cobra"
)

var (
	frictionDays   int
	frictionJSON   bool
	frictionDetail bool
)

var frictionCmd = &cobra.Command{
	Use:   "friction",
	Short: "Show aggregate friction metrics from agent sessions",
	Long: `Parse Friction: comments from .beads/issues.jsonl and surface aggregate metrics.

Shows:
  - Category breakdown (tooling/ceremony/bug/gap/capacity) with counts
  - Top recurring friction sources (clustered by pattern)
  - Per-skill friction rates (which skills hit friction most?)
  - Weekly trend (friction rate over time)

Examples:
  orch friction                # All time
  orch friction --days 30      # Last 30 days
  orch friction --detail       # Include full friction message list
  orch friction --json         # Machine-readable output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		emitCommandInvoked("friction", flagsFromCmd(cmd)...)
		return runFriction()
	},
}

func init() {
	frictionCmd.Flags().IntVar(&frictionDays, "days", 0, "Number of days to analyze (0 = all time)")
	frictionCmd.Flags().BoolVar(&frictionJSON, "json", false, "Output as JSON")
	frictionCmd.Flags().BoolVar(&frictionDetail, "detail", false, "Include full friction message list")
	rootCmd.AddCommand(frictionCmd)
}

func runFriction() error {
	beadsPath := findBeadsIssuesJSONL()
	if beadsPath == "" {
		return fmt.Errorf(".beads/issues.jsonl not found — run from a project with beads tracking")
	}

	var since time.Time
	if frictionDays > 0 {
		since = time.Now().AddDate(0, 0, -frictionDays)
	}

	entries, noneBySkill, err := friction.ParseJSONLFull(beadsPath, since)
	if err != nil {
		return err
	}

	noneCount := 0
	for _, n := range noneBySkill {
		noneCount += n
	}

	report := friction.Aggregate(entries, noneCount, frictionDays)
	report.SkillRates = friction.ComputeSkillRatesWithNone(entries, noneBySkill)

	if frictionDetail {
		report.Entries = entries
	}

	if frictionJSON {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Print(formatFrictionText(report))
	return nil
}

func findBeadsIssuesJSONL() string {
	// Check current directory first
	path := filepath.Join(".beads", "issues.jsonl")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	// Walk up to find .beads/ in parent dirs
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ".beads", "issues.jsonl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func formatFrictionText(r *friction.Report) string {
	var b strings.Builder

	period := "all time"
	if r.Days > 0 {
		period = fmt.Sprintf("last %d days", r.Days)
	}
	fmt.Fprintf(&b, "═══ FRICTION REPORT (%s) ═══\n\n", period)

	// Summary
	fmt.Fprintf(&b, "SUMMARY\n")
	fmt.Fprintf(&b, "  Total friction comments: %d (of %d sessions)\n", r.FrictionCount, r.TotalComments)
	fmt.Fprintf(&b, "  Friction rate:           %.1f%%\n", r.FrictionRate*100)
	fmt.Fprintf(&b, "  Issues with friction:    %d\n", r.TotalIssues)
	fmt.Fprintln(&b)

	// Category breakdown
	fmt.Fprintf(&b, "CATEGORIES\n")
	if len(r.Categories) == 0 {
		fmt.Fprintf(&b, "  No friction recorded.\n")
	}
	for _, c := range r.Categories {
		fmt.Fprintf(&b, "  %-12s %3d  (%5.1f%%)\n", c.Category, c.Count, c.Percentage)
	}
	fmt.Fprintln(&b)

	// Top sources
	if len(r.TopSources) > 0 {
		fmt.Fprintf(&b, "TOP RECURRING SOURCES\n")
		for i, s := range r.TopSources {
			fmt.Fprintf(&b, "  %d. %s (%dx)\n", i+1, s.Pattern, s.Count)
			fmt.Fprintf(&b, "     e.g. %s\n", s.Example)
		}
		fmt.Fprintln(&b)
	}

	// Per-skill rates
	if len(r.SkillRates) > 0 {
		fmt.Fprintf(&b, "PER-SKILL FRICTION\n")
		for _, s := range r.SkillRates {
			rateStr := "—"
			if s.Total > 0 {
				rateStr = fmt.Sprintf("%.0f%%", s.Rate*100)
			}
			fmt.Fprintf(&b, "  %-12s %3d friction / %3d total  (%s)\n",
				s.Skill, s.FrictionCount, s.Total, rateStr)
		}
		fmt.Fprintln(&b)
	}

	// Weekly trend
	if len(r.WeeklyTrend) > 0 {
		fmt.Fprintf(&b, "WEEKLY TREND\n")
		for _, w := range r.WeeklyTrend {
			bar := strings.Repeat("█", w.FrictionCount)
			fmt.Fprintf(&b, "  %s  %2d  %s\n", w.Week, w.FrictionCount, bar)
		}
		fmt.Fprintln(&b)
	}

	// Detail
	if len(r.Entries) > 0 {
		fmt.Fprintf(&b, "ALL FRICTION ENTRIES (%d)\n", len(r.Entries))
		for _, e := range r.Entries {
			fmt.Fprintf(&b, "  %s  %-8s  %-10s  %s\n",
				e.CreatedAt.Format("2006-01-02"),
				e.Category,
				e.Skill,
				truncateFriction(e.Message, 80))
		}
		fmt.Fprintln(&b)
	}

	return b.String()
}

func truncateFriction(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
