---
linked_issues:
  - orch-go-w0bm
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard integrations should be tiered by urgency - Beads + Focus are highest value (operational awareness), Servers moderate (dev context), KB/KN lowest (reference only).

**Evidence:** Analyzed all 5 proposed integrations against dashboard purpose (operational awareness for orchestrator). Beads provides actionable work queue (175 ready issues), Focus shows drift status. KB/KN are reference tools already CLI-accessible.

**Knowledge:** Dashboard purpose is real-time operational awareness, not knowledge discovery. Integrations should surface "what needs attention NOW" not "what exists."

**Next:** Implement in phases: Phase 1 - Beads summary + Focus indicator in stats bar. Phase 2 - Servers status. Phase 3 - KB/KN only if clear demand emerges.

**Confidence:** High (85%) - Design validated against actual data sources and dashboard architecture.

---

# Investigation: Design Dashboard Integrations Beyond Agents

**Question:** Which dashboard integrations provide the most value for orchestrator operational awareness, and how should they be displayed?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** og-inv-design-dashboard-integrations-24dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Current Dashboard Architecture

**Evidence:** The dashboard currently has:
1. **Stats bar** - Compact horizontal bar showing: Active agents (🟢), Recent (🕐), Archive (📦), Errors (❌), Usage (5h %, weekly %)
2. **Swarm Map** - Primary focus area with collapsible sections (Active/Recent/Archive)
3. **Event Panels** - Two side-by-side panels: Agent Lifecycle events and SSE Stream

**Source:** `web/src/routes/+page.svelte` (652 lines), `cmd/orch/serve.go` (666 lines)

**Significance:** The stats bar is the ideal location for high-urgency, at-a-glance information. Event panels are for detailed logs. Swarm Map is the primary focus. New integrations must fit within this information hierarchy.

---

### Finding 2: Beads Data Availability and Value

**Evidence:** Beads provides rich operational data via CLI with JSON output:
- `bd stats --json`: Summary stats (192 open, 9 in-progress, 175 ready, 17 blocked)
- `bd ready`: List of actionable work items (10 shown)
- `bd blocked`: Dependencies preventing progress

Current stats sample:
```json
{
  "summary": {
    "total_issues": 1146,
    "open_issues": 192,
    "ready_issues": 175,
    "blocked_issues": 17,
    "in_progress_issues": 9
  }
}
```

**Source:** Commands run: `bd stats --json`, `bd ready`, `bd blocked`

**Significance:** Beads is HIGH VALUE for dashboard. The "ready issues" count (175) directly answers "what work is available?" The blocked count (17) shows bottlenecks. This is actionable operational data. Integration cost: Shell out to `bd` CLI, parse JSON. Similar pattern already used for usage data.

---

### Finding 3: Focus/Drift Data Availability

**Evidence:** Focus system provides:
- Current goal: "System stability and hardening"
- Drift status: "✓ On track" or "⚠️ Drifting"
- Active issues aligned with focus

Code in `pkg/focus/focus.go` shows:
- `Store.Get()` returns current focus
- `Store.CheckDrift(activeIssues)` returns `DriftResult{IsDrifting, Goal, ActiveIssues}`
- Data stored in `~/.orch/focus.json`

**Source:** `pkg/focus/focus.go:22-62`, `orch focus` and `orch drift` CLI output

**Significance:** Focus is HIGH VALUE but lightweight. A single indicator "On track 🎯" or "Drifting ⚠️" in the stats bar provides immediate context. Implementation cost: Read JSON file directly (already Go code exists).

---

### Finding 4: KB/KN Data Accessibility

**Evidence:** 
- KB: `kb list investigations` returns 248 investigations, requires subcommands
- KN: `kn decisions --json` returns JSON array of decisions, `kn recent` shows last entries

Both are reference/discovery tools, not operational awareness tools. The orchestrator already has CLI access for these via `kb context` which provides unified search.

**Source:** Commands run: `kn decisions --json`, `kn recent`, `kb list investigations`

**Significance:** KB/KN are LOW VALUE for real-time dashboard. They're "what have we learned" not "what needs attention now." The dashboard purpose is operational awareness, not knowledge browsing. Adding these creates clutter without actionable value.

---

### Finding 5: Servers Status Data

**Evidence:** `orch servers list` shows:
- 24 projects with port allocations
- 3 currently running (beads-ui-svelte, orch-go, price-watch)
- Running status checked via `tmux.ListWorkersSessions()`

Code in `cmd/orch/servers.go:137-144` shows `ProjectServerInfo` struct:
```go
type ProjectServerInfo struct {
    Project string
    Ports   []port.Allocation
    Running bool
    Session string
}
```

**Source:** `cmd/orch/servers.go:146-229`, `orch servers list` output

**Significance:** Servers is MEDIUM VALUE. Knowing which dev servers are running provides context for agent work. But it's secondary to knowing what work exists (Beads) and whether work aligns with goals (Focus). Could be added as expandable section below stats bar.

---

## Synthesis

**Key Insights:**

1. **Dashboard purpose is operational awareness, not knowledge discovery** - The existing design (stats bar + swarm map + events) prioritizes real-time status. New integrations should follow this pattern: actionable data at a glance, not reference material that requires exploration.

2. **Beads + Focus are highest value because they answer "what should I do next"** - Ready issues count tells the orchestrator how much work is queued. Focus/drift status tells them if current work aligns with goals. These are the key operational questions.

3. **KB/KN are reference tools better accessed via CLI** - `kb context` already provides unified knowledge search. Putting this in the dashboard duplicates CLI functionality without adding real-time value. The orchestrator already knows to use `kb context` when starting work.

**Answer to Investigation Question:**

**Priority 1 (High Value):** Beads summary + Focus indicator in stats bar
- Beads: Ready count, blocked count, in-progress count
- Focus: Single "On track 🎯" or "Drifting ⚠️" indicator with tooltip showing goal

**Priority 2 (Medium Value):** Servers status as expandable section
- "3 running / 24 projects" in collapsed form
- Expandable to show running server list

**Priority 3 (Low Value - Skip):** KB/KN integrations
- Not recommended for dashboard
- CLI tools (`kb context`, `kn recent`) serve this purpose better

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Analysis is based on actual data exploration of all 5 proposed integrations, understanding of existing dashboard architecture, and alignment with the stated dashboard purpose (operational awareness for orchestrator).

**What's certain:**

- ✅ Beads provides actionable work queue data with JSON output (tested `bd stats --json`)
- ✅ Focus provides drift detection with low implementation cost (existing Go code)
- ✅ Current dashboard stats bar pattern is well-established and extensible

**What's uncertain:**

- ⚠️ User feedback - actual orchestrator may have different priorities
- ⚠️ Polling frequency - how often should beads stats refresh?
- ⚠️ Cross-project scope - beads data is per-project, dashboard serves all projects

**What would increase confidence to Very High (95%+):**

- User validation of proposed information hierarchy
- Prototype testing with actual orchestrator workflow
- Performance testing of beads CLI shell-out latency

---

## Implementation Recommendations

### Recommended Approach ⭐

**Phased Integration: Beads → Focus → Servers**

**Why this approach:**
- Starts with highest-value integration (Beads work queue)
- Low risk - each phase is independent
- Stats bar pattern already proven (usage display works well)

**Trade-offs accepted:**
- No KB/KN integration - defer until clear demand
- Shell-out to `bd` CLI instead of native integration - matches existing `usage` pattern

**Implementation sequence:**
1. **Phase 1a: Beads Stats** - Add ready/blocked/in-progress counts to stats bar
2. **Phase 1b: Focus Indicator** - Add drift status to stats bar
3. **Phase 2: Servers Panel** - Add collapsible servers section below stats bar
4. **Phase 3 (if needed): KB/KN** - Only if user feedback indicates demand

### Alternative Approaches Considered

**Option B: Sidebar with All Integrations**
- **Pros:** Dedicated space, doesn't crowd stats bar
- **Cons:** Changes dashboard layout significantly, adds permanent screen real estate
- **When to use instead:** If 5+ integrations all prove valuable

**Option C: Tabbed Interface**
- **Pros:** Clean separation of concerns
- **Cons:** Hides information, requires clicks to see data
- **When to use instead:** If dashboard becomes primarily non-agent focused

**Rationale for recommendation:** Stats bar extension is minimal UI change, highest impact (Beads shows work queue), and follows proven pattern.

---

### Implementation Details

**What to implement first:**

Phase 1a - Beads stats endpoint and display:

```go
// New API endpoint: GET /api/beads
type BeadsAPIResponse struct {
    Ready      int `json:"ready"`
    InProgress int `json:"in_progress"`
    Blocked    int `json:"blocked"`
    Error      string `json:"error,omitempty"`
}

func handleBeads(w http.ResponseWriter, r *http.Request) {
    // Shell out to: bd stats --json
    // Parse JSON, extract counts
    // Return BeadsAPIResponse
}
```

Stats bar addition:
```svelte
<!-- After usage display -->
{#if $beads && !$beads.error}
    <div class="h-4 w-px bg-border"></div>
    <div class="flex items-center gap-2" title="Beads Work Queue">
        <span class="text-sm">📋</span>
        <span class="text-sm font-medium">{$beads.ready}</span>
        <span class="text-xs text-muted-foreground">ready</span>
        {#if $beads.blocked > 0}
            <span class="text-sm text-yellow-600">{$beads.blocked}⛔</span>
        {/if}
    </div>
{/if}
```

Phase 1b - Focus endpoint and display:

```go
// New API endpoint: GET /api/focus
type FocusAPIResponse struct {
    Goal      string `json:"goal,omitempty"`
    IsDrifting bool  `json:"is_drifting"`
    Error     string `json:"error,omitempty"`
}

func handleFocus(w http.ResponseWriter, r *http.Request) {
    store, _ := focus.New("")
    f := store.Get()
    if f == nil {
        return FocusAPIResponse{} // No focus set
    }
    // Get active issues from current agents
    // Check drift
    drift := store.CheckDrift(activeIssues)
    return FocusAPIResponse{Goal: f.Goal, IsDrifting: drift.IsDrifting}
}
```

Stats bar addition:
```svelte
{#if $focus?.goal}
    <div class="h-4 w-px bg-border"></div>
    <div class="flex items-center gap-1" title={$focus.goal}>
        {#if $focus.is_drifting}
            <span class="text-sm">⚠️</span>
            <span class="text-xs text-yellow-600">Drifting</span>
        {:else}
            <span class="text-sm">🎯</span>
            <span class="text-xs text-green-600">On track</span>
        {/if}
    </div>
{/if}
```

**Things to watch out for:**

- ⚠️ `bd stats --json` may be slow - consider caching with 30s TTL
- ⚠️ Cross-project beads: current cwd affects `bd` output, may need to aggregate
- ⚠️ Focus drift requires knowing "active issues" - use agent beads IDs from `/api/agents`

**Areas needing further investigation:**

- Whether beads data should be real-time SSE or polling (recommend polling with 60s interval like usage)
- How to handle multi-project beads aggregation if needed
- Mobile/responsive display of expanded stats bar

**Success criteria:**

- ✅ Orchestrator can see ready work count at a glance without leaving dashboard
- ✅ Orchestrator knows immediately if current work is drifting from focus
- ✅ No significant performance impact (page load <500ms, polling <2s)

---

## References

**Files Examined:**
- `web/src/routes/+page.svelte:1-652` - Current dashboard implementation
- `cmd/orch/serve.go:1-666` - API server with existing endpoints
- `pkg/focus/focus.go:1-266` - Focus store implementation
- `cmd/orch/servers.go:137-228` - Servers list implementation

**Commands Run:**
```bash
# Beads data exploration
bd stats --json
bd ready
bd blocked

# Knowledge tools exploration
kn decisions --json
kn recent
kb list investigations

# Focus/servers exploration
orch focus
orch drift
orch servers list
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-24-inv-implement-progressive-disclosure-swarm-dashboard.md` - Recent dashboard work
- **Decision:** `kn-012f68` - Slide-out panel for agent card detail view

---

## Investigation History

**2025-12-24 ~10:00:** Investigation started
- Initial question: Which integrations provide most value for orchestrator operational awareness?
- Context: Dashboard currently shows agents, events, usage - want to expand

**2025-12-24 ~10:30:** Data source analysis complete
- Explored all 5 proposed integrations (Beads, KB, KN, Focus, Servers)
- Found JSON output available for Beads and KN
- Confirmed Focus has existing Go package

**2025-12-24 ~11:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Tiered approach - Beads+Focus high priority, Servers medium, KB/KN skip

---

## Self-Review

- [x] Real test performed (not code review) - Ran actual CLI commands to test data availability
- [x] Conclusion from evidence (not speculation) - Recommendations based on observed data formats
- [x] Question answered - Clear priority ordering for integrations
- [x] File complete - All sections filled

**Self-Review Status:** PASSED
