<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Daemon spawned duplicate agents due to race condition between spawn initiation and beads status update.

**Evidence:** 4 agents spawned for same issue kb-cli-0kk; daemon polls issues, spawns, but status isn't updated to in_progress until after `orch work` processes - subsequent polls see the issue still as "open".

**Knowledge:** Tracking spawned issues in daemon state (before async status update) prevents duplicates; 5-minute TTL allows stale entries to expire naturally.

**Next:** Fix implemented and tested - close issue.

---

# Investigation: Daemon Spawns Duplicate Agents for Same Issue

**Question:** Why does the daemon spawn duplicate agents for the same beads issue, and how can we prevent it?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** systematic-debugging
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Race condition between spawn initiation and status update

**Evidence:** The daemon flow is:
1. `ListReadyIssues()` returns issues with status="open" (line 207 in daemon.go)
2. `d.spawnFunc(issue.ID)` calls `orch work <beadsID>` (line 605)
3. `orch work` eventually calls `verify.UpdateIssueStatus(beadsID, "in_progress")` at spawn_cmd.go:698
4. But this happens AFTER many operations (API calls, context gathering, skill loading)
5. Meanwhile daemon can poll again, see the issue still as "open", and spawn another agent

**Source:** 
- `pkg/daemon/daemon.go:207` - `NextIssueExcluding` fetches fresh issues each call
- `pkg/daemon/daemon.go:605` - `spawnFunc` is called
- `cmd/orch/spawn_cmd.go:698` - Status update to in_progress happens late in spawn flow

**Significance:** The race window is the entire duration of spawn initialization (several seconds to tens of seconds), during which any daemon poll will see the issue as still available.

---

### Finding 2: Existing tracking mechanisms don't prevent the race

**Evidence:** The daemon has several tracking mechanisms:
- `WorkerPool` with slots that track `BeadsID` (pool.go:25)
- `RateLimiter` that tracks spawn history (rate_limiter.go)
- `skippedThisCycle` map for failed spawns (daemon.go run loop)

But none of these prevent spawning the same issue twice because:
- WorkerPool tracks capacity, not which issues are being processed
- RateLimiter tracks total spawns, not per-issue
- skippedThisCycle is cleared each poll cycle

**Source:** 
- `pkg/daemon/pool.go` - WorkerPool implementation
- `pkg/daemon/rate_limiter.go` - RateLimiter implementation
- `cmd/orch/daemon.go:284` - `skippedThisCycle` reset each cycle

**Significance:** A new tracking mechanism is needed specifically for recently-spawned issues.

---

### Finding 3: in_progress check alone is insufficient

**Evidence:** The daemon already skips in_progress issues (daemon.go:244):
```go
if issue.Status == "in_progress" {
    // Skip
}
```

But this only works AFTER the status is updated in beads. The race happens BEFORE the status update propagates.

**Source:** `pkg/daemon/daemon.go:244-248`

**Significance:** We need to track spawned issues BEFORE calling spawnFunc, not rely on beads status.

---

## Synthesis

**Key Insights:**

1. **The race window is architectural** - The daemon's stateless polling design (fetch fresh issues each time) combined with async spawn means there's always a window where duplicates can occur.

2. **Local tracking is the solution** - By tracking spawned issue IDs in daemon state immediately before calling spawnFunc, we close the race window without requiring synchronous status updates.

3. **TTL-based cleanup is simple and robust** - Rather than complex reconciliation, using a 5-minute TTL ensures stale entries expire naturally, even if spawns fail silently.

**Answer to Investigation Question:**

The daemon spawned duplicates because it polls beads for issues, spawns work via `orch work`, but the beads status update to "in_progress" happens after significant processing in the spawn flow. During this window, subsequent polls see the issue as still "open" and spawn additional agents.

The fix is to track spawned issue IDs in the daemon's state immediately before calling spawnFunc, and skip any issues that appear in this tracker during subsequent NextIssue calls. A 5-minute TTL allows entries to expire naturally.

---

## Structured Uncertainty

**What's tested:**

- ✅ SpawnedIssueTracker correctly tracks and expires entries (unit tests pass)
- ✅ NextIssue skips recently spawned issues (TestDaemon_SkipsRecentlySpawnedIssues passes)
- ✅ Once() marks issues before calling spawnFunc (TestDaemon_OnceMarksSpawned passes)
- ✅ Once() unmarks issues on spawn failure (TestDaemon_OnceUnmarksOnFailure passes)
- ✅ Duplicate spawns are prevented (TestDaemon_PreventsDuplicateSpawns passes)
- ✅ All existing daemon tests still pass

**What's untested:**

- ⚠️ Real-world daemon polling with actual `orch work` calls (would require integration test)
- ⚠️ Behavior under very high spawn rates (edge case)
- ⚠️ TTL value optimality (5 minutes is conservative estimate)

**What would change this:**

- Finding would be wrong if duplicates occur for reasons other than the race condition
- TTL might need adjustment if spawn times vary significantly

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**SpawnedIssueTracker** - Track spawned issue IDs in daemon state with TTL-based expiry

**Why this approach:**
- Closes race window by tracking BEFORE async spawn
- TTL cleanup is simple (no complex reconciliation)
- Minimal changes to existing code flow

**Trade-offs accepted:**
- 5-minute memory overhead for tracked IDs (negligible)
- Entries may persist slightly longer than necessary (acceptable)

**Implementation sequence:**
1. Create SpawnedIssueTracker type with TTL-based tracking
2. Add tracker to Daemon struct, initialize in constructors
3. Mark issues before spawnFunc, unmark on failure
4. Add skip check in NextIssueExcluding
5. Clean stale entries in ReconcileWithOpenCode

### Alternative Approaches Considered

**Option B: Synchronous status update**
- **Pros:** Simple conceptually
- **Cons:** Would require changing spawn flow to wait for status update; slows down spawning
- **When to use instead:** If we want single source of truth in beads

**Option C: Check OpenCode sessions for beads ID**
- **Pros:** Uses existing session data
- **Cons:** Session titles may not always contain beads ID; adds API call per issue
- **When to use instead:** If we need to verify truly active agents

**Rationale for recommendation:** Option A (SpawnedIssueTracker) is simplest and most robust. It doesn't depend on external services and has clear semantics.

---

### Implementation Details

**What was implemented:**

1. `pkg/daemon/spawn_tracker.go` - New SpawnedIssueTracker type
2. `pkg/daemon/daemon.go` - Added SpawnedIssues field, initialized in constructors
3. `pkg/daemon/daemon.go:229-235` - Skip check in NextIssueExcluding
4. `pkg/daemon/daemon.go:604-607, 609-611` - Mark/unmark in OnceExcluding
5. `pkg/daemon/daemon.go:691-694, 696-698` - Mark/unmark in OnceWithSlot
6. `pkg/daemon/daemon.go:379-381` - CleanStale call in ReconcileWithOpenCode
7. `pkg/daemon/spawn_tracker_test.go` - Comprehensive test coverage

**Things to watch out for:**
- ⚠️ TTL of 5 minutes may need adjustment if spawn times increase significantly
- ⚠️ If daemon restarts, tracked issues are lost (acceptable - status will catch up)

**Success criteria:**
- ✅ No duplicate spawns for same issue in daemon logs
- ✅ All daemon tests pass
- ✅ Memory usage remains stable (entries expire)

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - Core daemon logic, spawn flow
- `pkg/daemon/pool.go` - WorkerPool implementation (for comparison)
- `cmd/orch/spawn_cmd.go` - Spawn command, status update location
- `cmd/orch/daemon.go` - Daemon CLI, run loop

**Commands Run:**
```bash
# Build daemon package
go build ./pkg/daemon/...

# Run tests
go test ./pkg/daemon/... -v -count=1
```

**Related Artifacts:**
- **Decision:** N/A (implementation was straightforward)
- **Investigation:** This document

---

## Investigation History

**2026-01-06 10:00:** Investigation started
- Initial question: Why does daemon spawn 4 agents for same issue kb-cli-0kk?
- Context: Observed duplicate workspaces with same beads ID

**2026-01-06 10:30:** Root cause identified
- Race condition between spawn initiation and beads status update
- Daemon polls see issue as "open" before it's marked "in_progress"

**2026-01-06 11:00:** Fix implemented
- Created SpawnedIssueTracker
- Integrated into daemon spawn flow
- Added comprehensive tests

**2026-01-06 11:30:** Investigation completed
- Status: Complete
- Key outcome: SpawnedIssueTracker prevents duplicate spawns by tracking issues before async status update
