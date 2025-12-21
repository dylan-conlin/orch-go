## Summary (D.E.K.N.)

**Delta:** Verified that pointing the `orch serve` launchd service to `build/orch` prevents SIGKILL (exit 137) during `make install`.

**Evidence:** `orch serve` (PID 91733) remained running throughout `make build` and `make install`, and `orch spawn --tmux` worked correctly from both `build/orch` and `~/bin/orch`.

**Knowledge:** Decoupling long-running daemons from frequently updated binaries prevents `launchd` KeepAlive from interfering with active processes during binary replacement.

**Next:** Close the investigation and report success.

**Confidence:** Very High (95%) - Direct observation of process stability and successful command execution.

---

# Investigation: Test after plist fix

**Question:** Does the `orch` CLI work correctly after the `plist` fix?
**Status:** Complete

## Findings

### Finding 1: launchd plist points to build directory
**Evidence:** `~/Library/LaunchAgents/com.orch-go.serve.plist` correctly points to `/Users/dylanconlin/Documents/personal/orch-go/build/orch`.
**Source:** `cat ~/Library/LaunchAgents/com.orch-go.serve.plist`
**Significance:** This confirms the fix is in place, separating the daemon's binary from the one in `~/bin/orch`.

### Finding 2: Process stability during build/install
**Evidence:** The `orch serve` process (PID 91733) maintained the same start time (Sun Dec 21 03:02:48 2025) before and after running `make build` and `make install`.
**Source:** `ps -p 91733 -o lstart=`
**Significance:** This proves that overwriting `~/bin/orch` no longer triggers a `launchd` restart of the `serve` daemon, which was the cause of the SIGKILL issues.

### Finding 3: Successful spawn execution
**Evidence:** `orch spawn --tmux` succeeded when run from both the build directory and the install directory.
**Source:** 
- `orch spawn --tmux investigation "verification test"`
- `/Users/dylanconlin/bin/orch spawn --tmux investigation "test from bin"`
**Significance:** Confirms that the core functionality is working as expected with the new configuration.

---

## Test performed
**Test:** 
1. Load and start `com.orch-go.serve` via `launchctl`.
2. Run `orch spawn --tmux` to verify it works.
3. Run `make build` and `make install` while monitoring the `orch serve` PID and start time.
4. Run `orch spawn --tmux` again using the binary in `~/bin/orch`.

**Result:** 
1. `orch serve` stayed running throughout `make build` and `make install`.
2. `orch spawn --tmux` worked correctly in all cases.
3. No SIGKILL (exit 137) was encountered.

## Conclusion
The fix is effective. By pointing the `launchd` service to the `build/orch` binary, we've decoupled the long-running `serve` daemon from the `~/bin/orch` binary that is frequently updated via `make install`. This prevents `launchd` from interfering with active `orch` processes when the `~/bin` binary is replaced.

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED
