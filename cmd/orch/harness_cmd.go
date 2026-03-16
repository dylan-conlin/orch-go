package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/dylan-conlin/orch-go/pkg/events"
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
var harnessSnapshotJSON bool
var harnessSnapshotType string

var harnessSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Capture directory-level line count snapshot for accretion velocity tracking",
	Long: `Capture a codebase snapshot and emit an accretion.snapshot event to events.jsonl.

This is the native snapshot command — it both displays the snapshot AND records it
for harness report velocity calculations. The harness report requires 2+ snapshots
to compute velocity trends.

Use --type to label the snapshot: "baseline" (first capture), "weekly" (periodic),
or "manual" (ad-hoc).

Examples:
  orch harness snapshot                  # Emit snapshot (type: manual)
  orch harness snapshot --type baseline  # Emit baseline snapshot
  orch harness snapshot --json           # Machine-readable output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHarnessSnapshot()
	},
}

func init() {
	harnessSnapshotCmd.Flags().BoolVar(&harnessSnapshotJSON, "json", false, "Machine-readable JSON output")
	harnessSnapshotCmd.Flags().StringVar(&harnessSnapshotType, "type", "manual", "Snapshot type: baseline, weekly, manual")

	harnessCmd.AddCommand(harnessInitCmd)
	harnessCmd.AddCommand(harnessCheckCmd)
	harnessCmd.AddCommand(harnessLockCmd)
	harnessCmd.AddCommand(harnessUnlockCmd)
	harnessCmd.AddCommand(harnessStatusCmd)
	harnessCmd.AddCommand(harnessVerifyCmd)
	harnessCmd.AddCommand(harnessSnapshotCmd)
}

func runHarnessSnapshot() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	snapshots := collectAllSnapshots(projectDir)
	if len(snapshots) == 0 {
		return fmt.Errorf("no code directories found in %s", projectDir)
	}

	// Emit event to events.jsonl
	logger := events.NewDefaultLogger()
	data := events.AccretionSnapshotData{
		Directories:  snapshots,
		SnapshotType: harnessSnapshotType,
	}
	if err := logger.LogAccretionSnapshot(data); err != nil {
		return fmt.Errorf("emitting snapshot event: %w", err)
	}

	// Display
	totalLines := 0
	totalFiles := 0
	for _, s := range snapshots {
		totalLines += s.TotalLines
		totalFiles += s.FileCount
	}

	if harnessSnapshotJSON {
		out := map[string]interface{}{
			"directories":     snapshots,
			"total_lines":     totalLines,
			"total_files":     totalFiles,
			"directory_count": len(snapshots),
			"snapshot_type":   harnessSnapshotType,
			"emitted":         true,
		}
		enc, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(enc))
		return nil
	}

	fmt.Printf("Snapshot: %d directories, %d files, %d lines (type: %s)\n",
		len(snapshots), totalFiles, totalLines, harnessSnapshotType)
	for _, s := range snapshots {
		fmt.Printf("  %-20s %6d lines, %d files", s.Directory, s.TotalLines, s.FileCount)
		if s.LargestFile != "" {
			fmt.Printf(" (largest: %s @ %d)", s.LargestFile, s.LargestLines)
		}
		fmt.Println()
	}
	fmt.Printf("\nEvent emitted to events.jsonl (type: %s)\n", harnessSnapshotType)
	return nil
}
