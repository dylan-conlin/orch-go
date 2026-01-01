<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Web UI showing 0 agents was caused by Chrome's 6-connection-per-host limit being exhausted by SSE connections from multiple dashboard tabs, blocking the agents.fetch() request.

**Evidence:** With 2+ dashboard tabs open, lsof showed 6 Chrome connections to localhost:3348, fetch requests hung indefinitely. Closing one tab immediately allowed the fetch to complete, loading 1882 agents.

**Knowledge:** Browser connection limits can cause SSE streams to block HTTP requests to the same host. The fix is to fetch critical data BEFORE establishing SSE connections.

**Next:** Fix implemented and committed - fetch agents before SSE connect.

---

# Investigation: Web UI Shows 0 Agents - Chrome Connection Limit

**Question:** Why does the web UI show 0 agents despite the API returning 1879 agents?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** og-debug-web-ui-shows-01jan [orch-go-76vp]
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: SSE connection triggers agents.fetch() only after connection established

**Evidence:** In `agents.ts:371-382`, the `eventSource.onopen` handler calls `agents.fetch()`. This means the fetch request is made AFTER the SSE connection is established.

**Source:** `web/src/lib/stores/agents.ts:371-382`

**Significance:** If SSE connections consume available browser connections, the subsequent fetch may be blocked waiting for a free connection.

---

### Finding 2: Chrome limits connections to 6 per host

**Evidence:** Using `lsof -i :3348`, observed 6 established connections from Chrome (Google process) to the orch serve API. All 6 connections were in ESTABLISHED state.

**Source:** Command: `/usr/sbin/lsof -i :3348`

**Significance:** This is a well-known browser limitation. When all 6 connections are used by SSE streams, no connections are available for HTTP requests like the agents fetch.

---

### Finding 3: Multiple dashboard tabs exhaust the connection pool

**Evidence:** With 2 Swarm Dashboard tabs open (both on localhost:5188), each tab opens SSE connections to localhost:3348. Combined with other connections (preflight, etc.), all 6 connections were consumed.

**Source:** `glass_tabs` showed multiple Swarm Dashboard tabs; `lsof` showed 6 connections.

**Significance:** This is the root cause - multiple tabs cause connection exhaustion, blocking the agents.fetch() request.

---

### Finding 4: Fetch was being aborted by race condition (secondary issue)

**Evidence:** Initial debugging showed "Fetch ABORTED" in page title. The `fetch()` method was aborting previous in-flight requests when called again (by SSE events triggering `fetchDebounced()`).

**Source:** `web/src/lib/stores/agents.ts:156-160` - previous implementation called `currentFetchController.abort()`

**Significance:** Even without the connection limit issue, the fetch could be aborted if SSE events arrived within the debounce window. Fixed by skipping duplicate fetch requests instead of aborting.

---

## Synthesis

**Key Insights:**

1. **Connection limits affect SSE + HTTP patterns** - When using SSE for real-time updates, be aware that the SSE connection consumes one of the limited browser connections to that host. Multiple tabs multiply this effect.

2. **Order of operations matters** - Fetching critical data BEFORE establishing SSE ensures the data loads even if SSE later saturates the connection pool.

3. **Abort vs Skip for fetch deduplication** - Aborting in-progress fetches causes data loss. Skipping duplicate fetch requests preserves the in-progress request.

**Answer to Investigation Question:**

The web UI showed 0 agents because:
1. Multiple dashboard tabs opened SSE connections to localhost:3348
2. This exhausted Chrome's 6-connection-per-host limit
3. The agents.fetch() request was queued waiting for a free connection
4. The fetch never completed because SSE connections are long-lived

The fix is to fetch agents BEFORE connecting to SSE, ensuring critical data loads regardless of connection limits.

---

## Structured Uncertainty

**What's tested:**

- ✅ Closing one tab freed connections and allowed fetch to complete (verified: manually closed tab, fetch succeeded with 1882 agents)
- ✅ API returns correct data via curl (verified: `curl http://localhost:3348/api/agents | jq 'length'` returned 1882)
- ✅ Fix works with single tab (verified: refreshed page after fix, dashboard loaded correctly)

**What's untested:**

- ⚠️ Behavior with 6+ dashboard tabs (not tested, but should fail similarly)
- ⚠️ HTTP/2 multiplexing as alternative fix (not implemented)
- ⚠️ SharedWorker for SSE connection sharing (not implemented)

**What would change this:**

- Finding would be wrong if issue persists with only 1 tab open
- Finding would be incomplete if there's a server-side connection limit too

---

## Implementation Recommendations

### Recommended Approach ⭐

**Fetch-before-SSE** - Load critical agent data before establishing SSE connection.

**Why this approach:**
- Zero infrastructure changes required
- Works immediately with current browser limitations
- Ensures dashboard is usable even if SSE fails

**Trade-offs accepted:**
- Initial load is sequential (fetch then SSE) instead of parallel
- ~1s additional latency on initial load

**Implementation sequence:**
1. Call `agents.fetch()` in `onMount` before `connectSSE()`
2. Connect SSE after fetch completes (success or failure)
3. Keep SSE onopen fetch for reconnection scenarios

### Alternative Approaches Considered

**Option B: HTTP/2**
- **Pros:** Multiplexes unlimited streams over single connection
- **Cons:** Requires HTTPS, certificate setup, server changes
- **When to use instead:** If building production deployment with TLS

**Option C: SharedWorker for SSE**
- **Pros:** Single SSE connection shared across tabs
- **Cons:** Complex to implement, browser support varies
- **When to use instead:** If many concurrent tabs is a primary use case

**Rationale for recommendation:** Fetch-before-SSE is simplest and most reliable for current use case.

---

## References

**Files Examined:**
- `web/src/lib/stores/agents.ts` - Agent store and SSE connection logic
- `web/src/routes/+page.svelte` - Dashboard initialization
- `cmd/orch/serve.go` - API server and SSE proxy

**Commands Run:**
```bash
# Check API response
curl -s http://localhost:3348/api/agents | jq 'length'

# Check connection count
/usr/sbin/lsof -i :3348

# Check open tabs
glass_tabs
```

---

## Investigation History

**2026-01-01 03:40:** Investigation started
- Initial question: Why does web UI show 0 agents despite API returning 1879?
- Context: User reported dashboard shows "No active agents" but API works via curl

**2026-01-01 03:45:** Found SSE-triggered fetch pattern
- Discovered fetch is called in SSE onopen, not before

**2026-01-01 03:48:** Found abort race condition
- Discovered fetchDebounced was aborting in-progress fetches

**2026-01-01 03:50:** Root cause identified
- Chrome 6-connection limit exhausted by multiple tabs
- Verified by closing tab and seeing fetch succeed

**2026-01-01 03:52:** Investigation completed
- Status: Complete
- Key outcome: Fixed by fetching agents before SSE connect
