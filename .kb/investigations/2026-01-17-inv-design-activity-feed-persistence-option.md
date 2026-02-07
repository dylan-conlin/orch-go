<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The "Black Box Recorder" (Option C) should write ACTIVITY.json to workspace on agent completion, not during execution, with OpenCode API as primary source.

**Evidence:** OpenCode already persists all session data to disk and exposes via API; current hybrid SSE+API architecture works; workspace-based persistence provides archival value but would duplicate live data.

**Knowledge:** Activity persistence should be a two-tier system: (1) OpenCode API for live/recent sessions, (2) ACTIVITY.json export on completion for archival. Visual hierarchy from gy1o4.1.3 is partially implemented but needs refinement.

**Next:** Implement completion-time ACTIVITY.json export in orch complete/clean commands; enhance visual hierarchy in activity-tab.svelte.

**Promote to Decision:** Actioned - dashboard patterns documented in dashboard guide

---

# Investigation: Design Activity Feed Persistence Option C + Visual Hierarchy

**Question:** How should "Black Box Recorder" (Option C) persist activity to workspace, and how should visual hierarchy differentiate reasoning vs execution?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - Ready for implementation
**Status:** Complete

**Patches-Decision:** None
**Extracted-From:** Task orch-go-gy1o4.1.5
**Supersedes:** None
**Superseded-By:** None

---

## Findings

### Finding 1: OpenCode Already Persists Complete Activity History

**Evidence:** OpenCode stores all session data to `~/.local/share/opencode/storage/`:
- Sessions: `storage/session/{projectID}/{sessionID}.json`
- Messages: `storage/message/{sessionID}/{messageID}.json`
- Parts: `storage/part/{messageID}/{partID}.json`

The API endpoint `GET /session/:sessionID/message` returns complete message history with all parts.

**Source:**
- `~/.local/share/opencode/storage/` (verified via `ls`)
- `cmd/orch/serve_agents.go:1390-1420` (handleSessionMessages proxy)
- `.kb/investigations/2026-01-07-design-dashboard-activity-feed-persistence.md`

**Significance:** Writing ACTIVITY.json during agent execution would duplicate data that OpenCode already persists. The value of workspace-based persistence is for archival (survives OpenCode reinstall, portable with workspace).

---

### Finding 2: Current Hybrid SSE+API Architecture Works for Live/Recent Agents

**Evidence:** The frontend already implements hybrid loading:
1. `sessionHistory.fetchHistory(sessionId)` fetches from `/api/session/{sessionId}/messages`
2. Real-time SSE events are merged with historical events
3. Deduplication by event ID prevents duplicates
4. Per-session caching prevents redundant API calls

**Source:**
- `web/src/lib/stores/agents.ts:442-518` (sessionHistory store)
- `web/src/lib/components/agent-detail/activity-tab.svelte:42-75` (fetchHistoricalEvents)
- `web/src/lib/components/agent-detail/activity-tab.svelte:220-244` (mergedEvents)

**Significance:** The primary persistence mechanism (OpenCode API) is already implemented and functional. "Black Box Recorder" adds value only for archival/offline scenarios.

---

### Finding 3: Workspace Structure Supports ACTIVITY.json

**Evidence:** Workspaces already contain agent artifacts:
```
.orch/workspace/{name}/
├── AGENT_MANIFEST.json    # Agent metadata
├── SPAWN_CONTEXT.md       # Spawn context
├── SYNTHESIS.md           # Completion summary (for full tier)
├── .beads_id              # Beads issue link
├── .session_id            # OpenCode session link
└── screenshots/           # Visual verification
```

Adding `ACTIVITY.json` follows the established pattern of workspace-centric artifacts.

**Source:** `ls -la .orch/workspace/og-feat-implement-tiered-stuck-17jan-41c0/`

**Significance:** The workspace is already the canonical location for agent artifacts. ACTIVITY.json belongs here for archival, not in a separate location.

---

### Finding 4: Visual Hierarchy Partially Implemented

**Evidence:** Activity feed already has some visual differentiation:
- Reasoning: `text-muted-foreground/70`, bullet prefix (`•`)
- Tool calls: `text-blue-400` for tool name, `font-semibold`
- Results: Nested in expanded view with monospace

What's missing from gy1o4.1.3 design:
- Step events lack distinct styling
- Tool results could use more muted color
- No visual distinction between successful vs failed tool calls

**Source:**
- `web/src/lib/components/agent-detail/activity-tab.svelte:567-584` (current implementation)
- `.kb/investigations/2026-01-17-inv-visual-hierarchy-differentiate-reasoning-vs.md` (gy1o4.1.3)

**Significance:** Visual hierarchy refinement is incremental work on existing foundation, not a new system.

---

## Synthesis

**Key Insights:**

1. **Archival vs Live Data** - ACTIVITY.json serves a different purpose than OpenCode API. OpenCode is for live/recent data; ACTIVITY.json is for archival (survives cleanup, portable). Writing during execution is redundant; writing on completion captures final state.

2. **Two-Tier Architecture** - The recommended pattern is:
   - **Tier 1 (Primary):** OpenCode API for sessions within retention window (24h+)
   - **Tier 2 (Archival):** ACTIVITY.json in workspace for completed agents, written on `orch complete` or workspace archival

3. **Visual Hierarchy Integration** - gy1o4.1.3 visual hierarchy design can be implemented incrementally on the existing activity-tab.svelte foundation. The structure (reasoning/tool/results grouping) already exists; styling needs refinement.

**Answer to Investigation Question:**

**Option C (Black Box Recorder)** should be implemented as **completion-time export**, not streaming writes:

1. **When:** On `orch complete` or when workspace is archived
2. **What:** Full session history from OpenCode API, exported to `ACTIVITY.json`
3. **Format:** Array of SSE-compatible events (same format as current frontend)
4. **Loading:** Dashboard loads from ACTIVITY.json if session no longer exists in OpenCode

This approach:
- Avoids duplicate writes during execution
- Captures final, complete history
- Provides archival value (portable, survives OpenCode cleanup)
- Maintains OpenCode as source of truth for live data

**Visual Hierarchy** should refine existing implementation per gy1o4.1.3:
- Reasoning: Already has muted + bullet; consider slightly lighter shade
- Tool calls: Already has blue + bold; add state icons (pending/success/error)
- Results: Already nested; ensure consistent monospace + muted background
- Steps: Add distinct styling (currently inherits default)

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode persists all session data (verified: ls ~/.local/share/opencode/storage/)
- ✅ handleSessionMessages endpoint proxies OpenCode API (verified: read serve_agents.go)
- ✅ Frontend merges historical + SSE events (verified: read activity-tab.svelte)

**What's untested:**

- ⚠️ ACTIVITY.json loading when OpenCode session deleted (not yet implemented)
- ⚠️ Export performance for sessions with 1000+ events (not benchmarked)
- ⚠️ Visual hierarchy refinements (need visual verification)

**What would change this:**

- If OpenCode API becomes unreliable, would need streaming writes to ACTIVITY.json
- If workspace archival never happens, ACTIVITY.json adds no value
- If visual hierarchy needs are more complex, may need full redesign

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach: Completion-Time Export

**Two-Tier Persistence Architecture** - Keep OpenCode API as primary, add ACTIVITY.json export on completion.

**Why this approach:**
- Avoids redundant writes during execution (OpenCode already persists)
- Captures complete, final history (no partial states)
- Workspace-centric (travels with archived workspace)
- Minimal code changes (export function + loading fallback)

**Trade-offs accepted:**
- No ACTIVITY.json until completion (acceptable: OpenCode API serves live data)
- Extra API call on completion to export (acceptable: one-time cost)

**Implementation sequence:**
1. Add `exportActivityToWorkspace(sessionId, workspacePath)` function to pkg/workspace
2. Call export in `orch complete` after successful verification
3. Add fallback loading in dashboard: if OpenCode API fails, try ACTIVITY.json
4. Apply visual hierarchy refinements to activity-tab.svelte

### Alternative Approaches Considered

**Option A: Streaming Writes (write during execution)**
- **Pros:** Real-time persistence, survives crashes
- **Cons:** Duplicates OpenCode storage, sync complexity, I/O overhead
- **When to use instead:** If OpenCode becomes unreliable or we need offline-first

**Option B: No Workspace Persistence (OpenCode only)**
- **Pros:** Simplest, no duplication
- **Cons:** Loses history when OpenCode cleans old sessions, not portable
- **When to use instead:** If archival requirements don't exist

**Rationale for recommendation:** Completion-time export provides archival value without the complexity and duplication of streaming writes. OpenCode is already authoritative for live data.

---

### Implementation Details

**What to implement first:**
1. `exportActivityToWorkspace()` function in pkg/workspace
2. Call from `orch complete` command
3. Dashboard fallback loading (check ACTIVITY.json if API fails)

**Visual Hierarchy Refinements (from gy1o4.1.3):**

```svelte
<!-- Reasoning: muted + bullet (already implemented, refine shade) -->
<div class="text-muted-foreground/60">  <!-- was /70, make more muted -->
  <span class="opacity-60">•</span>
  <span class="leading-relaxed">{part.text}</span>
</div>

<!-- Tool calls: monospace + blue + state indicator -->
<div class="font-mono">
  <span class="text-blue-400 font-semibold">{toolName}</span>
  {#if state === 'running'}
    <span class="animate-pulse">...</span>
  {:else if state === 'error'}
    <span class="text-red-400"></span>
  {:else}
    <span class="text-green-400/60"></span>
  {/if}
</div>

<!-- Results: nested + muted + monospace (already implemented) -->
<div class="ml-6 text-muted-foreground/50 font-mono bg-black/10 rounded p-2">
  <pre class="whitespace-pre-wrap">{output}</pre>
</div>
```

**Things to watch out for:**
- ⚠️ Export must handle missing sessions gracefully (agent completed but OpenCode cleaned)
- ⚠️ ACTIVITY.json can be large for long sessions - consider size limits
- ⚠️ Visual hierarchy changes need Playwright visual verification

**Areas needing further investigation:**
- ACTIVITY.json retention policy (archive vs delete with workspace)
- Compression for large activity files
- Search/filter within archived activity

**Success criteria:**
- ✅ Completed agent workspace contains ACTIVITY.json
- ✅ Dashboard loads activity from ACTIVITY.json when OpenCode session missing
- ✅ Visual hierarchy clearly distinguishes reasoning/tools/results
- ✅ No performance regression in activity feed

---

## File Targets

**Files to create:**
- `pkg/workspace/activity.go` - Export activity to ACTIVITY.json

**Files to modify:**
- `cmd/orch/complete_cmd.go` - Call export on completion
- `cmd/orch/serve_agents.go` - Add fallback to ACTIVITY.json
- `web/src/lib/components/agent-detail/activity-tab.svelte` - Visual hierarchy refinements

**ACTIVITY.json Format:**
```json
{
  "version": 1,
  "session_id": "ses_xxx",
  "exported_at": "2026-01-17T15:45:00Z",
  "events": [
    {
      "id": "part_xxx",
      "type": "message.part",
      "timestamp": 1737146700000,
      "properties": {
        "part": {
          "type": "tool",
          "tool": "Bash",
          "state": {
            "status": "completed",
            "input": {"command": "ls -la"},
            "output": "..."
          }
        }
      }
    }
  ]
}
```

---

## References

**Files Examined:**
- `web/src/lib/stores/agents.ts` - Session history store implementation
- `web/src/lib/components/agent-detail/activity-tab.svelte` - Activity feed UI
- `cmd/orch/serve_agents.go` - Backend session messages proxy
- `.kb/investigations/2026-01-07-design-dashboard-activity-feed-persistence.md` - Prior investigation
- `.kb/investigations/2026-01-17-inv-visual-hierarchy-differentiate-reasoning-vs.md` - Visual hierarchy investigation

**Commands Run:**
```bash
# Check OpenCode storage
ls -la ~/.local/share/opencode/storage/

# Check workspace structure
ls -la .orch/workspace/og-feat-implement-tiered-stuck-17jan-41c0/
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-07-design-dashboard-activity-feed-persistence.md` - Original hybrid architecture design
- **Investigation:** `.kb/investigations/2026-01-17-inv-visual-hierarchy-differentiate-reasoning-vs.md` - Visual hierarchy (gy1o4.1.3)
- **Guide:** `.kb/guides/dashboard.md` - Dashboard architecture reference

---

## Investigation History

**2026-01-17 15:46:** Investigation started
- Initial question: How to implement Black Box Recorder (Option C) for activity feed persistence
- Context: Part of Activity Feed Redesign epic (gy1o4.1)

**2026-01-17 15:50:** Key discovery
- Found OpenCode already persists all session data
- Realized ACTIVITY.json is for archival, not live data

**2026-01-17 16:00:** Synthesis complete
- Recommended completion-time export vs streaming writes
- Integrated visual hierarchy findings from gy1o4.1.3

**2026-01-17 16:05:** Investigation completed
- Status: Complete
- Key outcome: Two-tier architecture (OpenCode primary, ACTIVITY.json archival) with completion-time export
