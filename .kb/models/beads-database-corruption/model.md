# Model: Beads SQLite Database Corruption

**Domain:** Beads / SQLite / Data Integrity
**Last Updated:** 2026-03-19
**Synthesized From:** 3 investigations (Jan 21-22), 3+ corruption incidents (Jan 21-22), daemon logs showing 57+ restart cycles, Feb 2026 architecture review

**Probes:**
- 2026-03-18: Knowledge decay verification — all 6 fix claims confirmed against current beads codebase, model accurate and current
- 2026-03-19: Model drift fix — removed stale `.beads/daemon.log` reference (file no longer exists since daemon doesn't run with JSONL-only default)

---

## Summary (30 seconds)

Beads SQLite corruption occurs when the daemon enters a **rapid restart loop** (any failure → retry → fail → retry). Each cycle opens/closes the database performing WAL checkpoint. High-frequency checkpoints across unstable conditions (sandbox filesystem, legacy validation, any daemon failure) create opportunities for incomplete WAL operations, manifesting as **0-byte WAL files** that corrupt the database. The fix is **preventing rapid restarts**, not fixing individual failure causes.

---

## Current Architecture (Feb 2026)

**Status: RESOLVED** - All recommended fixes have been implemented in Dylan's beads fork.

### Fixes Applied

| Commit | Fix | Effect |
|--------|-----|--------|
| `9953b9cb` | Prevent daemon auto-start in sandbox | No daemon attempts in Claude Code |
| `4da15127` | Detect Claude Code and Docker environments | Early sandbox detection |
| `98e5c750` | Use JSONL-only mode when sandbox detected | No SQLite WAL at all |
| `2198ad78` | Prevent rapid restart loops | Backoff between daemon attempts |
| `041af3fa` | Pre-flight fingerprint validation | Fail before DB open |
| `629441ad` | Make JSONL-only the default storage mode | SQLite now opt-in |

### Current Flow

```
Agent in sandbox → bd command → sandbox detected → JSONL-only mode
                                                 → no daemon started
                                                 → no SQLite WAL
                                                 → no corruption risk

Host CLI user → bd command → JSONL-only default → direct file ops
                           → --sqlite flag → daemon available → RPC
```

### Daemon Relevance

With JSONL-only as default, the daemon provides **minimal value**:

| Feature | Value with JSONL-only |
|---------|----------------------|
| RPC performance (10x) | None - JSONL ops already fast |
| Mutation events | None - no subscribers |
| Auto-sync coordination | None - direct writes |
| SQLite connection pooling | None - no SQLite |

**Conclusion:** Daemon is effectively legacy for agent workflows. JSONL-only is simpler, safer, sufficient.

### SQLite Still Required For

- `bd compact --auto/--analyze/--apply` (semantic compression)
- `bd wisp gc` (transient molecule cleanup)
- `bd gate wait` (blocking dependency waits)
- `bd mol burn` (molecule conversion)
- `bd cook` (recipe execution)

All daily operations work with JSONL-only.

### Fork Strategy

Dylan's fork stays diverged from upstream (steveyegge/beads). See `.kb/decisions/2026-02-05-beads-fork-stay-diverged.md`.

---

## Core Mechanism (Historical)

### The Corruption Cycle

```
Daemon starts → Opens database → Enables WAL mode → Fails for some reason
                                        ↓
                              Defer: WAL checkpoint + Close
                                        ↓
                              Daemon restarts (5-10 sec later)
                                        ↓
                              Opens database DURING checkpoint?
                                        ↓
                              Incomplete WAL operation
                                        ↓
                              0-byte WAL file = CORRUPTION
```

**Key insight:** The failure cause doesn't matter. What matters is:
1. Daemon fails after opening database
2. Daemon restarts quickly (< 10 seconds)
3. Repeat 10-50+ times in a day
4. Eventually hit race condition → WAL corruption

### Evidence: Multiple Failure Modes, Same Outcome

| Date | Failure Cause | Restart Count | Result |
|------|---------------|---------------|--------|
| Jan 21 AM | Sandbox chmod (socket permissions) | 57 | 0-byte WAL, corruption |
| Jan 21 PM | Unknown (2nd incident same day) | Unknown | Backup created |
| Jan 22 | Legacy database fingerprint | 9+ in 3 min | 0-byte WAL, no DB |

**Pattern confirmed:** Different root causes, identical corruption signature.

### Why Rapid Restarts Are Fatal

**Beads Close() implementation:**

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

The `TRUNCATE` checkpoint:
1. Copies WAL contents to main database
2. Resets WAL file to zero length
3. Releases WAL lock

**Race window:** If next daemon starts during steps 1-2, both processes have database open. SQLite handles concurrent *reads*, but concurrent checkpoint operations can leave WAL in inconsistent state.

**Exacerbating factor:** Claude Code sandbox uses Linux container accessing macOS host filesystem. The mount layer may not provide same atomicity guarantees as native access.

---

## Why This Fails

### 1. No Backoff on Daemon Failure

**What happens:** Daemon fails → restarts immediately → fails → restarts → cycles indefinitely.

**Root cause:** No exponential backoff between restart attempts. launchd `KeepAlive` causes immediate restart.

**Why detection is hard:** Each individual failure looks like "bad luck" - only aggregate pattern reveals problem.

**Fix:** Implement minimum interval between daemon starts (e.g., 30 seconds).

### 2. Sandbox Environment Not Detected Early

**What happens:** Daemon starts inside Claude Code sandbox, tries to chmod socket, fails, but has already opened database.

**Root cause:** Sandbox detection happens AFTER database open, not before daemon start.

**Fix:** Detect sandbox at CLI entry point, skip daemon auto-start entirely.

### 3. Legacy Database Validation Fails Late

**What happens:** Database opens successfully, WAL enabled, THEN fingerprint validation fails.

**Root cause:** Validation is post-open check, not pre-open gate.

**Fix:** Check fingerprint before enabling WAL mode.

### 4. No Health Gate Before Operations

**What happens:** Daemon starts despite known-bad state (missing fingerprint, sandbox environment).

**Root cause:** No pre-flight checks before daemon entry point.

**Fix:** `bd daemon start` should validate prerequisites before proceeding.

---

## Constraints

### Why WAL Mode?

**Constraint:** Beads uses SQLite WAL mode for concurrent read/write performance.

**Implication:** WAL requires atomic filesystem operations for checkpoint. Degrades to corruption risk in non-atomic environments.

**This enables:** Fast reads during writes, better concurrency
**This constrains:** Requires filesystem atomicity guarantees, vulnerable to incomplete checkpoints

### Why Daemon Auto-Start?

**Constraint:** Running `bd` commands auto-starts daemon if not running.

**Implication:** Any bd command in sandbox triggers daemon → triggers corruption cycle.

**This enables:** Zero-config user experience, daemon always available
**This constrains:** No way to prevent daemon start without explicit flag

### Why launchd KeepAlive?

**Constraint:** macOS launchd configured with `KeepAlive: true` for daemon resilience.

**Implication:** Failed daemon immediately restarts, creating rapid-cycle conditions.

**This enables:** Daemon survives crashes, auto-recovery
**This constrains:** No backoff between restarts, amplifies failure modes

---

## Recovery

### Immediate Recovery (Current Corruption)

```bash
# 1. Stop any daemon attempts
pkill -f "bd daemon"

# 2. Remove corrupted files
cd ~/Documents/personal/orch-go/.beads
rm -f beads.db beads.db-shm beads.db-wal daemon.lock

# 3. Rebuild from JSONL (authoritative source)
bd init --prefix orch-go

# 4. Verify recovery
bd doctor
bd list | head -5
```

**Why this works:** `issues.jsonl` is append-only authoritative source. Database is derived/cached state that can always be rebuilt.

### Prevention (Stop Future Corruption)

**Short-term (immediate):**

```bash
# Disable daemon auto-start in sandbox (agents should use direct mode)
# Add to SPAWN_CONTEXT.md or agent environment:
export BD_DIRECT_MODE=1
```

**Medium-term (beads changes needed):**

1. **Backoff between restarts** - Minimum 30s between daemon start attempts
2. **Pre-flight validation** - Check sandbox/fingerprint BEFORE database open
3. **Graceful degradation** - Fall back to direct mode if daemon prerequisites fail

**Long-term (architecture):**

1. **Separate daemon from CLI** - Daemon as standalone service, not auto-started by CLI
2. **Health endpoint** - Daemon exposes health check, don't start if unhealthy
3. **Watchdog with backoff** - launchd replacement with exponential backoff

---

## Incident History

| Date | Cause | Recovery | Time Lost |
|------|-------|----------|-----------|
| Jan 21 08:48 | Sandbox chmod | delete + init | ~30 min |
| Jan 21 14:44 | Unknown | delete + init | ~30 min |
| Jan 21 16:46 | Unknown | delete + init | ~30 min |
| Jan 22 17:xx | Legacy fingerprint | pending | ongoing |

**Total impact:** 3+ hours of disruption, recurring ~1-2x daily.

---

## Detection

### Symptoms

- `bd list` returns "database disk image is malformed"
- `bd doctor` shows "Fresh clone detected (no database)" despite JSONL existing
- 0-byte `beads.db-wal` file with 32KB `beads.db-shm` file
- Missing `beads.db` main file

### Monitoring

```bash
# Check for corruption signature
ls -la .beads/beads.db* 2>/dev/null

# Expected healthy state:
# beads.db      ~5MB   (main database)
# beads.db-wal  0-32KB (write-ahead log, can be 0 if checkpointed)
# beads.db-shm  32KB   (shared memory)

# Corruption signature:
# beads.db      MISSING
# beads.db-wal  0 bytes
# beads.db-shm  32KB

# Check daemon restart frequency (daemon.log only exists if daemon is running;
# with JSONL-only default, daemon.log is typically absent — that's healthy)
grep "Daemon started" .beads/daemon.log 2>/dev/null | tail -20
# If multiple starts within minutes = rapid-cycle problem
```

### orch doctor Check (Recommended)

Add to `orch doctor`:
```
- [ ] Beads WAL state healthy (no orphaned WAL/SHM without main DB)
- [ ] Daemon restart frequency < 1/minute over last hour
```

---

## References

**Investigations:**
- `.kb/investigations/archived/2026-01-21-inv-investigate-beads-sqlite-database-corruption.md` - Root cause (sandbox chmod)
- `.kb/investigations/2026-01-21-inv-fix-beads-sqlite-database-corruption.md` - Recovery procedure
- `.kb/investigations/archived/2026-01-21-urgent-beads-sqlite-corruption.md` - Incident tracking

**Source Code:**
- `~/Documents/personal/beads/internal/storage/sqlite/store.go:206-217` - Close() checkpoint logic
- `~/Documents/personal/beads/cmd/bd/main.go` - CLI entry point, daemon auto-start

**Related Models:**
- `.kb/models/beads-integration-architecture/model.md` - RPC vs direct mode, client design

**Backups:**
- `.beads/backup-corrupted-2026-01-21/` - First corruption backup
- `.beads/backup-corrupted-2026-01-21-1444/` - Second corruption backup
- `.beads/backup-corrupted-2026-01-22/` - Third corruption backup (mislabeled)

**Primary Evidence (Verify These):**
- `~/Documents/personal/beads/internal/storage/sqlite/store.go:206-217` - WAL checkpoint implementation showing TRUNCATE mode
- `~/Documents/personal/beads/cmd/bd/main.go` - Daemon auto-start logic
- `~/Documents/personal/beads/internal/storage/sqlite/` - SQLite storage implementation with WAL mode
- `.beads/daemon.log` - Historical (file no longer exists — daemon doesn't run with JSONL-only default, which is the expected healthy state)

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-05-inv-design-orchestrator-session-lifecycle-without.md
- .kb/investigations/archived/2025-12-26-inv-spawn-context-includes-invalid-beads.md
- .kb/investigations/archived/2025-12-19-inv-set-beads-issue-status-progress.md
- .kb/investigations/archived/2026-01-08-inv-handle-multiple-agents-same-beads.md
- .kb/investigations/archived/2025-12-25-inv-beads-database-pollution-orch-go.md
- .kb/investigations/archived/2025-12-24-inv-follow-up-ecosystem-audit-orch.md
