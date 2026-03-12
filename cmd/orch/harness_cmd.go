package main

import (
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/control"
	"github.com/dylan-conlin/orch-go/pkg/harness"
	"github.com/spf13/cobra"
)

var harnessCmd = &cobra.Command{
	Use:   "harness",
	Short: "Structural governance for Claude Code projects",
	Long: `Structural governance for Claude Code projects.

Manages OS-level immutability, deny rules, hook scripts, and pre-commit
accretion gates. Works standalone or with full orch orchestration.

For standalone use without orch, install the 'harness' binary directly:
  go install github.com/dylan-conlin/orch-go/cmd/harness@latest

Commands:
  init     Set up governance for this project
  check    Verify governance is healthy
  lock     Apply chflags uchg to all control plane files
  unlock   Remove chflags uchg to allow intentional modifications
  status   Show lock state of all control plane files
  verify   Verify all locked (for pre-commit hooks)`,
}

var harnessLockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock control plane files (chflags uchg)",
	RunE:  runHarnessLock,
}

var harnessUnlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock control plane files (chflags nouchg)",
	RunE:  runHarnessUnlock,
}

var harnessStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show lock state of all control plane files",
	RunE:  runHarnessStatus,
}

var harnessVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify all control plane files are locked (for pre-commit hooks)",
	RunE:  runHarnessVerify,
}

var harnessCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify governance is healthy",
	RunE:  runHarnessCheck,
}

func init() {
	harnessCmd.AddCommand(harnessLockCmd)
	harnessCmd.AddCommand(harnessUnlockCmd)
	harnessCmd.AddCommand(harnessStatusCmd)
	harnessCmd.AddCommand(harnessVerifyCmd)
	harnessCmd.AddCommand(harnessCheckCmd)
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

	if err := control.RemoveUnlockMarker(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not remove unlock marker: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Locked %d control plane files:\n", len(files))
	home, _ := os.UserHomeDir()
	for _, f := range files {
		fmt.Fprintf(os.Stderr, "  uchg %s\n", harness.ShortPath(f, home))
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

	if err := control.WriteUnlockMarker(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write unlock marker: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Unlocked %d control plane files:\n", len(files))
	home, _ := os.UserHomeDir()
	for _, f := range files {
		fmt.Fprintf(os.Stderr, "  ---- %s\n", harness.ShortPath(f, home))
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
			fmt.Fprintf(os.Stderr, "  ERR  %s: %v\n", harness.ShortPath(f, home), err)
			continue
		}
		if !status.Exists {
			fmt.Fprintf(os.Stderr, "  MISS %s\n", harness.ShortPath(f, home))
			missing++
			continue
		}
		if status.Locked {
			fmt.Fprintf(os.Stderr, "  uchg %s\n", harness.ShortPath(f, home))
			locked++
		} else {
			fmt.Fprintf(os.Stderr, "  ---- %s\n", harness.ShortPath(f, home))
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
		fmt.Fprintf(os.Stderr, "  ---- %s\n", harness.ShortPath(f, home))
	}
	fmt.Fprintf(os.Stderr, "\nFix: orch harness lock\n")
	fmt.Fprintf(os.Stderr, "Or for intentional edits: orch harness unlock\n")
	return fmt.Errorf("%d control plane file(s) unlocked without marker", len(unlocked))
}

func runHarnessCheck(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	result, err := harness.Check(projectDir)
	if err != nil {
		return err
	}

	modeStr := "standalone"
	if result.Mode == harness.ModeFull {
		modeStr = "full"
	}
	fmt.Fprintf(os.Stderr, "Mode: %s\n\n", modeStr)

	check := func(ok bool, label string) {
		if ok {
			fmt.Fprintf(os.Stderr, "  OK   %s\n", label)
		} else {
			fmt.Fprintf(os.Stderr, "  FAIL %s\n", label)
		}
	}

	check(result.DenyRulesOK, "Deny rules")
	check(result.HooksOK, "Hook scripts")
	check(result.PreCommitOK, "Pre-commit gate")
	if result.Mode == harness.ModeFull {
		check(result.LockOK, "Control plane lock")
	}

	if len(result.Issues) > 0 {
		fmt.Fprintf(os.Stderr, "\nIssues:\n")
		for _, issue := range result.Issues {
			fmt.Fprintf(os.Stderr, "  - %s\n", issue)
		}
		fmt.Fprintf(os.Stderr, "\nFix: orch harness init\n")
		return fmt.Errorf("%d issue(s) found", len(result.Issues))
	}

	fmt.Fprintf(os.Stderr, "\nAll checks passed.\n")
	return nil
}
