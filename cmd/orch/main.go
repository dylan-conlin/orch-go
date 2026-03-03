// Package main provides the CLI entry point for orch-go.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	serverURL string

	// Version information (set at build time via ldflags)
	version   = "dev"
	buildTime = "unknown"
	sourceDir = "unknown" // Absolute path to source directory
	gitHash   = "unknown" // Full git commit hash at build time
)

func main() {
	// Check if binary is stale and auto-rebuild if needed
	// This may replace the current process via syscall.Exec
	maybeAutoRebuild()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "orch-go",
	Short: "OpenCode orchestration CLI",
	Long: `orch-go is a CLI tool for orchestrating OpenCode sessions.

It provides commands for spawning new sessions, sending messages to existing
sessions, and monitoring session events via SSE.`,
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://127.0.0.1:4096", "OpenCode server URL")

	rootCmd.AddCommand(spawnCmd)
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(completeCmd)
	rootCmd.AddCommand(workCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(tailCmd)
	rootCmd.AddCommand(questionCmd)
	rootCmd.AddCommand(abandonCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(waitCmd)
	rootCmd.AddCommand(focusCmd)
	rootCmd.AddCommand(driftCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(reworkCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(portCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(retriesCmd)
	rootCmd.AddCommand(backlogCmd)
	rootCmd.AddCommand(orientCmd)
	rootCmd.AddCommand(hookCmd)
	rootCmd.AddCommand(debriefCmd)
	rootCmd.AddCommand(controlCmd)
	rootCmd.AddCommand(automationCmd)
}

var (
	versionSource bool // Show source info and staleness check
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long: `Print version information.

Use --source to see where the binary was built from and check if it's stale.`,
	Run: func(cmd *cobra.Command, args []string) {
		if versionSource {
			runVersionSource()
			return
		}
		fmt.Printf("orch version %s\n", version)
		fmt.Printf("build time: %s\n", buildTime)
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionSource, "source", false, "Show source location and staleness check")
}

// runVersionSource shows where the binary was built from and checks staleness.
func runVersionSource() {
	fmt.Printf("orch version %s\n", version)
	fmt.Printf("build time:  %s\n", buildTime)
	fmt.Printf("source dir:  %s\n", sourceDir)
	fmt.Printf("git hash:    %s\n", gitHash)

	// Check if source directory exists
	if sourceDir == "unknown" {
		fmt.Println("\n⚠️  Source directory not embedded (dev build)")
		return
	}

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		fmt.Printf("\n⚠️  Source directory not found: %s\n", sourceDir)
		return
	}

	// Check current git hash in source directory
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = sourceDir
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("\n⚠️  Could not get current git hash: %v\n", err)
		return
	}

	currentHash := strings.TrimSpace(string(output))

	// Compare hashes
	if gitHash == "unknown" {
		fmt.Println("\n⚠️  Git hash not embedded (dev build)")
		fmt.Printf("current HEAD: %s\n", shortID(currentHash))
	} else if currentHash == gitHash {
		fmt.Println("\nstatus: ✓ UP TO DATE")
	} else {
		fmt.Println("\nstatus: ⚠️  STALE")
		fmt.Printf("binary hash:  %s\n", shortID(gitHash))
		fmt.Printf("current HEAD: %s\n", shortID(currentHash))
		fmt.Printf("\nrebuild: cd %s && make install\n", sourceDir)
	}
}

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor SSE events for session completion",
	Long:  "Monitor the OpenCode server for session events and send notifications on completion.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMonitor(serverURL)
	},
}

func runMonitor(serverURL string) error {
	// Use the new CompletionService which handles:
	// - SSE monitoring with automatic reconnection
	// - Desktop notifications
	// - Registry updates
	// - Beads phase updates
	service, err := opencode.NewCompletionService(serverURL)
	if err != nil {
		return fmt.Errorf("failed to create completion service: %w", err)
	}

	fmt.Printf("Monitoring SSE events at %s/event...\n", serverURL)
	fmt.Println("On session completion:")
	fmt.Println("  - Desktop notification sent")
	fmt.Println("  - Registry updated")
	fmt.Println("  - Beads phase updated (if applicable)")
	fmt.Println("Press Ctrl+C to stop")

	service.Start()

	// Block forever - the user will Ctrl+C to stop
	select {}
}

// ============================================================================
// Usage Tracking
// ============================================================================

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show Claude Max usage for all accounts",
	Long: `Show Claude Max usage for all saved accounts side-by-side.

Fetches live capacity data for each account in ~/.orch/accounts.yaml
and recommends which account to switch to if the current one is low.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUsage()
	},
}

func init() {
	rootCmd.AddCommand(usageCmd)
}

func runUsage() error {
	accounts, err := account.ListAccountsWithCapacity()
	if err != nil {
		return fmt.Errorf("failed to fetch account capacity: %w", err)
	}

	if len(accounts) == 0 {
		fmt.Println("No saved accounts. Run: orch account add <name>")
		return nil
	}

	// Identify active account by checking OpenCode auth
	activeEmail := ""
	if auth, authErr := account.LoadOpenCodeAuth(); authErr == nil && auth.Anthropic.Access != "" {
		// Match refresh token to saved account
		cfg, cfgErr := account.LoadConfig()
		if cfgErr == nil {
			for _, acc := range cfg.Accounts {
				if acc.RefreshToken == auth.Anthropic.Refresh {
					activeEmail = acc.Email
					break
				}
			}
		}
	}

	fmt.Println("Claude Max Usage")
	fmt.Println(strings.Repeat("-", 60))

	var bestSwitch string
	var bestHeadroom float64

	for _, awc := range accounts {
		isActive := awc.Email != "" && awc.Email == activeEmail
		marker := "  "
		if isActive {
			marker = "> "
		}

		label := awc.Name
		if awc.Email != "" {
			label = fmt.Sprintf("%s (%s)", awc.Name, awc.Email)
		}

		fmt.Printf("\n%s%s", marker, label)
		if isActive {
			fmt.Print("  [active]")
		}
		if awc.IsDefault {
			fmt.Print("  [default]")
		}
		fmt.Println()

		if awc.Capacity == nil || awc.Capacity.Error != "" {
			errMsg := "unknown"
			if awc.Capacity != nil {
				errMsg = awc.Capacity.Error
			}
			fmt.Printf("    Error: %s\n", errMsg)
			continue
		}

		c := awc.Capacity

		// 5-hour bar
		fmt.Printf("    5-hour:  %s %.0f%% used", usageBar(c.FiveHourUsed), c.FiveHourUsed)
		if c.FiveHourResets != nil {
			fmt.Printf("  (resets %s)", timeUntilReset(c.FiveHourResets))
		}
		fmt.Println()

		// Weekly bar
		fmt.Printf("    Weekly:  %s %.0f%% used", usageBar(c.SevenDayUsed), c.SevenDayUsed)
		if c.SevenDayResets != nil {
			fmt.Printf("  (resets %s)", timeUntilReset(c.SevenDayResets))
		}
		fmt.Println()

		// Track best non-active account for switch recommendation
		if !isActive {
			headroom := minFloat(c.FiveHourRemaining, c.SevenDayRemaining)
			if headroom > bestHeadroom {
				bestHeadroom = headroom
				bestSwitch = awc.Name
			}
		}
	}

	// Switch recommendation
	// Find active account's headroom
	var activeHeadroom float64
	for _, awc := range accounts {
		isActive := awc.Email != "" && awc.Email == activeEmail
		if isActive && awc.Capacity != nil && awc.Capacity.Error == "" {
			activeHeadroom = minFloat(awc.Capacity.FiveHourRemaining, awc.Capacity.SevenDayRemaining)
			break
		}
	}

	if bestSwitch != "" && activeHeadroom < 20 && bestHeadroom > activeHeadroom+10 {
		fmt.Printf("\nRecommendation: switch to '%s' (%.0f%% headroom vs %.0f%%)\n", bestSwitch, bestHeadroom, activeHeadroom)
		fmt.Printf("  Run: orch account switch %s\n", bestSwitch)
	}

	fmt.Println()
	return nil
}

// usageBar returns a 20-char visual bar representing usage percentage.
func usageBar(pct float64) string {
	const barLen = 20
	filled := int(pct / 100.0 * barLen)
	if filled > barLen {
		filled = barLen
	}
	if filled < 0 {
		filled = 0
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat(".", barLen-filled) + "]"
}

// minFloat returns the smaller of two float64 values.
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// timeUntilReset returns a human-readable duration until a reset time.
func timeUntilReset(t *time.Time) string {
	if t == nil {
		return ""
	}
	d := time.Until(*t)
	if d <= 0 {
		return "now"
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if hours >= 24 {
		return fmt.Sprintf("%dd %dh", hours/24, hours%24)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
// test
