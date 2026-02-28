# Session Synthesis

**Agent:** og-arch-design-singleton-enforcement-27feb-0fad
**Issue:** orch-go-9nbk
**Duration:** 2026-02-27 22:50 → 2026-02-27 23:10
**Outcome:** success

---

## TLDR

Designed flock(2)-based singleton enforcement for the orch daemon to replace the existing PID lock's TOCTOU race condition. The current `pidlock.go` uses a read-check-write pattern that can allow two daemons to start simultaneously. flock(2) eliminates both the race and stale lock cleanup by using kernel-managed file locking that's automatically released on process exit (even crashes). Also identified the operational gap: no `orch daemon stop` or `orch daemon restart` commands exist, forcing users to manually find and kill daemon processes.

---

## Plain-Language Summary

The orch daemon is supposed to run as a single instance, but the current guard (a PID file) has a timing bug: two daemons starting at nearly the same time can both believe they got the lock. The fix is to use `flock(2)`, a Unix kernel feature that provides atomic file locking — the operating system guarantees only one process can hold the lock, and automatically releases it when the process dies (even from a crash). This replaces a fragile userspace check with a kernel guarantee. Additionally, we need `orch daemon stop` and `orch daemon restart` commands because currently there's no built-in way to manage the daemon lifecycle.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace for acceptance criteria.

Key outcomes:
- Investigation file complete at `.kb/investigations/2026-02-27-inv-design-singleton-enforcement-orch-daemon.md`
- Three sequenced implementation issues created with dependencies
- Design addresses all four failure modes: TOCTOU race, stale locks, missing lifecycle commands, binary upgrade workflows

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-27-inv-design-singleton-enforcement-orch-daemon.md` — Full architectural investigation with design recommendation
- `.orch/workspace/og-arch-design-singleton-enforcement-27feb-0fad/SYNTHESIS.md` — This file
- `.orch/workspace/og-arch-design-singleton-enforcement-27feb-0fad/VERIFICATION_SPEC.yaml` — Verification criteria

### Beads Issues Created
- `orch-go-9054` (P1, triage:ready) — Replace PID lock with flock(2) for daemon singleton enforcement
- `orch-go-yv4c` (P2, triage:review) — Add orch daemon stop and daemon restart commands (blocked by 9054)
- `orch-go-y0bj` (P3, triage:review) — Add --replace flag to daemon run for graceful takeover (blocked by yv4c)

---

## Evidence (What Was Observed)

- Existing PID lock at `pkg/daemon/pidlock.go` uses read-check-write pattern with TOCTOU race (lines 52-81)
- PID lock was added Feb 24, 2026 (commit `a46656c8f`) — recent addition
- Stale PID detection works via `kill(pid, 0)` but requires process restart to trigger
- No `daemon stop` or `daemon restart` subcommands exist
- `flock(2)` is available on macOS via `syscall.Flock()` in Go
- Current daemon already handles SIGINT/SIGTERM gracefully (daemon.go:301-302)
- PID lock and status file cleanup only happen on graceful shutdown (defer blocks)

---

## Architectural Choices

### flock(2) over enhanced PID file
- **What I chose:** flock(2) as primary lock mechanism, PID file as secondary status artifact
- **What I rejected:** Enhanced PID file with retry/tighter timing, Unix domain socket
- **Why:** flock eliminates both problems (TOCTOU race + stale cleanup) with less code. Unix domain socket is overkill — no IPC needed beyond stop/restart.
- **Risk accepted:** flock doesn't work on NFS, but `~/.orch/` is always local

### Hard rejection by default, --replace for takeover
- **What I chose:** Default behavior refuses to start if another daemon running. `--replace` flag sends SIGTERM to old daemon.
- **What I rejected:** Always-replace (dangerous — could kill daemon mid-spawn), always-reject (frustrating for binary upgrades)
- **Why:** Safe default + explicit override follows principle of least surprise
- **Risk accepted:** `--replace` could interrupt a spawn in progress, but SIGTERM triggers graceful shutdown

---

## Knowledge (What Was Learned)

### Decisions Made
- Use flock(2) not enhanced PID file because it eliminates two categories of bugs at once
- Sequence: flock first → lifecycle commands second → replace flag third (each depends on previous)

### Constraints Discovered
- `daemon once` deliberately does NOT acquire PID lock (one-shot command, no conflict)
- File descriptor for flock must remain open for daemon lifetime — can't close and reopen

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + 3 implementation issues)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-9nbk`

---

## Unexplored Questions

- Whether `orch-dashboard` script should call `orch daemon restart` instead of raw process management
- Whether the daemon guide (`.kb/guides/daemon.md`) needs a "Lifecycle Management" section after implementation
- Whether `daemon status` (not `orch status`) should be a separate subcommand for quick daemon health check

*(Tracked as potential follow-up work, not blocking)*

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-singleton-enforcement-27feb-0fad/`
**Investigation:** `.kb/investigations/2026-02-27-inv-design-singleton-enforcement-orch-daemon.md`
**Beads:** `bd show orch-go-9nbk`
