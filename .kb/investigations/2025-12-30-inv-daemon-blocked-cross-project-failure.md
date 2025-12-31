## Summary (D.E.K.N.)

**Delta:** Daemon blocked ALL spawning when any issue had an unfilled failure report because it breaks out of spawn loop on first failure.

**Evidence:** In `cmd/orch/daemon.go:275-288`, when `d.Once()` returns failure, daemon `break`s instead of trying next issue.

**Knowledge:** Spawn failures should skip to next issue, not block entire queue. Added `NextIssueExcluding` and `OnceExcluding` with per-cycle skip tracking.

**Next:** Close - fix implemented and tested.

---

# Investigation: Daemon Blocked Cross Project Failure

**Question:** Why does a failure report gate in one issue block ALL daemon spawning, including issues from other projects?

**Started:** 2025-12-30
**Updated:** 2025-12-31
**Owner:** og-debug-daemon-blocked-cross-30dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Daemon loop breaks on any spawn failure

**Evidence:** In `cmd/orch/daemon.go`, the daemon loop calls `d.Once()` and if the result is `!result.Processed`, it breaks:

```go
result, err := d.Once()
// ...
if !result.Processed {
    // No more issues or spawn blocked (capacity, error, etc.)
    // ...
    break  // <-- This breaks on ANY failure
}
```

**Source:** `cmd/orch/daemon.go:275-288`

**Significance:** When a spawn fails (e.g., failure report gate), the daemon stops trying other issues in that cycle.

---

### Finding 2: NextIssue always returns the same first eligible issue

**Evidence:** `NextIssue()` sorts issues by priority and returns the first one that passes all filters. It has no memory of previously attempted issues.

**Source:** `pkg/daemon/daemon.go:271-326`

**Significance:** After a spawn failure, the next `NextIssue()` call returns the SAME issue, creating an infinite failure loop until that issue's blocker is resolved.

---

### Finding 3: Failure report gate is correctly scoped per-issue

**Evidence:** `CheckFailureReport(projectDir, beadsID)` scans workspaces and only returns a failure report if the SPAWN_CONTEXT.md contains the specific beads ID:

```go
if !strings.Contains(contentStr, beadsID) {
    continue // This workspace is for a different issue
}
```

**Source:** `pkg/spawn/context.go:1137-1138`

**Significance:** The failure report check itself is NOT the bug - it correctly only finds reports for the specific issue.

---

## Synthesis

**Key Insights:**

1. **The bug is in the daemon loop, not the failure report check** - The check is correctly scoped, but the daemon's response to ANY failure is to break the entire spawn loop.

2. **No skip tracking existed** - Without tracking which issues failed this cycle, the daemon repeatedly attempts the same failing issue.

3. **Cross-project is a red herring** - The "cross-project" aspect mentioned in the issue title is less relevant than the general behavior of any spawn failure blocking all spawning.

**Answer to Investigation Question:**

A failure report gate doesn't block based on project - it blocks because ANY spawn failure causes the daemon to break out of its spawn loop. Since `NextIssue()` has no memory and returns the same failing issue on the next cycle, the daemon gets stuck retrying that issue forever, never reaching other issues in the queue.

The fix is to track issues that failed to spawn in the current cycle and skip them when selecting the next issue.

---

## Structured Uncertainty

**What's tested:**

- ✅ `NextIssueExcluding` correctly skips issues in the skip set (verified: 4 new unit tests)
- ✅ Build compiles successfully (verified: `go build ./...`)
- ✅ All daemon tests pass (verified: `go test ./pkg/daemon/...`)

**What's untested:**

- ⚠️ Production behavior with multiple failing issues (not end-to-end tested)
- ⚠️ Log output visibility for skipped issues (visual inspection only)

**What would change this:**

- Finding would be wrong if there's a different code path for the daemon that doesn't use `Once()`
- Finding would be incomplete if there are other places that break on spawn failure

---

## Implementation Recommendations

### Recommended Approach ⭐

**Per-cycle skip tracking** - Track failed issues in a map for each spawn cycle, skip them in subsequent issue selection.

**Why this approach:**
- Minimal code changes (3 files, ~80 lines added)
- Backwards compatible (existing methods delegate to new ones)
- Issues are retried on next cycle after potential fixes

**Trade-offs accepted:**
- Skip map is recreated each cycle (small memory allocation)
- Failed issues are still logged each time they're skipped (visibility vs noise)

**Implementation sequence:**
1. Add `NextIssueExcluding(skip map[string]bool)` to daemon package
2. Add `OnceExcluding(skip map[string]bool)` to daemon package
3. Update daemon loop to use skip tracking

### Alternative Approaches Considered

**Option B: Mark issues in beads as blocked**
- **Pros:** Persistent, visible in `bd list`
- **Cons:** Requires beads CLI calls, changes issue state permanently, harder to resume
- **When to use instead:** If issues should be blocked until human intervention

**Rationale for recommendation:** Per-cycle skip tracking is the least invasive fix that solves the immediate problem without changing issue state.

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - Daemon core logic, NextIssue, Once methods
- `cmd/orch/daemon.go` - Daemon loop, spawn cycle handling
- `pkg/spawn/context.go` - CheckFailureReport implementation

**Commands Run:**
```bash
# Build verification
go build ./...

# Test runs
go test ./pkg/daemon/... -v
```

---

## Investigation History

**2025-12-31 01:18:** Investigation started
- Initial question: Why does failure in one issue block all daemon spawning?
- Context: Daemon gets stuck on issues with unfilled failure reports

**2025-12-31 01:22:** Root cause identified
- Found the `break` statement in daemon loop that stops on any failure
- Identified that `NextIssue()` has no skip tracking

**2025-12-31 01:28:** Implementation completed
- Added `NextIssueExcluding` and `OnceExcluding` methods
- Updated daemon loop with per-cycle skip tracking
- All tests passing

**2025-12-31 01:30:** Investigation completed
- Status: Complete
- Key outcome: Daemon now skips failed issues and continues processing queue
