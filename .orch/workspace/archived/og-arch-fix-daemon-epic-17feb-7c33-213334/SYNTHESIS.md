# SYNTHESIS: Fix Daemon Epic Expansion Fail-Fast

**Issue:** orch-go-j26  
**Agent:** architect  
**Date:** 2026-02-17

---

## Plain-Language Summary

Fixed a silent work-dropping bug in the daemon's epic expansion. When the daemon tried to list children of a `triage:ready` epic and that operation failed, it would log a warning (only in verbose mode) and continue processing other epics. This meant that epic's children would never be added to the spawn queue, silently dropping spawnable work.

The fix changes the daemon to fail-fast on epic expansion errors: if listing children fails, the entire spawn attempt fails immediately with a clear error message. This prevents partial epic processing and ensures failures are visible rather than silent.

Why it matters: The daemon operates autonomously overnight. Silent warnings in verbose logs don't count as user notification. A transient beads API issue during epic expansion could cause an entire epic's worth of work to be skipped without anyone knowing.

---

## Verification Contract

See: `VERIFICATION_SPEC.yaml` (not created for this fix - see test evidence below)

**Changes Made:**
1. Function signature: Added `error` return to `expandTriageReadyEpics`  
   **Location:** `pkg/daemon/daemon.go:406`

2. Error handling: Changed from warn-and-continue to fail-fast  
   **Location:** `pkg/daemon/daemon.go:439-441`  
   **Before:** Log warning in verbose mode, continue to next epic  
   **After:** Return error immediately with epic ID in message

3. Call sites: Added error handling at both call locations  
   **Locations:** `pkg/daemon/daemon.go:316-319, 596-599`  
   **Behavior:** Both `NextIssueExcluding` and `Preview` now propagate epic expansion errors to their callers

4. Test coverage: Added new test for error case  
   **Location:** `pkg/daemon/daemon_test.go:2421` (new test)  
   **Test:** `TestExpandTriageReadyEpics_ListChildrenError`

**Test Evidence:**

```bash
$ go test ./pkg/daemon/... -v -run "TestExpand"
=== RUN   TestExpandTriageReadyEpics_NoEpics
--- PASS: TestExpandTriageReadyEpics_NoEpics (0.00s)
=== RUN   TestExpandTriageReadyEpics_NoLabelFilter
--- PASS: TestExpandTriageReadyEpics_NoLabelFilter (0.00s)
=== RUN   TestExpandTriageReadyEpics_EpicWithoutLabel
--- PASS: TestExpandTriageReadyEpics_EpicWithoutLabel (0.00s)
=== RUN   TestExpandTriageReadyEpics_FiltersClosedChildren
--- PASS: TestExpandTriageReadyEpics_FiltersClosedChildren (0.00s)
=== RUN   TestExpandTriageReadyEpics_ListChildrenError
--- PASS: TestExpandTriageReadyEpics_ListChildrenError (0.00s)
PASS
```

**Reproduction Verified:** ❌ (Cannot reproduce original bug without mocking beads failure)

Instead of reproducing the original bug (which would require forcing beads to fail during ListEpicChildren), I verified the fix through:
1. Code review: Confirmed warn-and-continue pattern was present
2. Unit test: Added test that simulates ListEpicChildren failure
3. Test result: Error is properly returned instead of being swallowed
4. Integration check: All existing epic expansion tests still pass

---

## Probe Findings

**Probe:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-epic-expansion-fail-fast.md`

**Model Impact:**

**Confirms:** The "Daemon Warn-and-Continue Anti-Pattern Audit" probe finding that the daemon has multiple instances of this pattern.

**Extends:** Adds new invariant to Daemon Autonomous Operation model:

> **Epic expansion errors are blocking:** If listing children of a `triage:ready` epic fails, the daemon MUST fail the entire spawn attempt for that cycle rather than silently skipping that epic's children. This prevents partial epic processing where some children get spawned but others are silently dropped.

**Rationale:** Epic expansion is a critical daemon operation. If you can't reliably expand an epic, you can't reliably process the work queue. Better to surface the error immediately than to silently process a subset of available work.

---

## Implementation Details

### Function Signature Change

```go
// Before
func (d *Daemon) expandTriageReadyEpics(issues []Issue) ([]Issue, map[string]bool)

// After
func (d *Daemon) expandTriageReadyEpics(issues []Issue) ([]Issue, map[string]bool, error)
```

### Error Handling Change

```go
// Before (lines 439-444)
if err != nil {
    if d.Config.Verbose {
        fmt.Printf("  DEBUG: Warning: could not list children of epic %s: %v\n", epicID, err)
    }
    continue  // ❌ Silently skip this epic
}

// After (lines 439-441)
if err != nil {
    return nil, nil, fmt.Errorf("failed to list children of epic %s: %w", epicID, err)
}
```

### Call Site Updates

Both call sites (`NextIssueExcluding` and `Preview`) updated to handle the new error return:

```go
// Before
issues, epicChildIDs := d.expandTriageReadyEpics(issues)

// After
issues, epicChildIDs, err := d.expandTriageReadyEpics(issues)
if err != nil {
    return nil, fmt.Errorf("failed to expand epic children: %w", err)
}
```

---

## Discovered Work

**No discovered work.** This was a focused bug fix with clear scope.

---

## Files Modified

- `pkg/daemon/daemon.go` - Function signature, error handling, call sites (3 locations)
- `pkg/daemon/daemon_test.go` - Updated 4 existing test calls, added 1 new test
- `.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-epic-expansion-fail-fast.md` - New probe

---

## Completion Checklist

- [x] Code changes implemented
- [x] Tests added for error case
- [x] All epic expansion tests passing
- [x] Probe file created and updated
- [x] SYNTHESIS.md created
- [x] Ready for commit
