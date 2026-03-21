# Decision: launchd Supervision Architecture

**Date:** 2026-01-10
**Status:** Accepted
**Enforcement:** context-only
**Context:** Dashboard Reliability Infrastructure (Epic orch-go-95vz4)

## Problem

Services need automatic restart on crash and self-healing for orphaned processes. Initial approach attempted to supervise overmind via launchd, but encountered tmux PATH issues.

## Context

**The launchd → overmind → tmux PATH problem:**
- Overmind uses `tmux -C` (control mode) to manage processes
- launchd sets `EnvironmentVariables.PATH` in plist
- But tmux control mode spawns with minimal environment
- PATH doesn't propagate through: launchd → overmind → tmux → processes
- Error: "overmind: Can't find tmux. Did you forget to install it?"

**What we tried:**
1. Setting PATH in launchd plist EnvironmentVariables - didn't work
2. Creating .env file with PATH - didn't work (overmind needs tmux BEFORE reading .env)
3. Symlink tmux to ~/.bun/bin (already in PATH) - didn't work

**Root cause:** Not a bug in any tool. It's the interaction pattern between launchd's environment isolation and tmux control mode's subprocess spawning.

## Decision

**Use hybrid supervision architecture:**

1. **orch doctor --daemon** - Supervised by launchd
   - Self-healing: kills orphaned processes (long-running `bd` processes)
   - Health monitoring: checks OpenCode, orch serve, Web UI every 30s
   - Auto-restart: launchd KeepAlive ensures daemon restarts on crash
   - No tmux dependency: simple Go process, no PATH issues
   - Plist: `~/Library/LaunchAgents/com.orch.doctor.plist`

2. **Overmind** - Run manually (no launchd supervision)
   - Service management: start/stop/restart api, web, opencode
   - Started manually: `overmind start -D` (or via shell init)
   - Stable: overmind itself doesn't crash, no supervision needed
   - Atomic deployment: `orch deploy` uses overmind restart

## Rationale

**Why not fix overmind launchd supervision?**
- Solving tmux PATH requires wrapper scripts or switching to foreman
- Adds complexity without clear benefit
- Overmind is stable (doesn't need KeepAlive)
- What we need is **service** health monitoring, not overmind supervision

**Why orch doctor --daemon is better:**
- Direct service monitoring (ports 4096, 3348, 5188)
- Kills orphans (overmind doesn't do this)
- Simple Go process (no tmux/environment issues)
- Easy to supervise via launchd (no PATH propagation needed)
- Already implemented as part of epic

**The key insight:**
Epic goal is "services stay healthy automatically," not "overmind supervised by launchd." The latter was a means to an end. We found a better means.

## Consequences

**Positive:**
- ✅ Self-healing without PATH complexity
- ✅ Orphan cleanup (doctor daemon kills long-running `bd` processes)
- ✅ Continuous health monitoring (30s polling)
- ✅ Auto-restart on daemon crash (launchd KeepAlive)
- ✅ All interventions logged (`~/.orch/doctor.log`)

**Negative:**
- ⚠️ Overmind must be started manually (not auto-started at login)
- ⚠️ If overmind crashes, services stay as orphans until manual restart
  - Mitigation: overmind is stable, crashes are rare
  - Mitigation: `orch deploy` restarts overmind atomically

**Trade-off accepted:**
Manual overmind start is acceptable because:
- Overmind doesn't crash (no supervision needed)
- Services auto-heal via doctor daemon
- `orch deploy` provides atomic restart when needed

## Implementation

**launchd plist created:** `~/Library/LaunchAgents/com.orch.doctor.plist`

```xml
<key>ProgramArguments</key>
<array>
    <string>~/bin/orch</string>
    <string>doctor</string>
    <string>--daemon</string>
</array>
<key>KeepAlive</key>
<true/>
<key>RunAtLoad</key>
<true/>
```

**Loaded and verified:**
```bash
$ launchctl list | grep orch.doctor
87329	0	com.orch.doctor

$ tail ~/.orch/doctor.log
[2026-01-10 08:06:01] kill_long_bd: PID 52798 success=true
```

## Success Metrics

**Epic criteria:** "Zero 'did you restart?' for 1 week"

With this architecture:
- Services monitored continuously (30s polling)
- Orphans killed automatically (bd processes > 10min)
- Daemon auto-restarts on crash (launchd)
- Deployment is atomic (`orch deploy`)

Testing begins 2026-01-10. Success = no manual intervention needed for service stability.

## Alternatives Considered

**1. Fix overmind launchd PATH**
- Wrapper script with hardcoded PATH
- Switch to foreman instead of overmind
- Rejected: complexity without benefit

**2. Supervise overmind via systemd (if on Linux)**
- Not applicable (macOS)

**3. Use tmuxinator instead of overmind**
- Doesn't solve the fundamental launchd → tmux PATH issue
- Overmind provides better process management (restart, logs)

**4. No supervision at all**
- Rejected: epic requires self-healing
- Manual restarts don't meet success criteria

## Related

- Epic: orch-go-95vz4 (Dashboard Reliability Infrastructure)
- Issue: orch-go-7ix6z (launchd PATH issue - closed as won't fix)
- Implementation: orch-go-axd33 (orch doctor --daemon)
- Plist: `~/Library/LaunchAgents/com.orch.doctor.plist`

## Review Date

2026-01-17 (one week) - Review if "zero 'did you restart?'" was achieved.
