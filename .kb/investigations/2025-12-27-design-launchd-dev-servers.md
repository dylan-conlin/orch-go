<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Docker Compose with `restart: unless-stopped` is the optimal solution - single declarative file per project, Docker Desktop handles boot-time startup, AI has excellent diagnostic tools (`docker compose logs`, `docker compose ps`), and Dylan never thinks about servers.

**Evidence:** Docker Compose is already used for price-watch; adding `restart: unless-stopped` + Docker Desktop auto-start gives operational invisibility with zero new infrastructure; AI can diagnose via standard Docker CLI.

**Knowledge:** Evaluation with correct lens (Dylan=zero mental load, AI=autonomous diagnosis) favors Docker Compose over launchd plists. Launchd has lower AI debuggability (scattered logs, less tooling).

**Next:** Add `restart: unless-stopped` to existing docker-compose files, ensure Docker Desktop starts on login, containerize remaining services (orch-go vite → node container).

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

### REFRAME: Correct Evaluation Lens

**Dylan doesn't write code - AI does everything.** So the evaluation criteria are:

1. **Operational invisibility** - Dylan never thinks about servers
2. **AI debuggability** - When broken, how fast can AI fix it autonomously?
3. **Conceptual simplicity** - Simple mental model (or no model needed)

Implementation complexity and learning curve are IRRELEVANT - AI handles both.

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

## Re-Evaluation: Correct Lens Applied

### Evaluation Framework

| Criterion | Weight | Question |
|-----------|--------|----------|
| **Operational Invisibility** | High | Does Dylan ever think about servers? |
| **AI Debuggability** | High | Can AI diagnose/fix issues autonomously? |
| **Conceptual Simplicity** | Medium | Simple mental model (or none needed)? |
| Implementation Complexity | ~~Low~~ | AI handles this - irrelevant |

### Option Analysis

#### Option A: launchd plists (original recommendation)

**Operational Invisibility:** ✅ Excellent - RunAtLoad + KeepAlive means servers just run
**AI Debuggability:** ⚠️ Medium
- Logs scattered: `~/.orch/logs/`, `~/Library/Logs/`
- Diagnosis requires: `launchctl list | grep`, reading multiple log files
- No unified view of "what's running and why"
- Less community tooling than Docker

**Conceptual Model:** ⚠️ Medium - "plists in LaunchAgents" is Mac-specific, less intuitive

#### Option B: Docker Compose for everything ⭐

**Operational Invisibility:** ✅ Excellent - `restart: unless-stopped` + Docker Desktop auto-start
**AI Debuggability:** ✅ Excellent
- Unified tools: `docker compose ps`, `docker compose logs -f`
- All services in one file, one command to diagnose
- Rich ecosystem of Docker debugging knowledge
- AI can run `docker compose logs service-name --tail 100` instantly

**Conceptual Model:** ✅ Simple - "everything is in docker-compose.yml"

**Trade-off:** Non-Docker projects (orch-go vite) need containerization or wrapper

#### Option C: Nix/devbox

**Operational Invisibility:** ⚠️ Unknown - Not currently installed
**AI Debuggability:** ⚠️ Unknown - Less AI training data than Docker
**Conceptual Model:** ✅ Elegant - `nix develop` starts everything

**Trade-off:** Requires installation, unfamiliar to most AI models

#### Option D: Hybrid (Docker Compose primary, launchd for edge cases)

**Operational Invisibility:** ✅ Excellent
**AI Debuggability:** ✅ Good - Docker is primary, launchd only when needed
**Conceptual Model:** ⚠️ Medium - Two systems to understand

### Revised Recommendation

**Docker Compose for everything** wins on AI debuggability:

| Approach | Dylan Thinks About | AI Debug Speed | Files to Manage |
|----------|-------------------|----------------|-----------------|
| launchd | Never | Medium (scattered logs) | N plists + servers.yaml |
| Docker Compose | Never | Fast (unified tools) | 1 docker-compose.yml |
| Nix | Never | Unknown | 1 flake.nix |

**Key insight:** AI can debug Docker issues faster because:
1. More training data on Docker than launchd
2. Unified logging (`docker compose logs`)
3. Single file shows all services (`docker compose ps`)
4. Standard error patterns AI recognizes

---

## Synthesis (Revised)

### Key Insights

1. **Docker Compose > launchd for AI debuggability** - When something breaks, AI can run `docker compose logs` and see everything. With launchd, AI must hunt across multiple log files and parse plist XML.

2. **Containerize remaining services** - orch-go vite should run in a node container. This unifies the approach: everything is Docker.

3. **Docker Desktop auto-start is the boot mechanism** - Configure Docker Desktop to start on login. All compose services with `restart: unless-stopped` come up automatically.

4. **One docker-compose.yml per project** - Dylan never sees it, AI maintains it, everything just works.

### Answer to Investigation Question

**How to make servers auto-start after reboot with zero human intervention:**

1. **Configure Docker Desktop to start on login** (one-time setup)
2. **Add `restart: unless-stopped` to all docker-compose.yml services**
3. **Containerize non-Docker services** (orch-go vite → node container)
4. Docker handles the rest: auto-start on boot, restart on crash, unified logging

**This is Session Amnesia-compliant:** The docker-compose.yml IS the externalized state. AI can diagnose any issue with `docker compose logs`.

---

## Implementation Recommendations

### Recommended Approach ⭐

**Docker Compose for everything**

**Why this approach:**
- **AI debuggability:** `docker compose logs` shows everything in one place
- **Unified tooling:** AI has extensive Docker training data
- **Zero Dylan mental load:** Everything is containers, one pattern
- **Already partially implemented:** price-watch uses Docker Compose

**Trade-offs accepted:**
- Non-Docker projects need containerization (orch-go vite)
- Docker Desktop must start on login (one-time config)
- Slight overhead for simple dev servers

**Implementation sequence:**

1. **Phase 1: Docker Desktop auto-start** - Configure to start on login
2. **Phase 2: Add restart policies** - `restart: unless-stopped` to existing compose files
3. **Phase 3: Containerize remaining services** - orch-go vite in node container
4. **Phase 4: Dashboard visibility** - `docker compose ps` via `/api/servers`

### docker-compose.yml Pattern

```yaml
# {project}/docker-compose.yml
services:
  vite:
    image: node:20-alpine
    working_dir: /app
    volumes:
      - .:/app
    command: npm run dev -- --host 0.0.0.0 --port 5173
    ports:
      - "5188:5173"
    restart: unless-stopped
    
  # Existing services just add restart policy
  postgres:
    restart: unless-stopped
    # ... existing config
```

### AI Debugging Commands

```bash
# See all services status
docker compose ps

# See logs for specific service
docker compose logs vite --tail 100

# Restart a service
docker compose restart vite

# Check why service failed
docker compose logs vite --since 5m
```

### Alternative: launchd (previous recommendation)

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
