<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** handleDismissReview in serve.go assumes SYNTHESIS.md exists, causing API errors for light-tier agents.

**Evidence:** Code trace shows ParseSynthesis at line 2665-2672 returns error when file doesn't exist; light-tier agents have no SYNTHESIS.md by design.

**Knowledge:** Light-tier agents need separate dismiss path that sets LightTierAcknowledged=true in ReviewState.

**Next:** Implement conditional handling in handleDismissReview to check for light-tier and update review state appropriately.

---

# Investigation: Dismiss Button Light Tier Pending

**Question:** Why does the "Dismiss All" button for light-tier pending reviews fail with API error?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Agent og-debug-dismiss-button-light-27dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: handleDismissReview assumes SYNTHESIS.md exists

**Evidence:** In `cmd/orch/serve.go` lines 2665-2672, `handleDismissReview` calls `verify.ParseSynthesis(workspacePath)` unconditionally:
```go
synthesis, err := verify.ParseSynthesis(workspacePath)
if err != nil {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(DismissReviewResponse{
        Success: false,
        Error:   fmt.Sprintf("Failed to parse synthesis: %v", err),
    })
    return
}
```

**Source:** `cmd/orch/serve.go:2665-2672`

**Significance:** When dismissing a light-tier agent's pending review, the API fails because `ParseSynthesis` returns an error when the file doesn't exist.

---

### Finding 2: Light-tier agents create a "pseudo-item" in pending reviews

**Evidence:** In `handlePendingReviews` (lines 2527-2548), light-tier completions are represented with a single pseudo-item:
```go
items := []PendingReviewItem{
    {
        WorkspaceID: dirName,
        BeadsID:     lightBeadsID,
        Index:       0,
        Text:        "Light tier agent completed - no synthesis produced (by design). Review and close via orch complete.",
        Reviewed:    false,
        ActedOn:     false,
        Dismissed:   false,
    },
}
```

**Source:** `cmd/orch/serve.go:2527-2536`

**Significance:** The pseudo-item has index 0, and dismissing it should set `LightTierAcknowledged = true` in the review state, not try to parse a synthesis file.

---

### Finding 3: ReviewState already has LightTierAcknowledged field

**Evidence:** In `pkg/verify/review_state.go` line 38:
```go
// LightTierAcknowledged is true when a light-tier agent completion has been reviewed.
// Light-tier agents don't produce SYNTHESIS.md, so this field tracks acknowledgment
// of the completion itself rather than synthesis recommendations.
LightTierAcknowledged bool `json:"light_tier_acknowledged,omitempty"`
```

**Source:** `pkg/verify/review_state.go:35-38`

**Significance:** The infrastructure for tracking light-tier dismissals already exists. We just need to use it in `handleDismissReview`.

---

## Synthesis

**Key Insights:**

1. **Design oversight** - `handleDismissReview` was implemented for full-tier agents with synthesis, but light-tier agents were added later without updating the dismiss handler.

2. **Solution is simple** - Before calling `ParseSynthesis`, check if the workspace is light-tier using `isLightTierWorkspace`. If so, set `LightTierAcknowledged = true` and return early.

3. **Validation is minimal** - Light-tier has only one pseudo-item at index 0, so no need to validate indices against synthesis.NextActions length.

**Answer to Investigation Question:**

The "Dismiss All" button fails for light-tier agents because `handleDismissReview` unconditionally tries to parse `SYNTHESIS.md`, which doesn't exist for light-tier agents. The fix is to check for light-tier workspaces first and handle them by setting `LightTierAcknowledged = true` in the review state.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code path traced from handleDismissReview to ParseSynthesis error (verified: read code)
- ✅ isLightTierWorkspace function exists and works (verified: used in handlePendingReviews successfully)
- ✅ LightTierAcknowledged field exists in ReviewState (verified: read review_state.go)

**What's untested:**

- ⚠️ Fix implementation (will test after implementing)
- ⚠️ Dashboard "Dismiss All" behavior after fix (will test via browser)

**What would change this:**

- Finding would be wrong if error came from a different source than ParseSynthesis (unlikely, error message matches)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach

**Add light-tier handling to handleDismissReview** - Check if workspace is light-tier before parsing synthesis; if so, set LightTierAcknowledged and return success.

**Why this approach:**
- Uses existing `isLightTierWorkspace` helper (no new code needed for detection)
- Uses existing `LightTierAcknowledged` field (infrastructure already built)
- Minimal code change with localized impact

**Implementation sequence:**
1. After workspace path validation, call `isLightTierWorkspace(workspacePath)`
2. If true, load review state, set `LightTierAcknowledged = true`, save, return success
3. Otherwise, continue with existing synthesis-based dismissal logic

### Alternative Approaches Considered

**Option B: Handle at dashboard level (skip dismiss API for light-tier)**
- **Pros:** No backend change needed
- **Cons:** API would still return error if called; duplicates light-tier detection logic in frontend
- **When to use instead:** If we wanted to remove light-tier from pending reviews entirely

**Rationale for recommendation:** Backend fix is cleaner and ensures API behaves correctly for all clients.

---

### Implementation Details

**What to implement first:**
- Add light-tier check at start of `handleDismissReview` function
- Set `LightTierAcknowledged = true` in ReviewState for light-tier dismissals

**Things to watch out for:**
- Need to ensure `reviewState.ReviewedAt` is set when acknowledging
- Need to set `WorkspaceID` from the request

**Success criteria:**
- ✅ "Dismiss All" button works for light-tier pending reviews
- ✅ API returns `{"success": true}` for light-tier dismissals
- ✅ Light-tier reviews disappear from pending list after dismissal

---

## References

**Files Examined:**
- `cmd/orch/serve.go:2625-2723` - handleDismissReview function
- `cmd/orch/serve.go:2400-2564` - handlePendingReviews function (light-tier handling)
- `cmd/orch/serve.go:2576-2609` - isLightTierWorkspace and isLightTierComplete helpers
- `pkg/verify/review_state.go` - ReviewState struct with LightTierAcknowledged field
- `pkg/verify/check.go:167-214` - ParseSynthesis function

**Related Artifacts:**
- **Decision:** Light-tier pending reviews are grouped by default in Swarm Dashboard (from kb context)

---

## Investigation History

**2025-12-27:** Investigation started
- Initial question: Why does Dismiss All button fail for light-tier pending reviews?
- Context: API error "Failed to parse synthesis: SYNTHESIS.md: no such file or directory"

**2025-12-27:** Root cause identified
- handleDismissReview assumes all workspaces have SYNTHESIS.md
- Light-tier agents don't produce SYNTHESIS.md by design
- Fix: Add light-tier handling before synthesis parsing
