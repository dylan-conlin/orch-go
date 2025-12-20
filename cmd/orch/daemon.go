// Package main provides the CLI entry point for orch-go.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Autonomous overnight processing",
	Long: `Daemon commands for autonomous overnight processing.

The daemon processes beads issues from the queue, spawning agents
for each issue in priority order.

Subcommands:
  run      Process issues in a loop until queue is empty
  once     Process a single issue and exit
  preview  Show what would be processed next without processing`,
}

var daemonRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Process issues in a loop until queue is empty",
	Long: `Process all open beads issues in priority order, spawning agents for each.

Runs until the queue is empty or interrupted with Ctrl+C.

Examples:
  orch-go daemon run              # Process all issues
  orch-go daemon run --delay 30   # Wait 30 seconds between spawns
  orch-go daemon run --dry-run    # Preview what would be processed without spawning`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonLoop()
	},
}

var daemonOnceCmd = &cobra.Command{
	Use:   "once",
	Short: "Process a single issue and exit",
	Long: `Process the next issue from the queue and exit.

Useful for testing or manual step-by-step processing.

Examples:
  orch-go daemon once`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonOnce()
	},
}

var daemonPreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Show what would be processed next without processing",
	Long: `Preview the next issue that would be processed by the daemon.

Shows issue details and inferred skill without actually spawning an agent.

Examples:
  orch-go daemon preview`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonPreview()
	},
}

var (
	// Daemon flags
	daemonDelay  int  // Delay between spawns in seconds
	daemonDryRun bool // Preview mode - show what would be processed without spawning
)

func init() {
	daemonCmd.AddCommand(daemonRunCmd)
	daemonCmd.AddCommand(daemonOnceCmd)
	daemonCmd.AddCommand(daemonPreviewCmd)

	daemonRunCmd.Flags().IntVar(&daemonDelay, "delay", 5, "Delay between spawns in seconds")
	daemonRunCmd.Flags().BoolVar(&daemonDryRun, "dry-run", false, "Preview mode - show what would be processed without spawning")
}

func runDaemonLoop() error {
	// Handle dry-run mode
	if daemonDryRun {
		return runDaemonDryRun()
	}

	d := daemon.New()

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, stopping daemon...")
		cancel()
	}()

	logger := events.NewLogger(events.DefaultLogPath())
	processed := 0

	fmt.Println("Starting daemon loop...")
	fmt.Printf("Delay between spawns: %d seconds\n\n", daemonDelay)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("\nDaemon stopped. Processed %d issues.\n", processed)
			return nil
		default:
		}

		result, err := d.Once()
		if err != nil {
			return fmt.Errorf("daemon error: %w", err)
		}

		if !result.Processed {
			fmt.Printf("Queue empty. Processed %d issues total.\n", processed)
			return nil
		}

		processed++
		fmt.Printf("[%d] Spawned: %s (%s) - %s\n",
			processed,
			result.Issue.ID,
			result.Skill,
			result.Issue.Title,
		)

		// Log the spawn
		event := events.Event{
			Type:      "daemon.spawn",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"beads_id": result.Issue.ID,
				"skill":    result.Skill,
				"title":    result.Issue.Title,
				"count":    processed,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
		}

		// Delay before next spawn
		select {
		case <-ctx.Done():
			fmt.Printf("\nDaemon stopped. Processed %d issues.\n", processed)
			return nil
		case <-time.After(time.Duration(daemonDelay) * time.Second):
		}
	}
}

func runDaemonDryRun() error {
	d := daemon.New()

	result, err := d.Preview()
	if err != nil {
		return fmt.Errorf("preview error: %w", err)
	}

	fmt.Println("[DRY-RUN] Would process the following issue:")
	fmt.Println()

	if result.Issue == nil {
		fmt.Println("No spawnable issues in queue")
		return nil
	}

	// Get current directory for context
	projectDir, _ := os.Getwd()
	projectName := filepath.Base(projectDir)

	fmt.Printf("  Project:  %s\n", projectName)
	fmt.Println(daemon.FormatPreview(result.Issue))
	fmt.Printf("\nInferred skill: %s\n", result.Skill)
	fmt.Println("\nNo agents were spawned (dry-run mode).")

	return nil
}

func runDaemonOnce() error {
	d := daemon.New()

	result, err := d.Once()
	if err != nil {
		return fmt.Errorf("daemon error: %w", err)
	}

	if !result.Processed {
		fmt.Println(result.Message)
		return nil
	}

	fmt.Printf("Spawned: %s\n", result.Issue.ID)
	fmt.Printf("  Title:  %s\n", result.Issue.Title)
	fmt.Printf("  Type:   %s\n", result.Issue.IssueType)
	fmt.Printf("  Skill:  %s\n", result.Skill)

	// Log the spawn
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "daemon.once",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id": result.Issue.ID,
			"skill":    result.Skill,
			"title":    result.Issue.Title,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	return nil
}

func runDaemonPreview() error {
	d := daemon.New()

	result, err := d.Preview()
	if err != nil {
		return fmt.Errorf("preview error: %w", err)
	}

	if result.Issue == nil {
		fmt.Println(result.Message)
		return nil
	}

	// Get current directory for context
	projectDir, _ := os.Getwd()
	projectName := filepath.Base(projectDir)

	fmt.Println("Next issue to process:")
	fmt.Printf("  Project:  %s\n", projectName)
	fmt.Println(daemon.FormatPreview(result.Issue))
	fmt.Printf("\nInferred skill: %s\n", result.Skill)
	fmt.Println("\nRun 'orch-go daemon once' to process this issue.")

	return nil
}
