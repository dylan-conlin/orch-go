<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch status` optimized from 12.2s to ~1s by batching beads CLI calls and parallelizing comment fetches.

**Evidence:** Before: 12.2s (85% CPU). After: 1.05s (430% CPU). All tests pass, agent count identical (37).

**Knowledge:** Sequential subprocess calls (bd show, bd comments) × N agents is O(N × latency). Parallel goroutines + batch APIs reduce to O(max_latency).

**Next:** Close - implementation complete, merged, tested.

**Confidence:** High (90%) - Smoke tested with real data, all unit tests pass, edge cases may exist for very large agent counts.

---

# Investigation: Orch Status Takes 11 Seconds

**Question:** Why does `orch status` take 11 seconds when the OpenCode API returns in 15ms?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: API response is fast, subprocess calls are slow

**Evidence:** 
- `curl http://127.0.0.1:4096/session` returns 54 sessions in 18ms
- `bd show <id> --json` takes ~140ms per call
- `bd comments <id> --json` takes ~95ms per call
- Total: 3 bd calls per agent × 37 agents × ~100ms = ~11 seconds

**Source:** 
- `cmd/orch/main.go:1749-1917` - runStatus function
- `pkg/state/reconcile.go:71-101` - GetLiveness calls verify.GetIssue
- `pkg/verify/check.go:428-446` - GetIssue shells out to bd

**Significance:** The bottleneck is not the API but the O(N) subprocess invocations.

---

### Finding 2: Redundant beads calls in multiple code paths

**Evidence:**
- `state.GetLiveness()` calls `verify.GetIssue()` for each agent
- `getPhaseAndTask()` calls `verify.GetIssue()` AND `verify.GetPhaseStatus()` for each agent
- Net result: 3 bd subprocess calls per agent (show twice, comments once)

**Source:**
- `cmd/orch/main.go:1782` - GetLiveness in tmux loop
- `cmd/orch/main.go:1789` - getPhaseAndTask in tmux loop  
- `cmd/orch/main.go:1874` - GetLiveness in session loop
- `cmd/orch/main.go:1880` - getPhaseAndTask in session loop

**Significance:** Even if bd calls were faster, we're making 3× more calls than necessary.

---

### Finding 3: bd CLI supports batch operations

**Evidence:**
- `bd show id1 id2 id3 --json` returns all issues in one call (~100ms for 3)
- `bd list --status open --json` returns all open issues in one call
- `bd comments` does NOT support batch, but can be parallelized

**Source:**
- `bd --help` and manual testing
- `bd list --status open --json | jq 'length'` returns all open issues

**Significance:** We can batch issue fetches and parallelize comment fetches.

---

## Synthesis

**Key Insights:**

1. **Subprocess overhead dominates** - Each `exec.Command("bd", ...)` takes ~100ms regardless of the underlying operation. With 37 agents and 3 calls each, this compounds to 11 seconds.

2. **Data is re-fetched unnecessarily** - The code calls `verify.GetIssue()` from multiple locations without caching. A single upfront batch fetch is far more efficient.

3. **Parallelization is safe for reads** - Comment fetching can be done concurrently since it's read-only. Go's goroutines make this trivial.

**Answer to Investigation Question:**

The 11 second delay comes from sequential subprocess calls to `bd` (beads CLI), not from the OpenCode API. The fix requires:
1. Batch fetching all open issues upfront with `bd list --status open --json`
2. Parallelizing comment fetches using goroutines
3. Removing redundant `state.GetLiveness()` calls that were also hitting beads

---

## Implementation (Completed)

**Changes made:**

1. **Added batch functions to `pkg/verify/check.go`:**
   - `GetIssuesBatch(beadsIDs []string)` - batch bd show
   - `ListOpenIssues()` - single bd list call
   - `GetCommentsBatch(beadsIDs []string)` - parallel goroutines for comments

2. **Rewrote `runStatus()` in `cmd/orch/main.go`:**
   - Fetch all OpenCode sessions upfront (single API call)
   - Fetch all open beads issues upfront (single bd list call)
   - Collect all beads IDs from tmux windows and sessions
   - Batch fetch all comments in parallel
   - Build agent list from pre-fetched data

**Performance improvement:**
- Before: 12.2 seconds (85% CPU - mostly waiting on subprocesses)
- After: 1.05 seconds (430% CPU - parallel execution)
- **11× improvement**

---

## References

**Files Modified:**
- `cmd/orch/main.go:1749-1917` - Rewrote runStatus function
- `pkg/verify/check.go:427-534` - Added batch functions

**Commands Run:**
```bash
# Profile original performance
time orch status  # 12.2s

# Test individual bd call latency
time bd show orch-go-3dem --json  # 0.14s
time bd comments orch-go-3dem --json  # 0.09s

# Verify batch support
bd show id1 id2 id3 --json  # Works, returns array

# Test optimized version
time /tmp/orch-new status  # 1.05s
```

**Tests:**
```bash
go test ./pkg/verify/...  # PASS
go test ./cmd/orch/...    # PASS
go test ./...             # All PASS
```

---

## Investigation History

**2025-12-23 03:10:** Investigation started
- Initial question: Why does orch status take 11s when API is 15ms?
- Context: User reported 11 second delay, suspected liveness checks or workspace scanning

**2025-12-23 03:20:** Root cause identified
- Found 3 bd subprocess calls per agent × 37 agents = ~11 seconds
- OpenCode API confirmed fast (18ms for 54 sessions)

**2025-12-23 03:40:** Optimization implemented
- Added batch/parallel beads functions
- Rewrote runStatus to use batch approach
- Tested: 12.2s → 1.05s (11× improvement)

**2025-12-23 03:50:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: orch status now runs in ~1 second, target achieved
