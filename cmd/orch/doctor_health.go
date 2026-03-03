package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/health"
)

var (
	doctorHealth     bool // Run health properties check with trend tracking
	doctorHealthJSON bool // Output health report as JSON
)

func init() {
	doctorCmd.Flags().BoolVar(&doctorHealth, "health", false, "Track system health invariants over time with trend analysis")
	doctorCmd.Flags().BoolVar(&doctorHealthJSON, "health-json", false, "Output health report as JSON")
}

// getHealthStore returns the store for health snapshots.
func getHealthStore() *health.Store {
	home, _ := os.UserHomeDir()
	return health.NewStore(filepath.Join(home, ".orch", "health-snapshots.jsonl"))
}

// runDoctorHealth collects health metrics, stores a snapshot, and displays trends.
func runDoctorHealth() error {
	store := getHealthStore()

	// Collect current metrics
	snap := collectHealthSnapshot()

	// Append to store
	if err := store.Append(snap); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save snapshot: %v\n", err)
	}

	// Read recent snapshots for trend analysis
	recent, err := store.ReadRecent(30)
	if err != nil {
		return fmt.Errorf("failed to read snapshots: %w", err)
	}

	report := health.GenerateReport(recent)

	if doctorHealthJSON {
		return outputHealthJSON(report)
	}

	return outputHealthText(report)
}

// collectHealthSnapshot gathers all health metrics from bd and git.
func collectHealthSnapshot() health.Snapshot {
	now := time.Now()
	snap := health.Snapshot{
		Timestamp: now,
	}

	// Collect issue metrics from bd
	snap.OpenIssues = bdCount("--status", "open")
	snap.BlockedIssues = bdCount("--status", "blocked")
	snap.StaleIssues = countStaleIssues()
	snap.OrphanedIssues = countOrphanedIssues()

	// Collect code metrics
	snap.BloatedFiles = countBloatedFiles()

	// Collect git commit metrics
	snap.FixCommits, snap.FeatCommits = countCommitTypes()
	if snap.FeatCommits > 0 {
		snap.FixFeatRatio = float64(snap.FixCommits) / float64(snap.FeatCommits)
	} else if snap.FixCommits > 0 {
		snap.FixFeatRatio = float64(snap.FixCommits) // All fixes, no features
	}

	return snap
}

// bdCount runs bd count with the given args and returns the count.
func bdCount(args ...string) int {
	cmdArgs := append([]string{"count"}, args...)
	cmd := exec.Command("bd", cmdArgs...)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return n
}

// countStaleIssues counts open issues with no updates in 14+ days.
func countStaleIssues() int {
	cutoff := time.Now().AddDate(0, 0, -14).Format("2006-01-02")
	return bdCount("--status", "open", "--updated-before", cutoff)
}

// countOrphanedIssues counts open issues with no updates in 7+ days.
func countOrphanedIssues() int {
	cutoff := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	return bdCount("--status", "open", "--updated-before", cutoff)
}

// countBloatedFiles counts source files exceeding 800 lines.
func countBloatedFiles() int {
	projectDir, err := os.Getwd()
	if err != nil {
		return 0
	}

	// Reuse existing hotspot bloat analysis
	_, count, err := analyzeBloatFiles(projectDir, 800)
	if err != nil {
		return 0
	}
	return count
}

// countCommitTypes counts fix: and feat: commits in the last 28 days.
func countCommitTypes() (fixes, feats int) {
	cmd := exec.Command("git", "log", "--since=28 days ago", "--pretty=format:%s")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	fixPattern := regexp.MustCompile(`(?i)^fix(\(.+\))?:`)
	featPattern := regexp.MustCompile(`(?i)^feat(\(.+\))?:`)

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if fixPattern.MatchString(line) {
			fixes++
		} else if featPattern.MatchString(line) {
			feats++
		}
	}

	return fixes, feats
}

// outputHealthJSON prints the health report as JSON.
func outputHealthJSON(report health.Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// outputHealthText prints the health report in human-readable format.
func outputHealthText(report health.Report) error {
	fmt.Println("orch doctor --health")
	fmt.Println("System Health Properties")
	fmt.Println("========================")
	fmt.Printf("Snapshots: %d | Generated: %s\n", report.SnapshotCount, report.GeneratedAt.Format("2006-01-02 15:04"))
	fmt.Println()

	c := report.Current
	t := report.Trends

	// Metric table
	fmt.Println("  Metric              Value    Trend")
	fmt.Println("  ──────────────────  ───────  ─────")
	fmt.Printf("  Open issues         %-7d  %s\n", c.OpenIssues, trendLabel(t.OpenIssues, false))
	fmt.Printf("  Blocked issues      %-7d  %s\n", c.BlockedIssues, trendLabel(t.BlockedIssues, false))
	fmt.Printf("  Stale issues (14d)  %-7d  %s\n", c.StaleIssues, trendLabel(t.StaleIssues, false))
	fmt.Printf("  Orphaned (7d)       %-7d  %s\n", c.OrphanedIssues, trendLabel(t.OrphanedIssues, false))
	fmt.Printf("  Bloated files       %-7d  %s\n", c.BloatedFiles, trendLabel(t.BloatedFiles, false))
	fmt.Printf("  Fix:feat ratio      %-7.1f  %s\n", c.FixFeatRatio, trendLabel(t.FixFeatRatio, false))
	fmt.Printf("  Fix commits (28d)   %-7d\n", c.FixCommits)
	fmt.Printf("  Feat commits (28d)  %-7d\n", c.FeatCommits)
	fmt.Println()

	// Alerts
	if len(report.Alerts) > 0 {
		fmt.Println("  Alerts:")
		for _, a := range report.Alerts {
			icon := "⚠️"
			if a.Level == "critical" {
				icon = "🚨"
			}
			fmt.Printf("  %s  %s\n", icon, a.Message)
		}
		fmt.Println()
	} else {
		fmt.Println("  No alerts — system health is nominal.")
		fmt.Println()
	}

	return nil
}

// trendLabel formats a trend direction with appropriate icon.
// lowerIsBetter inverts the color logic (e.g., for metrics where decrease is good).
func trendLabel(t health.Trend, lowerIsBetter bool) string {
	switch t {
	case health.TrendUp:
		if lowerIsBetter {
			return "↑ (worsening)"
		}
		return "↑"
	case health.TrendDown:
		if lowerIsBetter {
			return "↓ (improving)"
		}
		return "↓"
	default:
		return "→ stable"
	}
}
