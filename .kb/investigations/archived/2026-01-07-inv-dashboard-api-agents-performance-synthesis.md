## Summary (D.E.K.N.)

**Delta:** Fixed /api/agents from 32s to 627ms (50x improvement) by caching investigation directory listings and restoring 2-hour beadsFetchThreshold.

**Evidence:** Profiled endpoint with timing logs. Two root causes: O(n²) investigation discovery (362 agents × 590 files × 2-3 ReadDir calls = 427K+ comparisons) and 24h threshold regression causing 86 beads IDs instead of 6.

**Knowledge:** Dashboard performance issues follow predictable patterns - O(n) session/file scaling, threshold regressions, and cache misses. Always profile before fixing.

**Next:** Close - fix committed (f87bedf4). Created issues for remaining slow endpoints (/api/beads, /api/beads/ready).

---

# Investigation: Dashboard /api/agents Performance Synthesis

**Question:** Why does /api/agents take 32 seconds and how do we prevent recurrence?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Orchestrator + Dylan (interactive investigation)
**Phase:** Complete
**Status:** Complete

---

## Context: The Pattern of Recurring Dashboard Slowness

This is the **fourth** occurrence of dashboard API slowness since Dec 21:

| Date | Sessions | Root Cause | Fix |
|------|----------|------------|-----|
| Dec 22 | 209 | Svelte 5 runes broke reactivity | Remove runes |
| Dec 27 | 564 | O(N) sequential RPC calls | Parallelization |
| Jan 6 | 623 | Session accumulation, cache TTLs | 2h threshold (changed to 24h!) |
| Jan 7 | 226 | O(n²) investigation discovery + threshold regression | Cache + restore 2h |

**Key insight:** Each fix addressed symptoms but not the underlying architectural tension - the API does too much work per request.

---

## Findings

### Finding 1: O(n²) Investigation Directory Scanning

**Evidence:** `discoverInvestigationPath()` called `os.ReadDir()` 2-3 times per agent:
- Line 91: ReadDir for workspace keyword matching
- Line 108: ReadDir again for beads ID matching  
- Line 128: ReadDir for simple/ subdirectory

With 362 non-stale agents and 590 investigation files:
- 362 × 590 × 2 = 427,160 file entry comparisons
- Each ReadDir call ~1-2ms = 700+ ms just for directory listing

**Source:** `cmd/orch/serve_agents.go:80-164` (before fix)

**Significance:** This O(n²) pattern was invisible because it's spread across function calls. Timing logs revealed it.

---

### Finding 2: beadsFetchThreshold Regression (24h → should be 2h)

**Evidence:** The Jan 6 fix set `beadsFetchThreshold := 24 * time.Hour` instead of the original 2 hours. This caused:
- 86 beads IDs fetched instead of 6
- getAllIssues: 4.3 seconds (86 parallel RPC calls)
- getComments: 4.4 seconds (86 parallel RPC calls)

**Source:** `cmd/orch/serve_agents.go:279` showed `24 * time.Hour`

**Significance:** The "fix" made things worse. The 2-hour threshold limits to recently active sessions, which is the actual operational need.

---

### Finding 3: Caching Works When Hit

**Evidence:** After implementing fixes:
- Cold cache: 627ms
- Warm cache: 132ms

The beads cache has 15-60 second TTLs. When cache is warm, performance is excellent.

**Source:** `cmd/orch/serve_agents_cache.go:82-86`

**Significance:** The problem isn't the cache design, it's the cold cache path doing too much work.

---

### Finding 4: 406 of 426 Agents Have No project_dir

**Evidence:** Only 19 agents had `project_dir` set (from OpenCode sessions with beads IDs). The other 406 came from workspace scanning and had `project_dir: null`.

**Source:** `jq '[.[] | select(.project_dir == null)] | length'` on API response

**Significance:** Investigation discovery only runs for agents WITH project_dir. The cache optimization mainly helps the 19 active agents, but the directory scanning was still happening 362 times.

---

## Why This Keeps Happening

### Architectural Tension

The `/api/agents` endpoint tries to provide a complete picture by:
1. Fetching all OpenCode sessions (226)
2. Scanning all workspace directories (391)
3. Enriching with beads data (issues + comments)
4. Discovering investigation files
5. Computing status via Priority Cascade

This design worked at small scale but scales poorly:
- O(sessions) for beads fetch
- O(workspaces) for directory scanning
- O(agents × investigation_files) for discovery

### Why Fixes Don't Stick

Each fix addresses the immediate bottleneck but doesn't change the architecture:
- Dec 27: Parallelized RPC calls → helped until more sessions accumulated
- Jan 6: Added threshold → but set it to 24h, too generous
- Jan 7: Cached directory listings → helped but didn't reduce total agents

**The real fix would be:**
1. Server-side pagination (don't return 426 agents at once)
2. Lazy loading (fetch beads data on demand, not upfront)
3. Background refresh (pre-compute expensive data)

---

## Fixes Applied (Jan 7)

### Fix 1: Investigation Directory Cache

```go
type investigationDirCache struct {
    entries map[string][]string  // dir path -> list of .md filenames
}

func buildInvestigationDirCache(projectDirs []string) *investigationDirCache
```

Built once before agent loop, reused for all agents. Changed from O(n×m) to O(n+m).

### Fix 2: Restore 2-Hour Threshold

```go
beadsFetchThreshold := 2 * time.Hour  // was 24 * time.Hour
```

Reduced beads IDs from 86 to 6, cutting RPC time from 8+ seconds to ~400ms.

### Commit

`f87bedf4` - "perf: fix /api/agents O(n²) investigation discovery + restore 2h threshold"

---

## Performance Results

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Cold cache | 32,000ms | 627ms | 51x |
| Warm cache | N/A | 132ms | - |
| Beads IDs fetched | 86 | 6 | 14x fewer |
| Directory scans | 362 × 2 | 1 | 724x fewer |

---

## Remaining Issues

### /api/beads - 6.5 seconds
Likely similar pattern - fetching too much data. Needs profiling.

### /api/beads/ready - 5.2 seconds  
Should be fast (just listing ready issues). Needs investigation.

### Architectural Debt
The endpoint design doesn't scale. Long-term needs:
- Pagination
- Lazy loading
- Background pre-computation

---

## Lessons Learned

1. **Profile before fixing** - Timing logs revealed the actual bottlenecks. Without them, we'd have guessed wrong.

2. **Check previous fixes** - The 24h threshold was a regression from an earlier fix. Reading the investigation history would have caught this.

3. **O(n²) hides in function calls** - The investigation discovery looked innocent but scaled terribly.

4. **Thresholds need justification** - "2 hours" was chosen to match operational reality (active work). "24 hours" was arbitrary.

5. **Caching is necessary but not sufficient** - Cold cache performance matters for first load and after invalidation.

---

## References

**Related Investigations:**
- `.kb/investigations/2025-12-27-inv-api-agents-endpoint-takes-19s.md` - Parallelization fix
- `.kb/investigations/2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md` - HTTP/1.1 limits
- `.kb/investigations/2026-01-06-inv-dashboard-api-slow-again-623.md` - Session accumulation (introduced 24h regression)
- `.kb/guides/dashboard.md` - Consolidated dashboard knowledge

**Files Modified:**
- `cmd/orch/serve_agents.go` - Added investigationDirCache, restored 2h threshold
- `cmd/orch/serve_agents_test.go` - Updated test to use cache

**Commits:**
- `f87bedf4` - Performance fix
