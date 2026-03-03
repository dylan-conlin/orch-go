package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/notify"
	"github.com/spf13/cobra"
)

const (
	auditLabel     = "audit:deep-review"
	auditCount     = 2
	auditWindowStr = "168h" // 7 days
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Randomized completion audit selection",
	Long: `Randomly select completed agent issues for deep review.

Unpredictable selection prevents verification patterns from converging
around predictable heuristics (priority, area, etc.).`,
}

var auditSelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Randomly select completed issues for deep review",
	Long: `Select 2 completed issues from the last 7 days at random and label
them audit:deep-review. Sends a desktop notification when selection is made.

The randomness is the design feature — you cannot predict which completions
will be flagged, so verification depth cannot be gamed.`,
	RunE: runAuditSelect,
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show issues flagged for deep review",
	RunE:  runAuditList,
}

var auditInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install weekly launchd job for audit selection",
	Long: `Install a macOS launchd job that runs orch audit select every Monday
at 9:00 AM. The job runs in the orch-go project directory.`,
	RunE: runAuditInstall,
}

var auditUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the weekly audit launchd job",
	RunE:  runAuditUninstall,
}

func init() {
	auditCmd.AddCommand(auditSelectCmd)
	auditCmd.AddCommand(auditListCmd)
	auditCmd.AddCommand(auditInstallCmd)
	auditCmd.AddCommand(auditUninstallCmd)
}

// recentClosedIssues returns issues closed within the given duration.
func recentClosedIssues(window time.Duration) ([]beads.Issue, error) {
	issues, err := beads.FallbackList("closed")
	if err != nil {
		return nil, fmt.Errorf("listing closed issues: %w", err)
	}

	cutoff := time.Now().Add(-window)
	var recent []beads.Issue
	for _, issue := range issues {
		if issue.ClosedAt == "" {
			continue
		}
		closedAt, err := time.Parse(time.RFC3339Nano, issue.ClosedAt)
		if err != nil {
			// Try RFC3339 without nanoseconds
			closedAt, err = time.Parse(time.RFC3339, issue.ClosedAt)
			if err != nil {
				continue
			}
		}
		if closedAt.After(cutoff) {
			// Skip issues already labeled for audit
			if hasLabel(issue.Labels, auditLabel) {
				continue
			}
			recent = append(recent, issue)
		}
	}
	return recent, nil
}

// hasLabel checks if a label exists in a slice.
func hasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if l == target {
			return true
		}
	}
	return false
}

// cryptoRandSelection picks n items from a slice using crypto/rand.
// Returns up to n items (fewer if pool is smaller).
func cryptoRandSelection(issues []beads.Issue, n int) ([]beads.Issue, error) {
	if len(issues) <= n {
		return issues, nil
	}

	// Fisher-Yates shuffle with crypto/rand, then take first n
	pool := make([]beads.Issue, len(issues))
	copy(pool, issues)

	for i := len(pool) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return nil, fmt.Errorf("crypto rand: %w", err)
		}
		pool[i], pool[j.Int64()] = pool[j.Int64()], pool[i]
	}

	return pool[:n], nil
}

func runAuditSelect(cmd *cobra.Command, args []string) error {
	window, _ := time.ParseDuration(auditWindowStr)

	recent, err := recentClosedIssues(window)
	if err != nil {
		return err
	}

	if len(recent) == 0 {
		fmt.Println("No eligible closed issues in the last 7 days.")
		return nil
	}

	selected, err := cryptoRandSelection(recent, auditCount)
	if err != nil {
		return fmt.Errorf("selecting issues: %w", err)
	}

	fmt.Printf("Selected %d issue(s) for deep review:\n\n", len(selected))

	var ids []string
	for _, issue := range selected {
		// Label the issue
		if err := beads.FallbackAddLabel(issue.ID, auditLabel); err != nil {
			fmt.Printf("  WARNING: failed to label %s: %v\n", issue.ID, err)
			continue
		}
		fmt.Printf("  %s [P%d] [%s] %s\n", issue.ID, issue.Priority, issue.IssueType, issue.Title)
		ids = append(ids, issue.ID)
	}

	// Send notification
	if len(ids) > 0 {
		n := notify.Default()
		msg := fmt.Sprintf("Deep review: %s", strings.Join(ids, ", "))
		_ = n.Send("Audit Selection", msg)
	}

	fmt.Printf("\nLabeled with '%s'. Review with: orch audit list\n", auditLabel)
	return nil
}

const auditLaunchdLabel = "com.orch.audit-select"

// auditPlistContent generates the launchd plist XML for weekly audit selection.
// Runs every Monday at 9:00 AM.
func auditPlistContent(orchPath, projectDir, logPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>audit</string>
        <string>select</string>
    </array>
    <key>WorkingDirectory</key>
    <string>%s</string>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Weekday</key>
        <integer>1</integer>
        <key>Hour</key>
        <integer>9</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/Users/dylanconlin/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/Users/dylanconlin/go/bin</string>
    </dict>
</dict>
</plist>`, auditLaunchdLabel, orchPath, projectDir, logPath, logPath)
}

func runAuditInstall(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	orchPath := filepath.Join(home, "bin", "orch")
	if _, err := os.Stat(orchPath); os.IsNotExist(err) {
		return fmt.Errorf("orch binary not found at %s — run 'make install' first", orchPath)
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	logPath := filepath.Join(home, ".orch", "audit-select.log")
	plistPath := filepath.Join(home, "Library", "LaunchAgents", auditLaunchdLabel+".plist")

	content := auditPlistContent(orchPath, projectDir, logPath)

	if err := os.WriteFile(plistPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}

	// Load the job
	loadCmd := exec.Command("launchctl", "load", plistPath)
	if output, err := loadCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("loading launchd job: %w: %s", err, string(output))
	}

	fmt.Printf("Installed weekly audit selection job.\n")
	fmt.Printf("  Plist: %s\n", plistPath)
	fmt.Printf("  Log:   %s\n", logPath)
	fmt.Printf("  Schedule: Every Monday at 9:00 AM\n")
	fmt.Printf("  Command: orch audit select\n")
	return nil
}

func runAuditUninstall(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	plistPath := filepath.Join(home, "Library", "LaunchAgents", auditLaunchdLabel+".plist")

	// Unload if loaded
	unloadCmd := exec.Command("launchctl", "unload", plistPath)
	_ = unloadCmd.Run() // Ignore error if not loaded

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist: %w", err)
	}

	fmt.Println("Uninstalled weekly audit selection job.")
	return nil
}

func runAuditList(cmd *cobra.Command, args []string) error {
	issues, err := beads.FallbackListWithLabel(auditLabel)
	if err != nil {
		return fmt.Errorf("listing audit issues: %w", err)
	}

	if len(issues) == 0 {
		fmt.Println("No issues currently flagged for deep review.")
		return nil
	}

	fmt.Printf("Issues flagged for deep review (%s):\n\n", auditLabel)
	for _, issue := range issues {
		status := issue.Status
		if issue.ClosedAt != "" {
			status = "closed"
		}
		fmt.Printf("  %s [P%d] [%s] %s (%s)\n", issue.ID, issue.Priority, issue.IssueType, issue.Title, status)
	}

	fmt.Printf("\nTotal: %d issue(s)\n", len(issues))
	fmt.Println("\nAfter review, remove label: bd update <id> --remove-label audit:deep-review")
	return nil
}
