# Session Synthesis

**Agent:** og-debug-daemon-status-readers-27feb-1c03
**Issue:** orch-go-4omh
**Outcome:** success

---

## Plain-Language Summary

After an unclean daemon shutdown (crash, SIGKILL), the `daemon-status.json` file persists with `"status": "running"`. All three status readers (`handleDaemon` API endpoint, `readDaemonStatus` CLI, and `handleVerificationStatus` API endpoint) trusted the file's existence to mean the daemon was alive, causing false "running" reports on the dashboard and in `orch status`. The fix adds PID liveness validation: since the daemon already writes its PID into the status file, readers now check if that PID is actually alive using `kill(pid, 0)`. If the process is dead, the stale file is ignored and daemon is correctly reported as not running.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace directory.

Key outcomes:
- `ReadValidatedStatusFile()` returns nil for stale files with dead PIDs
- `ReadValidatedStatusFile()` returns valid status for live daemon PIDs
- Backward compatible: PID=0 (old daemon versions) still works
- 4 new tests, all passing

---

## Delta (What Changed)

### Files Modified
- `pkg/daemon/pidlock.go` - Exported `IsProcessAlive()` wrapper around existing `isProcessAlive()`
- `pkg/daemon/status.go` - Added `ReadValidatedStatusFile()` that reads + validates PID liveness
- `pkg/daemon/status_test.go` - 4 new tests for validated status file reading
- `cmd/orch/serve_system.go` - `handleDaemon()` now uses `ReadValidatedStatusFile()`
- `cmd/orch/serve_verification.go` - `handleVerificationStatus()` now uses `ReadValidatedStatusFile()`
- `cmd/orch/status_cmd.go` - Added PID field to local DaemonStatus struct, added `daemon.IsProcessAlive()` check in `readDaemonStatus()`

---

## Evidence (What Was Observed)

- `handleDaemon()` at serve_system.go:398 set `resp.Running = true` purely based on `ReadStatusFile()` returning non-nil — no liveness check
- `readDaemonStatus()` at status_cmd.go:1308 returned the parsed struct directly — no liveness check
- `handleVerificationStatus()` at serve_verification.go:67 had the same pattern — third caller missed by the original bug report
- `DetermineStatus()` at status.go:162 has a staleness threshold (2×pollInterval) but was only used by the daemon writer, never by readers
- The daemon already writes PID into the status file (daemon.go:663: `PID: os.Getpid()`)
- `isProcessAlive()` already existed in pidlock.go using `syscall.Kill(pid, 0)` — just needed exporting

### Tests Run
```bash
go test -v -run "TestReadValidatedStatusFile|TestIsProcessAlive|TestDetermineStatus" ./pkg/daemon/
# PASS: 11 tests (0.023s)

go test ./pkg/daemon/...
# ok github.com/dylan-conlin/orch-go/pkg/daemon 10.240s
```

---

## Architectural Choices

### PID liveness check vs. staleness threshold
- **What I chose:** PID liveness via `kill(pid, 0)` — checks if daemon process is actually alive
- **What I rejected:** Using `DetermineStatus()` staleness threshold (2×pollInterval)
- **Why:** PID check is definitive. A paused daemon has legitimately old timestamps but is still alive — staleness threshold would incorrectly report it as dead. PID check works correctly for all daemon states (running, paused, stalled).
- **Risk accepted:** PID reuse — extremely unlikely within the relevant timeframe (seconds to minutes between daemon death and next status read)

### ReadValidatedStatusFile() vs modifying ReadStatusFile()
- **What I chose:** New `ReadValidatedStatusFile()` function alongside unchanged `ReadStatusFile()`
- **What I rejected:** Modifying `ReadStatusFile()` to always validate PID
- **Why:** `ReadStatusFile()` is a raw reader — some callers may want the raw data. Validation is a layer above reading. The daemon itself calls `ReadStatusFile()` internally within `ReadValidatedStatusFile()`.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Pre-existing build error (RunPreFlightChecks signature mismatch in rework_cmd.go and spawn_cmd.go) exists from concurrent agent activity — unrelated to this fix

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (4 new, all existing pass)
- [x] All three reader paths fixed
- [x] Ready for `orch complete orch-go-4omh`

---

## Unexplored Questions

- The pre-existing build error in `rework_cmd.go` and `spawn_cmd.go` (RunPreFlightChecks arg count mismatch) should be tracked separately — it was introduced by another agent modifying the function signature.

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-daemon-status-readers-27feb-1c03/`
**Beads:** `bd show orch-go-4omh`
