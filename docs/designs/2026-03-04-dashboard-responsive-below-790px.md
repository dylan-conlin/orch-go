# Design: Dashboard Responsive Layout Below 790px

**Date:** 2026-03-04
**Beads:** orch-go-txau
**Status:** Draft
**Target:** Fully usable at 666px (half MacBook Pro screen), graceful down to 375px

---

## Problem

The dashboard has a hard minimum content width of ~790px. All viewports below this (768px, 640px, 375px) produce horizontal overflow. The root causes are:

1. **Container padding** — `padding: 2rem` (32px each side = 64px consumed)
2. **Stats bar** — Horizontal flex row with 7+ indicator groups using `gap-x-4`, no wrapping strategy
3. **Agent card grids** — `sm:grid-cols-2` kicks in at 640px but cards have implicit minimum width from badges/text
4. **Header** — Logo + 3 nav links + usage stats + connection indicator + theme toggle in single flex row
5. **Queue rows** — Review/ready queue items have 5+ inline elements (priority, title, type badge, labels, ID) with no wrap/collapse

---

## Breakpoint Strategy (per CLAUDE.md)

| Breakpoint | CSS Class | Structural Changes |
|------------|-----------|-------------------|
| < 640px | default | Single column everything. Header collapses. Stats bar stacks vertically. |
| 640px (sm:) | `sm:` | First structural shift: 2-col agent grids, stats bar wraps to 2 rows, header shows nav inline |
| 768px (md:) | `md:` | Minor tweaks only: spacing adjustments, font sizes. **No layout structure changes** |
| 1024px (lg:) | `lg:` | Full desktop: 4-col agent grids, event panels side-by-side, all stats bar items inline |

---

## Changes by Component

### 1. Container (tailwind.config.js + layout)

**Problem:** `padding: '2rem'` wastes 64px at narrow widths.

**Change:**
```js
// tailwind.config.js
container: {
    center: true,
    padding: {
        DEFAULT: '0.75rem',  // 12px on mobile (was 32px)
        sm: '1rem',          // 16px at 640px
        lg: '2rem'           // 32px at 1024px+ (original)
    },
    screens: {
        '2xl': '1400px'
    }
}
```

**Savings:** 40px recovered at narrow viewports (64px → 24px)

### 2. Header (+layout.svelte)

**Problem:** All header items in single flex row. Nav links + usage stats overflow at 640px.

**Changes:**
- Below `sm:` (< 640px): Hide nav link text, show compact icon-only nav. Hide usage reset timers and account name. Collapse connection status to dot-only.
- At `sm:` (640px+): Show full nav links, usage details.

```svelte
<!-- Header structure change -->
<div class="container flex h-10 items-center gap-2">
    <a href="/" class="flex items-center gap-1.5 flex-shrink-0">
        <span class="text-base">🐝</span>
        <span class="text-sm font-semibold hidden sm:inline">Swarm</span>
    </a>
    <nav class="flex items-center gap-0.5 sm:gap-1 min-w-0">
        <!-- Nav links: abbreviated below sm: -->
        <a href="/" class="px-1.5 sm:px-2 py-1 text-xs ...">
            <span class="sm:hidden">Dash</span>
            <span class="hidden sm:inline">Dashboard</span>
        </a>
        <a href="/work-graph" class="px-1.5 sm:px-2 py-1 text-xs ...">
            <span class="sm:hidden">Work</span>
            <span class="hidden sm:inline">Work Graph</span>
        </a>
        <a href="/knowledge-tree" class="px-1.5 sm:px-2 py-1 text-xs ...">
            <span class="sm:hidden">KB</span>
            <span class="hidden sm:inline">Knowledge Tree</span>
        </a>
    </nav>
    <div class="flex flex-1 items-center justify-end gap-1.5 sm:gap-3 min-w-0">
        <!-- Usage: simplified below sm: -->
        {#if $usage && !$usage.error}
            <span class="inline-flex items-center gap-1 sm:gap-2 text-xs">
                <span class="font-medium ...">{formatPercent($usage.five_hour_percent)}</span>
                <span class="text-muted-foreground hidden sm:inline">|</span>
                <span class="font-medium hidden sm:inline ...">{formatPercent($usage.weekly_percent)}</span>
                <!-- Reset timers and account: hidden below sm: -->
                ...
            </span>
        {/if}
        <!-- Connection status: dot only below sm: -->
        <span class="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
            <span class={`h-1.5 w-1.5 rounded-full ${statusColor}`}></span>
            <span class="hidden sm:inline">{$connectionStatus}</span>
        </span>
        <ThemeToggle />
    </div>
</div>
```

### 3. Stats Bar (stats-bar.svelte)

**Problem:** 7+ indicator groups in single `flex` row with `gap-x-4`. Mode toggle + time filter + follow button + errors + active + beads + review + verification + daemon + settings + connect = too many items for narrow viewport.

**Changes:**
- Structure as two rows below `sm:`: controls row (mode toggle, time, follow, settings, connect) and indicators row (errors, active, beads, review, etc.)
- Below `sm:`, secondary indicators wrap naturally with reduced gap.
- Hide text labels on indicators below `sm:` (icon + number only).

```svelte
<div class="rounded-lg border bg-card px-3 sm:px-4 py-2" data-testid="stats-bar">
    <!-- Row 1: Controls (always visible) -->
    <div class="flex flex-wrap items-center gap-x-2 sm:gap-x-4 gap-y-1.5">
        <!-- Mode toggle (compact at narrow) -->
        <div class="flex items-center gap-0.5 sm:gap-1 rounded-md bg-muted p-0.5">
            <button class="px-1.5 sm:px-2 py-1 rounded text-xs ...">⚡ Ops</button>
            <button class="px-1.5 sm:px-2 py-1 rounded text-xs ...">📦 Hist</button>
        </div>

        <!-- Time filter -->
        ...

        <!-- Follow toggle -->
        ...

        <!-- Indicators (wrap naturally) -->
        <div class="flex flex-wrap items-center gap-x-2 sm:gap-x-4 gap-y-1">
            <!-- Each indicator: hide text label below sm: -->
            <span class="inline-flex items-center gap-1 sm:gap-2">
                <span class="text-base sm:text-lg">❌</span>
                <span class="text-lg sm:text-xl font-bold">{$errorEvents.length}</span>
                <span class="text-xs text-muted-foreground hidden sm:inline">errors</span>
            </span>
            <!-- Same pattern for active, beads, review, daemon -->
            ...
        </div>

        <!-- Settings + Connect (pushed right) -->
        <div class="ml-auto flex items-center gap-1">...</div>
    </div>
</div>
```

**Key sizing changes:**
- Emoji indicators: `text-base sm:text-lg` (16px → 18px)
- Numbers: `text-lg sm:text-xl` (18px → 20px)
- Text labels: `hidden sm:inline`
- Gap: `gap-x-2 sm:gap-x-4`

### 4. Agent Card Grids (+page.svelte, needs-attention.svelte, services-section.svelte)

**Problem:** Grids use `sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5`. This is structurally correct per breakpoint rules, but `md:grid-cols-3` at 768px forces 3 columns when there isn't room for 3 cards with comfortable width.

**Changes:**
- Remove `md:grid-cols-3` (768px should not do layout structure changes per CLAUDE.md rules)
- Use: `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5`
- This means 1 col → 2 col at 640px → 3 col at 1024px → 4 col at 1280px → 5 col at 1536px

**Apply to ALL grid instances:**
- `+page.svelte`: Active agents, needs review, historical mode sections (6 instances)
- `needs-attention.svelte`: Dead agents, awaiting cleanup, at-risk, stalled (4 instances)
- `services-section.svelte`: Service cards (1 instance, currently `sm:grid-cols-2 lg:grid-cols-3` — this is already correct)

**Find-and-replace pattern:**
```
OLD: grid gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5
NEW: grid gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5
```

### 5. Queue Row Items (review-queue-section.svelte, ready-queue-section.svelte)

**Problem:** Each row has priority + title + type badge + labels + issue ID, all in a single flex row.

**Changes:**
- Below `sm:`, hide labels and show issue ID inline after title.
- Type badge remains (small).
- Issue ID: `hidden sm:inline` (or show truncated on next line).

```svelte
<!-- Ready queue item -->
<div class="flex flex-wrap sm:flex-nowrap items-center gap-1 sm:gap-2 rounded px-2 py-1.5 text-sm hover:bg-accent/50">
    <span class="flex-shrink-0 text-xs font-medium ...">{getPriorityClass(issue.priority)}</span>
    <span class="flex-1 truncate min-w-0" title={issue.title}>{issue.title}</span>
    <Badge variant="outline" class="h-5 px-1.5 text-xs flex-shrink-0">{issue.issue_type}</Badge>
    <!-- Labels: hidden below sm: -->
    {#if issue.labels && issue.labels.length > 0}
        <span class="hidden sm:flex items-center gap-1">
            ...labels...
        </span>
    {/if}
    <span class="text-xs text-muted-foreground flex-shrink-0 font-mono hidden sm:inline">{issue.id}</span>
</div>
```

### 6. Event Panels (bottom of +page.svelte, historical mode)

**Problem:** `grid gap-2 lg:grid-cols-2` — already correct (stacks below 1024px). No changes needed.

### 7. Coaching Section, Services Section Headers, Collapsible Sections

These use `flex items-center gap-2` patterns which naturally wrap. The main fix is reducing horizontal padding (`px-3` → `px-2 sm:px-3`) to recover a few pixels. Minor, non-structural.

---

## Summary of Files to Modify

| File | Change Type | Complexity |
|------|-------------|------------|
| `web/tailwind.config.js` | Container padding responsive | Low |
| `web/src/routes/+layout.svelte` | Header responsive collapse | Medium |
| `web/src/lib/components/stats-bar/stats-bar.svelte` | Stats bar wrap/hide labels | Medium |
| `web/src/routes/+page.svelte` | Grid column breakpoints (6 instances) | Low |
| `web/src/lib/components/needs-attention/needs-attention.svelte` | Grid column breakpoints (4 instances) | Low |
| `web/src/lib/components/review-queue-section/review-queue-section.svelte` | Queue row responsive | Low |
| `web/src/lib/components/ready-queue-section/ready-queue-section.svelte` | Queue row responsive | Low |

---

## Testing Strategy

1. **Resize browser** to 666px, 640px, 375px widths
2. **Verify no horizontal scrollbar** at any width down to 375px
3. **Verify all critical information visible** at 666px without scrolling
4. **Verify 1024px+** layout is unchanged from current
5. **Test both themes** (light and dark)
6. **Test both modes** (operational and historical)

---

## Non-Goals (Deferred)

- Touch target sizing (44px minimum) — separate issue, affects all pages
- WCAG contrast fixes — separate issue (orch-go audit findings #6)
- Nested interactive elements — separate issue
- Header hamburger menu — not needed if abbreviated nav links fit at 375px

---

## Risk Assessment

**Low risk.** All changes are CSS/layout only. No data flow, state management, or API changes. No new components. Reversible via git.

**Potential concern:** Abbreviated nav link text ("Dash", "Work", "KB") at narrow widths may not be obvious. But this is the standard responsive pattern for dashboards, and the icons + context make it clear.
