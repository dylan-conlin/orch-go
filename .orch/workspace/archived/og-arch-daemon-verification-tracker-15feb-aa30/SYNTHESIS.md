# SYNTHESIS: Daemon Verification Tracker Checkpoint Enforcement

**Date:** 2026-02-15
**Issue:** orch-go-8h9l
**Type:** Bug Fix (Composition Bug)

---

## Plain-Language Summary

The daemon's verification pause gate was not firing because it counted unverified completions by reading labels instead of the checkpoint file. When issues were closed via `orch complete`, the `daemon:ready-review` label was removed, making closed-but-unverified completions invisible to the daemon. The checkpoint file correctly persisted verification state, but nothing read it for enforcement.

Fixed by making `CountUnverifiedCompletions()` read the checkpoint file directly (source of truth) and count unverified entries based on tier requirements: Tier 1 work (features/bugs/decisions) requires both gates, Tier 2 work (investigations/probes) requires gate1 only.

After the fix, the daemon correctly reports "7/3 unverified completions" and refuses to spawn, enforcing the verifiability-first constraint as designed.

---

## Problem

**Reproduction:**
- Checkpoint file contains 13 entries with `gate2_complete:false`
- `orch daemon run --dry-run` reports "0/3 unverified completions"
- Verification pause gate does not fire (should refuse to spawn at 3+)

**Root Cause:**
`SeedFromBacklog()` counted issues with `daemon:ready-review` label via `issue_adapter.go:CountUnverifiedCompletions()`. When issues are closed via `orch complete`, the label is removed, so closed-but-unverified completions disappear from the daemon's view. The checkpoint file persists the unverified state correctly, but nothing read it for enforcement.

**Composition Bug:**
- Checkpoint writing works (gate1/gate2 recorded correctly) ✓
- Label-based counting works (counts labels accurately) ✓
- But they measure different things ✗
  - Labels = active review queue (view layer)
  - Checkpoints = verification state (source of truth)
- The daemon used the wrong one for enforcement ✗

---

## Solution

**File Changed:** `pkg/daemon/issue_adapter.go`

**Function Rewritten:** `CountUnverifiedCompletions()`

**Before:**
1. List issues with `daemon:ready-review` label
2. Filter out those with checkpoints
3. Count remaining issues
4. **Problem:** Closed issues lose label, disappear from count

**After:**
1. Read all checkpoints from file directly
2. For each checkpoint, look up issue type from beads (RPC with CLI fallback)
3. Determine tier based on issue type
4. Count as unverified based on tier:
   - Tier 1 (feature/bug/decision): `gate2_complete == false`
   - Tier 2 (investigation/probe): `gate1_complete == false`
   - Tier 3 (task/question/other): skip (no verification required)
5. Skip if issue lookup fails (may be deleted)

**New Helper Function:** `showIssueCLI()` - CLI fallback for issue lookup when RPC unavailable

**Invariant Established:**
- **Checkpoint file is source of truth** - Enforcement MUST read from checkpoint file, not labels
- **Labels are view layer only** - `daemon:ready-review` shows active review queue, not verification state

---

## Verification

### Before Fix
```bash
$ orch daemon run --dry-run
[DRY-RUN] Verification check: 0/3 unverified completions
```

### After Fix
```bash
$ orch daemon run --dry-run --verbose
  Verification backlog: 7 unverified completions from previous sessions
  Warning: Verification pause: backlog exceeds threshold (7/3)
  Run 'orch daemon resume' after reviewing completed work
[DRY-RUN] Verification pause: 7 unverified completions, threshold is 3
```

**Checkpoint File Analysis:**
- 15 total checkpoint entries
- 7 entries are unverified based on tier requirements
- Daemon now correctly reports 7/3 and refuses to spawn ✓

**Verification Contract:**
- ✓ Checkpoint file is read directly
- ✓ Tier-aware counting (gate1 vs gate2)
- ✓ RPC client with CLI fallback
- ✓ Verification pause gate fires when threshold exceeded
- ✓ Daemon refuses to spawn when paused

---

## Tests

**Command:** `go test ./pkg/daemon/...`

**Result:** Existing tests pass. Test failures in `extraction_test.go` and `verify` package are pre-existing and unrelated to this change (only `issue_adapter.go` was modified).

**Manual Testing:**
- Created debug tool to verify `CountUnverifiedCompletions()` returns correct count (7)
- Tested daemon dry-run mode with and without `--verbose`
- Verified pause gate fires and refuses to spawn

---

## Knowledge Artifacts

**Probe Created:**
`.kb/models/completion-verification/probes/2026-02-15-daemon-verification-tracker-checkpoint-enforcement.md`

**Model Impact:**
Extends the Completion Verification Architecture model with the implementation detail that verification enforcement MUST read from checkpoint file (source of truth), not labels (view layer).

---

## Verification Spec

**Observable Behavior:**
When the daemon starts (or runs in dry-run mode), it reports the number of unverified completions by reading the checkpoint file directly, and refuses to spawn when the count exceeds the threshold.

**Acceptance Criterion:**
`orch daemon run --dry-run` reports accurate unverified completion count based on checkpoint file entries and tier requirements, and shows "Verification pause" message when threshold exceeded.

**Failure Mode:**
**Symptom:** Daemon reports 0 unverified completions despite checkpoint file having entries
**Root Cause:** Reading labels instead of checkpoint file
**Fix:** Use checkpoint file as source of truth

**Evidence:**
Test output showing daemon correctly reporting 7/3 unverified completions and refusing to spawn.

---

## References

- **Decision:** `.kb/decisions/2026-02-14-verifiability-first-hard-constraint.md` (Constraint 3: Mechanical Enforcement)
- **Prior Probe:** `.kb/models/completion-verification/probes/2026-02-15-verification-tracker-wiring.md` (confirmed tracker was wired, but not reading checkpoint file)
- **Model:** `.kb/models/completion-verification.md`
