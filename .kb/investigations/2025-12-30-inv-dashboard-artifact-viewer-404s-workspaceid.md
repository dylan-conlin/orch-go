<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Fixed 404 errors in dashboard artifact viewer by adding `extractWorkspaceName()` function that strips `[beads-id]` suffix from agent IDs before API calls.

**Evidence:** Browser testing confirmed SYNTHESIS.md now loads correctly for completed agents. Previous error was `GET /api/agents/artifact?workspace=og-debug-orch-review-hangs-30dec+%5Borch-go-9t43%5D` returning 404, now correctly uses just workspace name.

**Knowledge:** Agent IDs in dashboard follow format `"workspace-name [beads-id]"` (from session title), but artifact API expects just workspace name to locate files in `.orch/workspace/{name}/`.

**Next:** Close - fix implemented and verified via browser.

---

# Investigation: Dashboard Artifact Viewer 404s Workspaceid

**Question:** Why does the dashboard artifact viewer return 404 when trying to load SYNTHESIS.md for completed agents?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** og-debug-dashboard-artifact-viewer-30dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Agent IDs contain beads suffix from session title

**Evidence:** API response shows agent ID as `"og-debug-orch-review-hangs-30dec [orch-go-9t43]"` which is the full session title format `"workspace-name [beads-id]"`.

**Source:** 
- `cmd/orch/serve.go:814` - `ID: s.Title` sets agent ID to session title
- `cmd/orch/main.go:2276-2284` - `extractBeadsIDFromTitle()` parses the `[beads-id]` suffix

**Significance:** The agent ID in the API is not the same as the workspace directory name. The workspace directory is just `og-debug-orch-review-hangs-30dec` without the beads suffix.

---

### Finding 2: Artifact API uses workspace parameter to locate files

**Evidence:** The `/api/agents/artifact` endpoint joins workspaceID with project directories to find workspace path:
```go
candidatePath := filepath.Join(projectDir, ".orch", "workspace", workspaceID)
```

**Source:** `cmd/orch/serve.go:1296` - workspace lookup logic

**Significance:** When frontend passes `"og-debug-orch-review-hangs-30dec [orch-go-9t43]"`, the API looks for a directory that doesn't exist (includes URL-encoded brackets).

---

### Finding 3: Frontend was passing full agent ID to artifact API

**Evidence:** Console showed:
```
GET /api/agents/artifact?workspace=og-debug-orch-review-hangs-30dec+%5Borch-go-9t43%5D&type=synthesis 404
```
Where `%5B` and `%5D` are URL-encoded `[` and `]`.

**Source:** 
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte:323` - `workspaceId={$selectedAgent.id}`
- `web/src/lib/stores/agents.ts:637-645` - `fetchArtifact()` builds query params

**Significance:** The fix location is clear: strip the beads suffix in the frontend before making the API call.

---

## Synthesis

**Key Insights:**

1. **Session title format** - OpenCode session titles use format `"workspace-name [beads-id]"` to embed tracking info, but this creates a mismatch with filesystem expectations.

2. **Data flow disconnect** - Agent IDs flow from API → frontend → API artifact endpoint, but the ID format changes meaning between those hops.

3. **Simple extraction pattern** - The fix mirrors existing `extractBeadsIDFromTitle()` logic but extracts the workspace portion instead.

**Answer to Investigation Question:**

The 404 errors occur because the frontend passes the full agent ID (including `[beads-id]` suffix) to the artifact API, but the API expects just the workspace name to construct the filesystem path. Fixed by adding `extractWorkspaceName()` helper that strips the suffix before API calls.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build passes with no errors (verified: `bun run build` succeeded)
- ✅ Artifact viewer loads SYNTHESIS.md for completed agents (verified: browser test with Glass)
- ✅ Path displayed correctly in viewer without beads suffix

**What's untested:**

- ⚠️ Investigation and decision artifacts (only tested synthesis)
- ⚠️ Agents without beads suffix (legacy/untracked agents)

**What would change this:**

- Finding would be wrong if workspaces are actually named with brackets (unlikely, would break filesystem)
- Fix might need adjustment if agent ID format changes

---

## Implementation Recommendations

### Recommended Approach ⭐

**Frontend extraction** - Add `extractWorkspaceName()` function in agent-detail-panel.svelte

**Why this approach:**
- Fixes the issue at the point of consumption (where the mismatch occurs)
- Simple string manipulation matching existing patterns
- No backend changes required

**Trade-offs accepted:**
- Logic duplicated from backend (acceptable for frontend isolation)
- Only fixes this one component (but this is the only consumer)

**Implementation sequence:**
1. Add `extractWorkspaceName()` helper function
2. Call it when passing to ArtifactViewer component
3. Verify with browser test

### Alternative Approaches Considered

**Option B: Backend parsing of workspace parameter**
- **Pros:** Single fix point, handles all clients
- **Cons:** Backend shouldn't need to clean up frontend mistakes; API contract is clear
- **When to use instead:** If multiple frontend components have same issue

**Option C: Add separate workspace field to API response**
- **Pros:** Clean separation, explicit data
- **Cons:** Adds redundant field, changes API contract
- **When to use instead:** If agent ID vs workspace distinction is needed elsewhere

---

## References

**Files Examined:**
- `cmd/orch/serve.go:335-356, 1265-1340` - AgentAPIResponse struct and artifact handler
- `cmd/orch/main.go:2276-2284` - extractBeadsIDFromTitle() pattern
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte` - Frontend consumer
- `web/src/lib/stores/agents.ts:631-671` - fetchArtifact() function

**Commands Run:**
```bash
# Build verification
cd web && bun run build
# ✓ built in 7.03s

# Type check (pre-existing errors in theme.ts, unrelated)
cd web && bun run check
# 2 errors in theme.ts (pre-existing)
```

---

## Investigation History

**2025-12-30 23:17:** Investigation started
- Initial question: Why does artifact viewer return 404?
- Context: Console showed URL-encoded brackets in workspace parameter

**2025-12-30 23:22:** Root cause identified
- Agent ID = session title with [beads-id] suffix
- Artifact API expects just workspace name

**2025-12-30 23:25:** Fix implemented and verified
- Added extractWorkspaceName() helper
- Browser test confirmed fix works
- Status: Complete
