<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard "create issue" action didn't persist to `.review-state.json` because no API endpoint existed for marking items as acted on.

**Evidence:** Traced code flow: `handleCreateIssue` calls `markActedOn()` which only updated local Svelte store state; dismiss works because it calls `/api/dismiss-review` endpoint which persists to filesystem.

**Knowledge:** Frontend state updates must be mirrored by backend persistence for changes to survive page reloads.

**Next:** Implementation complete - added `/api/act-on-review` endpoint and updated `markActedOn()` to call it.

---

# Investigation: Dashboard Attention Section Shows Items After Issue Creation

**Question:** Why do items in the "Needs Attention" section persist after clicking "create issue"?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Create Issue Flow Doesn't Persist ActedOn State

**Evidence:** In `needs-attention.svelte` line 81, when issue creation succeeds:
```typescript
const result = await createIssue(cleanAction.substring(0, 100), description, ['triage:ready']);
if (result) {
    createdIssues[key] = result.id;
    createdIssues = createdIssues;
    pendingReviews.markActedOn(agent.workspace_id, item.index);  // <-- Local only!
}
```

**Source:** `web/src/lib/components/needs-attention/needs-attention.svelte:77-82`

**Significance:** The `markActedOn` function only updated local Svelte store state without persisting to the backend.

---

### Finding 2: Dismiss Works Because It Has Backend Persistence

**Evidence:** In `pending-reviews.ts` lines 63-111, the `dismiss` function calls the backend:
```typescript
async dismiss(workspaceId: string, index: number): Promise<boolean> {
    const response = await fetch(`${API_BASE}/api/dismiss-review`, {
        method: 'POST',
        // ...
    });
    // Updates local state only after API success
}
```

The backend endpoint `handleDismissReview` in `serve.go` persists to `.review-state.json`.

**Source:** `web/src/lib/stores/pending-reviews.ts:63-111`, `cmd/orch/serve.go:2685-2832`

**Significance:** This is the correct pattern - local state update should follow successful API call, not replace it.

---

### Finding 3: ReviewState Supports ActedOn Tracking

**Evidence:** The `ReviewState` struct already has the `ActedOn` field:
```go
type ReviewState struct {
    // ActedOn contains indices of recommendations that became beads issues
    ActedOn []int `json:"acted_on,omitempty"`
    // Dismissed contains indices of recommendations explicitly skipped
    Dismissed []int `json:"dismissed,omitempty"`
    // ...
}
```

**Source:** `pkg/verify/review_state.go:20-21`

**Significance:** The data model already supports tracking acted-on items; only the API endpoint was missing.

---

## Synthesis

**Key Insights:**

1. **Asymmetric implementation** - Dismiss had full frontend-to-backend flow, but act-on only had frontend
2. **Persistence gap** - Without API persistence, items reappear on page reload
3. **Simple fix** - Adding `/api/act-on-review` endpoint mirrors the existing `/api/dismiss-review` pattern

**Answer to Investigation Question:**

Items persisted in "Needs Attention" after issue creation because the `markActedOn()` function only updated local Svelte store state without calling a backend API to persist the change. The `.review-state.json` file was never updated with the acted-on index, so when the page reloaded (or `/api/pending-reviews` was called), the item appeared unreviewed again.

---

## Structured Uncertainty

**What's tested:**

- ✅ Go code compiles successfully (verified: `go build ./cmd/orch/...`)
- ✅ No TypeScript errors in modified files (verified: `bun run check` shows no issues in pending-reviews.ts or needs-attention.svelte)
- ✅ Code pattern matches existing dismiss-review endpoint

**What's untested:**

- ⚠️ Full end-to-end browser testing not performed (no browser access in this environment)
- ⚠️ API endpoint manual testing not performed (would require running server)

**What would change this:**

- Finding would be wrong if there's additional caching or state management layer not discovered
- Implementation would fail if API response structure differs from what frontend expects

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Add /api/act-on-review endpoint** - Mirror the existing dismiss-review pattern

**Why this approach:**
- Consistent with existing `/api/dismiss-review` pattern
- Uses existing `ReviewState.ActedOn` field already in data model
- Minimal code change with clear semantics

**Trade-offs accepted:**
- Could have batched acted-on and dismiss into single endpoint, but separate endpoints are clearer

**Implementation sequence:**
1. Add endpoint registration in `serve.go` route setup
2. Add handler function `handleActOnReview`
3. Update `markActedOn` in frontend to call API

### Changes Made

**File 1: cmd/orch/serve.go**
- Added route: `mux.HandleFunc("/api/act-on-review", corsHandler(handleActOnReview))`
- Added types: `ActOnReviewRequest`, `ActOnReviewResponse`
- Added handler: `handleActOnReview()` - mirrors `handleDismissReview()` but updates `ActedOn` instead of `Dismissed`

**File 2: web/src/lib/stores/pending-reviews.ts**
- Changed `markActedOn` from sync function to async
- Added API call to `/api/act-on-review` before updating local state

**File 3: web/src/lib/components/needs-attention/needs-attention.svelte**
- Added `await` to `pendingReviews.markActedOn()` call

---

## References

**Files Examined:**
- `web/src/lib/components/needs-attention/needs-attention.svelte` - Dashboard component with create issue button
- `web/src/lib/stores/pending-reviews.ts` - Store managing pending review state
- `cmd/orch/serve.go` - API server with endpoint handlers
- `pkg/verify/review_state.go` - ReviewState struct definition

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# TypeScript check
bun run check

# Test run
go test ./cmd/orch/... -v -run "TestServe"
```

---

## Self-Review

- [x] Real test performed (build and type check)
- [x] Conclusion from evidence (traced actual code flow)
- [x] Question answered (identified missing API endpoint)
- [x] File complete
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-28 [start]:** Investigation started
- Initial question: Why do items persist in Needs Attention after issue creation?
- Context: Dashboard bug report

**2025-12-28 [finding]:** Root cause identified
- markActedOn only updates local state, doesn't persist to API

**2025-12-28 [complete]:** Investigation completed with implementation
- Status: Complete
- Key outcome: Added /api/act-on-review endpoint and updated frontend to call it
