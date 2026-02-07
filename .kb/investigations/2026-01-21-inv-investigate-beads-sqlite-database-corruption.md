<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Beads SQLite corruption is caused by daemon auto-start inside Claude Code sandbox combined with rapid open/close cycles when daemon fails to start (chmod socket error).

**Evidence:** Daemon logs show 57 start attempts on Jan 21, each opening database then failing at chmod. Corrupted backup shows empty WAL file (0 bytes) + 32KB shm file, indicating interrupted WAL operation.

**Knowledge:** Sandbox environment cannot run beads daemon (chmod fails on host filesystem), but auto-start keeps trying. Each failed attempt opens/closes database rapidly, risking WAL corruption. Direct mode should be forced in sandbox.

**Next:** Implement beads detection of sandbox environment to disable daemon auto-start and force direct mode. Consider adding periodic integrity checks.

**Promote to Decision:** Actioned - constraint documented (sandbox must use direct mode)

---

# Investigation: Beads SQLite Database Corruption Root Cause

**Question:** What causes recurring beads SQLite database corruption in orch-go/.beads/beads.db, and how can it be prevented?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** Agent
**Phase:** Complete
**Next Step:** None - create follow-up issue for fix
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** .kb/investigations/2026-01-21-inv-fix-beads-sqlite-database-corruption.md (extends recovery findings with root cause)
**Superseded-By:** N/A

---

## Findings

### Finding 1: Daemon repeatedly auto-starts inside Claude Code sandbox and fails

**Evidence:**
- Daemon log shows 57 start attempts on Jan 21, 2026
- Every attempt fails with: `failed to set socket permissions: chmod /Users/dylanconlin/Documents/personal/orch-go/.beads/bd.sock: invalid argument`
- The "invalid argument" error occurs because the sandbox (Linux container) cannot chmod Unix sockets on the host macOS filesystem
- Each failed start still opens the database, then closes it via defer

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/.beads/daemon.log`
- Sample entries:
  ```
  time=2026-01-21T15:52:34.167Z level=INFO msg="database opened" path=...
  time=2026-01-21T15:52:34.180Z level=ERROR msg="RPC server error" error="failed to set socket permissions: chmod ... invalid argument"
  ```

**Significance:** The daemon cannot function inside the Claude Code sandbox, but it keeps trying due to auto-start behavior. This creates high-frequency database open/close cycles.

---

### Finding 2: Corrupted database backup shows interrupted WAL operation

**Evidence:**
- Corrupted backup directory contains:
  - `beads.db` (5,107,712 bytes) - main database file
  - `beads.db-shm` (32,768 bytes) - shared memory file present
  - `beads.db-wal` (0 bytes) - **WAL file exists but is empty**
- An empty WAL file (0 bytes) is abnormal - indicates truncation or incomplete operation
- The presence of `-shm` file indicates WAL mode was active

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/.beads/backup-corrupted-2026-01-21/`
- `ls -la` output showing file sizes

**Significance:** The empty WAL file suggests corruption occurred during a WAL checkpoint or truncation operation. This is consistent with the hypothesis that rapid open/close cycles (57 attempts in one day) caused a race condition.

---

### Finding 3: Beads properly checkpoints WAL on Close() but rapid cycles may race

**Evidence:**
- `store.Close()` in beads properly checkpoints WAL:
  ```go
  func (s *SQLiteStorage) Close() error {
      s.closed.Store(true)
      s.reconnectMu.Lock()
      defer s.reconnectMu.Unlock()
      // Checkpoint WAL to ensure all writes are persisted
      _, _ = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
      return s.db.Close()
  }
  ```
- However, daemon starts happen rapidly (some within seconds of each other)
- If next daemon process starts before previous `Close()` completes the checkpoint, both processes could have the database open simultaneously

**Source:**
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/store.go:206-217`
- Daemon log timestamps showing rapid succession:
  - 15:52:33.993Z - start
  - 15:52:44.333Z - start (10 seconds later)
  - Pattern repeats throughout the day

**Significance:** While SQLite has locking to prevent concurrent writes, the rapid open/close/checkpoint cycle creates a window where corruption can occur if the filesystem doesn't handle the operations atomically (especially across sandbox/host boundary).

---

### Finding 4: Sandbox filesystem boundary may not support SQLite WAL atomicity

**Evidence:**
- Claude Code sandbox runs as a Linux container
- The `.beads/` directory is on the host macOS filesystem
- SQLite WAL mode requires certain atomic filesystem operations (especially for checkpoint)
- The sandbox-to-host filesystem mount may not provide the same guarantees as native filesystem access
- The `chmod` failure confirms the sandbox has limitations with host filesystem operations

**Source:**
- Daemon error: `chmod /path/to/bd.sock: invalid argument` - indicates sandbox filesystem limitation
- The "invalid argument" ERRNO is specific to operations not supported by the filesystem layer

**Significance:** WAL mode may not be safe across the sandbox/host boundary. This is a fundamental environmental constraint, not a bug in beads.

---

## Synthesis

**Key Insights:**

1. **Sandbox-host boundary is unsafe for WAL mode** - The Claude Code sandbox cannot properly manage SQLite WAL operations on the host filesystem. The chmod failure is the visible symptom, but WAL checkpoint may also be affected.

2. **Auto-start creates corruption-prone conditions** - When agents run `bd` commands, the auto-start behavior triggers daemon attempts inside the sandbox. Each attempt opens/closes the database rapidly, creating opportunities for race conditions or incomplete operations.

3. **Empty WAL file is the corruption signature** - The 0-byte WAL file indicates the corruption occurred during a WAL truncation (checkpoint) operation, not during normal writes. This is consistent with a race between close operations or interrupted checkpoint.

**Answer to Investigation Question:**

The root cause of recurring beads SQLite corruption is the combination of:

1. **Daemon auto-start inside Claude Code sandbox** - Agents running `bd` commands trigger daemon auto-start, which fails because the sandbox cannot chmod the Unix socket

2. **Rapid database open/close cycles** - Each failed daemon attempt still opens and closes the database, performing WAL checkpoints

3. **Race conditions in high-frequency close cycles** - With 57 daemon attempts in one day, there's opportunity for races between closing one database handle and opening the next

4. **Possible sandbox filesystem limitations** - The sandbox-to-host filesystem mount may not fully support SQLite WAL atomicity requirements

The corruption manifests as an empty (0-byte) WAL file with an orphaned shm file, which breaks WAL mode and causes "database disk image is malformed" errors.

---

## Structured Uncertainty

**What's tested:**

- ✅ Daemon auto-start fails in sandbox with chmod error (verified: daemon.log shows pattern)
- ✅ 57 daemon start attempts occurred on corruption day (verified: grep count)
- ✅ Corrupted backup has 0-byte WAL + 32KB shm (verified: ls -la)
- ✅ Recovery via delete + bd init works (verified: previous investigation)
- ✅ Beads properly checkpoints WAL in Close() (verified: source code review)

**What's untested:**

- ⚠️ Exact moment of corruption (would need more granular logging)
- ⚠️ Whether host daemon was also running during corruption (concurrent access)
- ⚠️ Whether sandbox filesystem mount actually fails atomicity for WAL operations
- ⚠️ Whether the race condition is between sequential daemons or daemon vs CLI

**What would change this:**

- If corruption occurs with daemon running on host only (no sandbox), root cause would be different
- If WAL file is non-empty in corruption, truncation race wouldn't be the cause
- If sandbox properly supports all SQLite operations, the issue would be elsewhere

---

## Implementation Recommendations

**Purpose:** Prevent beads SQLite corruption in sandbox environments.

### Recommended Approach ⭐

**Sandbox-aware daemon disable + direct mode** - Detect sandbox environment and automatically disable daemon auto-start, forcing direct mode for all operations.

**Why this approach:**
- Addresses root cause (daemon can't work in sandbox anyway)
- No code changes to SQLite handling needed
- Immediate fix with minimal risk
- Beads already has sandbox detection for this purpose

**Trade-offs accepted:**
- Sandbox agents won't benefit from daemon's automatic sync
- Need to rely on manual `bd sync` from host

**Implementation sequence:**
1. Verify beads sandbox detection is working correctly
2. Ensure sandbox detection forces `--no-daemon` behavior
3. Add logging when sandbox mode is detected
4. Consider adding periodic integrity check on database open

### Alternative Approaches Considered

**Option B: Add mutex/locking around daemon start**
- **Pros:** Would prevent rapid-fire daemon attempts
- **Cons:** Complex, doesn't address fundamental sandbox limitation
- **When to use instead:** If corruption occurs outside sandbox environment

**Option C: Disable WAL mode for shared filesystem access**
- **Pros:** DELETE journal mode is simpler and may be safer
- **Cons:** Performance regression, doesn't address sandbox-host boundary issue
- **When to use instead:** If WAL corruption occurs in non-sandbox environments

**Option D: Auto-recovery on corruption detection**
- **Pros:** Makes corruption less impactful
- **Cons:** Data loss risk, doesn't prevent corruption
- **When to use instead:** As supplementary measure, not primary fix

**Rationale for recommendation:** Since the daemon fundamentally cannot work in the sandbox (chmod fails), the cleanest fix is to prevent daemon attempts entirely when in sandbox. This eliminates the rapid open/close cycles that cause corruption.

---

### Implementation Details

**What to implement first:**
- Verify beads sandbox detection: Check if `cmd/bd/main.go:317` sandbox detection is triggering
- If sandbox detection isn't working, fix it
- Add explicit warning when sandbox forces direct mode

**Things to watch out for:**
- ⚠️ Sandbox detection must work for all agent spawn modes (Docker, Claude CLI, OpenCode)
- ⚠️ Some legitimate use cases might need daemon in containerized environments
- ⚠️ Need to handle case where daemon was already running before sandbox started

**Areas needing further investigation:**
- Why sandbox detection might not be preventing daemon auto-start
- Whether there are other sandbox-host filesystem limitations affecting beads
- Performance impact of forcing direct mode for all agent operations

**Success criteria:**
- ✅ No daemon start attempts logged from sandbox environment
- ✅ All `bd` commands from agents work in direct mode
- ✅ No database corruption after 1 week of agent operation
- ✅ Monitoring: Add integrity check on database open, log if corruption detected

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/store.go` - SQLite WAL handling, Close() checkpoint
- `/Users/dylanconlin/Documents/personal/beads/internal/rpc/server_lifecycle_conn.go` - RPC server Start(), signal handling
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/daemon.go` - Daemon startup, runDaemonLoop
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/daemon_server.go` - startRPCServer
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/main.go` - Sandbox detection, direct mode fallback
- `/Users/dylanconlin/Documents/personal/orch-go/.beads/daemon.log` - Daemon logs showing failure pattern
- `/Users/dylanconlin/Documents/personal/orch-go/.beads/backup-corrupted-2026-01-21/` - Corrupted database files

**Commands Run:**
```bash
# Count daemon start attempts on Jan 21
grep -E "level=INFO msg=\"Daemon started" daemon.log | grep "2026-01-21" | wc -l
# Result: 57

# Check corrupted backup files
ls -la backup-corrupted-2026-01-21/
# Shows 0-byte WAL file

# Search for corruption-related errors
grep -i "corrupt\|malformed\|error" daemon.log | tail -100
```

**External Documentation:**
- SQLite WAL mode documentation - Explains checkpoint behavior and atomicity requirements
- Claude Code sandbox architecture - Understanding container-host filesystem interaction

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-21-inv-fix-beads-sqlite-database-corruption.md - Recovery procedure (this investigation adds root cause)

---

## Investigation History

**2026-01-21 20:15:** Investigation started
- Initial question: What causes recurring beads SQLite corruption?
- Context: Previous investigation fixed corruption but noted root cause as "UNTESTED"

**2026-01-21 20:30:** Identified daemon failure pattern
- Found 57 daemon start attempts with chmod failure
- Identified sandbox environment as source of failures

**2026-01-21 20:45:** Analyzed corrupted backup
- Found 0-byte WAL file + 32KB shm file
- Concluded corruption occurred during WAL checkpoint

**2026-01-21 20:55:** Root cause identified
- Rapid daemon open/close cycles in sandbox cause WAL corruption
- Recommendation: Force direct mode in sandbox, disable daemon auto-start

**2026-01-21 21:00:** Investigation completed
- Status: Complete
- Key outcome: Root cause is daemon auto-start in sandbox causing rapid database open/close cycles leading to WAL corruption
