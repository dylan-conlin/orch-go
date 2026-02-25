# Probe: Work Graph 'unassigned' Label on Cross-Project In-Progress Issues

**Status:** Complete
**Date:** 2026-02-25
**Model:** Dashboard Architecture
**Issue:** orch-go-1228

## Question

Why does the work-graph UI show 'unassigned' on in_progress cross-project issues (e.g., toolshed-157) even when an agent is working on them? Does `buildActiveAgentMap()` include cross-project agents?

## What I Tested

### Test 1: API response for cross-project issues
```bash
curl -sk 'https://localhost:3348/api/beads/graph?project_dir=.../toolshed' | python3 -c "
import json, sys
data = json.load(sys.stdin)
for node in data.get('nodes', []):
    if node.get('status') == 'in_progress':
        print(f'ID: {node[\"id\"]}, active_agent: {node.get(\"active_agent\", \"MISSING\")}')"
```
**Output:** `ID: toolshed-148, Status: in_progress, active_agent: MISSING`

### Test 2: API response for local issues (control)
```bash
curl -sk 'https://localhost:3348/api/beads/graph' | python3 -c "..."
```
**Output:**
- `orch-go-1229: active_agent: {'phase': 'Planning - ...', 'model': 'anthropic/claude-opus-4-5-20251101'}`
- `orch-go-1228: active_agent: {'phase': 'Planning - ...', 'model': 'anthropic/claude-opus-4-5-20251101'}`

### Test 3: Traced data flow through code

**Layer 1 - Beads (data source):**
- `handleBeadsGraph()` in `serve_beads.go:920` fetches graph issues via `getGraphIssues(projectDir)` — correctly queries cross-project beads using `beads.NewCLIClient(beads.WithWorkDir(projectDir))`.
- Toolshed issues ARE returned in graph nodes. ✓

**Layer 2 - Active Agent Enrichment (`buildActiveAgentMap()` at serve_beads.go:1047):**
- Step 1: `globalTrackedAgentsCache.get(projectDirs)` → calls `queryTrackedAgents()` → calls `listTrackedIssues()` which queries **only local beads** via `beads.FindSocketPath("")`. Cross-project issues (toolshed-*) are NOT in local beads. ✗
- Step 2: `client.ListSessions("")` → queries OpenCode sessions **only for default project**. Cross-project sessions invisible. ✗
- Step 3/4: Only enriches beads IDs already in the result map. Toolshed IDs never enter the map.

**Layer 3 - Frontend (`getInProgressSubline` at work-graph-tree-helpers.ts:309):**
- Checks `node.active_agent?.phase || activeAgent?.phase` etc.
- Both are null for toolshed issues → falls through to `return { text: 'unassigned' }` at line 341. ✗

## What I Observed

**Root cause: Two scoping gaps in `buildActiveAgentMap()` (serve_beads.go:1047-1118):**

1. **`listTrackedIssues()` (query_tracked.go:104)** uses `beads.FindSocketPath("")` which connects to the local beads database only. Cross-project issues live in their own beads databases and are invisible. The function doesn't accept or use `projectDirs`.

2. **`client.ListSessions("")` (serve_beads.go:1069)** queries OpenCode with the default project scope. The existing `listSessionsAcrossProjects()` function (serve_agents_cache.go:396) was created to solve exactly this problem but is NOT used in `buildActiveAgentMap()`.

**Why previous fixes didn't stick:**
- orch-go-1183 fixed dashboard oscillation by consolidating `buildActiveAgentMap()` to use `globalTrackedAgentsCache` instead of independent tmux queries. This was a real fix for the oscillation bug, but it didn't address the cross-project scoping gap.
- The oscillation fix comment at serve_beads.go:1043 confirms this was about tmux flakiness, not cross-project data.

**Evidence that the fix is in the API, not frontend:**
- `getInProgressSubline` correctly shows agent info when `node.active_agent` has data (orch-go issues: ✓)
- `getInProgressSubline` correctly falls through to "unassigned" when no data exists (toolshed issues: ✗ — but the fallback logic is correct, it's the data that's wrong)

## Model Impact

**Extends** the dashboard-architecture model:

- **New invariant:** `buildActiveAgentMap()` is scoped to local project only. Cross-project graph requests get nodes but NOT active agent enrichment. This creates a systemic data gap where any cross-project in_progress issue will show 'unassigned'.

- **Confirms** the existing "Cross-Project Agents Not Visible" failure mode in the agent-lifecycle-state-model, but identifies a new manifestation: it's not just the dashboard agent list — the work-graph's active_agent enrichment has the same gap.

- **Existing fix pattern exists but isn't applied:** `listSessionsAcrossProjects()` was created for exactly this purpose in the agents handler but was never ported to `buildActiveAgentMap()`.

## Recommended Fix

**File: `cmd/orch/serve_beads.go`, function `buildActiveAgentMap()` (line 1047)**

Two changes needed:

### Change 1: Use cross-project session listing (line 1069)
Replace:
```go
client := opencode.NewClient(serverURL)
sessions, err := client.ListSessions("")
```
With:
```go
client := opencode.NewClient(serverURL)
sessions, err := listSessionsAcrossProjects(client, sourceDir)
```

### Change 2: Extend `listTrackedIssues()` to query cross-project beads
In `query_tracked.go`, `listTrackedIssues()` needs to accept `projectDirs` and query each project's beads for `orch:agent` labeled issues, then merge results.

### Change 3 (Optional, frontend hardening): Better fallback label
In `work-graph-tree-helpers.ts:340-343`, consider changing `'unassigned'` to `'no agent data'` for in_progress issues that legitimately have no agent (e.g., manually set to in_progress).

**Architectural note:** This is a hotspot area (work-graph-tree.svelte flagged as hotspot). Changes 1-2 are API-only and don't touch the hotspot file. Change 3 is optional and touches only the helpers file (not the hotspot svelte file).
