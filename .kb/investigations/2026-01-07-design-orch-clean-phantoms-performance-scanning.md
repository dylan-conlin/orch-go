<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch clean` performance improved from ~16.6s to ~0.15s by using batch beads API lookups instead of sequential per-workspace calls.

**Evidence:** Before: `time orch clean --dry-run` = 16.607s total. After: 0.154s total. Tested with 379 completed workspaces.

**Knowledge:** The bottleneck was 230 sequential beads API calls (~68ms each). Using `ListOpenIssues()` to get all open issues in one call, then checking if beads ID is absent (= closed), eliminates N API calls.

**Next:** Close - fix implemented and verified.

**Promote to Decision:** recommend-no (tactical performance fix, not architectural)

---

# Investigation: Orch Clean Performance Optimization

**Question:** Why does `orch clean --phantoms` (and `orch clean` generally) take >2 minutes when scanning 568 workspaces?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Phantom scanning is actually fast

**Evidence:** `orch clean --phantoms --dry-run` completes in 0.036s. The phantom scanning function `cleanPhantomWindows()` uses `ListSessions()` and tmux window scans, which are efficient.

**Source:** `cmd/orch/clean_cmd.go:589-689` (cleanPhantomWindows function)

**Significance:** The initial symptom ("phantoms takes >2 minutes") was misleading. The real bottleneck is in workspace completion scanning, not phantom detection.

---

### Finding 2: Bottleneck is in findCleanableWorkspaces

**Evidence:** `orch clean` (default) took 16.6 seconds with 416 workspaces. Profiling showed the slow path is checking beads issue status for workspaces without SYNTHESIS.md.

**Source:** `cmd/orch/clean_cmd.go:185-255` (findCleanableWorkspaces function, lines 241-246)

**Significance:** For each workspace without SYNTHESIS.md, the code calls `beadsChecker.IsIssueClosed(beadsID)` which makes an individual API call (~68ms per call).

---

### Finding 3: Sequential API calls cause O(n) latency

**Evidence:** 
- 376 completed workspaces found
- 146 have SYNTHESIS.md (fast file check)
- 230 need beads status check
- 230 × 68ms = ~15.6s (matches observed 16.6s)

**Source:** Command output analysis and timing measurements

**Significance:** The per-workspace beads API pattern is the root cause. Solution requires batch or parallel API calls.

---

### Finding 4: Batch API functions already exist

**Evidence:** `verify.GetIssuesBatch()` and `verify.ListOpenIssues()` are already implemented in `pkg/verify/beads_api.go`. These provide efficient batch operations.

**Source:** `pkg/verify/beads_api.go:303-386` (GetIssuesBatch), `pkg/verify/beads_api.go:392-448` (ListOpenIssues)

**Significance:** The fix doesn't require new infrastructure - just using existing batch APIs instead of sequential calls.

---

## Synthesis

**Key Insights:**

1. **File checks are instant (~0.01s)** - The fast path (SYNTHESIS.md exists) handles most workspaces efficiently. Only workspaces without SYNTHESIS.md trigger slow API calls.

2. **ListOpenIssues() is ideal for "is it closed?" checks** - Instead of asking "is X closed?" for 230 issues, get the open issues list once and check if X is absent.

3. **Fallback pattern preserved** - The optimized code falls back to sequential checking if the batch API fails, maintaining reliability.

**Answer to Investigation Question:**

The `orch clean` command was slow (16.6s) because `findCleanableWorkspaces()` made 230 sequential beads API calls to check if issues were closed. Each call took ~68ms, accumulating to ~15.6s of latency.

The fix uses `ListOpenIssues()` to get all open issues in a single API call (~100ms), then checks if each beads ID is absent from the open list (indicating it's closed). This reduces N API calls to 1, achieving 108x speedup (16.6s → 0.15s).

---

## Structured Uncertainty

**What's tested:**

- ✅ Performance improvement verified: 16.6s → 0.154s (tested with 379 workspaces)
- ✅ Correct output: 379 completed workspaces found (same as before)
- ✅ All tests pass: `go test ./cmd/orch/... -run "Clean"`
- ✅ --phantoms flag still works correctly: 0.037s

**What's untested:**

- ⚠️ Behavior when ListOpenIssues() returns error (fallback path)
- ⚠️ Very large number of open issues (>1000)
- ⚠️ Cross-project workspace handling

**What would change this:**

- Finding would be wrong if ListOpenIssues() has false negatives (misses open issues)
- Finding would be wrong if beads API semantics differ from file-based status

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Use ListOpenIssues() for batch status checking** - Single API call to get all open issues, then check absence = closed.

**Why this approach:**
- Single API call (~100ms) vs N calls (~16s)
- Simple logic: if beads ID not in open list, it's closed
- Matches existing pattern in other commands

**Trade-offs accepted:**
- Relies on ListOpenIssues() returning complete/correct data
- Fallback to sequential adds code complexity

**Implementation sequence:**
1. First pass: File-based checks (SYNTHESIS.md) - unchanged
2. Collect beads IDs needing status check
3. Single ListOpenIssues() call
4. Mark workspace complete if beads ID absent from open list

### Alternative Approaches Considered

**Option B: GetIssuesBatch() with parallel goroutines**
- **Pros:** Explicit status check per issue, uses existing batch function
- **Cons:** Still 230 API calls, just concurrent (20 at a time = ~12 batches × 68ms = ~816ms)
- **When to use instead:** If ListOpenIssues() has reliability issues

**Rationale for recommendation:** ListOpenIssues() provides the clearest semantic match ("get open issues") and lowest latency (1 call vs 12 batches).

---

## References

**Files Examined:**
- `cmd/orch/clean_cmd.go` - Main clean command implementation
- `pkg/verify/beads_api.go` - Batch API functions

**Commands Run:**
```bash
# Measure baseline performance
time orch clean --dry-run  # 16.607s

# Measure after fix
time ./orch clean --dry-run  # 0.154s

# Run tests
go test ./cmd/orch/... -run "Clean" -v  # All pass
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-26-inv-add-orch-clean-phantoms-cleanup.md` - Prior investigation confirming --phantoms works

---

## Investigation History

**2026-01-07:** Investigation started
- Initial question: Why does orch clean --phantoms take >2 minutes?
- Context: Previous session reported timeout scanning 568 workspaces

**2026-01-07:** Root cause identified
- Bottleneck is in findCleanableWorkspaces, not cleanPhantomWindows
- 230 sequential beads API calls at ~68ms each

**2026-01-07:** Fix implemented and verified
- Status: Complete
- Key outcome: 108x performance improvement (16.6s → 0.15s) using batch API
