<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Two files (+page.svelte and agents.ts) have 16 fix commits each, revealing 6 distinct root causes that can be addressed through targeted refactoring.

**Evidence:** Categorized all 32 fix commits into patterns: SSE State Management (10), Svelte Keyed Rendering (6), Responsive Layout (4), SSR/Hydration (4), Status Model Complexity (4), API Integration (4).

**Knowledge:** The hotspot exists because +page.svelte has accumulated 920 lines with mixed concerns (stats bar, mode toggle, filters, sections, event panels) and agents.ts handles both data fetching AND SSE event processing in 612 lines.

**Next:** Implement 3-phase refactor: (1) Extract SSE connection manager, (2) Extract stats bar and mode toggle components, (3) Consolidate agent status model. Creates 4 new files, modifies 2 existing.

---

# Investigation: Analyze Dashboard UI Hotspots

**Question:** Why do +page.svelte and agents.ts have 16 fix commits each, and how should they be refactored to reduce future fixes?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: SSE State Management is the Primary Pain Point (10 fixes)

**Evidence:** 10 of 32 fixes relate to SSE event handling:
- `ed772bac` - 125% CPU from per-session HTTP polling feedback loop
- `61a6aa20` - Gold border flashing from rapid busy/idle toggles
- `501d64f9` - Pulsing on completed agents from late SSE events
- `a3988806` - Duplicate events from missing part.id deduplication
- `7cf069a3` - Wrong sessionID path for message.part events
- `15356af3` - message.part.updated event type mismatch
- `de002c60` - Debounce processing state to reduce flapping
- `13f0a4b9` - Chrome 6-connection limit exhaustion

**Source:** 
- `web/src/lib/stores/agents.ts:366-532` - handleSSEEvent function (167 lines)
- `web/src/lib/stores/agents.ts:312-364` - connectSSE function
- `web/src/lib/stores/agentlog.ts:116-204` - duplicate SSE connection logic

**Significance:** SSE event handling is scattered across two files with duplicate connection management patterns. The handleSSEEvent function at 167 lines has grown organically with each fix, adding debounce timers, generation counters, and status checks without clear structure.

---

### Finding 2: Svelte Keyed Rendering Caused 6 Related Fixes

**Evidence:** 6 fixes for duplicate key errors and rendering issues:
- `485fb343` - Deduplicate agents by title
- `714b9bf7` - Use agent.id as key
- `24a7a3f4` - Use session_id as key
- `ce876a19` - Use unique event IDs instead of array index
- `5a5d7334` - Use index as key (temporary fix)
- `c6b94834` - Stabilize grid positions with spawned_at sort

**Source:**
- `web/src/routes/+page.svelte:579,718,741,759,775` - {#each} blocks with composite keys
- `web/src/lib/stores/agents.ts:87-102` - ID generation functions

**Significance:** The team tried 4 different keying strategies before settling on `${agent.id}-${agent.session_id ?? i}`. This pattern should be documented and reused consistently. The root cause is that agents can have duplicate IDs (OpenCode session resurrection) and duplicate session_ids (daemon respawns).

---

### Finding 3: +page.svelte Has Accumulated Mixed Concerns (920 lines)

**Evidence:** Single file handles:
- Stats bar (lines 352-557) - 205 lines with 10+ indicators
- Dashboard mode toggle (lines 355-368) 
- Section collapse state (lines 54-93)
- Filter/sort logic (lines 246-348) - 102 lines
- Active agents section (lines 567-592)
- Needs attention, recent wins, ready queue (lines 594-616)
- Historical mode swarm map with filters (lines 605-800)
- Event panels side-by-side (lines 804-914)
- Lifecycle/mount/destroy hooks (lines 101-178)

**Source:** `web/src/routes/+page.svelte` - 920 total lines

**Significance:** This is a "god component" anti-pattern. Each new feature adds more lines. The stats bar alone (205 lines) is complex enough to be its own component. The operational vs historical mode conditional rendering adds significant complexity.

---

### Finding 4: Status Model Complexity Caused 4 Semantic Fixes

**Evidence:** 4 fixes for agent status semantics:
- `6f62bd8a` - Separate working from dead/stalled agents
- `3a834ac0` - Include idle in active (API returns idle for optimization)
- `9e265fcf` - Show last activity on initial load
- `5a5d7334` - Add idle status type

**Source:**
- `web/src/lib/stores/agents.ts:200-239` - Derived stores for status filtering
- `web/src/lib/components/agent-card/agent-card.svelte:30-59` - getDisplayState function

**Significance:** The status model has evolved organically. "Active" means different things (has session, is processing, not completed). Agent-card has a local getDisplayState() function because the store-level status isn't sufficient. This logic should be centralized.

---

### Finding 5: SSR/Hydration Edge Cases Required 4 Fixes

**Evidence:**
- `1e34c04f` - Svelte 5 runes mode conflict with legacy syntax
- `e89a8a58` - Tooltip hydration error from block elements in buttons
- `f9742a61` - Dashboard mode toggle not re-rendering
- `7f4668fa` - Race condition in data loading

**Source:**
- `web/src/routes/+page.svelte:67-78` - loadSectionState with typeof window check
- `web/src/lib/stores/dashboard-mode.ts` - init() pattern for client lifecycle

**Significance:** SvelteKit SSR requires careful handling of browser APIs (localStorage, window). The pattern of `init()` called from `onMount()` is established but not documented. New components risk repeating these mistakes.

---

### Finding 6: Responsive Layout Required 4 Fixes

**Evidence:**
- `57170ec0` - Status bar layout at 666px
- `3632cc9b` - Flex-wrap for narrow viewports
- `c6b94834` - Stabilize grid positions (related to layout jostling)
- `dfd61e05` - Stable sort for Recent section

**Source:** `web/src/routes/+page.svelte:353-557` - Stats bar markup with flex-wrap and gap classes

**Significance:** The stats bar has grown to 10+ indicators. Each addition risks breaking responsive behavior. The abbreviated labels (err, rdy, blk) and hidden text at narrow widths is a patch, not a solution.

---

## Synthesis

**Key Insights:**

1. **SSE Connection Management is Duplicated** - agents.ts and agentlog.ts have near-identical SSE connection patterns (generation counters, reconnect timeouts, abort controllers). Extract to shared module.

2. **Agent Status is Computed in Multiple Places** - The store has derived stores, the API returns status, agent-card computes displayState, and handleSSEEvent updates is_processing. Consolidate to one authoritative source.

3. **+page.svelte is a Feature Aggregator, Not a Page** - The file orchestrates features but also implements them inline. Extract components to match the mental model of the dashboard.

**Answer to Investigation Question:**

The hotspots exist because both files grew through feature accumulation without refactoring. Each fix addressed a symptom without restructuring. The 16 fixes each suggest the files are past the complexity threshold where incremental fixes become more expensive than refactoring.

The recommended refactor addresses root causes:
- SSE fixes → Extract SSEConnectionManager 
- Keyed rendering fixes → Document and standardize key patterns
- Status model fixes → Centralize in agents store with clear semantics
- +page.svelte size → Extract StatsBar, ModeToggle, and EventPanels components

---

## Structured Uncertainty

**What's tested:**
- ✅ Fix commit categorization verified by reading each commit message and diff
- ✅ Line counts verified by reading files
- ✅ Duplicate SSE patterns verified by comparing agents.ts and agentlog.ts

**What's untested:**
- ⚠️ Performance improvement from SSE extraction (not benchmarked)
- ⚠️ Maintainability improvement (subjective, not measured)
- ⚠️ Whether extracting StatsBar breaks existing tests

**What would change this:**
- If StatsBar extraction proves more complex than expected (tight coupling to sectionState)
- If SSE deduplication logic is more entangled than it appears
- If the project moves to Svelte 5 runes, some patterns will change

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**3-Phase Extraction** - Extract shared modules first, then components, then consolidate status model.

**Why this approach:**
- Addresses highest-fix-count areas first (SSE: 10 fixes)
- Creates reusable patterns before component extraction
- Each phase is independently valuable (can stop after phase 1 if needed)

**Trade-offs accepted:**
- More commits than a single big refactor
- Some temporary duplication during transition

**Implementation sequence:**

1. **Phase 1: SSE Connection Manager** (Highest impact, 10 fixes)
   - Create `web/src/lib/services/sse-connection.ts`
   - Extract: generation counters, reconnect logic, abort controllers
   - Migrate agents.ts and agentlog.ts to use shared service
   - Keep event handlers in original files (domain-specific)

2. **Phase 2: Extract StatsBar Component** (Reduces +page.svelte by 200+ lines)
   - Create `web/src/lib/components/stats-bar/stats-bar.svelte`
   - Move: mode toggle, all indicators, connection button
   - Props: stores passed in, events emitted out
   - Includes responsive layout logic

3. **Phase 3: Consolidate Agent Status Model**
   - Add `computeDisplayState()` to agents.ts
   - Remove duplicate logic from agent-card.svelte
   - Update derived stores to use new function
   - Document the status model in code comments

### Alternative Approaches Considered

**Option B: Full Component Split**
- **Pros:** Maximum separation of concerns
- **Cons:** Over-engineering for current needs, more files to maintain
- **When to use instead:** If dashboard grows to 3+ modes or team grows

**Option C: Status-First Refactor**
- **Pros:** Addresses semantic confusion directly
- **Cons:** Doesn't reduce file sizes, doesn't fix SSE issues
- **When to use instead:** If most bugs are status-related going forward

**Rationale for recommendation:** Phase 1 addresses the most fixes (10/32) with a well-bounded extraction. It creates patterns reusable across the codebase. Phases 2-3 are optional but valuable.

---

### Implementation Details

**What to implement first:**
- `web/src/lib/services/sse-connection.ts` - Shared SSE connection manager
- Move: generation counter, reconnect timeout, abort controller patterns
- API: `createSSEConnection(url, options) → { connect, disconnect, status }`

**File targets:**
- Create: `web/src/lib/services/sse-connection.ts`
- Create: `web/src/lib/components/stats-bar/stats-bar.svelte`
- Create: `web/src/lib/components/stats-bar/index.ts`
- Modify: `web/src/lib/stores/agents.ts` (use SSE service, add computeDisplayState)
- Modify: `web/src/routes/+page.svelte` (import StatsBar, remove 200 lines)

**Things to watch out for:**
- ⚠️ StatsBar needs bind:expanded for sectionState - pass as prop with event
- ⚠️ SSE extraction must preserve handleSSEEvent calls (domain-specific)
- ⚠️ Agents store has module-level state (timers, controllers) - need cleanup

**Areas needing further investigation:**
- Whether Svelte 5 migration is planned (would change component patterns)
- Whether event panels (Agent Lifecycle, SSE Stream) should also be extracted

**Success criteria:**
- ✅ No more SSE-related fixes after Phase 1
- ✅ +page.svelte under 700 lines after Phase 2
- ✅ Agent status logic in one place after Phase 3
- ✅ All existing tests pass

---

## References

**Files Examined:**
- `web/src/routes/+page.svelte` - Main dashboard page (920 lines)
- `web/src/lib/stores/agents.ts` - Agent store with SSE handling (612 lines)
- `web/src/lib/stores/agentlog.ts` - Duplicate SSE pattern (222 lines)
- `web/src/lib/components/agent-card/agent-card.svelte` - Status display logic (484 lines)
- `web/src/lib/components/collapsible-section/collapsible-section.svelte` - Example of extracted component

**Commands Run:**
```bash
# Count fix commits
git log --oneline --all -- web/src/routes/+page.svelte | grep -E '^[a-f0-9]+\s+fix' | wc -l
# Result: 16

git log --oneline --all -- web/src/lib/stores/agents.ts | grep -E '^[a-f0-9]+\s+fix' | wc -l
# Result: 16

# Analyze each commit
for commit in $(git log --oneline --all -- [file] | grep -E '^[a-f0-9]+\s+fix' | cut -d' ' -f1); do
  git show --stat $commit | head -20
done
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-03-structured-logging-orch-go.md` - Prior architecture decision
- **Feature:** `feat-023` in `.orch/features.json` - Related dashboard status simplification

---

## Investigation History

**2026-01-04 09:00:** Investigation started
- Initial question: Why are +page.svelte and agents.ts hotspots with 16 fixes each?
- Context: Hotspot analysis identified these as top two files needing refactor

**2026-01-04 09:30:** Commit categorization complete
- Categorized all 32 commits into 6 patterns
- SSE emerged as primary pain point (10 fixes)

**2026-01-04 10:00:** Investigation completed
- Status: Complete
- Key outcome: 3-phase refactor plan with specific file targets
