## Summary (D.E.K.N.)

**Delta:** Daemon SIGKILL is caused by slow shutdown — `defer runReflectionAnalysis()` on exit runs `kb reflect --global` with no timeout, and the main loop doesn't check for context cancellation between major operations, exceeding launchd's 5-second exit timeout.

**Evidence:** `launchctl print` showed `runs = 23`, `last terminating signal = Killed: 9`, `exit timeout = 5`. After fix: daemon exits in 23-25ms, no SIGKILL, clean restart.

**Knowledge:** launchd sends SIGTERM → waits ExitTimeOut → SIGKILL. The daemon's deferred cleanup and mid-cycle operations must complete within this window. Context gates between operations are essential for responsive shutdown.

**Next:** Fix implemented and verified. Commit changes.

**Authority:** implementation - Bug fix within existing patterns, no architectural changes

---

# Investigation: Daemon Repeatedly Killed with SIGKILL (-9)

**Question:** Why does the daemon process get SIGKILL (-9) instead of shutting down cleanly?

**Started:** 2026-03-27
**Updated:** 2026-03-27
**Owner:** orch-go-e0h25
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** daemon-autonomous-operation

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/archived/2025-12-21-inv-tmux-spawn-killed.md | related | yes | Different cause (tmux, not launchd) |
| .kb/investigations/archived/2026-01-15-inv-set-up-daemon-launchd-plist.md | extends | yes | Plist was set up correctly but missing ExitTimeOut |

---

## Findings

### Finding 1: launchd confirms SIGKILL with 5-second exit timeout

**Evidence:** `launchctl print gui/501/com.orch.daemon` showed:
- `runs = 23` — daemon killed and restarted 23 times
- `last terminating signal = Killed: 9` — confirmed SIGKILL
- `exit timeout = 5` — only 5 seconds before escalation to SIGKILL
- `forks = 835` — many child processes spawned

**Source:** `launchctl print gui/$(id -u)/com.orch.daemon`

**Significance:** The SIGKILL is from launchd, not the OOM killer or kernel code signing. launchd sends SIGTERM, waits 5 seconds, then sends SIGKILL if the process hasn't exited.

---

### Finding 2: Exit-time reflection analysis blocks shutdown with no timeout

**Evidence:** `cmd/orch/daemon.go:27-28`:
```go
if daemonReflect {
    defer runReflectionAnalysis(daemonVerbose)
}
```
`daemonReflect` defaults to `true` (daemon_commands.go:201). The plist sets `--reflect-issues=false` and `--reflect-open=false` but NOT `--reflect=false`. `runReflectionAnalysis` calls `daemon.RunAndSaveReflection()` which runs `exec.Command("kb", "reflect", "--global", "--format", "json")` with NO timeout (reflect.go:136-137). `kb reflect --global` takes ~2.5 seconds.

**Source:** `pkg/daemon/reflect.go:136-137`, `cmd/orch/daemon.go:27-28`, `cmd/orch/daemon_commands.go:201`

**Significance:** The reflection analysis runs on every daemon exit. Combined with mid-cycle operations that haven't completed, the total shutdown time exceeds 5 seconds.

---

### Finding 3: Main loop doesn't check context between major operations

**Evidence:** The main loop (`cmd/orch/daemon.go:35-186`) only checks `ctx.Done()` at:
- Top of loop (line 36)
- Circuit breaker backoff (line 90)
- At-capacity wait (line 142)
- End-of-cycle sleep (line 180)

Between these checks, long-running operations execute without context awareness:
- `ReconcileWithOpenCode()` — HTTP calls to OpenCode
- `runPeriodicTasks()` — runs 10+ subsystems with network calls
- `processDaemonCompletions()` — processes completions
- `ListReadyIssuesMultiProject()` — beads RPC calls
- `runWorkGraphAnalysis()` — analysis across issues

**Source:** `cmd/orch/daemon.go:49-147`

**Significance:** If SIGTERM arrives during an active cycle, the daemon must wait for ALL remaining operations to complete before the loop returns and deferred cleanup runs. This adds 5-15+ seconds on top of the reflection analysis.

---

## Synthesis

**Key Insights:**

1. **Two-layer blocking** — Both the deferred cleanup (reflection analysis) and the main loop (mid-cycle operations) contribute to slow shutdown. Either alone might fit within 5 seconds; together they guaranteed exceeding it.

2. **Plist misconfiguration** — `--reflect-issues=false` and `--reflect-open=false` were set, but `--reflect=false` was not, leaving exit-time reflection enabled by default.

3. **launchd ExitTimeOut was at default (5s)** — Too short for a daemon that runs network-calling subsystems. The plist should specify an explicit ExitTimeOut.

**Answer to Investigation Question:**

The daemon gets SIGKILL because its shutdown takes longer than launchd's 5-second exit timeout. Two factors: (1) `defer runReflectionAnalysis()` runs `kb reflect --global` on exit with no timeout (~2.5s), and (2) the main loop doesn't check for context cancellation between major operations, adding 5-15s of mid-cycle delay. launchd sends SIGTERM, waits 5s, then escalates to SIGKILL.

---

## Structured Uncertainty

**What's tested:**

- ✅ Daemon exits in 23-25ms after SIGTERM with fix applied (measured via PID monitoring)
- ✅ launchctl stop produces clean exit (no `last terminating signal` in `launchctl print`)
- ✅ Context gates catch shutdown between operations (tested during active cycle)
- ✅ `kb reflect --global` runs in ~2.5s (timed directly)
- ✅ pkg/daemon tests pass (22.6s, all green)

**What's untested:**

- ⚠️ Behavior when `kb reflect` is slow (>3s) — the 3s timeout should kill it, but not tested with artificially slow kb
- ⚠️ Mid-cycle SIGTERM during heavy network operations — tested during light cycle, not during heavy beads RPC
- ⚠️ Memory leak theory — not investigated since SIGKILL was confirmed from launchd, not OOM

**What would change this:**

- Finding would be wrong if system logs showed Jetsam kills (OOM) instead of launchd-initiated SIGTERM
- Context gates would be insufficient if a single operation takes >15s (new ExitTimeOut)

---

## References

**Files Examined:**
- `cmd/orch/daemon.go:13-187` — Main daemon loop, deferred cleanup, context gates
- `cmd/orch/daemon_handlers.go:282-306` — runReflectionAnalysis function
- `cmd/orch/daemon_commands.go:156-206` — Flag definitions and defaults
- `cmd/orch/daemon_loop.go:47-179` — Signal handling setup
- `pkg/daemon/reflect.go:120-180` — RunReflection exec.Command with no timeout
- `~/Library/LaunchAgents/com.orch.daemon.plist` — launchd configuration

**Commands Run:**
```bash
# Confirmed SIGKILL source
launchctl print gui/501/com.orch.daemon

# Measured kb reflect timing
time kb reflect --global --format json

# Verified clean shutdown after fix
launchctl kill SIGTERM gui/501/com.orch.daemon

# Measured shutdown timing
# Result: 23-25ms
```

**Related Artifacts:**
- **Decision:** Prior decision "Use build/orch for serve daemon - Prevents SIGKILL during make install" — related but different mechanism
