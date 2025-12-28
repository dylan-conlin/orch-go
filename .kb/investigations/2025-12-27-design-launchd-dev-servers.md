<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dev servers should run as native launchd services via per-project plists, NOT on-demand health checks or tmuxinator - launchd provides auto-start on login, auto-restart on crash, and zero human intervention.

**Evidence:** Existing launchd infrastructure (orch daemon, orch serve, tmuxinator autoload) proves the pattern works; tmuxinator-based approach requires orchestrator attention which violates the hard constraint.

**Knowledge:** The key insight is servers are infrastructure like orch daemon itself - not dependencies to check, but services that should always be running. Use `orch servers gen-plist` to generate launchd plists per-project.

**Next:** Implement in 3 phases: (1) servers.yaml schema + gen-plist command, (2) launchd plist template + installation, (3) dashboard visibility via `/api/servers` health checks.

---

# Investigation: Launchd Dev Servers

**Question:** How should dev servers be redesigned so Dylan NEVER has to start/stop/check servers manually - after system reboot, all servers should already be running?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** Promote to decision if accepted, then create feature-impl issues
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** .kb/investigations/2025-12-27-inv-design-daemon-managed-development-servers.md
**Superseded-By:** N/A

---

## Problem Framing

### HARD CONSTRAINT (from orchestrator)

Dylan should NEVER have to start/stop/check servers manually. After system reboot, starting an orchestration session should just work - all servers already running.

### What the Prior Investigation Got Wrong

The prior investigation (2025-12-27-inv-design-daemon-managed-development-servers.md) recommended "on-demand health checks with SessionStart integration." This was rejected because:

1. It still requires orchestrator attention (checking health, starting servers)
2. It doesn't auto-start after reboot
3. SessionStart gating adds latency to every session

The correct framing: **Servers are infrastructure, not dependencies.**

### Success Criteria

1. **Zero intervention after reboot:** Mac login → servers already running
2. **Auto-restart on crash:** launchd's KeepAlive handles process failures
3. **Zero orchestrator attention:** No "can you start the servers?" ever again
4. **Per-project configuration:** Different projects have different server needs
5. **Dashboard visibility:** Can see server health without managing it

---

## Findings

### Finding 1: launchd provides everything we need

**Evidence:** Existing launchd services demonstrate the pattern:
- `com.orch.daemon.plist` - orch daemon with RunAtLoad + KeepAlive
- `com.orch-go.serve.plist` - orch serve with RunAtLoad + KeepAlive  
- `com.user.tmuxinator.plist` - autoload script for non-server tmux sessions

Key plist capabilities:
- `RunAtLoad: true` - Start immediately when plist is loaded (and on login)
- `KeepAlive: true` - Restart if process exits
- `WorkingDirectory` - Set CWD for the process
- `EnvironmentVariables` - Set PATH, project-specific vars
- `StandardOutPath/StandardErrorPath` - Logging

**Source:** 
- `~/Library/LaunchAgents/com.orch.daemon.plist`
- `~/Library/LaunchAgents/com.orch-go.serve.plist`

**Significance:** We don't need a new daemon or health check system. launchd IS the daemon - it already provides process supervision, auto-restart, and boot-time startup. We just need to generate the right plists.

---

### Finding 2: Different server types require different plist patterns

**Evidence:** Examining current server configurations:

**Type 1: Simple command** (orch-go vite)
```yaml
# workers-orch-go.yml
- bun run dev --port 5188
```
→ Plist: ProgramArguments=[bun, run, dev, --port, 5188], WorkingDirectory=/path/to/orch-go

**Type 2: Docker Compose** (price-watch)
```yaml
# workers-price-watch.yml  
- make up && sleep 5 && make logs
```
→ Plist: More complex - needs `docker compose up -d` (detached), health via docker compose

**Type 3: Multi-step startup** (some projects)
```
- make install && npm run dev
```
→ Plist: Use shell wrapper script, or break into separate services

**Source:**
- `~/.tmuxinator/workers-orch-go.yml`
- `~/.tmuxinator/workers-price-watch.yml`
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/Makefile`

**Significance:** The servers.yaml schema must support multiple server types. Each type generates a different plist pattern.

---

### Finding 3: Docker Compose has native "restart on boot" but we're not using it

**Evidence:** Docker Desktop on Mac can be configured to start on login. Docker Compose services with `restart: unless-stopped` will auto-restart. The price-watch docker-compose.yml currently has NO restart policy:

```yaml
services:
  postgres:
    # No restart: policy
  rails:
    # No restart: policy
```

**Source:** `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/docker-compose.yml`

**Significance:** For Docker-based servers, we have TWO options:
1. **Native Docker restart policy** - Add `restart: unless-stopped` to compose file + configure Docker Desktop to start on login
2. **launchd wrapper** - Create plist that runs `docker compose up -d` on boot

Option 1 is simpler for pure-Docker projects. Option 2 is more uniform with non-Docker projects.

---

### Finding 4: Port registry already tracks allocations per-project

**Evidence:** `~/.orch/ports.yaml` contains:
```yaml
allocations:
  - project: orch-go
    service: api
    port: 3348
  - project: orch-go
    service: web
    port: 5188
  - project: price-watch
    service: api
    port: 3338
  - project: price-watch
    service: web
    port: 5178
```

**Source:** `~/.orch/ports.yaml`, `pkg/port/port.go`

**Significance:** Port allocations already exist. The servers.yaml schema can reference these or we can extend ports.yaml to include startup commands. Either works.

---

## Synthesis

### Key Insights

1. **Servers ARE infrastructure, not dependencies** - The prior investigation framed servers as "blocking dependencies" to check before work. Wrong mental model. Servers are infrastructure that should always be running, like orch daemon itself. You don't "check if orch daemon is healthy before spawning" - it's just running.

2. **launchd provides the complete solution** - We don't need a custom daemon, health check loop, or polling mechanism. launchd IS the process supervisor. Generate a plist per server, bootstrap it, and launchd handles: start on boot, restart on crash, logging, environment.

3. **Per-project servers.yaml is the right abstraction** - Each project declares its servers (type, command, port, health check). `orch servers gen-plist <project>` generates launchd plists. `orch servers install <project>` bootstraps them.

4. **Docker is a special case** - For Docker Compose projects, we can either:
   - Use Docker's native restart policies (simpler, Docker-native)
   - Use launchd wrapper that runs `docker compose up -d` on boot
   
   Recommend: Use Docker's native approach + ensure Docker Desktop starts on login.

### Answer to Investigation Question

**How to make servers auto-start after reboot with zero human intervention:**

1. Define servers in per-project `servers.yaml`
2. Generate launchd plists via `orch servers gen-plist <project>`
3. Install via `orch servers install <project>` (runs `launchctl bootstrap`)
4. launchd handles the rest: RunAtLoad for boot, KeepAlive for crashes

For Docker projects specifically: Add `restart: unless-stopped` to compose files and configure Docker Desktop to start on login.

**This is Session Amnesia-compliant:** The plist IS the externalized state. Next Claude doesn't need to remember anything - the servers are just running.

---

## Implementation Recommendations

### Recommended Approach ⭐

**launchd-native services via per-project plists**

**Why this approach:**
- Uses existing, proven infrastructure (launchd)
- Zero custom daemon code
- Matches existing orch patterns (orch daemon, orch serve already use this)
- Zero orchestrator attention after initial setup

**Trade-offs accepted:**
- One-time setup per project (`orch servers install`)
- Docker projects need slightly different treatment
- No real-time health dashboard (but can add via `/api/servers` polling)

**Implementation sequence:**

1. **Phase 1: servers.yaml schema + gen-plist** - Define schema, generate plist files
2. **Phase 2: install/uninstall commands** - Bootstrap plists into launchd
3. **Phase 3: Dashboard visibility** - Add health check polling to `/api/servers`

### servers.yaml Schema

```yaml
# {project}/.orch/servers.yaml
servers:
  - name: vite
    type: command
    command: bun run dev --port 5188
    workdir: .  # relative to project root
    health:
      type: tcp
      port: 5188
      
  - name: docker
    type: docker-compose
    compose_file: docker-compose.yml
    services: [postgres, redis, rails]
    health:
      type: http
      url: http://localhost:3000/health
```

### Generated Plist (example)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "...">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.orch.server.orch-go.vite</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/dylanconlin/.bun/bin/bun</string>
        <string>run</string>
        <string>dev</string>
        <string>--port</string>
        <string>5188</string>
    </array>
    <key>WorkingDirectory</key>
    <string>/Users/dylanconlin/Documents/personal/orch-go</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/Users/dylanconlin/.orch/logs/orch-go-vite.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/dylanconlin/.orch/logs/orch-go-vite.err</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/Users/dylanconlin/.bun/bin:...</string>
    </dict>
</dict>
</plist>
```

### Alternative Approaches Considered

**Option B: Extend orch daemon with server supervision**
- **Pros:** Single daemon, unified monitoring
- **Cons:** Reinvents launchd's process supervision; adds complexity to daemon
- **When to use:** Never - launchd is better at process supervision

**Option C: On-demand health checks (prior investigation)**
- **Pros:** Simpler implementation
- **Cons:** Violates HARD CONSTRAINT - still requires checking, doesn't auto-start
- **When to use:** Never - the constraint rules this out

**Option D: Enhance tmuxinator autoload**
- **Pros:** Uses existing autoload mechanism
- **Cons:** tmuxinator creates tmux sessions (heavyweight); not process supervisor
- **When to use:** For non-dev-server tmux needs (editor sessions, etc.)

**Rationale for recommendation:** launchd is macOS's native process supervisor. Using it means we get all the benefits (auto-start, auto-restart, logging, system integration) without writing custom daemon code. The only work is generating plists.

---

### Implementation Details

**What to implement first:**
1. `servers.yaml` schema parser (`pkg/servers/config.go`)
2. Plist generator (`pkg/servers/launchd.go`)
3. CLI commands: `orch servers gen-plist`, `orch servers install`, `orch servers uninstall`

**Things to watch out for:**
- ⚠️ PATH must be fully expanded in plists (launchd doesn't inherit user PATH)
- ⚠️ WorkingDirectory must be absolute path
- ⚠️ For Docker: need to ensure Docker Desktop starts on login (separate config)
- ⚠️ Log rotation - launchd doesn't rotate; may need newsyslog.d config

**Areas needing further investigation:**
- How to handle servers that need environment variables beyond PATH
- Whether to support server dependencies (frontend waits for backend)
- How to handle price-watch's current `make up && make logs` pattern

**Success criteria:**
- ✅ After Mac reboot, orch-go vite server is running on port 5188
- ✅ After Mac reboot, price-watch Docker services are running
- ✅ Orchestrator never asks "can you start the servers?"
- ✅ `orch servers status` shows health without requiring intervention

---

## File Targets

**New files:**
- `pkg/servers/config.go` - servers.yaml schema parsing
- `pkg/servers/launchd.go` - plist generation
- `pkg/servers/health.go` - health check implementations (TCP, HTTP)

**Modified files:**
- `cmd/orch/servers.go` - add gen-plist, install, uninstall subcommands
- `cmd/orch/serve.go` - enhance `/api/servers` with health status

**New artifacts:**
- `.orch/servers.yaml` in orch-go and price-watch projects
- `~/Library/LaunchAgents/com.orch.server.*.plist` (generated)

---

## Structured Uncertainty

**What's tested:**

- ✅ launchd RunAtLoad + KeepAlive works for process supervision (verified: com.orch.daemon.plist works)
- ✅ Plist format is documented and stable (verified: multiple existing plists)
- ✅ Port registry exists and is parseable (verified: pkg/port/port.go)

**What's untested:**

- ⚠️ Bun as ProgramArguments[0] in launchd (may need full path)
- ⚠️ Docker Compose with launchd wrapper vs native restart policy
- ⚠️ Log rotation for launchd-managed services

**What would change this:**

- If Docker Compose services don't work well with launchd, use native restart policies instead
- If bun/node require shell wrapper, generate wrapper scripts instead of direct commands
- If log files grow unbounded, add newsyslog.d configuration

---

## References

**Files Examined:**
- `~/Library/LaunchAgents/com.orch.daemon.plist` - existing launchd pattern
- `~/Library/LaunchAgents/com.orch-go.serve.plist` - existing launchd pattern
- `~/Library/LaunchAgents/com.user.tmuxinator.plist` - tmuxinator autoload pattern
- `~/.tmuxinator/workers-orch-go.yml` - current server definition
- `~/.tmuxinator/workers-price-watch.yml` - current server definition
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/docker-compose.yml` - Docker pattern
- `cmd/orch/servers.go` - existing servers CLI
- `pkg/port/port.go` - port registry

**Commands Run:**
```bash
# List existing launchd agents
ls ~/Library/LaunchAgents/*.plist

# Read port registry
cat ~/.orch/ports.yaml

# Check tmuxinator configs
ls ~/.tmuxinator/workers-*.yml
```

**Related Artifacts:**
- **Investigation (superseded):** .kb/investigations/2025-12-27-inv-design-daemon-managed-development-servers.md - prior approach (rejected)
- **Principle:** Session Amnesia - servers as externalized infrastructure state

---

## Investigation History

**2025-12-27 17:30:** Investigation started
- Initial question: How to make servers auto-start after reboot with zero human intervention?
- Context: Prior investigation recommended on-demand health checks; orchestrator rejected this approach

**2025-12-27 17:45:** Key insight identified
- Servers are infrastructure, not dependencies
- launchd provides complete process supervision
- No custom daemon needed

**2025-12-27 18:00:** Exploration complete
- Analyzed 4 approaches: launchd-native, extend daemon, on-demand, tmuxinator
- Recommended launchd-native with per-project plists

**2025-12-27 18:15:** Investigation completed
- Status: Complete
- Key outcome: Use launchd-native services via `orch servers gen-plist/install` commands
