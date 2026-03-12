package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/control"
	"github.com/dylan-conlin/orch-go/pkg/harness"
	"github.com/spf13/cobra"
)

var (
	harnessInitDryRun bool
)

var harnessInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Automate Day 1 governance setup (MVH Tier 1)",
	Long: `Set up behavioral enforcement for agent-heavy projects.

Automates the Minimum Viable Harness (MVH) Tier 1 checklist:
  1. Deny rules — prevent agents from editing control plane files
  2. Hook scripts — generate gate scripts in .claude/hooks/
  3. Hook registration — register gates in settings.json
  4. Pre-commit gate — accretion warnings on file growth
  5. Control plane lock — chflags uchg on settings.json and hooks (macOS)

Works in two modes:
  - Standalone: Any project with Claude Code (no orch/beads required)
  - Full: Projects using orch orchestration (auto-detected)

Standalone mode is auto-detected when ~/.orch/hooks/ or .beads/ are absent.
In standalone mode, hook scripts are generated inline in .claude/hooks/ and
the pre-commit gate uses a self-contained accretion check script.

This command is idempotent — safe to run multiple times.
Use --dry-run to preview changes without applying them.`,
	RunE: runHarnessInit,
}

func init() {
	harnessInitCmd.Flags().BoolVar(&harnessInitDryRun, "dry-run", false, "Preview changes without applying")
	harnessCmd.AddCommand(harnessInitCmd)
}

// StepResult captures the result of a single harness init step.
// Kept as alias for backward compatibility with tests.
type StepResult = harness.StepResult

// detectStandaloneMode checks whether the project should use standalone mode.
// Returns (standalone bool, hasBeads bool).
func detectStandaloneMode(projectDir string) (standalone bool, hasBeads bool) {
	mode, beads := harness.DetectMode(projectDir)
	return mode == harness.ModeStandalone, beads
}

func runHarnessInit(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	mode, hasBeads := harness.DetectMode(projectDir)
	standalone := mode == harness.ModeStandalone
	orchHooksDir := filepath.Join(home, ".orch", "hooks")
	sp := harness.SettingsPath(mode, projectDir)

	// Create project settings file if standalone and doesn't exist
	if standalone {
		if _, err := os.Stat(sp); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(sp), 0755); err != nil {
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

	if standalone {
		fmt.Fprintf(os.Stderr, "Mode: standalone (no ~/.orch/hooks/ detected)\n\n")
	} else {
		fmt.Fprintf(os.Stderr, "Mode: full (orch infrastructure detected)\n\n")
	}

	if harnessInitDryRun {
		fmt.Fprintf(os.Stderr, "DRY RUN — no changes will be made\n\n")
	}

	// Step 0: Unlock control plane if locked (needed to modify settings.json)
	if !harnessInitDryRun && !standalone {
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
	if standalone {
		denyRules = harness.StandaloneDenyRules()
	}
	if harnessInitDryRun {
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

	// Step 2: Hook scripts (standalone generates them; full mode expects them pre-installed)
	if standalone {
		stepNum++
		projectHooksDir := filepath.Join(projectDir, ".claude", "hooks")
		fmt.Fprintf(os.Stderr, "%d. Hook scripts\n", stepNum)
		if harnessInitDryRun {
			if _, err := os.Stat(filepath.Join(projectHooksDir, "gate-git-add-all.py")); err == nil {
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
	if harnessInitDryRun {
		fmt.Fprintf(os.Stderr, "   WOULD REGISTER hooks in settings.json\n")
	} else {
		var hookResult *harness.StepResult
		if standalone {
			projectHooksDir := filepath.Join(projectDir, ".claude", "hooks")
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
		if harnessInitDryRun {
			hookPath := filepath.Join(projectDir, ".beads", "hooks", "on_close")
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
	if _, err := os.Stat(filepath.Join(projectDir, ".git")); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "   SKIP — not a git repository\n")
	} else if harnessInitDryRun {
		precommitPath := filepath.Join(projectDir, ".git", "hooks", "pre-commit")
		data, _ := os.ReadFile(precommitPath)
		content := string(data)
		if standalone && strings.Contains(content, "Accretion Gate") {
			fmt.Fprintf(os.Stderr, "   SKIP — already wired\n")
		} else if !standalone && strings.Contains(content, "orch precommit accretion") {
			fmt.Fprintf(os.Stderr, "   SKIP — already wired\n")
		} else {
			fmt.Fprintf(os.Stderr, "   WOULD ADD accretion gate to .git/hooks/pre-commit\n")
		}
	} else {
		var pcResult *harness.StepResult
		if standalone {
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

	// Step: Lock control plane (macOS only)
	stepNum++
	fmt.Fprintf(os.Stderr, "%d. Control plane lock\n", stepNum)
	if standalone {
		fmt.Fprintf(os.Stderr, "   SKIP — standalone mode (project files are git-tracked)\n")
	} else if runtime.GOOS != "darwin" {
		fmt.Fprintf(os.Stderr, "   SKIP — chflags uchg is macOS-only (current OS: %s)\n", runtime.GOOS)
		fmt.Fprintf(os.Stderr, "   TIP: On Linux, use 'chattr +i' manually for equivalent protection\n")
	} else if harnessInitDryRun {
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
				if err := control.RemoveUnlockMarker(); err != nil {
					fmt.Fprintf(os.Stderr, "   warning: could not remove unlock marker: %v\n", err)
				}
				fmt.Fprintf(os.Stderr, "   OK — locked %d files\n", len(files))
				steps = append(steps, fmt.Sprintf("Locked %d control plane files", len(files)))
			}
		}
	}

	// Summary
	fmt.Fprintf(os.Stderr, "\n")
	if harnessInitDryRun {
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
		if standalone {
			fmt.Fprintf(os.Stderr, "\nGenerated files (commit these to your repo):\n")
			fmt.Fprintf(os.Stderr, "  .claude/hooks/gate-git-add-all.py\n")
			fmt.Fprintf(os.Stderr, "\nNext steps:\n")
			fmt.Fprintf(os.Stderr, "  git add .claude/hooks/\n")
			fmt.Fprintf(os.Stderr, "  git commit -m 'chore: add harness governance gates'\n")
		} else {
			fmt.Fprintf(os.Stderr, "\nNext: Run 'orch harness verify' to confirm.\n")
		}
	}

	if hasErrors {
		return fmt.Errorf("harness init had errors")
	}
	return nil
}

// --- Standalone mode functions (delegated to pkg/harness) ---
// These are kept as thin wrappers for backward compatibility with existing tests.

func standaloneDenyRules() []string {
	return harness.StandaloneDenyRules()
}

func ensureDenyRules(settingsPath string) (*StepResult, error) {
	return harness.EnsureDenyRules(settingsPath, nil) // nil = full deny rules
}

func ensureDenyRulesWithRules(settingsPath string, rules []string) (*StepResult, error) {
	return harness.EnsureDenyRules(settingsPath, rules)
}

func checkDenyRules(settingsPath string) (*StepResult, error) {
	return harness.CheckDenyRules(settingsPath, nil)
}

func checkDenyRulesWithRules(settingsPath string, rules []string) (*StepResult, error) {
	return harness.CheckDenyRules(settingsPath, rules)
}

func ensureStandaloneHookScripts(hooksDir string) (*StepResult, error) {
	return harness.EnsureHookScripts(hooksDir)
}

func ensureStandalonePreCommitGate(projectDir string) (*StepResult, error) {
	return harness.EnsureStandalonePreCommitGate(projectDir)
}

func ensurePreCommitGate(projectDir string) (*StepResult, error) {
	return harness.EnsureOrchPreCommitGate(projectDir)
}

func ensureBeadsCloseHook(projectDir string) (*StepResult, error) {
	return harness.EnsureBeadsCloseHook(projectDir)
}

func ensureHookRegistration(settingsPath, hooksDir string) (*StepResult, error) {
	return harness.EnsureHookRegistration(settingsPath, hooksDir, harness.FullModeHookSpecs, false, "")
}

func ensureStandaloneHookRegistration(settingsPath, hooksDir, userSettingsPath string) (*StepResult, error) {
	return harness.EnsureHookRegistration(settingsPath, hooksDir, harness.StandaloneHookSpecs, true, userSettingsPath)
}

func collectRegisteredCommands(settingsPath string) map[string]bool {
	return harness.CollectRegisteredCommands(settingsPath)
}

func isEquivalentHookRegistered(registeredCommands map[string]bool, scriptName string) bool {
	return harness.IsEquivalentHookRegistered(registeredCommands, scriptName)
}

// removeTrailingExit delegates to pkg/harness.
func removeTrailingExit(content string) string {
	return harness.RemoveTrailingExit(content)
}

// ensureBashShebang delegates to pkg/harness.
func ensureBashShebang(content string) string {
	return harness.EnsureBashShebang(content)
}

// stripScriptShebang delegates to pkg/harness.
func stripScriptShebang(content string) string {
	return harness.StripScriptShebang(content)
}

// checkHookRegistration checks hook registration without modifying settings.json.
func checkHookRegistration(settingsPath, hooksDir string) (*StepResult, error) {
	result := &StepResult{}

	commands := harness.CollectRegisteredCommands(settingsPath)

	allPresent := true
	for _, spec := range harness.FullModeHookSpecs {
		scriptPath := filepath.Join(hooksDir, spec.Script)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			continue
		}
		command := fmt.Sprintf("python3 %s", scriptPath)
		if !commands[command] {
			allPresent = false
			break
		}
	}

	result.AlreadyPresent = allPresent
	return result, nil
}
