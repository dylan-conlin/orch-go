<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `GetBlockingDependencies()` treated all dependency types the same, blocking when status != "closed" - but parent-child dependencies should only block when parent is "open", not when "in_progress".

**Evidence:** Code at `pkg/beads/types.go:195` showed `if dep.Status != "closed"` without checking `dependency_type`. Tests confirmed fix: all 11 test cases pass including new parent-child scenarios.

**Knowledge:** Parent-child relationships have different semantics than "blocks" dependencies - when a parent epic transitions to in_progress, children should become unblocked so work can proceed.

**Next:** Smoke-test by creating epic with child tasks in price-watch, verify daemon picks up children when parent is in_progress.

---

# Investigation: Daemon Blocks Child Tasks When Parent In-Progress

**Question:** Why does daemon block child tasks when parent epic is `in_progress`, when they should be unblocked?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Agent og-debug-daemon-blocks-child-06jan-1a43
**Phase:** Complete
**Next Step:** None - fix implemented and tested
**Status:** Complete

---

## Findings

### Finding 1: GetBlockingDependencies treats all dependencies uniformly

**Evidence:** The function at `pkg/beads/types.go:186-204` had this logic:
```go
for _, dep := range deps {
    // A dependency blocks if it's not closed
    if dep.Status != "closed" {
        blocking = append(blocking, ...)
    }
}
```

**Source:** `pkg/beads/types.go:193-201`

**Significance:** This logic treats `in_progress` as blocking for ALL dependency types, including parent-child. But parent-child semantics should be: parent `open` = blocked, parent `in_progress` = unblocked.

---

### Finding 2: Dependency type field exists but wasn't being used

**Evidence:** The `Dependency` struct at `pkg/beads/types.go:156-161` has:
```go
type Dependency struct {
    ID             string `json:"id"`
    Title          string `json:"title"`
    Status         string `json:"status"`
    DependencyType string `json:"dependency_type"` // e.g., "blocks", "parent-child"
}
```

**Source:** `pkg/beads/types.go:160`

**Significance:** The data structure supports distinguishing parent-child from blocks dependencies, but the blocking logic wasn't utilizing this field.

---

### Finding 3: Tests only covered "blocks" dependency type

**Evidence:** All existing test cases in `TestGetBlockingDependencies` used `dependency_type: "blocks"`. No test cases for `parent-child` type existed.

**Source:** `pkg/beads/client_test.go:1296-1355`

**Significance:** The bug could have been caught earlier with comprehensive test coverage of dependency types.

---

## Synthesis

**Key Insights:**

1. **Dependency type semantics differ** - "blocks" means must be closed before proceeding; "parent-child" means work can start once parent is in_progress (epic is active).

2. **The fix is straightforward** - Check `DependencyType` and apply appropriate blocking logic: parent-child only blocks when status is "open", all others block when status is not "closed".

3. **Backward compatible** - The fix maintains existing behavior for "blocks" dependencies while adding correct semantics for "parent-child".

**Answer to Investigation Question:**

The daemon blocked child tasks because `GetBlockingDependencies()` treated `in_progress` as a blocking status for all dependency types. For parent-child dependencies, the correct behavior is: `open` parent = blocked (epic not started), `in_progress` parent = unblocked (epic active, work should proceed), `closed` parent = unblocked.

---

## Structured Uncertainty

**What's tested:**

- ✅ Parent-child open parent blocks child (test case passes)
- ✅ Parent-child in_progress parent does NOT block child (test case passes)
- ✅ Parent-child closed parent does NOT block child (test case passes)
- ✅ Mixed blocks + parent-child dependencies handled correctly (test cases pass)
- ✅ All existing "blocks" behavior unchanged (all original tests pass)

**What's untested:**

- ⚠️ End-to-end with real beads daemon (smoke test pending)
- ⚠️ Integration with daemon's checkRejectionReason flow

**What would change this:**

- Finding would be wrong if beads daemon returns different JSON structure than expected
- Finding would be wrong if there are other dependency types with different semantics

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Differentiate blocking logic by dependency type** - Check `DependencyType` field and apply appropriate blocking semantics.

**Why this approach:**
- Directly addresses root cause (uniform treatment of all dependency types)
- Maintains backward compatibility for "blocks" dependencies
- Clean, readable logic with explicit cases

**Trade-offs accepted:**
- Assumes only two dependency types exist (blocks, parent-child)
- Default case falls back to "blocks" behavior for safety

**Implementation sequence:**
1. Update GetBlockingDependencies with switch on DependencyType
2. Add comprehensive tests for parent-child scenarios
3. Verify no regression in existing tests

### Implementation Details

**What to implement first:**
- [x] Update GetBlockingDependencies in types.go
- [x] Add test cases for parent-child dependency behavior

**Things to watch out for:**
- ⚠️ Make sure default case (unknown dependency types) falls back to "blocks" behavior (safer)
- ⚠️ The fix is in pkg/beads, not pkg/daemon - daemon calls beads.CheckBlockingDependencies

**Success criteria:**
- ✅ All existing tests pass (11 test cases)
- ✅ New parent-child tests pass (5 new test cases)
- ✅ Smoke test: create epic with child tasks, verify daemon picks up children when epic is in_progress

---

## References

**Files Examined:**
- `pkg/beads/types.go` - GetBlockingDependencies function (root cause)
- `pkg/beads/client_test.go` - TestGetBlockingDependencies (test coverage)
- `pkg/daemon/daemon.go` - checkRejectionReason calling beads.CheckBlockingDependencies

**Commands Run:**
```bash
# Run tests to verify fix
go test ./pkg/beads/... -run TestGetBlockingDependencies -v
# Result: 11 tests passed (6 original + 5 new parent-child tests)

# Run all tests to verify no regression
go test ./... 2>&1 | tail -20
# Result: Only pre-existing tmux test failure (unrelated to this change)
```

---

## Investigation History

**2026-01-06 10:15:** Investigation started
- Initial question: Why does daemon block child tasks when parent is in_progress?
- Context: Reported bug in price-watch project with epic pw-u8th

**2026-01-06 10:25:** Root cause identified
- GetBlockingDependencies treats all deps the same, blocking when not closed
- Parent-child should only block when parent is "open"

**2026-01-06 10:35:** Fix implemented and tested
- Updated types.go with switch on DependencyType
- Added 5 new test cases for parent-child behavior
- All 11 tests pass

**2026-01-06 10:40:** Investigation completed
- Status: Complete
- Key outcome: Fixed parent-child dependency blocking logic in GetBlockingDependencies
