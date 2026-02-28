## Summary (D.E.K.N.)

**Delta:** The existing PID lock (added Feb 24) has a TOCTOU race condition in its read-check-write pattern and lacks daemon stop/restart commands, making singleton enforcement unreliable under concurrent startup.

**Evidence:** Code review of pidlock.go shows non-atomic read→check→write→verify pattern. Between checking if a PID is alive and writing the new PID, another process can do the same. The "verify read-back" step narrows but doesn't close the race window. No `orch daemon stop` or `orch daemon restart` commands exist. The incident (two daemons from different days) matches the "old binary without PID lock" or "nohup restart without killing old" failure modes.

**Knowledge:** flock(2) is the correct Unix primitive for daemon singleton enforcement — kernel releases the lock on process exit (even crash), eliminating both the TOCTOU race and stale lock cleanup problems. PID file should be kept as a secondary artifact for status reporting, not as the primary lock mechanism.

**Next:** Three implementation issues: (1) Replace read-check-write with flock as primary lock, (2) Add `orch daemon stop/restart` commands, (3) Add `--replace` flag for graceful takeover.

**Authority:** architectural - Cross-component impact (daemon startup, status reporting, CLI commands) with multiple valid approaches requiring synthesis.

---

# Investigation: Design Singleton Enforcement for Orch Daemon

**Question:** How should we make daemon singleton enforcement robust against all failure modes — TOCTOU races, stale locks from crashes, and accidental re-starts via nohup/launchd?

**Defect-Class:** race-condition

**Started:** 2026-02-27
**Updated:** 2026-02-27
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None - implementation issues created
**Status:** Complete

**Patches-Decision:** N/A (new enforcement mechanism)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/daemon-autonomous-operation/probes/2026-02-24-probe-daemon-single-instance-pid-lock.md` | extends | Yes - read code, confirmed TOCTOU race | PID lock "works" but has race window |
| `.kb/investigations/2026-02-27-investigate-phantom-agent-spawns.md` | confirms | Yes - stale status file bug confirms cleanup gap | N/A |
| `.kb/decisions/2026-01-16-single-daemon-orchestration-home.md` | aligns | Yes - single daemon is the intended architecture | N/A (superseded by project-group-model) |

---

## Findings

### Finding 1: PID Lock exists but has a TOCTOU race

**Evidence:** The current `AcquirePIDLockAt()` in `pkg/daemon/pidlock.go:44-87` uses this sequence:
1. Read PID file (line 52)
2. Parse PID and check if alive via `kill(pid, 0)` (lines 56-57)
3. Write current PID to file (line 70)
4. Read back and verify (lines 75-81)

Between step 2 (process dead) and step 3 (write PID), another daemon process can perform the same steps. Both see no running daemon, both write their PID. The verify step (4) only catches the case where both writes overlap — if writes are sequential (likely), the second writer wins and the first silently believes it has the lock.

**Source:** `pkg/daemon/pidlock.go:44-87`

**Significance:** This is the exact failure mode that allows multiple daemons. Even though the window is small (microseconds), `nohup orch daemon run &` invoked twice in quick succession can hit it. More commonly, the old daemon was from a pre-PID-lock binary.

---

### Finding 2: No daemon stop or restart commands

**Evidence:** Searched `cmd/orch/daemon.go` for stop/restart/kill commands. Only found:
- `daemon run` — start daemon
- `daemon once` — process single issue
- `daemon preview` — dry run
- `daemon reflect` — run kb reflect
- `daemon resume` — resume after verification pause

There is no `daemon stop` (to kill an existing daemon) or `daemon restart` (stop + start). The daemon guide recommends `launchctl kickstart -k` for launchd or manual `kill`. This means the only way to stop a `nohup` daemon is to find the PID and kill it manually.

**Source:** `cmd/orch/daemon.go:19-31` (command structure), `.kb/guides/daemon.md:260-270` (launchd control)

**Significance:** Without stop/restart commands, users accumulate daemon processes when they run `nohup orch daemon run` again without killing the old one. The PID lock error message tells you the old PID, but doesn't offer to kill it.

---

### Finding 3: Graceful shutdown exists but cleanup is defer-only

**Evidence:** The daemon registers signal handlers for SIGINT/SIGTERM (daemon.go:301-302) and uses `defer pidLock.Release()` (daemon.go:268) and `defer daemon.RemoveStatusFile()` (daemon.go:323). These only run on graceful shutdown. A crash (SIGKILL, OOM, power failure) leaves both files stale.

The PID lock handles stale files by checking `isProcessAlive()` on next startup. The status file's `ReadValidatedStatusFile()` also validates PID liveness. But neither is atomic — they can report inconsistent state during the window between process death and next startup.

**Source:** `cmd/orch/daemon.go:264-323`, `pkg/daemon/status.go:152-164`

**Significance:** flock(2) would eliminate this entire category of issues — kernel releases the lock automatically on process exit (any exit, including crash).

---

### Finding 4: flock(2) is available on macOS and Go supports it

**Evidence:** macOS supports `flock(2)` and Go exposes it via `syscall.Flock()`. The pattern is:
1. Open lock file with `O_CREATE|O_WRONLY`
2. `syscall.Flock(fd, LOCK_EX|LOCK_NB)` — non-blocking exclusive lock
3. If error → another process holds the lock
4. Write PID to the locked file (for status reporting)
5. Keep fd open for daemon lifetime
6. On process exit (any kind): kernel releases flock automatically

No stale file cleanup needed. No TOCTOU race possible. The flock is held as long as the file descriptor is open.

**Source:** `man 2 flock` (macOS/BSD), Go `syscall` package

**Significance:** This is the standard Unix daemon pattern. It eliminates both the TOCTOU race and the stale lock problem in one mechanism.

---

## Synthesis

**Key Insights:**

1. **flock replaces two problems with zero** — The current PID-file approach has two issues: TOCTOU race on acquisition and stale file cleanup on crash. flock eliminates both because the kernel manages the lock lifecycle. The PID file becomes a status artifact, not a locking mechanism.

2. **Stop/restart commands are the operational gap** — Even with perfect singleton enforcement, users need to manage the daemon lifecycle. Currently the only options are launchctl (if registered) or manual kill. Adding `orch daemon stop` and `orch daemon restart` completes the management interface.

3. **`--replace` flag enables zero-downtime transitions** — For `make install` workflows, the new binary needs to replace the old daemon. `--replace` (or `daemon restart`) should: send SIGTERM to old process, wait for graceful shutdown, then start.

**Answer to Investigation Question:**

Replace the read-check-write PID lock with `flock(2)` as the primary singleton mechanism. Keep the PID file for status reporting (write PID into the flock'd file). Add `orch daemon stop` and `orch daemon restart` commands for lifecycle management. Add `--replace` flag to `daemon run` for automated takeover during binary upgrades.

---

## Structured Uncertainty

**What's tested:**

- ✅ PID lock exists and is called on daemon startup (verified: `cmd/orch/daemon.go:264`)
- ✅ Stale PID detection works via `kill(pid, 0)` (verified: `pidlock_test.go` test suite passes)
- ✅ TOCTOU race exists in read-check-write pattern (verified: code review of `pidlock.go:52-81`)
- ✅ `flock(2)` is available via Go's `syscall.Flock()` (verified: Go stdlib docs)

**What's untested:**

- ⚠️ flock behavior across NFS (not relevant — `~/.orch/` is local filesystem)
- ⚠️ Exact timing of the TOCTOU race window (theoretical — hard to reproduce)
- ⚠️ Whether launchd respects flock when managing daemon lifecycle

**What would change this:**

- If `~/.orch/` were on a networked filesystem, flock semantics would be unreliable
- If macOS sandboxing prevented flock on `~/.orch/`, we'd need a different approach

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Replace PID lock with flock(2) | architectural | Changes daemon startup contract, affects all daemon consumers |
| Add daemon stop/restart | architectural | New CLI surface area, interacts with lifecycle management |
| Add --replace flag | implementation | Flag on existing command, clear behavior |

### Recommended Approach: flock(2) + PID file + lifecycle commands

**flock as primary, PID as secondary** — Use `syscall.Flock()` with `LOCK_EX|LOCK_NB` for atomic singleton enforcement. Write PID into the flock'd file for status reporting. Add stop/restart commands that read the PID and send signals.

**Why this approach:**
- Eliminates TOCTOU race (kernel-managed locking, not userspace)
- Eliminates stale lock cleanup (kernel releases on any process exit)
- Standard Unix daemon pattern (well-understood, battle-tested)
- PID file preserved for operational commands (stop, status)
- Minimal change surface (pidlock.go internals, no public API change)

**Trade-offs accepted:**
- flock(2) doesn't work on NFS — acceptable because `~/.orch/` is always local
- File descriptor must stay open for daemon lifetime — natural for a long-running process
- flock advisory locks can be bypassed by processes that don't acquire them — only the daemon command calls this

**Implementation sequence:**
1. **Issue 1: Replace PID lock internals with flock** — Change `AcquirePIDLockAt()` to use `flock(2)`. Return a `PIDLock` that holds the open fd. `Release()` closes the fd. Tests updated. This is the foundation.
2. **Issue 2: Add `orch daemon stop` and `orch daemon restart`** — `stop` reads PID from lock file, sends SIGTERM, waits up to 5s, then SIGKILL. `restart` is stop + run. Depends on Issue 1.
3. **Issue 3: Add `--replace` flag to `daemon run`** — When set, if flock fails (another daemon running), read PID, send SIGTERM, wait, re-acquire flock. Sugar for the common "just replace the old one" workflow. Depends on Issue 2.

### Alternative Approaches Considered

**Option B: Unix domain socket as lock**
- **Pros:** Can serve as IPC channel for daemon commands, atomically freed on exit
- **Cons:** More complex, requires socket management, overkill for singleton enforcement. IPC is not currently needed — daemon communicates via status file.
- **When to use instead:** If daemon needed rich IPC (query state, push commands) beyond stop/restart

**Option C: Enhanced PID lock with retry (keep current approach)**
- **Pros:** No new dependencies, already implemented
- **Cons:** TOCTOU race remains (narrowed but not eliminated), stale file cleanup still needed
- **When to use instead:** If flock is unavailable (not the case on macOS)

**Rationale for recommendation:** flock solves both problems (TOCTOU + stale cleanup) with less code than the current approach. It's the standard solution for this class of problem.

---

### Implementation Details

**What to implement first:**
- Issue 1 (flock) is foundational — everything else depends on it
- The change is internal to `pkg/daemon/pidlock.go` — public API stays the same
- `PIDLock` struct gains an `*os.File` field (the open, flock'd file descriptor)
- `AcquirePIDLockAt()` opens file → flock → write PID → return lock
- `Release()` closes the file (which releases flock) → removes file

**Implementation shape for flock replacement:**
```go
type PIDLock struct {
    path string
    pid  int
    file *os.File // held open to maintain flock
}

func AcquirePIDLockAt(lockPath string) (*PIDLock, error) {
    dir := filepath.Dir(lockPath)
    os.MkdirAll(dir, 0755)

    f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, fmt.Errorf("failed to open PID lock file: %w", err)
    }

    // Non-blocking exclusive lock
    err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
    if err != nil {
        f.Close()
        // Read existing PID for error message
        existingPID := readPIDFromFile(lockPath)
        return nil, fmt.Errorf("%w: PID %d", ErrDaemonAlreadyRunning, existingPID)
    }

    // Write our PID (truncate first)
    f.Truncate(0)
    f.Seek(0, 0)
    fmt.Fprintf(f, "%d", os.Getpid())
    f.Sync()

    return &PIDLock{path: lockPath, pid: os.Getpid(), file: f}, nil
}

func (l *PIDLock) Release() error {
    if l == nil || l.file == nil {
        return nil
    }
    l.file.Close() // releases flock
    os.Remove(l.path)
    return nil
}
```

**Things to watch out for:**
- ⚠️ Don't close the file descriptor accidentally (e.g., via GC of `*os.File`) — store it in the PIDLock struct
- ⚠️ `daemon once` does NOT acquire the PID lock (by design — it's a one-shot command)
- ⚠️ Tests need to account for flock behavior (test processes can flock temp files)
- ⚠️ The `--replace` flag's SIGTERM→wait→SIGKILL sequence needs a timeout to avoid hanging

**Areas needing further investigation:**
- None — this is a well-understood Unix pattern

**Success criteria:**
- ✅ Starting two `orch daemon run` concurrently: second fails immediately with clear error
- ✅ Kill daemon with SIGKILL: next `orch daemon run` starts successfully (no stale lock)
- ✅ `orch daemon stop` kills running daemon by PID
- ✅ `orch daemon restart` replaces running daemon cleanly
- ✅ `orch daemon run --replace` replaces existing daemon without separate stop
- ✅ All existing pidlock_test.go tests pass (updated for flock)

---

## References

**Files Examined:**
- `pkg/daemon/pidlock.go` — Current PID lock implementation (TOCTOU race identified)
- `pkg/daemon/pidlock_test.go` — Test coverage for PID lock (8 tests, all passing)
- `pkg/daemon/status.go` — Status file management, ReadValidatedStatusFile
- `cmd/orch/daemon.go:255-780` — Daemon run command, signal handling, status writing

**Commands Run:**
```bash
# Check existing daemon PID
cat ~/.orch/daemon.pid  # 73639

# Check daemon status file
cat ~/.orch/daemon-status.json  # Running, PID 73639

# Check running daemon processes
ps aux | grep "orch.*daemon"  # One daemon (PID 73639)

# Check launchd registration
launchctl list | grep orch  # Not registered
```

**Related Artifacts:**
- **Probe:** `.kb/models/daemon-autonomous-operation/probes/2026-02-24-probe-daemon-single-instance-pid-lock.md` — Original PID lock implementation and verification
- **Decision:** `.kb/decisions/2026-01-16-single-daemon-orchestration-home.md` — Single daemon architecture (superseded but principle holds)
- **Guide:** `.kb/guides/daemon.md` — Daemon reference guide (needs update after implementation)

---

## Investigation History

**2026-02-27 22:50:** Investigation started
- Initial question: How to prevent multiple daemon processes from running concurrently?
- Context: Found PIDs 34172 (Tuesday) and 54135 (today) running simultaneously

**2026-02-27 22:55:** Found existing PID lock mechanism
- `pidlock.go` added Feb 24, 2026 — post-dates the duplicate incident
- TOCTOU race identified in read-check-write pattern

**2026-02-27 23:00:** Substrate consultation complete
- Principles: Gate Over Remind, Infrastructure Over Instruction
- Prior probe confirms PID lock works but doesn't address race
- No daemon stop/restart commands exist

**2026-02-27 23:05:** Investigation completed
- Status: Complete
- Key outcome: Recommend flock(2) replacement + daemon lifecycle commands
