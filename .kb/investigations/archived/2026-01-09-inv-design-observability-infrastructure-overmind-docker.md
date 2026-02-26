<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Service observability should integrate via three phases: (1) background monitoring daemon with crash notifications, (2) dashboard integration with cross-project visibility, (3) full event streaming and log viewer.

**Evidence:** Current agent observability (SSE, events, notifications, dashboard) works well and provides proven patterns; overmind uses tmux underneath with socket API making it queryable; existing InfrastructureHealth type in status command provides integration point.

**Knowledge:** Extending existing observability patterns to services is lower risk than building new infrastructure; polling overmind status every 10s is sufficient for crash detection; cross-project aggregation requires project config similar to agent dashboard.

**Next:** Implement Phase 1 MVP (service monitoring daemon with crash notifications) to validate polling approach and notification UX, then iterate to Phase 2 (dashboard) and Phase 3 (events) based on usage.

**Promote to Decision:** recommend-yes - This establishes the architectural pattern for service observability (polling + notifications + dashboard integration) that should guide future docker-compose integration and other service managers.

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

# Investigation: Design Observability Infrastructure Overmind Docker

**Question:** How should we integrate overmind/docker-compose service health, crash detection, and log surfacing into orch status and dashboard?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** architect agent (orch-go-e79ao)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Agent Observability Works Well, Service Observability is Manual

**Evidence:**
- Agent observability has multiple layers:
  - SSE events from OpenCode (`pkg/opencode/sse.go`) with real-time streaming
  - Desktop notifications (`pkg/notify/notify.go`) for completion/errors
  - Event logging to `~/.orch/events.jsonl` (`pkg/events/logger.go`)
  - Dashboard web UI at localhost:5188 showing agent status, phase, runtime
  - `orch status` command aggregating agents, accounts, infrastructure
- Service observability requires manual commands:
  - `overmind status` - check if services running
  - `overmind echo` - view unified logs
  - No crash detection
  - No notifications when services fail
  - No integration with orch dashboard

**Source:**
- `pkg/events/logger.go:14-29` - Event types: session.spawned, session.completed, agent.completed, etc.
- `pkg/notify/notify.go:52-65` - SessionComplete notifications
- `cmd/orch/status_cmd.go:58-171` - Status command structure with SwarmStatus, AgentInfo, InfrastructureHealth
- `web/src/routes/+page.svelte:15-46` - Dashboard stores and SSE connection
- `Procfile:1-3` - Three services: api, web, opencode
- `.kb/investigations/2026-01-09-inv-overmind-vs-launchd-prototype.md` - Overmind provides status/logs but no integration

**Significance:** The infrastructure for observability exists (events, notifications, SSE, dashboard) but only applies to agents, not services. Extending the existing patterns to services would provide consistency and leverage proven mechanisms.

---

### Finding 2: Overmind Uses Tmux Under the Hood with Socket API

**Evidence:**
- Overmind creates tmux sessions for process management:
  ```
  tmux -C -L overmind-orch-go-{id} new -n api -s orch-go
  ```
- Communicates via Unix socket at `.overmind.sock`
- Commands available:
  - `overmind status` - List processes with PIDs and status (running/stopped)
  - `overmind echo` - Stream unified logs from all processes
  - `overmind restart <process>` - Restart individual process
  - `overmind connect <process>` - Attach to tmux window
- Process structure visible in `ps aux`:
  - Parent tmux session with multiple windows (api, web, opencode)
  - Each process runs in its own tmux pane
  - Shell wrappers in temp dir for each process

**Source:**
- `ps aux | grep overmind` output showing tmux session structure
- `.overmind.sock` file in project root
- `overmind help` command listing available operations
- `.kb/investigations/2026-01-09-inv-overmind-vs-launchd-prototype.md:69-105` - Overmind benefits section

**Significance:** Overmind's tmux integration means we can use existing `pkg/tmux` package to query service state and capture output. The socket API provides a programmatic interface for status/control operations.

---

### Finding 3: Infrastructure Health Already Tracked, But Missing Service Status

**Evidence:**
- `orch status` includes `InfrastructureHealth` section:
  ```go
  type InfrastructureHealth struct {
      AllHealthy bool
      Services   []InfraServiceStatus
      Daemon     *DaemonStatus
  }
  ```
- Currently checks TCP ports to determine service health:
  - API server (port 3348)
  - Web dashboard (port 5188)
  - OpenCode server (port 4096)
- `DaemonStatus` tracked via `~/.orch/daemon-status.json` with last poll, spawn, completion times
- Missing from infrastructure checks:
  - Process-level health (is overmind actually running?)
  - Per-service restart counts
  - Service-specific errors/crashes
  - Log snippets on failure

**Source:**
- `cmd/orch/status_cmd.go:108-135` - InfrastructureHealth and DaemonStatus types
- `cmd/orch/status_cmd.go:1335-1365` - checkTCPPort function and readDaemonStatus
- `cmd/orch/doctor.go:218-312` - Health check functions for OpenCode, orch serve, beads daemon

**Significance:** The status command already has infrastructure health tracking, but only uses TCP port checks. We need process-level health and service-specific diagnostics for complete observability.

---

### Finding 4: Cross-Project Service Visibility is Manual and Context-Specific

**Evidence:**
- Dylan has multiple projects with dev servers (e.g., price-watch, snap, orch-go)
- Each project has its own Procfile and overmind instance
- No unified view of "which projects have services running?"
- Current workflow:
  ```bash
  cd ~/Documents/personal/orch-go && overmind status
  cd ~/Documents/personal/price-watch && overmind status
  # Repeat for each project...
  ```
- Dashboard currently shows agents from all projects (via beads IDs with project prefixes)
- But no cross-project service status

**Source:**
- CLAUDE.md mentions overmind for orch-go dashboard services
- Spawn context mentions cross-project spawns via `--workdir`
- Dashboard web UI already aggregates agents across projects

**Significance:** Users need to see "what's running across all my projects?" in one view. Service observability should match the cross-project agent observability pattern already established in the dashboard.

---

## Synthesis

**Key Insights:**

1. **Extend Existing Patterns, Don't Reinvent** - The observability infrastructure (events, notifications, SSE, dashboard) already works well for agents. Service observability should use the same patterns: events to JSONL, SSE for real-time updates, dashboard integration, desktop notifications. (Findings 1, 3)

2. **Leverage Overmind's Tmux Foundation** - Overmind uses tmux under the hood with programmatic access via socket and commands. We can use existing `pkg/tmux` package to query service state and capture logs without reimplementing process management. (Finding 2)

3. **Cross-Project Aggregation is Key** - The dashboard already aggregates agents across projects. Service observability must do the same - show all projects' services in one view, not just current directory. This requires discovering Procfiles across Dylan's workspace. (Finding 4)

4. **Crash Detection Needs Monitoring Loop** - Unlike agents (which emit SSE events), services need active monitoring to detect crashes. Options: polling `overmind status`, watching tmux panes for exits, or parsing overmind's internal state.

**Answer to Investigation Question:**

Service observability should integrate via three layers:

**Layer 1: Event System**
- Emit service.started, service.stopped, service.crashed events to `~/.orch/events.jsonl`
- Monitor via polling `overmind status` every 5-10s (lightweight - just PID check)
- Desktop notifications on crash (reuse `pkg/notify`)

**Layer 2: Status Integration**
- Extend `orch status` to include service health from all projects
- Query each project's `.overmind.sock` or parse `overmind status --json` output
- Show per-service uptime, restart count, health status alongside agents

**Layer 3: Dashboard Integration**
- New dashboard section: "Services" with cards per service (similar to agent cards)
- SSE endpoint `/events/services` streaming service state changes
- Log viewer component showing last N lines from `overmind echo`

Cross-project discovery via `~/.orch/config.yaml`:
```yaml
projects:
  - name: orch-go
    path: ~/Documents/personal/orch-go
  - name: price-watch
    path: ~/Documents/personal/price-watch
```

This approach reuses proven patterns (Finding 1), leverages overmind's tmux foundation (Finding 2), and provides cross-project visibility (Finding 4).

---

## Structured Uncertainty

**What's tested:**

- ✅ Overmind uses tmux underneath (verified: `ps aux | grep overmind` shows tmux sessions)
- ✅ `.overmind.sock` exists and overmind commands work (verified: `overmind status` returns process list)
- ✅ `pkg/events` event logging works (verified: code exists and is used for agent events)
- ✅ Dashboard SSE integration exists (verified: `web/src/routes/+page.svelte` has SSE stores)
- ✅ InfrastructureHealth type exists in status command (verified: `cmd/orch/status_cmd.go:108-135`)

**What's untested:**

- ⚠️ Polling frequency 5-10s is appropriate (not benchmarked for CPU/battery impact)
- ⚠️ Cross-project config discovery will work reliably (not implemented or tested)
- ⚠️ `overmind status --json` provides structured output (flag not verified to exist)
- ⚠️ SSE streaming won't overload dashboard with service events (not load tested)
- ⚠️ Tmux pane monitoring can detect crashes faster than polling (not compared)

**What would change this:**

- Design is wrong if overmind doesn't have JSON output (must parse text or query tmux directly)
- Polling approach is wrong if CPU usage exceeds 1% sustained (need event-driven instead)
- Cross-project config is wrong if Dylan wants auto-discovery of Procfiles (need filesystem scanning)
- Service events are wrong if they're too noisy (need intelligent filtering/debouncing)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Phased Integration: Monitor-first, Dashboard-second, Events-third** - Start with background monitoring daemon, then dashboard integration, finally full event streaming.

**Why this approach:**
- Crash detection is the highest-value feature (Finding 1) - get notifications working first
- Dashboard integration builds on monitoring infrastructure - reuses agent patterns (Finding 1)
- Event streaming is lowest priority (nice-to-have audit trail, not critical path)
- Incremental rollout allows validation at each layer before adding complexity
- Each phase delivers standalone value, not blocked on later phases

**Trade-offs accepted:**
- No event audit trail until Phase 3 (acceptable - notifications + dashboard provide visibility)
- Cross-project support deferred to Phase 2 (start with single-project, expand later)
- No log streaming in Phase 1 (acceptable - users can run `overmind echo` manually)

**Implementation sequence:**

**Phase 1: Service Monitor Daemon (Crash Detection + Notifications)**
1. Create `cmd/orch/service-monitor.go` - background process polling `overmind status`
2. Emit desktop notifications on service crashes (reuse `pkg/notify`)
3. Track last-seen PIDs to detect restarts
4. Launch via `orch serve` or as separate `orch service-monitor` command

**Phase 2: Dashboard Integration (Visibility)**
1. Add `/api/services` endpoint to `cmd/orch/serve.go` returning service health
2. Create Svelte store `$lib/stores/services.ts` (similar to agents store)
3. Add "Services" section to dashboard with service cards (name, status, uptime, restart count)
4. Cross-project support: scan `~/.orch/config.yaml` for project paths, query each `.overmind.sock`

**Phase 3: Event Streaming (Audit Trail)**
1. Add service event types to `pkg/events/logger.go` (service.started, service.stopped, service.crashed)
2. Emit events from service monitor daemon
3. Add `/events/services` SSE endpoint for real-time dashboard updates
4. Log viewer component showing recent output from `overmind echo`

### Alternative Approaches Considered

**Option B: Event-Driven via Tmux Pane Monitoring**
- **Pros:** Immediate crash detection (no polling delay), lower overhead than polling
- **Cons:** Complex implementation (watch tmux pane exits), requires deep tmux integration, harder to maintain
- **When to use instead:** If polling proves too slow (<5s response time required) or CPU overhead is unacceptable

**Option C: Full Overmind Wrapper/Abstraction**
- **Pros:** Complete control over service lifecycle, can add features overmind lacks
- **Cons:** Reimplements what overmind already does well (Finding 2), high maintenance burden, loses overmind's battle-tested reliability
- **When to use instead:** If overmind limitations become blocking (e.g., need Windows support, custom restart policies)

**Option D: Dashboard-Only (No Background Monitoring)**
- **Pros:** Simpler (no daemon), all state derived from polling on dashboard page load
- **Cons:** No crash notifications when dashboard isn't open, no persistent state tracking, reactive not proactive
- **When to use instead:** If notifications aren't needed and dashboard is always open

**Rationale for recommendation:**

Phased approach (Option A) balances value delivery with complexity:
- Phase 1 solves the immediate pain (silent crashes) with minimal code
- Phase 2 provides visibility without requiring full event infrastructure
- Phase 3 adds audit trail only if needed

Option B (tmux monitoring) is premature optimization - polling works for 3-10 services. Option C (wrapper) violates "leverage existing tools" (Finding 2). Option D (dashboard-only) misses the core requirement (crash detection when not watching).

---

### Implementation Details

**What to implement first (Phase 1 MVP):**

1. **Service monitoring package** (`pkg/service/monitor.go`):
   ```go
   type ServiceMonitor struct {
       projectPath string
       lastState   map[string]ServiceState // service name -> state
       notifier    *notify.Notifier
   }

   type ServiceState struct {
       PID         int
       Status      string // "running", "stopped", "crashed"
       LastSeen    time.Time
       RestartCount int
   }

   func (m *ServiceMonitor) Poll() error {
       // Run: overmind status --json (or parse text output)
       // Compare PIDs to lastState
       // Detect crashes (PID changed or process missing)
       // Emit notifications on state changes
   }
   ```

2. **Integration point**: Launch from `orch serve` as background goroutine
   - Polls every 10s (configurable via `~/.orch/config.yaml`)
   - Runs only if `.overmind.sock` exists in project dir
   - Stops when `orch serve` stops

3. **Test overmind output format**:
   ```bash
   overmind status --json  # Check if JSON flag exists
   # If not, parse text output:
   # PROCESS   PID       STATUS
   # api       44568     running
   ```

**Things to watch out for:**

- ⚠️ **Overmind may not have `--json` flag** - Need to parse text output or query tmux directly via `pkg/tmux`
- ⚠️ **PID changes don't always mean crashes** - Deliberate restarts via `overmind restart` should not notify. Track reason (crashed vs restarted)
- ⚠️ **Cross-project locking** - Multiple orch instances might monitor same project. Use lockfile or accept duplication
- ⚠️ **Polling overhead** - 10s * N projects could add up. Benchmark CPU usage, adjust interval if needed
- ⚠️ **Socket permissions** - `.overmind.sock` may have restrictive permissions. Handle EACCES gracefully

**Areas needing further investigation:**

- **Overmind restart detection** - Can we distinguish deliberate `overmind restart api` from crash-restart? May need to parse logs
- **Docker-compose integration** - Similar patterns would apply, but uses `docker-compose ps` instead of overmind. Defer until overmind integration proven
- **Service health beyond process running** - Should we check HTTP endpoints (port 3348, 5188, 4096) or trust overmind? Reuse existing TCP checks from `cmd/orch/status_cmd.go:1335`
- **Log snippet extraction** - On crash, show last 50 lines from `overmind echo <service>`. Requires parsing overmind output or tmux capture

**Success criteria:**

- ✅ **Crash notifications work**: Kill a service process, get desktop notification within 15s
- ✅ **No false positives**: Deliberate `overmind restart` doesn't send crash notification
- ✅ **Dashboard shows service health**: `orch status` includes service section with running/stopped/crashed states
- ✅ **Cross-project visibility**: Dashboard aggregates services from all configured projects
- ✅ **Performance acceptable**: Polling overhead <0.5% CPU sustained, <10MB memory

---

## References

**Files Examined:**
- `pkg/events/logger.go` - Event logging system to understand existing patterns
- `pkg/notify/notify.go` - Desktop notification system for crash alerts
- `cmd/orch/status_cmd.go` - Status command structure and InfrastructureHealth type
- `cmd/orch/doctor.go` - Health check functions for services
- `pkg/opencode/sse.go` - SSE streaming for real-time updates
- `web/src/routes/+page.svelte` - Dashboard stores and SSE integration
- `Procfile` - Services managed by overmind
- `pkg/tmux/tmux.go` - Tmux integration for potential pane monitoring

**Commands Run:**
```bash
# Check overmind status format
overmind status

# Verify overmind process structure
ps aux | grep overmind

# Check for overmind socket
ls -la .overmind.sock

# Verify overmind help output
overmind help
overmind help echo
```

**External Documentation:**
- Overmind GitHub: https://github.com/DarthSim/overmind - Process manager documentation

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-09-inv-overmind-vs-launchd-prototype.md` - Why overmind was chosen over launchd
- **Investigation:** `.kb/investigations/2025-12-23-inv-explore-options-centralized-server-management.md` - Earlier exploration of centralized server management (rejected per-project tools)

---

## Investigation History

**[2026-01-09 17:30]:** Investigation started
- Initial question: How should we integrate overmind/docker-compose service health, crash detection, and log surfacing into orch status and dashboard?
- Context: Overmind replaced launchd for dashboard services, but observability remains manual (no crash detection, no dashboard integration, no notifications)

**[2026-01-09 17:45]:** Analyzed current state
- Agent observability: Strong (SSE, events, notifications, dashboard)
- Service observability: Manual (overmind status/echo commands)
- Overmind uses tmux underneath with socket API

**[2026-01-09 18:00]:** Designed three-phase integration approach
- Phase 1: Background monitoring daemon with crash notifications
- Phase 2: Dashboard integration with cross-project visibility
- Phase 3: Full event streaming and log viewer

**[2026-01-09 18:15]:** Investigation completed
- Status: Complete
- Key outcome: Phased integration design that reuses existing patterns (events, SSE, dashboard) and leverages overmind's tmux foundation
