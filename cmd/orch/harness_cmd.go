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

var harnessVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify all control plane files are locked (for pre-commit hooks)",
	Long: `Check that all control plane files have chflags uchg set.

Exits 0 if all files are locked or if the unlock marker exists (intentional unlock
via 'orch harness unlock'). Exits 1 if any files are unlocked without the marker.

Designed for use in pre-commit hooks to catch accidental uchg removal.`,
	RunE: runHarnessVerify,
}

func init() {
	harnessCmd.AddCommand(harnessLockCmd)
	harnessCmd.AddCommand(harnessUnlockCmd)
	harnessCmd.AddCommand(harnessStatusCmd)
	harnessCmd.AddCommand(harnessVerifyCmd)
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

	// Remove unlock marker — control plane is locked again
	if err := control.RemoveUnlockMarker(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not remove unlock marker: %v\n", err)
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

	// Write unlock marker so pre-commit hook knows this is intentional
	if err := control.WriteUnlockMarker(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write unlock marker: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Unlocked %d control plane files:\n", len(files))
	home, _ := os.UserHomeDir()
	for _, f := range files {
		fmt.Fprintf(os.Stderr, "  ---- %s\n", shortPath(f, home))
	}
	fmt.Fprintf(os.Stderr, "\nRemember to re-lock with: orch harness lock\n")
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

func runHarnessVerify(cmd *cobra.Command, args []string) error {
	// If unlock marker exists, skip verification (intentional unlock)
	if control.IsUnlockMarkerPresent() {
		fmt.Fprintf(os.Stderr, "harness verify: SKIP (unlock marker present — intentional unlock)\n")
		return nil
	}

	unlocked, err := control.VerifyLocked()
	if err != nil {
		return fmt.Errorf("verifying control plane: %w", err)
	}

	if len(unlocked) == 0 {
		fmt.Fprintf(os.Stderr, "harness verify: OK (all control plane files locked)\n")
		return nil
	}

	home, _ := os.UserHomeDir()
	fmt.Fprintf(os.Stderr, "BLOCKED: control plane files missing uchg flag:\n")
	for _, f := range unlocked {
		fmt.Fprintf(os.Stderr, "  ---- %s\n", shortPath(f, home))
	}
	fmt.Fprintf(os.Stderr, "\nFix: orch harness lock\n")
	fmt.Fprintf(os.Stderr, "Or for intentional edits: orch harness unlock\n")
	return fmt.Errorf("%d control plane file(s) unlocked without marker", len(unlocked))
}
