<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** /api/beads (6.5s) and /api/beads/ready (5.2s) were slow because they spawned bd CLI on every request without caching.

**Evidence:** Direct CLI calls take ~1.5s (stats) and ~80ms (ready), but API was 4-5x slower due to lack of caching. After fix, cached requests return in ~15ms.

**Knowledge:** TTL-based caching (30s for stats, 15s for ready) provides >400x improvement for cached requests with acceptable staleness tradeoff.

**Next:** Fix is complete - merged caching implementation.

**Promote to Decision:** recommend-no (tactical fix following established pattern from /api/agents)

---

# Investigation: Api Beads Endpoint Takes 5s

**Question:** Why are /api/beads and /api/beads/ready endpoints slow (6.5s and 5.2s)?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Agent (og-feat-api-beads-endpoint-07jan-3012)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: /api/beads and /api/beads/ready had no caching

**Evidence:** Handlers in serve_beads.go called `beads.FallbackStats()` and `beads.FallbackReady()` directly on every request, spawning `bd stats --json` and `bd ready --json` processes.

**Source:** `cmd/orch/serve_beads.go:24-66` (handleBeads), `cmd/orch/serve_beads.go:86-140` (handleBeadsReady)

**Significance:** Unlike `/api/agents` which used `globalBeadsCache` with TTL caching, these endpoints had no caching at all.

---

### Finding 2: Direct CLI timing vs API timing mismatch

**Evidence:** 
- `bd stats --json` directly: ~1.5s
- `/api/beads` API: 6.5s (4x slower)
- `bd ready --json --limit 0` directly: ~80ms
- `/api/beads/ready` API: 5.2s (65x slower)

**Source:** Command line profiling with `time` command

**Significance:** The large discrepancy on first call suggests the beadsClient RPC auto-reconnect (3 retries with exponential backoff) was adding ~5s overhead when daemon socket doesn't exist.

---

### Finding 3: Socket existence check optimization (concurrent fix)

**Evidence:** During this investigation, another agent added socket existence checks before attempting RPC to avoid slow timeout on dead daemon:
```go
socketPath, findErr := beads.FindSocketPath(beads.DefaultDir)
socketExists := findErr == nil && socketPath != ""
if socketExists {
    if _, statErr := os.Stat(socketPath); statErr != nil {
        socketExists = false
    }
}
```

**Source:** `cmd/orch/serve_beads.go:59-67`

**Significance:** This optimization short-circuits failed RPC attempts when no daemon is running.

---

## Synthesis

**Key Insights:**

1. **Caching is essential for dashboard polling** - The dashboard likely polls these endpoints every 5-10 seconds. Without caching, each poll spawns a bd process, creating unnecessary load.

2. **TTL must balance freshness vs performance** - Stats change less often (30s TTL reasonable) while ready queue might need fresher data (15s TTL). Cache invalidation endpoint exists for immediate refresh after changes.

3. **RPC client reconnect logic creates cold-start penalty** - When beads daemon isn't running, the auto-reconnect with backoff adds significant latency. Socket existence check mitigates this.

**Answer to Investigation Question:**

The endpoints were slow because:
1. No caching - every request spawned a bd process
2. RPC client reconnection overhead when daemon not running (~5s)
3. Fix adds TTL-based caching (30s stats, 15s ready) reducing cached response to ~15ms

---

## Structured Uncertainty

**What's tested:**

- ✅ Caching works - second request returns in ~15ms (verified via curl timing)
- ✅ Cache invalidation works - POST /api/cache/invalidate clears both stats and ready caches
- ✅ Tests pass - all existing tests continue to work

**What's untested:**

- ⚠️ Dashboard user experience (not verified in browser)
- ⚠️ Behavior when beads daemon IS running (couldn't test - no daemon in orch-go)

**What would change this:**

- Finding would be incomplete if there's additional latency under concurrent load
- Different caching strategy needed if 15-30s staleness is unacceptable for certain use cases

---

## Implementation Recommendations

**Purpose:** Fix is already implemented.

### Implemented Approach ⭐

**TTL-based beadsStatsCache** - New cache struct added to serve_beads.go with separate TTLs for stats (30s) and ready issues (15s).

**Changes made:**
1. Added `beadsStatsCache` struct with `getStats()` and `getReadyIssues()` methods
2. Added `globalBeadsStatsCache` variable initialized in `runServe()`
3. Updated handlers to use cache instead of direct calls
4. Added cache invalidation in `handleCacheInvalidate()`

**Files modified:**
- `cmd/orch/serve_beads.go` - Added cache implementation and updated handlers
- `cmd/orch/serve.go` - Added cache initialization
- `cmd/orch/serve_agents.go` - Added beadsStatsCache invalidation

---

## References

**Files Examined:**
- `cmd/orch/serve_beads.go` - Main endpoint handlers
- `cmd/orch/serve_agents_cache.go` - Reference pattern for caching
- `cmd/orch/serve.go` - Server initialization
- `pkg/beads/client.go` - RPC client with auto-reconnect

**Commands Run:**
```bash
# Profile endpoint timing
time curl -sk https://localhost:3348/api/beads

# Profile direct CLI timing
time bd stats --json

# Test cache behavior
curl -sk -X POST https://localhost:3348/api/cache/invalidate
time curl -sk https://localhost:3348/api/beads  # First call (miss)
time curl -sk https://localhost:3348/api/beads  # Second call (hit)
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-07-inv-dashboard-api-agents-performance-synthesis.md` - Prior /api/agents performance fix

---

## Investigation History

**2026-01-07 10:36:** Investigation started
- Initial question: Why do /api/beads (6.5s) and /api/beads/ready (5.2s) take so long?
- Context: Found during dashboard performance investigation

**2026-01-07 10:45:** Root cause identified
- Found missing caching compared to /api/agents
- Identified RPC reconnect overhead contributing to latency

**2026-01-07 10:51:** Implementation complete
- Added beadsStatsCache with TTL-based caching
- Verified ~450x improvement for cached requests (6.5s → 15ms)
