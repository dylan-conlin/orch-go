<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The concurrency fix (commit d0eae36) correctly excludes Phase: Complete agents from the active count when checking spawn limits.

**Evidence:** Tested with 8 sessions (5 completed, 3 running). With limit=3, spawn blocked correctly. With limit=4, spawn allowed. `orch status` shows "Active: 8 (running: 3, idle: 5)" confirming correct differentiation.

**Knowledge:** The fix uses `verify.IsPhaseComplete(beadsID)` which shells out to `bd comments --json` - this is the correct approach since beads is the source of truth for agent status.

**Next:** No further action needed. The fix is complete and verified working. Consider adding integration test documentation to `TestCheckConcurrencyLimitUsesOpenCodeAPI` to explain how to manually verify.

**Confidence:** High (90%) - Tested with live system, observed correct behavior

---

# Investigation: Test Concurrency Fix

**Question:** Does the concurrency fix (commit d0eae36) correctly exclude Phase: Complete agents from the active count?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Fix correctly filters completed agents

**Evidence:** 
- Tested with 8 active sessions within the 30-minute staleness threshold
- 5 sessions had reported "Phase: Complete" via beads comments
- 3 sessions were still running (no Phase: Complete)
- With `ORCH_MAX_AGENTS=3`, spawn was correctly blocked
- With `ORCH_MAX_AGENTS=4`, spawn was allowed
- `orch status` output: "Active: 8 (running: 3, idle: 5)"

**Source:** 
- `cmd/orch/main.go:890-945` - `checkConcurrencyLimit()` function
- Live testing with OpenCode sessions

**Significance:** The fix correctly uses the Phase: Complete status from beads to differentiate active vs idle agents, preventing false concurrency limit triggers.

---

### Finding 2: Implementation uses beads as source of truth

**Evidence:**
The filtering logic in `checkConcurrencyLimit()` at line 933-936:
```go
// Exclude completed agents (Phase: Complete) - they're idle and not consuming resources
if isComplete, _ := verify.IsPhaseComplete(beadsID); isComplete {
    continue // completed agent, don't count against limit
}
```

`verify.IsPhaseComplete()` calls `GetPhaseStatus()` which calls `GetComments()` which shells out to `bd comments --json <beadsID>`.

**Source:** 
- `pkg/verify/check.go:94-106` - `IsPhaseComplete()` function
- `pkg/verify/check.go:84-92` - `GetPhaseStatus()` function

**Significance:** Using beads as the source of truth for completion status is correct because:
1. Agents explicitly report "Phase: Complete" via beads comments
2. This is the same data used by `orch complete` for verification
3. Avoids duplicating state (no need for OpenCode-specific completion detection)

---

### Finding 3: All existing tests pass

**Evidence:**
```
$ go test -race ./...
ok  	github.com/dylan-conlin/orch-go/cmd/orch	4.329s
ok  	github.com/dylan-conlin/orch-go/pkg/verify	1.176s
[all other packages pass]
```

The existing test `TestCheckConcurrencyLimitUsesOpenCodeAPI` is a documentation test explaining the behavior but doesn't perform integration testing (would require a running OpenCode server).

**Source:** `make test` with `-race` flag

**Significance:** No regressions introduced. The fix is safe and integrates cleanly with existing code.

---

## Synthesis

**Key Insights:**

1. **Correct filtering logic** - The fix applies three filters in sequence: staleness (< 30 min), orch-spawned (has beadsID), and not completed (no Phase: Complete). This correctly identifies only truly active agents.

2. **Beads as source of truth** - Using beads comments for completion status is the right design choice because it's the canonical record of agent lifecycle.

3. **Observable in status command** - The `orch status` command clearly shows the distinction with "Active: N (running: X, idle: Y)" format.

**Answer to Investigation Question:**

Yes, the concurrency fix correctly excludes Phase: Complete agents from the active count. Testing confirmed:
- 8 sessions → 3 counted as active (5 were Phase: Complete)
- Spawn blocked at limit of 3 (correct)
- Spawn allowed at limit of 4 (correct)
- Status command shows accurate running/idle breakdown

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Strong evidence from live testing with real sessions, observed correct behavior in multiple scenarios.

**What's certain:**

- ✅ Fix correctly filters out Phase: Complete agents
- ✅ Staleness threshold (30 min) working correctly
- ✅ BeadsID extraction from session titles working
- ✅ All tests pass including race detector

**What's uncertain:**

- ⚠️ Edge case: What if `bd comments` fails silently? (Currently ignored, agent is counted)
- ⚠️ No automated integration test (relies on manual testing)

**What would increase confidence to Very High (95%+):**

- Add mock-based unit test for the filtering loop
- Add documentation for manual integration testing procedure

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

---

## Discovered Work

No new issues discovered. The fix is complete and working correctly.

**Leave it Better:** Straightforward investigation confirming fix behavior. No new knowledge to externalize - the fix is working as designed.

---

## References

**Files Examined:**
- `cmd/orch/main.go:890-945` - `checkConcurrencyLimit()` function
- `pkg/verify/check.go:84-106` - Phase status functions
- `cmd/orch/main_test.go:107-118` - Documentation test

**Commands Run:**
```bash
# Check recent commits
git show d0eae36 -p

# Run all tests with race detector
go test -race ./...

# Check live session phases
for id in orch-go-6qsq orch-go-ctvw orch-go-i1cm orch-go-3t8p orch-go-wa8z orch-go-w0bm orch-go-foko; do
  bd comments $id --json | jq -r '[.[] | select(.text | test("Phase:"))] | last | .text'
done

# Test concurrency limit
ORCH_MAX_AGENTS=3 orch spawn investigation "test" --no-track
ORCH_MAX_AGENTS=4 orch spawn investigation "test" --no-track
```

---

## Investigation History

**2025-12-24 10:04:** Investigation started
- Initial question: Does the concurrency fix correctly exclude Phase: Complete agents?
- Context: Spawned as `orch spawn investigation "test concurrency fix"`

**2025-12-24 10:10:** Analyzed fix implementation
- Found filtering logic in `checkConcurrencyLimit()`
- Identified use of `verify.IsPhaseComplete()`

**2025-12-24 10:15:** Live testing completed
- Verified 8 sessions, 5 completed, 3 running
- Confirmed correct behavior at limit boundaries

**2025-12-24 10:20:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Concurrency fix works correctly, no issues found
