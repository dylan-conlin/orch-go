package main

import (
	"fmt"
	"os"
	"os/exec"

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
  init     Set up governance artifacts for a project
  check    Scan project for file accretion violations
  lock     Apply chflags uchg to all control plane files
  unlock   Remove chflags uchg to allow intentional modifications
  status   Show lock state of all control plane files
  verify   Check that all control plane files are locked
  snapshot Capture directory-level line count snapshot
  report   Show harness measurement report (orch-native)

Workflow:
  orch harness unlock    # Allow modifications
  <edit settings.json or hooks>
  orch harness lock      # Re-protect

Most subcommands delegate to the standalone 'harness' binary.
Install it with: go install github.com/dylan-conlin/harness@latest`,
}

// findHarnessBinary locates the standalone harness binary.
// Checks PATH first, then falls back to ~/bin/harness.
func findHarnessBinary() (string, error) {
	if path, err := exec.LookPath("harness"); err == nil {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("harness binary not found in PATH and cannot determine home directory: %w", err)
	}
	fallback := home + "/bin/harness"
	if _, err := os.Stat(fallback); err == nil {
		return fallback, nil
	}
	return "", fmt.Errorf("harness binary not found.\n\nInstall it with:\n  go install github.com/dylan-conlin/harness@latest\n\nOr ensure it's in your PATH or at ~/bin/harness")
}

// runHarnessBinary executes the standalone harness binary with the given arguments.
// It connects stdin/stdout/stderr directly for transparent passthrough.
func runHarnessBinary(args ...string) error {
	bin, err := findHarnessBinary()
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// harnessDelegate creates a cobra command that delegates to the standalone harness binary.
func harnessDelegate(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:                use,
		Short:              short,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHarnessBinary(append([]string{use}, args...)...)
		},
	}
}

var harnessInitCmd = harnessDelegate("init", "Initialize governance artifacts in the current project")
var harnessCheckCmd = harnessDelegate("check", "Scan project for file accretion violations")
var harnessLockCmd = harnessDelegate("lock", "Lock control plane files (chflags uchg)")
var harnessUnlockCmd = harnessDelegate("unlock", "Unlock control plane files (chflags nouchg)")
var harnessStatusCmd = harnessDelegate("status", "Show lock state of all control plane files")
var harnessVerifyCmd = harnessDelegate("verify", "Verify all control plane files are locked (for pre-commit hooks)")
var harnessSnapshotCmd = harnessDelegate("snapshot", "Capture directory-level line count snapshot for accretion velocity tracking")

func init() {
	harnessCmd.AddCommand(harnessInitCmd)
	harnessCmd.AddCommand(harnessCheckCmd)
	harnessCmd.AddCommand(harnessLockCmd)
	harnessCmd.AddCommand(harnessUnlockCmd)
	harnessCmd.AddCommand(harnessStatusCmd)
	harnessCmd.AddCommand(harnessVerifyCmd)
	harnessCmd.AddCommand(harnessSnapshotCmd)
}
