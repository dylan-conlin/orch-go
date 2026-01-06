<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Designed tabbed agent detail pane interface with three distinct views: Activity (filtered SSE feed for active agents), Investigation (workspace artifact viewer), and Synthesis (structured D.E.K.N. output display).

**Evidence:** Analyzed existing agent-detail-panel.svelte (523 lines), agents store (649 lines), agent-card (445 lines); constraint found that dashboard must work at 666px width minimum.

**Knowledge:** The panel architecture needs state-based tab visibility (active agents show Activity, completed show Synthesis+Investigation), and all tabs must function at 80-85% viewport width.

**Next:** Create Epic with implementation tasks for: AgentDetailTabs component, ActivityTab, InvestigationTab, SynthesisTab, and update panel layout.

---

# Investigation: Dashboard Agent Detail Pane Redesign

**Question:** How should we design a tabbed interface for the agent detail pane that shows filtered message feed for active agents and Investigation/Synthesis tabs for completed agents?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** design-session agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current Panel Structure (523 lines, monolithic)

**Evidence:** The existing `agent-detail-panel.svelte` contains all content in a single scrollable view:
- Lines 220-448: Single scrollable content area with sequential sections
- Status bar, Live Activity (active only), Quick Copy, Context, Synthesis (completed only)
- Width: `sm:w-[66vw] lg:w-[60vw] xl:w-[55vw]` (line 201)

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/components/agent-detail/agent-detail-panel.svelte:201,220-448`

**Significance:** The current design mixes content types in one scroll. Tabbed interface would separate concerns and allow focused views per agent state.

---

### Finding 2: SSE Event Filtering Already Implemented

**Evidence:** Lines 169-176 show SSE event filtering for the selected agent:
```svelte
$: agentEvents = $selectedAgent?.session_id 
  ? $sseEvents.filter(e => {
    if (e.type !== 'message.part' && e.type !== 'message.part.updated') return false;
    const eventSessionId = e.properties?.part?.sessionID || e.properties?.sessionID;
    return eventSessionId === $selectedAgent?.session_id;
  }).slice(-50)
  : [];
```

**Source:** `agent-detail-panel.svelte:169-176`

**Significance:** The filtering logic for SSE messages per-agent exists. Activity tab can reuse this pattern but expand to more event types for richer activity feed.

---

### Finding 3: Width Constraint - 666px Minimum

**Evidence:** From SPAWN_CONTEXT.md prior knowledge:
> Dashboard must be fully usable at 666px width (half MacBook Pro screen). No horizontal scrolling. All critical info visible without scrolling.

Current panel uses percentage-based widths. Task requirement specifies 80-85% viewport width.

**Source:** Prior constraint in SPAWN_CONTEXT.md

**Significance:** Tab design must work at narrow widths. Tab labels should be concise or use icons at smaller breakpoints.

---

### Finding 4: Agent State Determines Available Views

**Evidence:** From agent store analysis:
- `DisplayState` enum: 'running' | 'ready-for-review' | 'idle' | 'waiting' | 'completed' | 'abandoned'
- Active agents have `is_processing`, `current_activity` fields
- Completed agents have `synthesis`, `close_reason` fields
- Investigation/workspace artifacts accessible via `primary_artifact` field

**Source:** `agents.ts:61-102`, `agent.synthesis` structure at lines 8-14

**Significance:** Tab visibility should be state-driven:
- Active agents: Activity tab (primary), Context tab
- Completed agents: Synthesis tab (primary), Investigation tab, Context tab

---

### Finding 5: Existing Synthesis Data Structure

**Evidence:** The `Synthesis` interface is well-defined:
```typescript
export interface Synthesis {
  tldr?: string;
  outcome?: string; // success, partial, blocked, failed
  recommendation?: string; // close, continue, escalate
  delta_summary?: string; // e.g., "3 files created, 2 modified, 5 commits"
  next_actions?: string[]; // Follow-up items
}
```

**Source:** `agents.ts:8-14`

**Significance:** Synthesis tab can display structured D.E.K.N. format with clear visual hierarchy.

---

## Design Proposal

### Tab Structure

```
+----------------------------+---------------------------+-----------------+
| [Activity] (active only)   | [Investigation]           | [Synthesis]     |
+----------------------------+---------------------------+-----------------+
|                                                                          |
|  Tab content area (fills remaining height, scrollable)                   |
|                                                                          |
+--------------------------------------------------------------------------+
```

### Tab Visibility by Agent State

| Agent State | Activity Tab | Investigation Tab | Synthesis Tab |
|-------------|--------------|-------------------|---------------|
| active      | ✅ Primary   | ❌ Hidden        | ❌ Hidden     |
| ready-for-review | ✅ Visible | ✅ Visible | ✅ Primary |
| completed   | ❌ Hidden    | ✅ Visible        | ✅ Primary    |
| abandoned   | ❌ Hidden    | ✅ Visible        | ❌ Hidden     |

### Activity Tab (For Active Agents)

**Content:**
- **Live Activity Stream**: Expanded, filterable SSE event feed
- **Message Type Filter**: Toggle visibility of text, tool, reasoning events
- **Auto-scroll**: Lock to bottom option for real-time following
- **Timestamps**: Show relative or absolute time

**Key UX decisions:**
- Replaces current "Live Activity" section (lines 245-288)
- Shows more events (increase from 50 to 100)
- Add event type badges for visual scanning
- Consider tool invocation details expansion

### Investigation Tab (For Completed Agents)

**Content:**
- **Artifact Viewer**: Display contents of `.kb/investigations/` file linked to agent
- **Workspace Links**: Quick access to workspace directory files
- **File Browser**: Simple list of workspace contents

**Data Source:**
- `agent.primary_artifact` path
- Workspace at `.orch/workspace/{agent.id}/`

**API Needed:**
- New endpoint: `GET /api/workspace/{id}/files` - List workspace files
- New endpoint: `GET /api/workspace/{id}/file?path=...` - Read file content
- Or reuse existing `/api/agents/{id}` with expanded response

### Synthesis Tab (For Completed Agents)

**Content:**
- **D.E.K.N. Summary**: Structured display of Delta, Evidence, Knowledge, Next
- **Outcome Badge**: Visual indicator (success/partial/blocked/failed)
- **Recommendation**: Close, continue, escalate with appropriate styling
- **Next Actions**: List with "Create Issue" buttons (existing functionality)
- **Delta Summary**: Git changes summary

**Key UX decisions:**
- Reuse existing synthesis display (lines 377-447)
- Make it more prominent as primary view for completed agents
- Add D.E.K.N. section headers for better structure

### Panel Layout Changes

**New Width:**
```svelte
class="w-[85vw] max-w-[1200px] lg:w-[80vw]"
```
- 85% viewport width on small/medium screens
- 80% on large screens  
- Max 1200px to prevent overly wide panels

**Structure:**
```svelte
<div class="fixed right-0 top-0 z-50 flex h-full w-[85vw] max-w-[1200px] lg:w-[80vw] flex-col border-l bg-card shadow-xl">
  <!-- Header (unchanged) -->
  <div class="flex items-center justify-between border-b px-4 py-3">...</div>
  
  <!-- Status Bar (unchanged) -->
  <div class="border-b p-4">...</div>
  
  <!-- Tabs -->
  <div class="border-b px-4">
    <nav class="flex gap-2" role="tablist">
      {#if agent.status === 'active'}
        <TabButton active={activeTab === 'activity'} onclick={() => activeTab = 'activity'}>Activity</TabButton>
      {/if}
      {#if agent.status === 'completed' || agent.status === 'abandoned'}
        <TabButton active={activeTab === 'investigation'} onclick={() => activeTab = 'investigation'}>Investigation</TabButton>
      {/if}
      {#if agent.status === 'completed'}
        <TabButton active={activeTab === 'synthesis'} onclick={() => activeTab = 'synthesis'}>Synthesis</TabButton>
      {/if}
    </nav>
  </div>
  
  <!-- Tab Content -->
  <div class="flex-1 overflow-y-auto">
    {#if activeTab === 'activity'}
      <ActivityTab {agent} events={agentEvents} />
    {:else if activeTab === 'investigation'}
      <InvestigationTab {agent} />
    {:else if activeTab === 'synthesis'}
      <SynthesisTab {agent} />
    {/if}
  </div>
  
  <!-- Quick Commands Footer (unchanged) -->
  <div class="border-t p-4">...</div>
</div>
```

### Component Breakdown

| Component | Responsibility | Estimated Lines |
|-----------|---------------|-----------------|
| `AgentDetailTabs.svelte` | Tab navigation logic | ~50 |
| `ActivityTab.svelte` | Live SSE feed with filters | ~150 |
| `InvestigationTab.svelte` | Workspace/artifact viewer | ~100 |
| `SynthesisTab.svelte` | D.E.K.N. display | ~120 |
| `agent-detail-panel.svelte` | Updated container | ~200 (reduced from 523) |

---

## Implementation Recommendations

### Recommended Approach ⭐

**Modular Tab Components** - Extract tab content into separate components with clear interfaces.

**Why this approach:**
- Reduces monolithic file from 523 lines to ~200
- Each tab component is testable in isolation
- Tab visibility logic centralized in parent
- Allows parallel development of tabs

**Trade-offs accepted:**
- More files to manage
- Some prop drilling for agent state
- Slightly more complex imports

**Implementation sequence:**
1. Create tab infrastructure (TabButton, activeTab state)
2. Extract ActivityTab from existing Live Activity section
3. Create SynthesisTab from existing Synthesis section
4. Create InvestigationTab (new functionality)
5. Update panel layout with new width
6. Add workspace file API endpoints

### Alternative Approaches Considered

**Option B: Accordion/Collapsible Sections (No Tabs)**
- **Pros:** Familiar pattern, all content visible
- **Cons:** Doesn't address width issue; content still mixed
- **When to use:** If tabs prove confusing in user testing

**Option C: Split Panel (Left = Info, Right = Content)**
- **Pros:** More screen real estate usage
- **Cons:** Complexity; hard at 666px width
- **When to use:** Desktop-only dashboard variant

---

## Implementation Details

**What to implement first:**
1. Tab navigation infrastructure (activeTab state, TabButton component)
2. Width adjustment to 80-85%
3. Extract ActivityTab (most straightforward)

**Things to watch out for:**
- ⚠️ SSE event filtering performance with 100+ events
- ⚠️ Tab state persistence (remember last tab per agent?)
- ⚠️ Mobile/narrow viewport tab label overflow

**Areas needing further investigation:**
- Workspace file API endpoint design
- Investigation artifact parsing (markdown rendering?)
- Real-time updates when tab not visible

**Success criteria:**
- ✅ Panel works at 666px width with all tabs functional
- ✅ Active agents show Activity tab by default
- ✅ Completed agents show Synthesis tab by default
- ✅ Tab switching is instant (<50ms)
- ✅ Existing functionality preserved

---

## References

**Files Examined:**
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte` - Current panel implementation
- `web/src/lib/stores/agents.ts` - Agent data types and SSE handling
- `web/src/lib/components/agent-card/agent-card.svelte` - Display state computation
- `web/src/routes/+page.svelte` - Dashboard layout context

**Related Constraints:**
- Dashboard 666px width minimum (SPAWN_CONTEXT.md)
- Panel should be 80-85% viewport width (task requirement)

---

## Investigation History

**2026-01-06:** Investigation started
- Initial question: How to design tabbed interface for agent detail pane
- Context: Task orch-go-hmj61 - Dashboard redesign

**2026-01-06:** Context gathering complete
- Reviewed 4 key files totaling ~2,300 lines
- Identified existing SSE filtering, synthesis structure, width constraints

**2026-01-06:** Design synthesis complete
- Proposed 3-tab structure with state-driven visibility
- Component breakdown for modular implementation
