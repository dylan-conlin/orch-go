# Session Synthesis

**Agent:** og-debug-debug-orchestrator-pretooluse-28feb-2060
**Issue:** orch-go-xm58
**Duration:** 2026-02-28
**Outcome:** success

---

## Plain-Language Summary

The orchestrator coaching nudge hook (`gate-orchestrator-code-access.py`) was invisible to Claude because it used `permissionDecisionReason` with `permissionDecision: "allow"`. Per Claude Code's hook docs, that field is only shown to the **human user** when the decision is "allow" — it's never injected into Claude's context. The fix was switching to `additionalContext`, which is always added to Claude's context as a `<system-reminder>`. This was confirmed working live during the debugging session itself — reading `.py` files triggered the coaching nudge.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — all 4 tests pass including live in-session verification.

---

## Delta (What Changed)

### Files Modified
- `~/.orch/hooks/gate-orchestrator-code-access.py` - Changed from `permissionDecisionReason` to `additionalContext` for coaching nudge delivery; updated `main()` to build output dict dynamically based on which fields are present in the result

### Commits
- (pending — will commit with this synthesis)

---

## Evidence (What Was Observed)

- **Root cause**: Claude Code's PreToolUse hook docs state: `permissionDecisionReason` — "For 'allow' and 'ask', shown to the user but not Claude. For 'deny', shown to Claude." ([source](https://code.claude.com/docs/en/hooks))
- **Why Task guard worked but code-access guard didn't**: `gate-orchestrator-task-tool.py` uses `"deny"` where `permissionDecisionReason` IS shown to Claude. `gate-orchestrator-code-access.py` uses `"allow"` where it is NOT.
- **The `additionalContext` field**: Per docs, "String added to Claude's context before the tool executes" — this is the correct mechanism for coaching nudges.
- **Live confirmation**: During this session, reading `.py` files after the fix triggered `<system-reminder>` blocks containing the coaching nudge text.

### Tests Run
```bash
# All 3 manual tests pass:
echo '{"tool_name": "Read", "tool_input": {"file_path": "/test/main.go"}}' | CLAUDE_CONTEXT=orchestrator python3 ~/.orch/hooks/gate-orchestrator-code-access.py
# → JSON with additionalContext coaching nudge ✓

echo '{"tool_name": "Read", "tool_input": {"file_path": "/test/CLAUDE.md"}}' | CLAUDE_CONTEXT=orchestrator python3 ~/.orch/hooks/gate-orchestrator-code-access.py
# → Silent (non-code file) ✓

echo '{"tool_name": "Read", "tool_input": {"file_path": "/test/main.go"}}' | CLAUDE_CONTEXT=worker python3 ~/.orch/hooks/gate-orchestrator-code-access.py
# → Silent (not orchestrator) ✓
```

---

## Architectural Choices

### Use `additionalContext` instead of `permissionDecisionReason`
- **What I chose:** `additionalContext` field in `hookSpecificOutput`
- **What I rejected:** Switching to `permissionDecision: "deny"` (which would make the reason visible to Claude but also block the read)
- **Why:** The whole point of the 2026-02-28 redesign was to allow reads while coaching. `additionalContext` achieves both: allows the read AND injects the nudge into Claude's context.
- **Risk accepted:** Coaching nudge appears as `<system-reminder>` which could be ignored by the model — but this is the intended design (coaching, not blocking).

---

## Knowledge (What Was Learned)

### Key Finding
Claude Code PreToolUse hook output has asymmetric visibility for `permissionDecisionReason`:
- With `"deny"` → reason shown to **Claude** (the agent)
- With `"allow"` or `"ask"` → reason shown to **human user** only
- `additionalContext` → always added to **Claude's context**

This means any hook that wants to "coach" the agent while allowing the action must use `additionalContext`, not `permissionDecisionReason`.

### Constraints Discovered
- PreToolUse `permissionDecisionReason` with `"allow"` is user-facing only, not agent-facing

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (manual + live verification)
- [x] Ready for `orch complete orch-go-xm58`

---

## Unexplored Questions

- The GitHub issue [#4669](https://github.com/anthropics/claude-code/issues/4669) suggests `permissionDecision: "deny"` may be broken in some versions — worth monitoring if the Task guard hook stops working
- Whether the coaching nudge frequency becomes noisy for orchestrators reading many code files in sequence (might need rate-limiting or a "first N times only" approach)

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-debug-orchestrator-pretooluse-28feb-2060/`
**Beads:** `bd show orch-go-xm58`
