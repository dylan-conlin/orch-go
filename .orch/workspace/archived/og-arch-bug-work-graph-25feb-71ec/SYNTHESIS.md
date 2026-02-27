# Session Synthesis

**Agent:** og-arch-bug-work-graph-25feb-71ec
**Issue:** orch-go-1228
**Outcome:** success

---

## Plain-Language Summary

The work-graph UI shows 'unassigned' under cross-project in_progress issues (like toolshed-157) because the API function `buildActiveAgentMap()` only looks for active agents in the local project. It queries the local beads database (orch-go) and the default OpenCode session scope — both of which exclude cross-project agents. The graph correctly shows cross-project issues, but the agent enrichment step can't find agents working on them. The fix requires two changes: (1) use the existing `listSessionsAcrossProjects()` function for OpenCode session queries instead of `client.ListSessions("")`, and (2) extend `listTrackedIssues()` to query beads across all registered projects.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace. Key verification: after fix, `curl .../api/beads/graph?project_dir=.../toolshed` should return `active_agent` data for in_progress issues.

---

## TLDR

Root-caused the 'unassigned' bug to two scoping gaps in `buildActiveAgentMap()` (serve_beads.go:1047). Cross-project issues get graph nodes but no active agent data because (1) `listTrackedIssues()` only queries local beads and (2) `client.ListSessions("")` only returns default-project sessions. Fix is API-only — no frontend changes needed.

---

## Delta (What Changed)

### Files Created
- `.kb/models/dashboard-architecture/probes/2026-02-25-probe-work-graph-unassigned-cross-project.md` - Probe documenting root cause and recommended fix

### Files Modified
- None (investigation-only deliverable)

---

## Evidence (What Was Observed)

### Live API Verification
- **Cross-project (toolshed):** `toolshed-148` is `in_progress` with `active_agent: MISSING` in `/api/beads/graph?project_dir=.../toolshed`
- **Local (orch-go):** `orch-go-1228` and `orch-go-1229` both have populated `active_agent` with phase and model fields

### Code Trace (3 layers)
1. **serve_beads.go:920 `handleBeadsGraph()`** — correctly fetches cross-project issues via `getGraphIssues(projectDir)` using `beads.NewCLIClient(beads.WithWorkDir(projectDir))`
2. **serve_beads.go:1047 `buildActiveAgentMap()`** — two scoping gaps:
   - Line 1055: `globalTrackedAgentsCache.get(projectDirs)` → `queryTrackedAgents()` → `listTrackedIssues()` uses `beads.FindSocketPath("")` (local beads only)
   - Line 1069: `client.ListSessions("")` (default project sessions only)
3. **work-graph-tree-helpers.ts:309 `getInProgressSubline()`** — correctly falls through to 'unassigned' when `node.active_agent` is null. The fallback logic is correct; the data is wrong.

### Root Cause Location
- **Primary:** `cmd/orch/serve_beads.go:1069` — uses `client.ListSessions("")` instead of `listSessionsAcrossProjects()`
- **Secondary:** `cmd/orch/query_tracked.go:104` — `listTrackedIssues()` queries only local beads, doesn't accept projectDirs parameter

### Why Previous Fixes Didn't Stick
- orch-go-1183 fixed oscillation (tmux flakiness causing dashboard to alternate between correct/wrong status). Comment at serve_beads.go:1043 confirms. That was a real fix for a different bug — it didn't address the cross-project scoping gap.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `buildActiveAgentMap()` and `queryTrackedAgents()` are scoped to local project only — any cross-project issue will have no active_agent data in graph responses
- `listSessionsAcrossProjects()` already exists (serve_agents_cache.go:396) and is used in agents handlers but NOT in the graph handler

### Decisions Made
- Fix should be API-only (serve_beads.go + query_tracked.go), not frontend — the frontend logic is correct
- The existing `listSessionsAcrossProjects()` function should be reused, not a new implementation

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Fix buildActiveAgentMap() cross-project scoping
**Skill:** feature-impl
**Context:**
```
Root cause is in cmd/orch/serve_beads.go buildActiveAgentMap() (line 1047).
Two changes: (1) Replace client.ListSessions("") at line 1069 with
listSessionsAcrossProjects(client, sourceDir). (2) Extend listTrackedIssues()
in query_tracked.go to accept projectDirs and query each project's beads.
See probe: .kb/models/dashboard-architecture/probes/2026-02-25-probe-work-graph-unassigned-cross-project.md
```

---

## Unexplored Questions

- For claude-backend agents (no OpenCode session), does the workspace manifest lookup in `lookupManifestsAcrossProjects` already cover cross-project workspaces? If so, Change 1 alone might catch most cases.
- Are there toolshed issues that are in_progress without any agent at all (manually moved to in_progress)? If so, 'unassigned' might be correct for those — consider differentiating "no agent data available" from "no agent assigned."

---

## Session Metadata

**Skill:** architect
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-bug-work-graph-25feb-71ec/`
**Probe:** `.kb/models/dashboard-architecture/probes/2026-02-25-probe-work-graph-unassigned-cross-project.md`
**Beads:** `bd show orch-go-1228`
