# Probe: Double-Gate — skip-phase-complete Propagation to bd close

**Date:** 2026-02-24
**Status:** Complete
**Model:** Completion Verification Architecture
**Claim Tested:** Targeted bypasses (`--skip-{gate} "reason"`) remain as an escape hatch for edge cases

## Question

Does `--skip-phase-complete` in `orch complete` successfully bypass the Phase: Complete check end-to-end, including bd's own internal check?

## What I Tested

1. Traced the `orch complete` flow from CLI flag → `skipConfig.PhaseComplete` → `verify.CloseIssue`
2. Examined `bd close --help` — found `bd close` has its own Phase: Complete check, bypassed only with `--force`
3. Verified that `verify.CloseIssue()` calls `bd close` without `--force`, creating a double-gate
4. Implemented fix: when `skipConfig.PhaseComplete || completeForce`, use `verify.ForceCloseIssue()` which calls `bd close --force`
5. Ran tests: `TestSkipPhaseCompleteTriggersForceClose` + all existing SkipConfig tests pass

### Code paths examined:
- `cmd/orch/complete_cmd.go:1081-1094` (close issue call site)
- `pkg/verify/beads_api.go:189-206` (CloseIssue → FallbackClose)
- `pkg/beads/client.go:993-1009` (FallbackClose)
- `bd close --help` (confirms --force flag bypasses Phase: Complete checks)

## What I Observed

**Before fix:** `orch complete --skip-phase-complete --skip-reason "..."` would:
1. ✅ Bypass orch's verification gate (phase_complete gate filtered out)
2. ❌ Fail at `verify.CloseIssue()` because `bd close` has its own Phase: Complete check
3. Error: "failed to close issue: bd close failed: ..."

**After fix:** Same command now:
1. ✅ Bypass orch's verification gate
2. ✅ Detect `skipConfig.PhaseComplete == true` → call `verify.ForceCloseIssue()`
3. ✅ `ForceCloseIssue()` calls `bd close --force` which bypasses bd's Phase: Complete check

## Model Impact

**Extends** the "Targeted bypasses" invariant:

The model states targeted bypasses are an escape hatch, but doesn't document that bypass must propagate through the full pipeline to `bd close`. The completion pipeline has two independent Phase: Complete checks:
1. **orch verification gates** — `verify.VerifyCompletionFull()` checks `phase_complete` gate
2. **bd close** — `bd close` independently checks for Phase: Complete before allowing close

The `--skip-phase-complete` flag correctly bypasses gate (1) but previously had no effect on gate (2). This is a **pipeline leak** where a skip at one layer doesn't propagate to downstream layers.

**New invariant suggestion:** Skip flags that bypass verification gates MUST propagate to all downstream systems that independently enforce the same check. Otherwise the skip is ineffective.

## Changes Made

| File | Change |
|------|--------|
| `cmd/orch/complete_cmd.go` | Route to `ForceCloseIssue` when `skipConfig.PhaseComplete \|\| completeForce` |
| `pkg/verify/beads_api.go` | Added `ForceCloseIssue()` function |
| `pkg/beads/client.go` | Added `FallbackForceClose()` function |
| `cmd/orch/complete_test.go` | Added `TestSkipPhaseCompleteTriggersForceClose` |
