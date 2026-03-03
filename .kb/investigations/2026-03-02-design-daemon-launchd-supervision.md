## Summary (D.E.K.N.)

**Delta:** The orch daemon has no process supervision — no com.orch.daemon.plist exists, so it only runs when manually started and dies on session end, terminal close, or interrupt. The Jan 10 decision punted because of tmux PATH issues with overmind, but the daemon itself (pure Go, no tmux dependency) is ideal for launchd.

**Evidence:** `ls ~/Library/LaunchAgents/com.orch.*` shows only `com.orch.token-keepalive.plist`. Daemon code uses `os/signal` for SIGINT/SIGTERM handling and `exec.Command("orch", "work", ...)` for spawning — no tmux in the daemon process itself. PATH solved: `~/.bun/bin` has symlinks for bd, kb, orch, go, tmux. Only `claude` is missing (at `~/.local/bin/claude`) but daemon doesn't need it directly — claude runs inside tmux windows which inherit user shell environment.

**Knowledge:** Daemon is the simplest launchd candidate in the stack: single Go binary, no tmux dependency, already has PID lock and signal handling, already has `make install-restart` target expecting launchd. The concurrency limit (5 agents) works fine for always-on — it's already designed for unattended operation. The interaction with orch-dashboard is already solved (daemon excluded by default, opt-in via `ORCH_DASHBOARD_START_DAEMON=1`).

**Next:** Create `~/Library/LaunchAgents/com.orch.daemon.plist` with KeepAlive, set PATH to include `~/.bun/bin`, add `claude` symlink to `~/.bun/bin` for completeness. Update daemon guide.

**Authority:** architectural - Cross-component decision affecting daemon lifecycle, dashboard interaction, and deployment workflow.

---

# Investigation: Daemon launchd Supervision Architecture

**Question:** Should the orch daemon be a launchd KeepAlive service, and what are the implications for concurrency limits, dashboard interaction, and PATH?

**Defect-Class:** configuration-drift

**Started:** 2026-03-02
**Updated:** 2026-03-02
**Owner:** orch-go architect agent
**Phase:** Complete
**Next Step:** Implementation (create plist, add claude symlink, update guide)
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-01-10-launchd-supervision-architecture.md`

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `2026-01-10-launchd-supervision-architecture.md` (decision) | extends | Yes - re-read; the tmux PATH issue was specific to overmind, not daemon | No conflict - decision was about overmind supervision, not daemon |
| `2026-01-15-inv-set-up-daemon-launchd-plist.md` | extends | Yes - plist was created and worked, but appears to have been removed since | No conflict - that investigation proved it works |
| `2026-02-09-dashboard-restart-daemon-autostart-default-disabled.md` (probe) | confirms | Yes - dashboard correctly excludes daemon by default | None |

---

## Findings

### Finding 1: The daemon has zero tmux dependency in its own process

**Evidence:** `pkg/daemon/daemon.go` shows the daemon is a pure Go polling loop. It shells out to `exec.Command("orch", "work", beadsID)` to spawn agents. The tmux involvement happens downstream: `orch work` → `orch spawn` → creates tmux window → sends `claude` command into tmux. The daemon process itself never touches tmux directly.

**Source:** `pkg/daemon/daemon.go`, `pkg/daemon/issue_adapter.go:352-367`, `pkg/spawn/claude.go:71-181`

**Significance:** The Jan 10 decision's reason for punting launchd ("tmux PATH issues") does not apply to the daemon. The daemon only needs PATH access to `orch` and `bd` (both symlinked in `~/.bun/bin`). The `claude` CLI is invoked inside tmux windows which inherit the user's interactive shell environment, not the daemon's launchd environment.

---

### Finding 2: The daemon already expects launchd supervision

**Evidence:**
- `Makefile` has `install-restart` target that runs `launchctl kickstart -k gui/$(id -u)/com.orch.daemon`
- `make install` prints hint: "The orch daemon may need restart: launchctl kickstart -k"
- Daemon guide (`.kb/guides/daemon.md:274-310`) documents launchd plist configuration with full XML example
- Jan 15 investigation proved the plist works (showed PID 63004 running under launchd)

**Source:** `Makefile:53-56`, `.kb/guides/daemon.md:274-310`, `.kb/investigations/archived/2026-01-15-inv-set-up-daemon-launchd-plist.md`

**Significance:** The codebase already assumes launchd is the daemon's supervisor. The plist just doesn't exist on disk. This is a configuration gap, not an architectural gap.

---

### Finding 3: PATH is already solved via ~/.bun/bin symlinks

**Evidence:** Current symlinks in `~/.bun/bin`:
- `bd` → `~/bin/bd` ✅
- `kb` → `~/Documents/personal/kb-cli/build/kb` ✅
- `orch` → `~/bin/orch` ✅
- `go` → `/opt/homebrew/bin/go` ✅
- `tmux` → `/opt/homebrew/bin/tmux` ✅

Missing: `claude` (at `~/.local/bin/claude` → `~/.local/share/claude/versions/2.1.63`)

The daemon's direct dependencies are `orch` and `bd`. Both present in `~/.bun/bin`. The `claude` CLI is NOT a direct dependency (runs in tmux), but adding the symlink would make the full tool chain available from launchd's PATH for any future use.

**Source:** `ls -la ~/.bun/bin/{bd,kb,orch,go}`, `which claude`, `pkg/daemon/issue_adapter.go:361`

**Significance:** Setting `PATH=/Users/dylanconlin/.bun/bin:/usr/local/bin:/usr/bin:/bin` in the launchd plist gives the daemon access to all required tools. No wrapper scripts or environment hacks needed.

---

### Finding 4: Concurrency limit is already designed for always-on unattended operation

**Evidence:**
- Default `max_agents: 5` in `~/.orch/config.yaml`
- WorkerPool with semaphore-based slot management and OpenCode reconciliation
- 6-layer spawn dedup prevents duplicate spawns
- Completion polling auto-frees slots when agents finish
- Daemon already runs overnight for batch processing (per daemon guide)
- Verification pause (N completions without human review → pause spawning) provides safety valve

**Source:** `pkg/daemon/pool.go`, `pkg/daemon/daemon.go`, `.kb/guides/daemon.md:149-230`

**Significance:** The concurrency limit does NOT need rethinking for always-on operation. It's already designed for exactly this: poll, spawn within capacity, auto-complete, free slots, repeat. The verification pause provides human oversight even when daemon is unsupervised.

---

### Finding 5: Dashboard interaction is cleanly separated

**Evidence:**
- Feb 9 probe confirmed: `orch-dashboard restart` does NOT auto-start daemon (disabled by default)
- Opt-in via `ORCH_DASHBOARD_START_DAEMON=1` env var
- Procfile includes daemon entry but `--can-die opencode,daemon` means dashboard survives daemon restarts
- Dashboard manages API (orch serve), Web UI, OpenCode; daemon is independent lifecycle

**Source:** `~/bin/orch-dashboard` (lines 214-284), `.kb/models/daemon-autonomous-operation/probes/2026-02-09-dashboard-restart-daemon-autostart-default-disabled.md`

**Significance:** launchd daemon and orch-dashboard are non-conflicting. Dashboard controls dashboard services. Launchd controls daemon. If both run daemon (dashboard with opt-in + launchd), the PID lock (`daemon.AcquirePIDLock()`) prevents dual instances. No coordination needed.

---

### Finding 6: build/orch vs ~/bin/orch — the binary path question

**Evidence:**
- Prior decision and kb constraint: "Use build/orch for serve daemon — Prevents SIGKILL during make install"
- `make install` creates symlink: `~/bin/orch` → `build/orch`
- Current `make install-restart` runs `launchctl kickstart` which restarts with new binary
- The `orch` symlink in `~/.bun/bin` points to `~/bin/orch` → `build/orch`

**Source:** `Makefile:39-50`, `.kb/investigations/archived/2026-01-15-inv-set-up-daemon-launchd-plist.md`

**Significance:** The ProgramArguments should use `~/bin/orch` (the symlink) rather than hardcoding `build/orch`. The symlink always resolves to the latest build output. After `make install`, `launchctl kickstart -k` restarts the daemon with the new binary. This is simpler than the earlier `build/orch` approach because the symlink indirection means the binary path never changes, and `kickstart -k` provides a clean restart.

---

## Synthesis

**Key Insights:**

1. **The Jan 10 decision's blocker doesn't apply to daemon** — The tmux PATH propagation issue was specific to overmind (which uses `tmux -C` control mode). The daemon is a pure Go process that shells out to `orch` and `bd` — both already in `~/.bun/bin`. No tmux in the daemon's process tree.

2. **This is a configuration gap, not an architectural gap** — The codebase already assumes launchd supervision (Makefile `install-restart` target, daemon guide plist documentation, Jan 15 investigation proving it works). The plist file simply doesn't exist on disk.

3. **All five concerns from the task are already resolved:**
   - **KeepAlive?** Yes — daemon should auto-restart on crash/exit
   - **Concurrency limit?** No change needed — already designed for always-on
   - **Dashboard interaction?** Already separated — PID lock prevents conflicts
   - **PATH?** Solved via `~/.bun/bin` symlinks (just need `claude` symlink for completeness)
   - **Binary path?** Use `~/bin/orch` (symlink to build/orch)

**Answer to Investigation Question:**

Yes, the daemon should be a launchd KeepAlive service. There are no architectural blockers. The concurrency limit works as-is for always-on operation (verification pause provides safety valve). Dashboard interaction is cleanly separated (PID lock handles overlap). PATH is solved via existing `~/.bun/bin` symlinks. The only implementation work is: (1) create the plist file, (2) add `claude` symlink to `~/.bun/bin`, (3) load via launchctl.

---

## Structured Uncertainty

**What's tested:**

- ✅ Daemon has no tmux dependency in its process (verified: traced code from daemon.go → issue_adapter.go → SpawnWork uses exec.Command("orch"))
- ✅ `~/.bun/bin` has symlinks for bd, kb, orch, go, tmux (verified: ls -la)
- ✅ Dashboard excludes daemon by default (verified: Feb 9 probe with before/after evidence)
- ✅ PID lock prevents dual daemon instances (verified: code at daemon.go:349-356)
- ✅ Jan 15 investigation proved launchd plist works (verified: showed PID and logs)
- ✅ `make install-restart` already expects launchd (verified: Makefile:53-56)

**What's untested:**

- ⚠️ Whether the current `~/bin/orch` → `build/orch` symlink survives `make clean` (would break launchd daemon)
- ⚠️ Whether `KeepAlive` + daemon's own crash recovery interact poorly (daemon catches panics — does launchd see this as "still running"?)
- ⚠️ Whether cross-project daemon (`--cross-project`) works correctly under launchd's working directory constraints
- ⚠️ Whether `BEADS_NO_DAEMON=1` env var is still needed (prevents beads CLI from auto-starting its own daemon)

**What would change this:**

- If daemon started using tmux directly (not via orch work), PATH would need tmux in launchd env
- If `make clean` removes `build/orch`, launchd daemon would fail to start until next build
- If dashboard's `ORCH_DASHBOARD_START_DAEMON=1` became the default, would need coordination with launchd

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Create launchd plist for daemon | architectural | Cross-component: affects daemon lifecycle, Makefile workflow, dashboard interaction |
| Add claude symlink to ~/.bun/bin | implementation | Single-scope, reversible, no cross-boundary impact |
| Update daemon guide | implementation | Documentation update, no behavioral change |

### Recommended Approach ⭐

**launchd KeepAlive with minimal plist** — Create `~/Library/LaunchAgents/com.orch.daemon.plist` with KeepAlive, PATH pointing to `~/.bun/bin`, and WorkingDirectory set to orch-go project root.

**Why this approach:**
- Zero new architecture — follows pattern already documented in daemon guide and Makefile
- Daemon designed for exactly this — has PID lock, signal handling, verification pause
- PATH already solved — `~/.bun/bin` symlinks cover all daemon dependencies
- Unblocks overnight autonomous operation without manual babysitting

**Trade-offs accepted:**
- Daemon runs always-on (burning minimal CPU on 60s poll loop) — acceptable given it's a lightweight Go process
- `make clean` could break daemon until next build — acceptable, `make clean` is rare and daemon auto-restarts after rebuild

**Plist specification:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.orch.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/dylanconlin/bin/orch</string>
        <string>daemon</string>
        <string>run</string>
        <string>--verbose</string>
    </array>
    <key>WorkingDirectory</key>
    <string>/Users/dylanconlin/Documents/personal/orch-go</string>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/Users/dylanconlin/.orch/daemon.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/dylanconlin/.orch/daemon.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/Users/dylanconlin/.bun/bin:/Users/dylanconlin/bin:/Users/dylanconlin/.local/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin</string>
        <key>HOME</key>
        <string>/Users/dylanconlin</string>
        <key>BEADS_NO_DAEMON</key>
        <string>1</string>
    </dict>
</dict>
</plist>
```

**Design decisions in this plist:**

1. **`/Users/dylanconlin/bin/orch`** (not `build/orch`) — the symlink always resolves to latest build. After `make install`, `launchctl kickstart -k` restarts with new binary.

2. **PATH includes `~/.bun/bin` first** — gives daemon access to bd, kb, orch, go, tmux. Also includes `~/.local/bin` for `claude` (though daemon doesn't use it directly, spawned `orch work` processes inherit this PATH).

3. **`BEADS_NO_DAEMON=1`** — prevents beads CLI calls from auto-starting beads daemon processes. The orch daemon makes many `bd` calls; without this, each could spawn a beads daemon.

4. **`WorkingDirectory` = orch-go root** — required for beads integration (`.beads/` directory lookup) and cross-project support.

5. **No `--max-agents` or `--poll-interval` flags** — uses defaults (5 agents, 60s poll). Can be added later if needed.

6. **`--verbose`** — ensures daemon.log has enough detail for debugging.

**Implementation sequence:**
1. Add `claude` symlink to `~/.bun/bin`: `ln -sf ~/.local/bin/claude ~/.bun/bin/claude`
2. Create plist file at `~/Library/LaunchAgents/com.orch.daemon.plist`
3. Load: `launchctl load ~/Library/LaunchAgents/com.orch.daemon.plist`
4. Verify: `launchctl list | grep orch.daemon` (should show PID and status 0)
5. Verify logs: `tail -f ~/.orch/daemon.log` (should show poll output)
6. Test restart: `make install-restart` (should cleanly restart daemon with new binary)

### Alternative Approaches Considered

**Option B: Overmind-only (no launchd)**
- **Pros:** Simpler (fewer moving parts), all services managed together
- **Cons:** Daemon dies when overmind stops, requires manual `orch-dashboard start` after reboot, orch-dashboard already excludes daemon by default
- **When to use instead:** If daemon should only run during active development sessions, not overnight/persistently

**Option C: systemd-style custom supervisor**
- **Pros:** More control over restart policy, health checks
- **Cons:** Reinventing launchd on macOS, unnecessary complexity
- **When to use instead:** Never on macOS. Consider for future VPS deployment (Linux systemd).

**Option D: Daemon self-daemonizes (fork + setsid)**
- **Pros:** No external supervisor needed
- **Cons:** No auto-restart on crash, no login-session integration, harder to manage
- **When to use instead:** Never — launchd is the macOS standard for exactly this purpose.

**Rationale for recommendation:** The daemon is already designed for launchd supervision (Makefile targets, guide documentation, PID lock, signal handling). The only thing missing is the plist file itself. Options B/C/D all add complexity or reduce reliability compared to the standard macOS approach.

---

### Implementation Details

**What to implement first:**
- Create the plist file (the only actual blocker)
- Add `claude` symlink (2-second task, completes the PATH story)

**Things to watch out for:**
- ⚠️ If overmind is running with daemon opt-in (`ORCH_DASHBOARD_START_DAEMON=1`), launchd daemon will fail to start due to PID lock. This is correct behavior — just use one or the other.
- ⚠️ After macOS updates, LaunchAgents may need re-loading (`launchctl load`)
- ⚠️ `make clean` removes `build/orch`, breaking the `~/bin/orch` symlink. Daemon will fail until `make build && make install`. KeepAlive will keep retrying, so it auto-recovers after rebuild.
- ⚠️ Daemon log file (`~/.orch/daemon.log`) will grow indefinitely. Consider adding log rotation (newsyslog or periodic truncation).

**Areas needing further investigation:**
- Log rotation strategy for `~/.orch/daemon.log` (out of scope for this decision)
- Whether cross-project daemon should be default or opt-in flag in plist (currently requires `--cross-project` flag)
- Whether verification pause threshold should be configurable in plist args

**Success criteria:**
- ✅ `launchctl list | grep orch.daemon` shows running PID
- ✅ `tail ~/.orch/daemon.log` shows poll cycles
- ✅ `kill -9 <daemon-pid>` followed by `launchctl list` shows new PID (KeepAlive restart)
- ✅ `make install-restart` cleanly restarts daemon with new binary
- ✅ Daemon survives terminal close, SSH disconnect, and system sleep/wake

---

## Decision Gate Guidance (if promoting to decision)

**Promote to decision: recommend-yes**

This extends the existing Jan 10 launchd supervision architecture decision with the daemon component that was originally deferred due to tmux PATH concerns.

**Suggested `blocks` keywords:**
- `daemon supervision`
- `daemon launchd`
- `daemon stalling`
- `daemon reliability`
- `always-on daemon`

---

## References

**Files Examined:**
- `.kb/decisions/2026-01-10-launchd-supervision-architecture.md` - Original decision that punted daemon supervision
- `.kb/guides/daemon.md` - Authoritative daemon reference (already documents plist pattern)
- `.kb/investigations/archived/2026-01-15-inv-set-up-daemon-launchd-plist.md` - Prior investigation proving plist works
- `pkg/daemon/daemon.go` - Daemon struct and Run loop (no tmux dependency)
- `pkg/daemon/issue_adapter.go:352-367` - SpawnWork shells out to `orch work`
- `pkg/spawn/claude.go:71-181` - Claude launch command built for tmux injection
- `cmd/orch/daemon.go:331-410` - runDaemonLoop: PID lock, signal handling, context
- `Makefile:53-56` - install-restart target already expects launchd
- `~/Library/LaunchAgents/com.orch.token-keepalive.plist` - Reference for working plist pattern

**Commands Run:**
```bash
# Check existing plists
ls ~/Library/LaunchAgents/com.orch.*

# Check PATH symlinks
ls -la ~/.bun/bin/{bd,kb,orch,go,claude}

# Check claude location
which claude

# Check launchd services
launchctl list | grep -E 'orch|opencode'
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-10-launchd-supervision-architecture.md` - Original decision this extends
- **Decision:** `.kb/decisions/2026-01-10-individual-launchd-services.md` - Pattern for individual service plists
- **Probe:** `.kb/models/daemon-autonomous-operation/probes/2026-02-09-dashboard-restart-daemon-autostart-default-disabled.md` - Confirms dashboard-daemon separation
- **Guide:** `.kb/guides/daemon.md` - Authoritative daemon reference

---

## Investigation History

**2026-03-02 17:50:** Investigation started
- Initial question: Daemon keeps stalling because it's not supervised by launchd
- Context: com.orch.daemon.plist doesn't exist, daemon only runs when manually started

**2026-03-02 18:10:** Key finding — tmux PATH issue doesn't apply to daemon
- Daemon is a pure Go process that shells out to `orch work`, no direct tmux dependency
- The Jan 10 decision's blocker was specific to overmind, not the daemon

**2026-03-02 18:20:** All five design forks navigated
- KeepAlive: yes (daemon is ideal candidate)
- Concurrency: no change needed (already designed for always-on)
- Dashboard: already separated (PID lock prevents conflicts)
- PATH: solved via ~/.bun/bin symlinks
- Binary: use ~/bin/orch symlink

**2026-03-02 18:30:** Investigation completed
- Status: Complete
- Key outcome: Create launchd plist — no architectural changes needed, daemon already designed for this
