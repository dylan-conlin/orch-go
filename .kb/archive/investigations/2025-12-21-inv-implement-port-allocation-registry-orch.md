<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented a complete port allocation registry at pkg/port/ with CLI commands.

**Evidence:** All 17 tests pass; manual testing of orch port allocate/list/release works correctly.

**Knowledge:** Port ranges (vite 5173-5199, api 3333-3399) are sufficient for typical multi-project setups; YAML storage at ~/.orch/ports.yaml follows existing patterns.

**Next:** Integration with orch init and tmuxinator generation (future scope).

**Confidence:** High (95%) - Implementation complete and tested.

---

# Investigation: Port Allocation Registry Implementation

**Question:** How to implement port allocation registry to prevent conflicts across projects?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Existing package patterns provide good templates

**Evidence:** pkg/focus/ uses JSON storage with load/save pattern; pkg/account/ uses YAML storage with similar structure. Both provide good patterns for registry implementation.

**Source:** pkg/focus/focus.go:101-144, pkg/account/account.go:84-125

**Significance:** Following existing patterns ensures consistency and maintainability.

---

### Finding 2: Port ranges are well-suited for typical project needs

**Evidence:** Vite range (5173-5199) provides 27 ports; API range (3333-3399) provides 67 ports. This covers most multi-project scenarios while leaving clear separation.

**Source:** SPAWN_CONTEXT.md task description

**Significance:** Range sizes are adequate without being wasteful; clear purpose separation prevents confusion.

---

### Finding 3: Idempotent allocation is essential for usability

**Evidence:** Calling allocate for the same project/service/purpose returns the existing port rather than allocating a new one.

**Source:** pkg/port/port_test.go:TestAllocateSameServiceReturnsExisting

**Significance:** Enables safe re-runs of tmuxinator or init scripts without port conflicts.

---

## Synthesis

**Key Insights:**

1. **YAML storage matches config file expectations** - Users expect ~/.orch/ports.yaml to be human-readable and editable.

2. **Idempotent operations prevent foot-guns** - Running scripts multiple times shouldn't cause port exhaustion.

3. **CLI follows established patterns** - orch port command group mirrors account/focus command structures.

**Answer to Investigation Question:**

Port allocation is implemented via a YAML-backed registry that tracks project/service/port allocations. The registry allocates from predefined purpose-based ranges (vite, api), prevents cross-project conflicts through centralized tracking, and provides CLI commands for allocation, listing, and release.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

All core requirements implemented and tested. Manual smoke testing confirms expected behavior.

**What's certain:**

- ✅ Port allocation from ranges works correctly (17 tests pass)
- ✅ YAML persistence works (TestPersistence passes)
- ✅ CLI commands work (manual testing complete)

**What's uncertain:**

- ⚠️ Integration with orch init not yet implemented (out of scope)
- ⚠️ Concurrent write safety not tested under high load

**What would increase confidence to 100%:**

- Integration testing with actual tmuxinator generation
- Load testing of concurrent allocations

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**YAML-backed registry with CLI commands** - Complete implementation delivered.

**Why this approach:**
- Matches existing orch-go patterns (accounts.yaml, focus.json)
- Human-readable format for debugging
- Simple CLI for manual management

**Trade-offs accepted:**
- No file locking (acceptable for single-user tool)
- No concurrent write protection beyond auto-save

**Implementation sequence:**
1. ✅ pkg/port/port.go - Core registry implementation
2. ✅ pkg/port/port_test.go - Comprehensive tests
3. ✅ cmd/orch/main.go - CLI commands (port allocate/list/release)

---

## References

**Files Created:**
- pkg/port/port.go - Core registry implementation
- pkg/port/port_test.go - Test suite (17 tests)

**Files Modified:**
- cmd/orch/main.go - Added port command group

**Commands Run:**
```bash
# Test port package
go test ./pkg/port/... -v

# Test full build
go build ./cmd/orch/...

# Manual testing
orch port allocate testproject web vite
orch port list
orch port release testproject web
```

---

## Investigation History

**2025-12-21 12:20:** Investigation started
- Initial question: How to implement port allocation registry?
- Context: Prevent port conflicts across projects

**2025-12-21 12:25:** Implementation complete
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Port allocation registry implemented with full test coverage
