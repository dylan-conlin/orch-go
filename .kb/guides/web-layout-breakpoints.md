# Web Layout Breakpoints Guide

**Status:** Active
**Created:** 2026-02-25
**Applies to:** All web projects (SvelteKit, React, etc.)

## Problem

MacBook Pro half-screen development creates viewports of ~750-870px CSS width:

| Model | Scaled Resolution | Half-Screen Width |
|-------|-------------------|-------------------|
| 14" MBP | 1512 × 982 px | ~756 px |
| 16" MBP | 1728 × 1117 px | ~864 px |

Most CSS frameworks (Tailwind, Bootstrap) use 768px as a structural breakpoint. At half-screen on a 14" MBP, you're **below** 768px — triggering mobile layout while trying to develop desktop features.

## Breakpoint Strategy

| Range | Target | Tailwind | Role |
|-------|--------|----------|------|
| < 640px | Phones | default (mobile-first) | Single column, stacked nav, hamburger menu |
| 640–1023px | Half-screen + tablets | `sm:` | Compact but functional — 2-col grids, inline nav, visible sidebar |
| 1024px+ | Full desktop | `lg:` | Multi-column grids, expanded sidebar, wider gutters |

### What each breakpoint does

**`sm:` (640px) — Structural layout shifts happen here:**
- Grid: `grid-cols-1 → sm:grid-cols-2`
- Sidebar: `hidden → sm:block`
- Nav: hamburger → `sm:flex` inline
- Visibility: `hidden → sm:inline` for labels

**`md:` (768px) — Minor tweaks only, never structure:**
- Spacing: `px-3 → md:px-4`
- Font size: `text-sm → md:text-base`
- Column width: `w-40 → md:w-48`
- **Never:** `hidden md:block`, `md:grid-cols-N`, `md:flex-row`

**`lg:` (1024px) — Desktop expansion:**
- Grid: `sm:grid-cols-2 → lg:grid-cols-4`
- Sidebar: `sm:w-48 → lg:w-64`
- Layout: side-by-side panels, expanded content areas

**`xl:` (1280px) — Wide desktop extras:**
- Grid: `lg:grid-cols-4 → xl:grid-cols-5`
- Extra info columns, wider margins

## Common Patterns

### Sidebar Layout
```html
<div class="flex flex-col sm:flex-row">
  <aside class="hidden sm:block sm:w-48 lg:w-64">...</aside>
  <main class="flex-1 min-w-0">...</main>
</div>
```

### Responsive Grid
```html
<!-- Stats cards -->
<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">

<!-- Content cards -->
<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
```

### Navigation
```html
<!-- Inline nav from 640px, hamburger below -->
<nav class="hidden sm:flex gap-4">...</nav>
<button class="sm:hidden">☰</button>
```

### Content Padding
```html
<!-- Gradual padding increase (not structural) -->
<div class="px-3 md:px-4 lg:px-6">
```

## Container Queries (Future Direction)

For reusable components, container queries eliminate viewport dependency:

```css
.card-container {
  container-type: inline-size;
}

@container (min-width: 400px) {
  .card { flex-direction: row; }
}
```

Use when: component appears in multiple contexts (sidebar vs main content, modal vs page).

## Migration Checklist

When updating an existing project:

1. **Find structural `md:` usage:** Search for `md:grid-cols`, `md:flex-row`, `hidden md:block`, `md:flex`
2. **Shift to `sm:`:** Replace structural shifts from `md:` → `sm:`
3. **Keep `md:` for tweaks:** Spacing (`md:px-6`), font sizes (`md:text-base`), minor width changes
4. **Test at 756px:** DevTools responsive mode at 756px width — should show compact but functional layout
5. **Test at 600px:** Should still be usable (phone landscape)
6. **Test at 375px:** Mobile layout, single column

## Decision Record

Chose 640px over 768px because:
- 14" MBP half-screen = ~756px, which is below 768px
- 640px captures phone landscape as the upper bound of "mobile"
- Tailwind's `sm:` already sits at 640px — no custom config needed
- Standard tablet breakpoint (768px) becomes a non-structural tweaking point
