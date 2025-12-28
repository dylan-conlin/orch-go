<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Daemon-managed dev servers should extend the existing orch daemon + status file pattern, with per-project servers.yaml declarations and health checks integrated into SessionStart.

**Evidence:** Three architectural patterns analyzed against Session Amnesia principle and existing infrastructure; orch daemon already writes status files, orch serve already exposes /api/servers endpoint, tmuxinator configs already exist per-project.

**Knowledge:** The key insight is servers are NOT orchestration infrastructure (like orch serve) - they are project dependencies like any other blocking condition. Health checks should gate work, not manage processes.

**Next:** Implement in 3 phases: (1) Add servers.yaml schema + health check command, (2) Integrate health checks into SessionStart hook, (3) Add auto-start to orch daemon polling loop.

---

# Investigation: Design Daemon Managed Development Servers

**Question:** How should the orch ecosystem manage development servers across projects so they survive session death, integrate with monitoring, and don't require orchestrator attention?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** Promote to decision if accepted, then create feature-impl issues
**Status:** Complete

---

## Problem Framing

### Design Question

How should the orch ecosystem manage development servers (vite, Docker, Rails) across multiple projects so that:
1. Servers survive agent session death
2. Orchestrator never asks Dylan "can you start the servers?"
3. Dashboard provides visibility into server health
4. SessionStart hook can verify servers are ready before work begins

### Success Criteria

1. **Survives session death:** Servers started by agents persist beyond agent session lifecycle
2. **Zero orchestrator friction:** No "please start the servers" questions
3. **Dashboard visibility:** `/api/servers` shows health per-project
4. **SessionStart integration:** Hook verifies servers healthy before work
5. **Multi-project support:** Works for orch-go, price-watch, beads-ui, glass, etc.
6. **Graceful degradation:** System works even if some servers are down

### Constraints

- **Session Amnesia principle:** Next session must know server state without memory
- **Local-First principle:** Use files/processes, not external services
- **Compose Over Monolith:** Extend existing infrastructure, don't create new daemon
- **Existing infrastructure:** Port registry at `~/.orch/ports.yaml`, tmuxinator configs, `orch serve` API, `orch daemon` polling loop

### Scope

**IN:**
- Per-project server declarations
- Health check mechanisms
- Auto-start on project entry
- Dashboard visibility
- SessionStart gating

**OUT:**
- Server log aggregation
- Automatic dependency installation
- Container orchestration (use existing Docker Compose)
- Cross-machine server management

---

## Findings

### Finding 1: Existing infrastructure is extensive but fragmented

**Evidence:** 
- Port registry at `~/.orch/ports.yaml` tracks allocations (pkg/port/port.go:91)
- tmuxinator configs at `~/.tmuxinator/workers-{project}.yml` define server commands
- `orch servers list|start|stop` manages tmux sessions (cmd/orch/servers.go)
- `/api/servers` endpoint already exists in serve.go (line 223)
- orch daemon writes status to `~/.orch/daemon-status.json`

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/servers.go:1-423`
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go:1541-1611`
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/status.go:1-128`

**Significance:** We don't need to build from scratch. The pieces exist but aren't connected: tmuxinator has the server definitions, serve.go has the API, daemon has the polling pattern. The missing piece is a unified manifest that includes health checks.

---

### Finding 2: Current pain points have clear patterns

**Evidence:** From issue orch-go-h674:
- price-watch: Docker services (`make up`) not running, orchestrator keeps asking Dylan
- orch-go: vite dev server on wrong port (5188 vs 5173), duplicate processes
- Servers started during session are lost when session ends
- No visibility into which servers are running for which projects

**Source:** `bd show orch-go-h674` description

**Significance:** The recurring pattern is: (1) server declaration is implicit in tmuxinator, (2) no health checks to verify readiness, (3) no auto-start mechanism on project entry. The solution must address all three.

---

### Finding 3: Session Amnesia requires externalized server state

**Evidence:** From principles.md:
> "Every pattern in this system compensates for Claude having no memory between sessions."
> "State must externalize to files (workspaces, artifacts, decisions)"

Current tmuxinator configs are static declarations. What's missing is:
1. Runtime state (is server actually running?)
2. Health state (is server responding correctly?)
3. Discovery mechanism (SessionStart needs to find and verify)

**Source:** `~/.kb/principles.md` lines 45-64

**Significance:** Server state must be queryable from files or lightweight checks, not from session memory. A health check command that writes status enables amnesia-resilient server management.

---

### Finding 4: Glass (orch-go-tiav) suggests shared daemon pattern

**Evidence:** From issue orch-go-tiav:
> "daemon-based shared browser control"
> "Persistent daemon connection to human's Chrome session"

The Glass approach: a persistent daemon that maintains connection to a resource (browser/server), surviving session boundaries.

**Source:** `bd show orch-go-tiav` description

**Significance:** Glass validates the daemon pattern for resources that should survive sessions. However, dev servers are simpler than browser state - they just need process supervision, not complex state management. The orch daemon already does process monitoring; servers can use similar pattern.

---

## Exploration: Three Approaches

### Approach A: Extend orch daemon with server management

**Mechanism:** Add server supervision to the existing orch daemon polling loop.
- Daemon reads `{project}/.orch/servers.yaml` for each active project
- Each poll cycle checks server health and auto-starts if needed
- Writes server status to `~/.orch/daemon-status.json`

**Pros:**
- Reuses existing daemon infrastructure (polling, status files)
- Single daemon to manage (already runs as launchd service)
- Natural integration with capacity management
- Status already exposed via `/api/daemon`

**Cons:**
- Daemon currently focused on agent spawning, not server management
- Risk of scope creep in daemon responsibilities
- Server management may need faster polling than agent spawning

**Complexity:** Medium - extends existing code but adds new responsibility domain

---

### Approach B: Separate orch-servers daemon

**Mechanism:** New dedicated daemon for server management.
- `orch servers daemon` runs independently via launchd
- Watches all registered projects' servers.yaml
- Writes status to `~/.orch/servers-status.json`
- SessionStart reads status file for gating

**Pros:**
- Single responsibility (just servers)
- Can poll more frequently for fast health detection
- Failure isolated from agent daemon

**Cons:**
- Another daemon to manage (more launchd plists)
- Duplicates daemon patterns (status file, polling)
- Coordination complexity if servers affect agent capacity

**Complexity:** High - new daemon, new plist, new status file, new API endpoint

---

### Approach C: On-demand health checks (no persistent daemon) ⭐

**Mechanism:** Server management via CLI commands, not persistent daemon.
- `orch servers check {project}` runs health checks and returns status
- SessionStart hook runs `orch servers check` before work begins
- `orch servers up {project}` starts servers if not healthy
- Status written to `{project}/.orch/servers-status.json` on each check

**Pros:**
- Simplest implementation (no new daemon)
- Works with existing tmuxinator for actual process management
- Health checks are explicit and visible in agent logs
- Per-project status files enable project-specific monitoring
- Graceful degradation (check fails = agent proceeds with warning)

**Cons:**
- No continuous monitoring (only checked at session boundaries)
- Agent must run check command (adds context window cost)
- Multiple sessions could race to start servers

**Complexity:** Low - extends existing CLI, uses existing tmuxinator

---

## Synthesis

### Key Insights

1. **Servers are dependencies, not orchestration** - Unlike agents which need capacity management, servers are blocking dependencies like any other precondition. The question is "is it ready?" not "should I spawn another one?"

2. **Health checks are the core primitive** - All approaches need the same thing: a way to ask "is server X for project Y healthy?" The difference is who asks and when. On-demand is simpler and more transparent than continuous polling.

3. **tmuxinator already manages processes** - We don't need to reinvent process supervision. tmuxinator starts/stops servers in tmux sessions that survive agent sessions. We just need health checks and auto-start triggers.

4. **SessionStart is the natural integration point** - Per Session Amnesia principle, each session needs to verify its environment. SessionStart already injects context; adding server health checks is natural extension.

### Answer to Investigation Question

**Recommended approach: On-demand health checks (Approach C) with SessionStart integration**

The key realization is that servers don't need a new daemon - they need:
1. A declaration format (servers.yaml)
2. A health check mechanism (orch servers check)
3. An auto-start trigger (SessionStart hook)
4. A status file (for dashboard visibility)

This leverages existing infrastructure:
- tmuxinator for process management
- orch servers for CLI commands
- SessionStart hook for environment verification
- /api/servers for dashboard visibility

---

## Implementation Recommendations

### Recommended Approach ⭐

**On-demand health checks with SessionStart integration**

**Why this approach:**
- Follows Compose Over Monolith - extends existing CLI, doesn't add new daemon
- Follows Session Amnesia - health checks are explicit in agent context
- Follows Local-First - status files are per-project, not central database
- Minimal new code - mostly wiring existing pieces together

**Trade-offs accepted:**
- No continuous monitoring (acceptable: servers are stable, failures are rare)
- Check adds ~1s to session start (acceptable: better than "start servers" request)
- Race condition on concurrent starts (acceptable: tmuxinator is idempotent)

**Implementation sequence:**

1. **Phase 1: servers.yaml schema + check command** (foundation)
   - Define `{project}/.orch/servers.yaml` schema with health checks
   - Implement `orch servers check {project}` command
   - Write status to `{project}/.orch/servers-status.json`
   
2. **Phase 2: SessionStart integration** (gating)
   - Add server health check to SessionStart hook
   - Surface warnings in agent context if servers unhealthy
   - Optionally block spawn if critical server is down
   
3. **Phase 3: Auto-start in daemon** (automation)
   - Add optional server health check to daemon polling loop
   - Auto-run `orch servers start` if check fails
   - Update `/api/servers` to include health status

### servers.yaml Schema

```yaml
# .orch/servers.yaml - Per-project server declarations
servers:
  - name: docker
    type: docker-compose
    command: make up
    workdir: .
    health:
      type: http
      url: http://localhost:3000/health
      timeout: 5s
    critical: true  # Block work if down
    
  - name: frontend
    type: command
    command: npm run dev
    workdir: frontend
    health:
      type: tcp
      port: 5173
      timeout: 2s
    critical: false  # Warn but proceed if down
```

### Alternative Approaches Considered

**Option A: Extend orch daemon**
- **Pros:** Single daemon, existing infrastructure
- **Cons:** Scope creep, mixed responsibilities
- **When to use instead:** If continuous monitoring becomes necessary (e.g., auto-restart on crash)

**Option B: Separate orch-servers daemon**
- **Pros:** Single responsibility, fast polling
- **Cons:** Another daemon to manage, duplication
- **When to use instead:** Never - too much overhead for the benefit

**Rationale for recommendation:** On-demand checks are simpler, more transparent, and sufficient for the use case. Dev servers are stable - they need occasional verification, not continuous monitoring.

---

### Implementation Details

**What to implement first:**
1. `servers.yaml` schema in pkg/config/servers.go
2. Health check implementations in pkg/servers/health.go (HTTP, TCP, command)
3. `orch servers check` command in cmd/orch/servers.go
4. Status file writing to `.orch/servers-status.json`

**Things to watch out for:**
- ⚠️ Docker Compose health checks can be slow (5-10s) - use generous timeouts
- ⚠️ Port conflicts with existing port registry - validate against `~/.orch/ports.yaml`
- ⚠️ tmuxinator start is not idempotent if session already exists - check first

**Areas needing further investigation:**
- How to handle servers that need environment variables (e.g., DATABASE_URL)
- Whether to support health check dependencies (frontend depends on backend)
- Integration with price-watch Docker workflow specifically

**Success criteria:**
- ✅ `orch servers check price-watch` returns health status for all declared servers
- ✅ SessionStart hook warns if servers are unhealthy
- ✅ Dashboard shows server health per-project
- ✅ Orchestrator never asks Dylan to start servers

---

## File Targets

**New files:**
- `pkg/servers/health.go` - Health check implementations
- `pkg/servers/config.go` - servers.yaml parsing
- `pkg/servers/status.go` - Status file management

**Modified files:**
- `cmd/orch/servers.go` - Add `check` subcommand
- `cmd/orch/serve.go` - Enhance `/api/servers` with health status
- SessionStart hook (in orch-knowledge) - Add server health gate

**Acceptance criteria:**
- [ ] servers.yaml schema defined with health check types
- [ ] `orch servers check {project}` returns JSON health status
- [ ] `.orch/servers-status.json` written on each check
- [ ] `/api/servers` includes health status when status file exists
- [ ] SessionStart hook verifies server health before spawning

---

## Structured Uncertainty

**What's tested:**
- ✅ tmuxinator manages tmux sessions that survive agent sessions (verified: workers sessions persist)
- ✅ orch serve /api/servers endpoint exists (verified: cmd/orch/serve.go:223)
- ✅ Port registry tracks allocations (verified: ~/.orch/ports.yaml exists)
- ✅ Daemon writes status files (verified: pkg/daemon/status.go)

**What's untested:**
- ⚠️ HTTP health checks can detect server readiness (not benchmarked with Docker)
- ⚠️ SessionStart hook overhead for health checks (not measured)
- ⚠️ Concurrent session starts racing to start servers (not tested)

**What would change this:**
- If health checks prove too slow (>5s), would need async/cached approach
- If servers need auto-restart on crash, would need daemon approach (Option A)
- If cross-project server dependencies emerge, would need dependency graph

---

## References

**Files Examined:**
- `cmd/orch/servers.go` - Existing servers CLI implementation
- `cmd/orch/serve.go` - API server with /api/servers endpoint
- `pkg/daemon/daemon.go` - Daemon polling pattern
- `pkg/daemon/status.go` - Status file writing pattern
- `pkg/port/port.go` - Port registry implementation
- `~/.tmuxinator/workers-price-watch.yml` - Example tmuxinator config
- `~/.tmuxinator/workers-orch-go.yml` - Example tmuxinator config

**Commands Run:**
```bash
# Get issue details
bd show orch-go-h674

# Check related glass issue
bd show orch-go-tiav

# List existing tmuxinator configs
ls ~/.tmuxinator/workers-*.yml
```

**Related Artifacts:**
- **Issue:** orch-go-h674 - Daemon-managed development servers per project
- **Issue:** orch-go-tiav - glass daemon-based shared browser control
- **Principle:** Session Amnesia - state must externalize to files

---

## Investigation History

**2025-12-27 16:03:** Investigation started
- Initial question: How to manage dev servers so they survive session death?
- Context: Recurring friction in price-watch (Docker) and orch-go (vite)

**2025-12-27 16:15:** Problem framing complete
- Identified 5 success criteria
- Scoped constraints around existing infrastructure

**2025-12-27 16:30:** Exploration phase complete
- Analyzed 3 approaches: extend daemon, new daemon, on-demand checks
- Identified on-demand as recommended approach

**2025-12-27 16:45:** Investigation completed
- Status: Complete
- Key outcome: Recommend on-demand health checks with servers.yaml schema and SessionStart integration
