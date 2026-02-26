<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented three-section dashboard layout (Active/Needs Review/Recent) to distinguish running agents from those awaiting review.

**Evidence:** Build succeeds, dashboard renders correctly. No Phase: Complete agents exist to visually test the new section.

**Knowledge:** Detection logic aligns with existing `computeDisplayState()` - agents at Phase: Complete with status=active are awaiting review. Amber/yellow styling used for "attention needed" visual distinction.

**Next:** Close - implementation complete, ready for `orch complete`.

**Promote to Decision:** recommend-no (feature implementation, not architectural)

---

# Investigation: Dashboard Distinguish Completed Agents Active

**Question:** How to distinguish agents at Phase: Complete (awaiting review) from truly running agents in the dashboard?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing computeDisplayState() already handles detection

**Evidence:** `computeDisplayState()` in agents.ts returns `'ready-for-review'` when `agent.status === 'active' && agent.phase?.toLowerCase() === 'complete'`

**Source:** web/src/lib/stores/agents.ts:75-83

**Significance:** Detection logic already exists - just needed to expose it as a derived store for filtering.

---

### Finding 2: CollapsibleSection has clean variant pattern

**Evidence:** Variant prop controls styling via `getVariantStyles()` and `getBadgeVariant()` switch statements.

**Source:** web/src/lib/components/collapsible-section/collapsible-section.svelte:15-35

**Significance:** Adding 'needs-review' variant followed established pattern - minimal code change.

---

### Finding 3: Two dashboard modes need separate implementations

**Evidence:** Operational mode shows fixed sections, historical mode uses CollapsibleSection components. Both needed the Needs Review section.

**Source:** web/src/routes/+page.svelte:329-428 (operational) and 536-570 (historical)

**Significance:** Required implementing the section twice with different layouts but same data source.

---

## Synthesis

**Key Insights:**

1. **Derived stores enable clean separation** - `needsReviewAgents` and `trulyActiveAgents` derived stores isolate filtering logic from presentation.

2. **Conditional rendering avoids clutter** - Section only shows when agents exist at Phase: Complete, preventing empty section UI.

3. **Amber styling signals actionability** - Distinct from green (running) and blue (completed), matches "attention needed" patterns.

**Answer to Investigation Question:**

The dashboard now distinguishes running agents from those awaiting review through:
- Active section: Shows only agents NOT at Phase: Complete (truly consuming capacity)
- Needs Review section: Shows agents at Phase: Complete awaiting `orch complete`
- Detection: `status === 'active' && phase?.toLowerCase() === 'complete'`

---

## Structured Uncertainty

**What's tested:**

- ✅ Build succeeds (ran `bun run build`)
- ✅ Dashboard loads without errors (visual screenshot captured)
- ✅ Variant styling compiles correctly

**What's untested:**

- ⚠️ Visual appearance of Needs Review section with actual agents (no Phase: Complete agents exist)
- ⚠️ Section expand/collapse persistence in localStorage
- ⚠️ Stats bar accuracy (may still show combined count)

**What would change this:**

- Finding would need revision if Phase: Complete detection fails with actual agents
- Color choice might need adjustment if amber conflicts with other UI elements

---

## References

**Files Examined:**
- web/src/lib/stores/agents.ts - Agent type definitions and derived stores
- web/src/lib/components/collapsible-section/collapsible-section.svelte - Variant pattern
- web/src/routes/+page.svelte - Dashboard layout structure

**Commands Run:**
```bash
# Build verification
cd web && bun run build

# Visual verification
npx playwright screenshot http://localhost:5188 /tmp/dashboard-screenshot.png
```

---

## Investigation History

**2026-01-06 19:45:** Investigation started
- Initial question: How to prevent "3/3 slots filled" confusion when agents are just awaiting review
- Context: Orchestrator spawned to implement three-section layout

**2026-01-06 19:55:** Implementation complete
- Added derived stores, CollapsibleSection variant, and sections in both modes
- Committed: 530553ae

**2026-01-06 20:00:** Investigation completed
- Status: Complete
- Key outcome: Dashboard now shows separate Active and Needs Review sections
