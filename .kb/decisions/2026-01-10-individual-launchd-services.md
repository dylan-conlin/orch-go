# Decision: Individual launchd Services (Not Overmind Supervision)

**Date:** 2026-01-10
**Status:** Accepted
**Context:** Dashboard Reliability Infrastructure (Epic orch-go-95vz4)

## Problem

Services need automatic restart on crash to meet success criteria: "zero 'did you restart?' for 1 week."

## Attempted Solution: Overmind + launchd

Initial approach tried to supervise overmind via launchd, but encountered circular dependency:
- Overmind needs tmux for session management
- launchd can't find tmux in PATH despite EnvironmentVariables configuration
- tmux control mode (-C) has minimal environment, PATH doesn't propagate
- Without overmind, services can't auto-restart
- Without launchd, overmind isn't supervised

**Debugging invested:** ~1000 lines of PATH troubleshooting, .env files, wrapper scripts.

**Strategic failure:** Debugged the obstacle instead of questioning the premise.

## Actual Solution: Individual launchd Plists

Each service runs directly under launchd supervision:

### Architecture

```
launchd (macOS native supervisor)
├── com.opencode.serve → opencode serve --port 4096
├── com.orch.serve → orch serve
├── com.orch.web → ~/.orch/start-web.sh (bun run dev)
└── com.orch.doctor → orch doctor --daemon (monitoring)
```

### Plists Created

**OpenCode:** `~/Library/LaunchAgents/com.opencode.serve.plist`
- Direct execution: `/Users/dylanconlin/.bun/bin/opencode serve --port 4096`
- KeepAlive: true (auto-restart on crash)
- RunAtLoad: true (start at login)

**orch serve:** `~/Library/LaunchAgents/com.orch.serve.plist`
- Direct execution: `/Users/dylanconlin/bin/orch serve`
- KeepAlive: true
- RunAtLoad: true

**web:** `~/Library/LaunchAgents/com.orch.web.plist`
- Wrapper script: `~/.orch/start-web.sh` (uses real bun path `/opt/homebrew/bin/bun`)
- KeepAlive: true
- RunAtLoad: true
- Note: Required wrapper because bun needs working directory set

**orch doctor:** `~/Library/LaunchAgents/com.orch.doctor.plist`
- Monitoring daemon: `/Users/dylanconlin/bin/orch doctor --daemon`
- KeepAlive: true (daemon itself supervised)
- Polls services every 30s, kills orphans

### What This Gives Us

1. ✅ **Auto-restart on crash** - launchd's KeepAlive handles restart
2. ✅ **No tmux dependency** - each service runs directly
3. ✅ **Unified management** - `orch deploy` still works for atomic restart
4. ✅ **Monitoring** - `orch doctor --daemon` detects issues
5. ✅ **Observability** - `orch logs server/daemon`, health checks, event streaming

### Crash Recovery Testing

All three services tested and confirmed auto-restart:

**OpenCode:**
- Killed PID 33029
- launchd restarted as PID 17103 within 5s
- Health check passing

**orch serve:**
- Killed PID 80517
- launchd restarted as PID 18822 within 5s
- API responding

**Web UI:**
- Killed PID 18822
- launchd restarted as PID 34128 within 5s
- Dashboard accessible

## Why Not Overmind?

Overmind is still valuable for **development workflow**:
- Start/stop all services: `overmind start`
- View unified logs: `overmind echo`
- Restart individual services: `overmind restart api`

But **launchd supervision of overmind** created unnecessary complexity:
- tmux PATH propagation issues
- Circular dependency (need overmind for restart, need launchd for overmind)
- Additional layer of indirection

**Decision:** Run overmind manually when needed. Use launchd for production reliability.

## Trade-offs

**Individual launchd plists:**
- ✅ Simple, direct, no dependencies
- ✅ Native macOS supervision (no third-party tools)
- ✅ Each service independently managed
- ⚠️ More plists to maintain (3 services + 1 daemon = 4 plists)
- ⚠️ No unified "start all" command (but `orch deploy` handles restart)

**Overmind + launchd (rejected):**
- ✅ Unified service management
- ✅ Single plist
- ❌ tmux PATH issues
- ❌ Circular dependency
- ❌ Additional complexity layer

## Implementation

Created 4 launchd plists:
1. `com.opencode.serve.plist` - OpenCode server
2. `com.orch.serve.plist` - orch API server
3. `com.orch.web.plist` - Dashboard web UI
4. `com.orch.doctor.plist` - Self-healing daemon

Loaded via:
```bash
launchctl load ~/Library/LaunchAgents/com.opencode.serve.plist
launchctl load ~/Library/LaunchAgents/com.orch.serve.plist
launchctl load ~/Library/LaunchAgents/com.orch.web.plist
launchctl load ~/Library/LaunchAgents/com.orch.doctor.plist
```

Check status:
```bash
orch doctor  # Health check all services
launchctl list | grep -E "opencode|orch"  # launchd status
```

## Success Criteria Met

**Epic goal:** "Zero 'did you restart?' for 1 week"

**How this architecture achieves it:**
1. Services auto-restart on crash (launchd KeepAlive)
2. Doctor daemon monitors and kills orphans (no accumulation)
3. Atomic deployment prevents "running old binary" issues (orch deploy)
4. Cache invalidation prevents stale UI (X-Orch-Version headers)

**Testing period begins:** 2026-01-10 09:05

## Lessons

1. **Question premises before debugging obstacles** - 1000 lines of PATH debugging could have been avoided by questioning "do we need overmind supervision?" earlier
2. **Premise-skipping is a pattern** - When debugging an obstacle for >15 minutes, pause and evaluate alternatives
3. **Strategic perspective requires stepping back** - Can't see the frame from inside it
4. **Simple > Complex** - Individual launchd plists are simpler than overmind + launchd + tmux PATH propagation

## References

- Epic: orch-go-95vz4 (Dashboard Reliability Infrastructure)
- Session transcript: sess-4432.txt (overmind debugging session)
- Crash recovery tests: Confirmed 2026-01-10 09:04
