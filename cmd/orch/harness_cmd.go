package main

import (
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/control"
	"github.com/spf13/cobra"
)

var harnessCmd = &cobra.Command{
	Use:   "harness",
	Short: "Manage control plane immutability (hard harness)",
	Long: `Manage OS-level immutability for control plane files.

The control plane consists of ~/.claude/settings.json and enforcement hook
scripts (PreToolUse, Stop events). These files define agent constraints and
are protected from modification using macOS chflags uchg.

This is the "hard harness" layer — it makes the wrong path structurally
impossible rather than relying on agent instructions.

Commands:
  lock     Apply chflags uchg to all control plane files
  unlock   Remove chflags uchg to allow intentional modifications
  status   Show lock state of all control plane files

Workflow:
  orch harness unlock    # Allow modifications
  <edit settings.json or hooks>
  orch harness lock      # Re-protect`,
}

var harnessLockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock control plane files (chflags uchg)",
	Long: `Apply chflags uchg to all control plane files, making them immutable.

Discovers enforcement hook scripts from settings.json (PreToolUse, Stop events)
and applies the user immutable flag to each file plus settings.json itself.

After locking, agents cannot modify these files via Edit, Write, rm, or any
other mechanism — the OS blocks all writes.`,
	RunE: runHarnessLock,
}

var harnessUnlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock control plane files (chflags nouchg)",
	Long: `Remove chflags uchg from all control plane files, allowing modification.

Use this before modifying settings.json or enforcement hooks, then re-lock
with 'orch harness lock' when done.`,
	RunE: runHarnessUnlock,
}

var harnessStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show lock state of all control plane files",
	RunE:  runHarnessStatus,
}

func init() {
	harnessCmd.AddCommand(harnessLockCmd)
	harnessCmd.AddCommand(harnessUnlockCmd)
	harnessCmd.AddCommand(harnessStatusCmd)
}

func runHarnessLock(cmd *cobra.Command, args []string) error {
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

func runHarnessUnlock(cmd *cobra.Command, args []string) error {
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

func runHarnessStatus(cmd *cobra.Command, args []string) error {
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

	return nil
}
