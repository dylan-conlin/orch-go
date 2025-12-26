# Design: How to Better Surface Ready Queue Items Nearing Front of Queue

**Status:** Complete
**Phase:** Complete
**Created:** 2025-12-26

## TLDR

Design how to highlight high-priority ready queue items in the dashboard without requiring users to expand the full queue. Current state: ready queue expands inline from stats bar (jarring UX). Issue orch-go-afsz will move it to dedicated section.

## Design Question

How should high-priority ready queue items be surfaced in the dashboard to provide at-a-glance visibility into upcoming work without requiring expand/collapse interaction?

## Problem Framing

### Current State
- Stats bar shows `📋 N ready` with click-to-expand chevron
- Clicking expands full ready queue inline below stats bar
- Jarring UX: large list appears/disappears, pushes content down
- Users must consciously click to see what's next
- No passive visibility into priority items

### Success Criteria
1. High-priority items (P0, P1) visible at a glance without interaction
2. Non-jarring UX that doesn't disrupt workflow
3. Clear indication of queue depth without overwhelming detail
4. Integration with existing dashboard patterns (stats bar, Pending Reviews section)
5. Focus alignment visibility (items aligned with `orch focus` highlighted)

### Constraints
- **Technical:** API already exists (`/api/beads/ready` returns issues with priority, labels, age)
- **UX:** Dashboard is primarily for agent monitoring; ready queue is secondary
- **Space:** Stats bar is already dense with indicators
- **Consistency:** Should match patterns from Pending Reviews section

### Scope
- **IN:** Surfacing ready queue visibility, priority highlighting, focus alignment
- **OUT:** Full queue management (that's beads-ui's job), issue creation, status updates

## Exploration

### Priority Signals Available

From `/api/beads/ready` response:
- `priority`: 0-4 (P0 = critical, P4 = low)
- `labels`: includes `skill:X`, `triage:ready`, focus labels
- `created_at`: age calculation possible
- `issue_type`: task, bug, feature, etc.

Additional signals:
- **Focus alignment:** Match issue labels/content against current `orch focus` goal
- **Blocking count:** Number of issues blocked by this one (requires dependency data)
- **Age:** Time in ready queue (staleness indicator)

### Approach 1: "Up Next" Mini-Section in Stats Bar

**Mechanism:**
- Add compact inline display of top 1-3 priority items directly in stats bar
- Show truncated title + P-level badge
- Clicking individual item could open beads-ui or show detail

**Pros:**
- Always visible, no interaction required
- Minimal space footprint
- Clear call-to-action

**Cons:**
- Stats bar already crowded (errors, focus, servers, beads, daemon)
- May not scale if many P0/P1 items
- Competes for attention with other indicators

**Complexity:** Low - just add more elements to stats bar

### Approach 2: Priority Badges on Stats Bar Indicator

**Mechanism:**
- Enhance existing `📋 N ready` indicator with priority breakdown
- Show as `📋 5 ready (2 P0, 1 P1)` or colored badges
- Full queue still expands on click

**Pros:**
- Minimal visual change
- Priority information surfaced without new UI
- Non-intrusive

**Cons:**
- Still requires click to see actual items
- May get too wide with many priority levels
- Doesn't show specific item titles

**Complexity:** Very low - modify existing badge text

### Approach 3: Dedicated "Up Next" Section (Parallel to Pending Reviews)

**Mechanism:**
- New collapsible section below stats bar (like Pending Reviews)
- Shows top N priority items (configurable, default 3-5)
- Expanded by default if P0/P1 items exist, collapsed otherwise
- Each item shows: priority badge, title (truncated), labels, age indicator
- Auto-updates via polling (same 60s interval as other stores)

**Pros:**
- Clean separation from stats bar
- Consistent with Pending Reviews pattern
- Room for rich item display
- Auto-expand on high priority respects urgency

**Cons:**
- Takes vertical space
- Another section to manage collapse state
- May duplicate beads-ui functionality

**Complexity:** Medium - new component, state management, auto-expand logic

### Approach 4: Notification/Toast for New P0/P1 Items

**Mechanism:**
- When new P0/P1 item appears in ready queue, show desktop notification or in-app toast
- Notification includes title and priority
- Clicking notification expands queue or opens item

**Pros:**
- Proactive surfacing without constant UI presence
- Works even when dashboard not focused (desktop notification)
- Doesn't consume screen space

**Cons:**
- Can be disruptive/annoying
- Easy to miss if dismissed
- Requires notification permissions
- Not suitable for passive monitoring

**Complexity:** Medium - SSE integration for real-time updates, notification API

## Synthesis

### Recommendation: Approach 3 - Dedicated "Up Next" Section

**Why this approach:**

1. **Consistent with existing patterns:** The Pending Reviews section already establishes the pattern of a collapsible section with actionable items. Users are familiar with this interaction.

2. **Clean information hierarchy:** Stats bar → summary metrics. Sections → actionable details. Ready queue items are actionable, so they belong in a section.

3. **Auto-expand logic addresses urgency:** When P0/P1 items exist, section auto-expands. This surfaces urgent items proactively without constant notification noise.

4. **Scalable display:** Can show 3-5 items comfortably with priority, title, labels, age. Doesn't crowd stats bar.

5. **Focus alignment integration:** Can highlight items matching current `orch focus` with visual indicator (star or highlight).

### Trade-offs Accepted

- **Vertical space:** Section takes ~150px when expanded. Acceptable given importance of queue visibility.
- **Duplication with beads-ui:** Dashboard shows queue preview; beads-ui is full management. Clear separation of concerns.
- **State management:** One more collapsible section to persist. Low cost.

### When This Would Change

- If stats bar becomes less crowded (e.g., indicators consolidated), Approach 1 might work better
- If desktop notifications become standard practice, Approach 4 could complement this
- If ready queue rarely has items, simpler Approach 2 might suffice

### Implementation Specification

**Component: `UpNextSection.svelte`**

```svelte
<script lang="ts">
  import { readyIssues } from '$lib/stores/beads';
  import { focus } from '$lib/stores/focus';
  
  export let maxItems = 5;
  export let expanded: boolean;
  
  // Filter to top priority items
  $: priorityItems = ($readyIssues?.issues ?? [])
    .sort((a, b) => a.priority - b.priority)
    .slice(0, maxItems);
  
  // Auto-expand if P0 or P1 items exist
  $: hasUrgent = priorityItems.some(i => i.priority <= 1);
  $: if (hasUrgent && !expanded) expanded = true;
  
  // Focus alignment check
  $: focusGoal = $focus?.goal?.toLowerCase() ?? '';
  function isFocusAligned(issue: ReadyIssue): boolean {
    return issue.title.toLowerCase().includes(focusGoal) ||
           (issue.labels ?? []).some(l => focusGoal.includes(l));
  }
</script>
```

**Display per item:**
- Priority badge (P0-P4 with color coding)
- Title (truncated to ~60 chars)
- Age indicator (e.g., "2h", "3d")
- Focus star (⭐) if aligned with current focus
- Skill label if present

**Collapse behavior:**
- Collapsed by default
- Auto-expands if P0 or P1 items in queue
- User can manually collapse/expand
- Persist state in localStorage (like other sections)

**Location:** Between stats bar and Pending Reviews section

### File Targets

| File | Action | Description |
|------|--------|-------------|
| `web/src/lib/components/up-next-section/up-next-section.svelte` | Create | New component |
| `web/src/lib/components/up-next-section/index.ts` | Create | Export |
| `web/src/routes/+page.svelte` | Modify | Add UpNextSection after stats bar |
| `web/src/lib/stores/beads.ts` | No change | Already has `readyIssues` store |

### Acceptance Criteria

- [ ] UpNextSection shows top 5 priority items from ready queue
- [ ] Items sorted by priority (P0 first)
- [ ] Each item displays: priority badge, title (truncated), age, labels
- [ ] Focus-aligned items have ⭐ indicator
- [ ] Section auto-expands when P0/P1 items exist
- [ ] Section collapses/expands via header click
- [ ] Collapse state persists in localStorage
- [ ] Section updates every 60s (via existing readyIssues.fetch interval)

### Out of Scope

- Full queue management (filtering, sorting)
- Issue creation from dashboard
- Status updates from dashboard
- Desktop notifications (future enhancement)

## Unexplored Questions

1. Should clicking an item open beads-ui or show inline detail?
2. Should there be a "View All" link that opens beads-ui?
3. How to handle focus alignment when no focus is set?
4. Should age calculation show absolute time or relative (e.g., "stale > 3d")?

## References

- Current implementation: `web/src/routes/+page.svelte:487-550`
- Pending Reviews pattern: `web/src/lib/components/pending-reviews-section/`
- Ready issues store: `web/src/lib/stores/beads.ts`
- API endpoint: `cmd/orch/serve.go:1391-1445` (handleBeadsReady)
