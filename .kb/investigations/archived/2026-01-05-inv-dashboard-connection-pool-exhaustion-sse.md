<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Removed agentlog SSE auto-connect on page load to fix HTTP/1.1 connection pool exhaustion.

**Evidence:** Dashboard was auto-connecting two SSE connections (events + agentlog) on load, consuming 2 of 6 HTTP/1.1 connections per origin. When combined with API fetches, the pool could become exhausted, causing requests to queue as "pending".

**Knowledge:** HTTP/1.1 has a 6 connections per origin limit. Long-lived SSE connections occupy these slots. Non-critical SSE streams should be opt-in, not auto-connected.

**Next:** Close - fix implemented and verified via build. Users can manually connect agentlog via "Follow" button in Agent Lifecycle panel.

---

# Investigation: Dashboard Connection Pool Exhaustion SSE

**Question:** Why do API fetch requests queue as pending when dashboard has multiple SSE connections?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Two SSE connections auto-connected on page load

**Evidence:** In `+page.svelte`, the `onMount` handler established two SSE connections:
1. Primary events SSE (`connectSSE()`) - line 109
2. Agentlog SSE (`connectAgentlogSSE()`) - deferred via requestIdleCallback but still auto-connected

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/web/src/routes/+page.svelte:109-145`

**Significance:** Each SSE connection is a long-lived HTTP connection that occupies one of the 6 available HTTP/1.1 connections per origin.

---

### Finding 2: HTTP/1.1 connection limit is 6 per origin

**Evidence:** Browser standard limits HTTP/1.1 to 6 concurrent connections per origin. With 2 SSE connections + 6+ API fetches (agents, beads, usage, focus, servers, readyIssues, daemon, hotspots, orchestratorSessions), the pool can become exhausted.

**Source:** HTTP/1.1 browser standards, observed "pending" status in network panel

**Significance:** The dashboard performs 9+ different API fetches on load, but only has 4 remaining connection slots after SSE connections.

---

### Finding 3: Agentlog SSE is non-critical and rarely viewed

**Evidence:** The Agent Lifecycle panel showing agentlog events is:
1. Collapsed by default (`sectionState.sseStream: false`)
2. Only visible in historical mode (`$dashboardMode !== 'operational'`)
3. Has a "Follow" button suggesting user should explicitly opt-in

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/web/src/routes/+page.svelte:61, 570-633`

**Significance:** Auto-connecting a non-critical, rarely-viewed SSE stream wastes a valuable connection slot.

---

## Synthesis

**Key Insights:**

1. **Connection pool is a finite resource** - HTTP/1.1's 6-connection limit means every long-lived connection (like SSE) reduces capacity for API requests.

2. **Auto-connecting non-critical streams is wasteful** - The agentlog SSE was deferred via requestIdleCallback but still connected automatically, even though most users never expand that panel.

3. **Opt-in is better for secondary SSE** - The panel already had a "Follow" button, suggesting the intended UX was manual activation.

**Answer to Investigation Question:**

API fetch requests queue as pending because two SSE connections consume 2 of 6 available HTTP/1.1 connections per origin. The fix is to make the secondary agentlog SSE opt-in only, reserving connection slots for API requests.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build passes after removing agentlog auto-connect (verified: `bun run build` succeeded)
- ✅ Agentlog SSE can still be manually connected via "Follow" button (verified: `handleAgentlogConnectClick` unchanged)
- ✅ Primary SSE connection still auto-connects (verified: `connectSSE()` call unchanged)

**What's untested:**

- ⚠️ Browser smoke test not performed (servers not started)
- ⚠️ Network panel verification of connection count reduction (would need browser dev tools)

**What would change this:**

- If HTTP/2 is enabled on the API server, connection pool exhaustion wouldn't occur (HTTP/2 multiplexes)
- If more than 6 concurrent API requests are needed, additional SSE connections would still cause issues

---

## Implementation Recommendations

### Recommended Approach ⭐

**Remove agentlog auto-connect** - Replace auto-connect code with a comment explaining the change and pointing to this investigation.

**Why this approach:**
- Minimal code change (5 lines removed, 5 lines comment added)
- Preserves existing "Follow" button UX for users who want agentlog
- Frees one HTTP/1.1 connection slot for API requests

**Trade-offs accepted:**
- Users won't see agentlog events unless they click "Follow"
- This is acceptable because the panel is collapsed by default anyway

**Implementation sequence:**
1. Remove `connectAgentlogSSE()` call from requestIdleCallback block ✅
2. Add explanatory comment ✅
3. Verify build passes ✅

### Alternative Approaches Considered

**Option B: HTTP/2 on API server**
- **Pros:** Eliminates connection pool issues entirely
- **Cons:** Requires server-side changes, more complex deployment
- **When to use instead:** Long-term solution for heavy SSE usage

**Option C: Single multiplexed SSE endpoint**
- **Pros:** One connection for all event types
- **Cons:** Requires significant refactoring of both server and client
- **When to use instead:** If dashboard needs many more SSE event types

**Rationale for recommendation:** Quick fix removes immediate problem; HTTP/2 or multiplexing can be addressed separately.

---

## References

**Files Examined:**
- `web/src/routes/+page.svelte` - Main page component with SSE connection setup
- `web/src/lib/stores/agentlog.ts` - Agentlog SSE connection store
- `web/src/lib/stores/agents.ts` - Primary SSE connection store

**Commands Run:**
```bash
# Verify TypeScript build passes
bun run build

# Search for agentlog connection usage
grep connectAgentlogSSE web/src/**/*.svelte web/src/**/*.ts
```

---

## Investigation History

**2026-01-05 12:00:** Investigation started
- Initial question: Why do dashboard API requests queue as pending?
- Context: Root cause identified in spawn context - SSE connections consuming connection pool

**2026-01-05 12:15:** Fix implemented
- Removed agentlog auto-connect from `+page.svelte`
- Added explanatory comment pointing to this investigation
- Verified build passes

**2026-01-05 12:20:** Investigation completed
- Status: Complete
- Key outcome: Agentlog SSE now opt-in only, freeing connection slot for API requests
