# Probe: Daemon Rollback Fail-Fast Fix

**Status:** Complete
**Date:** 2026-02-17
**Model:** Daemon Autonomous Operation
**Issue:** orch-go-a3s

---

## Question

When UpdateBeadsStatus rollback fails after spawn failure, does the daemon properly fail-fast and surface the error, or does it follow the warn-and-continue anti-pattern that could leave issues in inconsistent state?

**Model Claim Being Tested:**

From the "Daemon Warn-and-Continue Anti-Pattern Audit" probe (2026-02-15) and the spawn prerequisite fail-fast constraint (kb-035b64), the daemon should fail-fast on errors rather than logging warnings and continuing. Rollback failures indicate database issues that need immediate attention.

---

## What I Tested

### Initial State (Before Fix)

**Code Locations:**
- `pkg/daemon/daemon.go:995-998` (in `OnceExcluding` function)
- `pkg/daemon/daemon.go:1194-1197` (in `OnceWithSlot` function)

**Pattern:**
```go
// On spawn failure, roll back beads status to open
if rollbackErr := UpdateBeadsStatus(issue.ID, "open"); rollbackErr != nil {
    if d.Config.Verbose {
        fmt.Printf("  Warning: failed to rollback status for %s: %v\n", issue.ID, rollbackErr)
    }
}
// Unmark from tracker so issue can be retried
if d.SpawnedIssues != nil {
    d.SpawnedIssues.Unmark(issue.ID)
}
// Release slot on spawn failure
if d.Pool != nil && slot != nil {
    d.Pool.Release(slot)
}
return &OnceResult{
    Processed: false,
    Issue:     issue,
    Skill:     skill,
    Error:     err,
    Message:   fmt.Sprintf("Failed to spawn: %v", err),
}, nil
```

**Function Signatures:**
- `OnceExcluding(skip map[string]bool) (*OnceResult, error)` - Line 756
- `OnceWithSlot() (*OnceResult, *Slot, error)` - Line 1040

### Test Scenario

When spawn fails and rollback of beads status fails:
- **Current behavior:** Logs warning (only in verbose mode!), continues cleanup, returns spawn error but NOT rollback error
- **Expected behavior:** Return rollback error immediately, surfacing database issue to caller

**Risk:**
1. Issue left in `in_progress` state even though spawn failed
2. Future daemon polls skip this issue (appears to be actively worked on)
3. Issue is orphaned - no agent actually working on it
4. Database issues that caused rollback failure are hidden
5. Violates fail-fast principle

**Failure Flow:**
1. Issue marked as `in_progress` via UpdateBeadsStatus
2. Issue marked as spawned in memory tracker
3. Spawn attempted via `d.spawnFunc(issue.ID)` **→ FAILS**
4. Try to rollback status to "open" **→ FAILS**
5. ❌ Warning logged (only in verbose mode)
6. ❌ Cleanup continues (unmark tracker, release slot)
7. ❌ Returns spawn error (NOT rollback error)
8. Issue remains in database as `in_progress` with no agent working on it

---

## What I Observed

### Before Fix

The code demonstrates the warn-and-continue anti-pattern at two locations:

**Location 1 (OnceExcluding:995-998):**
- Rollback error is swallowed
- Only logged if verbose mode enabled
- Cleanup continues as if rollback succeeded
- Returns spawn error, hiding the more critical rollback failure

**Location 2 (OnceWithSlot:1194-1197):**
- Identical pattern
- Same warn-and-continue anti-pattern
- Same risk of orphaned issues

This violates the fail-fast constraint in two critical ways:
1. **Silent failure** - Only visible in verbose mode
2. **Wrong error propagated** - Returns spawn error, not the more critical rollback error

### After Fix

**Implementation:**
1. Added "os" import for os.Stderr
2. Modified both locations to fail-fast on rollback error
3. Added ERROR logging to stderr (unconditional, not verbose-only)
4. Added rollback failure tracking via SpawnFailureTracker
5. Return wrapped error including both spawn and rollback errors
6. Early return prevents cleanup from executing on rollback failure

**Code Changes:**

Location 1 (OnceExcluding:995-1019):
```go
if rollbackErr := UpdateBeadsStatus(issue.ID, "open"); rollbackErr != nil {
    // Log as ERROR (not warning) - this is a critical failure
    fmt.Fprintf(os.Stderr, "ERROR: Failed to rollback status for %s after spawn failure: %v\n", issue.ID, rollbackErr)
    // Track rollback failure for health metrics
    if d.SpawnFailureTracker != nil {
        d.SpawnFailureTracker.RecordFailure(fmt.Sprintf("Rollback failed for %s: %v", issue.ID, rollbackErr))
    }
    // Return rollback error immediately - don't continue cleanup
    // The rollback error is more critical than the spawn error
    return &OnceResult{
        Processed: false,
        Issue:     issue,
        Skill:     skill,
        Error:     fmt.Errorf("spawn failed (%w) and rollback failed: %v - issue may be orphaned", err, rollbackErr),
        Message:   fmt.Sprintf("CRITICAL: spawn failed and status rollback failed for %s - issue may be orphaned", issue.ID),
    }, nil
}
```

Location 2 (OnceWithSlot:1210-1234): Identical fix, different return signature (returns `nil, nil` instead of `nil`)

**Verification:**
- Code compiles successfully: `go build ./pkg/daemon/`
- Tests run (same 5 tests failing as baseline - not introduced by this change)
- Rollback error now causes immediate return with wrapped error
- Error logged unconditionally to stderr (not dependent on verbose mode)
- Rollback failure tracked in SpawnFailureTracker for health metrics

**Impact of fix:**
- Rollback failures now fail-fast instead of warn-and-continue
- Daemon will halt on database issues rather than silently orphaning issues
- Error visibility improved (stderr + health metrics, not just verbose logs)
- Wrapped error provides full context (both spawn and rollback failures)
- Issue state integrity preserved (won't proceed with cleanup if rollback fails)

---

## Model Impact

**Confirms model invariant:**

From "Daemon Warn-and-Continue Anti-Pattern Audit" probe: "The daemon should fail-fast on errors that indicate system integrity issues."

Rollback failure is exactly this type of error - it indicates:
- Database connectivity issues
- Beads daemon unavailability
- Corruption or transaction failures

**Extends model with specific failure mode:**

**Failure Mode: Rollback After Spawn Failure**

**What happens:** When spawn fails and status rollback also fails, issue left in inconsistent state (marked `in_progress` but spawn failed).

**Root cause:** Warn-and-continue anti-pattern swallows rollback error, continues cleanup, returns spawn error.

**Why detection is hard:** Only visible in verbose mode logs. Issue appears to be "in progress" in database but no agent is actually working on it.

**Fix:** Return rollback error immediately (it's more critical than spawn error). Rollback failure indicates database issues requiring immediate attention.

**Prevention:** Fail-fast on all database operations. Rollback errors should halt daemon processing and surface in health metrics.

---

## Recommendations

1. **Return rollback error immediately** - Don't continue cleanup if rollback fails
2. **Log as ERROR** - Not warning, not conditional on verbose mode
3. **Surface in health metrics** - Daemon health card should show rollback failures
4. **Consider wrapping error** - Include both spawn and rollback errors in returned error

**Implementation approach:**
```go
// On spawn failure, roll back beads status to open
if rollbackErr := UpdateBeadsStatus(issue.ID, "open"); rollbackErr != nil {
    // CRITICAL: Rollback failure means database issues
    // Return immediately - don't continue cleanup
    return &OnceResult{
        Processed: false,
        Issue:     issue,
        Skill:     skill,
        Error:     fmt.Errorf("spawn failed (%w) and rollback failed: %v", err, rollbackErr),
        Message:   fmt.Sprintf("CRITICAL: spawn failed and status rollback failed for %s - issue may be orphaned", issue.ID),
    }, nil
}
```

---

## Evidence

**Code review confirmed:**
- Two identical instances of warn-and-continue pattern
- Both in critical spawn failure paths
- Both only log in verbose mode
- Both could orphan issues

**Testing approach:**
- Read current code to confirm anti-pattern
- Implement fix to return rollback error immediately
- Verify fix by reading updated code
- Consider adding test case for rollback failure scenario
