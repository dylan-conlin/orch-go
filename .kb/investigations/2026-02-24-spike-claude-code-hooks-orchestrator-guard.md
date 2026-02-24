# Spike: Claude Code Hooks for Orchestrator Action Enforcement

**Question:** Can Claude Code hooks intercept tool calls for orchestrator action enforcement? Is PreToolUse feasible for blocking prohibited tool usage (Task tool, Edit/Write, bd close) in orchestrator sessions?

**Started:** 2026-02-24
**Updated:** 2026-02-24
**Owner:** feature-impl (orch-go-1181)
**Phase:** Complete
**Status:** Complete

**Parent:** `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` (Layer 2)

---

## TL;DR: Fully Feasible

Claude Code's `PreToolUse` hook can intercept any tool call, see tool name + input, and **deny execution with a reason message injected into the agent's context**. This is exactly the mechanism needed for Layer 2 enforcement. An existing hook (`gate-bd-close.py`) already demonstrates the pattern. Session detection via `CLAUDE_CONTEXT` env var is already implemented.

**Estimated implementation effort:** ~2 hours (one Python script + settings.json update)

---

## Findings

### Finding 1: PreToolUse Hook Has Full Interception Capability

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

Three response mechanisms available:
1. **`permissionDecision: "deny"`** — Cancels the tool call, sends reason to Claude as error
2. **`permissionDecision: "allow"`** — Proceeds without permission prompt
3. **`permissionDecision: "ask"`** — Shows permission prompt to user
4. **Exit code 2** — Blocks action, stderr fed back to Claude
5. **`updatedInput`** — Can modify tool input before execution (not needed here)

**Source:** Claude Code hooks documentation, verified against existing hooks in `~/.claude/settings.json`

**Significance:** PreToolUse provides everything Layer 2 needs: tool name identification, input inspection, and denial with contextual feedback.

---

### Finding 2: Matchers Target Specific Tools by Name

**Evidence:** Hook configuration uses regex matchers on tool names:

```json
{
  "PreToolUse": [
    { "matcher": "Task", "hooks": [...] },
    { "matcher": "Edit|Write", "hooks": [...] },
    { "matcher": "Bash", "hooks": [...] }
  ]
}
```

Tool names that can be matched: `Task`, `Edit`, `Write`, `Read`, `Bash`, `Glob`, `Grep`, `NotebookEdit`, `mcp__*` (MCP tools).

**Source:** `~/.claude/settings.json` — existing hooks already use matchers: `"Bash"`, `"Edit|Write"`, `"Read|Glob|Grep"`

**Significance:** Each prohibited tool can have a targeted hook. No need for a single catch-all — matchers provide precise targeting.

---

### Finding 3: Existing Hook Demonstrates Exact Pattern Needed

**Evidence:** `~/.orch/hooks/gate-bd-close.py` already:
1. Reads JSON from stdin (tool_name, tool_input)
2. Checks `CLAUDE_CONTEXT` env var for session type
3. Inspects bash command content via regex
4. Returns `permissionDecision: "deny"` with detailed reason

Key detection code (line 124-126):
```python
context = os.environ.get("CLAUDE_CONTEXT", "")
if context != "worker":
    return None
```

The orchestrator-guard hook inverts this: instead of gating workers, it gates orchestrators.

**Source:** `~/.orch/hooks/gate-bd-close.py` (191 lines, fully functional)

**Significance:** The implementation pattern is proven and production-tested. The new hook is structurally identical — only the detection logic and tool targets differ.

---

### Finding 4: Session Detection Already Implemented

**Evidence:** `CLAUDE_CONTEXT` env var is set by `orch spawn`:
- `"worker"` — worker agents (feature-impl, investigation, etc.)
- `"orchestrator"` — orchestrator agents
- `"meta-orchestrator"` — meta-orchestrator agents
- Empty/unset — interactive Claude Code sessions (human at keyboard)

The `load-orchestration-context.py` hook (line 496-497) already reads this:
```python
ctx = os.environ.get('CLAUDE_CONTEXT', '')
return ctx in ('worker', 'orchestrator', 'meta-orchestrator')
```

**Source:** `~/.orch/hooks/load-orchestration-context.py`, `~/.orch/hooks/gate-bd-close.py`

**Significance:** No new detection mechanism needed. `CLAUDE_CONTEXT == "orchestrator"` is the reliable signal.

---

### Finding 5: Three Handler Types Available (Command, Prompt, Agent)

**Evidence:** Claude Code hooks support three handler types:

| Type | Description | Use Case |
|------|-------------|----------|
| `command` | Shell script, JSON stdin/stdout | Deterministic rules (our case) |
| `prompt` | Single LLM call (Haiku default) | Fuzzy intent detection |
| `agent` | Multi-turn subagent with tools | Complex evaluation |

For orchestrator-guard, `command` type is correct — the rules are deterministic (if orchestrator + prohibited tool → deny). No LLM evaluation needed.

**Source:** Claude Code hooks documentation

**Significance:** `command` type is fastest (<100ms), most reliable, and appropriate for enforcement rules.

---

### Finding 6: Denial Reason Is Injected as Context to Claude

**Evidence:** When a PreToolUse hook returns `permissionDecision: "deny"`, the `permissionDecisionReason` string is fed back to Claude as an error message. This means the agent:
1. Sees that its tool call was blocked
2. Receives the reason explaining WHY it was blocked
3. Gets guidance on what to do instead

This is the "pain as signal" pattern from the investigation — the violation message appears in the agent's context immediately when the prohibited tool is used.

**Source:** Claude Code docs; verified behavior via `gate-bd-close.py` which uses this pattern

**Significance:** The hook doesn't just block — it teaches. The denial reason can include the correct alternative (`orch spawn` instead of Task tool), creating real-time behavioral correction.

---

## Design: Orchestrator-Guard Hook

### Architecture

```
PreToolUse event
    │
    ├── matcher: "Task"
    │   └── orchestrator-guard.py → deny if CLAUDE_CONTEXT == "orchestrator"
    │       reason: "Use orch spawn, not Task tool"
    │
    ├── matcher: "Edit|Write"
    │   └── orchestrator-guard.py → deny if CLAUDE_CONTEXT == "orchestrator"
    │       reason: "Orchestrators don't edit code. Delegate via orch spawn"
    │
    └── matcher: "Bash" (extends existing)
        └── orchestrator-guard.py → deny if CLAUDE_CONTEXT == "orchestrator"
            and command matches "bd close" → reason: "Use orch complete, not bd close"
```

### Single Script, Multiple Matchers

One script handles all three cases — the `tool_name` field in stdin differentiates:

```python
#!/usr/bin/env python3
"""
Orchestrator-Guard Hook: Enforce action space constraints for orchestrator sessions.

PreToolUse hook that blocks orchestrator agents from using worker-level tools:
- Task tool → should use orch spawn
- Edit/Write → orchestrators don't edit code
- bd close → should use orch complete

Detection: CLAUDE_CONTEXT == "orchestrator" (set by orch spawn)
"""

TOOL_VIOLATIONS = {
    "Task": {
        "reason": (
            "⚠️ ORCHESTRATOR ACTION VIOLATION: Task tool is not available to orchestrators.\n\n"
            "Orchestrators delegate work via:\n"
            "  - `orch spawn SKILL \"task\"` — spawn a worker agent\n"
            "  - `bd create --title \"...\" -l triage:ready` — create work for daemon to spawn\n\n"
            "The Task tool launches Claude Code subagents. Orchestrators use orch spawn\n"
            "which provides skill context, beads tracking, and verification gates."
        )
    },
    "Edit": {
        "reason": (
            "⚠️ ORCHESTRATOR ACTION VIOLATION: Edit tool is not available to orchestrators.\n\n"
            "Orchestrators don't write or edit code. To make code changes:\n"
            "  - `orch spawn feature-impl \"description\"` — delegate to a worker\n"
            "  - `orch spawn systematic-debugging \"description\"` — delegate debugging\n\n"
            "Your role: ORIENT → DELEGATE → RECONNECT (never implement)."
        )
    },
    "Write": {
        "reason": (
            "⚠️ ORCHESTRATOR ACTION VIOLATION: Write tool is not available to orchestrators.\n\n"
            "Orchestrators don't create code files. To create files:\n"
            "  - `orch spawn feature-impl \"description\"` — delegate to a worker\n\n"
            "Exception: Writing to .kb/ files (investigations, decisions) is allowed."
        )
    }
}

# Bash command patterns that are prohibited for orchestrators
BASH_VIOLATIONS = {
    r"^\s*bd\s+close\b": {
        "reason": (
            "⚠️ ORCHESTRATOR ACTION VIOLATION: Use `orch complete`, not `bd close`.\n\n"
            "`orch complete <agent-id>` runs verification gates before closing:\n"
            "  - Gate 1 (explain-back): What was built and why?\n"
            "  - Gate 2 (behavioral): Is the behavior verified?\n\n"
            "`bd close` bypasses these gates. Always use `orch complete` for agent work."
        )
    }
}
```

### Settings.json Integration

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Task",
        "hooks": [
          {
            "type": "command",
            "command": "$HOME/.orch/hooks/orchestrator-guard.py",
            "timeout": 5
          }
        ]
      },
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "$HOME/.orch/hooks/orchestrator-guard.py",
            "timeout": 5
          }
        ]
      },
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "$HOME/.orch/hooks/orchestrator-guard.py",
            "timeout": 5
          },
          {
            "type": "command",
            "command": "$HOME/.orch/hooks/gate-bd-close.py",
            "timeout": 10
          },
          {
            "type": "command",
            "command": "$HOME/.orch/hooks/pre-commit-knowledge-gate.py",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
```

### Exceptions (Write Tool)

The Write tool denial should exclude `.kb/` paths — orchestrators legitimately write investigation files, decisions, and probes:

```python
if tool_name == "Write":
    file_path = tool_input.get("file_path", "")
    if "/.kb/" in file_path or file_path.endswith(".md"):
        return None  # Allow knowledge file writes
```

### Graduated Response (Future Enhancement)

The investigation recommends graduated response (warn → stronger warn → block). The initial implementation should use immediate denial because:
1. Graduated response requires state tracking across tool calls (complexity)
2. The denial message IS the "pain as signal" — it teaches the correct behavior
3. If needed later, a counter could be added via a temp file per session

---

## Structured Uncertainty

**What's tested:**
- ✅ PreToolUse hook can deny tool calls with contextual reasons (verified via existing gate-bd-close.py)
- ✅ `CLAUDE_CONTEXT` env var exists and is set correctly by orch spawn
- ✅ Matchers can target specific tools by name (verified via settings.json)
- ✅ Denial reason is injected into agent context (documented behavior, verified via existing hooks)

**What's untested:**
- ⚠️ Whether the Task tool matcher actually fires (no existing PreToolUse hook targets "Task" — all existing ones target "Bash", "Edit|Write", "Read|Glob|Grep")
- ⚠️ Whether denying Task tool causes graceful degradation or confuses the agent
- ⚠️ Performance impact of adding a 4th PreToolUse hook (should be negligible at 5s timeout)
- ⚠️ Whether Write tool exception for .kb/ paths is sufficient (might need SYNTHESIS.md, VERIFICATION_SPEC.yaml paths too)

**What would change this:**
- If Claude Code hooks don't fire for subagent-spawning tools (would need to verify Task specifically)
- If `CLAUDE_CONTEXT` is not available inside hook processes (would need SessionStart hook to set it via CLAUDE_ENV_FILE)

---

## Implementation Recommendations

### Recommendation: Build the orchestrator-guard.py hook

**Authority:** implementation (follows directly from Layer 2 recommendation in parent investigation)

**Effort:** ~2 hours
- Script: ~100 lines Python (modeled on gate-bd-close.py)
- Settings: Add 2 new PreToolUse entries + extend existing Bash entry
- Testing: Manual test with orchestrator session

**Implementation order:**
1. Create `~/.orch/hooks/orchestrator-guard.py`
2. Add PreToolUse entries for `Task` and `Edit|Write` matchers
3. Add orchestrator-guard.py to existing `Bash` matcher
4. Test with `CLAUDE_CONTEXT=orchestrator` in a test session
5. Deploy to orchestrator sessions via orch spawn env var

### Risk: CLAUDE_CONTEXT Propagation

Verify that `CLAUDE_CONTEXT` is available to hook processes. The env var is set in the spawning environment, but hooks might run in a different process context. If not available:
- **Fallback:** Use SessionStart hook to detect orchestrator context and write to `$CLAUDE_ENV_FILE` for persistence
- **Alternative:** Check for orchestrator skill markers in the transcript path or cwd

---

## References

**Files Examined:**
- `~/.claude/settings.json` — Current hook configuration (13+ hooks across 6 event types)
- `~/.orch/hooks/gate-bd-close.py` — Existing PreToolUse deny pattern (exact template for new hook)
- `~/.orch/hooks/load-orchestration-context.py` — Session type detection via CLAUDE_CONTEXT
- Claude Code hooks documentation — PreToolUse API, matchers, response format

**Parent Investigation:**
- `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` — Layer 2 recommendation that this spike validates

**Key API Surface:**
- PreToolUse: matcher (regex on tool name), stdin JSON (tool_name, tool_input), response (permissionDecision: allow|deny|ask)
- Environment: CLAUDE_CONTEXT, CLAUDE_PROJECT_DIR, CLAUDE_ENV_FILE
- Handler types: command (shell), prompt (single LLM), agent (multi-turn subagent)
