<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `getCompletionsForSurfacing()` was missing the `filterClosedIssues()` call that `getCompletionsForReview()` has.

**Evidence:** `orch review --architects` correctly showed 0 recommendations while `orch status` showed 1 for closed issue orch-go-ow3g.

**Knowledge:** When adding "lightweight" versions of functions for performance, filtering logic (not just verification) must be preserved.

**Next:** Fix applied and verified. Close issue.

---

# Investigation: Orch Status Shows Stale Architect Recommendations

**Question:** Why does `orch status` show architect recommendations for closed beads issues?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** og-debug-orch-status-shows-30dec agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Two different functions serve completions data

**Evidence:** 
- `getCompletionsForReview()` at review.go:139 - used by `orch review --architects`
- `getCompletionsForSurfacing()` at review.go:313 - used by `orch status` session start

**Source:** cmd/orch/review.go:139-380

**Significance:** The discrepancy between `orch status` and `orch review --architects` is explained by them using different data sources.

---

### Finding 2: `getCompletionsForReview()` filters closed issues

**Evidence:** Line 256 calls `filterClosedIssues(candidates)` which checks beads issue status and removes closed/deferred/tombstone issues.

**Source:** cmd/orch/review.go:256, 259-305

**Significance:** This is why `orch review --architects` correctly shows no recommendations for closed issues.

---

### Finding 3: `getCompletionsForSurfacing()` did NOT filter closed issues

**Evidence:** The original function returned `results` directly without calling `filterClosedIssues()`. The comment even noted it was a "lightweight version" that skips "expensive verification" but inadvertently also skipped the filtering.

**Source:** cmd/orch/review.go:313-381 (original)

**Significance:** This is the root cause. The lightweight function preserved the scanning but dropped the filtering.

---

## Synthesis

**Key Insights:**

1. **Filtering is not just verification** - The `filterClosedIssues()` function is a data integrity filter, not a verification step. When creating "lightweight" versions of functions, all filtering must be preserved.

2. **Two functions, same source, different behavior** - Both functions scan the same workspace directory for SYNTHESIS.md files, but `getCompletionsForReview()` had an additional step that `getCompletionsForSurfacing()` lacked.

3. **Simple one-line fix** - Adding `return filterClosedIssues(results), nil` fixes the issue completely.

**Answer to Investigation Question:**

`orch status` shows stale architect recommendations for closed issues because `getCompletionsForSurfacing()` was missing the `filterClosedIssues()` call that `getCompletionsForReview()` has. The fix was to add this filtering call to the lightweight function.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build passes after fix (`go build ./...`)
- ✅ All existing tests pass (`go test ./cmd/orch/... -run "Review|Architect"`)
- ✅ `orch status` no longer shows stale architect recommendation (verified via CLI)
- ✅ `orch review --architects` still works correctly (verified via CLI)

**What's untested:**

- ⚠️ Performance impact of adding beads RPC call to surfacing (likely minimal, uses batch fetch)

**What would change this:**

- Finding would be wrong if there's another code path for surfacing that also needs the fix

---

## Implementation Recommendations

**Purpose:** The fix has already been implemented.

### Recommended Approach ⭐

**Add filterClosedIssues() call to getCompletionsForSurfacing()** - One-line fix at the return statement.

**Why this approach:**
- Uses existing, tested filtering function
- Minimal change, minimal risk
- Maintains consistency between review and surfacing code paths

**Trade-offs accepted:**
- Adds one batch RPC call to beads for surfacing (negligible cost)

**Implementation sequence:**
1. Add `filterClosedIssues(results)` call before return - DONE
2. Update function comment to document the filtering - DONE
3. Run tests and verify - DONE

---

### Implementation Details

**What was implemented:**
- Added `filterClosedIssues(results)` call at line ~380 in review.go
- Updated function comment to mention beads filtering

**Things to watch out for:**
- ⚠️ N/A - straightforward fix

**Areas needing further investigation:**
- None identified

**Success criteria:**
- ✅ `orch status` no longer shows recommendations for closed issues
- ✅ `orch review --architects` still works correctly
- ✅ All tests pass

---

## References

**Files Examined:**
- cmd/orch/review.go - Main location of both functions
- cmd/orch/main.go - Where `GetArchitectRecommendationsSurface()` is called

**Commands Run:**
```bash
# Verify closed issue status
bd show orch-go-ow3g  # Shows Status: closed

# Build and test
go build ./...  # PASS
go test ./cmd/orch/... -run "Review|Architect" -v  # PASS

# Smoke test
orch status  # No longer shows stale architect recommendation
orch review --architects  # Shows "No architect recommendations awaiting review"
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md - Related to ghost sessions
- **Investigation:** .kb/investigations/2025-12-23-inv-orch-status-shows-active-agents.md - Related to orch status issues

---

## Investigation History

**2025-12-30 ~10:30:** Investigation started
- Initial question: Why does `orch status` show architect recommendations for closed beads issues?
- Context: Symptom reported by orchestrator - inconsistency between status and review commands

**2025-12-30 ~10:35:** Root cause identified
- Found two different functions: `getCompletionsForReview()` vs `getCompletionsForSurfacing()`
- `filterClosedIssues()` call missing from surfacing function

**2025-12-30 ~10:40:** Investigation completed
- Status: Complete
- Key outcome: One-line fix - add `filterClosedIssues()` call to `getCompletionsForSurfacing()`
