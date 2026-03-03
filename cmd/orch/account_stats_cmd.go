// account_stats_cmd.go - Account spawn distribution stats from events.jsonl
package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

var accountStatsDays int

var accountStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show spawn distribution across accounts",
	Long: `Show how agent spawns are distributed across Claude accounts.

Reads session.spawned events from events.jsonl, groups by account name,
and shows counts + percentages for the configured time window.

Examples:
  orch account stats            # Last 7 days (default)
  orch account stats --days 1   # Last 24 hours
  orch account stats --days 30  # Last 30 days`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountStats(accountStatsDays)
	},
}

func init() {
	accountStatsCmd.Flags().IntVar(&accountStatsDays, "days", 7, "Number of days to analyze")
	accountCmd.AddCommand(accountStatsCmd)
}

// AccountStat holds per-account spawn stats.
type AccountStat struct {
	Name    string
	Count   int
	Percent float64
}

// AccountStatsReport holds the aggregated account stats.
type AccountStatsReport struct {
	Accounts []AccountStat
	Total    int
	Days     int
}

func aggregateAccountStats(events []StatsEvent, days int) AccountStatsReport {
	cutoff := time.Now().Unix() - int64(days)*24*3600
	counts := make(map[string]int)

	for _, event := range events {
		if event.Type != "session.spawned" {
			continue
		}
		if event.Timestamp < cutoff {
			continue
		}

		acct := "(unknown)"
		if data := event.Data; data != nil {
			if a, ok := data["account"].(string); ok && a != "" {
				acct = a
			}
		}
		counts[acct]++
	}

	total := 0
	for _, c := range counts {
		total += c
	}

	var accounts []AccountStat
	for name, count := range counts {
		pct := 0.0
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		accounts = append(accounts, AccountStat{
			Name:    name,
			Count:   count,
			Percent: pct,
		})
	}

	// Sort by count descending
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Count > accounts[j].Count
	})

	return AccountStatsReport{
		Accounts: accounts,
		Total:    total,
		Days:     days,
	}
}

func runAccountStats(days int) error {
	path := getEventsPath()
	events, err := parseEvents(path)
	if err != nil {
		return err
	}

	report := aggregateAccountStats(events, days)

	fmt.Printf("Account Spawn Distribution (last %d days)\n", days)
	fmt.Printf("%s\n", "─────────────────────────────────────────")

	if report.Total == 0 {
		fmt.Println("No spawns recorded in this time window.")
		return nil
	}

	fmt.Printf("\n%-15s %8s %8s\n", "ACCOUNT", "SPAWNS", "SHARE")
	fmt.Printf("%s\n", "─────────────────────────────────────")

	for _, a := range report.Accounts {
		fmt.Printf("%-15s %8d %7.1f%%\n", a.Name, a.Count, a.Percent)
	}

	fmt.Printf("%s\n", "─────────────────────────────────────")
	fmt.Printf("%-15s %8d %7.1f%%\n", "TOTAL", report.Total, 100.0)

	return nil
}
