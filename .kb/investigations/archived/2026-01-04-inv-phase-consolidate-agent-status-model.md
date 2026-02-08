## Summary (D.E.K.N.)

**Delta:** Consolidated getDisplayState logic from agent-card.svelte into computeDisplayState in agents.ts.

**Evidence:** Added computeDisplayState function and DisplayState type to agents.ts; agent-card.svelte now imports and uses these instead of local duplicate.

**Knowledge:** Agent display state computation is a domain concern that belongs in the agent store, not in individual components.

**Next:** close - implementation complete, ready for visual verification.

---

# Investigation: Phase Consolidate Agent Status Model

**Question:** How to consolidate agent status model by moving display state logic to agents.ts?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Duplicate DisplayState Logic Existed

**Evidence:** agent-card.svelte contained a local `getDisplayState` function (lines 22-59) that computed display state from agent status, phase, and activity.

**Source:** `web/src/lib/components/agent-card/agent-card.svelte:22-59`

**Significance:** This logic should be centralized in the agent store for reuse and consistency across components.

---

### Finding 2: agents.ts is the Right Location

**Evidence:** agents.ts already contains the Agent interface, AgentState type, and all agent-related derived stores (activeAgents, completedAgents, etc.).

**Source:** `web/src/lib/stores/agents.ts:1-59`

**Significance:** Placing computeDisplayState here maintains cohesion with related agent types and makes it available to any component needing display state.

---

## Implementation

Added to agents.ts after the Agent interface:

```typescript
// Display state for agent cards - derived from agent status + phase + activity
export type DisplayState = 'running' | 'ready-for-review' | 'idle' | 'waiting' | 'completed' | 'abandoned';

export function computeDisplayState(agent: Agent): DisplayState {
  if (agent.status === 'completed') return 'completed';
  if (agent.status === 'abandoned') return 'abandoned';
  
  if (agent.status === 'active') {
    if (agent.phase?.toLowerCase() === 'complete') return 'ready-for-review';
    if (agent.is_processing) return 'running';
    if (agent.current_activity?.timestamp) {
      const idleMs = Date.now() - agent.current_activity.timestamp;
      if (idleMs > 60000) return 'idle';
    }
    return 'waiting';
  }
  
  return 'waiting';
}
```

Updated agent-card.svelte:
- Import: `import { selectedAgentId, computeDisplayState, type DisplayState } from '$lib/stores/agents';`
- Usage: `$: displayState = computeDisplayState(agent);`
- Removed: Local DisplayState type and getDisplayState function

---

## References

**Files Modified:**
- `web/src/lib/stores/agents.ts` - Added DisplayState type and computeDisplayState function
- `web/src/lib/components/agent-card/agent-card.svelte` - Replaced local logic with import

**Commands Run:**
```bash
git diff --stat web/
# 2 files changed, 45 insertions(+), 41 deletions(-)
```
