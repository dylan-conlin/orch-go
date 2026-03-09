package main

import (
	"encoding/json"
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
  status   Show lock state of all control plane files`,
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
	Short: "Show lock state of all control plane files",
	RunE:  runControlStatus,
}

var controlDenyCmd = &cobra.Command{
	Use:   "deny",
	Short: "Check deny rules for control plane files in settings.json",
	Long: `Verify that settings.json contains deny rules preventing agent edits
of control plane files. Shows which rules are present and which are missing.

These deny rules provide defense-in-depth on top of chflags uchg.`,
	RunE: runControlDeny,
}

func init() {
	controlCmd.AddCommand(controlLockCmd)
	controlCmd.AddCommand(controlUnlockCmd)
	controlCmd.AddCommand(controlStatusCmd)
	controlCmd.AddCommand(controlDenyCmd)
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

	return nil
}

func runControlDeny(cmd *cobra.Command, args []string) error {
	sp := settingsPath()
	data, err := os.ReadFile(sp)
	if err != nil {
		return fmt.Errorf("reading settings: %w", err)
	}

	var settings struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parsing settings: %w", err)
	}

	existing := make(map[string]bool)
	for _, rule := range settings.Permissions.Deny {
		existing[rule] = true
	}

	required := control.DenyRules()
	allPresent := true
	for _, rule := range required {
		if existing[rule] {
			fmt.Fprintf(os.Stderr, "  OK   %s\n", rule)
		} else {
			fmt.Fprintf(os.Stderr, "  MISS %s\n", rule)
			allPresent = false
		}
	}

	if allPresent {
		fmt.Fprintf(os.Stderr, "\nDeny rules: ALL PRESENT (%d rules)\n", len(required))
	} else {
		fmt.Fprintf(os.Stderr, "\nDeny rules: INCOMPLETE — add missing rules to settings.json permissions.deny\n")
		return fmt.Errorf("missing deny rules")
	}
	return nil
}

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
