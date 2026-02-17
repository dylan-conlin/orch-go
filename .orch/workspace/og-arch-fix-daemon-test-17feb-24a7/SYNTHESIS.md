# Synthesis: Fix Daemon Test Failures from Fail-Fast Changes

**Issue:** orch-go-1007  
**Date:** 2026-02-17  
**Worker:** architect

## Plain-Language Summary

Fixed 5 daemon tests that broke after the orch-go-nff fail-fast changes. The issue had two root causes: (1) callers in daemon.go weren't updated to check the new error return from `expandTriageReadyEpics`, and (2) tests couldn't mock the `UpdateBeadsStatus` function because it was called directly as a package-level function instead of through an injectable field. Added error checks in NextIssueExcluding() and Preview(), made UpdateBeadsStatus injectable via a new `updateBeadsStatusFunc` field on the Daemon struct, and updated 50+ tests to provide mock implementations.

## Verification Contract

See: `VERIFICATION_SPEC.yaml`

**Key outcomes:**
- All 5 originally failing tests now pass
- Full daemon test suite (100+ tests) passes
- Tests run without requiring .beads directory setup

## Changes Made

### Code Changes

1. **daemon.go**: Added error checks after `expandTriageReadyEpics()` calls
   - In `NextIssueExcluding()` (line ~322)
   - In `Preview()` (line ~599)

2. **daemon.go**: Made `UpdateBeadsStatus` injectable
   - Added `updateBeadsStatusFunc` field to Daemon struct (line ~232)
   - Updated `OnceExcluding()` to use mockable function (line ~976)
   - Updated `OnceWithSlot()` to use mockable function (line ~1197)

3. **daemon_test.go**: Added `updateBeadsStatusFunc` mocks to ~50 tests
   - All tests that call Once(), OnceExcluding(), or OnceWithSlot()
   - Mock always returns nil (success) for test scenarios

4. **spawn_tracker_test.go**: Added `updateBeadsStatusFunc` mocks to tests

### Probe File

Created probe file documenting the investigation:
- `.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-test-fail-fast-fix.md`

## Technical Details

### Root Cause #1: Missing Error Checks

The orch-go-nff change modified `expandTriageReadyEpics` to return `([]Issue, map[string]bool, error)` instead of `([]Issue, map[string]bool)`. Two callers assigned the error but never checked it:

```go
// Before (implicit nil check)
issues, epicChildIDs, err := d.expandTriageReadyEpics(issues)
// Sort continues...

// After
issues, epicChildIDs, err := d.expandTriageReadyEpics(issues)
if err != nil {
    return nil, fmt.Errorf("failed to expand epics: %w", err)
}
// Sort continues...
```

### Root Cause #2: Unmockable Beads Operations

Tests create Daemon structs with mock functions for external dependencies:
- `listIssuesFunc` - mocks `bd list`
- `spawnFunc` - mocks `orch work`  
- `listEpicChildrenFunc` - mocks epic child listing

But `UpdateBeadsStatus(beadsID, status)` was called directly, requiring a .beads directory. Tests failed with:
```
Error initializing JSONL-only mode: no .beads directory found
```

**Solution:** Follow the established pattern of injectable functions:
```go
// Add to Daemon struct
updateBeadsStatusFunc func(beadsID string, status string) error

// Use in methods
updateStatus := d.updateBeadsStatusFunc
if updateStatus == nil {
    updateStatus = UpdateBeadsStatus
}
if err := updateStatus(issue.ID, "in_progress"); err != nil {
    // handle error
}
```

## Model Impact

The probe confirms the daemon model's claim that orch-go-nff introduced fail-fast error handling. The fix validates that the pattern is correctly implemented - errors are returned and must be checked.

The probe extends the model with a new pattern: All beads operations must be injectable via function fields to support testing without a beads database.

## Lessons Learned

1. **Signature changes require caller updates:** When changing a function to return errors, grep for all callers and add error checks
2. **Test infrastructure patterns must be followed:** The daemon uses injectable functions for all external dependencies - new operations must follow this pattern
3. **Fail-fast is not automatic:** Adding error returns doesn't guarantee errors are checked - requires discipline at call sites
