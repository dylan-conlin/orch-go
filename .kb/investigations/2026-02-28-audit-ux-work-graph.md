# Investigation: UX Audit — Work Graph

**TLDR:** 14 findings across 6 dimensions. 0 blockers, 4 major, 7 minor, 3 cosmetic. Primary issues: horizontal overflow at mobile, no headings (h1-h6), missing aria-current on nav, and several small touch targets. The page is functional at desktop but degrades significantly at narrow viewports.

**Status:** Complete
**Date:** 2026-02-28
**Mode:** quick
**Target:** http://localhost:5188/work-graph
**Viewports:** 1280, 640, 375

---

## Baseline Metrics

| Metric | Value | Prior Audit | Delta |
|--------|-------|-------------|-------|
| Total findings | 14 | first audit | — |
| Blocker | 0 | — | — |
| Major | 4 | — | — |
| Minor | 7 | — | — |
| Cosmetic | 3 | — | — |
| axe-core violations | N/A (CDN blocked) | — | — |
| axe-core passes | N/A | — | — |
| Console errors | 3 | — | — |

---

## Findings by Dimension

### Visual Consistency: 2 findings

#### 1. Dark theme only — no light theme tested
**Severity:** Cosmetic
**Viewport(s):** All
**Evidence:** Body background `rgb(15, 15, 15)`, text `rgb(237, 236, 238)`. Dark mode class `dark` on HTML element. Theme toggle button present.
**Impact:** Design audit references Toolshed light theme tokens (FAFBFC background, etc.) — dark theme uses different values. Not a bug, but design token compliance should be validated against dark theme specs.
**Recommendation:** Establish dark theme token reference for future audits.

#### 2. No box shadows found on any element
**Severity:** Cosmetic (positive finding)
**Evidence:** Zero shadow elements detected across entire page. Depth conveyed through borders only.
**Impact:** Consistent with Toolshed depth strategy (borders over shadows).
**Recommendation:** None — this is correct per design direction.

### Responsive: 4 findings

#### 3. Horizontal overflow at 375px mobile viewport
**Severity:** Major
**Viewport(s):** 375px
**Evidence:** `body.scrollWidth: 455px` vs `viewport: 375px` (80px overflow). Header container, status bar, and event ticker all overflow. The work-graph-container itself has `scrollWidth: 936` vs `clientWidth: 300`.
**Impact:** User sees horizontal scrollbar at mobile. Content is clipped or requires horizontal scrolling which is a poor mobile UX pattern.
**Recommendation:** Add `overflow-x: hidden` or `max-width: 100vw` to the root container. The status bar (`flex items-center gap-6`) needs `flex-wrap` or responsive rearrangement. The event ticker (font-mono) needs `overflow: hidden` with text truncation.

#### 4. Status bar content truncated at narrow viewports without readable fallback
**Severity:** Major
**Viewport(s):** 375px, 640px
**Screenshot:** `baseline-375.png`
**Evidence:** Status bar shows "Daemon: pau" truncated at 375px. Key metrics (issues, edges, project name) are hidden.
**Impact:** User cannot see daemon status, issue counts, or project context at mobile — this is operational information.
**Recommendation:** Stack status bar info vertically at narrow viewports, or use a collapsible summary with the most critical metric visible.

#### 5. Feature type badges clipped at right edge on mobile
**Severity:** Minor
**Viewport(s):** 375px
**Evidence:** Badge text shows "featur" or "f" — clipped at viewport edge without ellipsis.
**Impact:** User can't see issue type badges. Functional loss since badges encode issue type (feature/task/bug).
**Recommendation:** Allow badges to wrap below the title at narrow viewports, or hide them and rely on color coding.

#### 6. Keyboard shortcut bar overlaps content at mobile
**Severity:** Minor
**Viewport(s):** 375px
**Evidence:** Fixed bottom bar shows "j/k navigate · h/l collapse/expand · enter details · i side panel · v verify · x close · c copy ID · t/w WIP+tree · G cycle groups" — wraps to multiple lines, covering page content.
**Impact:** Keyboard shortcuts are not useful on mobile (no keyboard) and the bar wastes ~60px of vertical space.
**Recommendation:** Hide keyboard shortcut bar at viewports <=640px (`hidden sm:block`).

### Accessibility: 4 findings

#### 7. No heading elements on the page (h1-h6)
**Severity:** Major
**Viewport(s):** All
**Evidence:** `document.querySelectorAll('h1, h2, h3, h4, h5, h6')` returns empty array. The page has zero headings.
**Impact:** Screen reader users cannot navigate by headings — the primary navigation method for screen readers. The page appears as an undifferentiated block of text/interactive elements.
**Recommendation:** Add `<h1>Work Graph</h1>` as the page title. Use `<h2>` for section headers like "Ready to Complete", "Set review tier...", "Independent Issues".

#### 8. No `aria-current="page"` on active nav link
**Severity:** Minor
**Viewport(s):** All
**Evidence:** "Work Graph" link at `/work-graph` has `ariaCurrent: null`. All three nav links use the same `text-muted-foreground` class — no visual active state differentiation.
**Impact:** Screen reader users cannot tell which page they're on from the navigation. Sighted users also lack a visual indicator of the current page.
**Recommendation:** Add `aria-current="page"` to the active nav link. Apply `text-foreground` (or accent color) class to active link instead of `text-muted-foreground`.

#### 9. Multiple small touch targets at mobile
**Severity:** Minor
**Viewport(s):** 375px
**Evidence:** Nav links height 24px (Dashboard: 79x24, Swarm logo: 69x24). Theme toggle: 36x36. "Close" buttons on completion cards: 51x22. All below WCAG 2.5.5 minimum of 44x44.
**Impact:** Mobile users may have difficulty tapping small targets, especially the Close buttons on completion review cards.
**Recommendation:** Increase nav link tap targets to min 44px height on mobile. Increase Close button size or add padding.

#### 10. Tree items use generic `div` roles instead of proper treeview ARIA
**Severity:** Minor
**Viewport(s):** All
**Evidence:** The a11y snapshot shows `tree [active]` and `treeitem` roles — this is actually good. However, the tree items contain nested `button` elements that duplicate the treeitem's text, creating redundant interactive elements.
**Impact:** Screen readers may announce items twice. The nested button inside a treeitem is an unusual pattern.
**Recommendation:** Review whether the inner button is needed — if the treeitem itself is interactive, the inner button is redundant.

### Data Presentation: 1 finding

#### 11. "runtime unknown" and "tokens unknown" displayed as raw text
**Severity:** Minor
**Viewport(s):** All
**Evidence:** Completion review cards show "runtime unknown" and "tokens unknown" — these are internal system values exposed to the user.
**Impact:** Looks unfinished/raw. A user unfamiliar with the system wouldn't know what "tokens unknown" means.
**Recommendation:** Either hide unknown values (show only when known) or use a dash "—" placeholder with tooltip "Not yet available".

### Navigation: 2 findings

#### 12. No visual active state on current nav link
**Severity:** Major
**Viewport(s):** All
**Evidence:** All three nav links (Dashboard, Work Graph, Knowledge Tree) use identical `text-muted-foreground` styling. No underline, no accent color, no bold weight on the active "Work Graph" link.
**Screenshot:** `baseline-1280.png` — all three nav items appear identical in the header.
**Impact:** User has no visual cue indicating which page they're viewing. Must rely on page content to know where they are.
**Recommendation:** Apply accent color or `text-foreground` to the active link. Add an underline or bottom border indicator.

#### 13. Page title is generic "Swarm Dashboard" on all pages
**Severity:** Minor
**Viewport(s):** All
**Evidence:** `document.title === "Swarm Dashboard"` on the Work Graph page.
**Impact:** Users with multiple tabs can't distinguish pages. Bookmarks would all have the same name.
**Recommendation:** Set page title to "Work Graph — Swarm Dashboard" (or similar).

### Interactive States: 1 finding

#### 14. Console errors from orch serve connection failures
**Severity:** Cosmetic
**Viewport(s):** All
**Evidence:** 3 `ERR_CONNECTION_CLOSED` errors for `https://localhost:3348/api/events/context`, `/api/events`, and `/api/agentlog?follow=true`.
**Impact:** The page appears to function despite these errors (SSE streams reconnect). However, the "disconnected" indicator was visible briefly before switching to "connected". If orch serve is down, user sees persistent "disconnected" without clear guidance.
**Recommendation:** Suppress console errors for expected SSE reconnection patterns. Consider adding a brief message when backend is unavailable.

---

## What Works Well

- **Landmarks present:** header, nav, main — proper semantic structure
- **No unlabeled interactive elements:** All buttons have text content
- **Tree view uses proper ARIA roles:** `tree` and `treeitem` roles are correctly applied
- **No horizontal overflow at desktop:** 1280px layout is clean
- **Keyboard shortcut discoverability:** Bottom bar clearly shows available shortcuts with descriptions
- **Depth strategy:** Zero box shadows — borders only, consistent with design direction
- **Typography:** Inter font family correctly applied as primary UI font
- **Progressive disclosure:** Dependency chains are collapsible, Independent Issues section is collapsible
- **Real-time status:** Event ticker shows agent activity, daemon status bar shows system health
- **Grouping dropdown:** Multiple grouping modes (Priority, Area, Effort, Dependency Chain)

---

## Screenshot Index

| Filename | Viewport | State | Description |
|----------|----------|-------|-------------|
| baseline-1280.png | 1280px | default | Desktop view, dark theme, dependency chain grouping |
| baseline-1280-full.png | 1280px | full page | Full page scroll at desktop |
| baseline-640.png | 640px | default | sm breakpoint, content starts truncating |
| baseline-375.png | 375px | default | Mobile viewport, significant overflow |
| baseline-375-full.png | 375px | full page | Full mobile scroll showing clipped content |

---

## Reproducibility

**Auth:** None required (local dev server)
**Commands:** browser_navigate(localhost:5188/work-graph), browser_resize, browser_snapshot, browser_evaluate
**Re-audit schedule:** After responsive fixes are implemented

---

## Recommended Next Steps

**Immediate actions (Major findings):**
- [ ] Fix horizontal overflow at 375px mobile viewport (finding #3)
- [ ] Add heading structure h1-h3 to the page (finding #7)
- [ ] Add visual active state to current nav link (finding #12)
- [ ] Make status bar responsive — stack or summarize at narrow viewports (finding #4)

**Quick wins (Minor findings):**
- [ ] Add `aria-current="page"` to active nav link (finding #8)
- [ ] Set page-specific `<title>` (finding #13)
- [ ] Hide keyboard shortcut bar on mobile (finding #6)
- [ ] Replace "runtime unknown"/"tokens unknown" with dash placeholder (finding #11)

**Focused audits needed:**
- [ ] Run ux-audit focused:accessibility with local axe-core (CDN was blocked)
- [ ] Run ux-audit focused:responsive after fixes to verify overflow resolution
