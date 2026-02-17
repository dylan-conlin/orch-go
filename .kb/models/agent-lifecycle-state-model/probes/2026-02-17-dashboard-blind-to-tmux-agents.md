# Probe: Dashboard Blind to Claude CLI Tmux Agents

**Model:** agent-lifecycle-state-model
**Date:** 2026-02-17
**Status:** Complete

---

## Question

The model claims "Multiple sources must be reconciled" (Invariant 5) and that the dashboard uses a Priority Cascade to reconcile four state layers. Does this reconciliation work correctly for Claude CLI tmux agents, or does a scan-ordering bug prevent tmux agents from being visible?

---

## What I Tested

### Test 1: Verify tmux window exists but agent missing from dashboard

```bash
# Confirmed tmux window exists for active agent
tmux list-windows -t workers-orch-go -F '#{window_name}' | grep 1010
# Output: 🔬 og-inv-dashboard-blind-claude-17feb-24d6 [orch-go-1010]

# Checked /api/agents (default 12h filter)
curl -sk 'https://localhost:3348/api/agents' | python3 -c "
import sys, json; data = json.load(sys.stdin)
a = [x for x in data if x.get('beads_id')=='orch-go-1010']
print(json.dumps(a[0], indent=2) if a else 'NOT FOUND')"
```

**Result:** Agent found but with WRONG data:
```json
{
  "id": "og-inv-dashboard-blind-claude-17feb-24d6",
  "beads_id": "orch-go-1010",
  "skill": "investigation",
  "status": "completed",     // WRONG - agent is actively running
  "phase": null,              // WRONG - Phase: Planning was reported
  "window": null,             // WRONG - tmux window exists
  "updated_at": "2026-02-17T11:06:00-08:00",
  "spawned_at": "2026-02-17T11:06:00-08:00"
}
```

### Test 2: Verify beads comments exist with Phase: data

```bash
bd comments orch-go-1010
# Output confirms: "Phase: Planning - Tracing dashboard data sources..."
```

Phase: Planning comment EXISTS in beads but is NOT reflected in the API response.

### Test 3: Verify graph API never populates active_agent

```bash
curl -sk 'https://localhost:3348/api/beads/graph?status=open' | python3 -c "
import sys, json; data = json.load(sys.stdin)
nodes = [n for n in data['nodes'] if n['status']=='in_progress']
for n in nodes: print(f\"{n['id']}: active_agent={n.get('active_agent', 'NONE')}\")"
```

**Result:** ALL in_progress nodes have `active_agent=NONE`. Backend `GraphNode` Go struct lacks this field entirely.

### Test 4: Code trace of the scan ordering bug

Read `cmd/orch/serve_agents.go` and traced the data flow:

1. **Line 449-549**: OpenCode session scan (tmux agent NOT found here - no OpenCode session)
2. **Line 598-707**: Completed workspace scan finds `.orch/workspace/og-inv-dashboard-blind-claude-17feb-24d6/` with SPAWN_CONTEXT.md → marks as `"completed"`. Does NOT add beads_id to `beadsIDsToFetch` (line 700-703 optimization comment: "completed workspaces don't need phase updates").
3. **Line 551-596**: Tmux scan finds window, extracts beads_id `orch-go-1010`, BUT duplicate check at line 567 finds `a.BeadsID == "orch-go-1010"` already in list → `alreadyIn = true` → skips.
4. **Line 712-898**: Beads enrichment runs but `orch-go-1010` is NOT in `beadsIDsToFetch` → no phase data fetched.

**Root cause chain:**
```
Workspace scan runs FIRST → claims workspace as "completed"
  → Tmux scan finds matching beads_id → skips (duplicate)
    → Beads enrichment skipped (optimization for completed workspaces)
      → No phase, no window, wrong status
```

### Test 5: WIP store is stubbed

Read `web/src/lib/stores/wip.ts`:
```typescript
setRunningAgents: (agents: any[]) => {
    console.warn('wip store: setRunningAgents not implemented');
    set([]);  // Always empty!
},
```

This means `runningAgentDetailsByIssueId` in work-graph-tree is ALWAYS empty for ALL agents (not just tmux). The "No active agent linked" text appears for any in_progress issue without an attention badge.

---

## What I Observed

### Three distinct bugs combine to create the "blind dashboard" symptom:

**Bug 1: Scan ordering** — Completed workspace scan (line 598) runs before tmux scan (line 551). Workspace scan finds SPAWN_CONTEXT.md and assumes workspace is completed. Tmux scan defers to the already-existing entry. The tmux agent is never registered as "active".

**Bug 2: Beads enrichment skipped** — The optimization at line 700 ("completed workspaces don't need phase updates") means beads comments are never fetched for workspace-scan agents. Phase: Planning/Implementing/Complete comments are invisible.

**Bug 3: active_agent never populated** — The backend `GraphNode` struct (serve_beads.go:596) lacks `active_agent` field. The frontend interface has it (work-graph.ts:20) but it's never populated. Combined with the stubbed WIP store (wip.ts:28), the work-graph tree has NO way to link agents to issues for the in-progress subline.

### Quantified impact:

- **Tmux agents**: Appear as "completed" with no phase, no window. Dashboard completely blind.
- **OpenCode agents**: Appear with correct status but tree rows still show "No active agent linked" (WIP store stubbed). Only "Ready to Complete" section works (direct $agents join).
- **Cross-project**: Confirmed same behavior on both orch-go and price-watch dashboards.

---

## Model Impact

- [x] **Confirms** invariant: "Multiple sources must be reconciled" (Invariant 5) — the bug is precisely a reconciliation failure between workspace files and tmux windows.
- [x] **Confirms** invariant: "Tmux windows are UI layer only" (Invariant 6) — tmux data IS treated as low authority, but the problem is it never gets a chance to be consulted.
- [x] **Contradicts** the Priority Cascade model claim at `dashboard-agent-status.md` — the cascade assumes all agents reach the enrichment phase, but workspace-scan agents skip beads enrichment entirely. The Priority Cascade is correct in theory but never runs for tmux agents.
- [x] **Extends** model with: **Scan ordering determines which state layer claims the agent first**, and the first claim wins via the duplicate check. This is an undocumented dimension of the reconciliation model. The model describes the Priority Cascade for status CALCULATION but not for agent DISCOVERY.

### New failure mode to add to model:

**Failure Mode 5: Workspace Scan Pre-empts Tmux Discovery**

**Symptom:** Dashboard shows tmux agent as "completed" with no phase, no window, wrong status.

**Root cause:** Completed workspace scan runs before tmux scan. Both find the same agent (by beads_id), but workspace scan claims it first as "completed" without beads enrichment.

**Why it happens:**
- Claude CLI agent creates workspace (SPAWN_CONTEXT.md) AND tmux window
- Workspace scan at line 598 finds SPAWN_CONTEXT.md → adds as "completed"
- Workspace scan optimization: does NOT add beads_id to fetch list
- Tmux scan at line 551 finds window → duplicate check matches beads_id → skips
- Beads enrichment never runs for this agent
- Result: status="completed", phase=null, window=null

**Fix:** Move tmux scan before completed workspace scan + add beads_id matching to workspace scan's duplicate check.

---

## Notes

### Minimal fix (architectural recommendation):

1. **Reorder scans**: Move tmux scan (line 551-596) BEFORE completed workspace scan (line 598-707). This ensures tmux agents are discovered with correct "active" status and their beads_ids are added to the enrichment queue.

2. **Fix workspace scan duplicate check**: Add beads_id matching (currently only checks workspace name, which doesn't match tmux window names that include emoji prefix and beads_id suffix).

3. **Populate active_agent in GraphNode**: Add agent data to the beads/graph response by joining agents list with graph nodes by beads_id. This gives the frontend a second signal path.

4. **Implement WIP store's setRunningAgents()**: Transform the agents list into wipItems so `runningAgentDetailsByIssueId` gets populated. This fixes "No active agent linked" for ALL agents, not just tmux.

### Escalation items:
- Fix #1-2 are within worker authority (bug fix within scope)
- Fix #3-4 are architectural scope expansion — escalate to orchestrator
