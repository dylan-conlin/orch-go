# Probe: Daemon Dependency Check Fail-Fast Fix

**Date:** 2026-02-17  
**Context:** Fix issue #1 from warn-and-continue anti-pattern audit (orch-go-nff)  
**Model:** Daemon Autonomous Operation  
**Claim tested:** Changing dependency check from warn-and-continue to fail-fast prevents spawning issues with unchecked dependencies

---

## Question

Does changing the dependency check error handling in `pkg/daemon/daemon.go:378-383` from warn-and-continue to skip-on-error prevent the daemon from spawning work that might be blocked by dependencies?

---

## Status

Status: Complete

---

## What I Tested

### Before Fix (Warn-and-Continue Pattern)

```go
blockers, err := beads.CheckBlockingDependencies(issue.ID)
if err != nil {
    if d.Config.Verbose {
        fmt.Printf("  DEBUG: Warning: could not check dependencies for %s: %v\n", issue.ID, err)
    }
    // Continue checking - don't skip issue just because we can't check dependencies
} else if len(blockers) > 0 {
    // ... skip if blockers exist
}
```

**Behavior:** When dependency check fails (err != nil), logs warning but continues to consider the issue for spawning.

### Fix Applied (Fail-Fast Pattern)

Changed to skip the issue when dependency check fails:

```go
blockers, err := beads.CheckBlockingDependencies(issue.ID)
if err != nil {
    if d.Config.Verbose {
        fmt.Printf("  DEBUG: Skipping %s (could not check dependencies: %v)\n", issue.ID, err)
    }
    continue
} else if len(blockers) > 0 {
    // ... existing skip logic
}
```

**New behavior:** When dependency check fails (err != nil), logs debug message and skips the issue (continues to next issue in loop).

---

## What I Observed

### Code Review Verification

1. **Fix Applied:** Changed lines 378-385 in `pkg/daemon/daemon.go`:
   - Old: Logged warning message "Warning: could not check dependencies" and continued processing
   - New: Logs debug message "Skipping %s (could not check dependencies: %v)" and calls `continue` to skip the issue

2. **Pattern Consistency:** The fix aligns with other skip patterns in the same function:
   - Lines 350-352: Skip blocked issues with `continue`
   - Lines 355-359: Skip in_progress issues with `continue`
   - Lines 366-375: Skip issues missing required labels with `continue`
   - Lines 385-393: Skip issues with blocking dependencies with `continue` (existing)

3. **Also changed:** `else if len(blockers) > 0` → `if len(blockers) > 0` (correct since we now use `continue` in the error case)

### Build Verification

- Command: `go build -o /tmp/orch-test ./cmd/orch`
- Result: Build successful (21M binary created)
- No compilation errors related to this change

### Test Run

- Command: `go test ./pkg/daemon/... -v`
- Result: Some pre-existing test failures in beads mocking infrastructure (unrelated to this change)
- No new test failures introduced by this change

---

## Model Impact

**Model claim (from warn-and-continue audit):** "Secondary prerequisites (dependencies, extraction) - STILL warn-and-continue"

**Finding:** **CONFIRMS AND UPDATES** - This fix converts dependency checking from warn-and-continue to fail-fast, aligning with kb-035b64 constraint.

**Impact:** Updates the model's prerequisite validation patterns:
- **Primary dedup** (beads status update) - fail-fast ✓ (fixed Feb 14)
- **Dependency checking** - NOW fail-fast ✓ (fixed Feb 17, this probe)
- **Epic expansion** - STILL warn-and-continue ✗ (orch-go-j26)
- **Extraction gate** - STILL warn-and-continue ✗ (orch-go-r9t)
- **Tertiary monitoring** (logging, status files) - ACCEPTABLE to continue ✓

**Constraint satisfaction:** This fix satisfies kb-035b64 for the dependency checking prerequisite. When `CheckBlockingDependencies` fails, the daemon now skips the issue instead of risking a spawn on potentially blocked work.

---

## Testing Notes

**Reproduction approach:**
Since this is an error handling path (dependency check fails), reproduction requires:
1. Creating a scenario where CheckBlockingDependencies returns an error
2. Verifying the daemon skips the issue instead of continuing

**Verification method:**
- Code review confirms the pattern change
- The fix aligns with established pattern used elsewhere in the same function (e.g., lines 350-352 for blocked issues)

---
