# Session Synthesis

**Agent:** og-inv-dashboard-blind-claude-17feb-24d6
**Issue:** orch-go-1010
**Duration:** 2026-02-17T11:06 → 2026-02-17T11:30
**Outcome:** success

---

## Plain-Language Summary

The dashboard shows "No active agent linked" for Claude CLI tmux agents because of a **scan ordering bug** in `serve_agents.go`. The completed workspace scan (line 598) runs before the tmux window scan (line 551). When both find the same agent, the workspace scan claims it first as "completed" — and an optimization that skips beads enrichment for completed workspaces means phase comments are never read. The tmux scan then sees the beads_id already exists and defers, so the agent appears with wrong status, no phase, and no window link. A secondary issue is that the GraphNode API never populates `active_agent` and the WIP store's `setRunningAgents()` is a no-op, which causes "No active agent linked" even for OpenCode agents in the tree view (though those agents work correctly in the "Ready to Complete" section).

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root.

---

## Delta (What Changed)

### Files Created
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-17-dashboard-blind-to-tmux-agents.md` - Probe documenting root cause analysis with test evidence

### Files Modified
- None (investigation only — no code changes)

### Commits
- Pending: probe file + synthesis

---

## Evidence (What Was Observed)

1. **Tmux window exists but agent missing from dashboard** — `tmux list-windows -t workers-orch-go` shows `🔬 og-inv-dashboard-blind-claude-17feb-24d6 [orch-go-1010]` but `/api/agents` returns this agent with `status=completed, phase=null, window=null`.

2. **Beads comments exist but aren't read** — `bd comments orch-go-1010` shows "Phase: Planning" comment, but `/api/agents` returns `phase=null` for this agent. Root cause: beads_id not in `beadsIDsToFetch` (optimization skip at serve_agents.go:700-703).

3. **Workspace scan pre-empts tmux scan** — Code trace confirms: workspace scan at line 598 finds SPAWN_CONTEXT.md → adds as "completed" → tmux scan at line 551 finds beads_id match → skips. The order of operations determines which state layer claims the agent.

4. **`active_agent` never populated in graph API** — Backend `GraphNode` struct (serve_beads.go:596) lacks `active_agent` field. Frontend TypeScript interface has it (work-graph.ts:20) but backend never sends it.

5. **WIP store is stubbed** — `web/src/lib/stores/wip.ts:28-33`: `setRunningAgents()` sets store to empty array, so `runningAgentDetailsByIssueId` is always empty for ALL agents. The "No active agent linked" subline affects ALL in_progress issues (not just tmux), but OpenCode agents are partially rescued by the "Ready to Complete" section and attention badges.

6. **Confirmed on live system** — orch-go-1010 (this agent) is a Claude CLI tmux agent. Dashboard API returns wrong data. orch-go-1008 (parallel tmux agent) has same symptoms.

### Tests Run
```bash
# Tmux window verification
tmux list-windows -t workers-orch-go -F '#{window_name}' | grep 1010
# Output: 🔬 og-inv-dashboard-blind-claude-17feb-24d6 [orch-go-1010]

# API response verification
curl -sk 'https://localhost:3348/api/agents' | python3 -c "..."
# Agent found with status=completed, phase=null, window=null

# Beads comments verification
bd comments orch-go-1010
# Phase: Planning comment exists but API doesn't see it

# Graph API verification
curl -sk 'https://localhost:3348/api/beads/graph?status=open' | python3 -c "..."
# All in_progress nodes have active_agent=NONE
```

---

## Knowledge (What Was Learned)

### Root Cause: Three Bugs Compound

| Bug | Location | Impact |
|-----|----------|--------|
| **Scan ordering** | serve_agents.go:598 before :551 | Workspace scan claims tmux agents as "completed" |
| **Beads enrichment skip** | serve_agents.go:700-703 | Phase comments never fetched for workspace-scan entries |
| **active_agent never populated** | serve_beads.go:596-606 | Graph API nodes lack agent data for tree sublines |

### Contributing Factor: WIP Store Stub

The WIP store's `setRunningAgents()` is a no-op (`wip.ts:28-33`), making `runningAgentDetailsByIssueId` always empty. This affects ALL agents (not just tmux) but is masked for OpenCode agents by the "Ready to Complete" section and attention badges.

### Constraints Discovered
- **Scan ordering determines agent identity** — whichever scan finds the agent first wins the duplicate check. This is an undocumented property of the multi-source reconciliation model.
- **Completed workspace optimization is aggressive** — Assumes all workspaces with SPAWN_CONTEXT.md are done. True for archived workspaces, false for actively-running tmux agents.

### Externalized via `kb`
- See probe file for model impact (extends agent-lifecycle-state-model with new failure mode)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Recommended Fix (3 changes, in priority order)

**Fix 1 (Critical): Reorder scans + fix duplicate check** — Move tmux scan (line 551-596) BEFORE completed workspace scan (line 598-707). Add beads_id matching to workspace scan's duplicate check (currently only matches workspace name, which doesn't match tmux window names with emoji prefix). This is the minimal fix that makes tmux agents visible.

**Fix 2 (Important): Populate active_agent in GraphNode** — In `handleBeadsGraph()`, join the beads graph nodes with agent data (from `/api/agents` or the same data source) by beads_id. Set `active_agent.phase`, `.runtime`, `.model` on graph nodes. This gives the frontend's `getInProgressSubline()` its primary signal path.

**Fix 3 (Nice-to-have): Implement WIP store setRunningAgents()** — Transform the `$agents` array into `wipItems` by mapping agents with beads_id to WIPItem format. This populates `runningAgentDetailsByIssueId` and fixes "No active agent linked" for ALL agents, not just tmux.

### Spawn Follow-up Issue
**Title:** Fix dashboard blind to tmux agents: reorder scans in serve_agents.go
**Skill:** feature-impl
**Context:**
```
Root cause documented in .kb/models/agent-lifecycle-state-model/probes/2026-02-17-dashboard-blind-to-tmux-agents.md.
Three changes needed: (1) Move tmux scan before workspace scan in serve_agents.go, (2) Add beads_id to workspace scan duplicate check, (3) Optionally populate active_agent in beads/graph API.
```

---

## Unexplored Questions

- **Should workspace scan check for active tmux windows?** Instead of assuming all orphan workspaces are completed, could check if a matching tmux window exists and skip those. More defensive but adds tmux dependency to workspace scan.
- **Is the "completed" assumption ever wrong for non-tmux workspaces?** If an OpenCode session crashes mid-execution, the workspace has SPAWN_CONTEXT.md but is NOT completed. This may cause the same phantom-completed behavior.
- **Should active_agent be populated server-side or client-side?** Server-side join in handleBeadsGraph is simpler but adds latency. Client-side join in the Svelte store would use existing $agents data without extra API work.

---

## Session Metadata

**Skill:** investigation
**Model:** Claude Opus (via Claude Code CLI)
**Workspace:** `.orch/workspace/og-inv-dashboard-blind-claude-17feb-24d6/`
**Probe:** `.kb/models/agent-lifecycle-state-model/probes/2026-02-17-dashboard-blind-to-tmux-agents.md`
**Beads:** `bd show orch-go-1010`
