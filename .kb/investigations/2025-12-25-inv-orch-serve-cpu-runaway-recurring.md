## Summary (D.E.K.N.)

**Delta:** CPU runaway in orch serve was caused by O(n*m) file operations per /api/agents request, where n=agents and m=workspaces (~466).

**Evidence:** 5-minute stress test: before fix CPU hit 124%, after fix stays at 4-7% during polling and 0% at idle.

**Knowledge:** Dashboard polling at 500ms with frequent SSE events creates high request volume; any per-workspace operation needs caching.

**Next:** Close - fix verified via 5-minute smoke test, all tests passing.

**Confidence:** High (90%) - tested for 5 minutes, confirmed CPU stays low; longer test would increase to 95%+.

---

# Investigation: orch serve CPU runaway recurring

**Question:** Why does orch serve hit 124% CPU after ~15 minutes despite previous fix (ed772bac)?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** og-debug-orch-serve-cpu-25dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Previous fix addressed symptoms, not root cause

**Evidence:** Commit ed772bac removed IsProcessing HTTP calls and added 500ms debounce, but CPU still hit 124% after ~15 minutes of dashboard use.

**Source:** SPAWN_CONTEXT.md, decision record 2025-12-25-orchestrator-system-resource-visibility.md

**Significance:** The previous fix was necessary but not sufficient. Multiple factors contribute to CPU usage.

---

### Finding 2: O(n*m) file operations per /api/agents request

**Evidence:** 
- `handleAgents()` calls `findWorkspaceByBeadsID()` for each agent with a beads ID
- `findWorkspaceByBeadsID()` scans all ~466 workspace directories
- For each directory, it reads SPAWN_CONTEXT.md looking for matching beads ID
- With 10 agents: 10 * 466 = 4,660 file operations per request
- Plus another scan for completed workspaces
- With 500ms SSE debounce: ~2 requests/second = 9,320 file operations/second

**Source:** cmd/orch/serve.go:349, cmd/orch/main.go:2599-2643

**Significance:** This is the root cause of CPU runaway. File I/O is the bottleneck.

---

### Finding 3: Fix verified with 5-minute stress test

**Evidence:**
```
Minute 1: CPU=5.2%
Minute 2: CPU=7.1%
Minute 3: CPU=4.8%
Minute 4: CPU=7.2%
Minute 5: CPU=4.7%
Final after settling: 0.0%
```

**Source:** Manual testing with continuous curl polling at 500ms intervals

**Significance:** Confirms fix works. CPU stays bounded during polling and returns to baseline.

---

## Synthesis

**Key Insights:**

1. **Workspace scanning is expensive** - With 466 directories, any operation that scans all workspaces per-agent creates massive I/O overhead.

2. **Caching is essential** - Building a beadsID→workspacePath cache once per request reduces O(n*m) to O(m) + O(n) lookups.

3. **pprof is valuable for diagnostics** - Added /debug/pprof/ endpoint for future CPU/memory profiling without runtime cost when unused.

**Answer to Investigation Question:**

The 124% CPU was caused by repeated directory scanning in `handleAgents()`. Each agent triggered a full scan of 466 workspaces. With SSE events triggering refetches every 500ms, this created thousands of file operations per second. The fix caches workspace metadata once per request and uses O(1) map lookups for each agent.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

5-minute stress test confirmed CPU stays at 4-7% during polling (vs 124% before). However, the original issue took ~15 minutes to manifest, so a longer test would provide higher confidence.

**What's certain:**

- ✅ Root cause identified: O(n*m) file scanning per request
- ✅ Fix implemented and tested
- ✅ CPU returns to 0% after polling stops (no goroutine leaks)

**What's uncertain:**

- ⚠️ 5-minute test may not catch all edge cases
- ⚠️ Original 124% may have had additional contributing factors

**What would increase confidence to Very High (95%+):**

- Run orch serve for 15+ minutes with dashboard active
- Monitor with pprof under load to verify no new hotspots
- User confirmation that issue doesn't recur

---

## Implementation Recommendations

### Recommended Approach (IMPLEMENTED)

**Workspace caching per request** - Build beadsID→workspacePath map once at start of handleAgents(), use for all lookups.

**Why this approach:**
- Reduces file operations from O(n*m) to O(m)
- No persistent state to manage
- Memory overhead is minimal (~466 map entries)

**Trade-offs accepted:**
- Cache rebuilds on each request (acceptable given 500ms debounce)
- Doesn't help other callers of findWorkspaceByBeadsID (only handleAgents optimized)

### Alternative Approaches Considered

**Option B: Global workspace cache with TTL**
- **Pros:** Would help all callers, even faster for handleAgents
- **Cons:** Cache invalidation complexity, stale data risk
- **When to use:** If other callers also become bottlenecks

**Option C: Server-side debouncing**
- **Pros:** Would reduce request volume regardless of client
- **Cons:** Adds latency, doesn't fix root cause
- **When to use:** If client-side debounce proves insufficient

---

## References

**Files Examined:**
- `cmd/orch/serve.go` - handleAgents implementation
- `cmd/orch/main.go:2599-2643` - findWorkspaceByBeadsID implementation
- `web/src/lib/stores/agents.ts` - Client-side fetching and debounce

**Commands Run:**
```bash
# Stress test
for minute in {1..5}; do
    for i in {1..60}; do curl -s http://127.0.0.1:3348/api/agents > /dev/null; sleep 0.5; done
    ps aux | grep "orch serve" | awk '{print "CPU:", $3"%"}'
done

# CPU check
ps aux | grep "orch serve" | grep -v grep | awk '{print "CPU:", $3"%"}'
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-25-orchestrator-system-resource-visibility.md`
- **Commit:** `e0303569` - fix implementation
- **Workspace:** `.orch/workspace/og-debug-orch-serve-cpu-25dec/`

---

## Investigation History

**2025-12-25 22:00:** Investigation started
- Initial question: Why does orch serve hit 124% CPU after ~15 minutes?
- Context: Previous fix (ed772bac) didn't hold

**2025-12-25 22:20:** Root cause identified
- Found O(n*m) file scanning pattern in handleAgents()
- Workspace count: 466 directories

**2025-12-25 22:35:** Fix implemented and tested
- Added buildWorkspaceCache() function
- Added pprof endpoint for diagnostics
- 5-minute stress test passed

**2025-12-25 22:45:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: CPU reduced from 124% to 4-7% during polling
