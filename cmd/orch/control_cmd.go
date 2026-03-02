package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylan-conlin/orch-go/pkg/control"
	"github.com/spf13/cobra"
)

var controlCmd = &cobra.Command{
	Use:   "control",
	Short: "Manage control plane immutability",
	Long: `Manage OS-level immutability for control plane files.

The control plane consists of settings.json and enforcement hook scripts
(PreToolUse, Stop events). These files define agent constraints and are
protected from modification using macOS chflags uchg.

Commands:
  lock     Apply chflags uchg to all control plane files
  unlock   Remove chflags uchg from all control plane files
  status   Show lock state of all control plane files
  ack      Signal human presence (touch heartbeat)
  resume   Clear circuit breaker halt and signal human presence`,
}

var controlLockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock control plane files (chflags uchg)",
	Long: `Apply chflags uchg to all control plane files, making them immutable.

Discovers enforcement hook scripts from settings.json (PreToolUse, Stop events)
and applies the user immutable flag to each file plus settings.json itself.

After locking, agents cannot modify these files via Edit, Write, rm, or any
other mechanism — the OS blocks all writes.`,
	RunE: runControlLock,
}

var controlUnlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock control plane files (chflags nouchg)",
	Long: `Remove chflags uchg from all control plane files, allowing modification.

Use this before modifying settings.json or enforcement hooks, then re-lock
with 'orch control lock' when done.`,
	RunE: runControlUnlock,
}

var controlStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show lock state and circuit breaker status",
	RunE:  runControlStatus,
}

var controlAckCmd = &cobra.Command{
	Use:   "ack",
	Short: "Signal human presence (touch heartbeat)",
	Long: `Touch the heartbeat file to signal that a human is actively monitoring.

The circuit breaker's unverified velocity check uses heartbeat staleness
to detect autonomous drift. Run this periodically to acknowledge that
you are reviewing agent output.`,
	RunE: runControlAck,
}

var controlResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Clear circuit breaker halt and signal human presence",
	Long: `Clear the halt file written by the circuit breaker and touch the
heartbeat to signal human presence. The daemon will resume spawning
on its next poll cycle.`,
	RunE: runControlResume,
}

func init() {
	controlCmd.AddCommand(controlLockCmd)
	controlCmd.AddCommand(controlUnlockCmd)
	controlCmd.AddCommand(controlStatusCmd)
	controlCmd.AddCommand(controlAckCmd)
	controlCmd.AddCommand(controlResumeCmd)
}

func settingsPath() string {
	if p := os.Getenv("ORCH_SETTINGS_PATH"); p != "" {
		return p
	}
	return control.DefaultSettingsPath()
}

func runControlLock(cmd *cobra.Command, args []string) error {
	sp := settingsPath()
	files, err := control.DiscoverControlPlaneFiles(sp)
	if err != nil {
		return fmt.Errorf("discovering control plane: %w", err)
	}

	if err := control.Lock(files); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Locked %d control plane files:\n", len(files))
	home, _ := os.UserHomeDir()
	for _, f := range files {
		fmt.Fprintf(os.Stderr, "  uchg %s\n", shortPath(f, home))
	}
	return nil
}

func runControlUnlock(cmd *cobra.Command, args []string) error {
	sp := settingsPath()
	files, err := control.DiscoverControlPlaneFiles(sp)
	if err != nil {
		return fmt.Errorf("discovering control plane: %w", err)
	}

	if err := control.Unlock(files); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Unlocked %d control plane files:\n", len(files))
	home, _ := os.UserHomeDir()
	for _, f := range files {
		fmt.Fprintf(os.Stderr, "  ---- %s\n", shortPath(f, home))
	}
	return nil
}

func runControlStatus(cmd *cobra.Command, args []string) error {
	sp := settingsPath()
	files, err := control.DiscoverControlPlaneFiles(sp)
	if err != nil {
		return fmt.Errorf("discovering control plane: %w", err)
	}

	home, _ := os.UserHomeDir()
	locked := 0
	missing := 0
	for _, f := range files {
		status, err := control.FileStatus(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERR  %s: %v\n", shortPath(f, home), err)
			continue
		}
		if !status.Exists {
			fmt.Fprintf(os.Stderr, "  MISS %s\n", shortPath(f, home))
			missing++
			continue
		}
		if status.Locked {
			fmt.Fprintf(os.Stderr, "  uchg %s\n", shortPath(f, home))
			locked++
		} else {
			fmt.Fprintf(os.Stderr, "  ---- %s\n", shortPath(f, home))
		}
	}

	total := len(files) - missing
	if locked == total && total > 0 {
		fmt.Fprintf(os.Stderr, "\nControl plane: LOCKED (%d/%d files)\n", locked, total)
	} else if locked == 0 {
		fmt.Fprintf(os.Stderr, "\nControl plane: UNLOCKED (%d files)\n", total)
	} else {
		fmt.Fprintf(os.Stderr, "\nControl plane: PARTIAL (%d/%d locked)\n", locked, total)
	}

	// Circuit breaker status
	cbStatus, err := control.CircuitBreakerStatus(control.DefaultHaltPath(), control.DefaultHeartbeatPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nCircuit breaker: ERR (%v)\n", err)
		return nil
	}

	fmt.Fprintln(os.Stderr)
	if cbStatus.Halted {
		fmt.Fprintf(os.Stderr, "Circuit breaker: HALTED\n")
		fmt.Fprintf(os.Stderr, "  Reason:  %s\n", cbStatus.HaltReason)
		fmt.Fprintf(os.Stderr, "  Trigger: %s\n", cbStatus.HaltTrigger)
		fmt.Fprintf(os.Stderr, "  Resume:  orch control resume\n")
	} else {
		fmt.Fprintf(os.Stderr, "Circuit breaker: OK\n")
	}
	fmt.Fprintf(os.Stderr, "  Heartbeat: %s ago\n", formatDuration(cbStatus.HeartbeatAge))

	return nil
}

func runControlAck(cmd *cobra.Command, args []string) error {
	if err := control.Ack(control.DefaultHeartbeatPath()); err != nil {
		return fmt.Errorf("touching heartbeat: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Heartbeat acknowledged.")

	// Show current circuit breaker status
	cbStatus, err := control.CircuitBreakerStatus(control.DefaultHaltPath(), control.DefaultHeartbeatPath())
	if err != nil {
		return nil // ack succeeded, status is bonus info
	}
	if cbStatus.Halted {
		fmt.Fprintf(os.Stderr, "Note: circuit breaker is HALTED (%s). Run 'orch control resume' to clear.\n", cbStatus.HaltReason)
	}
	return nil
}

func runControlResume(cmd *cobra.Command, args []string) error {
	// Check if actually halted first (for messaging)
	halt, _ := control.HaltStatus(control.DefaultHaltPath())

	if err := control.Resume(control.DefaultHaltPath(), control.DefaultHeartbeatPath()); err != nil {
		return fmt.Errorf("resuming: %w", err)
	}

	if halt.Halted {
		fmt.Fprintf(os.Stderr, "Circuit breaker cleared (was: %s).\n", halt.Reason)
	} else {
		fmt.Fprintln(os.Stderr, "Not halted. Heartbeat refreshed.")
	}
	return nil
}

// formatDuration is declared in wait.go — reused here for heartbeat age display.

// shortPath replaces $HOME prefix with ~ for display.
func shortPath(path, home string) string {
	if home != "" {
		rel, err := filepath.Rel(home, path)
		if err == nil && !filepath.IsAbs(rel) {
			return "~/" + rel
		}
	}
	return path
}
