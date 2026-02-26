## Summary (D.E.K.N.)

**Delta:** Added ReviewState tracking to workspace via .review-state.json, enabling CLI and dashboard to share synthesis recommendation review state.

**Evidence:** All tests pass (33 verify tests); build compiles; ReviewState persists ReviewedAt, ActedOn, Dismissed fields with Load/Save functions.

**Knowledge:** Review state must be persisted per-workspace at completion time; tracking indices as ints works well for multi-recommendation synthesis files.

**Next:** Close - implementation complete; Phase 3 (dashboard UI for synthesis review) can build on this foundation.

**Confidence:** High (90%) - All core functionality implemented and tested.

---

# Investigation: Add Review State Tracking Workspace

**Question:** How to track which synthesis recommendations have been reviewed, acted on, or dismissed across CLI and dashboard?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: ReviewState struct created in pkg/verify

**Evidence:** Created `review_state.go` with ReviewState struct containing:
- `ReviewedAt time.Time` - when orchestrator reviewed
- `ActedOn []int` - indices of recommendations that became issues
- `Dismissed []int` - indices of recommendations explicitly skipped
- `WorkspaceID`, `BeadsID`, `TotalRecommendations` for reference

**Source:** `pkg/verify/review_state.go:17-33`

**Significance:** Provides clean separation of review state from synthesis parsing. Helper methods (IsReviewed, AllActedOn, UnreviewedCount) enable dashboard queries.

---

### Finding 2: Review state persisted during runReviewDone()

**Evidence:** Updated `cmd/orch/review.go:631-702` to track which recommendations were acted on vs dismissed during the prompt flow, then save via `verify.SaveReviewState()`.

**Source:** `cmd/orch/review.go:627-704`

**Significance:** Review state is saved immediately after user responds, enabling dashboard to show which recommendations are still pending.

---

### Finding 3: Dashboard API endpoints implemented

**Evidence:** Added handlers:
- `GET /api/pending-reviews` - returns agents with unreviewed recommendations
- `POST /api/dismiss-review` - dismisses a specific recommendation by workspace+index

**Source:** `cmd/orch/serve.go:2270-2485`

**Significance:** Dashboard can now display pending synthesis reviews and allow dismissal without re-running the full `orch review done` flow.

---

## Synthesis

**Key Insights:**

1. **Per-workspace file storage** - Using `.review-state.json` in each workspace directory allows state to persist across sessions and be easily cleaned up when workspace is deleted.

2. **Index-based tracking** - Tracking recommendations by index (not text) is resilient to future synthesis changes while being simple to implement and query.

3. **Shared state enables hybrid workflows** - CLI `orch review done` saves state that dashboard can query, and dashboard dismissals are respected by CLI on next run.

**Answer to Investigation Question:**

Review state is tracked via a `.review-state.json` file in each workspace directory, containing timestamp, acted-on indices, and dismissed indices. The pkg/verify package provides ReviewState struct with Load/Save functions. The CLI persists state after prompts, and the dashboard can query/update state via API endpoints.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

All core functionality implemented and passing tests. The design follows the pattern established in the investigation doc.

**What's certain:**

- ReviewState struct with correct fields
- Load/Save functions work correctly (tested)
- CLI persists state after user responds
- Dashboard endpoints compile and follow correct pattern

**What's uncertain:**

- Dashboard UI not implemented (Phase 3 future work)
- Integration testing with real synthesis files not done
- Edge case: what happens if synthesis changes between review sessions

---

## Implementation Recommendations

Not applicable - implementation is complete. This was the implementation task.

---

## References

**Files Created/Modified:**
- `pkg/verify/review_state.go` - New ReviewState struct and persistence
- `pkg/verify/review_state_test.go` - Tests for ReviewState
- `cmd/orch/review.go` - Updated runReviewDone() to persist state
- `cmd/orch/serve.go` - Added pending-reviews and dismiss-review endpoints

---

## Investigation History

**2025-12-26:** Implementation started
- Task: Add review state tracking to workspace per Phase 2 design
- Created ReviewState struct, tests, CLI integration, dashboard endpoints

**2025-12-26:** Implementation completed
- All tests passing
- Build compiles
- Ready for commit
