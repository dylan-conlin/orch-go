<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The TTL cache in orch serve never invalidated when agents completed, causing the dashboard to show stale "active" status until TTL expired.

**Evidence:** Traced cache implementation in serve_agents.go - beadsCache and globalWorkspaceCacheType only used TTL-based expiration with no invalidation mechanism. orch complete ran as separate CLI process with no way to notify the server.

**Knowledge:** In-process caches require explicit invalidation when external state changes. TTL alone is insufficient for event-driven updates.

**Next:** Fix implemented and tested. Ready for merge.

---

# Investigation: Orch Serve Cache Not Invalidated

**Question:** Why does the dashboard show stale "active" status for agents after orch complete runs?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Cache uses TTL only, no invalidation

**Evidence:** The `beadsCache` struct in serve_agents.go:27-46 uses time-based TTL (10-30 seconds) for cache expiration. There were no methods to clear the cache on demand.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_agents.go:27-46`

**Significance:** When an agent completes, the cache holds stale data until the TTL expires. Dashboard polls show "active" status for 10-30 seconds after completion.

---

### Finding 2: orch complete runs as separate process

**Evidence:** The `runComplete` function in main.go is a CLI command that runs as a separate process from `orch serve`. The server's in-memory cache is unreachable from the CLI.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:196-1203`

**Significance:** The only way to invalidate the cache is via an HTTP API call from the CLI to the server.

---

### Finding 3: Similar pattern exists for workspace cache

**Evidence:** The `globalWorkspaceCacheType` in serve_agents.go:48-64 also uses TTL-based caching with no invalidation. This cache holds workspace metadata scanned from .orch/workspace/ directories.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_agents.go:48-64`

**Significance:** Both caches need invalidation on agent completion.

---

## Synthesis

**Key Insights:**

1. **Cache invalidation is event-driven, not TTL-driven** - When an agent completes, the dashboard should reflect this immediately. TTL is appropriate for reducing API load, but events (like completion) need explicit invalidation.

2. **CLI-to-server communication requires an API** - Since orch complete runs as a separate process, it needs to notify the server via HTTP to invalidate caches.

3. **Silent failure is acceptable** - If orch serve isn't running, cache invalidation fails silently. The TTL will eventually refresh the data, so this is a graceful degradation.

**Answer to Investigation Question:**

The dashboard showed stale "active" status because the TTL cache held cached data for 10-30 seconds after agent completion. The fix adds explicit cache invalidation via a POST /api/cache/invalidate endpoint that orch complete calls after closing the beads issue.

---

## Structured Uncertainty

**What's tested:**

- ✅ Cache invalidation clears all cached data (verified: TestBeadsCacheInvalidate passes)
- ✅ Workspace cache invalidation clears data (verified: TestGlobalWorkspaceCacheInvalidate passes)
- ✅ API endpoint returns 200 and invalidates caches (verified: TestHandleCacheInvalidate passes)
- ✅ All existing tests still pass (verified: go test ./... passes)

**What's untested:**

- ⚠️ End-to-end flow with real dashboard (requires manual testing with orch serve + dashboard)
- ⚠️ Behavior when orch serve is not running (silent failure expected but not integration tested)

**What would change this:**

- Finding would be wrong if cache invalidation doesn't solve the staleness issue (would require investigating other caching layers)
- If dashboard has its own caching, this fix wouldn't be sufficient

---

## Implementation Recommendations

**Purpose:** Document the fix that was implemented.

### Recommended Approach ⭐

**Add cache invalidation with HTTP API** - Three-part fix:
1. Add `invalidate()` methods to both cache types
2. Add `POST /api/cache/invalidate` endpoint
3. Call endpoint from `orch complete` after closing the beads issue

**Why this approach:**
- Minimal code change (adds ~60 lines)
- No architectural changes needed
- Silent failure is graceful (TTL still works as fallback)

**Trade-offs accepted:**
- Extra HTTP call on every orch complete (2s timeout, non-blocking)
- No invalidation if orch serve isn't running (acceptable - TTL handles it)

**Implementation sequence:**
1. Add `invalidate()` methods to cache types
2. Add API endpoint handler
3. Add CLI call from orch complete
4. Add tests for new functionality

### Alternative Approaches Considered

**Option B: SSE push from CLI to dashboard**
- **Pros:** Real-time, no polling
- **Cons:** Complex, requires SSE channel management, more code
- **When to use instead:** If latency requirements are stricter

**Option C: Reduce TTL significantly**
- **Pros:** Simpler, no new code
- **Cons:** Increases bd process spawning (the original problem TTL caching solved)
- **When to use instead:** Not recommended - defeats the purpose of caching

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_agents.go` - Cache implementation
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go` - Server setup and routes
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go` - orch complete implementation

**Commands Run:**
```bash
# Verify build
go build ./cmd/orch/...

# Run all tests
go test ./... -count=1

# Run cache-specific tests
go test ./cmd/orch/... -run "Cache" -v -count=1
```

---

## Investigation History

**2026-01-04:** Investigation started
- Initial question: Why does dashboard show stale status after orch complete?
- Context: TTL cache added in orch-go-ez2f doesn't invalidate on completion

**2026-01-04:** Root cause identified
- Traced cache implementation in serve_agents.go
- Found no invalidation mechanism

**2026-01-04:** Fix implemented
- Added invalidate() methods to both cache types
- Added POST /api/cache/invalidate endpoint
- Added invalidateServeCache() call from orch complete
- Added 4 new tests

**2026-01-04:** Investigation completed
- Status: Complete
- Key outcome: Fix implemented and tested, ready for merge
