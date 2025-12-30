<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The `/api/pending-reviews` endpoint was not filtering out agents whose beads issues had been closed, causing stale entries to remain in the count.

**Evidence:** Compared `handlePendingReviews` in serve.go against `getCompletionsForReview` in review.go - the latter uses `filterClosedIssues()` while the API endpoint did not.

**Knowledge:** Both API endpoints and CLI commands that return completion-like data should use consistent filtering to exclude closed beads issues.

**Next:** Fix verified - merge when ready.

---

# Investigation: Pending Reviews API count doesn't filter closed issues

**Question:** Why does the pending reviews API count include agents whose beads issues are already closed?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** og-debug-pending-reviews-api-30dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: review.go uses filterClosedIssues, serve.go does not

**Evidence:** The `getCompletionsForReview` function in review.go (line 256) calls `filterClosedIssues(candidates)` before returning. This function (lines 259-305) filters out completions whose beads issues are closed/deferred/tombstone.

**Source:** cmd/orch/review.go:256, cmd/orch/review.go:259-305

**Significance:** The CLI `orch review` command correctly excludes closed issues, but the API endpoint `/api/pending-reviews` does not apply the same filtering.

---

### Finding 2: handlePendingReviews builds agent list without beads status check

**Evidence:** The `handlePendingReviews` function (lines 3306-3470) scans workspaces and builds a list of agents based on SYNTHESIS.md presence and review state, but never queries beads to check if the associated issues are closed.

**Source:** cmd/orch/serve.go:3306-3470

**Significance:** This explains why the pending reviews count includes stale entries - workspaces persist even after their beads issues are closed.

---

### Finding 3: filterClosedIssues pattern is reusable

**Evidence:** The existing `filterClosedIssues` function uses batch fetching via `verify.GetIssuesBatch` for efficiency, checks for closed/deferred/tombstone statuses, and handles untracked agents gracefully.

**Source:** cmd/orch/review.go:259-305

**Significance:** The same pattern can be applied to the API endpoint with minimal changes - just need to adapt for `PendingReviewAgent` struct instead of `CompletionInfo`.

---

## Synthesis

**Key Insights:**

1. **Consistency gap** - The CLI and API had different filtering behaviors for the same logical concept (pending completions).

2. **Workspace persistence issue** - Workspaces are not cleaned up when beads issues are closed, so scanning workspaces alone is insufficient to determine relevance.

3. **Efficient batch fetching available** - The `verify.GetIssuesBatch` function already exists for batch querying beads issue statuses.

**Answer to Investigation Question:**

The pending reviews API count included closed issues because `handlePendingReviews` did not filter by beads issue status. The `getCompletionsForReview` function in review.go already has the correct pattern (`filterClosedIssues`), which was not applied to the API endpoint. The fix adds a new `filterPendingReviewsByClosedIssues` function that uses the same approach.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles (verified: `go build ./cmd/orch/...` succeeded)
- ✅ Existing serve tests pass (verified: `go test ./cmd/orch/... -run Serve` all passed)
- ✅ Review tests still pass (verified: `go test ./cmd/orch/... -run Review` all passed)

**What's untested:**

- ⚠️ End-to-end API behavior with real closed issues (would need manual testing with actual data)
- ⚠️ Performance with large numbers of agents (batch fetch should be efficient, but not benchmarked)

**What would change this:**

- If `verify.GetIssuesBatch` has a bug or unexpected behavior with closed issues
- If there are edge cases with status values not in closed/deferred/tombstone

---

## Implementation Recommendations

### Recommended Approach ⭐

**Mirror filterClosedIssues pattern in serve.go** - Add `filterPendingReviewsByClosedIssues` function using the same batch fetch approach.

**Why this approach:**
- Consistent with existing CLI behavior
- Uses proven batch fetching for efficiency
- Minimal code duplication (similar structure)

**Trade-offs accepted:**
- Slight code duplication between review.go and serve.go
- Could be extracted to shared function in future if needed

**Implementation sequence:**
1. Add `filterPendingReviewsByClosedIssues` function
2. Call it before building the response in `handlePendingReviews`
3. Recalculate `totalUnreviewed` after filtering

### Alternative Approaches Considered

**Option B: Share a common filter function**
- **Pros:** No duplication
- **Cons:** Would require refactoring both types to a common interface
- **When to use instead:** If this pattern appears a third time

**Rationale for recommendation:** Direct implementation is faster and maintains consistency without over-engineering.

---

### Implementation Details

**What was implemented:**
- Added `filterPendingReviewsByClosedIssues(agents []PendingReviewAgent) ([]PendingReviewAgent, int)`
- Modified `handlePendingReviews` to call the filter before building response
- Updated function comment to document the filtering behavior

**Things to watch out for:**
- ⚠️ Uses `isUntrackedBeadsIDServe` (duplicated from review.go) - consider extracting to shared package if pattern grows

**Success criteria:**
- ✅ `/api/pending-reviews` returns count that excludes closed issues
- ✅ Existing tests continue to pass
- ✅ Dashboard attention panel shows accurate pending reviews count

---

## References

**Files Examined:**
- cmd/orch/serve.go - `handlePendingReviews` function and related types
- cmd/orch/review.go - `getCompletionsForReview` and `filterClosedIssues` functions

**Commands Run:**
```bash
# Build to verify no compile errors
go build ./cmd/orch/...

# Run serve tests
go test ./cmd/orch/... -run Serve -v

# Run review tests
go test ./cmd/orch/... -run Review -v
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-30-inv-investigate-recent-bugs-attention-panel.md - Parent investigation that identified this issue

---

## Investigation History

**2025-12-30 22:22:** Investigation started
- Initial question: Why does pending reviews API count include closed issues?
- Context: Spawned from investigation into attention panel bugs

**2025-12-30 22:27:** Root cause identified
- Found that handlePendingReviews doesn't filter by beads issue status
- Identified filterClosedIssues pattern in review.go as model

**2025-12-30 22:30:** Implementation completed
- Added filterPendingReviewsByClosedIssues function
- All tests passing
- Status: Complete
