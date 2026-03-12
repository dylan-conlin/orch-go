package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/control"
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
type StepResult struct {
	AlreadyPresent  bool
	RulesAdded      int
	HooksRegistered int
	Created         bool
	Error           error
}

// detectStandaloneMode checks whether the project should use standalone mode.
// Returns (standalone bool, hasBeads bool).
// Standalone mode is used when ~/.orch/hooks/ doesn't exist (no orch infrastructure).
func detectStandaloneMode(projectDir string) (standalone bool, hasBeads bool) {
	home, _ := os.UserHomeDir()
	orchHooksDir := filepath.Join(home, ".orch", "hooks")
	_, orchErr := os.Stat(orchHooksDir)
	_, beadsErr := os.Stat(filepath.Join(projectDir, ".beads"))
	hasBeads = beadsErr == nil
	standalone = os.IsNotExist(orchErr)
	return
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

	standalone, hasBeads := detectStandaloneMode(projectDir)
	orchHooksDir := filepath.Join(home, ".orch", "hooks")

	// Determine settings path based on mode.
	// Standalone mode uses project-level settings (.claude/settings.json in project dir)
	// because hook scripts use relative paths that resolve from the project root.
	// Writing relative paths to user-level settings would break every other project.
	var sp string
	if standalone {
		sp = filepath.Join(projectDir, ".claude", "settings.json")
		// Create project settings file if it doesn't exist
		if _, err := os.Stat(sp); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(sp), 0755); err != nil {
				return fmt.Errorf("creating .claude directory: %w", err)
			}
			if err := os.WriteFile(sp, []byte("{}\n"), 0644); err != nil {
				return fmt.Errorf("creating project settings: %w", err)
			}
		}
	} else {
		sp = settingsPath()
		// Check minimal prerequisite: user-level settings.json must exist
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
	// Skip in standalone mode — project-level settings are git-tracked,
	// chflags uchg is only used for user-level files.
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
	if standalone {
		if harnessInitDryRun {
			denyResult, _ := checkDenyRulesWithRules(sp, standaloneDenyRules())
			if denyResult.AlreadyPresent {
				fmt.Fprintf(os.Stderr, "   SKIP — all rules present\n")
			} else {
				fmt.Fprintf(os.Stderr, "   WOULD ADD deny rules to settings.json\n")
			}
		} else {
			denyResult, err := ensureDenyRulesWithRules(sp, standaloneDenyRules())
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
	} else {
		if harnessInitDryRun {
			denyResult, _ := checkDenyRules(sp)
			if denyResult.AlreadyPresent {
				fmt.Fprintf(os.Stderr, "   SKIP — all rules present\n")
			} else {
				fmt.Fprintf(os.Stderr, "   WOULD ADD %d deny rules to settings.json\n", 6-denyResult.RulesAdded)
			}
		} else {
			denyResult, err := ensureDenyRules(sp)
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
			hookResult, err := ensureStandaloneHookScripts(projectHooksDir)
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
	if standalone {
		projectHooksDir := filepath.Join(projectDir, ".claude", "hooks")
		if harnessInitDryRun {
			fmt.Fprintf(os.Stderr, "   WOULD REGISTER hooks in settings.json\n")
		} else {
			hookResult, err := ensureStandaloneHookRegistration(sp, projectHooksDir, control.DefaultSettingsPath())
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
	} else {
		if harnessInitDryRun {
			hookResult, _ := checkHookRegistration(sp, orchHooksDir)
			if hookResult.AlreadyPresent {
				fmt.Fprintf(os.Stderr, "   SKIP — hooks already registered\n")
			} else {
				fmt.Fprintf(os.Stderr, "   WOULD REGISTER hooks in settings.json\n")
			}
		} else {
			hookResult, err := ensureHookRegistration(sp, orchHooksDir)
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
			beadsResult, err := ensureBeadsCloseHook(projectDir)
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
	} else if standalone {
		if harnessInitDryRun {
			precommitPath := filepath.Join(projectDir, ".git", "hooks", "pre-commit")
			data, _ := os.ReadFile(precommitPath)
			if strings.Contains(string(data), "Accretion Gate") {
				fmt.Fprintf(os.Stderr, "   SKIP — already wired\n")
			} else {
				fmt.Fprintf(os.Stderr, "   WOULD ADD accretion gate to .git/hooks/pre-commit\n")
			}
		} else {
			pcResult, err := ensureStandalonePreCommitGate(projectDir)
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
	} else {
		if harnessInitDryRun {
			precommitPath := filepath.Join(projectDir, ".git", "hooks", "pre-commit")
			data, _ := os.ReadFile(precommitPath)
			if strings.Contains(string(data), "orch precommit accretion") {
				fmt.Fprintf(os.Stderr, "   SKIP — already wired\n")
			} else {
				fmt.Fprintf(os.Stderr, "   WOULD ADD accretion gate to .git/hooks/pre-commit\n")
			}
		} else {
			pcResult, err := ensurePreCommitGate(projectDir)
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
	}

	// Step: Lock control plane (macOS only)
	// Skip in standalone mode — project-level settings and hooks are git-tracked,
	// chflags uchg would break git checkout/pull operations.
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

// ensureBeadsCloseHook creates .beads/hooks/on_close if it doesn't exist.
func ensureBeadsCloseHook(projectDir string) (*StepResult, error) {
	result := &StepResult{}

	hookDir := filepath.Join(projectDir, ".beads", "hooks")
	hookPath := filepath.Join(hookDir, "on_close")

	if _, err := os.Stat(hookPath); err == nil {
		result.AlreadyPresent = true
		return result, nil
	}

	if err := os.MkdirAll(hookDir, 0755); err != nil {
		return nil, fmt.Errorf("creating hooks dir: %w", err)
	}

	content := `#!/bin/bash
# .beads/hooks/on_close
# Emit agent.completed event when issues are closed via bd close
#
# This closes the tracking gap where work completes but bypasses orch complete.
# The event is logged to ~/.orch/events.jsonl for stats aggregation.
#
# Called by beads with:
#   Args: <issue_id> <event_type>

ISSUE_ID="$1"
EVENT_TYPE="$2"

# Only emit for close events (safety check)
if [ "$EVENT_TYPE" != "close" ]; then
    exit 0
fi

# Skip if no issue ID provided
if [ -z "$ISSUE_ID" ]; then
    exit 0
fi

# Remove orch:agent label so bd list -l orch:agent returns only active agents
bd label remove "$ISSUE_ID" orch:agent 2>/dev/null

# Skip event emission when called from orch complete/review done/reconcile/clean.
# These paths emit their own enriched agent.completed event with skill/outcome/duration.
if [ "$ORCH_COMPLETING" = "1" ]; then
    exit 0
fi

# Emit the agent.completed event (only for direct bd close, not via orch complete)
orch emit agent.completed --beads-id "$ISSUE_ID" --reason "Closed via bd close" 2>/dev/null

# Exit successfully even if orch emit fails (hooks should not block bd close)
exit 0
`
	if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
		return nil, fmt.Errorf("writing hook: %w", err)
	}

	result.Created = true
	return result, nil
}
