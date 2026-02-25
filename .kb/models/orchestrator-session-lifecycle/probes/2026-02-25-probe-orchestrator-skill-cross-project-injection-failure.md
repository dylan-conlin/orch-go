# Probe: Orchestrator Skill Cross-Project Injection Failure

**Model:** orchestrator-session-lifecycle
**Date:** 2026-02-25
**Status:** Complete

---

## Question

The model documents that skill content reaches orchestrator sessions via `load-orchestration-context.py` (SessionStart hook). But launching Claude Code via `cc personal` in non-orch-go projects (e.g., toolshed) fails to inject the orchestrator SKILL.md. Why?

Prior probe (2026-02-17) mapped 5 injection paths but didn't test the cross-project failure case. This probe specifically traces the `cc personal` → hook → model path in projects without `.orch/` directories.

---

## What I Tested

### 1. Traced the `cc()` function in ~/.zshrc

```bash
cc() {
  local account="${1:?Usage: cc <personal|work> [claude args...]}"
  shift
  case "$account" in
    personal) export CLAUDE_CONFIG_DIR=~/.claude-personal ;;
    work)     unset CLAUDE_CONFIG_DIR ;;
  esac
  local context="orchestrator"  # Default
  # ... flag parsing ...
  CLAUDE_CONTEXT="$context" claude "${args[@]}"
}
```

**Key observation:** Sets `CLAUDE_CONTEXT=orchestrator` for ALL interactive sessions by default.

### 2. Verified settings.json loaded from correct config dir

Both `~/.claude/settings.json` and `~/.claude-personal/settings.json` contain identical SessionStart hooks, including `load-orchestration-context.py`. So the hook IS registered regardless of account.

### 3. Simulated the hook's spawn detection logic

```bash
$ echo '{"source":"startup"}' | CLAUDE_CONTEXT=orchestrator python3 -c "
import os
ctx = os.environ.get('CLAUDE_CONTEXT', '')
print(f'is_spawned_agent={ctx in (\"worker\", \"orchestrator\", \"meta-orchestrator\")}')
"
# Output: is_spawned_agent=True
```

**ROOT CAUSE #1 CONFIRMED:** `is_spawned_agent()` at line 496-497 checks `CLAUDE_CONTEXT in ('worker', 'orchestrator', 'meta-orchestrator')`. When `cc personal` sets `CLAUDE_CONTEXT=orchestrator`, the hook incorrectly identifies the interactive session as a spawned agent and exits at line 505-506 without injecting anything.

### 4. Verified the .orch directory gate

```bash
$ ls ~/Documents/personal/toolshed/.orch
# NO .orch dir
```

`find_orch_directory()` (line 56-70) traverses from cwd upward looking for `.orch/`. For projects without this directory, it returns None, causing the hook to exit at line 522-524.

**ROOT CAUSE #2 CONFIRMED:** Even if Bug 1 were fixed, the orchestrator skill (project-independent, at `~/.claude/skills/meta/orchestrator/SKILL.md`) is gated behind the `.orch/` directory check. The `main()` function exits before loading the skill if no `.orch/` directory exists.

### 5. Checked spawn env var landscape

```bash
# ORCH_WORKER=1 is set by:
# - pkg/tmux/tmux.go (BuildOpencodeAttachCommand, BuildSpawnCommand)
# - pkg/spawn/backends/inline.go
# - pkg/opencode/client.go (HTTP header)

# ORCH_SPAWNED is NOT set anywhere
# CLAUDE_CONTEXT is set by:
# - pkg/spawn/claude.go:BuildClaudeLaunchCommand (spawned agents)
# - cc() function in ~/.zshrc (interactive sessions)
```

**Key finding:** `ORCH_WORKER=1` exists for OpenCode-backend spawns but not for Claude CLI spawns. The `is_spawned_agent()` function checks `CLAUDE_CONTEXT` because it's the only env var common to all spawn paths. But `CLAUDE_CONTEXT` is also set by the interactive `cc()` launcher — hence the collision.

### 6. Checked Claude Code native skill discovery

Claude Code 2.1.56 does NOT natively discover `~/.claude/skills/meta/orchestrator/SKILL.md` as a user-invocable skill. The skill discovery mechanism in Claude Code only surfaces specific skill types (e.g., `keybindings-help` was the only skill visible in a test session). The orchestrator SKILL.md is injected solely via hooks.

---

## What I Observed

### Complete Failure Chain (Interactive Orchestrator in Non-Orch-Go Project)

```
cc personal (in ~/Documents/personal/toolshed)
  ↓ sets CLAUDE_CONTEXT=orchestrator
  ↓ sets CLAUDE_CONFIG_DIR=~/.claude-personal
claude starts
  ↓ reads ~/.claude-personal/settings.json
  ↓ fires SessionStart hooks
load-orchestration-context.py executes
  ↓ line 496: is_spawned_agent() → True (CLAUDE_CONTEXT=orchestrator)
  ↓ line 505: sys.exit(0)  ← EXITS WITHOUT INJECTING SKILL

Result: Claude operates as generic assistant, not orchestrator
```

### The Conflation

`CLAUDE_CONTEXT=orchestrator` is used for TWO distinct purposes:

| Purpose | Set By | Consumed By |
|---------|--------|-------------|
| Skip duplicate skill injection for spawned agents | `orch spawn` (claude.go:87) | `load-orchestration-context.py:496` |
| Gate code file access for orchestrators | `cc personal` (zshrc) + `orch spawn` | `gate-orchestrator-code-access.py:114` |

The env var conflates "is an orchestrator" with "was spawned by orch". Interactive orchestrators need the skill injected AND the code access gate — but currently get neither because the hook exits early.

### Two Approaches Evaluated

**Approach A: Fix the hook mechanism**
- Add `export ORCH_SPAWNED=1;` to `BuildClaudeLaunchCommand()` in claude.go
- Change `is_spawned_agent()` to check `ORCH_SPAWNED=1` (not CLAUDE_CONTEXT)
- Restructure `main()` to load skill before the `.orch/` gate

**Approach B: Inject via `cc()` wrapper using `--append-system-prompt`**
- `cc()` would cat the skill file and pass via `--append-system-prompt`
- Simple and direct, no hook dependency

**Recommendation: Approach A** because:
1. The hook is the established mechanism for context injection
2. It handles both skill (static) and dynamic state (frontier, beads, roadmap)
3. Approach B creates double-injection in orch-go projects where the hook also fires
4. The `--append-system-prompt` flag with ~19KB content is fragile for shell argument limits
5. Two clear bugs with clear fixes — not a design problem, just logic errors

---

## Model Impact

- [x] **Extends** model with: The 2026-02-17 injection path trace documented 5 paths but did not test the cross-project interactive case. Two additional failure gates exist: (1) `is_spawned_agent()` conflates spawned agents with interactive orchestrators due to shared `CLAUDE_CONTEXT=orchestrator` env var, and (2) project-independent skill loading is gated behind project-specific `.orch/` directory existence.

- [x] **Extends** model with: The constraint "Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading" is only partially implemented — `ORCH_WORKER=1` exists for OpenCode-backend spawns but the hook checks `CLAUDE_CONTEXT` instead of `ORCH_WORKER`. For Claude CLI spawns, neither `ORCH_WORKER` nor `ORCH_SPAWNED` is set; only `CLAUDE_CONTEXT` is used, creating the conflation bug.

- [x] **Contradicts** model assumption: The model implies orchestrator sessions always have skill content available. In practice, interactive orchestrator sessions in non-orch-go projects receive NO skill content due to the double-gate failure. This is the root cause of the "generic collaborator" behavior reported in orch-go-1233.

---

## Recommended Fix (3 Changes)

### Change 1: Add ORCH_SPAWNED to spawn command (claude.go)

```go
// BuildClaudeLaunchCommand line 87:
// Before:
return fmt.Sprintf("%sexport CLAUDE_CONTEXT=%s; cat %q | claude ...")
// After:
return fmt.Sprintf("%sexport ORCH_SPAWNED=1; export CLAUDE_CONTEXT=%s; cat %q | claude ...")
```

### Change 2: Fix spawn detection (load-orchestration-context.py)

```python
# Before (line 496-497):
def is_spawned_agent():
    ctx = os.environ.get('CLAUDE_CONTEXT', '')
    return ctx in ('worker', 'orchestrator', 'meta-orchestrator')

# After:
def is_spawned_agent():
    return os.environ.get('ORCH_SPAWNED') == '1' or os.environ.get('ORCH_WORKER') == '1'
```

### Change 3: Restructure main() to decouple skill from .orch gate

```python
def main():
    if is_spawned_agent():
        sys.exit(0)

    input_data = json.load(sys.stdin)
    if input_data.get('source') not in ('startup', 'resume'):
        sys.exit(0)

    context_parts = ["# 🎯 Orchestration Context\n", "*Auto-loaded via SessionStart hook*\n\n"]

    # Load skill FIRST — project-independent
    skill_content = load_orchestrator_skill()
    if skill_content:
        context_parts.append("---\n\n")
        context_parts.append(skill_content)

    # Load dynamic state — project-dependent (requires .orch/)
    orch_dir = find_orch_directory()
    if orch_dir:
        # ... existing dynamic state loading (beads, frontier, roadmap, etc.) ...

    # Output if we have anything beyond the header
    if len(context_parts) > 2:
        # ... existing output logic ...
```

---

## Cross-Reference

- Prior probe: `2026-02-17-orchestrator-skill-injection-path-trace.md` (mapped all 5 injection paths)
- Constraint: "Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading"
- Issue: orch-go-1233
- Gate hook: `~/.orch/hooks/gate-orchestrator-code-access.py` (uses CLAUDE_CONTEXT for code access gating — unaffected by this fix)
