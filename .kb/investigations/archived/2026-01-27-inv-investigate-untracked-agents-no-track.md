<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Untracked agents ARE returned by the API but filtered out by the dashboard's project filter because their project name is `{project}-untracked` (e.g., `orch-go-untracked`) which is not in the orchestrator's `included_projects` list.

**Evidence:** API without filter returns 27 agents including 12 untracked; API with dashboard's project filter returns only 4 agents (0 untracked). Verified via curl tests comparing `/api/agents` vs `/api/agents?project=orch-go,orch-cli,...`.

**Knowledge:** This is intentional design, not a bug. Untracked spawns (--no-track) are test/ad-hoc spawns that should be excluded from production views by default. The `orch frontier` CLI shows them because it doesn't use project filtering.

**Next:** Document this as expected behavior. If users want to see untracked agents in dashboard, they can disable "Follow Orchestrator" filter or add `{project}-untracked` to included_projects.

**Promote to Decision:** recommend-no (working as designed - untracked spawns are intentionally excluded from production dashboard views)

---

# Investigation: Why Untracked Agents Not Visible in Dashboard UI

**Question:** Why are untracked agents (--no-track spawns) not visible in the dashboard UI despite being returned by the /api/agents endpoint?

**Started:** 2026-01-27
**Updated:** 2026-01-27
**Owner:** orch-go investigation
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Project Filter Mechanism in Dashboard

**Evidence:** The dashboard has `followOrchestrator: true` enabled by default (line 32 in `web/src/lib/stores/context.ts`). When enabled, it:
1. Polls `/api/context` to get the orchestrator's current project context
2. Builds a query string like `?project=orch-go,orch-cli,beads,...` from `included_projects`
3. Sends this filter to `/api/agents` endpoint

**Source:**
- `web/src/lib/stores/context.ts:30-33` - Default filter state with `followOrchestrator: true`
- `web/src/lib/stores/context.ts:163-181` - `buildFilterQueryString()` function
- `web/src/routes/+page.svelte:245-249` - Reactive block that updates project filter from orchestrator context

**Significance:** The dashboard actively filters agents by project name, and this filter is derived from the orchestrator's context.

---

### Finding 2: Untracked Agent Project Names Don't Match Filter

**Evidence:**
- Beads ID for untracked spawns: `orch-go-untracked-1769534403`
- `extractProjectFromBeadsID()` (line 130-142 in `cmd/orch/shared.go`) extracts project as `orch-go-untracked`
- Orchestrator context returns `included_projects: ["orch-go", "orch-cli", "beads", ...]` - note NO `orch-go-untracked`
- When API filters by project, `orch-go-untracked != orch-go`, so untracked agents are excluded

**Source:**
- `cmd/orch/shared.go:130-142` - `extractProjectFromBeadsID()` function
- `cmd/orch/serve_agents.go:1037-1056` - Project filtering logic
- `curl -sk https://localhost:3348/api/context` output showing `included_projects`

**Significance:** The project name extraction creates a distinct project (`orch-go-untracked`) that is intentionally not in the orchestrator's project list.

---

### Finding 3: Verified via API Tests

**Evidence:**
```
Without project filter: 27 agents (including 12 untracked)
With project filter: 4 agents (0 untracked)
```

Test commands:
```bash
# Without filter - shows untracked
curl -sk https://localhost:3348/api/agents | jq -r '.[] | select(.project | contains("untracked"))'
# Returns: 12 agents with project=orch-go-untracked or opencode-untracked

# With dashboard's filter - no untracked
curl -sk "https://localhost:3348/api/agents?project=orch-go,orch-cli,beads,kb-cli,orch-knowledge,opencode" | jq length
# Returns: 4 (only tracked agents matching included_projects)
```

**Source:** Live API testing during investigation

**Significance:** Confirms the root cause is project filtering, not rendering issues or frontend bugs.

---

## Synthesis

**Key Insights:**

1. **Intentional Design** - Untracked spawns (--no-track) are test/ad-hoc spawns explicitly designed to be excluded from production views. The beads ID format `{project}-untracked-{timestamp}` creates a distinct project name that isn't in the orchestrator's tracked projects.

2. **CLI vs Dashboard Difference** - `orch frontier` shows untracked agents because it queries tmux windows and OpenCode sessions directly without project filtering. The dashboard uses project filtering by default via `followOrchestrator: true`.

3. **User Control Exists** - Users can see untracked agents by either:
   - Disabling "Follow Orchestrator" mode in dashboard settings
   - Removing the project filter via URL query params
   - Using `orch frontier` CLI instead

**Answer to Investigation Question:**

Untracked agents are not visible in the dashboard UI because:
1. Dashboard has `followOrchestrator: true` by default
2. This creates a project filter from `included_projects` (e.g., `orch-go,orch-cli,...`)
3. Untracked agents have project names like `orch-go-untracked` which don't match the filter
4. The API correctly filters them out

This is **working as designed** - untracked spawns are intentionally excluded from production dashboard views to reduce noise from test/ad-hoc spawns.

---

## Structured Uncertainty

**What's tested:**

- ✅ API returns untracked agents when no project filter applied (verified: curl without query params)
- ✅ API filters out untracked agents when project filter applied (verified: curl with ?project=...)
- ✅ Dashboard sends project filter via `followOrchestrator` mode (verified: code review of context.ts)
- ✅ `orch frontier` shows untracked agents (verified: ran command, saw 8 active including untracked)

**What's untested:**

- ⚠️ What happens when user disables "Follow Orchestrator" in dashboard UI (not tested interactively)
- ⚠️ Whether adding `{project}-untracked` to included_projects would show them (not tested)

**What would change this:**

- Finding would be wrong if dashboard showed untracked agents with followOrchestrator=true
- Finding would be wrong if untracked agents had project name matching the base project (e.g., `orch-go` instead of `orch-go-untracked`)

---

## Implementation Recommendations

**Purpose:** No code changes required - this is working as designed.

### Recommended Approach: Document as Expected Behavior

**Why this approach:**
- Untracked spawns are intentionally test/ad-hoc spawns
- Excluding them from production views reduces noise
- Users have workarounds if they need to see them

**Workarounds for users who want to see untracked agents:**
1. Use `orch frontier` CLI (shows all agents regardless of project)
2. Disable "Follow Orchestrator" in dashboard settings
3. Access API directly without project filter

### Alternative Approaches Considered

**Option B: Add `{project}-untracked` to included_projects**
- **Pros:** Would show untracked in dashboard
- **Cons:** Defeats purpose of separating test spawns from production
- **When to use:** Never recommended as default; users can do this manually if needed

**Option C: Add "Show Untracked" toggle to dashboard**
- **Pros:** User control without changing default behavior
- **Cons:** Additional UI complexity for edge case
- **When to use:** If users frequently request this visibility

---

## References

**Files Examined:**
- `web/src/lib/stores/context.ts` - Filter state management and query string building
- `web/src/lib/stores/agents.ts` - Agent store and fetch logic
- `web/src/routes/+page.svelte` - Dashboard page and filter application
- `cmd/orch/serve_agents.go` - API endpoint and project filtering
- `cmd/orch/shared.go` - `extractProjectFromBeadsID()` function
- `cmd/orch/frontier.go` - CLI frontier command (no project filtering)
- `.kb/models/dashboard-architecture.md` - Dashboard architecture reference
- `.kb/models/dashboard-agent-status.md` - Agent status calculation reference

**Commands Run:**
```bash
# Check orchestrator context
curl -sk https://localhost:3348/api/context | jq .

# Test API without filter
curl -sk https://localhost:3348/api/agents | jq length
# Result: 27

# Test API with dashboard's filter
curl -sk "https://localhost:3348/api/agents?project=orch-go,orch-cli,beads,kb-cli,orch-knowledge,opencode,price-watch,pw" | jq length
# Result: 4

# Check orch frontier output
orch frontier
# Result: Shows 8 ACTIVE including untracked agents
```

**Related Artifacts:**
- **Model:** `.kb/models/dashboard-architecture.md` - Documents two-mode design and filtering
- **Investigation:** `.kb/investigations/2025-12-26-inv-investigate-untracked-agents-lingering-orch.md` - Previous untracked agent investigation (different issue)

---

## Investigation History

**2026-01-27 ~12:30:** Investigation started
- Initial question: Why aren't untracked agents visible in dashboard despite being in API response?
- Context: User observed 6 untracked agents in `orch frontier` but not in dashboard

**2026-01-27 ~12:45:** Root cause identified
- Dashboard uses `followOrchestrator` filter by default
- Untracked agents have project names like `orch-go-untracked`
- These don't match the `included_projects` filter

**2026-01-27 ~12:50:** Verified via API tests
- Confirmed 27 agents without filter vs 4 with filter
- All 12 untracked agents filtered out when project filter applied

**2026-01-27 ~12:55:** Investigation completed
- Status: Complete
- Key outcome: Working as designed - untracked spawns intentionally excluded from production dashboard views
