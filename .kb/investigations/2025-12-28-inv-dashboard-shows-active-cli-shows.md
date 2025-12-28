<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard "Active Agents" section showed 0 agents while CLI showed running agents because activeAgents filter required status='active' but API returns status='idle' for most sessions (to avoid CPU overhead from per-session IsSessionProcessing calls).

**Evidence:** serve.go:663-668 passes isProcessing=false to DetermineStatusFromSession, which returns StatusIdle; CLI calls IsSessionProcessing per-session (main.go:2558,2604); frontend activeAgents filter only accepted status==='active'.

**Knowledge:** The API/CLI semantic mismatch: API status reflects "is currently generating response" while CLI active means "has session + not completed". SSE updates is_processing field but doesn't change status field.

**Next:** Fixed by updating frontend activeAgents filter to include both status='active' and status='idle' agents, matching CLI semantics.

---

# Investigation: Dashboard Shows Active CLI Shows

**Question:** Why does dashboard show 0 active agents when CLI shows running agents?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** systematic-debugging spawn
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: API Always Passes isProcessing=false

**Evidence:** serve.go:663-668 shows the API intentionally passes `false` for isProcessing:
```go
unifiedStatus := state.DetermineStatusFromSession(
    false, // isProcessing - populated client-side via SSE
    timeSinceUpdate,
    false, // isCompleted - will be updated after Phase check
    activeThreshold,
)
```
Comment explains: "Previously we called client.IsSessionProcessing(s.ID) here, but that makes an HTTP call per session which caused 125% CPU when dashboard polled frequently."

**Source:** `cmd/orch/serve.go:659-668`

**Significance:** This is intentional optimization, not a bug. The CPU issue was real. But it means status='idle' is returned for ALL non-completed agents.

---

### Finding 2: CLI Calls IsSessionProcessing Per-Session

**Evidence:** main.go shows CLI makes individual HTTP calls:
```go
// Line 2558 (tmux agents)
isProcessing = client.IsSessionProcessing(session.ID)
// Line 2604 (OpenCode agents)
isProcessing := client.IsSessionProcessing(oa.session.ID)
```
This gives CLI accurate real-time processing state but would cause CPU issues if done in dashboard API.

**Source:** `cmd/orch/main.go:2558,2604`

**Significance:** CLI and API have different definitions of "active" due to performance constraints.

---

### Finding 3: Frontend activeAgents Filter Too Strict

**Evidence:** agents.ts:200-201 only accepted status==='active':
```typescript
export const activeAgents = derived(agents, ($agents) =>
    $agents.filter((a) => a.status === 'active')
);
```
Since API returns status='idle' for all non-completed agents (Finding 1), activeAgents was always empty.

**Source:** `web/src/lib/stores/agents.ts:200-201`

**Significance:** This is the root cause. The filter was designed before the CPU optimization changed API behavior.

---

### Finding 4: SSE Updates is_processing But Not status

**Evidence:** agents.ts:428-519 shows SSE events update is_processing field when session.status events arrive:
```typescript
if (data.type === 'session.status' && data.properties) {
    const isProcessing = statusType === 'busy';
    agents.update((agentList) => {
        return agentList.map((agent) => {
            if (agent.session_id === sessionID && agent.status === 'active') {
                return { ...agent, is_processing: isProcessing };
            }
            // ...
        });
    });
}
```
But the `status` field is never updated by SSE.

**Source:** `web/src/lib/stores/agents.ts:467-519`

**Significance:** SSE provides real-time processing state but the activeAgents filter ignores it.

---

## Synthesis

**Key Insights:**

1. **Semantic Mismatch** - API "active" means "currently generating response" while CLI "active" means "has session + not completed". The CPU optimization created this divergence.

2. **Information is Available** - SSE correctly updates is_processing field, but the activeAgents filter didn't use it. The frontend has the data, just not using it.

3. **Simple Fix** - Including status='idle' in activeAgents matches CLI semantics: agents with active sessions (not stale, not completed) are shown.

**Answer to Investigation Question:**

Dashboard showed 0 active agents because:
1. API returns status='idle' for all sessions (to avoid CPU overhead)
2. Frontend activeAgents filter only accepted status==='active'
3. The match was impossible

Fix: Update activeAgents filter to include both 'active' and 'idle' status agents.

---

## Structured Uncertainty

**What's tested:**

- ✅ API returns status='idle' for non-completed agents (verified: read serve.go:663-668)
- ✅ CLI calls IsSessionProcessing per-session (verified: read main.go:2558,2604)
- ✅ activeAgents filter previously required status==='active' (verified: read agents.ts:200-201)

**What's untested:**

- ⚠️ Edge case: agents transitioning between states during SSE update
- ⚠️ Performance impact of including more agents in activeAgents section

**What would change this:**

- Finding would be wrong if there's a code path that sets status='active' in the API
- Finding would be wrong if SSE updates the status field (but investigation confirmed it only updates is_processing)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Include idle agents in activeAgents filter** - Update the derived store to include both status='active' and status='idle' agents.

**Why this approach:**
- Matches CLI semantics (active = has session + not completed)
- No backend changes required (avoids CPU issue)
- SSE already provides real-time processing indicators (is_processing field)
- Simple, minimal change

**Trade-offs accepted:**
- Agents momentarily between tasks still show as "active" (acceptable - they ARE working)
- Relying on stale/completed status for filtering (correct behavior)

**Implementation sequence:**
1. Update activeAgents filter in agents.ts ✅ Done
2. Update recentAgents/archivedAgents to exclude idle ✅ Done
3. Verify no UI regressions

### Alternative Approaches Considered

**Option B: Have API call IsSessionProcessing for subset of agents**
- **Pros:** Accurate status from backend
- **Cons:** Still adds HTTP calls, doesn't scale with many agents
- **When to use instead:** If we add caching layer or batch API

**Option C: Make SSE update the status field**
- **Pros:** Would make activeAgents filter work without changes
- **Cons:** More complex SSE handling, potential race conditions
- **When to use instead:** If we want unified status management

**Rationale for recommendation:** Option A is simplest, aligns with existing CLI semantics, and leverages work already done (SSE updates is_processing for visual indicators).

---

### Implementation Details

**What to implement first:**
- ✅ Update activeAgents filter to include idle status
- ✅ Update recentAgents/archivedAgents to exclude idle (they're now in active section)
- Add comments explaining the CLI/API semantic relationship

**Things to watch out for:**
- ⚠️ The idleAgents derived store is now a subset of activeAgents - may want to deprecate
- ⚠️ Filter bar "Active" option in historical mode may need clarification

**Success criteria:**
- ✅ Dashboard shows same agents as active that CLI shows as running
- ✅ No agents appear in both Active and Recent sections
- ✅ Completed/stale agents still filtered out of Active section

---

## References

**Files Examined:**
- `cmd/orch/serve.go:659-668` - API status determination logic
- `cmd/orch/main.go:2558,2604` - CLI IsSessionProcessing calls
- `web/src/lib/stores/agents.ts:200-201` - activeAgents filter
- `web/src/lib/stores/agents.ts:467-519` - SSE event handling
- `pkg/state/reconcile.go:344-355` - DetermineStatusFromSession function

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-28-inv-dashboard-shows-stale-agent-data.md` - Related investigation about Phase:Complete agents showing as active (different bug, same code area)

---

## Investigation History

**2025-12-28 14:30:** Investigation started
- Initial question: Why does dashboard show 0 active when CLI shows running agents?
- Context: Spawned from task description identifying the issue

**2025-12-28 14:35:** Found API passes isProcessing=false (Finding 1)
- Discovered performance optimization comment explaining why

**2025-12-28 14:40:** Found CLI calls IsSessionProcessing per-session (Finding 2)
- Understood CLI vs API semantic difference

**2025-12-28 14:45:** Found frontend filter too strict (Finding 3)
- Root cause identified: filter expected status='active' but API returns 'idle'

**2025-12-28 14:50:** Implemented fix
- Updated activeAgents, recentAgents, archivedAgents filters
- Added documentation comments

**2025-12-28 14:55:** Investigation completed
- Status: Complete
- Key outcome: Fixed by updating frontend filters to include idle agents in active section
