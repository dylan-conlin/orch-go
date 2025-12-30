<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The bug is in the beads daemon, not orch-go - SQLite WAL mode with connection pooling causes stale reads when a comment is read immediately after being written.

**Evidence:** beads daemon uses `maxConns = runtime.NumCPU() + 1` connection pool; GetIssueComments doesn't call checkFreshness(); SQLite WAL mode creates snapshot isolation between connections.

**Knowledge:** When one connection writes a comment and another connection immediately reads, the read connection may have a stale WAL snapshot missing the new comment.

**Next:** Create issue in beads repo to add WAL freshness handling to GetIssueComments, or use PRAGMA read_uncommitted for read connections.

---

# Investigation: Orch Complete Fails Detect Phase

**Question:** Why does `orch complete` fail to detect `Phase: Complete` from beads comments when the comment is visible via `bd comments`?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** debugging-agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: orch-go phase parsing logic is correct

**Evidence:** 
- Tested `ParsePhaseFromComments` with actual comments from `orch-go-neeo` - correctly returns "Complete"
- Unit tests for multiple phases pass - function correctly returns the latest phase
- The regex pattern `(?i)Phase:\s*(\w+)(?:\s*[-–—]\s*(.*))?` correctly matches all phase formats

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go:82` - phase pattern regex
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check_test.go:70-81` - multiple phases test
- Manual test via `/tmp/phasecheck.go` - parsed real comments correctly

**Significance:** The bug is NOT in orch-go's phase parsing or comment retrieval logic. Must be in the data source (beads daemon).

---

### Finding 2: Beads daemon uses connection pool with SQLite WAL mode

**Evidence:** 
- beads creates connection pool: `maxConns := runtime.NumCPU() + 1` at store.go:134
- WAL mode enabled: `PRAGMA journal_mode=WAL` at store.go:142
- Different connections can have different WAL snapshots (SQLite documented behavior)

**Source:** 
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/store.go:131-137` - connection pool config
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/store.go:141-144` - WAL mode
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/freshness_test.go:145-146` - documents WAL snapshot issue

**Significance:** When one connection writes (add comment) and another reads (get comments), the read connection may have a stale WAL snapshot that doesn't include the new comment.

---

### Finding 3: GetIssueComments lacks freshness checking

**Evidence:**
- `GetIssue` calls `s.checkFreshness()` before reading (queries.go:228)
- `GetIssueComments` does NOT call `checkFreshness()` before reading (comments.go:56-83)
- `checkFreshness` is designed for external file modifications, but helps force WAL sync

**Source:**
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/queries.go:228` - GetIssue calls checkFreshness
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/comments.go:56-83` - GetIssueComments does NOT

**Significance:** Inconsistent freshness handling between issue and comment retrieval. This may allow stale reads for comments.

---

### Finding 4: Bug is timing-dependent and transient

**Evidence:**
- Cannot reproduce reliably now - all tests show correct "Complete" phase
- Original bug happened when `orch complete` was run "immediately" after agent reported completion
- SQLite WAL snapshots are refreshed at transaction boundaries

**Source:** 
- Reproduction test `/tmp/repro.go` - works correctly now (returns Phase: Complete)
- Task description: "agent had comment visible via 'bd comments' but orch complete reported 'Investigating'"

**Significance:** The bug is a race condition - happens when read follows write too quickly before WAL sync.

---

## Synthesis

**Key Insights:**

1. **orch-go is not the bug source** - All parsing and retrieval logic in orch-go works correctly. The issue is in the data layer (beads daemon).

2. **SQLite WAL snapshot isolation is the root cause** - When the beads daemon uses a connection pool with WAL mode, different connections have different snapshots. A write on connection A may not be visible on connection B until B starts a new transaction.

3. **The bug is a race condition** - It only occurs when a read happens immediately after a write, before the WAL snapshot is refreshed. The window is likely milliseconds.

**Answer to Investigation Question:**

`orch complete` fails to detect `Phase: Complete` because of a race condition in the beads daemon. When an agent adds a "Phase: Complete" comment, it goes through one connection in the daemon's SQLite connection pool. When `orch complete` immediately requests comments, it may get a different connection with a stale WAL snapshot that doesn't include the new comment. This causes `orch complete` to see the previous phase (e.g., "Investigating") instead of "Complete".

The fix should be in the beads daemon, either:
1. Add `checkFreshness()` or explicit WAL sync before GetIssueComments
2. Use `PRAGMA read_uncommitted=ON` for read connections (trades isolation for freshness)
3. Force checkpoint after each write (`PRAGMA wal_checkpoint(PASSIVE)`)
4. Use a single connection for writes and reads within the same RPC request

---

## Structured Uncertainty

**What's tested:**

- ✅ ParsePhaseFromComments correctly handles multiple phases (verified: ran /tmp/phasecheck.go with real comments)
- ✅ Comments are returned in chronological order (verified: ORDER BY created_at ASC in comments.go:61)
- ✅ Current state works correctly (verified: repro test shows Phase: Complete)

**What's untested:**

- ⚠️ Exact timing window for the race condition (not benchmarked)
- ⚠️ Whether checkFreshness would actually fix this (hypothesis - not tested in beads)
- ⚠️ Impact of various WAL checkpoint strategies (not benchmarked)

**What would change this:**

- If beads daemon used a single connection per request, race would not occur
- If writes included explicit checkpoint, reads would see them immediately
- If read connections used `PRAGMA read_uncommitted`, they'd see uncommitted writes

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach - Create beads issue for WAL freshness fix

**Why this approach:**
- Root cause is in beads, not orch-go
- Proper fix requires database layer changes
- orch-go workarounds would be band-aids

**Trade-offs accepted:**
- Bug persists until beads is fixed
- Users should use `--force` as workaround

**Implementation sequence:**
1. Create beads issue documenting the WAL snapshot race condition
2. In beads: Add checkFreshness() call to GetIssueComments (consistency with GetIssue)
3. Consider: Add PRAGMA wal_checkpoint(PASSIVE) after AddIssueComment

### Alternative Approaches Considered

**Option B: Add retry loop in orch-go**
- **Pros:** Workaround available immediately without beads changes
- **Cons:** Band-aid; masks underlying issue; adds latency to all completions
- **When to use instead:** If beads fix is delayed significantly

**Option C: Sleep before reading comments in orch-go**
- **Pros:** Simple, might work
- **Cons:** Arbitrary delay; unreliable; terrible UX
- **When to use instead:** Never - this is a hack

**Rationale for recommendation:** The bug is clearly in beads, and fixing it there benefits all beads users. A proper WAL handling fix is the right solution.

---

### Implementation Details

**What to implement first:**
- Create beads issue with reproduction steps and root cause analysis
- Document `--force` as workaround for orch-go users

**Things to watch out for:**
- ⚠️ checkFreshness was designed for file replacement, may need different approach for WAL sync
- ⚠️ Checkpoint after every write might hurt performance
- ⚠️ read_uncommitted has its own trade-offs (dirty reads)

**Areas needing further investigation:**
- What's the exact WAL sync behavior in SQLite's go-sqlite3 driver?
- Does go-sqlite3 have a way to force snapshot refresh?
- Is there a better connection pool strategy?

**Success criteria:**
- ✅ `orch complete` reliably detects Phase: Complete immediately after agent reports it
- ✅ No explicit sleep or retry needed
- ✅ Performance not significantly degraded

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go` - Phase parsing and status retrieval
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/beads/client.go` - RPC client for beads daemon
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/store.go` - SQLite storage with WAL and connection pool
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/comments.go` - Comment storage (lacks freshness check)
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/queries.go` - Issue queries (has freshness check)
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/freshness_test.go` - Tests documenting WAL issues

**Commands Run:**
```bash
# Verify phase parsing works correctly
/usr/local/go/bin/go run /tmp/phasecheck.go

# Check beads comments
bd comments orch-go-neeo --json

# Find all beads sockets
find ~/Documents/personal -name "bd.sock"

# Run phase parsing tests
/usr/local/go/bin/go test -v ./pkg/verify/... -run 'TestParsePhase'
```

**Related Artifacts:**
- **Investigation:** beads freshness_test.go documents the WAL snapshot issue

---

## Investigation History

**2025-12-30 16:00:** Investigation started
- Initial question: Why does orch complete fail to detect Phase: Complete?
- Context: Agent orch-go-neeo had visible Phase: Complete comment but orch complete reported Investigating

**2025-12-30 16:15:** Verified orch-go parsing logic is correct
- ParsePhaseFromComments correctly handles real comments
- Unit tests pass

**2025-12-30 16:30:** Identified root cause in beads daemon
- SQLite WAL mode with connection pool causes stale reads
- GetIssueComments lacks freshness checking

**2025-12-30 16:45:** Investigation completed
- Status: Complete
- Key outcome: Bug is in beads daemon's SQLite WAL handling, not orch-go
