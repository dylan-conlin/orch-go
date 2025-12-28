<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** SSE Stream section shows event count but content is only visible when manually expanded - needs preview or clearer expand indication.

**Evidence:** Code at web/src/routes/+page.svelte:926-989 shows collapsible section with events hidden by default (sseStream: false).

**Knowledge:** Users seeing "100 events" without visual content is confusing; adding a collapsed preview will make the data accessible without overwhelming the layout.

**Next:** Implement - add a preview row showing last 3 events when collapsed, matching the Agent Lifecycle panel pattern.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Dashboard SSE Events Section Shows Count But No Content

**Question:** Why does the SSE Stream section show "100 events" but users cannot see the actual events?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: SSE Stream section is collapsible but collapsed by default

**Evidence:** Line 63 sets `sseStream: false` (collapsed by default). The section at lines 926-989 only shows events when expanded via `{#if sectionState.sseStream}`.

**Source:** web/src/routes/+page.svelte:63, 926-989

**Significance:** Users see "100 events" in the header but have no immediate visual indication of content. Unlike the Agent Lifecycle panel (always visible), SSE content is completely hidden when collapsed.

---

### Finding 2: Agent Lifecycle panel provides a better pattern

**Evidence:** The Agent Lifecycle panel at lines 861-924 shows events immediately without requiring expansion - it uses max-h-64 overflow-y-auto to bound the height while keeping content visible.

**Source:** web/src/routes/+page.svelte:861-924

**Significance:** The solution should follow this pattern - show a preview of events when collapsed, with full expansion for more detail.

---

### Finding 3: Constraint requires max-h-64 for event panels

**Evidence:** From spawn context constraints: "Dashboard event panels max-h-64 for visibility without overwhelming layout"

**Source:** SPAWN_CONTEXT.md constraint

**Significance:** Must respect height constraint - solution should add preview when collapsed, not remove collapsibility entirely.

---

## Synthesis

**Key Insights:**

1. **Collapsible sections need preview content** - When a section is collapsed but shows a count, users expect to see some content without expanding. The Agent Lifecycle panel already does this well.

2. **Progressive disclosure requires visual feedback** - Showing "100 events" without any visible events is confusing. Adding a 3-event preview provides immediate value while respecting the max-h-64 constraint.

**Answer to Investigation Question:**

Users cannot see SSE events because the section is collapsed by default and shows no content when collapsed. The fix adds a 3-event preview when collapsed, with a clear "click to expand" indicator when more events exist.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build succeeds (verified: `bun run build` completed successfully)
- ✅ Web server starts (verified: curl to localhost:5188 returns HTML)
- ✅ API responds (verified: curl to localhost:3348/api/agents returns JSON)

**What's untested:**

- ⚠️ Visual appearance in browser (light tier spawn, not browser-tested)
- ⚠️ SSE event streaming with preview visible

**What would change this:**

- If preview text is unreadable at 10px font size on certain screens
- If truncate behavior cuts off important information

---

## References

**Files Examined:**
- web/src/routes/+page.svelte:926-989 - SSE Stream section implementation
- web/src/lib/stores/agents.ts - SSE event types and store

---

## Investigation History

**2025-12-27:** Investigation completed
- Status: Complete
- Key outcome: Added 3-event preview to collapsed SSE Stream section
