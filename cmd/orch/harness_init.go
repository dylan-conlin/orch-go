package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
  2. Hook registration — register gate-bd-close and gate-worker-git-add-all
  3. Beads close hook — emit completion events on bd close
  4. Pre-commit gate — accretion warnings on file growth
  5. Control plane lock — chflags uchg on settings.json and hooks

This command is idempotent — safe to run multiple times.
It unlocks the control plane, applies changes, then re-locks.

Prerequisites:
  - orch init (structural scaffold) must have been run first
  - Hook scripts must exist in ~/.orch/hooks/
  - .beads/ must be initialized

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
	hooksDir := filepath.Join(home, ".orch", "hooks")

	// Check prerequisites
	if _, err := os.Stat(sp); os.IsNotExist(err) {
		return fmt.Errorf("settings.json not found at %s\nRun Claude Code first to create it", sp)
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".beads")); os.IsNotExist(err) {
		return fmt.Errorf(".beads/ not found — run 'orch init' first")
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

	// Step 1: Deny rules
	fmt.Fprintf(os.Stderr, "1. Deny rules\n")
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

	// Step 2: Hook registration
	fmt.Fprintf(os.Stderr, "2. Hook registration\n")
	if harnessInitDryRun {
		hookResult, _ := checkHookRegistration(sp, hooksDir)
		if hookResult.AlreadyPresent {
			fmt.Fprintf(os.Stderr, "   SKIP — hooks already registered\n")
		} else {
			fmt.Fprintf(os.Stderr, "   WOULD REGISTER hooks in settings.json\n")
		}
	} else {
		hookResult, err := ensureHookRegistration(sp, hooksDir)
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

	// Step 3: Beads close hook
	fmt.Fprintf(os.Stderr, "3. Beads close hook\n")
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

	// Step 4: Pre-commit accretion gate
	fmt.Fprintf(os.Stderr, "4. Pre-commit accretion gate\n")
	if _, err := os.Stat(filepath.Join(projectDir, ".git")); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "   SKIP — not a git repository\n")
	} else if harnessInitDryRun {
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

	// Step 5: Lock control plane
	fmt.Fprintf(os.Stderr, "5. Control plane lock\n")
	if harnessInitDryRun {
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
		fmt.Fprintf(os.Stderr, "\nNext: Run 'orch harness verify' to confirm.\n")
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

# Emit the agent.completed event
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
