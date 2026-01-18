<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard Replay UI should use debugger-style stepping with playback controls, not timeline scrubber; activity-aware resumption works via ACTIVITY.json bundling in spawn context.

**Evidence:** Existing activity-tab already has filtering by type, event grouping, and expand/collapse; adding prev/next stepping and auto-play requires minimal UI changes. ACTIVITY.json persistence is already implemented at completion time.

**Knowledge:** Three design forks resolved: (1) Debugger stepping > timeline scrubber (precision over continuous), (2) Two-tier filtering: meta-filters (Reasoning/Execution) + granular type filters, (3) Activity bundling via API endpoint > spawn context embedding.

**Next:** Implement feature-impl for Replay UI starting with stepping controls, then filtering enhancement, then activity-aware API endpoint.

**Promote to Decision:** recommend-yes - Establishes the pattern for how agents consume predecessor activity (API-based, not context-embedded).

---

# Investigation: Design Dashboard Replay UI and Activity-Aware Resumption

**Question:** How should the dashboard enable replaying agent activity sequences, and how should new agents access predecessor activity?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** architect (og-arch-design-dashboard-replay-17jan-6087)
**Phase:** Complete
**Next Step:** None - Ready for implementation
**Status:** Complete

<!-- Lineage -->
**Related-Epic:** orch-go-gy1o4.1 (Activity Feed Redesign)
**Related-Issue:** orch-go-gy1o4.1.9 (Dashboard Replay UI)
**Builds-On:** `.kb/investigations/2026-01-07-design-dashboard-activity-feed-persistence.md`

---

## Problem Framing

### Design Questions

1. **Replay UI:** How should users navigate through historical activity for completed agents?
2. **Filtering:** How should reasoning vs execution be distinguished in the activity feed?
3. **Activity-Aware Resumption:** How should new agents (resuming or following up) access predecessor activity?

### Success Criteria

- Replay UI enables precise navigation through historical events
- Filtering clearly separates "thinking" from "doing" at a glance
- New agents can be spawned with context about what a predecessor did
- Works within 666px minimum width constraint
- No breaking changes to existing live activity feed

### Constraints

- **Dashboard width:** Must work at 666px (half MacBook Pro screen)
- **Activity ephemeral:** Live activity is ephemeral; replay is for completed agents only
- **Event panel height:** max-h-64 for visibility (existing constraint)
- **ACTIVITY.json:** Already created at completion time by `pkg/activity/export.go`
- **Hybrid architecture:** SSE for live, API for historical (already implemented)

### Scope

**In scope:**
- Replay UI for completed agents
- Enhanced filtering (reasoning vs execution)
- API/mechanism for predecessor activity access

**Out of scope:**
- Browser-side persistence (IndexedDB)
- Export/download of activity as file
- Video-style timeline with scrubber

---

## Findings

### Finding 1: Activity-Tab Already Has Event Grouping and Filtering

**Evidence:** The current `activity-tab.svelte` has:
- Type filtering: text, tool, reasoning, step (`enabledTypes` state at line 23)
- Tool grouping with expand/collapse (`groupToolEvents` at line 258-311)
- Event limit per agent (500 events, line 15)
- Auto-scroll toggle (lines 18, 317-325)
- Keyboard navigation (Ctrl+O to expand, line 364)

**Source:**
- `web/src/lib/components/agent-detail/activity-tab.svelte:23` (type filters)
- `web/src/lib/components/agent-detail/activity-tab.svelte:258-311` (event grouping)

**Significance:** The foundation for replay exists. Adding stepping (prev/next) and playback controls requires incremental changes, not a rewrite.

---

### Finding 2: ACTIVITY.json Is SSE-Compatible Format

**Evidence:** The `pkg/activity/export.go` transforms OpenCode messages into SSE-compatible `MessagePartResponse` format:
```go
type MessagePartResponse struct {
    ID         string                `json:"id"`
    Type       string                `json:"type"` // Always "message.part"
    Properties MessagePartProperties `json:"properties"`
    Timestamp  int64                 `json:"timestamp,omitempty"`
}
```

This exactly matches the `SSEEvent` interface in `agents.ts`, enabling seamless loading.

**Source:**
- `pkg/activity/export.go:25-30` (MessagePartResponse struct)
- `web/src/lib/stores/agents.ts:116-140` (SSEEvent interface)

**Significance:** ACTIVITY.json can be loaded directly into the activity-tab without transformation. The fallback mechanism in `handleSessionMessages` already does this (lines 1486-1498 in serve_agents.go).

---

### Finding 3: Session Messages Endpoint Already Falls Back to ACTIVITY.json

**Evidence:** The `/api/session/:sessionID/messages` endpoint tries OpenCode API first, then falls back to ACTIVITY.json:
```go
// OpenCode API failed - fall back to ACTIVITY.json
workspacePath := findWorkspaceBySessionID(projectDir, sessionID)
if workspacePath != "" {
    if events := loadActivityFromWorkspace(workspacePath); events != nil {
        // Successfully loaded from ACTIVITY.json
    }
}
```

**Source:** `cmd/orch/serve_agents.go:1485-1499`

**Significance:** Completed agents already serve historical activity via ACTIVITY.json. The data layer is complete; only UI changes needed.

---

### Finding 4: Live vs Replay Are Different Cognitive Modes

**Evidence:** Existing constraint states: "Activity state should be ephemeral in UI". This aligns with the two-mode design pattern the dashboard already uses (Operational vs Historical modes).

- **Live mode:** Following along as events stream in, auto-scroll enabled, waiting for processing
- **Replay mode:** Navigating through completed history, stepping through at own pace, analyzing what happened

**Source:**
- `kb context` constraint: "Activity state should be ephemeral in UI"
- `.kb/models/dashboard-architecture.md:73-86` (mode lifecycle)

**Significance:** Replay UI should be a distinct mode activated for completed agents, not a modification of live streaming behavior.

---

### Finding 5: Spawn Context Already Supports Rich KB Context Injection

**Evidence:** The spawn context template includes `{{.KBContext}}` placeholder that gets populated via `kb context` queries:
```go
data := contextData{
    // ...
    KBContext:         cfg.KBContext,
    // ...
}
```

This demonstrates the pattern for injecting contextual information into new agent sessions.

**Source:**
- `pkg/spawn/context.go:452-454` (KBContext field)
- SPAWN_CONTEXT.md template line 83-85

**Significance:** Activity-aware resumption could follow the same pattern - summary of predecessor activity injected into spawn context.

---

## Design Forks

### Fork 1: Replay UI Navigation Style

**Options:**
- **A: Debugger-style stepping** - Prev/Next buttons, current position indicator, play/pause for auto-advance
- **B: Timeline scrubber** - Drag handle along timeline, seek to any point
- **C: Log viewer style** - Scroll-based, search/filter only, no stepping

**Substrate says:**
- Constraint: "Dashboard must be fully usable at 666px width" - Timeline scrubber needs horizontal space
- Model: Dashboard uses progressive disclosure - debugger stepping fits this pattern
- Principle: Dense but scannable - stepping preserves event context

**RECOMMENDATION:** **Option A: Debugger-style stepping**

**Why:**
- Fits 666px width constraint (vertical controls)
- Preserves event grouping and expand/collapse
- More precise than scrubbing for analyzing what an agent did
- Natural keyboard navigation (arrow keys)

**Trade-off accepted:** Can't jump to arbitrary position (must step through) - acceptable because most replay sessions analyze sequential flow.

**When this would change:** If agents routinely produce 10,000+ events and users need to seek to specific times.

---

### Fork 2: Reasoning vs Execution Filtering

**Options:**
- **A: Two-tier filtering** - Meta-filters (Reasoning/Execution) that toggle groups of types
- **B: Replace current filters** - Replace text/tool/reasoning/step with thinking/acting
- **C: Keep current only** - Current granular filters are sufficient

**Substrate says:**
- Prior decision: "Visual hierarchy: differentiate reasoning vs tools vs results" (orch-go-gy1o4.1.3 - still open)
- Current implementation: Four independent toggles (text, tool, reasoning, step)
- Model: "Reasoning text differentiated (bullet points, muted)" per Activity Feed Redesign target state

**RECOMMENDATION:** **Option A: Two-tier filtering**

```
┌─────────────────────────────────────────┐
│ [Reasoning ▼] [Execution ▼] │ Filter ▾  │
├─────────────────────────────────────────┤
│ 💬 text  🤔 reasoning                   │  ← Reasoning (thinking)
│ 🔧 tool  ▶️ step                        │  ← Execution (doing)
└─────────────────────────────────────────┘
```

**Why:**
- Preserves granular control for power users
- Adds quick "show me thinking" / "show me doing" modes
- Maps to natural mental model: what was the agent thinking vs what did it do

**Trade-off accepted:** Slight UI complexity increase - acceptable for the analytical use case.

**Mapping:**
- **Reasoning (Thinking):** text, reasoning
- **Execution (Doing):** tool, step

---

### Fork 3: Activity-Aware Resumption Mechanism

**Options:**
- **A: API endpoint for predecessor activity** - New endpoint `GET /api/agent/:id/activity-summary` returns structured summary
- **B: Embed in spawn context** - Bundle ACTIVITY.json or summary directly into SPAWN_CONTEXT.md
- **C: SYNTHESIS.md only** - Rely on existing SYNTHESIS.md as the handoff artifact

**Substrate says:**
- Principle: "Coherence over patches" - should use existing patterns
- Prior implementation: SYNTHESIS.md already provides D.E.K.N. summary
- Constraint: Context windows have limits - embedding full ACTIVITY.json would be huge
- Model: Spawn context supports KBContext injection

**RECOMMENDATION:** **Option A: API endpoint for predecessor activity**

New endpoint: `GET /api/agent/:workspaceName/activity-summary`

Returns:
```json
{
  "predecessor_id": "og-worker-feature-impl-17jan-abc1",
  "beads_id": "orch-go-xyz123",
  "summary": {
    "total_events": 342,
    "duration_minutes": 45,
    "tools_used": ["Bash", "Read", "Edit", "Grep"],
    "files_touched": ["cmd/orch/serve.go", "web/src/..."],
    "key_findings": ["..extracted from SYNTHESIS.md.."],
    "outcome": "partial"
  },
  "activity_url": "/api/session/{sessionID}/messages"
}
```

**Why:**
- Keeps spawn context lean (just reference, not full activity)
- Successor agent can fetch full activity if needed
- Summary provides quick context without token cost
- Works with dashboard UI (can show "View predecessor activity" link)

**Trade-off accepted:** Requires API call instead of immediate context - acceptable because most successor agents only need summary, not full history.

**Spawn context addition:**
```markdown
## PREDECESSOR CONTEXT

This task continues work from a previous agent: **{{.PredecessorID}}**

**Summary:** {{.PredecessorSummary}}

**Activity endpoint:** `GET /api/agent/{{.PredecessorID}}/activity-summary`

Review predecessor's work if you need to understand what was tried or why decisions were made.
```

---

## Synthesis

### Key Insights

1. **Incremental enhancement, not rewrite** - The activity-tab already has event grouping, filtering, and expand/collapse. Replay mode adds stepping controls and a mode switch, building on existing patterns.

2. **Two-tier filtering serves both glance and analysis** - Meta-filters (Reasoning/Execution) for quick context, granular filters (text/tool/reasoning/step) for precise analysis. This follows progressive disclosure.

3. **Activity-aware resumption is a reference pattern, not embedding** - Instead of bloating spawn context with full activity, provide an API endpoint and inject a summary. Successor agents can drill down if needed.

### Answer to Investigation Questions

**Q1: How should users navigate through historical activity?**
A: Debugger-style stepping with prev/next controls, auto-play option, and position indicator. This works within 666px width, preserves event context, and enables precise analysis.

**Q2: How should reasoning vs execution be distinguished?**
A: Two-tier filtering - meta-filters for quick "show me thinking" / "show me doing" toggles, with existing granular filters preserved for power users.

**Q3: How should new agents access predecessor activity?**
A: API endpoint returning structured summary plus activity URL. Spawn context gets a "Predecessor Context" section with summary and API reference, not full activity embedding.

---

## Structured Uncertainty

**What's tested:**
- ✅ ACTIVITY.json format is SSE-compatible (verified: read export.go struct definitions)
- ✅ Session messages endpoint falls back to ACTIVITY.json (verified: read serve_agents.go:1485-1499)
- ✅ Activity-tab has event grouping and type filtering (verified: read activity-tab.svelte)

**What's untested:**
- ⚠️ Performance of stepping through 500+ events (not benchmarked)
- ⚠️ Auto-play timing (how fast is comfortable? 500ms? 1s?)
- ⚠️ Activity summary generation from ACTIVITY.json (not implemented)

**What would change this:**
- If stepping performance is poor, might need virtualization
- If users want to jump to specific tool calls, might need search/goto feature
- If SYNTHESIS.md is insufficient for successor context, might need richer summary

---

## Implementation Recommendations

### Recommended Approach ⭐

**Phased implementation** - Build replay UI first, then filtering enhancement, then activity-aware resumption.

**Why this approach:**
- Each phase is independently useful
- Replay UI validates the UX before adding complexity
- Activity-aware resumption depends on having good activity data

**Trade-offs accepted:**
- Three separate implementation efforts instead of one big change
- Acceptable because each phase can ship independently

### Implementation Sequence

**Phase 1: Replay UI (Stepping Controls)**
1. Add `replayMode` state to activity-tab (toggle for completed agents)
2. Add `currentEventIndex` state for position tracking
3. Add Prev/Next buttons and position indicator
4. Add Play/Pause for auto-advance with speed control
5. Disable auto-scroll in replay mode

**Phase 2: Two-Tier Filtering**
1. Add meta-filter buttons: "Reasoning" / "Execution"
2. Meta-filters toggle groups of existing type filters
3. Preserve existing granular filter buttons
4. Add visual grouping (separator or button group)

**Phase 3: Activity-Aware Resumption**
1. Add `/api/agent/:workspaceName/activity-summary` endpoint in serve_agents.go
2. Parse SYNTHESIS.md for key_findings and outcome
3. Aggregate ACTIVITY.json for tool usage and files touched
4. Add "Predecessor Context" section to spawn context template
5. Add `--predecessor` flag to orch spawn

### Alternative Approaches Considered

**Timeline Scrubber:**
- **Pros:** Visual, familiar from video players
- **Cons:** Needs horizontal space (666px constraint), loses event grouping context
- **When to use:** If replay sessions routinely involve 1000+ events and seek is essential

**Full Activity in Spawn Context:**
- **Pros:** Immediate access, no API call needed
- **Cons:** Token cost, bloats context, most of it unused
- **When to use:** Never for full activity; maybe for 10-event summary

---

## File Targets

**Files to modify:**

| File | Change |
|------|--------|
| `web/src/lib/components/agent-detail/activity-tab.svelte` | Add replay mode, stepping controls, two-tier filters |
| `cmd/orch/serve_agents.go` | Add `/api/agent/:workspaceName/activity-summary` endpoint |
| `pkg/spawn/context.go` | Add PredecessorContext template section |
| `pkg/spawn/config.go` | Add PredecessorID, PredecessorSummary fields |

**Files to create:**
- None required (all modifications to existing files)

**Acceptance Criteria:**

Phase 1:
- [ ] Completed agents show "Enter Replay Mode" button
- [ ] Prev/Next buttons navigate through events
- [ ] Current position indicator shows "Event 42 of 342"
- [ ] Play button auto-advances at configurable speed
- [ ] Auto-scroll disabled in replay mode

Phase 2:
- [ ] "Reasoning" meta-filter toggles text + reasoning types
- [ ] "Execution" meta-filter toggles tool + step types
- [ ] Existing granular filters still work
- [ ] Filter state persists in localStorage

Phase 3:
- [ ] `/api/agent/:workspaceName/activity-summary` returns structured summary
- [ ] `orch spawn --predecessor <workspace>` injects context
- [ ] Dashboard shows "View predecessor" link for continuation issues

**Out of Scope:**
- Timeline scrubber UI
- Export activity to file
- Browser-side persistence (IndexedDB)
- Full activity embedding in spawn context

---

## References

**Files Examined:**
- `web/src/lib/components/agent-detail/activity-tab.svelte` - Current activity UI implementation
- `web/src/lib/stores/agents.ts` - SSEEvent interface and history fetching
- `pkg/activity/export.go` - ACTIVITY.json creation
- `cmd/orch/serve_agents.go` - Session messages endpoint with ACTIVITY.json fallback
- `pkg/spawn/context.go` - Spawn context generation
- `.kb/investigations/2026-01-07-design-dashboard-activity-feed-persistence.md` - Prior work on activity persistence

**Related Artifacts:**
- **Decision:** Hybrid SSE + API architecture for activity feed
- **Constraint:** "Activity state should be ephemeral in UI"
- **Constraint:** "Dashboard must be fully usable at 666px width"
- **Open Issue:** orch-go-gy1o4.1.3 (Visual hierarchy: differentiate reasoning vs tools)

---

## Investigation History

**2026-01-17 21:22:** Investigation started
- Initial question: How to implement Dashboard Replay UI and Activity-Aware Resumption
- Context: Part of Activity Feed Redesign epic (orch-go-gy1o4.1)

**2026-01-17 21:45:** Problem framing complete
- Identified three design questions: Replay UI, Filtering, Resumption
- Reviewed existing implementations and constraints

**2026-01-17 22:15:** Fork navigation complete
- Resolved all three forks with recommendations
- Debugger stepping > timeline, two-tier filtering, API endpoint for resumption

**2026-01-17 22:30:** Investigation complete
- Status: Complete
- Key outcome: Three-phase implementation plan with clear file targets and acceptance criteria
