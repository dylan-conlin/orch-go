# Synthesis: Design Observability Infrastructure for Overmind Services

**Agent:** orch-go-e79ao (architect)
**Skill:** architect
**Date:** 2026-01-09
**Investigation:** `.kb/investigations/2026-01-09-inv-design-observability-infrastructure-overmind-docker.md`

---

## TLDR (30-second handoff)

Designed three-phase integration to extend agent observability patterns (events, SSE, notifications, dashboard) to overmind services. Phase 1: background monitoring daemon with crash notifications (MVP). Phase 2: dashboard integration with cross-project visibility. Phase 3: event streaming and log viewer. Approach reuses existing infrastructure and leverages overmind's tmux foundation.

---

## Context

**Problem:** Services managed by overmind (api, web, opencode) crash silently with no detection or notification. Observability is manual (run `overmind status`, `overmind echo`). No dashboard integration. No cross-project visibility.

**Why it matters:** Silent failures block development work. Users discover service crashes when something else fails (API calls, page loads). Agent observability works well - need parity for services.

---

## Key Findings

1. **Agent observability infrastructure is proven and reusable**
   - Events logged to `~/.orch/events.jsonl`
   - Desktop notifications via `pkg/notify`
   - SSE streaming for real-time updates
   - Dashboard integration with stores and cards
   - Same patterns should apply to services

2. **Overmind uses tmux underneath with programmatic access**
   - Creates tmux sessions with windows per service
   - Communicates via `.overmind.sock` Unix socket
   - Commands: `overmind status`, `overmind echo`, `overmind restart`
   - Can leverage existing `pkg/tmux` package

3. **Cross-project aggregation is required**
   - Dashboard already aggregates agents across projects
   - Service observability needs same cross-project view
   - Requires project discovery via config or filesystem scanning

4. **Polling is sufficient for crash detection**
   - 10s poll interval provides 15s worst-case detection time
   - Simpler than event-driven tmux pane monitoring
   - Lower overhead for 3-10 services per project

---

## Design Decision

**Recommendation:** **Phased Integration - Monitor-first, Dashboard-second, Events-third**

### Phase 1: Service Monitor Daemon (Crash Detection + Notifications)

**What:**
- Background goroutine launched from `orch serve`
- Polls `overmind status` every 10s
- Tracks last-seen PIDs to detect crashes
- Emits desktop notifications on service crashes

**Implementation:**
```go
// pkg/service/monitor.go
type ServiceMonitor struct {
    projectPath string
    lastState   map[string]ServiceState
    notifier    *notify.Notifier
}

type ServiceState struct {
    PID          int
    Status       string // "running", "stopped", "crashed"
    LastSeen     time.Time
    RestartCount int
}

func (m *ServiceMonitor) Poll() error {
    // Run: overmind status (parse text output)
    // Compare PIDs to lastState
    // Detect crashes and emit notifications
}
```

**Success criteria:**
- Kill a service process → get desktop notification within 15s
- Deliberate `overmind restart` doesn't send crash notification
- CPU overhead <0.5% sustained

### Phase 2: Dashboard Integration (Visibility)

**What:**
- `/api/services` endpoint returning service health
- Svelte store `$lib/stores/services.ts`
- Dashboard section with service cards (name, status, uptime, restart count)
- Cross-project support via `~/.orch/config.yaml`

**Config format:**
```yaml
projects:
  - name: orch-go
    path: ~/Documents/personal/orch-go
  - name: price-watch
    path: ~/Documents/personal/price-watch
```

**Success criteria:**
- Dashboard shows all services across all configured projects
- Service status updates in real-time
- Click service card → view logs

### Phase 3: Event Streaming (Audit Trail)

**What:**
- Service event types: `service.started`, `service.stopped`, `service.crashed`
- Events logged to `~/.orch/events.jsonl`
- `/events/services` SSE endpoint
- Log viewer component showing `overmind echo` output

**Success criteria:**
- Service crashes recorded in events.jsonl for post-mortem
- SSE updates dashboard in <1s
- Log viewer shows last 100 lines per service

---

## Alternative Approaches Considered

**Option B: Event-Driven via Tmux Pane Monitoring**
- Watch tmux pane exits for immediate crash detection
- Pros: No polling delay, lower overhead
- Cons: Complex implementation, deep tmux integration, harder to maintain
- Rejected: Polling is sufficient for 3-10 services, premature optimization

**Option C: Full Overmind Wrapper/Abstraction**
- Replace overmind with custom process manager
- Pros: Complete control, can add custom features
- Cons: Reimplements battle-tested tool, high maintenance burden
- Rejected: Violates "leverage existing tools" principle

**Option D: Dashboard-Only (No Background Monitoring)**
- All status derived from polling on dashboard page load
- Pros: Simpler, no daemon
- Cons: No crash notifications when dashboard isn't open
- Rejected: Misses core requirement (proactive crash detection)

---

## Implementation Plan

**Phase 1 MVP (Crash Detection):**
1. Create `pkg/service/monitor.go` with ServiceMonitor type
2. Integrate with `orch serve` as background goroutine
3. Test overmind output parsing (check if `--json` flag exists)
4. Implement PID tracking and crash detection logic
5. Emit desktop notifications via `pkg/notify`

**Phase 2 (Dashboard Integration):**
1. Add `/api/services` endpoint to `cmd/orch/serve.go`
2. Create `$lib/stores/services.ts` Svelte store
3. Add "Services" section to dashboard with service cards
4. Implement cross-project discovery via config

**Phase 3 (Event Streaming):**
1. Add service event types to `pkg/events/logger.go`
2. Emit events from service monitor
3. Add `/events/services` SSE endpoint
4. Create log viewer component

---

## Risks and Mitigations

**Risk: Overmind may not have `--json` flag**
- Mitigation: Parse text output or query tmux directly via `pkg/tmux`
- Validated: Need to test `overmind status --json` before implementation

**Risk: PID changes don't always mean crashes**
- Mitigation: Track restart reason (crashed vs deliberate restart)
- May need to parse overmind logs or track restart commands

**Risk: Polling overhead with many projects**
- Mitigation: Configurable poll interval, benchmark CPU usage
- Start with 10s, adjust if needed

**Risk: Cross-project locking conflicts**
- Mitigation: Accept duplication or use lockfile per project
- Low priority - multiple monitors won't harm

---

## Open Questions

1. **Overmind restart detection:** Can we distinguish deliberate `overmind restart api` from crash-restart? May need log parsing.

2. **Docker-compose integration:** Similar patterns would apply (poll `docker-compose ps`), but defer until overmind integration proven.

3. **Service health beyond process running:** Should we check HTTP endpoints or trust overmind? Could reuse existing TCP checks from `cmd/orch/status_cmd.go:1335`.

4. **Log snippet extraction:** On crash, show last 50 lines from `overmind echo <service>`. Requires parsing output or tmux capture.

---

## Recommendations

### For Orchestrator

**Promote to decision:** Yes. This establishes the architectural pattern for service observability (polling + notifications + dashboard integration) that should guide future docker-compose integration and other service managers.

**Next action:** Spawn feature-impl agent for Phase 1 MVP (service monitoring daemon with crash notifications).

### For Future Work

- **Docker-compose support:** Apply same phased approach (monitor → dashboard → events)
- **Service health checks:** Extend beyond process monitoring to HTTP endpoint checks
- **Log aggregation:** Centralized log viewer across all services and agents
- **Alert rules:** Configurable thresholds (e.g., notify after 3 restarts in 5 minutes)

---

## References

**Investigation:** `.kb/investigations/2026-01-09-inv-design-observability-infrastructure-overmind-docker.md`

**Related Artifacts:**
- `.kb/investigations/2026-01-09-inv-overmind-vs-launchd-prototype.md` - Why overmind was chosen
- `.kb/investigations/2025-12-23-inv-explore-options-centralized-server-management.md` - Earlier server management exploration

**Code References:**
- `pkg/events/logger.go:14-29` - Event types
- `pkg/notify/notify.go:52-65` - Notification system
- `cmd/orch/status_cmd.go:108-135` - InfrastructureHealth type
- `web/src/routes/+page.svelte:15-46` - Dashboard stores

---

## Conclusion

Service observability should extend existing agent observability patterns through three phases. Phase 1 MVP (background monitoring daemon with crash notifications) solves the immediate pain (silent crashes) with minimal complexity. Phase 2 (dashboard integration) provides cross-project visibility. Phase 3 (event streaming) adds audit trail if needed.

The phased approach allows validation at each step and delivers value incrementally. Reusing proven infrastructure (events, SSE, notifications, dashboard) reduces risk and maintains consistency with agent observability.
