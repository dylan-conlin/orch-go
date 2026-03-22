# Probe: Hook cwd Path Resolution Failure Mode

**Model:** claude-code-agent-configuration
**Date:** 2026-03-22
**Agent:** og-debug-hook-cwd-shift-22mar-ae0c
**Issue:** orch-go-w5ais

## Question

Can relative paths in settings.json hook commands fail when Claude Code's Bash cwd differs from the project root? What is the blast radius?

## What I Tested

1. Ran project-level hook (`python3 .claude/hooks/gate-git-add-all.py`) from a subdirectory (`.harness/openscad/`) to simulate cwd shift
2. Checked Claude Code hook exit code behavior (what happens on exit code 2)
3. Tested regex false-positive with `bd create "description containing git add -A"`
4. Verified `$CLAUDE_PROJECT_DIR` as the correct path resolution mechanism

## What I Observed

### Finding 1: Relative paths + cwd shift = total Bash blockage
When python3 can't find the script (wrong cwd), it exits with code 2. Claude Code treats exit code 2 as a **deny decision**, blocking the tool call. Since the hook matches ALL Bash commands, every Bash command is blocked — not just git-add commands.

**Evidence:** Agent orch-go-4dz0u (issue orch-go-4dz0u) had all Bash commands blocked after reading `.harness/openscad/CLAUDE.md`. Only reached Planning phase, all work lost.

### Finding 2: False-positive on quoted patterns
The regex `r'\bgit\s+add\s+(-A|--all)\b'` matches anywhere in the command string, including inside quoted arguments to other commands like `bd create "description containing git add -A"`.

### Finding 3: Global hooks are immune
All `~/.orch/hooks/` hooks use `~/` prefix (absolute), so they work regardless of cwd. Only project-level hooks in `.claude/settings.json` had relative paths.

### Finding 4: control.Lock doesn't protect relative-path hooks
`DiscoverControlPlaneFiles()` skips hooks whose expanded path isn't absolute (`pkg/control/control.go:69`). The project hook's uchg was set by `harness init`, not by `control.Lock`.

## Model Impact

### New Failure Mode: Hook Path Resolution Failure (Failure Mode 4)
**Severity:** Critical (total agent blockage)
**Trigger:** Project-level hook with relative path + cwd differs from project root
**Mechanism:** python3 exits 2 (file not found) → Claude Code treats as deny → all Bash blocked
**Fix:** Use `$CLAUDE_PROJECT_DIR` in hook commands instead of relative paths
**Prevention:** `harness init` should generate `$CLAUDE_PROJECT_DIR`-based paths

### Extends Failure Mode 2: Configuration Drift Across Layers
The relative-path vulnerability is a form of configuration drift — the hook path assumes a specific cwd that isn't guaranteed. The `$CLAUDE_PROJECT_DIR` fix addresses this at the source.

### Updates to Constraints section
Add: **Why hook commands must use absolute paths** — Relative paths in settings.json hook commands resolve against the Bash tool's cwd, which may not match the project root. Use `$CLAUDE_PROJECT_DIR` or `~/` prefix for reliable resolution.

### Confirms: Zero observability (from Mar 12 audit)
The hook failure produced no log output. The agent saw "all Bash commands blocked" but had no way to diagnose WHY. This confirms the Mar 12 finding that hook infrastructure has zero observability.
