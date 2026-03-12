// cmd/harness is a standalone CLI for structural governance of Claude Code projects.
//
// It works without orch or beads — any project with Claude Code can use it.
// When orch infrastructure is detected, it uses the full feature set.
//
// Usage:
//
//	harness init          # Set up governance for this project
//	harness check         # Verify governance is healthy
//	harness lock          # Lock control plane files (chflags uchg)
//	harness unlock        # Unlock for intentional modifications
//	harness status        # Show lock state of control plane files
//	harness verify        # Verify all locked (for pre-commit hooks)
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/dylan-conlin/orch-go/pkg/control"
	"github.com/dylan-conlin/orch-go/pkg/harness"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "harness",
	Short: "Structural governance for Claude Code projects",
	Long: `Harness provides structural governance for AI-assisted development projects.

It manages:
  - Deny rules: prevent agents from editing control plane files
  - Hook scripts: gate dangerous operations (blanket git add)
  - Pre-commit gates: block unbounded file growth (accretion)
  - Control plane lock: OS-level immutability (macOS chflags uchg)

Works standalone in any project with Claude Code, or with full orch orchestration.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

// --- init command ---

var initDryRun bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up governance for this project",
	Long: `Automate Day 1 governance setup for Claude Code projects.

Steps:
  1. Deny rules — prevent agents from editing control plane files
  2. Hook scripts — generate gate scripts in .claude/hooks/
  3. Hook registration — register gates in settings.json
  4. Pre-commit gate — accretion warnings on file growth
  5. Control plane lock — chflags uchg on settings.json and hooks (macOS)

Modes:
  - Standalone: Any project (no orch/beads required, auto-detected)
  - Full: Projects using orch orchestration

This command is idempotent — safe to run multiple times.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	mode, hasBeads := harness.DetectMode(projectDir)
	sp := harness.SettingsPath(mode, projectDir)
	orchHooksDir := fmt.Sprintf("%s/.orch/hooks", home)

	// Create project settings file if standalone and doesn't exist
	if mode == harness.ModeStandalone {
		if _, err := os.Stat(sp); os.IsNotExist(err) {
			if err := os.MkdirAll(fmt.Sprintf("%s/.claude", projectDir), 0755); err != nil {
				return fmt.Errorf("creating .claude directory: %w", err)
			}
			if err := os.WriteFile(sp, []byte("{}\n"), 0644); err != nil {
				return fmt.Errorf("creating project settings: %w", err)
			}
		}
	} else {
		if _, err := os.Stat(sp); os.IsNotExist(err) {
			return fmt.Errorf("settings.json not found at %s\nRun Claude Code first to create it, or create it manually: echo '{}' > %s", sp, sp)
		}
	}

	if mode == harness.ModeStandalone {
		fmt.Fprintf(os.Stderr, "Mode: standalone (no ~/.orch/hooks/ detected)\n\n")
	} else {
		fmt.Fprintf(os.Stderr, "Mode: full (orch infrastructure detected)\n\n")
	}

	if initDryRun {
		fmt.Fprintf(os.Stderr, "DRY RUN — no changes will be made\n\n")
	}

	// Step 0: Unlock control plane if locked (needed to modify settings.json)
	if !initDryRun && mode == harness.ModeFull {
		files, err := control.DiscoverControlPlaneFiles(sp)
		if err == nil {
			control.Unlock(files)
		}
	}

	var steps []string
	hasErrors := false
	stepNum := 0

	// Step 1: Deny rules
	stepNum++
	fmt.Fprintf(os.Stderr, "%d. Deny rules\n", stepNum)
	var denyRules []string
	if mode == harness.ModeStandalone {
		denyRules = harness.StandaloneDenyRules()
	}
	if initDryRun {
		denyResult, _ := harness.CheckDenyRules(sp, denyRules)
		if denyResult != nil && denyResult.AlreadyPresent {
			fmt.Fprintf(os.Stderr, "   SKIP — all rules present\n")
		} else {
			fmt.Fprintf(os.Stderr, "   WOULD ADD deny rules to settings.json\n")
		}
	} else {
		denyResult, err := harness.EnsureDenyRules(sp, denyRules)
		if err != nil {
			fmt.Fprintf(os.Stderr, "   FAIL — %v\n", err)
			hasErrors = true
		} else if denyResult.AlreadyPresent {
			fmt.Fprintf(os.Stderr, "   SKIP — all rules present\n")
		} else {
			fmt.Fprintf(os.Stderr, "   OK — added %d deny rules\n", denyResult.RulesAdded)
			steps = append(steps, fmt.Sprintf("Added %d deny rules", denyResult.RulesAdded))
		}
	}

	// Step 2: Hook scripts (standalone generates them; full mode expects pre-installed)
	if mode == harness.ModeStandalone {
		stepNum++
		projectHooksDir := fmt.Sprintf("%s/.claude/hooks", projectDir)
		fmt.Fprintf(os.Stderr, "%d. Hook scripts\n", stepNum)
		if initDryRun {
			if _, err := os.Stat(fmt.Sprintf("%s/gate-git-add-all.py", projectHooksDir)); err == nil {
				fmt.Fprintf(os.Stderr, "   SKIP — already exists\n")
			} else {
				fmt.Fprintf(os.Stderr, "   WOULD CREATE .claude/hooks/gate-git-add-all.py\n")
			}
		} else {
			hookResult, err := harness.EnsureHookScripts(projectHooksDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "   FAIL — %v\n", err)
				hasErrors = true
			} else if hookResult.AlreadyPresent {
				fmt.Fprintf(os.Stderr, "   SKIP — already exists\n")
			} else {
				fmt.Fprintf(os.Stderr, "   OK — created .claude/hooks/gate-git-add-all.py\n")
				steps = append(steps, "Created hook script: .claude/hooks/gate-git-add-all.py")
			}
		}
	}

	// Step 3: Hook registration
	stepNum++
	fmt.Fprintf(os.Stderr, "%d. Hook registration\n", stepNum)
	if initDryRun {
		fmt.Fprintf(os.Stderr, "   WOULD REGISTER hooks in settings.json\n")
	} else {
		var hookResult *harness.StepResult
		if mode == harness.ModeStandalone {
			projectHooksDir := fmt.Sprintf("%s/.claude/hooks", projectDir)
			hookResult, err = harness.EnsureHookRegistration(sp, projectHooksDir, harness.StandaloneHookSpecs, true, control.DefaultSettingsPath())
		} else {
			hookResult, err = harness.EnsureHookRegistration(sp, orchHooksDir, harness.FullModeHookSpecs, false, "")
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "   FAIL — %v\n", err)
			hasErrors = true
		} else if hookResult.AlreadyPresent {
			fmt.Fprintf(os.Stderr, "   SKIP — hooks already registered\n")
		} else {
			fmt.Fprintf(os.Stderr, "   OK — registered %d hooks\n", hookResult.HooksRegistered)
			steps = append(steps, fmt.Sprintf("Registered %d hooks", hookResult.HooksRegistered))
		}
	}

	// Step: Beads close hook (only if .beads/ exists)
	if hasBeads {
		stepNum++
		fmt.Fprintf(os.Stderr, "%d. Beads close hook\n", stepNum)
		if initDryRun {
			hookPath := fmt.Sprintf("%s/.beads/hooks/on_close", projectDir)
			if _, err := os.Stat(hookPath); err == nil {
				fmt.Fprintf(os.Stderr, "   SKIP — already exists\n")
			} else {
				fmt.Fprintf(os.Stderr, "   WOULD CREATE .beads/hooks/on_close\n")
			}
		} else {
			beadsResult, err := harness.EnsureBeadsCloseHook(projectDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "   FAIL — %v\n", err)
				hasErrors = true
			} else if beadsResult.AlreadyPresent {
				fmt.Fprintf(os.Stderr, "   SKIP — already exists\n")
			} else {
				fmt.Fprintf(os.Stderr, "   OK — created .beads/hooks/on_close\n")
				steps = append(steps, "Created beads close hook")
			}
		}
	}

	// Step: Pre-commit accretion gate
	stepNum++
	fmt.Fprintf(os.Stderr, "%d. Pre-commit accretion gate\n", stepNum)
	if _, err := os.Stat(fmt.Sprintf("%s/.git", projectDir)); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "   SKIP — not a git repository\n")
	} else if initDryRun {
		fmt.Fprintf(os.Stderr, "   WOULD ADD accretion gate to .git/hooks/pre-commit\n")
	} else {
		var pcResult *harness.StepResult
		if mode == harness.ModeStandalone {
			pcResult, err = harness.EnsureStandalonePreCommitGate(projectDir)
		} else {
			pcResult, err = harness.EnsureOrchPreCommitGate(projectDir)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "   FAIL — %v\n", err)
			hasErrors = true
		} else if pcResult.AlreadyPresent {
			fmt.Fprintf(os.Stderr, "   SKIP — already wired\n")
		} else {
			fmt.Fprintf(os.Stderr, "   OK — added accretion gate to pre-commit\n")
			steps = append(steps, "Added pre-commit accretion gate")
		}
	}

	// Step: Lock control plane (macOS only, full mode only)
	stepNum++
	fmt.Fprintf(os.Stderr, "%d. Control plane lock\n", stepNum)
	if mode == harness.ModeStandalone {
		fmt.Fprintf(os.Stderr, "   SKIP — standalone mode (project files are git-tracked)\n")
	} else if runtime.GOOS != "darwin" {
		fmt.Fprintf(os.Stderr, "   SKIP — chflags uchg is macOS-only (current OS: %s)\n", runtime.GOOS)
	} else if initDryRun {
		fmt.Fprintf(os.Stderr, "   WOULD LOCK control plane files\n")
	} else {
		files, err := control.DiscoverControlPlaneFiles(sp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "   FAIL — %v\n", err)
			hasErrors = true
		} else {
			if err := control.Lock(files); err != nil {
				fmt.Fprintf(os.Stderr, "   FAIL — %v\n", err)
				hasErrors = true
			} else {
				control.RemoveUnlockMarker()
				fmt.Fprintf(os.Stderr, "   OK — locked %d files\n", len(files))
				steps = append(steps, fmt.Sprintf("Locked %d control plane files", len(files)))
			}
		}
	}

	// Summary
	fmt.Fprintf(os.Stderr, "\n")
	if initDryRun {
		fmt.Fprintf(os.Stderr, "Dry run complete. Run without --dry-run to apply.\n")
	} else if hasErrors {
		fmt.Fprintf(os.Stderr, "Harness init completed with errors. Fix issues above and re-run.\n")
	} else if len(steps) == 0 {
		fmt.Fprintf(os.Stderr, "Harness already fully configured — nothing to do.\n")
	} else {
		fmt.Fprintf(os.Stderr, "Harness init complete:\n")
		for _, step := range steps {
			fmt.Fprintf(os.Stderr, "  - %s\n", step)
		}
		if mode == harness.ModeStandalone {
			fmt.Fprintf(os.Stderr, "\nGenerated files (commit these to your repo):\n")
			fmt.Fprintf(os.Stderr, "  .claude/hooks/gate-git-add-all.py\n")
			fmt.Fprintf(os.Stderr, "\nNext steps:\n")
			fmt.Fprintf(os.Stderr, "  git add .claude/hooks/\n")
			fmt.Fprintf(os.Stderr, "  git commit -m 'chore: add harness governance gates'\n")
		} else {
			fmt.Fprintf(os.Stderr, "\nNext: Run 'harness verify' to confirm.\n")
		}
	}

	if hasErrors {
		return fmt.Errorf("harness init had errors")
	}
	return nil
}

// --- check command ---

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify governance is healthy",
	Long: `Check that all harness governance components are properly configured.

Verifies:
  - Deny rules present in settings.json
  - Hook scripts exist and are registered
  - Pre-commit accretion gate is wired
  - Control plane files are locked (full mode, macOS)`,
	RunE: runCheck,
}

func runCheck(cmd *cobra.Command, args []string) error {
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
		fmt.Fprintf(os.Stderr, "\nFix: harness init\n")
		return fmt.Errorf("%d issue(s) found", len(result.Issues))
	}

	fmt.Fprintf(os.Stderr, "\nAll checks passed.\n")
	return nil
}

// --- lock/unlock/status/verify commands ---

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock control plane files (chflags uchg)",
	RunE: func(cmd *cobra.Command, args []string) error {
		sp := resolveSettingsPath()
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

		home, _ := os.UserHomeDir()
		fmt.Fprintf(os.Stderr, "Locked %d control plane files:\n", len(files))
		for _, f := range files {
			fmt.Fprintf(os.Stderr, "  uchg %s\n", harness.ShortPath(f, home))
		}
		return nil
	},
}

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock control plane files (chflags nouchg)",
	RunE: func(cmd *cobra.Command, args []string) error {
		sp := resolveSettingsPath()
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

		home, _ := os.UserHomeDir()
		fmt.Fprintf(os.Stderr, "Unlocked %d control plane files:\n", len(files))
		for _, f := range files {
			fmt.Fprintf(os.Stderr, "  ---- %s\n", harness.ShortPath(f, home))
		}
		fmt.Fprintf(os.Stderr, "\nRemember to re-lock with: harness lock\n")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show lock state of all control plane files",
	RunE: func(cmd *cobra.Command, args []string) error {
		sp := resolveSettingsPath()
		files, err := control.DiscoverControlPlaneFiles(sp)
		if err != nil {
			return fmt.Errorf("discovering control plane: %w", err)
		}

		home, _ := os.UserHomeDir()
		locked := 0
		missing := 0
		for _, f := range files {
			st, err := control.FileStatus(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  ERR  %s: %v\n", harness.ShortPath(f, home), err)
				continue
			}
			if !st.Exists {
				fmt.Fprintf(os.Stderr, "  MISS %s\n", harness.ShortPath(f, home))
				missing++
				continue
			}
			if st.Locked {
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
	},
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify all control plane files are locked (for pre-commit hooks)",
	RunE: func(cmd *cobra.Command, args []string) error {
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
		fmt.Fprintf(os.Stderr, "\nFix: harness lock\n")
		fmt.Fprintf(os.Stderr, "Or for intentional edits: harness unlock\n")
		return fmt.Errorf("%d control plane file(s) unlocked without marker", len(unlocked))
	},
}

func resolveSettingsPath() string {
	if p := os.Getenv("ORCH_SETTINGS_PATH"); p != "" {
		return p
	}
	return control.DefaultSettingsPath()
}

func init() {
	initCmd.Flags().BoolVar(&initDryRun, "dry-run", false, "Preview changes without applying")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(lockCmd)
	rootCmd.AddCommand(unlockCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(verifyCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
