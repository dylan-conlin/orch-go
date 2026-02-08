<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Auto-rebuild lock file mechanism only checked file existence, not whether the holding process was still running, causing permanent "rebuild already in progress" deadlock after process termination.

**Evidence:** Stale lock file from Jan 8 (PID 20214) persisted for 9 days; process 20214 no longer existed; all orch commands showed warning.

**Knowledge:** Unix lock files require PID validation using `kill -0` to detect stale locks; Go's `os: process already finished` error requires string matching, not just `syscall.ESRCH`.

**Next:** Fix applied and verified. Pattern should be propagated to beads CLI which has same bug.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural decision)

---

# Investigation: Auto-Rebuild Reports "Already In Progress" Deadlock

**Question:** Why does auto-rebuild report 'already in progress' even when no build is running?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Worker agent (og-debug-investigate-auto-rebuild-17jan-8f42)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Stale Lock File with Dead PID

**Evidence:**
- Lock file existed at `.autorebuild.lock` created Thu Jan 8 15:45:26 2026
- Contained PID 20214
- `ps aux | grep 20214` showed no such process running

**Source:**
- `ls -la /Users/dylanconlin/Documents/personal/orch-go/.autorebuild.lock`
- `cat /Users/dylanconlin/Documents/personal/orch-go/.autorebuild.lock`

**Significance:** The lock was orphaned 9 days ago when a build process was killed/crashed before releasing the lock.

---

### Finding 2: isRebuildInProgress Only Checked File Existence

**Evidence:** Original code at `cmd/orch/autorebuild.go:65-69`:
```go
func isRebuildInProgress(lockPath string) bool {
    _, err := os.Stat(lockPath)
    return err == nil
}
```

**Source:** `cmd/orch/autorebuild.go:65-69`

**Significance:** No validation that the process holding the lock was still running. Any orphaned lock file would permanently block rebuilds.

---

### Finding 3: Go Error Handling for Dead Process

**Evidence:** Testing `os.FindProcess(999999).Signal(0)` on macOS returns:
- Error string: `"os: process already finished"`
- Error type: `*errors.errorString`
- NOT `syscall.ESRCH`

**Source:** Test program run at `/tmp/test_signal.go`

**Significance:** Fix needed to handle both `syscall.ESRCH` and string-based error messages from Go's process API.

---

## Synthesis

**Key Insights:**

1. **Lock files need liveness validation** - A lock file is only valid if the process that created it is still running. File existence alone is insufficient.

2. **Go's process API wraps syscall errors** - On macOS/Unix, `process.Signal(0)` errors come wrapped as Go error strings, not raw syscall errors.

3. **Cross-process lock cleanup** - The fix safely removes stale locks without race conditions by only removing when the holding PID is confirmed dead.

**Answer to Investigation Question:**

The auto-rebuild reported "already in progress" because the `isRebuildInProgress()` function only checked if the lock file existed, not whether the process that created it was still running. A build process was killed on Jan 8 (9 days ago) without releasing its lock, and all subsequent rebuild attempts failed because the stale lock file remained.

---

## Structured Uncertainty

**What's tested:**

- Lock file with live PID (current process) correctly reports rebuild in progress
- Lock file with dead PID (999999) is detected as stale and cleaned up
- Lock file with invalid content (empty, "test", negative, zero) is cleaned up
- After fix: orch commands run without auto-rebuild warning

**What's untested:**

- EPERM scenario (process exists but belongs to different user)
- Windows behavior (different process signaling API)
- High-frequency concurrent rebuild attempts

**What would change this:**

- If Windows behaves differently, platform-specific code may be needed
- If EPERM case occurs in practice, may need to add timeout-based cleanup

---

## Implementation Recommendations

### Recommended Approach (Implemented)

**PID validation in isRebuildInProgress** - Read PID from lock file, check if process is still running using signal 0.

**Why this approach:**
- Non-invasive single-function change
- Standard Unix pattern for lock validation
- Automatic cleanup of stale locks

**Trade-offs accepted:**
- Slightly more complex than simple file existence check
- Platform-specific (Unix signal 0 behavior)

**Implementation sequence:**
1. Read lock file contents and parse PID
2. Use `os.FindProcess().Signal(0)` to check process liveness
3. Remove lock if process is dead, invalid PID, or file unreadable

---

## References

**Files Examined:**
- `cmd/orch/autorebuild.go` - Main auto-rebuild implementation
- `cmd/orch/autorebuild_test.go` - Unit tests

**Commands Run:**
```bash
# Check stale lock file
ls -la /Users/dylanconlin/Documents/personal/orch-go/.autorebuild.lock
cat /Users/dylanconlin/Documents/personal/orch-go/.autorebuild.lock
ps aux | grep 20214

# Run tests
go test -v ./cmd/orch/ -run "TestAutoRebuild|TestIsRebuild"

# Verify fix
orch version
orch status
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-24-inv-auto-rebuild-after-go-changes.md` - Original auto-rebuild implementation

---

## Investigation History

**2026-01-17 22:55:** Investigation started
- Initial question: Why does auto-rebuild report 'already in progress' when no build is running?
- Context: Spawned from beads issue orch-go-2ayrx

**2026-01-17 22:56:** Root cause identified
- Found stale lock file from Jan 8 with dead PID 20214
- isRebuildInProgress() only checked file existence

**2026-01-17 23:00:** Fix implemented and tested
- Modified isRebuildInProgress() to validate PID is still running
- Added 3 new test cases for stale lock detection

**2026-01-17 23:02:** Fix verified
- All tests pass (6/6 autorebuild tests)
- orch commands run without warning
- Stale lock cleanup confirmed via manual test
