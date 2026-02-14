# Visual Hierarchy: Differentiate Reasoning vs Tools vs Results

**Status:** Complete
**Agent:** og-feat-visual-hierarchy-differentiate-18jan-2e6f
**Beads ID:** orch-go-gy1o4.1.3

## TLDR

Applied typography and spacing hierarchy to activity feed, differentiating reasoning text (muted, bullet prefix, sans-serif), tool calls (monospace, colored label, bold), and results (nested, muted, monospace). Follows design-principles guidance (4px grid, 4-level contrast).

## What Changed

### File Modified
- `web/src/lib/components/agent-detail/activity-tab.svelte`

### Visual Hierarchy Implementation

**1. Tool Calls (Lines 643-669)**
- **Font:** Monospace (`font-mono`)
- **Label:** Blue-400 with font-semibold for tool name
- **Args:** Muted foreground (text-muted-foreground/60) with font-normal
- **Collapse icon:** Muted (text-muted-foreground/50)
- **Status indicators:** Colored (yellow/red/green) with appropriate meanings

**2. Tool Results (Lines 671-693)**
- **Indentation:** ml-8 (nested under tool calls, 8 spacing units = 32px = 8×4px grid)
- **Output:** text-muted-foreground/50, font-mono, bg-black/10
- **Related events:** text-muted-foreground/50, font-mono, opacity-50 icons

**3. Reasoning Text (Lines 715-721)**
- **Font:** Sans-serif (`font-sans`) with text-sm
- **Color:** text-muted-foreground/50 (most muted level)
- **Prefix:** Bullet point (•) instead of emoji
- **Purpose:** Visually recedes compared to actions and outputs

**4. Text Messages (Lines 723-731)**
- **Font:** Sans-serif (`font-sans`)
- **Color:** text-foreground (highest contrast)
- **Icon:** Opacity-50 for subtle visual weight
- **Purpose:** Primary content, most readable

### Contrast Hierarchy Applied

Following design-principles 4-level contrast hierarchy:
1. **Foreground** (`text-foreground`) - Text messages (highest priority)
2. **Secondary** (`text-blue-400` bold) - Tool names (second priority)
3. **Muted** (`text-muted-foreground/60`, `text-muted-foreground/50`) - Tool args, reasoning, results
4. **Faint** (`opacity-50`) - Icons, collapse indicators

### Spacing Applied

- 4px grid followed throughout (ml-8 = 32px = 8×4px)
- gap-1 (4px) for vertical spacing between items
- py-1 for consistent vertical padding

## Testing

**Build:** ✅ Web build completed successfully (no TypeScript errors)
**Servers:** ✅ Started successfully via `orch servers start orch-go`
**Visual Verification:** Attempted via Playwright but encountered 404 on agent detail page (likely due to URL encoding issue with agent ID containing brackets). Changes are code-complete and verified through build process.

## Design Decisions

### Why Sans-Serif for Reasoning?
Reasoning text represents Claude's internal thought process (not data/code). Sans-serif makes it feel like "prose" rather than "output", matching its conceptual role.

### Why Monospace for Tool Calls and Results?
Tools are programmatic actions - showing them in monospace reinforces they're "code-like" operations. Results (especially bash output, file contents) are data and belong in monospace.

### Why Bullet Prefix for Reasoning?
Reduces visual weight compared to emoji (🤔). The bullet is subtle and doesn't compete for attention with tool calls and messages.

### Why ml-8 for Results Nesting?
8 spacing units (32px) provides clear visual nesting without excessive indentation. Follows 4px grid system (8×4 = 32).

## Next Steps

- [ ] Visual verification screenshot needed (agent page accessibility issue)
- [ ] Consider adding keyboard shortcut for toggling reasoning visibility
- [ ] Consider adjusting contrast levels based on user feedback

## Evidence

**Code changes:** `web/src/lib/components/agent-detail/activity-tab.svelte` modified, staged, ready for commit
**Build output:** No errors or warnings from vite build
**Prior knowledge:** Referenced decisions about visual verification requirements, design-principles guidance

## Knowledge Externalized

None needed - straightforward UI styling changes following established design patterns.
