# Session Synthesis

**Agent:** og-arch-daemon-single-instance-24feb-b05a
**Issue:** orch-go-1223
**Outcome:** success

---

## Plain-Language Summary

Added a PID file lock to the daemon so only one instance can run at a time. Before this fix, multiple `orch daemon run` processes could silently accumulate from different sessions (tmux windows, restarts, etc.) and fight over the status file and spawns — causing the daemon to appear running but not actually pick up work. Now the second invocation fails fast with a clear error message showing the existing daemon's PID, and stale PID files from crashed processes are automatically cleaned up.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

Key outcomes:
- Second `daemon run` invocation fails with `cannot start daemon: daemon already running: PID <N>`
- Stale PID files (from crashed daemons) are detected and overwritten
- PID lock is cleaned up on clean shutdown (SIGINT/SIGTERM)
- PID is included in status file for dashboard visibility

---

## Delta (What Changed)

### Files Created
- `pkg/daemon/pidlock.go` - PID lock mechanism with AcquirePIDLock/Release, stale detection via kill(pid,0)
- `pkg/daemon/pidlock_test.go` - 9 tests covering acquire, double-acquire, stale cleanup, release, nil safety

### Files Modified
- `cmd/orch/daemon.go` - Added PID lock acquisition at start of runDaemonLoop(), PID in status file
- `pkg/daemon/status.go` - Added PID field to DaemonStatus struct

---

## Evidence (What Was Observed)

- Before fix: no single-instance guard existed. Multiple daemons could start without error.
- After fix: second daemon fails with `cannot start daemon: daemon already running: PID 17650`
- `kill(pid, 0)` correctly detects stale PIDs on macOS (darwin)
- PID file cleanup via defer works through both normal exit and signal-based shutdown

### Tests Run
```bash
go test ./pkg/daemon/... -count=1 -timeout 60s
# PASS: ok github.com/dylan-conlin/orch-go/pkg/daemon 6.534s

go test ./cmd/orch/... -count=1 -timeout 60s
# PASS: ok github.com/dylan-conlin/orch-go/cmd/orch 3.273s

go build ./cmd/orch/ && go vet ./cmd/orch/
# No errors
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/daemon-autonomous-operation/probes/2026-02-24-probe-daemon-single-instance-pid-lock.md` - Documents the process-level dedup layer

### Decisions Made
- PID file approach over flock: PID file is more debuggable (can `cat` it), works cross-platform, and reports which process holds the lock
- Lock only `daemon run`, not `daemon once`: The continuous polling mode is where multi-instance conflicts occur. `once` is a manual one-shot that exits quickly.
- Stale detection via `kill(pid, 0)`: Standard Unix approach, handles both ESRCH (not found) and EPERM (exists but different user)

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Tests passing (9 new + all existing)
- [x] Probe file complete
- [x] Ready for `orch complete orch-go-1223`

No discovered work.

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-daemon-single-instance-24feb-b05a/`
**Probe:** `.kb/models/daemon-autonomous-operation/probes/2026-02-24-probe-daemon-single-instance-pid-lock.md`
**Beads:** `bd show orch-go-1223`
