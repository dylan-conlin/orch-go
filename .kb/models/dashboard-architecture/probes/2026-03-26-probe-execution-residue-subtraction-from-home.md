# Probe: Execution Residue Subtraction — Does the Home Surface Survive as Comprehension-Only?

**Model:** dashboard-architecture
**Date:** 2026-03-26
**Status:** Complete
**claim:** DA-01
**verdict:** extends

---

## Question

The dashboard-architecture model claims a "two-mode design" (operational/historical) is a critical invariant. The investigation (orch-go-vo51p) classified 11 execution sections as residue that should be moved or deleted. Can the home page function as a pure comprehension surface after removing all execution rendering, or does it break the user's daily workflow?

---

## What I Tested

Performed a surgical subtraction of the home page:

1. **Deleted dead /thinking route** — `web/src/routes/thinking/+page.svelte` and `web/src/lib/stores/digest.ts` (no backend, concept absorbed by threads+briefs)

2. **Removed from home page:**
   - StatsBar component
   - Coaching health indicator
   - ServicesSection (Overmind process status)
   - Active Agents grid
   - Needs Review agent cards
   - NeedsAttention section
   - RecentWins section
   - ReadyQueueSection
   - SwarmMap (full agent archive with filters)
   - Agent Lifecycle Events panel
   - SSE Stream panel
   - AgentDetailPanel slide-out
   - Dashboard mode toggle (operational/historical)
   - CacheValidationBanner

3. **Kept on home page:**
   - Threads section (thinking spine) — unchanged
   - Unread Briefs quick link — unchanged
   - Open Questions quick link — unchanged
   - QuestionsSection detail — **promoted above fold** (was inside operational mode conditional)
   - ReviewQueueSection — unchanged
   - **New:** Condensed operational summary: "{N} agents active · {N} ready · {N} need review — View Work →"

4. **Removed imports:** AgentCard, AgentDetailPanel, CollapsibleSection, Button, Tooltip, StatsBar, RecentWins, NeedsAttention, ReadyQueueSection, UpNextSection, ServicesSection, CacheValidationBanner, agentlog stores, servicelog stores, usage, focus, servers, daemon, verification, dashboardMode, config, hotspots, services, coaching

```bash
# Verified: no type errors in home page after subtraction
npx svelte-check --threshold error 2>&1 | grep "routes/+page.svelte"
# (no output — zero errors)

# File size reduction
wc -l web/src/routes/+page.svelte
# Before: 1055 lines
# After: 345 lines (67% reduction)
```

---

## What I Observed

### Observation 1: The Subtraction Was Clean

All 11 execution sections could be removed without affecting the comprehension layer. No shared state between comprehension sections (threads, briefs, questions, review queue) and execution sections (agent grids, coaching, services, event streams). The only connection was the `agents` store — still imported for count-only display in the summary line.

### Observation 2: The Mode Toggle Was Purely Execution-Organizing

The operational/historical mode toggle existed solely to organize execution content below the fold. With execution content removed, the toggle has no purpose. Both modes had the same comprehension sections above the fold — the only difference was which execution sections appeared below.

### Observation 3: QuestionsSection Gains Visibility

QuestionsSection was previously gated behind `$dashboardMode === 'operational'` — invisible in historical mode. After promotion above the fold (always visible), blocking questions are consistently surfaced regardless of any mode state.

### Observation 4: 67% Line Reduction Matches Expected Ratio

The investigation classified the home page as ~30% comprehension / ~70% execution by code weight. The actual reduction (1055→345 lines = 67%) confirms the 70% execution estimate was accurate.

---

## Model Impact

- [x] **Extends** model with: Home page is now comprehension-only. Invariant #1 updated from "two-mode design is mutually exclusive" to "home page is comprehension-only, execution lives on Work route." The mode toggle concept is dead.
- [x] **Confirms** invariant #10 (content mode vs metadata mode): The subtraction removes execution *metadata* (agent counts, coaching status, service health) from the primary surface. The comprehension layer retains content-mode elements (threads with expandable prose, question text, review queue with brief availability). The summary line is explicitly metadata-mode but appropriately subordinated.
- [x] **Extends** model with: `/thinking` route confirmed as dead code — concept absorbed by threads+briefs. Route and digest store deleted.

---

## Notes

- The dashboard-mode store (`web/src/lib/stores/dashboard-mode.ts`) still exists but is no longer imported on the home page. It may still be referenced elsewhere — left in place for safety rather than deleting.
- The condensed summary line still requires the SSE connection for real-time agent counts. This is a lightweight connection (one slot) compared to the previous three SSE connections (primary + agentlog + servicelog).
- **Tension (from brief orch-go-vo51p):** The classification assumed Dylan doesn't use below-fold execution content daily. If he does start mornings by checking the Active Agents grid or coaching health from the home page, those should be re-added as bridge elements. The current subtraction is independently revertible per section.
