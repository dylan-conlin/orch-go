<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The dashboard's +page.svelte (920 lines) is a monolithic component with high complexity from interleaved concerns; agents.ts (612 lines) is well-structured but could benefit from splitting SSE handling into a separate module.

**Evidence:** Analysis shows +page.svelte contains: 12 store imports, 6 filter state variables, 7 section collapse states, 8 helper functions, 170+ lines of template per mode (operational vs historical), and inline sorting logic that should be extracted.

**Knowledge:** The codebase already follows good patterns (domain components like AgentCard, NeedsAttention are well-extracted), but the main page has become a "god component" that orchestrates too many concerns; the store pattern works well and should be preserved.

**Next:** Implement 4-phase refactor: (1) Extract stats bar, (2) Extract filter bar + sorting logic, (3) Create mode-specific page components, (4) Split agents.ts SSE handling.

---

# Investigation: Dashboard UI Structure Analysis and Refactor Plan

**Question:** What is the complexity and component extraction opportunities in the dashboard UI (+page.svelte and agents.ts)?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: +page.svelte is a 920-line monolithic component with mixed concerns

**Evidence:** 
- Lines 1-46: 18 imports from stores and components
- Lines 47-64: 7 filter/sort state variables + 7 section collapse state variables
- Lines 67-93: localStorage persistence logic for section state
- Lines 95-99: 2 derived reactive statements for unique skills/projects
- Lines 101-178: onMount/onDestroy lifecycle with SSE, data fetching, intervals
- Lines 180-348: 8 helper functions (formatTime, getEventIcon, sortAgents, applyFilters, etc.)
- Lines 350-557: Stats bar with mode toggle, indicators, connection button (~207 lines)
- Lines 559-604: Operational mode template (~45 lines)
- Lines 605-914: Historical mode template (~309 lines)
- Lines 918-919: Agent detail panel import

**Source:** web/src/routes/+page.svelte

**Significance:** The file handles too many responsibilities: state management, data fetching, filtering logic, sorting logic, UI rendering for two different modes, and section collapse persistence. This makes it hard to maintain and test.

---

### Finding 2: agents.ts is well-structured but has growing SSE complexity (612 lines)

**Evidence:**
- Lines 1-85: Type definitions (Agent, SSEEvent, Synthesis, GapAnalysis)
- Lines 87-103: Event ID generation utilities
- Lines 105-195: Agent store creation with fetch/debounce/cancel logic
- Lines 197-239: Derived stores (activeAgents, idleAgents, completedAgents, recentAgents, archivedAgents)
- Lines 241-290: SSE event store with deduplication
- Lines 292-426: SSE connection manager (connect, handlers, event listeners)
- Lines 428-532: handleSSEEvent function (104 lines) - handles message.part, session.status, agent lifecycle
- Lines 534-559: extractActivityText helper
- Lines 561-611: createIssue API call, disconnectSSE

**Source:** web/src/lib/stores/agents.ts

**Significance:** The file is logically organized but handleSSEEvent (104 lines) is a complex function handling multiple event types. SSE connection management could be extracted into a dedicated module.

---

### Finding 3: Domain components are well-extracted and follow good patterns

**Evidence:**
- `AgentCard` (484 lines): Self-contained card with display state logic, context indicators, title formatting
- `NeedsAttention` (239 lines): Consolidated attention items (errors, blocked, pending reviews)
- `UpNextSection` (191 lines): Priority queue with auto-expand on urgent items
- `ReadyQueueSection` (106 lines): Simple collapsible queue display
- `RecentWins` (109 lines): Completed agents with outcome display
- `AgentDetailPanel` (523 lines): Full agent details with quick copy, synthesis, commands
- `CollapsibleSection`, `PendingReviewsSection`, `SettingsPanel`: Clean extraction patterns

**Source:** web/src/lib/components/*/

**Significance:** The component library follows good patterns - each component handles its own domain concerns. The problem is not component design but rather the orchestration layer in +page.svelte.

---

### Finding 4: Stats bar deserves its own component (207+ lines of template)

**Evidence:** Lines 351-557 in +page.svelte contain:
- Mode toggle (operational/historical)
- Error indicator with tooltip
- Active agents indicator with tooltip
- Focus indicator (conditional on mode/drift)
- Servers indicator (conditional on mode)
- Beads indicator (clickable, toggles section)
- Daemon indicator with capacity display
- Connection button with status
- Settings panel

Each indicator follows the same pattern: Tooltip.Root > Tooltip.Trigger > content > Tooltip.Content

**Source:** web/src/routes/+page.svelte:351-557

**Significance:** This is a self-contained "orchestration status bar" that could be extracted. It has its own interaction patterns (clickable beads indicator) and conditional rendering logic.

---

### Finding 5: Filter bar and sorting logic should be extracted together

**Evidence:**
- Lines 47-53: Filter state variables (statusFilter, skillFilter, projectFilter, sortBy, activeOnly)
- Lines 235-242: clearFilters function and hasActiveFilters computed
- Lines 246-321: sortAgents function (75 lines) with 6 sorting modes
- Lines 324-338: applySkillFilter, applyProjectFilter, applyFilters helper chain
- Lines 340-348: Computed sorted/filtered agent lists
- Lines 633-710: Filter bar template in historical mode (~77 lines)

**Source:** web/src/routes/+page.svelte

**Significance:** Filtering and sorting is a cohesive concern that spans state, logic, and UI. Extracting into a composable (useAgentFilters) + component (FilterBar) would improve testability and reuse.

---

### Finding 6: Operational and Historical modes have distinct rendering needs

**Evidence:**
- Operational mode (lines 559-604): UpNext, Active Agents (always visible), NeedsAttention, RecentWins, ReadyQueue
- Historical mode (lines 605-914): UpNext, ReadyQueue, PendingReviews, SwarmMap (with filters), CollapsibleSections, Event Panels

Key differences:
1. Operational: Active agents are non-collapsible, always prominent
2. Historical: Full archive with filters and collapsible sections
3. Historical: Has event panels (Agent Lifecycle, SSE Stream)
4. Both: Share UpNext and ReadyQueue but with different collapse defaults

**Source:** web/src/routes/+page.svelte

**Significance:** The two modes are distinct enough to warrant separate components (OperationalDashboard, HistoricalDashboard) that share common sub-components.

---

### Finding 7: Stores follow consistent patterns and are well-organized

**Evidence:** 
- All stores in `web/src/lib/stores/` follow the same pattern:
  - `writable<T>()` for base state
  - `createXStore()` factory with fetch/set/update methods
  - Export single store instance
  - Helper functions exported separately (getDaemonEmoji, getDriftEmoji, etc.)

- Stores: agents.ts (612), agentlog.ts, beads.ts (97), config.ts, daemon.ts (74), dashboard-mode.ts (67), focus.ts (52), hotspot.ts, pending-reviews.ts (142), servers.ts, theme.ts, usage.ts

**Source:** web/src/lib/stores/*.ts

**Significance:** The store pattern is working well. Changes should preserve this architecture rather than restructuring it.

---

## Synthesis

**Key Insights:**

1. **Component extraction is proven effective** - The existing components (AgentCard, NeedsAttention, etc.) show the codebase can handle well-extracted components. The problem is that +page.svelte hasn't kept pace with this pattern.

2. **The stats bar is a natural extraction target** - With 200+ lines of template and distinct interaction patterns, it's a clear first extraction that reduces +page.svelte complexity immediately.

3. **Filtering is a cross-cutting concern** - The filter state, sorting logic, and filter UI are interleaved throughout the file. A composable pattern (hook + component) would clean this up.

4. **Mode separation is already implicit** - The code already has `{#if $dashboardMode === 'operational'}...{:else}...{/if}` structure. Making this explicit with separate components just formalizes what's already there.

5. **SSE handling is a candidate for extraction** - The handleSSEEvent function (104 lines) with its debouncing and processing state timers could be a dedicated module, making agents.ts more focused on agent state.

**Answer to Investigation Question:**

The +page.svelte file is a 920-line monolithic component that violates single responsibility by handling: state management (14 variables), lifecycle management (SSE, data fetching, intervals), sorting/filtering logic (75+ lines of sort function), UI for two distinct modes (operational/historical), and section collapse persistence. 

The agents.ts store (612 lines) is better organized but has a 104-line handleSSEEvent function that handles multiple event types and could be modularized.

The existing component library proves extraction patterns work well in this codebase. The refactor should:
1. Extract StatsBar component (200+ lines)
2. Extract filter composable + FilterBar component (150+ lines)
3. Create OperationalDashboard and HistoricalDashboard mode components
4. Optionally split SSE event handling from agents.ts

---

## Structured Uncertainty

**What's tested:**

- ✅ Line counts verified by reading actual files
- ✅ Component structure verified by glob and file reads
- ✅ Store patterns verified across multiple store files

**What's untested:**

- ⚠️ Actual refactor compilation (not built/tested)
- ⚠️ Performance impact of additional component layers (assumed negligible)
- ⚠️ Svelte 5 runes compatibility for extracted composables (assumed compatible)

**What would change this:**

- If performance profiling shows component overhead matters, keep more inline
- If Svelte 5 composables don't work well, use different extraction pattern
- If filter state needs cross-route persistence, store-based approach instead of composable

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Phased Component Extraction** - Extract in order of independence, testing each phase before proceeding.

**Why this approach:**
- Reduces risk by making incremental, testable changes
- Each phase delivers value independently
- Maintains existing patterns (stores, component structure)

**Trade-offs accepted:**
- More git commits than a single large refactor
- Some temporary duplication during transition
- May take 2-4 hours across phases

**Implementation sequence:**

#### Phase 1: Extract StatsBar Component (1 hour)
1. Create `web/src/lib/components/stats-bar/stats-bar.svelte`
2. Move lines 351-557 from +page.svelte
3. Props: `sectionState` (bind), connection callbacks, store subscriptions
4. Verify: Dashboard loads, mode toggle works, indicators display

#### Phase 2: Extract Filter Composable and FilterBar Component (1 hour)
1. Create `web/src/lib/composables/use-agent-filters.ts`:
   - Export filter state variables
   - Export sortAgents function
   - Export applyFilters function
   - Export clearFilters, hasActiveFilters
2. Create `web/src/lib/components/filter-bar/filter-bar.svelte`:
   - Move filter UI template (lines 633-710)
   - Accept filter state via props/context
3. Verify: Historical mode filtering works, sorts correctly

#### Phase 3: Extract Mode-Specific Dashboard Components (1 hour)
1. Create `web/src/lib/components/operational-dashboard/operational-dashboard.svelte`:
   - Move lines 559-604 content
   - Props: sectionState bindings, sorted agent lists
2. Create `web/src/lib/components/historical-dashboard/historical-dashboard.svelte`:
   - Move lines 605-914 content
   - Import FilterBar, CollapsibleSection
3. Simplify +page.svelte to:
   - State management
   - Lifecycle (onMount/onDestroy)
   - Mode switch rendering
4. Verify: Both modes render correctly

#### Phase 4 (Optional): Extract SSE Handler Module (30 min)
1. Create `web/src/lib/stores/sse-handler.ts`:
   - Move handleSSEEvent function
   - Move extractActivityText helper
   - Move processing state timers
   - Export connectSSE, disconnectSSE
2. Import in agents.ts, re-export for consumers
3. Verify: SSE connection works, agent updates propagate

### Alternative Approaches Considered

**Option B: Create a single DashboardPage component with sub-components**
- **Pros:** Simpler extraction, fewer new files
- **Cons:** Still leaves +page.svelte handling too much logic
- **When to use instead:** If team prefers fewer files over separation of concerns

**Option C: Move to route-based mode separation (+page/operational, +page/historical)**
- **Pros:** Complete separation of modes, better code splitting
- **Cons:** May complicate shared state, breaks URL convention
- **When to use instead:** If modes diverge significantly in future

**Rationale for recommendation:** Phase 1-3 approach preserves URL structure while achieving good separation. Each phase is independently valuable and testable.

---

### Implementation Details

**What to implement first:**
- Phase 1 (StatsBar) is the cleanest extraction with minimal dependencies
- Delivers immediate value by reducing +page.svelte by 200+ lines
- Tests basic extraction pattern before more complex phases

**Things to watch out for:**
- ⚠️ Section state binding needs two-way flow (bind:expanded pattern)
- ⚠️ Filter composable may need Svelte 5 `$state` instead of `writable` for reactivity
- ⚠️ Event handlers reference parent scope - need to pass callbacks or use stores

**Areas needing further investigation:**
- Whether to use Svelte 5 runes ($state, $derived) or stick with stores
- Performance of additional component boundaries (likely fine)
- Test coverage for new components

**Success criteria:**
- ✅ +page.svelte reduced from 920 lines to ~300 lines
- ✅ No visual regressions (both modes render identically)
- ✅ All existing functionality preserved (SSE, filtering, sections)
- ✅ Each new component is independently testable

---

## References

**Files Examined:**
- `web/src/routes/+page.svelte` (920 lines) - Main dashboard page
- `web/src/lib/stores/agents.ts` (612 lines) - Agent state and SSE handling
- `web/src/lib/components/agent-card/agent-card.svelte` (484 lines) - Agent card component
- `web/src/lib/components/needs-attention/needs-attention.svelte` (239 lines) - Attention section
- `web/src/lib/components/up-next-section/up-next-section.svelte` (191 lines) - Priority queue
- `web/src/lib/components/recent-wins/recent-wins.svelte` (109 lines) - Recent completions
- `web/src/lib/components/ready-queue-section/ready-queue-section.svelte` (106 lines) - Ready queue
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte` (523 lines) - Detail panel
- `web/src/lib/stores/` - Various domain stores (beads, daemon, focus, etc.)

**Commands Run:**
```bash
# Pattern search for components
glob "web/src/lib/components/**/*.svelte"

# Pattern search for stores
glob "web/src/lib/stores/*.ts"
```

**Related Artifacts:**
- **Decision:** Prior constraint notes high patch density signals missing coherent model
- **Context:** Dashboard has accumulated complexity from incremental additions

---

## Investigation History

**2026-01-04 15:00:** Investigation started
- Initial question: Analyze dashboard UI structure and plan refactor
- Context: Dashboard complexity from accumulated patches needs architectural review

**2026-01-04 15:30:** Analysis complete
- Read +page.svelte, agents.ts, and key components
- Identified 7 key findings about structure and extraction opportunities

**2026-01-04 15:45:** Investigation completed
- Status: Complete
- Key outcome: 4-phase refactor plan with clear extraction targets and success criteria
