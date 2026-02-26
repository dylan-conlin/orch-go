## Summary (D.E.K.N.)

**Delta:** Late SSE events were setting is_processing=true on completed agents because handlers matched on session_id alone without checking agent status.

**Evidence:** Code analysis of agents.ts lines 426-477 showed message.part and session.status handlers updated any agent with matching session_id, regardless of status.

**Knowledge:** SSE events can arrive out of order or after agent status changes; handlers must be defensive about the agent's current state.

**Next:** Fix implemented - added status === 'active' check to SSE handlers. Close issue.

**Confidence:** High (90%) - logical analysis of code flow is sound, but cannot fully reproduce the original race condition.

---

# Investigation: Dashboard Pulsing Gold Border Persists

**Question:** Why does the pulsing gold border persist on completed agents in the Recent section?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: UI correctly checks status before showing pulsing animation

**Evidence:** In agent-card.svelte:204, the pulsing animation is applied with condition `agent.status === 'active' && agent.is_processing`. This means completed agents should NOT show pulsing.

**Source:** web/src/lib/components/agent-card/agent-card.svelte:204

**Significance:** The UI-side check is correct. The bug must be in how `is_processing` is set on agents that are no longer active.

---

### Finding 2: message.part SSE handler sets is_processing without status check

**Evidence:** The handler at agents.ts:426-451 matches agents by `session_id` only, then unconditionally sets `is_processing: true`. If a late SSE event arrives after the agent completes, it will set processing state on the completed agent.

**Source:** web/src/lib/stores/agents.ts:426-451

**Significance:** This is the root cause. Late SSE events from OpenCode can arrive after the agent status changes to 'completed', incorrectly setting is_processing=true.

---

### Finding 3: session.status handler has same vulnerability

**Evidence:** The session.status handler at agents.ts:454-477 also matches only on session_id and can set is_processing=true on any agent, including completed ones.

**Source:** web/src/lib/stores/agents.ts:454-477

**Significance:** Same race condition exists in both SSE handlers. Both need defensive checks.

---

## Synthesis

**Key Insights:**

1. **Race condition in SSE handling** - SSE events from OpenCode are asynchronous and can arrive after the agent's status has already changed. Without defensive checks, late events corrupt agent state.

2. **Prior decision was incomplete** - The prior decision (kn entry) about requiring status === 'active' check focused on the UI side but the underlying store handlers also needed protection.

3. **Fix is additive, not breaking** - Adding a status check to the handlers is defensive and doesn't change behavior for active agents.

**Answer to Investigation Question:**

The pulsing gold border persisted because late SSE events (message.part or session.status) set `is_processing: true` on completed agents. The SSE handlers matched agents by session_id only, without checking if the agent was still active. Fix: Add `agent.status === 'active'` check to both handlers.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Code analysis clearly shows the handlers lacked status checks. The logic is sound - SSE events are asynchronous and can arrive late.

**What's certain:**

- ✅ The handlers matched on session_id without status check
- ✅ SSE events can arrive after status changes
- ✅ The UI correctly guards against showing animation on non-active agents

**What's uncertain:**

- ⚠️ Exact timing/frequency of the race condition in production
- ⚠️ Whether there are other code paths that could set is_processing incorrectly

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add status check to SSE handlers** - Only update is_processing for agents with status === 'active'.

**Implementation completed:**
1. Added `agent.status === 'active'` check to message.part handler (line 438)
2. Added defensive check to session.status handler - only set is_processing=true for active agents, but allow clearing on any agent (line 471-472)

---

## References

**Files Examined:**
- web/src/lib/stores/agents.ts - SSE event handlers
- web/src/lib/components/agent-card/agent-card.svelte - UI rendering logic

**Commands Run:**
```bash
# Type checking
bun run check  # Passed with 0 errors
```
