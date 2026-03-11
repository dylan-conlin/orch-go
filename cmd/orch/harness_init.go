package main

import (
	"encoding/json"
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

	sp := settingsPath()

	// Check minimal prerequisite: settings.json must exist
	if _, err := os.Stat(sp); os.IsNotExist(err) {
		return fmt.Errorf("settings.json not found at %s\nRun Claude Code first to create it, or create it manually: echo '{}' > %s", sp, sp)
	}

	standalone, hasBeads := detectStandaloneMode(projectDir)
	orchHooksDir := filepath.Join(home, ".orch", "hooks")

	if standalone {
		fmt.Fprintf(os.Stderr, "Mode: standalone (no ~/.orch/hooks/ detected)\n\n")
	} else {
		fmt.Fprintf(os.Stderr, "Mode: full (orch infrastructure detected)\n\n")
	}

	if harnessInitDryRun {
		fmt.Fprintf(os.Stderr, "DRY RUN — no changes will be made\n\n")
	}

	// Step 0: Unlock control plane if locked (needed to modify settings.json)
	if !harnessInitDryRun {
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
			hookResult, err := ensureStandaloneHookRegistration(sp, projectHooksDir)
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
	stepNum++
	fmt.Fprintf(os.Stderr, "%d. Control plane lock\n", stepNum)
	if runtime.GOOS != "darwin" {
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

// ensureDenyRules adds missing deny rules to settings.json.
// Preserves existing content. Returns result with count of rules added.
func ensureDenyRules(settingsPath string) (*StepResult, error) {
	result := &StepResult{}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("reading settings: %w", err)
	}

	// Parse as generic JSON to preserve structure
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}

	// Get or create permissions
	perms, ok := settings["permissions"].(map[string]any)
	if !ok {
		perms = make(map[string]any)
		settings["permissions"] = perms
	}

	// Get existing deny rules
	var existing []string
	if denyRaw, ok := perms["deny"].([]any); ok {
		for _, r := range denyRaw {
			if s, ok := r.(string); ok {
				existing = append(existing, s)
			}
		}
	}

	existingSet := make(map[string]bool)
	for _, r := range existing {
		existingSet[r] = true
	}

	// Add missing rules
	required := control.DenyRules()
	var added []string
	for _, rule := range required {
		if !existingSet[rule] {
			existing = append(existing, rule)
			added = append(added, rule)
		}
	}

	if len(added) == 0 {
		result.AlreadyPresent = true
		return result, nil
	}

	// Convert back to []any for JSON
	denyAny := make([]any, len(existing))
	for i, s := range existing {
		denyAny[i] = s
	}
	perms["deny"] = denyAny

	// Write back
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling settings: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(settingsPath, out, 0644); err != nil {
		return nil, fmt.Errorf("writing settings: %w", err)
	}

	result.RulesAdded = len(added)
	return result, nil
}

// checkDenyRules checks deny rules without modifying settings.json.
func checkDenyRules(settingsPath string) (*StepResult, error) {
	result := &StepResult{}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	perms, _ := settings["permissions"].(map[string]any)
	existingSet := make(map[string]bool)
	if perms != nil {
		if denyRaw, ok := perms["deny"].([]any); ok {
			for _, r := range denyRaw {
				if s, ok := r.(string); ok {
					existingSet[s] = true
				}
			}
		}
	}

	missing := 0
	for _, rule := range control.DenyRules() {
		if !existingSet[rule] {
			missing++
		}
	}

	result.AlreadyPresent = missing == 0
	result.RulesAdded = len(control.DenyRules()) - missing // count of present rules (reused field)
	return result, nil
}

// requiredHooks defines the hooks that harness init registers.
// Each entry: matcher -> command template (with %s for hooks dir).
type hookSpec struct {
	Matcher string
	Script  string // filename in hooksDir
}

var requiredHooks = []hookSpec{
	{Matcher: "Bash", Script: "gate-bd-close.py"},
	{Matcher: "Bash", Script: "gate-worker-git-add-all.py"},
}

// ensureHookRegistration registers required hooks in settings.json PreToolUse.
func ensureHookRegistration(settingsPath, hooksDir string) (*StepResult, error) {
	result := &StepResult{}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("reading settings: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}

	// Get or create hooks section
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		hooks = make(map[string]any)
		settings["hooks"] = hooks
	}

	// Get existing PreToolUse entries
	var ptu []any
	if existing, ok := hooks["PreToolUse"].([]any); ok {
		ptu = existing
	}

	// Check which hooks are already registered by scanning command strings
	registeredCommands := make(map[string]bool)
	for _, entry := range ptu {
		if group, ok := entry.(map[string]any); ok {
			if hookList, ok := group["hooks"].([]any); ok {
				for _, h := range hookList {
					if hookMap, ok := h.(map[string]any); ok {
						if cmd, ok := hookMap["command"].(string); ok {
							registeredCommands[cmd] = true
						}
					}
				}
			}
		}
	}

	// Register missing hooks
	registered := 0
	for _, spec := range requiredHooks {
		scriptPath := filepath.Join(hooksDir, spec.Script)

		// Check if this script exists
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			continue // Skip hooks that aren't installed
		}

		command := fmt.Sprintf("python3 %s", scriptPath)
		if registeredCommands[command] {
			continue // Already registered
		}

		// Add hook registration
		entry := map[string]any{
			"matcher": spec.Matcher,
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": command,
				},
			},
		}
		ptu = append(ptu, entry)
		registered++
	}

	if registered == 0 {
		result.AlreadyPresent = true
		return result, nil
	}

	hooks["PreToolUse"] = ptu

	// Write back
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling settings: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(settingsPath, out, 0644); err != nil {
		return nil, fmt.Errorf("writing settings: %w", err)
	}

	result.HooksRegistered = registered
	return result, nil
}

// checkHookRegistration checks hook registration without modifying settings.json.
func checkHookRegistration(settingsPath, hooksDir string) (*StepResult, error) {
	result := &StepResult{}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	registeredCommands := make(map[string]bool)
	if hooks, ok := settings["hooks"].(map[string]any); ok {
		if ptu, ok := hooks["PreToolUse"].([]any); ok {
			for _, entry := range ptu {
				if group, ok := entry.(map[string]any); ok {
					if hookList, ok := group["hooks"].([]any); ok {
						for _, h := range hookList {
							if hookMap, ok := h.(map[string]any); ok {
								if cmd, ok := hookMap["command"].(string); ok {
									registeredCommands[cmd] = true
								}
							}
						}
					}
				}
			}
		}
	}

	allPresent := true
	for _, spec := range requiredHooks {
		scriptPath := filepath.Join(hooksDir, spec.Script)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			continue
		}
		command := fmt.Sprintf("python3 %s", scriptPath)
		if !registeredCommands[command] {
			allPresent = false
			break
		}
	}

	result.AlreadyPresent = allPresent
	return result, nil
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

// --- Standalone mode functions ---
// These generate self-contained artifacts that work without orch/beads infrastructure.

// standaloneDenyRules returns deny rules for standalone mode.
// These protect Claude Code's settings files from agent modification.
// Unlike the full mode, these don't include ~/.orch/hooks/** paths.
func standaloneDenyRules() []string {
	return []string{
		"Edit(~/.claude/settings.json)",
		"Edit(~/.claude/settings.local.json)",
		"Write(~/.claude/settings.json)",
		"Write(~/.claude/settings.local.json)",
	}
}

// ensureDenyRulesWithRules adds the given deny rules to settings.json.
func ensureDenyRulesWithRules(settingsPath string, rules []string) (*StepResult, error) {
	result := &StepResult{}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("reading settings: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}

	perms, ok := settings["permissions"].(map[string]any)
	if !ok {
		perms = make(map[string]any)
		settings["permissions"] = perms
	}

	var existing []string
	if denyRaw, ok := perms["deny"].([]any); ok {
		for _, r := range denyRaw {
			if s, ok := r.(string); ok {
				existing = append(existing, s)
			}
		}
	}

	existingSet := make(map[string]bool)
	for _, r := range existing {
		existingSet[r] = true
	}

	var added []string
	for _, rule := range rules {
		if !existingSet[rule] {
			existing = append(existing, rule)
			added = append(added, rule)
		}
	}

	if len(added) == 0 {
		result.AlreadyPresent = true
		return result, nil
	}

	denyAny := make([]any, len(existing))
	for i, s := range existing {
		denyAny[i] = s
	}
	perms["deny"] = denyAny

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling settings: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(settingsPath, out, 0644); err != nil {
		return nil, fmt.Errorf("writing settings: %w", err)
	}

	result.RulesAdded = len(added)
	return result, nil
}

// checkDenyRulesWithRules checks deny rules without modifying settings.json.
func checkDenyRulesWithRules(settingsPath string, rules []string) (*StepResult, error) {
	result := &StepResult{}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	existingSet := make(map[string]bool)
	if perms, ok := settings["permissions"].(map[string]any); ok {
		if denyRaw, ok := perms["deny"].([]any); ok {
			for _, r := range denyRaw {
				if s, ok := r.(string); ok {
					existingSet[s] = true
				}
			}
		}
	}

	missing := 0
	for _, rule := range rules {
		if !existingSet[rule] {
			missing++
		}
	}

	result.AlreadyPresent = missing == 0
	return result, nil
}

// standaloneGitAddAllHook is the content of the standalone gate-git-add-all.py hook.
// It blocks `git add -A`, `git add .`, and `git add --all` to prevent agents from
// accidentally staging unrelated files in multi-agent projects.
const standaloneGitAddAllHook = `#!/usr/bin/env python3
"""
Gate: Block 'git add -A' and 'git add .' in Claude Code sessions.

Generated by: orch harness init

== WHY THIS GATE EXISTS ==

In multi-agent projects, the working directory often contains changes from
multiple agents, build artifacts, lock files, and other unrelated modifications.
When an agent runs 'git add -A' or 'git add .', it stages EVERYTHING — including
other agents' in-progress work, .env files, and build output.

This gate forces agents to stage files explicitly by name:
  git add src/feature.go src/feature_test.go    # Good — explicit
  git add -A                                     # Blocked
  git add .                                      # Blocked

== HOW IT WORKS ==

This is a Claude Code PreToolUse hook. When an agent tries to run a Bash command,
Claude Code sends the command to this script via stdin as JSON. If the command
matches a blanket git-add pattern, the script returns a deny decision that
prevents the command from executing and shows the agent an error message.

== CONFIGURATION ==

- To disable temporarily: set SKIP_GIT_ADD_ALL_GATE=1 in your environment
- To remove permanently: delete this file and remove its entry from settings.json

== HOOK PROTOCOL ==

Input (stdin): JSON with tool_name and tool_input.command
Output (stdout): JSON with hookSpecificOutput.permissionDecision = "deny" to block
Exit 0 always (hooks should not crash Claude Code)
"""
import json
import os
import re
import sys

# Patterns that match blanket git add commands
BLANKET_GIT_ADD_PATTERNS = [
    r'\bgit\s+add\s+(-A|--all)\b',
    r'\bgit\s+add\s+\.(?:\s|$|&&|\||;)',
]


def is_blanket_git_add(command: str) -> bool:
    """Detect if a Bash command uses blanket git add."""
    for pattern in BLANKET_GIT_ADD_PATTERNS:
        if re.search(pattern, command):
            return True
    return False


def main():
    # Escape hatch: set this env var to bypass the gate
    if os.environ.get("SKIP_GIT_ADD_ALL_GATE", "") == "1":
        sys.exit(0)

    try:
        input_data = json.load(sys.stdin)
    except json.JSONDecodeError:
        sys.exit(0)

    if input_data.get("tool_name") != "Bash":
        sys.exit(0)

    command = input_data.get("tool_input", {}).get("command", "")
    if not is_blanket_git_add(command):
        sys.exit(0)

    output = {
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": (
                "BLOCKED: Do not use 'git add -A' or 'git add .'\n\n"
                "In multi-agent projects, the working directory often has unrelated\n"
                "changes from other agents, build artifacts, or sensitive files.\n\n"
                "Stage ONLY the specific files you created or modified:\n"
                "  git add path/to/file1 path/to/file2\n\n"
                "To bypass: set SKIP_GIT_ADD_ALL_GATE=1"
            ),
        }
    }
    print(json.dumps(output))
    sys.exit(0)


if __name__ == "__main__":
    main()
`

// ensureStandaloneHookScripts generates hook scripts in the project's .claude/hooks/ directory.
func ensureStandaloneHookScripts(hooksDir string) (*StepResult, error) {
	result := &StepResult{}

	gatePath := filepath.Join(hooksDir, "gate-git-add-all.py")
	if _, err := os.Stat(gatePath); err == nil {
		result.AlreadyPresent = true
		return result, nil
	}

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return nil, fmt.Errorf("creating hooks dir: %w", err)
	}

	if err := os.WriteFile(gatePath, []byte(standaloneGitAddAllHook), 0755); err != nil {
		return nil, fmt.Errorf("writing hook: %w", err)
	}

	result.Created = true
	return result, nil
}

// standaloneHookSpecs defines hooks to register in standalone mode.
var standaloneHookSpecs = []hookSpec{
	{Matcher: "Bash", Script: "gate-git-add-all.py"},
}

// ensureStandaloneHookRegistration registers standalone hooks in settings.json.
// Unlike the full mode, this uses project-local hook paths.
func ensureStandaloneHookRegistration(settingsPath, hooksDir string) (*StepResult, error) {
	result := &StepResult{}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("reading settings: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}

	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		hooks = make(map[string]any)
		settings["hooks"] = hooks
	}

	var ptu []any
	if existing, ok := hooks["PreToolUse"].([]any); ok {
		ptu = existing
	}

	// Check which hooks are already registered
	registeredCommands := make(map[string]bool)
	for _, entry := range ptu {
		if group, ok := entry.(map[string]any); ok {
			if hookList, ok := group["hooks"].([]any); ok {
				for _, h := range hookList {
					if hookMap, ok := h.(map[string]any); ok {
						if cmd, ok := hookMap["command"].(string); ok {
							registeredCommands[cmd] = true
						}
					}
				}
			}
		}
	}

	registered := 0
	for _, spec := range standaloneHookSpecs {
		scriptPath := filepath.Join(hooksDir, spec.Script)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			continue
		}

		// Use relative path (.claude/hooks/...) so it works across clones.
		// Claude Code runs hooks from the project root, so relative paths resolve correctly.
		command := fmt.Sprintf("python3 .claude/hooks/%s", spec.Script)
		if registeredCommands[command] {
			continue
		}

		// Also check if already registered with an absolute path
		absCommand := fmt.Sprintf("python3 %s", scriptPath)
		if registeredCommands[absCommand] {
			continue
		}

		entry := map[string]any{
			"matcher": spec.Matcher,
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": command,
				},
			},
		}
		ptu = append(ptu, entry)
		registered++
	}

	if registered == 0 {
		result.AlreadyPresent = true
		return result, nil
	}

	hooks["PreToolUse"] = ptu

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling settings: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(settingsPath, out, 0644); err != nil {
		return nil, fmt.Errorf("writing settings: %w", err)
	}

	result.HooksRegistered = registered
	return result, nil
}

// standalonePreCommitScript is the content of the standalone pre-commit accretion gate.
// It warns when staged files exceed size thresholds, catching unbounded file growth
// before it becomes a problem.
const standalonePreCommitScript = `#!/bin/bash
# =============================================================================
# Pre-commit: Accretion Gate (standalone)
# Generated by: orch harness init
# =============================================================================
#
# == WHY THIS GATE EXISTS ==
#
# In AI-assisted development, files grow fast. An agent adding 200 lines to an
# already-large file can push it past the point where other agents (or humans)
# can effectively reason about it. This gate catches unbounded growth early.
#
# == WHAT IT CHECKS ==
#
# For each staged file:
#   - HARD BLOCK: Files exceeding 1500 lines (exit 1, commit blocked)
#   - WARNING:    Files >800 lines gaining 30+ lines (informational)
#   - WARNING:    Files >600 lines gaining 50+ lines (informational)
#
# == HOW TO BYPASS ==
#
#   SKIP_ACCRETION_GATE=1 git commit -m "..."
#
# == WHAT TO DO WHEN BLOCKED ==
#
# 1. Extract a cohesive subset into a new file (e.g., helpers, types, handlers)
# 2. Split the file along domain boundaries
# 3. If the growth is justified, bypass with the env var above
#
# =============================================================================

# Escape hatch
if [ "${SKIP_ACCRETION_GATE}" = "1" ]; then
    echo "pre-commit: accretion gate bypassed (SKIP_ACCRETION_GATE=1)"
    exit 0
fi

HARD_LIMIT=1500
WARN_THRESHOLD_HIGH=800
WARN_LINES_HIGH=30
WARN_THRESHOLD_LOW=600
WARN_LINES_LOW=50

blocked=0
warned=0

# Check each staged file
while IFS=$'\t' read -r added removed file; do
    # Skip binary files (shown as - - by git)
    [ "$added" = "-" ] && continue
    # Skip deleted files
    [ -z "$file" ] && continue
    [ ! -f "$file" ] && continue

    total=$(wc -l < "$file" 2>/dev/null || echo 0)
    total=$(echo "$total" | tr -d ' ')
    net=$((added - removed))

    if [ "$total" -gt "$HARD_LIMIT" ]; then
        echo "BLOCKED: $file is $total lines (limit: $HARD_LIMIT)"
        echo "  Extract code into smaller files before committing."
        blocked=1
    elif [ "$total" -gt "$WARN_THRESHOLD_HIGH" ] && [ "$net" -ge "$WARN_LINES_HIGH" ]; then
        echo "WARNING: $file is $total lines (+$net net lines added)"
        warned=1
    elif [ "$total" -gt "$WARN_THRESHOLD_LOW" ] && [ "$net" -ge "$WARN_LINES_LOW" ]; then
        echo "WARNING: $file is $total lines (+$net net lines added)"
        warned=1
    fi
done < <(git diff --cached --numstat)

if [ "$blocked" -eq 1 ]; then
    echo ""
    echo "Commit blocked by accretion gate. Extract large files before committing."
    echo "Bypass: SKIP_ACCRETION_GATE=1 git commit -m '...'"
    exit 1
fi

if [ "$warned" -eq 1 ]; then
    echo "pre-commit: accretion gate passed (with warnings above)"
else
    echo "pre-commit: accretion gate passed"
fi
`

// ensureStandalonePreCommitGate adds a self-contained accretion gate to .git/hooks/pre-commit.
// Unlike the full mode, this doesn't require the orch binary.
func ensureStandalonePreCommitGate(projectDir string) (*StepResult, error) {
	result := &StepResult{}

	hooksDir := filepath.Join(projectDir, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "pre-commit")

	marker := "Accretion Gate"

	// Check if already present
	data, err := os.ReadFile(hookPath)
	if err == nil && strings.Contains(string(data), marker) {
		result.AlreadyPresent = true
		return result, nil
	}

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return nil, fmt.Errorf("creating hooks dir: %w", err)
	}

	if os.IsNotExist(err) {
		// Create new pre-commit hook with the full standalone script
		if err := os.WriteFile(hookPath, []byte(standalonePreCommitScript), 0755); err != nil {
			return nil, fmt.Errorf("writing pre-commit: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("reading pre-commit: %w", err)
	} else {
		// Append a source call to the existing hook
		gate := fmt.Sprintf("\n# Accretion gate (added by orch harness init)\n%s", standalonePreCommitScript)
		f, err := os.OpenFile(hookPath, os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			return nil, fmt.Errorf("opening pre-commit: %w", err)
		}
		defer f.Close()
		if _, err := f.WriteString(gate); err != nil {
			return nil, fmt.Errorf("appending to pre-commit: %w", err)
		}
	}

	result.Created = true
	return result, nil
}

// ensurePreCommitGate adds the accretion gate to .git/hooks/pre-commit.
func ensurePreCommitGate(projectDir string) (*StepResult, error) {
	result := &StepResult{}

	hooksDir := filepath.Join(projectDir, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "pre-commit")

	accretionLine := "orch precommit accretion"

	// Check if already present
	data, err := os.ReadFile(hookPath)
	if err == nil && strings.Contains(string(data), accretionLine) {
		result.AlreadyPresent = true
		return result, nil
	}

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return nil, fmt.Errorf("creating hooks dir: %w", err)
	}

	gate := "\n# Accretion warning gate (added by orch harness init)\norch precommit accretion 2>/dev/null || true\n"

	if os.IsNotExist(err) {
		// Create new pre-commit hook
		content := "#!/bin/bash\n" + gate
		if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
			return nil, fmt.Errorf("writing pre-commit: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("reading pre-commit: %w", err)
	} else {
		// Append to existing
		f, err := os.OpenFile(hookPath, os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			return nil, fmt.Errorf("opening pre-commit: %w", err)
		}
		defer f.Close()
		if _, err := f.WriteString(gate); err != nil {
			return nil, fmt.Errorf("appending to pre-commit: %w", err)
		}
	}

	result.Created = true
	return result, nil
}
