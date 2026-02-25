# Probe: Dashboard Web UI Framework, CSS, and Responsive Patterns

**Model:** dashboard-architecture
**Date:** 2026-02-25
**Status:** Complete

---

## Question

The dashboard-architecture model states the dashboard is "a Svelte 5 web UI served by `orch serve`" with "SSE connections" and "progressive disclosure." This probe tests the full technology stack, CSS approach, responsive breakpoints, layout structure, component architecture, and theming system — areas the model mentions only at a high level.

---

## What I Tested

Exhaustive file-system exploration and source code analysis of the `web/` directory:

```bash
# File tree
find web/src -type f | sort

# Config files read
cat web/package.json
cat web/tailwind.config.js
cat web/svelte.config.js
cat web/vite.config.ts
cat web/components.json
cat web/postcss.config.js

# Source files read
cat web/src/app.css
cat web/src/app.html
cat web/src/routes/+layout.svelte
cat web/src/routes/+page.svelte
cat web/src/routes/work-graph/+page.svelte
cat web/src/routes/knowledge-tree/+page.svelte
cat web/src/lib/components/agent-card/agent-card.svelte
cat web/src/lib/components/stats-bar/stats-bar.svelte
cat web/src/lib/components/collapsible-section/collapsible-section.svelte
cat web/src/lib/components/ui/badge/badge.svelte
cat web/src/lib/utils.ts

# Pattern searches
rg "sm:|md:|lg:|xl:|2xl:|@media|min-width|max-width" web/src
rg "grid-cols|flex-wrap|hidden\s+sm:" web/src
```

---

## What I Observed

### Framework & Build Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Framework | **Svelte 5** (runes: `$props()`, `$derived`, `$state`) | ^5.43.8 |
| Meta-framework | **SvelteKit** | ^2.48.5 |
| Build | **Vite** | ^6.0.0 |
| Package manager | **Bun** (bun.lock present) | — |
| Adapter | `@sveltejs/adapter-static` (SPA mode, `fallback: index.html`) | ^3.0.0 |
| Type checking | TypeScript ^5.9.3, svelte-check ^4.3.4 | — |
| Testing | Playwright ^1.57.0 | — |
| Icons | `@lucide/svelte` ^0.544.0 | — |

### CSS Approach

| Aspect | Detail |
|--------|--------|
| Primary styling | **Tailwind CSS v3** (utility-first, all inline classes) |
| Dark mode | `darkMode: ['class']` — toggled via `dark` class on `<html>` |
| Custom colors | HSL CSS variables (`--background`, `--foreground`, `--primary`, etc.) via shadcn-svelte convention |
| Swarm-specific colors | `swarm.active` (green), `swarm.completed` (blue), `swarm.abandoned` (red), `swarm.idle` (yellow) |
| UI component library | **shadcn-svelte** (registry: `tw3.shadcn-svelte.com/registry/default`, base color: slate) |
| Class merging | `cn()` utility using `clsx` + `tailwind-merge` |
| Variant system | `tailwind-variants` ^3.1.1 (used in shadcn components like Badge, Button) |
| PostCSS plugins | `tailwindcss` + `autoprefixer` |
| Global CSS | `app.css` — Tailwind directives, `:root` CSS variables, custom scrollbar styles |
| No CSS modules | Zero `.module.css` files; no scoped `<style>` blocks in components |
| Fonts | Inter (sans), JetBrains Mono (mono) |

### Theming System

- **28 theme files** in `web/src/lib/themes/` (JSON format, OpenCode theme schema)
- Themes include: catppuccin, dracula, gruvbox, nord, tokyonight, material, monokai, solarized, etc.
- Theme store (`$lib/stores/theme`) manages initialization and CSS variable application
- `app.html` starts with `class="dark"` hardcoded, theme store overrides on mount

### Responsive Breakpoints (Tailwind defaults)

| Breakpoint | Width | Usage Pattern |
|------------|-------|---------------|
| `sm:` | 640px | 2-col grids, show/hide text labels |
| `md:` | 768px | 3-col grids |
| `lg:` | 1024px | 4-col grids, 2-col event panels, side panel sizing |
| `xl:` | 1280px | 5-col agent grids, show extra info columns |
| `2xl:` | 1400px | Container max-width only |

**Primary responsive pattern (agent grids):**
```
grid gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5
```
This pattern repeats 8+ times across the dashboard page for all agent sections (Active, Needs Review, Recent, Archive, Active Only mode, and all collapsible sections in Historical mode).

**Secondary patterns:**
- `flex-wrap` on stats bar, filter bar, badge rows (graceful wrapping at any width)
- `hidden sm:inline` for text labels that collapse to icon-only on small screens
- Side panels: `w-full sm:w-[85vw] lg:w-[80vw] max-w-[1200px]` (full-screen on mobile, proportional on larger)
- Event panels: `grid gap-2 lg:grid-cols-2` (stacked on mobile, side-by-side on desktop)

### Layout Architecture

**3-page SPA:**

1. **`/` (Dashboard)** — Agent monitoring, operational/historical modes
2. **`/work-graph`** — Beads issue tree with keyboard navigation
3. **`/knowledge-tree`** — Knowledge base exploration with timeline toggle

**Layout structure (`+layout.svelte`):**
- Sticky header (h-10, z-50) with: logo, 3-page nav, usage display, connection status, theme toggle
- `container` class wrapper with `py-3` padding
- `container` configured: centered, 2rem padding, max 1400px at 2xl

**Page layout patterns:**
- Dashboard: `space-y-3` vertical stack (StatsBar → ReviewQueue → Coaching → Services → mode-dependent sections)
- Work Graph: `h-[calc(100vh-4rem)]` full-height flexbox with header, content, keyboard-shortcuts footer
- Knowledge Tree: `height: calc(100vh - 2.5rem)` full-height with header, content, footer

### Component Structure

**40+ custom components** organized in `$lib/components/`, each in its own directory with:
- `{name}.svelte` — component file
- `index.ts` — barrel export

**Component categories:**

| Category | Components | Pattern |
|----------|-----------|---------|
| **UI primitives** (shadcn) | Badge, Button, Card (6 sub-components), DropdownMenu (7), Tooltip (2) | `$lib/components/ui/` |
| **Agent display** | AgentCard, AgentDetailPanel (5 tabs: activity, investigation, screenshots, synthesis, tab-button) | Card-based with status indicators |
| **Section containers** | CollapsibleSection, ReviewQueueSection, ReadyQueueSection, UpNextSection, RecentWins, NeedsAttention, QuestionsSection, ServicesSection | Collapsible panels with count badges |
| **Work/Knowledge** | WorkGraphTree (+helpers), KnowledgeTree, Timeline (+SessionGroup), ArtifactFeed | Tree views with keyboard nav |
| **Data panels** | StatsBar, MarkdownContent, SynthesisCard, DeliverableChecklist, IssueSidePanel, CloseIssueModal | Feature-specific UI |
| **Controls** | ThemeToggle, ViewToggle, GroupByDropdown, LabelFilter, SettingsPanel, DaemonConfigPanel, CacheValidationBanner | Toolbar/filter controls |

**State management — 25 Svelte stores** in `$lib/stores/`:
agents, agentlog, attention, beads, cache-validation, coaching, config, context, daemon, daemonConfig, dashboard-mode, deliverables, focus, hotspot, kb-artifacts, kb-model-probes, knowledge-tree, pending-reviews, questions, servers, servicelog, services, theme, timeline, usage, verification, wip, work-graph

**SSE service** (`$lib/services/sse-connection.ts`) — shared SSE connection management

### 666px Constraint Validation

The model constraints mention "Dashboard must be fully usable at 666px width." At 666px:
- Agent grids: `sm:grid-cols-2` activates (640px < 666px), so 2 columns
- Stats bar: `flex-wrap` ensures content wraps gracefully
- Filter bar: `flex-wrap` wraps filter controls
- Side panels: `w-full` on mobile, `sm:w-[85vw]` at 640px = ~566px, which fits
- No horizontal scrolling detected in layout patterns

---

## Model Impact

- [x] **Confirms** invariant: "Svelte 5 web UI" — verified Svelte 5 with runes (`$props()`, `$derived.by()`, `$state`)
- [x] **Confirms** invariant: "Two-mode design is mutually exclusive" — `$dashboardMode === 'operational'` vs `'historical'` conditional rendering in `+page.svelte`
- [x] **Confirms** invariant: "SSE Events auto-connect, Agentlog is opt-in" — `connectSSE()` called on mount, agentlog via "Follow" button
- [x] **Confirms** invariant: "Progressive disclosure via collapsed panels" — CollapsibleSection pattern with localStorage persistence of expansion state
- [x] **Confirms** invariant: "Event panels max-h-64" — `max-h-64` on both Agent Lifecycle and SSE Stream panels
- [x] **Extends** model with: Full CSS/theming stack details — shadcn-svelte + Tailwind v3 + HSL CSS variables + 28 JSON themes + tailwind-variants
- [x] **Extends** model with: Responsive breakpoint map — 5-tier grid (1→2→3→4→5 cols) as primary pattern, flex-wrap as secondary, side panels use vw-based sizing
- [x] **Extends** model with: Component architecture inventory — 40+ components in 6 categories, 25+ stores, barrel-export pattern
- [x] **Extends** model with: SPA architecture — adapter-static with fallback, Vite dev server on port 5188 with API proxy to localhost:4096
- [x] **Extends** model with: State persistence pattern — localStorage used extensively (section collapse, group-by mode, view mode, expansion state, seen issues) for cross-session UI state

---

## Notes

- The codebase uses **Svelte 5 runes syntax** (`$props()`, `$derived`, `$state`) alongside **Svelte 4 legacy syntax** (`export let`, `$:` reactive statements, `$store` auto-subscriptions). The main page (`+page.svelte`) is predominantly Svelte 4 patterns, while newer components like `stats-bar.svelte` use `$props()`. This mixed syntax is expected during a Svelte 4→5 migration.
- The `container` class in Tailwind config only defines a 2xl breakpoint (1400px), using Tailwind defaults for sm/md/lg/xl. This means the container has no explicit max-width below 1400px — it fills available space with 2rem padding.
- No custom media queries in CSS — all responsive behavior is via Tailwind utility classes.
- The work-graph and knowledge-tree pages use full-viewport-height layouts (`calc(100vh - Xrem)`) unlike the dashboard page which scrolls naturally.
