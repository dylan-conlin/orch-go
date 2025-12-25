<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Added Phase: Complete check to spawn preflight to prevent duplicate spawns when work is done but not closed.

**Evidence:** Build passes, all tests pass. The check uses existing `verify.IsPhaseComplete()` which is already tested.

**Knowledge:** The "work done but not closed" gap in spawn preflight was a guardrail gap - spawn only checked for closed issues or active sessions, not for completed work awaiting `orch complete`.

**Next:** Close - implementation complete.

**Confidence:** High (90%) - Uses well-tested existing function; small, focused change.

---

# Investigation: Pre-Spawn Phase: Complete Check

**Question:** How to prevent duplicate spawns when an agent has reported Phase: Complete but the orchestrator hasn't run `orch complete` yet?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Spawn Preflight Has Incomplete Guardrails

**Evidence:** The spawn preflight code at `cmd/orch/main.go:1095-1114` only checked for:
1. Closed issues (blocks spawn)
2. In-progress issues with active OpenCode sessions (blocks spawn)

But it did NOT check for issues where work was completed (Phase: Complete in beads comments) but the orchestrator hadn't yet run `orch complete`.

**Source:** `cmd/orch/main.go:1095-1114` (before fix)

**Significance:** This gap allowed duplicate spawns for the same issue - agent finishes, reports Phase: Complete, but before orchestrator runs `orch complete`, another spawn attempt could succeed.

---

### Finding 2: Existing Infrastructure for Phase Checking

**Evidence:** The `verify` package already has `IsPhaseComplete(beadsID)` function that:
1. Fetches comments for a beads issue (via RPC or CLI fallback)
2. Parses comments for "Phase: <phase>" pattern
3. Returns true if latest phase is "Complete" (case-insensitive)

**Source:** `pkg/verify/check.go:91-103`, tested in `pkg/verify/check_test.go:149-186`

**Significance:** No new infrastructure needed - the fix is a simple integration of existing, well-tested functionality.

---

### Finding 3: Fix Location and Logic

**Evidence:** The fix was added inside the `in_progress` status check, after verifying no active session exists but before allowing respawn:

```go
// No active session - check if Phase: Complete was reported
// If so, orchestrator needs to run 'orch complete' before respawning
if complete, err := verify.IsPhaseComplete(beadsID); err == nil && complete {
    return fmt.Errorf("issue %s has Phase: Complete but is not closed. Run 'orch complete %s' first", beadsID, beadsID)
}
```

**Source:** `cmd/orch/main.go:1110-1113` (after fix)

**Significance:** The error message is actionable - it tells the orchestrator exactly what to do (`orch complete`).

---

## Synthesis

**Key Insights:**

1. **Guardrail completeness matters** - The spawn preflight had multiple checks but missed the "work done, not closed" state.

2. **Leverage existing infrastructure** - The `verify.IsPhaseComplete()` function was already available and tested, making the fix minimal.

3. **Actionable error messages** - The error tells the user exactly what to do next.

**Answer to Investigation Question:**

Add a check for Phase: Complete in beads comments during spawn preflight, specifically when the issue is in_progress but has no active session. If Phase: Complete is found, block spawn with an actionable error message telling the orchestrator to run `orch complete` first. This closes the "work done but not closed" gap that was allowing duplicate spawns.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

The fix uses existing, well-tested infrastructure (`verify.IsPhaseComplete()`). The change is small and focused. All tests pass.

**What's certain:**

- ✅ The fix compiles and all existing tests pass
- ✅ The error message is actionable
- ✅ The check uses existing, tested infrastructure

**What's uncertain:**

- ⚠️ End-to-end testing requires a beads daemon and would be complex to set up in tests

**What would increase confidence to Very High:**

- Integration test with mocked beads comments

---

## Implementation Recommendations

**Purpose:** N/A - already implemented.

### Recommended Approach ⭐

**Check Phase: Complete in spawn preflight** - Already implemented.

The fix adds 4 lines of code that check for Phase: Complete when:
1. Issue status is in_progress
2. No active OpenCode session exists for this issue

If Phase: Complete is found, spawn is blocked with actionable error message.

---

## References

**Files Modified:**
- `cmd/orch/main.go:1110-1113` - Added Phase: Complete check

**Commands Run:**
```bash
# Build verification
make build  # PASS

# Test verification
go test ./... -short  # All pass
```

**Related Artifacts:**
- **Package:** `pkg/verify/check.go` - Contains IsPhaseComplete() function used by fix

---

## Investigation History

**2025-12-25:** Investigation started and completed
- Fixed spawn preflight to check for Phase: Complete before allowing respawn
- Build and tests pass
