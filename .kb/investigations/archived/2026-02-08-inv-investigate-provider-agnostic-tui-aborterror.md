## Summary (D.E.K.N.)

**Delta:** The `AbortError` is triggered by OpenCode's own session cancellation path (`SessionPrompt.cancel`), and normal abort flow is handled, but queued prompts can still emit an unhandled `AbortError` rejection.

**Evidence:** Synthetic runtime tests showed a clean abort result (`MessageAbortedError` + `Tool execution aborted`) for active prompt cancellation, and a separate queued-prompt test emitted `unhandledRejection: AbortError: Aborted`.

**Knowledge:** This behavior is provider-agnostic because cancellation originates in product control flow (`session.abort`, task cancellation), and stability risk appears tied to promise rejection handling in queued/caller paths.

**Next:** Implement queued-cancel rejection hardening with regression tests, then validate in interactive TUI.

**Authority:** implementation - localized cancellation handling and test coverage work in existing session architecture.

---

# Investigation: Investigate Provider Agnostic Tui Aborterror

**Question:** Who cancels the abort signal that causes `DOMException AbortError` during tool execution, and why does TUI sometimes crash instead of degrading gracefully?

**Started:** 2026-02-08
**Updated:** 2026-02-08
**Owner:** OpenCode worker (gpt-5.3-codex)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Defect Class:** race-condition (manual entry; `kb create investigation --defect-class` unsupported in current CLI)

**Patches-Decision:** N/A
**Extracted-From:** `/Users/dylanconlin/Documents/personal/opencode`

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/archived/2025-12-24-inv-model-provider-architecture-orch-vs.md` | extends | yes | none |

---

## Findings

### Finding 1: Abort origin is internal session cancellation

**Evidence:** `SessionPrompt.cancel(sessionID)` calls `abort()` on the active controller and rejects queued callbacks with `DOMException("Aborted", "AbortError")`. This is reached from `session.abort` route and from task tool cancel wiring.

**Source:** `../opencode/packages/opencode/src/session/prompt.ts:245`, `../opencode/packages/opencode/src/session/prompt.ts:253`, `../opencode/packages/opencode/src/session/prompt.ts:255`, `../opencode/packages/opencode/src/server/routes/session.ts:358`, `../opencode/packages/opencode/src/server/routes/session.ts:382`, `../opencode/packages/opencode/src/tool/task.ts:145`

**Significance:** Confirms provider-agnostic ownership of abort.

---

### Finding 2: Active abort path is handled as designed

**Evidence:** Synthetic `SessionProcessor` repro returned:

```json
{
  "result": "stop",
  "toolState": "error",
  "toolError": "Tool execution aborted",
  "assistantError": "MessageAbortedError"
}
```

**Source:** runtime probe command in `../opencode` with mocked `LLM.stream`; processing behavior from `../opencode/packages/opencode/src/session/processor.ts:81`, `../opencode/packages/opencode/src/session/processor.ts:369`, `../opencode/packages/opencode/src/session/message-v2.ts:722`

**Significance:** Core abort mapping survives expected cancellation and should not crash by itself.

---

### Finding 3: Queued cancel can emit unhandled rejection

**Evidence:** Synthetic queued prompt + cancel probe captured:

```json
{
  "unhandledRejections": 1,
  "sample": "AbortError: Aborted"
}
```

Mechanism: if `SessionPrompt.loop()` is already active, it queues callbacks in `state()[sessionID].callbacks`; `cancel()` rejects those callbacks with DOMException.

**Source:** runtime probe command in `../opencode`; queue/callback logic from `../opencode/packages/opencode/src/session/prompt.ts:64`, `../opencode/packages/opencode/src/session/prompt.ts:255`, `../opencode/packages/opencode/src/session/prompt.ts:265`

**Significance:** Strongest tested crash/stability candidate under race conditions.

---

## Synthesis

**Key Insights:**

1. **Provider-agnostic by construction** - abort is produced by local session-control paths.
2. **Active cancellation path is resilient** - normal interruption state is persisted and rendered cleanly.
3. **Race path remains risky** - queued callback rejection can escape as unhandled promise rejection.

**Answer to Investigation Question:**

The abort signal is canceled by OpenCode's own control flow (`SessionPrompt.cancel`) rather than by provider-specific behavior. The TUI generally survives because abort is mapped to `MessageAbortedError` and `Tool execution aborted`, and UI rendering suppresses noisy abort toasts. The likely instability vector is queued-prompt cancellation producing unhandled `AbortError` rejection when callers do not catch the promise in a race. I did not fully instrument an interactive TUI run to directly capture renderer teardown at the moment of rejection.

---

## Structured Uncertainty

**What's tested:**

- ✅ Active abort path returns clean interrupted state (`MessageAbortedError` + tool error)
- ✅ `SessionPrompt.cancel` is the owning abort source for interactive/task paths
- ✅ Queued prompt cancellation can emit unhandled rejection if uncaught

**What's untested:**

- ⚠️ Exact UI event chain from unhandled rejection to full TUI process exit
- ⚠️ Full call-site matrix of all prompt/command submitters for catch consistency
- ⚠️ Runtime-version differences in unhandled rejection severity

**What would change this:**

- Interactive trace proving repeated queued-cancel cannot destabilize TUI
- Evidence that all queued callers already catch cancel rejections
- Evidence of provider-side abort bypassing local cancel path

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Harden queued cancellation to avoid uncaught `AbortError` | implementation | Local session prompt behavior and tests |

### Recommended Approach

**Queued-Cancel Rejection Hardening** - make queued-cancel outcomes cancellation-safe without unhandled promise rejection.

**Why this approach:**
- Directly targets the only reproduced instability path
- Preserves existing interrupt semantics
- Avoids unsupported provider-specific complexity

**Trade-offs accepted:**
- Defer deeper renderer tracing until after prompt-layer fix lands
- Do not globally suppress all abort errors

**Implementation sequence:**
1. Add failing regression test for queued prompt + cancel unhandled rejection
2. Update callback rejection/resolve behavior in `SessionPrompt.cancel` or queue contract
3. Audit high-risk caller paths for explicit catch handling

---

## References

**Files Examined:**
- `../opencode/packages/opencode/src/session/prompt.ts`
- `../opencode/packages/opencode/src/session/processor.ts`
- `../opencode/packages/opencode/src/session/message-v2.ts`
- `../opencode/packages/opencode/src/server/routes/session.ts`
- `../opencode/packages/opencode/src/tool/task.ts`
- `../opencode/packages/opencode/src/cli/cmd/tui/routes/session/index.tsx`
- `../opencode/packages/opencode/src/cli/cmd/tui/app.tsx`

**Commands Run:**
```bash
kb create investigation investigate-provider-agnostic-tui-aborterror --defect-class race-condition
kb create investigation investigate-provider-agnostic-tui-aborterror
git show 0d841aeaca24b83eaaf333184278cc51ae50a201
bun -e '<SessionProcessor abort repro>'
bun -e '<queued prompt cancel + unhandledRejection probe>'
```

**Related Artifacts:**
- Issue: `orch-go-5ho4r`
- Follow-up issue: `orch-go-n773h`
- Workspace: `.orch/workspace/og-inv-investigate-provider-agnostic-08feb-2f9d/`

---

## Investigation History

**2026-02-08 16:20:** Investigation started
- Goal: identify abort owner and crash path for provider-agnostic `AbortError`

**2026-02-08 16:40:** Cancellation source traced
- Verified `session.abort` and task cancellation both call `SessionPrompt.cancel`

**2026-02-08 16:50:** Active abort flow reproduced
- Confirmed clean interrupted assistant/tool state and no backend failure in synthetic test

**2026-02-08 17:00:** Race path reproduced
- Confirmed queued cancel can emit unhandled rejection (`AbortError: Aborted`)

**2026-02-08 17:05:** Investigation completed
- Outcome: provider-agnostic source confirmed; queued-cancel rejection is strongest tested instability candidate
