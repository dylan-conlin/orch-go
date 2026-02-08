<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Agent card synthesis section was duplicating TLDR content (shown in title AND synthesis) and showing "No synthesis available" placeholder when empty.

**Evidence:** Code review of agent-card.svelte lines 344-369 showed TLDR displayed twice; API data confirmed most completed agents have null synthesis but some have close_reason.

**Knowledge:** The `getDisplayTitle()` function already shows TLDR/close_reason for completed agents, making the synthesis section redundant. Placeholder text creates unnecessary whitespace.

**Next:** Close - fix implemented and tested.

**Confidence:** High (95%) - Fix is straightforward code removal with clear logic.

---

# Investigation: Agent Card Has Excess Whitespace

**Question:** Why do agent cards show excess whitespace when TLDR is missing, and how to fix it?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (95%)

---

## Findings

### Finding 1: TLDR/close_reason displayed redundantly

**Evidence:** The `getDisplayTitle()` function (lines 164-182) already extracts and displays TLDR for completed agents in the title. The synthesis section (lines 344-369) then showed the same TLDR again.

**Source:** `web/src/lib/components/agent-card/agent-card.svelte:164-182` (getDisplayTitle), lines 344-369 (synthesis section)

**Significance:** Users see the same content twice - once in the title and once in the synthesis section. This is wasteful and confusing.

---

### Finding 2: Placeholder text creates whitespace for agents without synthesis

**Evidence:** When `agent.synthesis?.tldr` and `agent.close_reason` are both null/empty, the code showed a "No synthesis available" placeholder text. This creates a visible section with no meaningful content.

**Source:** `web/src/lib/components/agent-card/agent-card.svelte:362-366` (original placeholder code)

**Significance:** Most completed agents have `synthesis: null` (confirmed via API). The placeholder creates unnecessary vertical space in every such card.

---

### Finding 3: Synthesis outcome badge is the only unique content

**Evidence:** The synthesis section's only non-redundant content is the `synthesis.outcome` badge (success/failure). The TLDR is already in the title, and close_reason falls back to the title as well.

**Source:** `web/src/lib/components/agent-card/agent-card.svelte:357-360` (outcome badge)

**Significance:** The entire synthesis section can be simplified to only show when an outcome badge exists.

---

## Synthesis

**Key Insights:**

1. **Content duplication** - The title section already displays TLDR/close_reason via `getDisplayTitle()`, making the synthesis section's text redundant.

2. **Unnecessary placeholders** - "No synthesis available" text provides no value and wastes vertical space.

3. **Minimal viable section** - The only unique content in the synthesis section is the outcome badge, so the section should only render when that badge exists.

**Answer to Investigation Question:**

The excess whitespace was caused by two issues: (1) redundant TLDR/close_reason text that was already displayed in the title, and (2) a "No synthesis available" placeholder that rendered even when there was nothing meaningful to show. The fix removes the redundant text display and only shows the synthesis section when there's an actual outcome badge to display.

---

## Confidence Assessment

**Current Confidence:** High (95%)

**Why this level?**

The fix is a straightforward simplification - removing code that duplicates information already displayed elsewhere. The logic is clear and the change is minimal.

**What's certain:**

- ✅ getDisplayTitle() already shows TLDR/close_reason in the title for completed agents
- ✅ Most agents have null synthesis (confirmed via API inspection)
- ✅ All Playwright tests pass after the change (21/21)

**What's uncertain:**

- ⚠️ Edge case where card height consistency matters for grid layout (mitigated by other sections having consistent rendering)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Remove redundant synthesis text display** - Only show the synthesis section when an outcome badge exists.

**Why this approach:**
- Eliminates content duplication (TLDR shown once, not twice)
- Removes unnecessary whitespace from placeholder
- Keeps meaningful content (outcome badge) when available

**Implementation sequence:**
1. Change condition from `agent.status === 'completed'` to `agent.status === 'completed' && agent.synthesis?.outcome`
2. Remove the TLDR/close_reason text display (already in title)
3. Remove the "No synthesis available" placeholder

---

## References

**Files Examined:**
- `web/src/lib/components/agent-card/agent-card.svelte` - Main component with the issue

**Commands Run:**
```bash
# Check API data for synthesis fields
curl -s http://localhost:3348/api/agents | jq '[.[] | select(.status == "completed") | {id, synthesis, close_reason}] | .[0:5]'

# Run Playwright tests
npx playwright test
```

---

## Investigation History

**2025-12-25 22:20:** Investigation started
- Initial question: Why does agent card show excess whitespace when TLDR is missing?
- Context: User reported visual issue with completed agent cards

**2025-12-25 22:25:** Root cause identified
- Found TLDR duplication and placeholder whitespace issues
- Implemented fix by simplifying synthesis section

**2025-12-25 22:27:** Investigation completed
- Final confidence: High (95%)
- Status: Complete
- Key outcome: Simplified synthesis section to only show outcome badge when available
