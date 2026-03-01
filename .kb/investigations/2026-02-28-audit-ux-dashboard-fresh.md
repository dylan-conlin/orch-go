# Investigation: UX Audit — Dashboard Page (Fresh)

**TLDR:** Dashboard has 15 findings across 6 dimensions (0 Blocker, 5 Major, 7 Minor, 3 Cosmetic). The most impactful issues are horizontal overflow below 790px breaking all narrow viewports, WCAG AA contrast failures on 59 elements using muted-foreground color (#8a8a8a = 3.45:1 ratio), and nested interactive elements causing screen reader confusion. axe-core: 3 violations (1 critical, 2 serious), 16 passes.

**Status:** Complete
**Date:** 2026-02-28
**Beads:** orch-go-d5g6
**Mode:** full
**Target:** http://localhost:5188/
**Viewports:** 1280, 1024, 768, 640, 375
**Prior audit:** `2026-02-28-audit-ux-dashboard.md` (orch-go-mzqk, dark theme)

---

## Page Metadata

| Field | Value |
|-------|-------|
| URL | http://localhost:5188/ |
| Page title | Swarm Dashboard |
| Auth method | None (local dev) |
| Auth state | N/A |
| Data state | Loaded (SSE connected, 2 services running, 0 active agents) |
| Viewports tested | 1280, 1024, 768, 640, 375 |
| Audit date | 2026-02-28 |
| Audit mode | full |
| Dashboard mode | Operational |
| Theme | Light |

---

## Baseline Metrics

| Metric | Value | Prior Audit (mzqk) | Delta |
|--------|-------|---------------------|-------|
| Total findings | 15 | 14 | +1 |
| Blocker | 0 | 0 | — |
| Major | 5 | 7 | -2 |
| Minor | 7 | 5 | +2 |
| Cosmetic | 3 | 2 | +1 |
| axe-core violations | 3 (68 nodes) | 3 (51 nodes) | +17 nodes |
| axe-core critical | 1 | 0 | +1 |
| axe-core serious | 2 | 2 | — |
| axe-core passes | 16 | 18 | -2 |
| axe-core incomplete | 1 (11 nodes) | 1 | — |
| Console errors | 26 | 22 | +4 |

---

## Findings by Dimension

### Visual Consistency: 2 findings

#### 1. Body background is white instead of design token #FAFBFC
**Severity:** Cosmetic
**Design Token:** `--background`
**Expected:** `#FAFBFC` / `rgb(250, 251, 252)`
**Actual:** `rgb(255, 255, 255)` (pure white)
**Elements Affected:** `<body>` element
**Impact:** Minor visual deviation from the Toolshed design direction. Cards at `rgb(250, 250, 250)` end up darker than the body, inverting the expected surface hierarchy (body should be slightly off-white with cards being white).
**Recommendation:** Set CSS variable `--background` to the correct #FAFBFC value.

#### 2. Foreground color uses neutral gray instead of Slate 900
**Severity:** Cosmetic
**Design Token:** `--foreground`
**Expected:** `#0F172A` / `rgb(15, 23, 42)` (Slate 900 — cool blue-black)
**Actual:** `rgb(26, 26, 26)` (neutral dark gray)
**Elements Affected:** Body text color (all foreground text)
**Impact:** Subtle warmth difference. The neutral gray lacks the blue undertone of Slate 900 that the Toolshed design specifies. Not functionally impactful.
**Recommendation:** Update `--foreground` CSS variable to use Slate 900 values.

### Responsive: 3 findings

#### 3. Horizontal overflow below 790px (all narrow viewports broken)
**Severity:** Major
**Viewport(s):** 768px, 640px, 375px
**Screenshot:** `baseline-768.png`, `baseline-640.png`, `baseline-375.png`
**Evidence:** At 375px, `body.scrollWidth` = 790px vs viewport 375px (over 2x). At 768px, 790px vs 768px (22px overflow, triggers horizontal scrollbar). The content has a minimum width of ~790px and does not reflow below that.
**Impact:** Users on tablets and mobile devices get a horizontal scrollbar and must pan left-right. The stats bar, agent sections, and needs-attention cards overflow. The page is essentially unusable below 790px width.
**Recommendation:** This is an architectural responsive issue — the `container` class, stats bar flex layout, and content sections all need responsive redesign. **Recommend `architect` follow-up** rather than piecemeal fixes.

#### 4. Touch targets below 44px on all viewports
**Severity:** Major
**Viewport(s):** All (375px most critical)
**Evidence:** Nav links: 24px tall. Theme toggle: 36x36px. Mode buttons (Ops, History): 24px tall. Stats bar buttons: 24px tall. Even at desktop, most interactive elements are 24px — well below WCAG 2.5.5's 44px minimum.
**Impact:** On mobile (375px), tapping interactive elements is unreliable. At 24px tall, targets are roughly half the recommended minimum.
**Recommendation:** Increase touch target sizes to minimum 44x44px using padding: `min-h-[44px]` on interactive elements. For nav links, increase padding rather than changing visual size.

#### 5. Header content overflows at narrow viewports
**Severity:** Minor
**Viewport(s):** 375px, 640px
**Screenshot:** `header-mobile.png`
**Evidence:** At 375px, header `scrollWidth` = 459px vs 375px viewport. Nav links ("Dashboard", "Work Graph", "Knowledge Tree") plus logo plus status indicator don't fit. Connection status text is partially clipped.
**Impact:** The sticky header — the primary navigation — overflows on mobile. Navigation links require horizontal scrolling to access.
**Recommendation:** At sm (640px) breakpoint, collapse nav links into a hamburger menu or reduce to icon-only nav. This is the expected structural shift per CLAUDE.md breakpoint rules.

### Accessibility: 5 findings

#### 6. WCAG AA color contrast failure on muted-foreground (59 elements)
**Severity:** Major
**WCAG:** 1.4.3 Contrast (Minimum)
**Source:** axe-core (serious) + manual contrast check
**Evidence:** `text-muted-foreground` color `#8a8a8a` on white background = 3.45:1 ratio. WCAG AA requires 4.5:1 for normal text (< 18.66px / 14px bold). Affects 59 elements: nav links, badge text, helper text, timestamps, section subtitles, empty state messages.
**Impact:** Users with low vision cannot reliably read muted text. Nav links — the primary navigation — use this insufficient color.
**Recommendation:** Darken `--muted-foreground` from `#8a8a8a` (3.45:1) to at least Slate 500 (`#64748B` = 4.6:1 on white) or `#636363` (~5:1 on white).

#### 7. Critical: Select element has no accessible name
**Severity:** Major
**WCAG:** 4.1.2 Name, Role, Value
**Source:** axe-core (critical)
**Evidence:** `<select class="h-6 rounded border..." data-testid="time-filter">` has no `<label>`, no `aria-label`, and no `aria-labelledby`. It's the "Since:" time filter in the stats bar. The adjacent "Since:" text is visual only, not programmatically associated.
**Impact:** Screen reader users cannot determine the purpose of this dropdown. It's announced as just "select" with no context.
**Recommendation:** Add `aria-label="Filter time range"` to the select element, or wrap "Since:" in a `<label for="time-filter-id">`.

#### 8. Nested interactive elements (8 instances)
**Severity:** Major
**WCAG:** 4.1.2 Name, Role, Value
**Source:** axe-core (serious)
**Evidence:** Agent cards and service cards contain `<button>` elements nested inside other `<button>` elements (tooltip triggers wrapping action buttons). Example: `<button data-tooltip-trigger><button class="text-...">`.
**Impact:** Screen readers may not announce nested buttons correctly, and keyboard focus order becomes unpredictable. Some assistive technologies skip inner interactive elements entirely.
**Recommendation:** Restructure tooltip triggers to use `<span>` or `<div>` with appropriate roles instead of nesting `<button>` inside `<button>`.

#### 9. No h1 element on the page
**Severity:** Minor
**WCAG:** 1.3.1 Info and Relationships
**Source:** Structural review
**Evidence:** `document.querySelectorAll('h1').length === 0`. The header shows "Swarm" as `<span class="text-sm font-semibold">`, not an `<h1>`. Section headers use `<span>` and `<h2>`/`<h3>` but there's no h1 to anchor the hierarchy.
**Impact:** Screen reader users navigating by headings have no page-level heading.
**Recommendation:** Add a visually-hidden `<h1 class="sr-only">Swarm Dashboard</h1>` or make "Swarm" an h1.

#### 10. Nav element has no aria-label
**Severity:** Minor
**WCAG:** 1.3.1 Info and Relationships
**Source:** Structural review
**Evidence:** `<nav class="flex items-center gap-1">` has no `aria-label` or `aria-labelledby`. When multiple nav regions exist on a page, screen readers can't distinguish them.
**Impact:** Users hear "navigation" without context about which navigation region they're in.
**Recommendation:** Add `aria-label="Main navigation"` to the `<nav>` element.

### Navigation: 2 findings

#### 11. No active state on current page nav link
**Severity:** Major
**Viewport(s):** All
**Screenshot:** `header-desktop.png`
**Evidence:** All three nav links have identical styling: `color: rgb(138, 138, 138)`, `font-weight: 500`. No `aria-current="page"` on any link. No active/selected class. Current path is `/` which matches Dashboard link's `href="/"`.
**Impact:** Users cannot tell which page they are currently viewing. This is a fundamental orientation problem.
**Recommendation:** Add `aria-current="page"` to the active nav link. Apply visual distinction: `text-foreground` (dark) + `font-weight: 600` for active, keep `text-muted-foreground` for inactive. Consider an underline indicator.

#### 12. Page title doesn't vary with mode or route
**Severity:** Cosmetic
**Viewport(s):** All
**Evidence:** `document.title` = "Swarm Dashboard" regardless of dashboard mode (Ops vs History).
**Impact:** Tab management and bookmarking aren't differentiated. Low priority until modes become separate routes.
**Recommendation:** Consider updating title to include mode if Ops/History become distinct routes.

### Data Presentation: 1 finding

#### 13. Service uptime shows "0s"
**Severity:** Minor
**Viewport(s):** All
**Screenshot:** `baseline-1280.png`
**Evidence:** Both service cards (api, web) display "0s" for uptime in the top-right corner. Either uptime data isn't being provided or the calculation is zeroing out.
**Impact:** "0s" suggests services just started, which may not be true and could cause unnecessary concern.
**Recommendation:** If uptime data isn't available, show "—" instead of "0s". If available, fix the calculation.

### Interactive States: 2 findings

#### 14. Some aria-expanded buttons have no accessible label
**Severity:** Minor
**Viewport(s):** All
**Evidence:** Two `<button aria-expanded>` elements have empty visible text. They appear to be stats bar expansion buttons. Screen readers announce "button, collapsed" with no description of what they expand.
**Impact:** Keyboard and screen reader users can't determine the purpose of these toggle buttons.
**Recommendation:** Add `aria-label` describing the action, e.g., `aria-label="Expand ready queue details"`.

#### 15. Console errors: ERR_INSUFFICIENT_RESOURCES during load
**Severity:** Minor
**Viewport(s):** All (headless browser context)
**Evidence:** 26 `ERR_INSUFFICIENT_RESOURCES` errors during page load. Affected stores: beads, review queue, verification, context. These are connection pool exhaustion from simultaneous SSE connections + API fetches in headless Chromium (6 connection limit per origin for HTTP/1.1).
**Impact:** In constrained environments (CI, automated monitoring, headless), some dashboard data may fail to load. Normal browser usage is less affected.
**Recommendation:** Consider staggering initial API fetches or adding a fetch queue. The existing `requestIdleCallback` pattern helps but `Promise.all` for critical fetches fires many concurrent requests.

---

## What Works Well

- **Card depth strategy is correct** — All cards use borders, not shadows, following the Toolshed design direction.
- **Border radius is consistent** — All cards use 8px, matching the design system 4/6/8px scale.
- **Typography system is correct** — Inter for UI text, JetBrains Mono for code/data. Both loaded correctly.
- **Empty states are informative** — "No active agents" with `orch spawn` hint provides clear, actionable guidance.
- **Section collapse persistence** — Expand/collapse state persists via localStorage across page loads.
- **Semantic color usage** — Green for active/running, amber for warnings/review, red for errors. Consistent and meaningful.
- **SSE connection indicator** — Clear connection status with color-coded dot (green/yellow/red) in header.
- **Coaching banner** — Warning state immediately visible with yellow border and emoji indicator.
- **No data presentation issues** — No null/undefined values, no raw database values exposed. Clean data rendering.

---

## Comparison with Prior Audit

A prior audit exists at `2026-02-28-audit-ux-dashboard.md` (orch-go-mzqk) conducted earlier the same day in **dark theme**. This fresh audit was conducted in **light theme**.

**Findings overlap:**
- Horizontal overflow below 1024px — confirmed in both audits (this audit: min-width ~790px)
- Touch targets below 44px — confirmed in both
- Nested interactive elements — confirmed (8 nodes)
- Select missing accessible name — confirmed (critical)
- No active nav state — confirmed
- Color contrast failures — confirmed (this audit: 59 nodes at 3.45:1 in light theme; prior: 42 nodes at 3.59:1 in dark theme)

**Differences:**
- Prior audit found agent card text overflow in Needs Attention section (Major). This audit didn't reproduce it (no completed agents in current session state).
- Prior audit found no headings at all (h1=0, h2=0, h3=0). This audit confirmed h1=0 but some sections may have h2/h3 depending on content state.
- This audit identified 26 console errors vs prior's 22 — likely increased due to headless browser resource constraints.

---

## Screenshot Index

| Filename | Viewport | State | Description |
|----------|----------|-------|-------------|
| baseline-1280.png | 1280px | default | Full desktop — stats bar, coaching, services, active agents, needs attention |
| baseline-1024.png | 1024px | default | Desktop with recent wins, ready queue visible below fold |
| baseline-768.png | 768px | default | Horizontal overflow begins — content exceeds viewport by 22px |
| baseline-640.png | 640px | default | Significant overflow — stats bar spills past viewport |
| baseline-375.png | 375px | default | Mobile — 790px content in 375px viewport, severe overflow |
| header-desktop.png | 1280px | header crop | Clean header with nav links, status indicator, theme toggle |
| header-mobile.png | 375px | header crop | Nav wrapping, status truncated, header overflows |

---

## Accessibility Metrics

| Metric | Value |
|--------|-------|
| axe-core violations (total) | 3 |
| axe-core critical | 1 (select-name: 1 node) |
| axe-core serious | 2 (color-contrast: 59 nodes, nested-interactive: 8 nodes) |
| axe-core moderate | 0 |
| axe-core minor | 0 |
| axe-core passes | 16 |
| axe-core incomplete (needs review) | 1 (color-contrast: 11 nodes) |
| Heading levels used | None (h1 absent, h2/h3 absent in current state) |
| Landmark regions | 3 (header, nav, main) |
| Unlabeled interactive elements | 1 (select) + 2 (aria-expanded buttons) |
| Keyboard navigable | Partial |
| Skip-to-content link | Absent |
| Focus visibility | Default browser outline |

---

## Reproducibility

**Auth:** None required (local dev server)
**Theme:** Light (system default)
**Dashboard mode:** Operational
**Tools:** Playwright 1.58.2 (chromium headless), axe-core 4.11.1 (local injection)
**Scripts:** `.kb/investigations/scripts/dashboard-audit.cjs`, `.kb/investigations/scripts/axe-audit.cjs`
**Re-audit schedule:** After responsive fixes; monthly during active development

---

## Recommended Next Steps

**Immediate actions (Major severity):**
- [ ] Fix `--muted-foreground` contrast: darken from `#8a8a8a` to Slate 500 (`#64748B`, 4.6:1)
- [ ] Add `aria-label="Filter time range"` to time filter `<select>`
- [ ] Refactor nested interactive elements (tooltip buttons inside card buttons)
- [ ] Add active state + `aria-current="page"` to Dashboard nav link

**Architect follow-up recommended:**
- [ ] Responsive architecture below 790px — needs structural approach, not piecemeal fixes
- [ ] Header collapse behavior at sm breakpoint (hamburger menu or icon-only nav)
- [ ] Touch target sizing strategy — affects all pages, not just dashboard

**Re-scan:** After responsive and a11y fixes are implemented
