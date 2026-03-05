<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** PreToolUse hooks CAN match 'Agent' as a tool name — AND the current `--disallowedTools 'Task,...'` is a live bug that blocks a non-existent tool name, leaving orchestrators ungated from Agent tool usage.

**Evidence:** Claude Code docs confirm Agent is a matchable tool name; the tool was renamed from Task to Agent; hook unit test passed (deny JSON output correct for orchestrator context); existing matchers (Bash, Read|Edit|Write) use identical patterns.

**Knowledge:** `--disallowedTools` and PreToolUse hooks are complementary — disallowedTools removes the tool entirely (zero cost, no bypass), PreToolUse provides runtime feedback ("pain as signal"). Both should be used for defense-in-depth.

**Next:** Fix the live bug: change `'Task,Edit,Write,NotebookEdit'` to `'Agent,Edit,Write,NotebookEdit'` in `pkg/spawn/claude.go:105`. Optionally add a PreToolUse hook for defense-in-depth. Route through architect (hotspot area).

**Authority:** architectural - Cross-component fix (spawn code + hook infrastructure), affects all orchestrator spawns

---

# Investigation: PreToolUse Hook Feasibility for Gating Agent Tool in Orchestrator Context

**Question:** Can Claude Code PreToolUse hooks match 'Agent' as a tool name? If yes, can we build a gate that blocks Agent tool usage when CLAUDE_CONTEXT=orchestrator?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** investigation (orch-go-wozz5)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** Post-mortem orch-go-ny84l (R3)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-02-24-spike-claude-code-hooks-orchestrator-guard.md | extends | yes | **CRITICAL**: Prior investigation used "Task" as the Agent tool name — tool was renamed since then, making the `--disallowedTools 'Task,...'` recommendation from that investigation a no-op for Agent blocking |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** The prior investigation's findings about PreToolUse mechanics (deny/allow/ask responses, matcher patterns, CLAUDE_CONTEXT detection) are all still valid. Only the tool name "Task" → "Agent" rename invalidates the specific `--disallowedTools` recommendation.

---

## Findings

### Finding 1: The Agent Tool Was Renamed from "Task" — Current Blocking is a No-Op

**Evidence:** The prior investigation (2026-02-24) shows this PreToolUse input:
```json
{
  "tool_name": "Task",
  "tool_input": {
    "description": "Research codebase",
    "prompt": "...",
    "subagent_type": "general-purpose"
  }
}
```

The current Claude Code system prompt (verified in this session) defines the tool as `Agent` with identical input fields (`prompt`, `description`, `subagent_type`). Claude Code documentation confirms the full list of matchable tool names: `Bash, Edit, Write, Read, Glob, Grep, Agent, WebFetch, WebSearch`.

The current code at `pkg/spawn/claude.go:105`:
```go
disallowFlag = " --disallowedTools 'Task,Edit,Write,NotebookEdit'"
```

This blocks `Task` (which no longer exists) instead of `Agent`. Edit, Write, and NotebookEdit are still correctly named.

**Source:** `pkg/spawn/claude.go:105`, Claude Code documentation, this session's system prompt tool definitions

**Significance:** **This is a live bug.** Orchestrator agents can freely use the Agent tool to spawn subagents, bypassing spawn gates, skill loading, beads tracking, and all the enforcement infrastructure built for `orch spawn`. The test cases at `pkg/spawn/claude_test.go:238,251` also assert "Task" is present in the command, so they'd need updating too.

---

### Finding 2: PreToolUse Hooks Can Match "Agent" — Confirmed by Docs and Unit Test

**Evidence:** Created `~/.orch/hooks/test-agent-hook.py` and tested with simulated Agent tool input:

```bash
# Worker context — logs only, no deny
echo '{"tool_name":"Agent",...}' | python3 ~/.orch/hooks/test-agent-hook.py
# Output: logs to file, no stdout

# Orchestrator context — deny with reason
CLAUDE_CONTEXT=orchestrator echo '{"tool_name":"Agent",...}' | \
  CLAUDE_CONTEXT=orchestrator python3 ~/.orch/hooks/test-agent-hook.py
# Output:
# {"hookSpecificOutput": {"hookEventName": "PreToolUse",
#   "permissionDecision": "deny",
#   "permissionDecisionReason": "ORCHESTRATOR GATE: Agent tool is not available..."}}
```

The hook correctly:
- Reads `tool_name` from stdin JSON
- Checks `CLAUDE_CONTEXT` environment variable
- Returns `permissionDecision: "deny"` with actionable reason for orchestrators
- Passes through silently for worker context

Claude Code docs explicitly list `Agent` as a matchable tool name for PreToolUse hooks.

**Source:** `~/.orch/hooks/test-agent-hook.py` (test hook), Claude Code hooks documentation

**Significance:** A PreToolUse hook with `"matcher": "Agent"` would work identically to existing hooks (`"matcher": "Bash"`, `"matcher": "Read|Edit|Write"`). The infrastructure is proven.

---

### Finding 3: Two Complementary Enforcement Mechanisms — Both Should Be Used

**Evidence:** Comparison matrix (validated from prior investigation + current findings):

| Mechanism | What it does | Agent experience | Runtime cost | Bypass risk |
|---|---|---|---|---|
| `--disallowedTools "Agent"` | Removes Agent from toolset at spawn time | Tool simply doesn't exist | Zero | None — tool absent |
| PreToolUse hook on "Agent" | Intercepts Agent calls at runtime, denies with reason | Gets denial + corrective instruction | ~50-100ms per call | None — hook fires every call |

The `--disallowedTools` approach is strictly superior for enforcement (tool doesn't exist = can't be tried). The PreToolUse hook adds "pain as signal" value — if an orchestrator somehow gets the Agent tool (e.g., interactive session, --disallowedTools not set), the hook provides the corrective feedback directing them to `orch spawn`.

**Source:** Prior investigation Finding 6, Claude Code CLI documentation

**Significance:** Defense-in-depth: `--disallowedTools "Agent"` as primary (zero-cost, deterministic), PreToolUse hook as secondary (catches edge cases, provides coaching). This follows the existing pattern for code access (Edit/Write blocked by --disallowedTools, coaching nudge provided by gate-orchestrator-code-access.py).

---

### Finding 4: Settings.json Configuration for Agent Hook is Trivial

**Evidence:** Current `~/.claude/settings.json` has PreToolUse hooks for `"Bash"` (8 hooks) and `"Read|Glob|Grep"` (1 hook). Adding an Agent matcher follows the identical pattern:

```json
{
  "matcher": "Agent",
  "hooks": [
    {
      "type": "command",
      "command": "$HOME/.orch/hooks/gate-orchestrator-agent-tool.py",
      "timeout": 10
    }
  ]
}
```

No changes needed to settings.json structure. The hook script at `~/.orch/hooks/test-agent-hook.py` is a working prototype (30 lines).

**Source:** `~/.claude/settings.json`, `~/.orch/hooks/test-agent-hook.py`

**Significance:** Implementation is trivial — rename test hook, add settings.json entry.

---

## Synthesis

**Key Insights:**

1. **Tool rename created a silent regression** — The "Task" → "Agent" rename happened after the prior investigation (2026-02-24) that established the `--disallowedTools` pattern. The code and tests were never updated, so orchestrators have been ungated from Agent tool usage since the rename.

2. **Defense-in-depth is the right pattern** — `--disallowedTools` handles the common case (spawned orchestrators), PreToolUse hook handles edge cases (interactive sessions, settings overrides). This mirrors the existing Edit/Write pattern.

3. **The bug has not been noticed** — likely because the orchestrator skill text says "DO NOT use the Agent tool" and agents generally comply. But this is exactly the kind of soft prohibition that fails under pressure (the post-mortem trigger for this investigation).

**Answer to Investigation Question:**

Yes, PreToolUse hooks can match 'Agent' as a tool name. The hook prototype works correctly — it receives Agent tool calls via stdin JSON, checks CLAUDE_CONTEXT, and returns deny with corrective reason. Additionally, a critical bug was discovered: the current `--disallowedTools` flag blocks the non-existent "Task" tool instead of "Agent", meaning orchestrators are currently ungated from Agent tool usage.

---

## Structured Uncertainty

**What's tested:**

- ✅ PreToolUse hook can process Agent tool input JSON (verified: test-agent-hook.py unit test)
- ✅ Hook correctly denies in orchestrator context with actionable reason (verified: CLAUDE_CONTEXT=orchestrator test)
- ✅ Hook correctly passes through in worker context (verified: worker context test)
- ✅ "Agent" is a documented matchable tool name for PreToolUse (verified: Claude Code docs)
- ✅ "Task" does not appear in current Claude Code tool list (verified: system prompt, docs)
- ✅ Current `--disallowedTools 'Task,...'` is blocking a non-existent tool (verified: code + docs)

**What's untested:**

- ⚠️ Integration test of `"matcher": "Agent"` in settings.json with a real Claude session (can't run nested Claude sessions)
- ⚠️ Whether `--disallowedTools "Agent"` correctly removes Agent from spawned session toolset (can't test from within Claude session)
- ⚠️ Whether any orchestrator has actually used the Agent tool in production (would need to audit event logs or transcripts)

**What would change this:**

- If "Task" is still accepted as an alias for "Agent" in `--disallowedTools`, the bug severity is lower (but docs say no)
- If Claude Code changes the tool name again, both mechanisms need updating
- If interactive orchestrator sessions need Agent tool access, the PreToolUse hook needs a bypass flag

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Fix `--disallowedTools` Task→Agent rename | architectural | Cross-component fix affecting all orchestrator spawns, touches hotspot spawn code |
| Add PreToolUse hook for Agent gating | architectural | New hook in shared settings.json, affects all sessions |

**Authority Levels:**
- **architectural**: Both changes affect orchestrator infrastructure across all spawns/sessions

### Recommended Approach ⭐

**Hybrid Fix: `--disallowedTools "Agent"` + PreToolUse Hook** — Fix the live bug and add defense-in-depth

**Why this approach:**
- Fixes the immediate regression (Task→Agent rename in `--disallowedTools`)
- Adds runtime coaching for edge cases (interactive sessions without --disallowedTools)
- Follows established pattern (identical to Edit/Write enforcement + gate-orchestrator-code-access.py)
- Both changes are trivial (~5 line code change + ~30 line hook script)

**Trade-offs accepted:**
- PreToolUse hook adds ~50-100ms overhead per Agent tool call in orchestrator sessions (acceptable — Agent calls are rare and this is coaching, not hot-path)
- Two enforcement mechanisms to maintain (mitigated: both are simple, test-covered)

**Implementation sequence:**
1. **Fix bug** — Change `'Task,Edit,Write,NotebookEdit'` to `'Agent,Edit,Write,NotebookEdit'` in `pkg/spawn/claude.go:105`
2. **Update tests** — Change "Task" to "Agent" in `pkg/spawn/claude_test.go:238,251`
3. **Rename hook** — Move `~/.orch/hooks/test-agent-hook.py` to `~/.orch/hooks/gate-orchestrator-agent-tool.py`
4. **Add settings entry** — Add `"Agent"` matcher to PreToolUse in `~/.claude/settings.json`
5. **Rebuild + verify** — `make test && make install`

### Alternative Approaches Considered

**Option B: `--disallowedTools` fix only (no hook)**
- **Pros:** Simpler, single change, zero runtime overhead
- **Cons:** No defense-in-depth, no feedback for interactive sessions
- **When to use instead:** If hook maintenance overhead is a concern

**Option C: Hook only (no `--disallowedTools` fix)**
- **Pros:** Provides feedback ("pain as signal"), works for all sessions
- **Cons:** Agent tool still appears in toolset (can confuse agents), runtime overhead
- **When to use instead:** Never — --disallowedTools is strictly superior for spawned sessions

**Rationale for recommendation:** Option A (hybrid) provides both deterministic enforcement (tool absent) and coaching feedback (denial reason). This matches the established pattern for Edit/Write gating.

---

### Implementation Details

**What to implement first:**
- The `--disallowedTools` bug fix (1-line change in `claude.go:105`) — this is the critical fix
- Test update (2 lines in `claude_test.go`) — prevent regression

**Things to watch out for:**
- ⚠️ The existing test cases at lines 238 and 251 assert "Task" is present — these will FAIL when the fix is applied (expected — update them to "Agent")
- ⚠️ If `--disallowedTools` uses comma-separated parsing, ensure "Agent" doesn't collide with any other tool name prefix
- ⚠️ Need to verify the hook doesn't fire for workers' use of Agent tool (agent spawning subagents is legitimate in worker context)

**Success criteria:**
- ✅ `go test ./pkg/spawn/...` passes with updated tool name
- ✅ `BuildClaudeLaunchCommand` output contains "Agent" not "Task" for orchestrator context
- ✅ Hook script returns deny for orchestrator context, passes through for worker
- ✅ Spawned orchestrator session does NOT have Agent tool available

---

## References

**Files Examined:**
- `pkg/spawn/claude.go:105` — Current `--disallowedTools` with stale "Task" name
- `pkg/spawn/claude_test.go:238,251` — Test cases asserting "Task" presence
- `~/.claude/settings.json` — Current hook configuration (PreToolUse matchers)
- `~/.orch/hooks/gate-orchestrator-code-access.py` — Existing PreToolUse hook pattern (coaching nudge)
- `~/.orch/hooks/gate-bd-close.py` — Existing PreToolUse deny pattern
- `.kb/investigations/2026-02-24-spike-claude-code-hooks-orchestrator-guard.md` — Prior investigation

**Commands Run:**
```bash
# Unit test hook with Agent tool input
echo '{"tool_name":"Agent",...}' | python3 ~/.orch/hooks/test-agent-hook.py

# Unit test hook with orchestrator context (deny output)
CLAUDE_CONTEXT=orchestrator echo '{"tool_name":"Agent",...}' | \
  CLAUDE_CONTEXT=orchestrator python3 ~/.orch/hooks/test-agent-hook.py
```

**External Documentation:**
- Claude Code hooks documentation — Confirmed Agent as matchable tool name

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-02-24-spike-claude-code-hooks-orchestrator-guard.md` — Prior work on hook infrastructure
- **Post-mortem:** `orch-go-ny84l` — Trigger for this investigation (R3)
