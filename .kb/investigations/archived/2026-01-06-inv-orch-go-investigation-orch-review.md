<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch review` was slow due to O(n) git log + go build operations for each workspace, reduced from 70s to 5.5s with targeted optimizations.

**Evidence:** Profiled execution showing 101 git log calls (~330ms each = 33s) and 65 potential go build calls (~1s each = 65s) contributing to 70s total runtime.

**Knowledge:** Review listing needs only Phase status + SYNTHESIS.md presence; full verification (git diff, build) should be deferred to `orch complete`. Single `bd list` call is much faster than N individual `bd show` calls.

**Next:** close - implementation complete and validated.

**Promote to Decision:** recommend-no - tactical performance fix, not architectural change.

---

# Investigation: orch review Performance

**Question:** Why does `orch review` take 45+ seconds, and how can we reduce it?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: O(n) Git Operations Per Workspace

**Evidence:** Each workspace with SYNTHESIS.md triggered:
- `git log --name-only --since=<spawn_time>` taking ~330ms per call
- 101 workspaces × 330ms = ~33 seconds

**Source:** `pkg/verify/git_diff.go:125-130`, profiled via `time git log --name-only --pretty=format: --since="2025-12-20"` (0.33s per call)

**Significance:** Git operations were running inside `VerifyCompletionFullWithComments` for each workspace, even though the review command only needs to LIST completions, not fully verify them.

---

### Finding 2: O(n) Build Operations Per Workspace

**Evidence:** Each feature-impl skill workspace potentially triggered:
- `go build ./...` taking ~1s per call
- 65 feature-impl workspaces could trigger 65s of builds

**Source:** `pkg/verify/build_verification.go:133-147`, profiled via `time go build ./...` (0.975s per call)

**Significance:** Build verification is appropriate for `orch complete` (final gate) but wasteful for `orch review` (just listing).

---

### Finding 3: O(n) Beads API Calls for Issue Status

**Evidence:** `filterClosedIssues` made individual `bd show` calls per beads ID:
- 72+ tracked beads IDs
- ~65ms per CLI call (no RPC daemon running)
- 72 × 65ms = ~4.7s just for issue status checks

**Source:** `pkg/verify/beads_api.go:281-364` (`GetIssuesBatch` function)

**Significance:** `bd list` returns all issues in one call (~250ms total) vs individual Show calls. Using ListOpenIssues instead eliminates this O(n) overhead.

---

## Synthesis

**Key Insights:**

1. **Verification scope mismatch** - `orch review` was using full completion verification (`VerifyCompletionFullWithComments`) which includes expensive git diff and build checks. For review purposes, only Phase status and SYNTHESIS.md presence matter.

2. **Batch vs individual API calls** - Individual `bd show` calls for each beads ID are much slower than a single `bd list` call. The existing `ListOpenIssues` function was already available but unused.

3. **Early filtering saves work** - Stale light-tier workspaces (older than 24h) can be skipped before beads API calls, reducing the number of IDs to fetch.

**Answer to Investigation Question:**

The 70-second runtime was caused by three O(n) operations:
1. Git log per workspace (33s)
2. Potential go build per workspace (65s)
3. Individual bd show calls per beads ID (5s)

Fixed by:
1. Created `VerifyCompletionForReview` - lightweight verification skipping git/build
2. Used `ListOpenIssues` instead of `GetIssuesBatch`
3. Added early stale filtering for light-tier workspaces

Result: 70s → 5.5s (12.7x improvement)

---

## Structured Uncertainty

**What's tested:**

- ✅ Runtime reduced from 70s to 5.5s (verified: 3 timed runs, consistent results)
- ✅ All existing review tests pass (verified: `go test ./cmd/orch/... -run Review`)
- ✅ Full test suite passes (verified: `go test ./...`)

**What's untested:**

- ⚠️ Performance with beads RPC daemon running (should be even faster with native RPC)
- ⚠️ Behavior with very large number of open issues (ListOpenIssues might be slow)

**What would change this:**

- Finding would be wrong if beads RPC daemon provides O(1) batch operations
- Finding would be wrong if git operations are already cached

---

## Implementation Recommendations

### Recommended Approach ⭐

**Lightweight Review Verification** - Defer expensive checks to `orch complete`

**Why this approach:**
- Review only needs to show completion status, not validate all requirements
- `orch complete` already runs full verification before closing
- Maintains separation between "list" and "act" operations

**Trade-offs accepted:**
- Review shows "OK" for completions that might fail full verification
- User discovers issues at `orch complete` time, not `orch review` time

**Implementation sequence:**
1. Add `VerifyCompletionForReview` - lightweight verification ✅
2. Use `ListOpenIssues` for batch issue status ✅
3. Add early stale filtering for light-tier workspaces ✅

---

## References

**Files Examined:**
- `cmd/orch/review.go` - Main review command logic
- `pkg/verify/check.go` - Verification functions
- `pkg/verify/beads_api.go` - Beads API helpers
- `pkg/verify/git_diff.go` - Git diff verification
- `pkg/verify/build_verification.go` - Build verification

**Commands Run:**
```bash
# Time orch review (before)
time orch review  # 1:09.95 total

# Time git log operation
time git log --name-only --pretty=format: --since="2025-12-20"  # 0.333s

# Time go build
time go build ./...  # 0.975s

# Time bd commands
time bd list --json  # 0.244s (single call for all issues)
time bd show orch-go-70r3k --json  # 0.065s per call

# Time orch review (after)
time orch review  # 5.489s total
```

---

## Investigation History

**2026-01-07 03:08:** Investigation started
- Initial question: Why does orch review take 45+ seconds?
- Context: User reported slow `orch review` command

**2026-01-07 03:15:** Root cause identified
- Found O(n) git log and go build operations per workspace
- Profiled individual operations to confirm timing

**2026-01-07 03:30:** Solution implemented
- Created VerifyCompletionForReview
- Replaced GetIssuesBatch with ListOpenIssues
- Added early stale filtering

**2026-01-07 03:45:** Investigation completed
- Status: Complete
- Key outcome: Reduced orch review from 70s to 5.5s (12.7x improvement)
