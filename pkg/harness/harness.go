// Package harness provides structural governance for Claude Code projects.
//
// It manages:
//   - Deny rules in settings.json to prevent agent modification of control plane
//   - Hook scripts that gate dangerous operations (blanket git add, etc.)
//   - Pre-commit accretion gates that block unbounded file growth
//   - OS-level immutability (chflags uchg) for control plane files
//   - Measurement reports for governance effectiveness
//
// Designed to work standalone (any project with Claude Code) or with full
// orch orchestration infrastructure.
package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/control"
)

// Mode indicates whether harness operates standalone or with full orch infrastructure.
type Mode int

const (
	// ModeStandalone works without orch/beads dependencies.
	// Uses project-level settings and self-contained hook scripts.
	ModeStandalone Mode = iota

	// ModeFull uses orch infrastructure (hooks in ~/.orch/hooks/, beads close hook, etc.)
	ModeFull
)

// StepResult captures the result of a single harness init step.
type StepResult struct {
	AlreadyPresent  bool
	RulesAdded      int
	HooksRegistered int
	Created         bool
	Error           error
}

// DetectMode checks whether the project should use standalone or full mode.
// Returns the mode and whether .beads/ exists.
// Standalone mode is used when ~/.orch/hooks/ doesn't exist (no orch infrastructure).
func DetectMode(projectDir string) (mode Mode, hasBeads bool) {
	home, _ := os.UserHomeDir()
	orchHooksDir := filepath.Join(home, ".orch", "hooks")
	_, orchErr := os.Stat(orchHooksDir)
	_, beadsErr := os.Stat(filepath.Join(projectDir, ".beads"))
	hasBeads = beadsErr == nil
	if os.IsNotExist(orchErr) {
		mode = ModeStandalone
	} else {
		mode = ModeFull
	}
	return
}

// SettingsPath returns the appropriate settings.json path for the given mode.
// Standalone mode uses project-level settings (.claude/settings.json in project dir)
// because hook scripts use relative paths that resolve from the project root.
// Full mode uses the user-level settings.json.
func SettingsPath(mode Mode, projectDir string) string {
	if mode == ModeStandalone {
		return filepath.Join(projectDir, ".claude", "settings.json")
	}
	if p := os.Getenv("ORCH_SETTINGS_PATH"); p != "" {
		return p
	}
	return control.DefaultSettingsPath()
}

// --- Deny Rules ---

// StandaloneDenyRules returns deny rules for standalone mode.
// These protect Claude Code's settings files from agent modification.
// Unlike full mode, these don't include ~/.orch/hooks/** paths.
func StandaloneDenyRules() []string {
	return []string{
		"Edit(~/.claude/settings.json)",
		"Edit(~/.claude/settings.local.json)",
		"Write(~/.claude/settings.json)",
		"Write(~/.claude/settings.local.json)",
	}
}

// EnsureDenyRules adds the given deny rules to settings.json.
// If rules is nil, uses the full set from control.DenyRules().
func EnsureDenyRules(settingsPath string, rules []string) (*StepResult, error) {
	if rules == nil {
		rules = control.DenyRules()
	}
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

// CheckDenyRules checks deny rules without modifying settings.json.
func CheckDenyRules(settingsPath string, rules []string) (*StepResult, error) {
	if rules == nil {
		rules = control.DenyRules()
	}
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

// --- Hook Scripts ---

// GitAddAllHookContent returns the content of the standalone gate-git-add-all.py hook.
const GitAddAllHookContent = `#!/usr/bin/env python3
"""
Gate: Block 'git add -A' and 'git add .' in Claude Code sessions.

Generated by: harness init

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

// EnsureHookScripts generates hook scripts in the project's .claude/hooks/ directory.
func EnsureHookScripts(hooksDir string) (*StepResult, error) {
	result := &StepResult{}

	gatePath := filepath.Join(hooksDir, "gate-git-add-all.py")
	if _, err := os.Stat(gatePath); err == nil {
		result.AlreadyPresent = true
		return result, nil
	}

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return nil, fmt.Errorf("creating hooks dir: %w", err)
	}

	if err := os.WriteFile(gatePath, []byte(GitAddAllHookContent), 0755); err != nil {
		return nil, fmt.Errorf("writing hook: %w", err)
	}

	result.Created = true
	return result, nil
}

// --- Hook Registration ---

// HookSpec defines a hook to register.
type HookSpec struct {
	Matcher string
	Script  string // filename in hooksDir
}

// StandaloneHookSpecs defines hooks to register in standalone mode.
var StandaloneHookSpecs = []HookSpec{
	{Matcher: "Bash", Script: "gate-git-add-all.py"},
}

// FullModeHookSpecs defines hooks to register in full (orch) mode.
var FullModeHookSpecs = []HookSpec{
	{Matcher: "Bash", Script: "gate-bd-close.py"},
	{Matcher: "Bash", Script: "gate-worker-git-add-all.py"},
}

// hookEquivalents maps standalone hook script names to functional identifiers.
var hookEquivalents = map[string]string{
	"gate-git-add-all.py": "git-add-all",
}

// IsEquivalentHookRegistered checks if a functionally equivalent hook is
// already registered.
func IsEquivalentHookRegistered(registeredCommands map[string]bool, scriptName string) bool {
	pattern, ok := hookEquivalents[scriptName]
	if !ok {
		return false
	}
	for cmd := range registeredCommands {
		if strings.Contains(cmd, pattern) {
			return true
		}
	}
	return false
}

// CollectRegisteredCommands reads all hook commands from a settings.json file.
func CollectRegisteredCommands(settingsPath string) map[string]bool {
	commands := make(map[string]bool)
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return commands
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return commands
	}
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return commands
	}
	ptu, ok := hooks["PreToolUse"].([]any)
	if !ok {
		return commands
	}
	for _, entry := range ptu {
		if group, ok := entry.(map[string]any); ok {
			if hookList, ok := group["hooks"].([]any); ok {
				for _, h := range hookList {
					if hookMap, ok := h.(map[string]any); ok {
						if cmd, ok := hookMap["command"].(string); ok {
							commands[cmd] = true
						}
					}
				}
			}
		}
	}
	return commands
}

// EnsureHookRegistration registers hooks in settings.json.
// For standalone mode, uses relative paths (.claude/hooks/) and checks userSettingsPath
// for equivalent hooks to avoid double-gating.
// For full mode, uses absolute paths from hooksDir.
func EnsureHookRegistration(settingsPath, hooksDir string, specs []HookSpec, useRelativePaths bool, userSettingsPath string) (*StepResult, error) {
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

	// Collect registered commands from the target settings
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

	// Also check user-level settings for equivalent hooks to avoid double-gating.
	if userSettingsPath != "" && userSettingsPath != settingsPath {
		for cmd := range CollectRegisteredCommands(userSettingsPath) {
			registeredCommands[cmd] = true
		}
	}

	registered := 0
	for _, spec := range specs {
		scriptPath := filepath.Join(hooksDir, spec.Script)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			continue
		}

		var command string
		if useRelativePaths {
			command = fmt.Sprintf("python3 .claude/hooks/%s", spec.Script)
		} else {
			command = fmt.Sprintf("python3 %s", scriptPath)
		}

		if registeredCommands[command] {
			continue
		}

		// Also check if already registered with the other path format
		var altCommand string
		if useRelativePaths {
			altCommand = fmt.Sprintf("python3 %s", scriptPath)
		} else {
			altCommand = fmt.Sprintf("python3 .claude/hooks/%s", spec.Script)
		}
		if registeredCommands[altCommand] {
			continue
		}

		// Check if a functionally equivalent hook is already registered
		if IsEquivalentHookRegistered(registeredCommands, spec.Script) {
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

// --- Pre-commit Gate ---

// AccretionGateScript is the content of the standalone pre-commit accretion gate.
const AccretionGateScript = `#!/bin/bash
# =============================================================================
# Pre-commit: Accretion Gate (standalone)
# Generated by: harness init
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

// RemoveTrailingExit removes a trailing 'exit 0' or 'exit $?' from a script
// so that appended code is reachable.
func RemoveTrailingExit(content string) string {
	trimmed := strings.TrimRight(content, " \t\n\r")
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 {
		return content
	}
	last := strings.TrimSpace(lines[len(lines)-1])
	if last == "exit 0" || last == "exit $?" {
		return strings.Join(lines[:len(lines)-1], "\n") + "\n"
	}
	return content
}

// EnsureBashShebang upgrades #!/bin/sh to #!/bin/bash.
func EnsureBashShebang(content string) string {
	if strings.HasPrefix(content, "#!/bin/sh\n") {
		return "#!/bin/bash\n" + content[len("#!/bin/sh\n"):]
	}
	if strings.HasPrefix(content, "#!/bin/sh\r\n") {
		return "#!/bin/bash\r\n" + content[len("#!/bin/sh\r\n"):]
	}
	return content
}

// StripScriptShebang removes the shebang line from a script.
func StripScriptShebang(content string) string {
	if strings.HasPrefix(content, "#!") {
		if idx := strings.Index(content, "\n"); idx >= 0 {
			return content[idx+1:]
		}
	}
	return content
}

// EnsureStandalonePreCommitGate adds a self-contained accretion gate to .git/hooks/pre-commit.
func EnsureStandalonePreCommitGate(projectDir string) (*StepResult, error) {
	result := &StepResult{}

	hooksDir := filepath.Join(projectDir, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "pre-commit")

	marker := "Accretion Gate"

	data, err := os.ReadFile(hookPath)
	if err == nil && strings.Contains(string(data), marker) {
		result.AlreadyPresent = true
		return result, nil
	}

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return nil, fmt.Errorf("creating hooks dir: %w", err)
	}

	if os.IsNotExist(err) {
		if err := os.WriteFile(hookPath, []byte(AccretionGateScript), 0755); err != nil {
			return nil, fmt.Errorf("writing pre-commit: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("reading pre-commit: %w", err)
	} else {
		content := EnsureBashShebang(string(data))
		content = RemoveTrailingExit(content)
		gateBody := StripScriptShebang(AccretionGateScript)
		newContent := content + "\n# Accretion gate (added by harness init)\n" + gateBody

		if err := os.WriteFile(hookPath, []byte(newContent), 0755); err != nil {
			return nil, fmt.Errorf("writing pre-commit: %w", err)
		}
	}

	result.Created = true
	return result, nil
}

// EnsureOrchPreCommitGate adds the orch-aware accretion gate to .git/hooks/pre-commit.
func EnsureOrchPreCommitGate(projectDir string) (*StepResult, error) {
	result := &StepResult{}

	hooksDir := filepath.Join(projectDir, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "pre-commit")

	accretionLine := "orch precommit accretion"

	data, err := os.ReadFile(hookPath)
	if err == nil && strings.Contains(string(data), accretionLine) {
		result.AlreadyPresent = true
		return result, nil
	}

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return nil, fmt.Errorf("creating hooks dir: %w", err)
	}

	gate := "\n# Accretion warning gate (added by harness init)\norch precommit accretion 2>/dev/null || true\n"

	if os.IsNotExist(err) {
		content := "#!/bin/bash\n" + gate
		if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
			return nil, fmt.Errorf("writing pre-commit: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("reading pre-commit: %w", err)
	} else {
		content := RemoveTrailingExit(string(data))
		newContent := content + gate

		if err := os.WriteFile(hookPath, []byte(newContent), 0755); err != nil {
			return nil, fmt.Errorf("writing pre-commit: %w", err)
		}
	}

	result.Created = true
	return result, nil
}

// --- Beads Close Hook ---

// EnsureBeadsCloseHook creates .beads/hooks/on_close if it doesn't exist.
func EnsureBeadsCloseHook(projectDir string) (*StepResult, error) {
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

// --- Check (Status/Health) ---

// CheckResult contains the results of a harness health check.
type CheckResult struct {
	Mode           Mode
	DenyRulesOK   bool
	DenyRuleCount int
	HooksOK       bool
	HookCount     int
	PreCommitOK   bool
	LockOK        bool // only relevant for full mode
	Issues         []string
}

// Check performs a health check of the harness configuration.
func Check(projectDir string) (*CheckResult, error) {
	mode, _ := DetectMode(projectDir)
	sp := SettingsPath(mode, projectDir)

	result := &CheckResult{Mode: mode}

	// Check deny rules
	var rules []string
	if mode == ModeStandalone {
		rules = StandaloneDenyRules()
	}
	denyResult, err := CheckDenyRules(sp, rules)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Cannot read settings: %v", err))
	} else {
		result.DenyRulesOK = denyResult.AlreadyPresent
		if !denyResult.AlreadyPresent {
			result.Issues = append(result.Issues, "Missing deny rules in settings.json")
		}
	}

	// Check hook scripts
	if mode == ModeStandalone {
		hookPath := filepath.Join(projectDir, ".claude", "hooks", "gate-git-add-all.py")
		if _, err := os.Stat(hookPath); err == nil {
			result.HooksOK = true
			result.HookCount = 1
		} else {
			result.Issues = append(result.Issues, "Missing hook script: .claude/hooks/gate-git-add-all.py")
		}
	} else {
		home, _ := os.UserHomeDir()
		orchHooksDir := filepath.Join(home, ".orch", "hooks")
		for _, spec := range FullModeHookSpecs {
			if _, err := os.Stat(filepath.Join(orchHooksDir, spec.Script)); err == nil {
				result.HookCount++
			}
		}
		result.HooksOK = result.HookCount == len(FullModeHookSpecs)
		if !result.HooksOK {
			result.Issues = append(result.Issues, fmt.Sprintf("Missing hook scripts (%d/%d)", result.HookCount, len(FullModeHookSpecs)))
		}
	}

	// Check pre-commit gate
	precommitPath := filepath.Join(projectDir, ".git", "hooks", "pre-commit")
	data, err := os.ReadFile(precommitPath)
	if err == nil {
		content := string(data)
		if mode == ModeStandalone {
			result.PreCommitOK = strings.Contains(content, "Accretion Gate")
		} else {
			result.PreCommitOK = strings.Contains(content, "orch precommit accretion")
		}
	}
	if !result.PreCommitOK {
		result.Issues = append(result.Issues, "Missing accretion gate in pre-commit hook")
	}

	// Check control plane lock (full mode only, macOS only)
	if mode == ModeFull {
		unlocked, err := control.VerifyLocked()
		if err != nil {
			result.Issues = append(result.Issues, fmt.Sprintf("Cannot verify lock state: %v", err))
		} else {
			result.LockOK = len(unlocked) == 0
			if !result.LockOK && !control.IsUnlockMarkerPresent() {
				result.Issues = append(result.Issues, fmt.Sprintf("%d control plane file(s) unlocked", len(unlocked)))
			}
		}
	}

	return result, nil
}

// ShortPath abbreviates an absolute path by replacing home dir with ~.
func ShortPath(path, home string) string {
	if home != "" && strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
