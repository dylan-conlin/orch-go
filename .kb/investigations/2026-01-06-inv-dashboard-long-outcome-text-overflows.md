<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Long outcome text in agent cards now truncates to short form (e.g., "success") with full text on hover tooltip.

**Evidence:** Build succeeds, function tests pass - "success (fix already implemented by prior agents)" correctly becomes "success" badge with tooltip.

**Knowledge:** Outcome field can contain parenthetical details; extracting first word before '(' provides clean display while preserving full info on hover.

**Next:** Close - fix implemented and verified.

---

# Investigation: Dashboard Long Outcome Text Overflows

**Question:** How to prevent long outcome text like 'success (fix already implemented by prior agents)' from overflowing agent card layout?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Agent (architect skill)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Outcome text stored with parenthetical details

**Evidence:** API returns outcomes like:
- `success`
- `success (bug already fixed by prior agents)`
- `success (could-not-reproduce)`
- `success (fix already implemented by prior agents)`

**Source:** `curl -k -s https://localhost:3348/api/agents | jq -r '.[] | select(.synthesis.outcome != null) | .synthesis.outcome' | sort -u`

**Significance:** The extra context in parentheses is valuable for understanding why an agent succeeded, but displaying it inline causes overflow.

---

### Finding 2: Badge component has no built-in truncation

**Evidence:** The Badge component at `web/src/lib/components/ui/badge/badge.svelte` is a simple div wrapper that passes through content without truncation.

**Source:** `web/src/lib/components/ui/badge/badge.svelte:15-17`

**Significance:** Truncation must be handled at the usage site (agent-card) rather than modifying the shared Badge component.

---

### Finding 3: Existing tooltip pattern works well for truncated content

**Evidence:** The agent card already uses `Tooltip.Root/Trigger/Content` pattern for other truncated fields like display title (line 307-318) and workspace ID (line 321-334).

**Source:** `web/src/lib/components/agent-card/agent-card.svelte:307-334`

**Significance:** Following the same pattern maintains UI consistency - truncated text with full content on hover is an established pattern in this codebase.

---

## Synthesis

**Key Insights:**

1. **Structured outcome text** - Outcomes follow pattern `{type}` or `{type} ({details})` where type is one of: success, partial, blocked, failed.

2. **Extractable short form** - Splitting on `(` gives clean short form for display while preserving full context in tooltip.

3. **Conditional tooltip** - Only wrap in tooltip when details exist, avoiding unnecessary tooltip overhead for simple outcomes like "success".

**Answer to Investigation Question:**

Solution 3 from the issue ("Show short outcome with details in tooltip") is the correct approach. Extract the core outcome type (success/partial/blocked/failed) for badge display, show full text on hover. This:
- Prevents overflow without cutting off information
- Follows existing UI patterns in the codebase
- Preserves full context for those who want it

---

## Structured Uncertainty

**What's tested:**

- ✅ Build succeeds with changes (verified: `bun run build` completes)
- ✅ Helper functions extract short outcome correctly (verified: node test script)
- ✅ No TypeScript errors introduced in agent-card.svelte (verified: svelte-check grep)

**What's untested:**

- ⚠️ Visual appearance in browser not directly verified (requires manual/Playwright test)
- ⚠️ Tooltip hover interaction not tested (requires browser)

**What would change this:**

- If outcome format changes to not use parentheses, the split would need adjustment
- If very long outcomes without parentheses appear (>20 chars), additional truncation would be needed

---

## Implementation Details

### Changes Made

**File:** `web/src/lib/components/agent-card/agent-card.svelte`

**Added helper functions (lines 205-220):**

```typescript
function getShortOutcome(outcome: string): string {
  // Extract just the first word before any parenthetical details
  // e.g., "success (fix already implemented by prior agents)" -> "success"
  return outcome.split(/\s*\(/)[0].trim();
}

function hasOutcomeDetails(outcome: string): boolean {
  return outcome.includes('(') || outcome.length > 20;
}
```

**Updated outcome badge display (lines 443-464):**

```svelte
{#if hasOutcomeDetails(agent.synthesis.outcome)}
  <Tooltip.Root>
    <Tooltip.Trigger>
      <Badge variant={getShortOutcome(agent.synthesis.outcome) === 'success' ? 'default' : 'secondary'} class="h-4 px-1 text-[10px]">
        {getShortOutcome(agent.synthesis.outcome)}
      </Badge>
    </Tooltip.Trigger>
    <Tooltip.Content class="max-w-xs">
      <p>{agent.synthesis.outcome}</p>
    </Tooltip.Content>
  </Tooltip.Root>
{:else}
  <Badge variant={agent.synthesis.outcome === 'success' ? 'default' : 'secondary'} class="h-4 px-1 text-[10px]">
    {agent.synthesis.outcome}
  </Badge>
{/if}
```

---

## References

**Files Examined:**
- `web/src/lib/components/agent-card/agent-card.svelte` - Main component to modify
- `web/src/lib/stores/agents.ts` - Agent and Synthesis type definitions
- `web/src/lib/components/ui/badge/badge.svelte` - Badge component structure

**Commands Run:**
```bash
# Build verification
cd web && bun run build

# Type check
cd web && bun run check

# API data check
curl -k -s https://localhost:3348/api/agents | jq -r '.[] | select(.synthesis.outcome != null) | .synthesis.outcome' | sort -u
```

---

## Investigation History

**2026-01-06 17:30:** Investigation started
- Initial question: How to fix long outcome text overflow in agent cards
- Context: Screenshot showed purple outcome text extending beyond card bounds

**2026-01-06 17:35:** Solution implemented
- Added getShortOutcome and hasOutcomeDetails helper functions
- Updated Badge display to use truncation with tooltip
- Build and typecheck pass

**2026-01-06 17:38:** Investigation completed
- Status: Complete
- Key outcome: Truncate to short form with tooltip for full text
