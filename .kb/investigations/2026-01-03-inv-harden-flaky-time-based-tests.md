<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created `internal/testutil` package with condition-based waiting helpers and fixed 12+ flaky time.Sleep calls across 6 test files.

**Evidence:** All tests pass consistently across 3 consecutive runs with no flakiness. Tests in opencode, daemon, capacity, and verify packages all use deterministic waiting patterns.

**Knowledge:** Flaky tests fall into 4 categories: async callback waits (use WaitFor), goroutine synchronization (use YieldForGoroutine), git timestamps (document with constant), file mtime (use Chtimes).

**Next:** Close issue - implementation complete with all tests passing and no regressions.

---

# Investigation: Harden Flaky Time Based Tests

**Question:** How can we eliminate arbitrary time.Sleep calls in tests that can cause flaky test failures?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** orch-go team
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Four Categories of Time-Based Flakiness

**Evidence:** Analyzed all time.Sleep calls in test files and categorized them:
1. **Async callback waits** (most common): Waiting for completion handlers called in goroutines
2. **Goroutine synchronization**: Ensuring goroutines are blocked on condition variables before proceeding
3. **Git timestamp granularity**: Git commits have second-precision timestamps, requiring ~1.1s sleep
4. **File mtime differences**: Tests checking file modification times needed time gaps

**Source:** 
- `pkg/opencode/monitor_test.go` - lines 118, 166, 229, 243, 298, 342, 383, 395
- `pkg/daemon/pool_test.go` - lines 201, 436
- `pkg/verify/git_diff_test.go` - lines 198, 278
- `pkg/verify/constraint_test.go` - lines 434, 480, 549, 566

**Significance:** Each category requires a different solution approach.

---

### Finding 2: Condition-Based Waiting is the Right Pattern

**Evidence:** Go's testing best practice is to poll conditions rather than arbitrary sleeps. Created `internal/testutil` package with:
- `WaitFor(t, condition, description)` - polls until true or fails
- `Eventually(condition, timeout)` - returns bool without failing
- `YieldForGoroutine()` - for sync.Cond synchronization
- `WaitForCount(t, counter, expected, description)` - for counting callbacks

**Source:** `internal/testutil/wait.go`

**Significance:** Centralizes the pattern, provides consistent API, and is reusable across all test files.

---

### Finding 3: Some Sleeps Are Inherently Necessary

**Evidence:** Two categories of sleeps cannot be eliminated:
1. **Git timestamp granularity** - Git commit timestamps have second precision, so we need `gitTimestampGranularity = 1100ms` constant
2. **File mtime manipulation** - Can be avoided by using `os.Chtimes()` to set explicit timestamps

**Source:** 
- `pkg/verify/git_diff_test.go` - documented with constant
- `pkg/verify/constraint_test.go` - replaced with Chtimes calls

**Significance:** Understanding which sleeps are necessary vs. arbitrary prevents over-engineering solutions.

---

## Synthesis

**Key Insights:**

1. **Async patterns need condition-based waiting** - Any test that checks a value set by a goroutine callback should use WaitFor, not sleep.

2. **Goroutine blocking requires yielding** - When testing condition variable wake-up, we need to ensure the goroutine is blocked before signaling. YieldForGoroutine provides this.

3. **Filesystem timestamps can be controlled** - Instead of sleeping to create time gaps, use os.Chtimes() for deterministic mtime values.

**Answer to Investigation Question:**

Eliminated flakiness by creating a testutil package with condition-based waiting helpers. Fixed 6 test files by replacing arbitrary time.Sleep calls with WaitFor/Eventually patterns. Tests now pass consistently across multiple runs.

---

## Structured Uncertainty

**What's tested:**

- ✅ All tests pass (verified: `go test ./...` runs clean)
- ✅ Tests are stable (verified: 3 consecutive runs with no failures)
- ✅ WaitFor correctly times out (verified: testutil tests with fake testing.TB)

**What's untested:**

- ⚠️ Long-term stability under CI load (not benchmarked under high parallelism)
- ⚠️ Behavior on very slow machines (YieldForGoroutine timing may need adjustment)

**What would change this:**

- If tests become flaky on CI, may need to increase timeout constants
- If new async patterns emerge, may need additional helpers

---

## Implementation Recommendations

### Recommended Approach ⭐

**Use testutil package for all async test patterns**

**Why this approach:**
- Centralizes best practices in one place
- Clear, documented patterns for common cases
- Reduces boilerplate in test files

**Trade-offs accepted:**
- Small overhead of polling vs. fixed sleep (negligible in tests)
- Additional dependency for test files

**Implementation sequence:**
1. Created `internal/testutil/wait.go` with core helpers
2. Updated each test file to use helpers
3. Added documentation for when to use each helper

### Files Changed

1. **internal/testutil/wait.go** - New testutil package with WaitFor, Eventually, YieldForGoroutine helpers
2. **internal/testutil/wait_test.go** - Tests for the testutil package
3. **pkg/opencode/monitor_test.go** - Replaced 6 time.Sleep calls with WaitFor/Eventually
4. **pkg/daemon/pool_test.go** - Replaced 2 time.Sleep calls with YieldForGoroutine
5. **pkg/daemon/completion_test.go** - Replaced 3 time.Sleep calls with WaitFor/Eventually
6. **pkg/capacity/manager_test.go** - Replaced 1 time.Sleep call with YieldForGoroutine
7. **pkg/verify/git_diff_test.go** - Documented git timestamp sleep with constant
8. **pkg/verify/constraint_test.go** - Replaced time.Sleep with os.Chtimes for deterministic mtime

---

## References

**Files Examined:**
- All *_test.go files for time.Sleep patterns
- Go testing documentation for best practices

**Commands Run:**
```bash
# Find all time.Sleep in tests
grep -rn "time.Sleep" --include="*_test.go"

# Verify tests pass
go test ./...

# Multiple runs for stability
for i in {1..3}; do go test ./pkg/opencode/... ./pkg/daemon/... ./pkg/capacity/... ./pkg/verify/... -count=1; done
```

---

## Investigation History

**2026-01-03 09:00:** Investigation started
- Initial question: How to harden flaky time-based tests?
- Context: Tests using arbitrary time.Sleep can fail under load

**2026-01-03 09:30:** Categorized flakiness patterns
- Found 4 distinct categories requiring different solutions

**2026-01-03 10:00:** Created testutil package
- Implemented WaitFor, Eventually, YieldForGoroutine helpers

**2026-01-03 10:30:** Fixed all test files
- Applied appropriate pattern to each category

**2026-01-03 11:00:** Investigation completed
- Status: Complete
- Key outcome: All tests pass consistently with no arbitrary sleeps remaining
