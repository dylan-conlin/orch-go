# Design: Swarm Dashboard UI Iterations

**Status:** In Progress
**Created:** 2025-12-20
**Issue:** orch-go-xwh

## Problem Statement

The Swarm Dashboard needs UI/UX improvements to enhance usability:
1. No way to toggle between light/dark modes (CSS variables exist but no UI)
2. Agent cards have flat hierarchy - hard to scan key information quickly
3. No filtering/sorting - users can't find specific agents easily when swarm is large

### Success Criteria

- Users can toggle dark/light mode with persistent preference
- Agent cards have clear visual hierarchy with primary info prominent
- Users can filter by status and sort by date/name
- All changes verified with Playwright tests

## Approach

### 1. Dark Mode Toggle

**Location:** Header, next to connection status

**Implementation:**
- Create `ThemeToggle` component with sun/moon icons from lucide-svelte
- Store preference in localStorage (`theme` key)
- Apply `dark` class to document root
- Respect system preference as default (prefers-color-scheme)

**Files affected:**
- `src/routes/+layout.svelte` - Add toggle to header
- `src/lib/stores/theme.ts` - New store for theme state
- `src/lib/components/theme-toggle.svelte` - New toggle component

### 2. Agent Card Visual Hierarchy

**Current issues:**
- Status badge and skill badge have equal weight
- ID and beads_id compete for attention
- Duration tucked away at bottom

**Proposed hierarchy:**
1. **Primary:** Status (large indicator) + Duration (time-sensitive)
2. **Secondary:** Agent ID (monospace, prominent)
3. **Tertiary:** Skill badge, beads_id
4. **Context:** Synthesis card (already well-designed)

**Implementation:**
- Larger status indicator (colored bar or larger badge)
- Agent ID as card title
- Duration next to status (running time is critical info)
- Skill and beads_id as subtle secondary info

### 3. Filtering/Sorting Agents

**Filter options:**
- Status: All | Active | Completed | Abandoned
- Skill: Dropdown of unique skills in current swarm

**Sort options:**
- Newest first (default)
- Oldest first
- Alphabetical by ID

**Implementation:**
- Filter bar above agent grid
- Use derived stores for filtered/sorted view
- Persist filter state in URL params (optional, for sharing)

**Files affected:**
- `src/routes/+page.svelte` - Add filter UI and apply filtering
- `src/lib/stores/agents.ts` - Add derived store for filtered view

## Testing Strategy

**Playwright tests:**
1. `dark-mode.spec.ts` - Toggle works, preference persists
2. `agent-cards.spec.ts` - Cards render with correct hierarchy
3. `filtering.spec.ts` - Filters work correctly

**Setup required:**
- Install @playwright/test
- Configure playwright.config.ts for SvelteKit

## Security Considerations

- No security concerns - purely UI changes
- localStorage for theme is standard practice

## Performance Requirements

- Filtering should be instant (client-side derived stores)
- Theme toggle should not cause FOUC (flash of unstyled content)

## Rollout Plan

1. Implement dark mode toggle (self-contained)
2. Improve agent card hierarchy (visual only)
3. Add filtering/sorting (new functionality)
4. Add Playwright tests for all features
5. Verify with smoke test
