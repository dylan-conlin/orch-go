## Summary (D.E.K.N.)

**Delta:** Implemented dashboard visibility for orchestrator sessions with visual distinction from worker agents.

**Evidence:** API endpoint returns valid session data (tested with curl), frontend build succeeds, components integrate correctly.

**Knowledge:** Orchestrator sessions are stored in ~/.orch/sessions.json registry and tracked separately from worker agents.

**Next:** Close - implementation complete, visual verification deferred to orchestrator.

---

# Investigation: Dashboard Visibility Orchestrator Sessions

**Question:** How to show orchestrator sessions distinctly from worker agents in the web dashboard?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Orchestrator Session Data Structure

**Evidence:** Registry stores sessions with workspace_name, session_id, goal, project_dir, spawn_time, and status fields. The CLI `orch status` already has a getOrchestratorSessions function that queries this registry.

**Source:** pkg/session/registry.go:29-48, cmd/orch/status_cmd.go:600-631

**Significance:** Reusable data structure and query pattern for the API endpoint.

---

### Finding 2: Visual Distinction Pattern

**Evidence:** Worker agents use green (active), yellow (running), blue (ready-for-review), red (abandoned) color scheme. Using purple for orchestrators provides clear visual distinction without conflicting with existing semantics.

**Source:** web/src/lib/components/agent-card/agent-card.svelte:209

**Significance:** Purple color scheme with "O" badge and border-2 styling makes orchestrators immediately identifiable.

---

### Finding 3: Child Agent Count

**Evidence:** API can count active agents per project by querying OpenCode sessions and extracting project from beads ID. This provides "child_agent_count" field showing active workers in same project as orchestrator.

**Source:** cmd/orch/serve_system.go (handleOrchestratorSessions function)

**Significance:** Shows orchestrator's current coordination load without needing explicit parent-child tracking.

---

## Synthesis

**Key Insights:**

1. **Separate Display Layer** - Orchestrator sessions appear in their own section at top of dashboard, always visible regardless of operational/historical mode.

2. **Visual Hierarchy** - Purple color scheme with "O" badge differentiates orchestrators from worker agents (green/yellow/blue).

3. **Contextual Information** - Shows goal prominently (main purpose), duration, project, and count of child agents in same project.

**Answer to Investigation Question:**

Orchestrator sessions are shown via a new OrchestratorSessionsSection component that renders at the top of the dashboard. Visual distinction achieved through:
- Purple border/badge color scheme (distinct from worker green/yellow/blue/red)
- "O" badge indicating orchestrator type
- Larger card format with goal prominently displayed
- Child agent count showing coordination scope

---

## Structured Uncertainty

**What's tested:**

- ✅ API endpoint returns valid data (verified: curl http://localhost:3348/api/orchestrator-sessions)
- ✅ Frontend build succeeds without errors (verified: bun run build)
- ✅ Go build succeeds (verified: go build ./cmd/orch/)

**What's untested:**

- ⚠️ Visual appearance in browser (no Playwright MCP access for screenshot)
- ⚠️ Responsive behavior at 666px width constraint

**What would change this:**

- If visual rendering has issues, styling adjustments may be needed
- If child_agent_count calculation is inaccurate, may need beads-based tracking

---

## Implementation Details

**Files created:**
- cmd/orch/serve_system.go - handleOrchestratorSessions handler (+104 lines)
- cmd/orch/serve.go - endpoint registration (+4 lines)
- web/src/lib/stores/orchestrator-sessions.ts - Svelte store (+68 lines)
- web/src/lib/components/orchestrator-session-card/ - Card component (+137 lines)
- web/src/lib/components/orchestrator-sessions-section/ - Section component (+42 lines)
- web/src/routes/+page.svelte - Integration (+14 lines)

**Success criteria:**
- ✅ Orchestrator sessions visible in dashboard
- ✅ Visual distinction from worker agents
- ✅ Goal and duration prominently displayed
- ✅ Child agent count shown

---

## References

**Files Examined:**
- cmd/orch/status_cmd.go - Existing getOrchestratorSessions pattern
- pkg/session/registry.go - OrchestratorSession data structure
- web/src/lib/components/agent-card/agent-card.svelte - Worker card styling pattern
- web/src/routes/+page.svelte - Dashboard integration points

**Commands Run:**
```bash
# Test API endpoint
curl http://localhost:3348/api/orchestrator-sessions | jq .

# Build verification
go build ./cmd/orch/
cd web && bun run build
```

---

## Investigation History

**2026-01-05 15:10:** Investigation started
- Initial question: How to show orchestrator sessions in dashboard with visual distinction
- Context: Spawned from beads issue orch-go-k300.8

**2026-01-05 15:45:** Implementation complete
- Status: Complete
- Key outcome: Dashboard shows orchestrator sessions with purple styling, goal/duration/child count
