# Investigation: orch complete Session Expiration Handling

**Date:** 2026-02-05  
**Status:** Complete  
**Investigator:** feature-impl agent

## Goal

Fix orch complete to handle agents whose sessions expired before reporting Phase: Complete.

## Problem Statement

When OpenCode sessions go idle/expire, agents commit their work but don't report Phase: Complete. Then:

1. `orch complete` refuses to close (requires Phase: Complete comment)
2. Even with `--skip-phase-complete`, `bd close` also independently requires Phase: Complete
3. Requires manual `bd comment` + retry

## Findings

### Finding 1: bd close has independent Phase: Complete gate

**Evidence:**

```bash
$ bd close --help
  -f, --force            Force close (bypasses pinned and Phase: Complete checks)
```

**Source:** CLI help output  
**Significance:** The `bd close` command has its own Phase: Complete gate that is independent of `orch complete`'s gate. This means `--skip-phase-complete` in `orch complete` only bypasses the orch gate, not the bd gate.

### Finding 2: CloseIssue calls bd close without --force

**Evidence:**

```go
// pkg/beads/client.go:980-991
func FallbackClose(beadsID, reason string) error {
	args := []string{"close", beadsID, "--reason", reason}
	// ... no --force flag added
	cmd := exec.Command(getBdPath(), args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd close failed: %w: %s", err, string(output))
	}
	return nil
}
```

**Source:** pkg/beads/client.go:980-991  
**Significance:** The current implementation never passes `--force` to `bd close`, so even when `--skip-phase-complete` is set in `orch complete`, the bd close call will still fail if Phase: Complete wasn't reported.

### Finding 3: skipConfig.PhaseComplete is already checked

**Evidence:**

```go
// cmd/orch/complete_cmd.go:225-226
if c.PhaseComplete {
	gates = append(gates, verify.GatePhaseComplete)
}
```

**Source:** cmd/orch/complete_cmd.go:225-226  
**Significance:** The skip config already tracks whether `--skip-phase-complete` was set. We just need to use this information when calling CloseIssue.

### Finding 4: CloseIssue is called at line 990

**Evidence:**

```go
// cmd/orch/complete_cmd.go:990
if err := verify.CloseIssue(beadsID, reason); err != nil {
	return fmt.Errorf("failed to close issue: %w", err)
}
```

**Source:** cmd/orch/complete_cmd.go:990  
**Significance:** This is where we need to pass the force flag based on skipConfig.PhaseComplete.

## Synthesis

The bug exists because:

1. `orch complete --skip-phase-complete` bypasses the orch verification gate
2. BUT it still calls `verify.CloseIssue()` which calls `bd close` without `--force`
3. `bd close` has its own independent Phase: Complete gate
4. So the skip flag only bypasses one of two gates

The fix requires:

1. Add a `force` parameter to `CloseIssue()` (or create `CloseIssueForce()`)
2. When `force=true`, pass `--force` to the `bd close` command
3. In `complete_cmd.go`, pass `force=skipConfig.PhaseComplete` when calling CloseIssue

## Open Questions

1. Should we also implement auto-completion signal detection (commits exist + session idle)?
   - The spawn context mentions this as a "better" approach
   - This would auto-add the Phase: Complete comment before closing
   - More complex but provides better tracking

2. Should we add a new function `CloseIssueForce()` or add a parameter to existing `CloseIssue()`?
   - Adding parameter is backward compatible if we use default=false
   - New function is clearer but requires updating call sites

## Implementation Summary

### Implemented (Phase 1)

✅ Added `FallbackCloseForce(id, reason, force bool)` to pkg/beads/client.go
✅ Made `FallbackClose()` call `FallbackCloseForce()` with force=false (backward compatible)
✅ Added `CloseIssueForce(beadsID, reason, force bool)` to pkg/verify/beads_api.go
✅ Made `CloseIssue()` call `CloseIssueForce()` with force=false (backward compatible)
✅ Updated complete_cmd.go line 998 to pass `skipConfig.PhaseComplete` as force parameter
✅ Added test for FallbackCloseForce in pkg/beads/client_test.go

### Result

When `orch complete --skip-phase-complete` is used, the flag now:

1. Bypasses orch's Phase: Complete verification gate (existing behavior)
2. Passes `--force` to `bd close` to bypass bd's Phase: Complete gate (new behavior)

This fixes the original issue where expired agent sessions couldn't be closed even with `--skip-phase-complete`.

### Not Implemented (Future Enhancement)

❌ Auto-completion detection (commits exist + session idle → auto-add Phase: Complete)

- Marked as "Better" approach in spawn context, but not required for minimal fix
- Would provide better audit trail (Phase: Complete comment always present)
- Deferred to future enhancement

## Verification

The fix has been:

- ✅ Implemented with TDD (test first, then implementation)
- ✅ Tested (TestFallbackCloseForce passes)
- ✅ Committed (commit d7c583da)

Original reproduction:

> When opencode sessions go idle/expire, agents don't report Phase: Complete. Then orch complete refuses, and even --skip-phase-complete doesn't fully work because bd close also requires it.

Expected behavior after fix:

```bash
orch complete <beads-id> --skip-phase-complete --skip-reason "Session expired before completion"
# Should now succeed - both orch and bd gates bypassed
```
