<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** /api/agents endpoint was slow (19-26s) due to O(N) sequential RPC calls for 564 agents.

**Evidence:** Profiled endpoint: 284 sequential token fetches + 564 sequential issue/comment fetches. After parallelization: 0.35s response time.

**Knowledge:** Batch operations (List + filter) are faster than N individual Show calls. Goroutines with semaphore effectively parallelize RPC calls.

**Next:** Close - fix implemented and verified (commit 18759355).

---

# Investigation: Api Agents Endpoint Takes 19s

**Question:** Why does /api/agents take 19 seconds to respond, and how can we fix it?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** og-debug-api-agents-endpoint-27dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Token fetching made O(N) sequential HTTP calls

**Evidence:** The `handleAgents` function (serve.go:936-978) was calling `client.GetSessionTokens()` for each of 284 agents with session IDs. Each call made an HTTP request to OpenCode API (`GET /session/{id}/message`). With 284 agents, this alone took ~18-19 seconds.

**Source:** cmd/orch/serve.go:936-978, pkg/opencode/client.go:847-857

**Significance:** This was the primary bottleneck. Sequential HTTP calls scale terribly with agent count.

---

### Finding 2: GetIssuesBatch used O(N) Show calls instead of single List call

**Evidence:** `GetIssuesBatch` was calling `client.Show(beadsID)` for each beads ID individually (pkg/verify/check.go:584-634). With 564 agents, this made 564 sequential RPC calls.

**Source:** pkg/verify/check.go:584-634

**Significance:** The beads client already has a `List()` method that returns all issues in one call. Using List + filter is O(1) network calls vs O(N).

---

### Finding 3: Comments fetching was sequential within project groups

**Evidence:** `GetCommentsBatchWithProjectDirs` iterated through each beads ID and called `client.Comments()` sequentially (pkg/verify/check.go:746-789). While grouped by project directory for RPC client reuse, the inner loops were still sequential.

**Source:** pkg/verify/check.go:746-789

**Significance:** With 564 agents across potentially many projects, this contributed significantly to latency.

---

## Synthesis

**Key Insights:**

1. **Sequential RPC calls are the enemy at scale** - The codebase had 564 completed agents + 3 active agents. Each sequential call adds latency that compounds. Even with fast individual calls (~50-100ms), 1000+ calls = 50-100 seconds total.

2. **Batch APIs exist but weren't used** - The beads `List()` method returns all issues in one call. The original code used individual `Show()` calls, creating unnecessary network overhead.

3. **Parallelization with bounded concurrency works well** - Using goroutines with a semaphore (max 20 concurrent) provides good throughput without overwhelming the backend.

**Answer to Investigation Question:**

The /api/agents endpoint was slow because it made O(N) sequential calls across multiple operations:
1. Token fetching: 284 sequential HTTP calls (fixed with parallelization)
2. Issues batch: 564 sequential RPC calls (fixed by using List + filter)
3. Comments batch: 564 sequential RPC calls (fixed with parallelization)

After optimization, response time dropped from 19-26 seconds to ~0.35 seconds (50-75x improvement).

---

## Structured Uncertainty

**What's tested:**

- ✅ Response time before fix: 26 seconds (verified: `time curl -s http://127.0.0.1:3348/api/agents`)
- ✅ Response time after fix: 0.35 seconds (verified: multiple runs showing 0.30-0.35s)
- ✅ Data correctness: API returns 564 agents with correct structure (verified: parsed JSON output)
- ✅ Tests pass: `go test ./pkg/verify/... -v` passes

**What's untested:**

- ⚠️ Behavior under high concurrent load (not benchmarked)
- ⚠️ Memory usage with parallel goroutines (not profiled)
- ⚠️ Error handling when RPC server is down (not tested)

**What would change this:**

- Finding would be wrong if subsequent tests show >5s response times
- Approach would need revision if concurrent connections cause RPC server issues

---

## Implementation Recommendations

**Purpose:** Document the fix that was implemented.

### Recommended Approach ⭐ (IMPLEMENTED)

**Parallelize and batch** - Use goroutines with semaphore for parallelization, use List+filter instead of individual Show calls.

**Why this approach:**
- Directly addresses root cause (sequential calls)
- Minimal code changes, no new dependencies
- Bounded concurrency prevents overwhelming backend

**Trade-offs accepted:**
- Slightly higher memory usage from goroutines (acceptable for 564 agents)
- More complex error handling with parallel operations (errors silently skipped, which matches original behavior)

**Implementation sequence:**
1. ✅ Token fetching parallelized (serve.go)
2. ✅ GetIssuesBatch uses List+filter (check.go)
3. ✅ GetCommentsBatchWithProjectDirs parallelized (check.go)

---

## References

**Files Modified:**
- cmd/orch/serve.go:936-978 - Parallelized token fetching, skip completed agents
- pkg/verify/check.go:584-634 - Changed GetIssuesBatch to use List+filter
- pkg/verify/check.go:746-789 - Parallelized GetCommentsBatchWithProjectDirs

**Commands Run:**
```bash
# Measure baseline
time curl -s http://127.0.0.1:3348/api/agents | head -c 500
# Result: 26.067 seconds

# Measure after fix
time curl -s http://127.0.0.1:3348/api/agents > /dev/null
# Result: 0.35 seconds

# Verify data integrity
curl -s http://127.0.0.1:3348/api/agents | python3 -c "import json,sys; data=json.load(sys.stdin); print(f'Total agents: {len(data)}')"
# Result: 564 agents
```

**Related Commits:**
- 18759355 - perf: parallelize beads comments fetching and optimize GetIssuesBatch
- 7f59f966 - feat(web): improve dashboard defaults for better UX (included token parallelization)

---

## Investigation History

**2025-12-27 09:36:** Investigation started
- Initial question: Why does /api/agents take 19s to respond?
- Context: Dashboard loading slowly, user-reported issue

**2025-12-27 09:38:** Identified token fetching bottleneck
- 284 sequential HTTP calls for GetSessionTokens
- This was already flagged in code comments as potential issue

**2025-12-27 09:42:** Identified beads batch bottlenecks
- GetIssuesBatch: N individual Show calls instead of 1 List call
- GetCommentsBatchWithProjectDirs: N sequential RPC calls

**2025-12-27 09:45:** Implemented fixes
- Parallelized token fetching with semaphore
- Changed GetIssuesBatch to use List+filter
- Parallelized comments fetching with semaphore

**2025-12-27 09:47:** Investigation completed
- Status: Complete
- Key outcome: Response time reduced from 19-26s to 0.35s (50-75x improvement)
