# Spike: Claude Code Hooks for Orchestrator Action Enforcement

**Question:** Can Claude Code hooks intercept tool calls for orchestrator action enforcement? Is PreToolUse feasible for blocking prohibited tool usage (Task tool, Edit/Write, bd close) in orchestrator sessions? Is there a simpler built-in mechanism?

**Started:** 2026-02-24
**Updated:** 2026-02-24
**Owner:** feature-impl (orch-go-1181)
**Phase:** Complete
**Status:** Complete

**Parent:** `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` (Layer 2)

---

## TL;DR: Two Mechanisms, Use Both

Claude Code provides **two** enforcement mechanisms for Layer 2:

1. **`--disallowedTools` CLI flag** — removes tools entirely from the agent's toolset at spawn time. Simpler, more deterministic, zero runtime cost. Best for blanket tool restrictions (Task, Edit, Write, NotebookEdit).

2. **`PreToolUse` hook** — intercepts tool calls at runtime with conditional logic. Best for command-level restrictions within an allowed tool (e.g., blocking `bd close` within Bash while allowing other Bash commands).

**Recommended approach:** `--disallowedTools` as primary enforcement (Layer 2a) + PreToolUse hook for Bash command gating (Layer 2b). Implementation: ~1 hour total — one-line change to `BuildClaudeLaunchCommand` + small hook for bd close.

---

## Findings

### Finding 1: `--disallowedTools` CLI Flag Removes Tools at Spawn Time

**Evidence:** The `claude` CLI accepts `--disallowedTools` (alias `--disallowed-tools`) to deny specific tools for the entire session:

```bash
claude --disallowedTools "Task,Edit,Write,NotebookEdit" --dangerously-skip-permissions
```

This removes the tools from the agent's available toolset entirely — the agent never sees them, can't attempt to use them, and receives no error. The tools simply don't exist in that session's tool inventory.

Additionally, `--allowedTools` (whitelist) and `--tools` (explicit full list) are available for even stricter control.

**Source:** `claude --help` output, Claude Code permissions documentation

**Significance:** This is strictly superior to hooks for blanket tool restrictions because:
- Zero runtime overhead (no hook fires per tool call)
- No bypass risk (tool doesn't exist, not just denied)
- No error/denial messages to confuse the agent
- No Python script to maintain
- Scoped to spawned sessions only (doesn't affect interactive sessions)

---

### Finding 2: orch-go Already Has the Injection Point

**Evidence:** `pkg/spawn/claude.go:71` constructs the Claude launch command:

```go
return fmt.Sprintf("export CLAUDE_CONTEXT=%s; cat %q | claude --dangerously-skip-permissions%s",
    claudeContext, contextPath, mcpFlag)
```

Adding `--disallowedTools` is a one-line change. The `claudeContext` variable already distinguishes orchestrator from worker sessions (line 104-112):

```go
switch {
case cfg.IsMetaOrchestrator:
    claudeContext = "meta-orchestrator"
case cfg.IsOrchestrator:
    claudeContext = "orchestrator"
default:
    claudeContext = "worker"
}
```

**Source:** `pkg/spawn/claude.go` lines 57-72, 76-129

**Significance:** The implementation is trivial — add a `disallowedTools` string based on `cfg.IsOrchestrator` and append it to the command.

---

### Finding 3: PreToolUse Hook Needed Only for Bash Command Gating

**Evidence:** `--disallowedTools` works at the tool level (Task, Edit, Write) but cannot distinguish between Bash commands. Orchestrators need Bash for legitimate commands (`orch spawn`, `orch complete`, `bd create`, `bd show`, `kb context`, `git status`), but should be blocked from specific Bash commands like `bd close`.

PreToolUse hooks can inspect the `tool_input.command` field for Bash calls:

```json
{
  "tool_name": "Bash",
  "tool_input": { "command": "bd close orch-go-1234" }
}
```

An existing hook (`gate-bd-close.py`) already demonstrates this exact pattern — reading stdin JSON, checking `CLAUDE_CONTEXT`, inspecting the command via regex, and returning `permissionDecision: "deny"` with a reason.

**Source:** `~/.orch/hooks/gate-bd-close.py`, Claude Code hooks documentation

**Significance:** The hook approach is needed only for the `bd close` → `orch complete` enforcement. All other tool-level restrictions use the simpler `--disallowedTools` flag.

---

### Finding 4: PreToolUse Has Full Interception Capability

**Evidence:** Claude Code supports 17 hook event types. `PreToolUse` fires before every tool call and provides:

```json
{
  "session_id": "abc123",
  "tool_name": "Task",
  "tool_input": {
    "description": "Research codebase",
    "prompt": "...",
    "subagent_type": "general-purpose"
  },
  "tool_use_id": "toolu_01ABC123...",
  "cwd": "/Users/dylanconlin/Documents/personal/orch-go",
  "hook_event_name": "PreToolUse"
}
```

Response mechanisms:
1. **`permissionDecision: "deny"`** — Cancels the tool call, sends reason to Claude as error
2. **`permissionDecision: "allow"`** — Proceeds without permission prompt
3. **`permissionDecision: "ask"`** — Shows permission prompt to user
4. **Exit code 2** — Blocks action, stderr fed back to Claude
5. **`updatedInput`** — Can modify tool input before execution

Three handler types: `command` (shell script), `prompt` (single LLM call), `agent` (multi-turn subagent).

**Source:** Claude Code hooks documentation, verified against existing hooks in `~/.claude/settings.json`

**Significance:** PreToolUse is the right mechanism for command-level gating within Bash.

---

### Finding 5: Session Detection Already Implemented

**Evidence:** `CLAUDE_CONTEXT` env var is set by `orch spawn` (`pkg/spawn/claude.go:104-112`):
- `"worker"` — worker agents (feature-impl, investigation, etc.)
- `"orchestrator"` — orchestrator agents
- `"meta-orchestrator"` — meta-orchestrator agents
- Empty/unset — interactive Claude Code sessions (human at keyboard)

Multiple existing hooks already read this: `load-orchestration-context.py`, `gate-bd-close.py`.

**Source:** `pkg/spawn/claude.go`, `~/.orch/hooks/load-orchestration-context.py`, `~/.orch/hooks/gate-bd-close.py`

**Significance:** No new detection mechanism needed. Both `--disallowedTools` (via Config flags) and hooks (via env var) can detect orchestrator sessions.

---

### Finding 6: `--disallowedTools` vs PreToolUse Hook — Head-to-Head

**Evidence:** Comparison of the two enforcement mechanisms:

| Dimension | `--disallowedTools` (CLI flag) | PreToolUse hook |
|---|---|---|
| **Enforcement** | Tool removed from toolset | Tool call intercepted at runtime |
| **Granularity** | Per-tool (Task, Edit, Write) | Per-command within a tool |
| **Bypass risk** | None (tool doesn't exist) | None (hook fires every call) |
| **Error UX** | Agent never sees tool | Agent tries, gets denied + reason |
| **Runtime cost** | Zero | ~50-100ms per matching tool call |
| **Implementation** | One-line change to spawn code | ~50 line Python script |
| **Scope** | Only spawned sessions (not interactive) | All sessions (needs CLAUDE_CONTEXT check) |
| **Exceptions** | No (blanket per-tool) | Yes (conditional logic) |
| **Maintenance** | Inline with Go code | Separate Python script |
| **"Pain as signal"** | No feedback (tool absent) | Denial reason injected into context |

**Source:** Claude Code CLI help, hooks documentation, existing hook implementations

**Significance:** `--disallowedTools` is superior for tool-level blocking (simpler, no runtime cost, no bypass). PreToolUse is superior for command-level blocking (can inspect Bash commands). Using both covers the full enforcement surface.

---

## Revised Design: Hybrid Approach

### Layer 2a: `--disallowedTools` at Spawn Time (Primary)

**Change in `pkg/spawn/claude.go`:**

```go
func BuildClaudeLaunchCommand(contextPath, claudeContext, mcp string) string {
    mcpFlag := ""
    // ... existing MCP logic ...

    // Orchestrator tool restrictions: remove worker-level tools
    disallowFlag := ""
    if claudeContext == "orchestrator" || claudeContext == "meta-orchestrator" {
        disallowFlag = " --disallowedTools 'Task,Edit,Write,NotebookEdit'"
    }

    return fmt.Sprintf("export CLAUDE_CONTEXT=%s; cat %q | claude --dangerously-skip-permissions%s%s",
        claudeContext, contextPath, mcpFlag, disallowFlag)
}
```

**Tools blocked for orchestrators:**
- `Task` — must use `orch spawn` / `bd create -l triage:ready`
- `Edit` — orchestrators don't edit code
- `Write` — orchestrators don't create code files
- `NotebookEdit` — orchestrators don't edit notebooks

**Tools remaining available:**
- `Bash` — needed for `orch spawn`, `orch complete`, `bd create`, `bd show`, `kb context`, `git status`
- `Read` — needed for reading .kb/ files, CLAUDE.md, SYNTHESIS.md
- `Glob` — needed for finding knowledge artifacts
- `Grep` — needed for searching knowledge artifacts
- `WebFetch` / `WebSearch` — needed for research

**Write exception for .kb/ files:** Not needed with `--disallowedTools` because orchestrators write knowledge files via Bash (`kb quick decide`, etc.) and don't typically use the Write tool directly. If this becomes a problem, fall back to the hook approach for Write only.

### Layer 2b: PreToolUse Hook for `bd close` (Supplementary)

**Modify existing `gate-bd-close.py`** or create new `orchestrator-guard.py`:

```python
# In PreToolUse hook for Bash matcher:
# Block 'bd close' for orchestrator sessions → redirect to 'orch complete'

context = os.environ.get("CLAUDE_CONTEXT", "")
if context not in ("orchestrator", "meta-orchestrator"):
    return None  # Only gate orchestator sessions

command = input_data.get("tool_input", {}).get("command", "")

if re.match(r'^\s*bd\s+close\b', command):
    return {
        "permissionDecision": "deny",
        "permissionDecisionReason": (
            "⚠️ ORCHESTRATOR ACTION VIOLATION: Use `orch complete`, not `bd close`.\n\n"
            "`orch complete <agent-id>` runs verification gates before closing:\n"
            "  - Gate 1 (explain-back): What was built and why?\n"
            "  - Gate 2 (behavioral): Is the behavior verified?\n\n"
            "`bd close` bypasses these gates. Always use `orch complete` for agent work."
        )
    }
```

This can either extend the existing `gate-bd-close.py` (which currently only gates worker architect/orchestrator skills without kn entries) or be a new script.

---

## Structured Uncertainty

**What's tested:**
- ✅ `--disallowedTools` flag exists and accepts tool names (verified via `claude --help`)
- ✅ `BuildClaudeLaunchCommand` is the single injection point for CLI flags (verified via code)
- ✅ `CLAUDE_CONTEXT` env var exists and differentiates session types (verified via code)
- ✅ PreToolUse hook can deny Bash commands with contextual reasons (verified via gate-bd-close.py)

**What's untested:**
- ⚠️ Whether `--disallowedTools` works correctly when piped via stdin (`cat file | claude --disallowedTools ...`) — standard CLI behavior, likely fine but untested
- ⚠️ Whether removing the Task tool causes the agent to attempt workarounds (e.g., using Bash to run `claude` directly) — unlikely but possible
- ⚠️ Whether orchestrators need Write tool for VERIFICATION_SPEC.yaml or SYNTHESIS.md — if so, either allow Write or have workers handle these files
- ⚠️ Whether `--disallowedTools` interacts correctly with `--dangerously-skip-permissions` — both are permission-related flags

**What would change this:**
- If `--disallowedTools` doesn't work with piped input, fall back to hook-only approach
- If orchestrators need Write for specific files, use hook with exceptions instead of blanket `--disallowedTools`
- If `--dangerously-skip-permissions` overrides `--disallowedTools`, use hooks as the enforcement layer

---

## Implementation Recommendations

### Recommended: Hybrid `--disallowedTools` + Hook

**Authority:** implementation (follows from Layer 2 in parent investigation)

**Effort:** ~1 hour total

**Step 1 (Layer 2a):** Modify `pkg/spawn/claude.go` (~10 lines)
- Add `disallowedTools` logic to `BuildClaudeLaunchCommand`
- Update `BuildClaudeLaunchCommand` tests in `claude_test.go`
- Gate on `claudeContext == "orchestrator" || claudeContext == "meta-orchestrator"`

**Step 2 (Layer 2b):** Modify `~/.orch/hooks/gate-bd-close.py` (~20 lines)
- Add orchestrator-specific `bd close` → `orch complete` gating
- Currently gates on `CLAUDE_CONTEXT == "worker"` for kn entries; add `"orchestrator"` check for bd close

**Step 3:** Manual test
- Spawn an orchestrator session
- Verify Task tool is absent
- Verify Edit/Write tools are absent
- Verify `bd close` is denied with reason
- Verify `orch spawn`, `bd create`, `kb context` all work

### Risk: Write Tool for Knowledge Files

If orchestrators use Write tool for `.kb/` files (investigation files, probes), `--disallowedTools "Write"` would break this. Mitigations:
1. **Primary:** Orchestrators write knowledge via `kb quick` commands (Bash), not Write tool
2. **Fallback:** Remove Write from `--disallowedTools` and use hook with .kb/ exception instead
3. **Monitor:** After deployment, check if orchestrator sessions fail on Write attempts

---

## References

**Files Examined:**
- `pkg/spawn/claude.go` — BuildClaudeLaunchCommand, SpawnClaude (injection point for --disallowedTools)
- `~/.claude/settings.json` — Current hook configuration (13+ hooks across 6 event types)
- `~/.orch/hooks/gate-bd-close.py` — Existing PreToolUse deny pattern
- `~/.orch/hooks/load-orchestration-context.py` — Session type detection via CLAUDE_CONTEXT
- `claude --help` — CLI flags including --disallowedTools, --allowedTools, --tools, --permission-mode

**Parent Investigation:**
- `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` — Layer 2 recommendation that this spike validates

**Key API Surface:**
- CLI: `--disallowedTools` (blanket tool removal), `--allowedTools` (whitelist), `--tools` (explicit list), `--permission-mode`
- PreToolUse: matcher (regex on tool name), stdin JSON (tool_name, tool_input), response (permissionDecision: allow|deny|ask)
- Environment: CLAUDE_CONTEXT, CLAUDE_PROJECT_DIR, CLAUDE_ENV_FILE
- Handler types: command (shell), prompt (single LLM), agent (multi-turn subagent)
