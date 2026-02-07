// Package main provides the CLI entry point for orch-go.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/usage"
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
	rootCmd.AddCommand(claimCmd)
	rootCmd.AddCommand(questionCmd)
	rootCmd.AddCommand(abandonCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(waitCmd)
	rootCmd.AddCommand(focusCmd)
	rootCmd.AddCommand(driftCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(portCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(retriesCmd)
	rootCmd.AddCommand(frontierCmd)
	rootCmd.AddCommand(reworkCmd)
}

var (
	versionSource bool // Show source info and staleness check
	versionJSON   bool // Output as JSON
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long: `Print version information.

Use --source to see where the binary was built from and check if it's stale.
Use --json to output version information in JSON format.`,
	Run: func(cmd *cobra.Command, args []string) {
		if versionJSON {
			runVersionJSON()
			return
		}
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
	versionCmd.Flags().BoolVar(&versionJSON, "json", false, "Output as JSON")
}

// VersionInfo represents version information in JSON format.
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	SourceDir string `json:"source_dir"`
	GitHash   string `json:"git_hash"`
}

// runVersionJSON outputs version information in JSON format.
func runVersionJSON() {
	info := VersionInfo{
		Version:   version,
		BuildTime: buildTime,
		SourceDir: sourceDir,
		GitHash:   gitHash,
	}

	output, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
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
		fmt.Printf("current HEAD: %s\n", currentHash[:12])
	} else if currentHash == gitHash {
		fmt.Println("\nstatus: ✓ UP TO DATE")
	} else {
		fmt.Println("\nstatus: ⚠️  STALE")
		fmt.Printf("binary hash:  %s\n", gitHash[:12])
		fmt.Printf("current HEAD: %s\n", currentHash[:12])
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
// Usage Tracking (Placeholder - defers to Python for now)
// ============================================================================

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show Claude Max usage limits",
	Long: `Show Claude Max weekly usage limits.

Reads OAuth token from OpenCode's auth.json and fetches usage data
from the Anthropic API.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUsage()
	},
}

func init() {
	rootCmd.AddCommand(usageCmd)
}

func runUsage() error {
	info := usage.FetchUsage()
	fmt.Println(usage.FormatDisplay(info))
	return nil
}
