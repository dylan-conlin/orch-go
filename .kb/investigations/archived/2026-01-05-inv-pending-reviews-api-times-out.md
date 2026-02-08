<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The N+1 beads API calls in `/api/pending-reviews` have been eliminated by batching comment fetches for light-tier workspaces.

**Evidence:** API response time dropped from timeout (>30s with 700+ workspaces) to ~10ms consistently (verified: 3 curl tests).

**Knowledge:** The `isLightTierComplete` function was called per-workspace during iteration, causing O(n) sequential API calls. Using `GetCommentsBatch` collects all beads IDs first, then fetches in parallel with a semaphore.

**Next:** Close this issue - fix verified working.

---

# Investigation: Pending Reviews API Times Out with N+1 Beads API Calls

**Question:** Why does `/api/pending-reviews` timeout with 700+ workspaces, and how can we fix the N+1 beads API call pattern?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: N+1 Query Pattern in handlePendingReviews

**Evidence:** In `serve_reviews.go:88`, `isLightTierComplete(dirPath)` was called inside the workspace loop. This function (lines 234-254) calls `verify.GetComments(beadsID)` for each light-tier workspace.

**Source:** `cmd/orch/serve_reviews.go:88`, `cmd/orch/serve_reviews.go:234-254`

**Significance:** With 213 light-tier workspaces, this caused 213 sequential API calls to beads. Even at 100ms per call, that's 21+ seconds of blocking I/O.

---

### Finding 2: Batch Fetching Already Exists in Codebase

**Evidence:** `verify.GetCommentsBatch()` in `pkg/verify/beads_api.go:428` already implements parallel comment fetching with a semaphore (max 20 concurrent calls). This pattern was already used in `review.go:230` but not in `serve_reviews.go`.

**Source:** `pkg/verify/beads_api.go:428-521`, `cmd/orch/review.go:230`

**Significance:** The solution was to apply the same batch-fetch pattern used elsewhere, not to create new infrastructure.

---

### Finding 3: Same Pattern Existed in review.go

**Evidence:** `review.go:181` also called `isLightTierComplete()` during workspace scanning, before the batch fetch at line 230. This needed the same fix.

**Source:** `cmd/orch/review.go:181`

**Significance:** Both files shared the same N+1 pattern and required the same fix approach.

---

## Synthesis

**Key Insights:**

1. **Deferred checking pattern** - Light-tier completion status (Phase: Complete) should be checked AFTER batch fetching comments, not during the initial workspace scan.

2. **Three-phase approach** - The fix restructured both functions to: (1) scan workspaces and collect beads IDs, (2) batch fetch all comments in parallel, (3) process workspaces using pre-fetched data.

3. **Existing infrastructure sufficed** - `GetCommentsBatch` with its parallel goroutines and semaphore was already available; the issue was not using it in these hot paths.

**Answer to Investigation Question:**

The timeout was caused by `isLightTierComplete()` making individual `GetComments()` calls for each of 213+ light-tier workspaces. The fix collects all light-tier beads IDs first, batch fetches comments using `GetCommentsBatch()`, then processes each workspace using the pre-fetched comment map. API response time dropped from timeout to ~10ms.

---

## Structured Uncertainty

**What's tested:**

- ✅ API response time is ~10ms (verified: ran `curl -s http://localhost:5199/api/pending-reviews` 3 times, all <10ms)
- ✅ Build compiles without errors (verified: `go build ./...`)
- ✅ Existing tests pass (verified: `go test -short ./cmd/orch/... ./pkg/verify/...`)

**What's untested:**

- ⚠️ Performance under load with many concurrent requests (not benchmarked)
- ⚠️ Behavior when beads daemon is completely unavailable (not tested)

**What would change this:**

- Finding would be wrong if batch fetching introduces new bottlenecks (e.g., overwhelming beads RPC server)
- If beads daemon stability issues cause batch fetches to fail more frequently than sequential

---

## Implementation Recommendations

**Purpose:** Document the implemented fix for future reference.

### Implemented Approach ⭐

**Batch Comment Fetching with Three-Phase Processing**

**What was implemented:**
1. **serve_reviews.go**: Refactored `handlePendingReviews()` to collect light-tier beads IDs during workspace scan, batch fetch with `GetCommentsBatch()`, then process using pre-fetched comments
2. **review.go**: Applied same pattern to `getCompletionsForReview()` - check `isLightTierWorkspace()` during scan, defer completion check until after batch fetch

**Why this approach:**
- Reuses existing `GetCommentsBatch()` infrastructure
- No new dependencies or API changes
- Pattern already proven in other parts of codebase

**Trade-offs accepted:**
- Memory usage increases slightly (storing all comments in map before processing)
- All workspaces must be scanned before any can be processed (but this was already the case)

---

## References

**Files Modified:**
- `cmd/orch/serve_reviews.go` - Batched comment fetching in handlePendingReviews
- `cmd/orch/review.go` - Batched comment fetching in getCompletionsForReview

**Commands Run:**
```bash
# Build verification
go build ./...

# API performance test
curl -s http://localhost:5199/api/pending-reviews  # ~10ms response time

# Test runs
go test -short ./cmd/orch/... ./pkg/verify/...
```

---

## Investigation History

**2026-01-05 19:15:** Investigation started
- Initial question: Why does pending-reviews API timeout with 700+ workspaces?
- Context: N+1 beads API calls reported in issue description

**2026-01-05 19:30:** Root cause identified
- `isLightTierComplete()` called per-workspace, each making `GetComments()` API call
- 213 light-tier workspaces = 213 sequential API calls

**2026-01-05 19:45:** Fix implemented
- Refactored serve_reviews.go to use batch fetching
- Applied same fix to review.go

**2026-01-05 19:50:** Investigation completed
- Status: Complete
- Key outcome: API response time reduced from timeout to ~10ms
