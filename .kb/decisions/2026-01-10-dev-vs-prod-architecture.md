---
stability: foundational
---
# Decision: Dev vs Prod Architecture Separation

**Date:** 2026-01-10
**Status:** Accepted
**Context:** Mac Development Environment Cleanup

## Problem

Mac was set up with production-style launchd supervision (auto-restart, monitoring daemon, individual service plists), but Mac is actually a development environment. This created unnecessary complexity.

## Decision

**Separate dev and prod architectures explicitly:**

### Development (Mac)

**Tool:** Overmind (simple process supervisor)

**Architecture:**
```
overmind (via Procfile)
├── api: orch serve
├── web: bun run dev
└── opencode: opencode serve --port 4096
```

**Characteristics:**
- ✅ Manual start/stop (no auto-restart)
- ✅ Simple Procfile configuration
- ✅ Unified logs with color coding
- ✅ Easy restart of individual services
- ✅ Hot reload for web changes (Vite)
- ✅ Standard tool (same as other dev projects)

**Management:**
```bash
overmind start -D  # Start all
overmind status    # Check status
overmind restart api  # Restart service
overmind quit      # Stop all
```

### Production (VPS - Future)

**Tool:** systemd (native Linux supervisor)

**Architecture:**
```
systemd
├── orch-serve.service
├── orch-web.service
└── opencode.service
```

**Characteristics:**
- ✅ Auto-restart on crash
- ✅ Auto-start at boot
- ✅ Log aggregation (journalctl)
- ✅ Resource limits and security
- ✅ Health monitoring
- ✅ Standard for VPS deployments

**Management:**
```bash
systemctl start orch-serve
systemctl status orch-serve
systemctl restart orch-serve
journalctl -u orch-serve -f
```

## Rationale

### Why Not Production-Style on Mac?

**Development doesn't need:**
- Auto-restart on crash (want to see crashes during dev)
- Auto-start at login (manual control is fine)
- Process supervision complexity (launchd plists, PATH issues)
- Self-healing daemons (orch doctor was overkill for dev)

**Development benefits from:**
- Simple configuration (Procfile vs XML plists)
- Easy experimentation (restart services frequently)
- Unified logs (see all services at once)
- Standard tooling (overmind used in many projects)

### Why launchd Failed for Dev

**The attempt:** Individual launchd plists for each service (com.orch.serve, com.orch.web, com.orch.doctor, com.overmind.orch-go)

**Problems encountered:**
1. **tmux PATH propagation** - launchd → overmind → tmux didn't work
2. **Circular dependency** - Need overmind for restart, need launchd to supervise overmind
3. **XML verbosity** - 120+ lines of XML vs 3 lines of Procfile
4. **Wrapper scripts** - Needed ~/.orch/start-web.sh for bun PATH
5. **Complexity** - Multiple plists to manage, unload/load cycle for changes

**See:** `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md` for detailed post-mortem.

### Why Overmind for Dev?

**Industry standard:** Used by Rails, Node, Python projects for dev environments

**Simple:** 3-line Procfile vs 120+ lines of launchd XML

**Unified logs:** `overmind echo` shows all services with color coding

**No PATH issues:** Runs in user environment, not launchd's minimal environment

**Easy restart:** `overmind restart api` vs `launchctl kickstart -k gui/$(id -u)/com.orch.serve`

### Why systemd for Prod (Future)?

**Native VPS tool:** Every Linux VPS has systemd

**Production features:**
- Process supervision with restart policies
- Resource limits (memory, CPU)
- Security sandboxing
- Log aggregation via journalctl
- Boot-time startup
- Health monitoring

**No tmux dependency:** Each service runs directly (no PATH issues)

## The Core Realization

**Initial assumption:** "We need always-on for Mac, but we couldn't figure out how to build it."

**Actual insight:** Mac was *resisting* being production because **dev environments aren't supposed to be always-on.**

### The Circular Debugging Pattern

**What happened (Jan 9-10):**
1. Tried launchd plists → hit tmux PATH issues
2. Debugged PATH for 1000+ lines → circular dependency discovered
3. Tried launchd → overmind supervision → more PATH issues
4. Went back to individual launchd plists → same architecture we started with
5. Realized: "We're going in circles"

**Why the circle existed:**
- Goal: Always-on behavior (production requirement)
- Environment: Development machine (Mac)
- These are **fundamentally incompatible**

**The system kept breaking because the architecture was fighting the use case.**

### What "Always-On" Actually Means

**Always-on is a production requirement:**
- Services running 24/7
- Survives reboots/crashes without intervention
- Accessible from anywhere
- Uptime matters

**Development doesn't need this:**
- You're actively working on it (started = working, stopped = not working)
- You WANT to see crashes (not auto-restart them away)
- You rebuild/restart constantly while iterating
- Services should stop when you close your laptop (saves battery)

### The Healthy Mental Model

**Before (confused):**
- "Mac should be like production"
- "Need auto-restart so dashboard 'just works'"
- "Why does launchd keep breaking?"
- Fighting the environment

**After (clear):**
- "Mac is dev, VPS is prod"
- "Dev = manual start/stop when working"
- "Prod = always-on with supervision"
- Working with the environment

### Why This Matters

**We didn't fail to build it.** We correctly identified that trying to make Mac production-like was the wrong goal.

The 2 days of circular debugging wasn't wasted effort - it was the system teaching us that dev and prod are different use cases requiring different architectures.

**The crashes today (12+ service crashes in 5 hours) validated the need for supervision - but that supervision belongs on the VPS, not the Mac.**

---

## Implementation

### Cleanup Completed (2026-01-10)

**Removed:**
- `~/Library/LaunchAgents/com.orch.doctor.plist`
- `~/Library/LaunchAgents/com.orch.serve.plist`
- `~/Library/LaunchAgents/com.orch.web.plist`
- `~/Library/LaunchAgents/com.overmind.orch-go.plist`
- `~/.orch/start-web.sh` (wrapper script)

**Updated:**
- `CLAUDE.md` - Dev environment section now references overmind
- `.kb/guides/dev-environment-setup.md` - Detailed overmind guide

**Kept:**
- `Procfile` - Already existed for overmind workflow

### Production Deployment (Future Work)

**When deploying to VPS:**
1. Create systemd service files (`.service` units)
2. Set up nginx reverse proxy (ports 3348, 5188)
3. Configure journalctl log rotation
4. Set up health checks and alerting
5. Document in `.kb/guides/production-deployment.md`

## Consequences

**Positive:**
- ✅ Simpler dev environment (Procfile vs launchd plists)
- ✅ Standard tooling (overmind is industry norm for dev)
- ✅ No PATH propagation issues
- ✅ Easier to onboard new developers
- ✅ Clear separation of dev vs prod concerns

**Negative:**
- ⚠️ No auto-restart on Mac (but dev doesn't need it)
- ⚠️ Services don't auto-start at login (manual `overmind start -D`)

**Trade-off accepted:**
- Manual lifecycle management is appropriate for dev environment
- Production will have proper supervision when deployed to VPS

## Success Criteria

**For dev environment:**
- ✅ Services start with single command (`overmind start -D`)
- ✅ Easy to restart individual services (`overmind restart api`)
- ✅ Unified logs viewable (`overmind echo`)
- ✅ No launchd complexity or PATH issues

**For future prod deployment:**
- ⏳ Services auto-restart on crash (systemd)
- ⏳ Services auto-start at boot (systemd)
- ⏳ Centralized logging (journalctl)
- ⏳ Health monitoring and alerting

## Related

**Context:**
- `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md` - Why launchd was wrong
- `.kb/decisions/2026-01-10-launchd-supervision-architecture.md` - Prior decision (superseded)
- `.kb/decisions/2026-01-10-individual-launchd-services.md` - Prior decision (superseded)

**Implementation:**
- `.kb/guides/dev-environment-setup.md` - Overmind workflow details
- `Procfile` - Service definitions
- `CLAUDE.md` - Quick reference

**Issue:** orch-go-je67h (cleanup task)
