# Probe: Daemon Test Fail-Fast Fix

**Date:** 2026-02-17
**Status:** Complete
**Related Issue:** orch-go-1007

## Question

Does the fail-fast change to `expandTriageReadyEpics` (adding error return) require test updates beyond just updating function signatures?

## What I Tested

1. Ran the 5 failing daemon tests to see the specific failures
2. Examined `expandTriageReadyEpics` signature change (now returns 3 values: issues, epicChildIDs, error)
3. Checked daemon test setup to identify what's missing

## What I Observed

**Test Failures:**
- TestDaemon_Once_ProcessesOneIssue: Processed=false, spawnFunc not called
- TestDaemon_Run_ProcessesAllIssues: 0 results (expected 3), 0 spawn calls
- TestDaemon_Run_RespectsMaxIterations: 0 results (expected 5), 0 spawn calls  
- TestDaemon_Once_WithPool_AcquiresSlot: Processed=false, Pool.Active()=0 (want 1)
- TestDaemon_OnceWithSlot_ReturnsSlot: Processed=false, then panic (nil pointer dereference)

**Key Finding:** The tests that directly call `expandTriageReadyEpics` are already updated for the 3-return signature (lines 2182, 2206, 2230, 2390, 2864). The failing tests are higher-level tests calling `Once()` and `Run()` which internally call `expandTriageReadyEpics`.

**Hypothesis:** There's likely an indentation issue in `expandTriageReadyEpics` causing it to return nil/error incorrectly, which makes the daemon flow stop early without processing issues.

## Root Cause Analysis

After investigation, found TWO distinct issues:

1. **Missing Error Checks (Primary Issue):**
   - `NextIssueExcluding()` called `expandTriageReadyEpics()` but didn't check the returned error
   - `Preview()` had the same missing error check
   - Added error checks after both calls:
     ```go
     issues, epicChildIDs, err := d.expandTriageReadyEpics(issues)
     if err != nil {
         return nil, fmt.Errorf("failed to expand epics: %w", err)
     }
     ```

2. **Unmockable UpdateBeadsStatus Calls (Secondary Issue):**
   - `Once()` and `OnceWithSlot()` called `UpdateBeadsStatus()` directly
   - This package-level function couldn't be mocked in tests
   - Tests failed with "no .beads directory found" errors
   - **Solution:** Made function injectable:
     - Added `updateBeadsStatusFunc func(beadsID string, status string) error` field to Daemon struct
     - Updated `OnceExcluding()` and `OnceWithSlot()` to use mockable function:
       ```go
       updateStatus := d.updateBeadsStatusFunc
       if updateStatus == nil {
           updateStatus = UpdateBeadsStatus
       }
       if err := updateStatus(issue.ID, "in_progress"); err != nil {
       ```
     - Updated 50+ daemon tests to provide mock implementation

## Test Results

**Before fix:** 5 tests failing
**After fix:** All tests passing

**Tests fixed:**
- TestDaemon_Once_ProcessesOneIssue
- TestDaemon_Run_ProcessesAllIssues
- TestDaemon_Run_RespectsMaxIterations
- TestDaemon_Once_WithPool_AcquiresSlot
- TestDaemon_OnceWithSlot_ReturnsSlot

Plus ~50 additional tests that needed updateBeadsStatusFunc mocks.

## Model Impact

**Confirms** the model's understanding that the daemon uses fail-fast error handling. The orch-go-nff change introduced error returns to make failures visible, but the callers weren't updated to check those errors.

**Extends** the model with new finding: The daemon's test infrastructure relies heavily on injectable function fields for mocking external dependencies (beads operations). When adding new beads operations (like status updates), they must be made injectable to support testing.
