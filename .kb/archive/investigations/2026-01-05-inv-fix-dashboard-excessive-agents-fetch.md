<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard request storm caused by debounce not accounting for in-flight requests - fixed by adding isFetching/needsRefetch state tracking.

**Evidence:** Code analysis shows `session.status` SSE events trigger `fetchDebounced()` (line 549-551), which correctly waits 500ms but starts new requests even while one is in-flight, causing aborted/pending requests.

**Knowledge:** Debouncing alone doesn't prevent concurrent requests; must also track in-flight state and defer new requests until current completes.

**Next:** Verify fix by opening dashboard and checking Network panel shows clean single requests instead of storm.

---

# Investigation: Fix Dashboard Excessive Agents Fetch

**Question:** Why does the dashboard create a request storm of agents fetch requests despite 500ms debounce?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** Agent (og-debug-fix-dashboard-excessive-05jan-c973)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: session.status events trigger agent refetch on every busy/idle transition

**Evidence:** In `agents.ts:541-551`, the `handleSSEEvent` function calls `agents.fetchDebounced()` for `session.status` events. These events fire every time an agent transitions between busy/idle states, which happens frequently during active work.

**Source:** `web/src/lib/stores/agents.ts:541-551`

**Significance:** High frequency of SSE events is the trigger for the request storm - each agent switching busy/idle causes a fetch attempt.

---

### Finding 2: Debounce doesn't prevent concurrent in-flight requests

**Evidence:** The debounce correctly collapses multiple calls within 500ms into one, but when the timer fires and `this.fetch()` is called:
1. It aborts any previous in-flight request (line 192-194)
2. Starts a new request
3. Meanwhile, more SSE events arrive and start new debounce timers
4. When those timers fire, they abort the current request and start new ones
5. This creates the cascade of aborted/pending requests

**Source:** `web/src/lib/stores/agents.ts:190-226`

**Significance:** The root cause - debounce prevents rapid immediate calls but doesn't prevent requests from overlapping.

---

### Finding 3: onOpen callback bypasses debounce

**Evidence:** In `agents.ts:412-415`, the `onOpen` callback calls `agents.fetch()` directly (not debounced). This means when SSE connection opens, it immediately fetches, and then SSE events immediately trigger debounced fetches - potentially causing an initial burst.

**Source:** `web/src/lib/stores/agents.ts:411-416`

**Significance:** Not the primary cause but contributes to request timing issues on connection establishment.

---

## Synthesis

**Key Insights:**

1. **In-flight tracking is required** - Debouncing handles call frequency but not concurrent execution. The fix must prevent new fetches from starting while one is already in-flight.

2. **Deferred refetch pattern** - When requests arrive during an in-flight fetch, they should set a flag to trigger another fetch after completion, not start immediately.

3. **Single fetch at a time** - The solution ensures at most one fetch request is in-flight, with SSE events during that time collapsed into a single follow-up fetch.

**Answer to Investigation Question:**

The request storm occurs because the debounce timer prevents rapid immediate calls but doesn't prevent multiple requests from being in-flight simultaneously. Each time a debounced timer fires while a request is already running, it aborts that request and starts a new one. The fix adds `isFetching` and `needsRefetch` state variables to track in-flight requests and defer new requests until the current one completes.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build succeeds with changes (verified: `bun run build` completed successfully)
- ✅ Code analysis confirms root cause mechanism (verified: traced full SSE event -> fetch flow)

**What's untested:**

- ⚠️ Network panel shows clean requests instead of storm (needs browser verification)
- ⚠️ No regression in agent data freshness (needs functional test)

**What would change this:**

- Finding would be wrong if browser shows same request storm after fix
- Finding would be wrong if there are multiple SSE connections being created

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach (IMPLEMENTED)

**In-flight tracking with deferred refetch** - Added `isFetching` and `needsRefetch` state variables to prevent concurrent fetches.

**Why this approach:**
- Prevents request storm by ensuring only one fetch at a time
- Still captures all SSE events (via needsRefetch flag)
- Maintains data freshness (follow-up fetch after current completes)

**Trade-offs accepted:**
- Slightly delayed updates when events arrive during fetch
- Acceptable because data arrives after short delay, not lost

**Implementation sequence:**
1. Added `isFetching` and `needsRefetch` state variables
2. Modified `fetch()` to check/set isFetching and handle needsRefetch
3. Updated `cancelPending()` to reset new state variables

---

## References

**Files Examined:**
- `web/src/lib/stores/agents.ts` - Main fetch and debounce logic
- `web/src/lib/services/sse-connection.ts` - SSE connection lifecycle
- `web/src/routes/+page.svelte` - Dashboard mount and connection

**Commands Run:**
```bash
# Build verification
/opt/homebrew/bin/bun run build  # Success

# Type check (has pre-existing errors in theme.ts)
/opt/homebrew/bin/bun run check
```

---

## Investigation History

**2026-01-05 21:00:** Investigation started
- Initial question: Why does dashboard create request storm despite debounce?
- Context: Network panel shows rapid fetch requests with many canceled/pending

**2026-01-05 21:03:** Root cause identified
- session.status SSE events trigger fetchDebounced() on every busy/idle transition
- Debounce doesn't prevent concurrent in-flight requests
- AbortController aborts previous requests but doesn't prevent new ones

**2026-01-05 21:03:** Fix implemented
- Added isFetching and needsRefetch state tracking
- Modified fetch() to defer new requests when one is in-flight
- Build verification passed

**2026-01-05 21:10:** Investigation completed
- Status: Complete
- Commit: 25c6edf3
- Outcome: Fixed request storm by adding in-flight tracking with isFetching/needsRefetch state
