<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Parent-child dependencies in orch-go daemon were incorrectly blocking epic children, fixed by changing logic to NEVER block on parent-child dependency type.

**Evidence:** Verified: `orch daemon preview` no longer shows "blocked by dependencies: orch-go-lv3yx (open)" for children, unit tests pass (12/12 blocking tests).

**Knowledge:** Epic children must be independently spawnable because epics close when children complete (circular if children wait for parent). The `dependency_type: parent-child` is for hierarchy, not sequence.

**Next:** Mark constraint kb-8294e7 as obsolete since the bug is now fixed.

**Promote to Decision:** recommend-no (tactical bug fix, behavior now matches expected semantics)

---

# Investigation: Fix Epic Children Blocking Orch

**Question:** Why does `orch daemon preview` show epic children as blocked by their parent epic when they should be independently spawnable?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Bug location identified in GetBlockingDependencies()

**Evidence:** The function `GetBlockingDependencies()` in `pkg/beads/types.go:187-223` had logic that blocked parent-child dependencies when parent status was "open":
```go
case "parent-child":
    isBlocking = dep.Status == "open"  // BUG: This blocked children when parent is open
```

**Source:** `pkg/beads/types.go:204-208`

**Significance:** This is the root cause. The code intended to unblock children when parent transitions to "in_progress", but the design was wrong - epic children should NEVER be blocked by the parent-child relationship because the epic closes when children complete (circular dependency).

---

### Finding 2: Verified reproduction from beads comment

**Evidence:** Running `orch daemon preview` before fix showed:
```
orch-go-lv3yx.7: blocked by dependencies: orch-go-lv3yx (open)
orch-go-lv3yx.6: blocked by dependencies: orch-go-lv3yx (open)
orch-go-lv3yx.5: blocked by dependencies: orch-go-lv3yx (open)
orch-go-lv3yx.4: blocked by dependencies: orch-go-lv3yx (open)
```

**Source:** `orch daemon preview` command output, beads comment #246 on orch-go-tuofe

**Significance:** Confirms the bug was in orch-go daemon's `GetBlockingDependencies()` call, not in beads itself (beads `bd ready` correctly shows children as ready).

---

### Finding 3: Test isolation issue required additional fix

**Evidence:** The test `TestPreview_EpicWithTriageReadyShowsHelpfulMessage` was not properly isolated - it mocked `listIssuesFunc` but `expandTriageReadyEpics()` called the real `ListEpicChildren()` function.

**Source:** `pkg/daemon/daemon_test.go:2290-2322`

**Significance:** Added `listEpicChildrenFunc` to Daemon struct for testability. Without this, the test started failing after the fix because real children were being returned and were now spawnable (not blocked).

---

## Synthesis

**Key Insights:**

1. **Dependency semantics matter** - `parent-child` dependency type represents hierarchy (child-of), not sequence (depends-on). Blocking semantics should only apply to `blocks` dependency type.

2. **Epic lifecycle has circular dependency** - Epic closes when children complete, so children can't wait for parent to close. The only valid states for children are: blocked by something else, or ready to work.

3. **Test isolation critical for correctness** - The failing test exposed a pre-existing test isolation gap. Making `ListEpicChildren` mockable improves test reliability.

**Answer to Investigation Question:**

Epic children were blocked because `GetBlockingDependencies()` treated `parent-child` dependencies as blocking when parent status was "open". The fix changes this to NEVER block on parent-child dependencies. This matches the expected behavior: children should be independently spawnable while the parent epic remains open.

---

## Structured Uncertainty

**What's tested:**

- ✅ Unit tests: 12/12 blocking dependency tests pass (`go test ./pkg/beads/... -run Block`)
- ✅ Daemon tests: All daemon tests pass including Preview tests (`go test ./pkg/daemon/...`)
- ✅ Smoke test: `orch daemon preview` no longer shows children as blocked by open parent

**What's untested:**

- ⚠️ Edge case where epic has both `parent-child` AND `blocks` dependencies (should still respect `blocks`)

**What would change this:**

- If there's a legitimate use case where children SHOULD be blocked by open parent (none known)
- If the `blocks` dependency type behavior is also incorrect (unlikely - tests pass)

---

## References

**Files Modified:**
- `pkg/beads/types.go:187-223` - Changed parent-child blocking logic
- `pkg/beads/client_test.go:1337-1365` - Updated test expectations
- `pkg/daemon/daemon.go:142,156,344` - Added listEpicChildrenFunc for testability
- `pkg/daemon/daemon_test.go:2296-2300` - Fixed test isolation

**Commands Run:**
```bash
# Reproduce the bug
orch daemon preview | grep "lv3yx"

# Run tests
go test ./pkg/beads/... -v -run "Block"
go test ./pkg/daemon/...

# Verify fix
go build ./cmd/orch/ && ./orch daemon preview | grep "lv3yx"
```

**Related Artifacts:**
- **Issue:** orch-go-tuofe - Bug report with reproduction steps
- **Constraint:** kb-8294e7 - Should be marked obsolete (bug is fixed)

---

## Investigation History

**2026-01-08 10:19:** Investigation started
- Initial question: Why are epic children blocked by parent in orch daemon?
- Context: Previous investigation looked in wrong repo (beads vs orch-go)

**2026-01-08 10:25:** Root cause identified
- Bug is in `GetBlockingDependencies()` treating parent-child as blocking when parent is open

**2026-01-08 10:30:** Fix implemented and verified
- Changed parent-child to NEVER block
- All tests pass
- Smoke test confirms fix
