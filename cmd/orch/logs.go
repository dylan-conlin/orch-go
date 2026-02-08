package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	// Logs command flags
	logsLines  int
	logsFollow bool
)

var logsCmd = &cobra.Command{
	Use:    "logs",
	Short:  "Access server and system logs",
	Hidden: true,
	Long: `Access logs from various orch-go services and systems.

Commands:
  server   Show overmind server logs (api, web, opencode)
  daemon   Show daemon logs

Examples:
  orch logs server              # Show last 50 lines of server logs
  orch logs server --lines 100  # Show last 100 lines
  orch logs server --follow     # Follow logs in real-time
  orch logs daemon              # Show daemon logs`,
}

var logsServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Show overmind server logs",
	Long: `Show logs from the overmind server which manages all dashboard services.

This includes output from:
- api: orch serve (API + static dashboard UI on port 3348)
- daemon: orch daemon run
- doctor: orch doctor --daemon
- opencode: OpenCode server (port 4096)

Examples:
  orch logs server              # Show last 50 lines
  orch logs server --lines 100  # Show last 100 lines
  orch logs server --follow     # Follow logs in real-time`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logPath := filepath.Join(os.Getenv("HOME"), ".orch", "overmind-stdout.log")
		return showLogs("server", logPath, logsLines, logsFollow)
	},
}

var logsDaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Show daemon logs",
	Long: `Show logs from the orch daemon which manages autonomous spawning.

Examples:
  orch logs daemon              # Show last 50 lines
  orch logs daemon --lines 100  # Show last 100 lines
  orch logs daemon --follow     # Follow logs in real-time`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logPath := filepath.Join(os.Getenv("HOME"), ".orch", "daemon.log")
		return showLogs("daemon", logPath, logsLines, logsFollow)
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.AddCommand(logsServerCmd)
	logsCmd.AddCommand(logsDaemonCmd)

	// Add flags to all subcommands
	logsCmd.PersistentFlags().IntVarP(&logsLines, "lines", "n", 50, "Number of lines to show")
	logsCmd.PersistentFlags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output in real-time")
}

// showLogs displays log content from the specified file.
func showLogs(source string, logPath string, lines int, follow bool) error {
	// Check if log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return fmt.Errorf("%s logs not found at %s\nIs overmind/daemon running?", source, logPath)
	}

	if follow {
		return followLogs(source, logPath, lines)
	}
	return tailLogs(source, logPath, lines)
}

// tailLogs shows the last N lines of a log file.
func tailLogs(source string, logPath string, lines int) error {
	cmd := exec.Command("tail", fmt.Sprintf("-n%d", lines), logPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("=== %s logs (last %d lines) ===\n", source, lines)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tail %s logs: %w", source, err)
	}
	fmt.Printf("=== End of %s logs ===\n", source)
	return nil
}

// followLogs follows a log file in real-time (like tail -f).
func followLogs(source string, logPath string, lines int) error {
	// First show last N lines
	cmd := exec.Command("tail", fmt.Sprintf("-n%d", lines), logPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("=== Following %s logs (Ctrl+C to exit) ===\n", source)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to show initial %s logs: %w", source, err)
	}

	// Now follow the log file
	cmd = exec.Command("tail", "-f", logPath)

	// Set up signal handling for clean exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start the tail process
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start tail: %w", err)
	}

	// Stream output in a goroutine
	done := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			done <- err
			return
		}
		done <- cmd.Wait()
	}()

	// Wait for either completion or interrupt signal
	select {
	case <-sigChan:
		fmt.Printf("\n=== Stopped following %s logs ===\n", source)
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill tail process: %w", err)
		}
		return nil
	case err := <-done:
		if err != nil {
			return fmt.Errorf("tail process failed: %w", err)
		}
		return nil
	}
}
