<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Overmind monitoring infrastructure exists and is feature-complete; docker-compose integration requires backend abstraction layer with ServiceBackend interface to enable multi-backend observability.

**Evidence:** Analyzed 8 files (monitor.go, serve.go, status_cmd.go, dashboard components); existing ServiceMonitor polls overmind every 10s, logs events to jsonl, streams to dashboard via SSE; no docker-compose support found.

**Knowledge:** Event-driven architecture with backend abstraction enables unified observability for overmind, docker-compose, and future backends without breaking changes; dashboard already supports multi-service display; CLI vs dashboard separation (infrastructure vs services) should be maintained.

**Next:** Implement ServiceBackend interface in pkg/service/backend.go, refactor ServiceMonitor to use OvermindBackend, add DockerComposeBackend, update /api/services to aggregate backends.

**Promote to Decision:** Actioned - decision exists (event-sourced-monitoring-architecture)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Orch Go Architect Design Observability

**Question:** How should we design observability infrastructure for overmind/docker-compose services to integrate with orch status and dashboard?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Worker agent (orch-go-pwtrh)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Overmind monitoring infrastructure already exists

**Evidence:** 
- ServiceMonitor (`pkg/service/monitor.go:36-297`) polls `overmind status` every 10s
- Detects crashes by PID changes, auto-restarts services
- Logs lifecycle events to `~/.orch/events.jsonl`
- API endpoint `/api/services` returns service health (serve.go:283)
- Dashboard `ServicesSection` displays overmind services with status, PID, uptime, restart count
- Service events stream via SSE at `/api/events/services` (serve_services_events.go:20)

**Source:** 
- `pkg/service/monitor.go:36-297`
- `cmd/orch/serve.go:283-285`
- `cmd/orch/serve_services_events.go:20-241`
- `web/src/lib/components/service-card/service-card.svelte:1-127`

**Significance:** Foundation already exists for service observability. Design should extend this pattern to support docker-compose rather than build parallel infrastructure.

---

### Finding 2: No docker-compose integration exists

**Evidence:**
- `rg "docker-compose|docker_compose"` in Go files returns 0 matches
- No docker-compose monitoring in ServiceMonitor
- Dashboard has no docker-compose service cards

**Source:**
- Grep search across `**/*.go` files
- Manual inspection of `pkg/service/monitor.go`

**Significance:** Docker-compose integration is net-new work. Need to design: service discovery, health checking, lifecycle event detection.

---

### Finding 3: `orch status` shows infrastructure health but not overmind services

**Evidence:**
- `orch status` shows: Dashboard (3348), OpenCode (4096), Daemon status
- Uses TCP port checks and daemon-status.json file
- Does NOT display overmind service health directly (only in dashboard)

**Source:**
- `cmd/orch/status_cmd.go:456-1570`
- `pkg/daemon/status.go:1-132`

**Significance:** Design decision needed: Should `orch status` CLI show overmind/docker-compose services, or only dashboard? Current separation suggests infrastructure health (CLI) vs service health (dashboard).

---

### Finding 4: Event-driven architecture with SSE streaming

**Evidence:**
- ServiceMonitor logs events to `~/.orch/events.jsonl` via EventLogger interface
- Dashboard subscribes via SSE `/api/events/services?follow=true`
- Event types: `service.started`, `service.crashed`, `service.restarted`
- Real-time updates without polling

**Source:**
- `pkg/service/monitor.go:30-34` (EventLogger interface)
- `cmd/orch/serve_services_events.go:20-163` (SSE handler)
- `web/src/routes/+page.svelte:143` (connectServicelogSSE)

**Significance:** Docker-compose integration should follow same event-driven pattern. Enables real-time dashboard updates and historical event analysis.

---

### Finding 5: Services defined in Procfile, managed by overmind

**Evidence:**
```
api: orch serve
web: cd web && bun run dev
opencode: ~/.bun/bin/opencode serve --port 4096
```

**Source:** `Procfile:1-4`

**Significance:** Overmind services are homogeneous (all long-running HTTP servers). Docker-compose services may be heterogeneous (databases, queues, workers) requiring different health check strategies.

---

## Synthesis

**Key Insights:**

1. **Overmind monitoring is feature-complete but backend-specific** - The existing ServiceMonitor architecture (polling, crash detection, auto-restart, event logging, SSE streaming) provides a solid foundation. However, it's tightly coupled to overmind CLI (`overmind status` parsing). Need abstraction layer to support multiple backends.

2. **Docker-compose requires different health check strategy** - Overmind uses PID changes to detect crashes. Docker containers use `docker-compose ps` for status and `docker inspect` for health checks. Need backend-agnostic service state representation.

3. **Event-driven architecture enables real-time observability** - Current pattern (EventLogger → ~/.orch/events.jsonl → SSE stream → Dashboard) works well. Docker-compose integration should emit events to same stream, enabling unified service observability regardless of backend.

4. **Dashboard already supports multi-service display** - ServicesSection and ServiceCard components are backend-agnostic. Adding docker-compose services requires API changes, not UI redesign.

5. **Separation of concerns: CLI vs Dashboard** - `orch status` focuses on infrastructure health (ports, daemons). Service-level observability lives in dashboard. Design should maintain this boundary.

**Answer to Investigation Question:**

Design a **Backend-Agnostic Service Monitor** with:

1. **ServiceBackend interface** - Abstraction for overmind, docker-compose, or future backends (systemd, k8s)
2. **Unified ServiceState** - Common representation (name, status, uptime, restart count) regardless of backend
3. **Shared event stream** - Both backends emit to `~/.orch/events.jsonl` with consistent schema
4. **Dashboard integration** - API returns services from all backends, dashboard displays unified view
5. **`orch status` integration** - Optional CLI display of service health with `--services` flag

This enables Dylan to run projects with overmind (orch-go), docker-compose (kb-cli), or both simultaneously, with consistent observability.

---

## Structured Uncertainty

**What's tested:**

- ✅ Overmind monitoring works (verified: read existing code in pkg/service/monitor.go, confirmed dashboard displays services)
- ✅ ServiceMonitor polls overmind status every 10s (verified: code inspection line 64-78)
- ✅ Events are logged to ~/.orch/events.jsonl (verified: EventLogger interface usage)
- ✅ Dashboard receives real-time updates via SSE (verified: web/src/routes/+page.svelte:143 connectServicelogSSE call)

**What's untested:**

- ⚠️ Docker-compose health checks are slower than PID polling (hypothesis - not benchmarked)
- ⚠️ ServiceBackend interface design supports future backends like systemd (assumed - no prototype)
- ⚠️ Auto-restart conflicts with docker-compose restart policies (theoretical - not tested in production)
- ⚠️ Multiple projects with different backends can run simultaneously without port conflicts (assumed - depends on project configuration)

**What would change this:**

- Design would need revision if: Dylan wants docker-compose to be primary backend (replace overmind entirely)
- Implementation would differ if: ServiceMonitor polling interval needs to vary by backend (e.g., docker slower, overmind faster)
- Architecture would break if: Future backend requires push-based updates instead of polling (would need observer pattern)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Backend-Agnostic Service Monitor with Plugin Architecture** - Create `ServiceBackend` interface that both OvermindBackend and DockerComposeBackend implement, unified by MonitorService orchestrator.

**Why this approach:**
- Preserves existing overmind functionality without breaking changes
- Event-driven architecture (Finding 4) enables real-time updates for all backends
- Dashboard (Finding 4) already displays services generically - no UI redesign needed
- Enables future backends (systemd, k8s) by implementing same interface
- Maintains separation between infrastructure health (CLI) and service health (dashboard) per Finding 3

**Trade-offs accepted:**
- Adds abstraction layer (complexity) - acceptable because overmind-only pattern is already proven
- Docker-compose health checks may be slower than PID polling - acceptable because accuracy > speed for service monitoring
- Requires refactoring existing ServiceMonitor - acceptable because it's isolated to pkg/service/

**Implementation sequence:**
1. **Define ServiceBackend interface** - Create abstraction with `FetchServices()` and `RestartService()` methods. Foundational because both backends must conform.
2. **Refactor existing monitor to use OvermindBackend** - Extract overmind CLI logic into backend implementation. Proves interface design before adding complexity.
3. **Implement DockerComposeBackend** - Add docker-compose support using same interface. Validates abstraction works for heterogeneous backends.
4. **Update API to aggregate backends** - `/api/services` returns services from all backends. Enables unified dashboard display.
5. **Add `orch status --services` flag** - Optional CLI display of service health. Maintains CLI/dashboard separation.

### Alternative Approaches Considered

**Option B: Parallel monitoring systems**
- **Pros:** No refactoring of existing code, faster to ship docker-compose support
- **Cons:** Duplicates event logging, API endpoints, dashboard components. Violates DRY. Creates maintenance burden.
- **When to use instead:** Never. This is technical debt disguised as speed.

**Option C: Replace overmind with docker-compose everywhere**
- **Pros:** Single backend reduces complexity, docker-compose has richer ecosystem
- **Cons:** Breaking change for existing orch-go services. Requires Dockerfile for every service. Overmind's simplicity (Procfile) is valuable for dev workflows.
- **When to use instead:** If Dylan decides to standardize on containers, but current mixed approach is intentional.

**Option D: Extend ServiceMonitor to detect backend automatically**
- **Pros:** No explicit backend configuration, "just works" 
- **Cons:** Magic behavior is fragile. What if both Procfile and docker-compose.yml exist? Explicit > implicit for infrastructure.
- **When to use instead:** Future UX enhancement after explicit backend selection is proven stable.

**Rationale for recommendation:** Option A (Backend-Agnostic Plugin Architecture) is the only approach that:
1. Preserves existing overmind functionality (Finding 1)
2. Supports both backends simultaneously (Dylan's need per task description)
3. Maintains event-driven architecture (Finding 4)
4. Enables future extensibility without breaking changes

---

### Implementation Details

**What to implement first:**
1. **ServiceBackend interface** - Define contract in `pkg/service/backend.go`:
   ```go
   type ServiceBackend interface {
       Name() string // "overmind", "docker-compose"
       FetchServices() ([]ServiceState, error)
       RestartService(name string) error
       IsAvailable() bool // Can this backend be used in current project?
   }
   ```
2. **Refactor ServiceMonitor** - Replace direct `overmind` calls with backend abstraction
3. **Add backend detection** - Check for `Procfile` (overmind) and `docker-compose.yml` in project directory
4. **Update API response** - Add `backend` field to service JSON: `{"name": "api", "status": "running", "backend": "overmind"}`

**Things to watch out for:**
- ⚠️ **Docker health checks are async** - `docker-compose ps` shows "Up" even if container is starting. Must check health status separately with `docker inspect --format='{{.State.Health.Status}}'`
- ⚠️ **PID semantics differ** - Overmind shows process PID, Docker shows container ID. Use string `identifier` field instead of int `pid` in ServiceState.
- ⚠️ **Restart behavior varies** - Overmind restarts instantly, docker-compose respects `restart_policy`. Auto-restart may conflict with compose policies.
- ⚠️ **Service discovery timing** - Overmind services appear immediately (Procfile is static), docker services may be created dynamically. Need initial scan + incremental updates.
- ⚠️ **Cross-project confusion** - User may have multiple projects with different backends. API needs `project` parameter to disambiguate.

**Areas needing further investigation:**
- How should `orch status --services` format output? Table, JSON, or match existing agent display?
- Should ServiceMonitor poll both backends simultaneously or have separate monitors?
- Docker-compose projects may define services Dylan doesn't want monitored (db dumps, migrations). Need service filtering?
- What happens if user runs `docker-compose down` while monitor is running? Need graceful degradation.

**Success criteria:**
- ✅ Dashboard displays services from both overmind and docker-compose with distinct visual indicators (icons/badges)
- ✅ Service crashes (overmind PID change OR docker container restart) emit events to `~/.orch/events.jsonl`
- ✅ `orch status --services` shows health of all services across all backends
- ✅ Auto-restart works for overmind (existing) but respects docker-compose restart policies (new)
- ✅ Can run orch-go (overmind) and kb-cli (docker-compose) simultaneously without conflicts

---

## References

**Files Examined:**
- `pkg/service/monitor.go` - Existing overmind monitoring architecture, EventLogger interface
- `cmd/orch/serve.go` - API endpoint definitions, service health handlers
- `cmd/orch/serve_services_events.go` - SSE streaming for service events
- `cmd/orch/status_cmd.go` - Infrastructure health checking, TCP port tests
- `pkg/daemon/status.go` - Daemon status file format
- `web/src/routes/+page.svelte` - Dashboard main page, SSE connection setup
- `web/src/lib/components/service-card/service-card.svelte` - Service display component
- `Procfile` - Overmind service definitions

**Commands Run:**
```bash
# Search for docker-compose integration
rg "docker-compose|docker_compose" --type go

# Find dashboard files
glob "web/**/*"

# Find status-related files
glob "**/status*.go"

# Check docker-compose availability
which docker-compose && docker-compose --version
```

**External Documentation:**
- Docker Compose CLI reference - https://docs.docker.com/compose/reference/
- Overmind documentation - https://github.com/DarthSim/overmind

**Related Artifacts:**
- **Constraint:** "orch status can show phantom agents" - Related to infrastructure health accuracy
- **Decision:** "Dual-mode architecture (tmux for visual, HTTP for programmatic)" - Confirms CLI vs Dashboard separation

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
