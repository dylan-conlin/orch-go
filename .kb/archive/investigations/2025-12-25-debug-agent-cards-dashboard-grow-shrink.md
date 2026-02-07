<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Agent cards in dashboard grew/shrank due to conditional rendering of bottom sections (live activity/synthesis) that only appeared when content existed.

**Evidence:** Code inspection of agent-card.svelte revealed `{#if agent.status === 'active' && agent.current_activity}` pattern that omitted the section entirely when no activity present; visual smoke test confirmed fix works - both active cards now have consistent height with placeholder text visible.

**Knowledge:** UI components with variable content should reserve space even when empty to prevent layout jitter in grid layouts; placeholder states ("Waiting for activity...", "No synthesis available") provide visual feedback while maintaining consistent dimensions.

**Next:** Close - fix implemented and verified via smoke test.

**Confidence:** High (95%) - Visual verification confirms fix works; edge cases (abandoned agents) also handled.

---

# Investigation: Agent Cards Dashboard Grow Shrink

**Question:** Why do agent cards in the dashboard grow/shrink in height when the live activity section has no content?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (95%)

---

## Findings

### Finding 1: Conditional rendering caused height variance

**Evidence:** In `agent-card.svelte` lines 265-277, the live activity section was wrapped in:
```svelte
{#if agent.status === 'active' && agent.current_activity}
```
This meant when `current_activity` was undefined/null, the entire section (~30px height) was not rendered.

**Source:** `web/src/lib/components/agent-card/agent-card.svelte:265-277`

**Significance:** This is the root cause - the conditional removed the DOM element entirely instead of rendering an empty placeholder, causing the card height to shrink.

---

### Finding 2: Completed agents had same issue

**Evidence:** Lines 281-298 had similar pattern:
```svelte
{#if agent.status === 'completed' && (agent.synthesis?.tldr || agent.synthesis?.outcome || agent.close_reason)}
```
Cards without synthesis data would be shorter than those with it.

**Source:** `web/src/lib/components/agent-card/agent-card.svelte:281-298`

**Significance:** The issue affected multiple agent states, not just active agents.

---

### Finding 3: Grid layout doesn't auto-equalize heights

**Evidence:** The dashboard uses CSS grid:
```svelte
<div class="grid gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
```
While grid can stretch items to fill row height, the cards were individually sized based on their content.

**Source:** `web/src/routes/+page.svelte:519`

**Significance:** The fix needed to be in the card component itself, not the grid container.

---

## Synthesis

**Key Insights:**

1. **Reserved space pattern** - UI components with optional content should always reserve space for that content to prevent layout shifts. This is a common UI/UX pattern.

2. **Placeholder states provide feedback** - Instead of just reserving empty space, placeholder text like "Waiting for activity..." provides visual feedback about the agent's state.

3. **All agent states need consideration** - The fix needed to handle active, completed, AND abandoned agents to ensure consistency across all card types.

**Answer to Investigation Question:**

Agent cards grew/shrank because the live activity section was conditionally rendered with `{#if agent.current_activity}` which completely removed the DOM element when no activity was present. The fix changes the condition to `{#if agent.status === 'active'}` (always render for active agents) with a nested conditional that shows placeholder text when no activity exists. This maintains consistent card height while providing visual feedback.

---

## Confidence Assessment

**Current Confidence:** High (95%)

**Why this level?**

Visual smoke test confirmed the fix works - both active agent cards in the dashboard now have identical heights with the placeholder "Waiting for activity..." visible on the card without current activity.

**What's certain:**

- ✅ Root cause identified: conditional rendering pattern
- ✅ Fix implemented: always-render pattern with placeholder
- ✅ Visual verification: screenshot shows consistent card heights
- ✅ Build passes: no compilation errors

**What's uncertain:**

- ⚠️ Edge cases with very long synthesis text (could still cause variance)
- ⚠️ Behavior with synthesis.outcome badge (adds slight height when present)

**What would increase confidence to Very High (98%+):**

- Test with variety of real agent data
- Verify behavior when live activity rapidly appears/disappears
- Check responsiveness across different screen sizes

---

## Implementation Recommendations

**Purpose:** Already implemented - documenting the approach used.

### Recommended Approach ⭐

**Reserved space with placeholder pattern** - Always render the bottom section container, display placeholder when no content

**Why this approach:**
- Maintains consistent DOM structure across all states
- Provides visual feedback to users about agent state
- Simple to implement with minimal code changes

**Trade-offs accepted:**
- Slightly more visual noise with placeholder text
- Minor increase in DOM elements when no activity

**Implementation sequence:**
1. Changed outer conditional from `status && content` to just `status`
2. Added nested conditional for content vs placeholder
3. Applied same pattern to completed and abandoned agents

---

## References

**Files Examined:**
- `web/src/lib/components/agent-card/agent-card.svelte` - Main agent card component
- `web/src/routes/+page.svelte` - Dashboard page with grid layout

**Commands Run:**
```bash
# Build verification
npm run build

# Visual verification  
snap window "Firefox"
```

---

## Investigation History

**2025-12-25 21:10:** Investigation started
- Initial question: Why do agent cards grow/shrink when live activity has no content?
- Context: Live activity feature recently added to bottom of agent cards

**2025-12-25 21:12:** Root cause identified
- Found conditional rendering pattern causing height variance

**2025-12-25 21:14:** Fix implemented
- Applied reserved space pattern with placeholders

**2025-12-25 21:16:** Investigation completed
- Final confidence: High (95%)
- Status: Complete
- Key outcome: Fixed by always rendering bottom section with placeholder when no content
