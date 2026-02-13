## Summary (D.E.K.N.)

**Delta:** `restartOrchServe()` now detects Overmind via `.overmind.sock` and uses `overmind restart api` instead of kill+nohup, preserving the dashboard process group.

**Evidence:** Tested: `overmind restart api` restarts api (PID changed 4815->6300) while web stayed alive (PID 4816 unchanged). API health check and web UI both confirmed healthy after restart.

**Knowledge:** Overmind tears down all processes NOT in `--can-die` when one exits. Since `api` isn't in `--can-die`, killing it directly kills the whole dashboard.

**Next:** Close - fix is implemented, tested, committed.

**Authority:** implementation - Bug fix within existing restartOrchServe function, no architectural changes.

---

# Investigation: orch complete kills dashboard

**Question:** Why does `orch complete` kill the entire dashboard, and how to fix it?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: restartOrchServe uses pgrep + SIGTERM to kill orch serve directly

**Evidence:** `complete_cmd.go:1422` - `pgrep -f "orch.*serve"` finds PIDs, then `kill -TERM` sends SIGTERM to each.

**Source:** `cmd/orch/complete_cmd.go:1417-1475`

**Significance:** When orch serve runs as Overmind's `api` process, SIGTERM kills the child. Overmind sees its child die and since `api` is NOT in `--can-die`, Overmind tears down all processes (web, opencode).

### Finding 2: Overmind `--can-die` controls cascade behavior

**Evidence:** `orch-dashboard` starts with `--can-die opencode,daemon` (line 256). Only opencode and daemon can die without cascading. `api` and `web` are NOT in the can-die list.

**Source:** `/Users/dylanconlin/bin/orch-dashboard:256`

**Significance:** This is the root cause of the cascade. Killing api triggers Overmind's full shutdown because api is a critical process.

### Finding 3: `.overmind.sock` is a reliable Overmind detection signal

**Evidence:** Socket created at `$ORCH_GO_DIR/.overmind.sock` when overmind starts (confirmed in orch-dashboard:209 cleanup logic). Absent when overmind isn't running (confirmed by testing).

**Source:** `orch-dashboard:209`, `/Users/dylanconlin/Documents/personal/orch-go/.overmind.sock`

**Significance:** Checking for this socket file is a clean, reliable way to determine if orch serve is under Overmind management.

---

## Structured Uncertainty

**What's tested:**

- `overmind restart api` restarts only the api process (verified: PID changed, web PID unchanged)
- API health endpoint responds after overmind restart (verified: curl https://localhost:3348/health returned ok)
- `go build` and `go vet` pass with the fix (verified: both clean)
- `.overmind.sock` present when overmind running, absent when not (verified both states)

**What's untested:**

- Full end-to-end `orch complete` flow (would need a real agent completion to trigger)
- Behavior when overmind socket is stale (socket exists but overmind died)

**What would change this:**

- If overmind changes socket location behavior in future versions
- If stale sockets become common (overmind crash without cleanup)

---

## References

**Files Examined:**
- `cmd/orch/complete_cmd.go:1417-1475` - restartOrchServe function (modified)
- `/Users/dylanconlin/bin/orch-dashboard` - Dashboard startup script (read for context)
- `Procfile` - Process definitions (api, web, opencode)

**Commands Run:**
```bash
# Verify overmind restart api works without killing web
overmind restart api  # SUCCESS - api PID changed, web PID unchanged

# Verify health after restart
curl -ks https://localhost:3348/health  # {"status":"ok"}

# Build verification
go build ./cmd/orch/  # clean
go vet ./cmd/orch/    # clean
```
