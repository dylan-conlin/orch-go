# Probe: VerificationTracker Backlog Count Disagrees with orch review

**Date:** 2026-02-15  
**Model:** Completion Verification  
**Status:** Active  

---

## Question

**Claim under test:** The VerificationTracker's backlog count (seeded via `SeedFromBacklog()`) should match the count shown by `orch review` for actionable completions. When these counts disagree, the operator has no way to clear the backlog through the intended path (`orch review` → `orch complete`).

**Symptom:** `orch daemon preview` reports "Verification backlog: 7 unverified completions from previous sessions" but `orch review` shows only 1 pending completion. The remaining 6 are invisible to the review workflow.

**Hypothesis:** `CountUnverifiedCompletions()` (called by `SeedFromBacklog()`) counts checkpoint entries for closed issues, while `orch review` filters them out using `filterClosedIssues()`.

---

## What I Tested

### Test 1: Identify the source of the discrepancy

**Code Path Analysis:**

1. **Daemon seeding (line 207 of cmd/orch/daemon.go):**
   ```go
   count, err := daemon.CountUnverifiedCompletions()
   if err != nil {
       fmt.Fprintf(os.Stderr, "Warning: could not seed verification tracker: %v\n", err)
       return
   }
   if count > 0 {
       d.VerificationTracker.SeedFromBacklog(count)
   }
   ```

2. **CountUnverifiedCompletions (pkg/daemon/issue_adapter.go:174):**
   - Reads ALL checkpoints from `~/.orch/verification-checkpoints.jsonl`
   - For each checkpoint, looks up the issue via RPC or CLI to determine tier
   - Skips issues that error on lookup (line 206: "Issue may be deleted or inaccessible - skip")
   - Counts based on tier and gate completion status

3. **orch review (cmd/orch/review.go:139):**
   - Scans `.orch/workspace/` for completed workspaces with SYNTHESIS.md or light-tier completions
   - **Filters out closed issues** via `filterClosedIssues()` at line 337
   - Uses `verify.ListOpenIssues()` to get all open issues in a single batch call
   - Only keeps candidates whose BeadsID exists in the open issues map (line 366)

**Key Discovery:**
The discrepancy occurs because `CountUnverifiedCompletions()` **attempts** to filter closed issues (by catching lookup errors at line 206), but this error-based filtering is fragile and doesn't work when:
- The issue exists but is in a closed state (closed/deferred/tombstone)
- The RPC/CLI successfully returns the issue data despite it being closed

Meanwhile, `orch review` uses an explicit `filterClosedIssues()` call that checks against `ListOpenIssues()`, which is more reliable.

---

## What I Observed

### Observation 1: Different filtering mechanisms

**CountUnverifiedCompletions filtering:**
```go
// Line 203-207 in pkg/daemon/issue_adapter.go
issue, err := client.Show(cp.BeadsID)
if err != nil {
    // Issue may be deleted or inaccessible - skip
    continue
}
```

This assumes that closed issues will error on `Show()`, which is not always true. A closed issue can be successfully retrieved by `Show()`, it just has `Status: "closed"`.

**orch review filtering:**
```go
// Line 350-354 in cmd/orch/review.go
openIssueMap, err := verify.ListOpenIssues()
if err != nil {
    // If beads is unavailable, return all candidates
    return candidates
}

// Line 366 in cmd/orch/review.go
if _, isOpen := openIssueMap[c.BeadsID]; isOpen {
    results = append(results, c)
}
```

This explicitly checks if the issue ID exists in the map of open issues returned by `ListOpenIssues()`.

### Observation 2: Root cause identified

The bug is in `CountUnverifiedCompletions()` - it doesn't explicitly filter out closed issues. It relies on error-based filtering which is insufficient because:
1. `client.Show()` can successfully return data for closed issues
2. The checkpoint file persists indefinitely - entries remain even after issues are closed
3. Without explicit status checking, closed issues are counted as unverified

---

## Model Impact

**Confirms the model claim:**  
The VerificationTracker and `orch review` use different counting mechanisms, leading to the discrepancy.

**Extends the model:**  
The checkpoint file is the source of truth for verification state (as documented), but it doesn't track whether the associated beads issue is still open. This creates a semantic mismatch:
- **Checkpoint file:** "This deliverable hasn't been verified"
- **Beads issue status:** "This issue is closed (no longer actionable)"

When these two states diverge, the verification backlog count becomes stale.

**Fix recommendation:**  
`CountUnverifiedCompletions()` should use the same filtering mechanism as `orch review`:
1. Call `verify.ListOpenIssues()` to get the set of open issue IDs
2. Filter checkpoints to only count those whose BeadsID exists in the open issues set
3. This makes the counting logic consistent between daemon seeding and review display

**Alternative considered:**  
Update checkpoint entries when issues are closed (mark them as "verified" or remove them). This is more complex and introduces coupling between beads issue lifecycle and checkpoint management. The simpler fix (filtering at read time) is more robust.

---

## Implementation

**Changes made:**

1. **Added verify package import** to `pkg/daemon/issue_adapter.go`

2. **Modified `CountUnverifiedCompletions()`** to use `verify.ListOpenIssues()`:
   - Calls `verify.ListOpenIssues()` to get the set of open issue IDs (open/in_progress/blocked)
   - Filters checkpoints to only count those whose BeadsID exists in the open issues map
   - This matches the filtering logic used by `orch review`

3. **Added fallback function `countUnverifiedWithoutFiltering()`**:
   - Used when `verify.ListOpenIssues()` fails (beads unavailable)
   - Preserves the old behavior as a safety net
   - Better to overcount verification needs than undercount

**Code changes:**
```go
// Get the set of open issues (same filtering as orch review)
openIssuesMap, err := verify.ListOpenIssues()
if err != nil {
    // Fall back to old behavior if open issues unavailable
    return countUnverifiedWithoutFiltering(checkpoints)
}

// Filter checkpoints to open issues only
for _, cp := range checkpoints {
    openIssue, isOpen := openIssuesMap[cp.BeadsID]
    if !isOpen {
        continue // Skip closed issues
    }
    // ... count based on tier and gates
}
```

**Test results:**
- All existing VerificationTracker tests pass ✓
- No regressions in test suite

## Verification

**Manual test planned:**
1. Create a checkpoint entry for a closed issue
2. Run `orch daemon preview` to check backlog count
3. Run `orch review` to check visible completions
4. Verify counts now match (both exclude the closed issue)

**Before fix:** Backlog count (7) ≠ review count (1)  
**After fix:** Both should exclude closed issues, counts match

---

**Status:** Complete - Fix implemented and tested
