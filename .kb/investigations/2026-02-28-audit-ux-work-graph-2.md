# Investigation: UX Audit — Work Graph (Second Audit)

**TLDR:** 16 findings across 6 dimensions. 0 blockers, 5 major, 8 minor, 3 cosmetic. Primary issues: WCAG AA contrast failure on muted text/nav links (3.76:1 ratio), no heading elements (h1-h6), horizontal overflow at mobile, no visual active state on nav. The page is well-built at desktop (1280px) with proper ARIA tree roles, real-time updates, and a functional side panel. Mobile degrades significantly.

**Status:** Complete
**Date:** 2026-02-28
**Beads:** orch-go-xb51
**Mode:** full (6 dimensions, full depth)
**Target:** http://localhost:5188/work-graph
**Viewports:** 1280, 1024, 768, 640, 375
**Prior Audit:** 2026-02-28-audit-ux-work-graph.md (quick scan, 14 findings)

---

## Baseline Metrics

| Metric | Value | Prior Audit | Delta |
|--------|-------|-------------|-------|
| Total findings | 16 | 14 | +2 |
| Blocker | 0 | 0 | 0 |
| Major | 5 | 4 | +1 |
| Minor | 8 | 7 | +1 |
| Cosmetic | 3 | 3 | 0 |
| axe-core violations | N/A (CDN blocked by CSP) | N/A | — |
| axe-core passes | N/A | N/A | — |
| Console errors | 0 | 3 | -3 (improved) |

---

## Findings by Dimension

### Visual Consistency: 2 findings

#### 1. Dark theme only — design token compliance untested against light theme
**Severity:** Cosmetic
**Viewport(s):** All
**Evidence:** Body background `rgb(15, 15, 15)`, text `rgb(237, 236, 238)`. HTML element has `dark` class. Theme toggle button present.
**Impact:** Toolshed design tokens reference light theme (FAFBFC background). Dark theme uses different values. Not a bug — but design token compliance can only be validated against a dark theme spec.
**Recommendation:** Establish dark theme token reference for future audits.

#### 2. Zero box shadows — depth via borders only (positive)
**Severity:** Cosmetic (positive finding)
**Evidence:** `shadowCount: 0` across entire page. No box-shadow on any element.
**Impact:** Consistent with Toolshed depth strategy: borders over shadows.
**Recommendation:** None — this is correct per design direction.

### Responsive: 4 findings

#### 3. Horizontal overflow at 375px mobile viewport
**Severity:** Major
**Viewport(s):** 375px
**Screenshot:** `baseline-375.png`
**Evidence:** `body.scrollWidth: 455px` vs `viewport: 375px` (80px overflow). Header container overflows. `work-graph-container` has `scrollWidth: 999` vs `clientWidth: 300`. The `flex items-center gap-6` status bar has `scrollWidth: 991` vs `clientWidth: 284`.
**Impact:** User sees horizontal scrollbar at mobile. Content is clipped and requires horizontal scrolling — a poor mobile UX pattern.
**Recommendation:** Add `overflow-x: hidden` or `max-width: 100vw` to root container. The status bar needs `flex-wrap` or responsive rearrangement below 640px. The event ticker needs `overflow: hidden` with text truncation.

#### 4. Status bar content truncated without readable fallback at narrow viewports
**Severity:** Major
**Viewport(s):** 375px, 640px
**Screenshot:** `baseline-375.png`, `baseline-640.png`
**Evidence:** At 375px, status bar shows "Daemon: runn" truncated. Key metrics (issue count, edge count, project name) hidden. At 640px, metrics visible but "188 q..." truncated.
**Impact:** Operational information (daemon status, issue counts, project context) invisible at mobile.
**Recommendation:** Stack status bar info vertically at viewports <=640px, or use a collapsible summary showing the most critical metric.

#### 5. Feature type badges clipped at right edge on mobile
**Severity:** Minor
**Viewport(s):** 375px, 640px
**Screenshot:** `baseline-375.png`
**Evidence:** Badge text shows "featur" or "fe" — clipped at viewport edge without ellipsis. At 375px, some issues show no title at all — only priority badge and issue ID visible.
**Impact:** Users cannot see issue type badges. Functional loss since badges encode type (feature/task/bug).
**Recommendation:** Allow badges to wrap below title at narrow viewports, or hide them and rely on color coding at mobile.

#### 6. Keyboard shortcut bar wraps to multiple lines at mobile
**Severity:** Minor
**Viewport(s):** 375px, 640px
**Screenshot:** `baseline-375.png`
**Evidence:** Fixed bottom bar wraps to 2-3 lines at 640px, consuming ~60-90px of vertical space. Shows "j/k navigate · h/l collapse/expand · enter details · i side panel · v verify · x close · c copy ID · t/w WIP+tree · G cycle groups".
**Impact:** Keyboard shortcuts are not useful on mobile (no keyboard) and the bar wastes significant vertical space.
**Recommendation:** Hide keyboard shortcut bar at viewports <=640px (`hidden sm:block`).

### Accessibility: 5 findings

#### 7. No heading elements on the page (h1-h6)
**Severity:** Major
**Viewport(s):** All
**Evidence:** `document.querySelectorAll('h1, h2, h3, h4, h5, h6')` returns empty array. Zero headings on the page. (Note: the side panel dialog does contain an h2 when opened.)
**Impact:** Screen reader users cannot navigate by headings — the primary navigation method. The page appears as an undifferentiated block of text/interactive elements.
**Recommendation:** Add `<h1>Work Graph</h1>` as the page title. Use `<h2>` for section headers like "Ready to Complete", group headers, "Independent Issues".

#### 8. Muted text and nav links fail WCAG AA contrast
**Severity:** Major
**Viewport(s):** All
**Evidence:** Contrast spot check results:
- Muted text: `rgb(110,110,110)` on `rgb(15,15,15)` = **3.76:1** (needs 4.5:1 for 12px text)
- Nav links: same `rgb(110,110,110)` = **3.76:1** (needs 4.5:1 for 12px text)
- "runtime unknown" text: same = **3.76:1**
- Primary text: `rgb(237,236,238)` on `rgb(15,15,15)` = **16.28:1** (passes)
- Issue IDs (mono): same = **16.28:1** (passes)
**Impact:** Users with low vision may struggle to read nav links, muted text, and status information. This affects navigation and operational awareness.
**WCAG:** 1.4.3 Contrast (Minimum)
**Recommendation:** Increase muted text color from `rgb(110,110,110)` to at least `rgb(137,137,137)` for 4.5:1 ratio on dark background. Consider `rgb(150,150,150)` for comfortable margin.

#### 9. No `aria-current="page"` on active nav link
**Severity:** Minor
**Viewport(s):** All
**Evidence:** "Work Graph" link at `/work-graph` has `ariaCurrent: null`. All three nav links use identical styling — no active state differentiation.
**Impact:** Screen reader users cannot determine which page they're on from the navigation.
**Recommendation:** Add `aria-current="page"` to the active nav link.

#### 10. Navigation element has no `aria-label`
**Severity:** Minor
**Viewport(s):** All
**Evidence:** `<nav>` element has `ariaLabel: 'unlabeled'`. No `aria-label` or `aria-labelledby` attribute.
**Impact:** Screen readers announce "navigation" without context. When multiple nav elements exist, users can't distinguish them.
**Recommendation:** Add `aria-label="Main navigation"` to the `<nav>` element.

#### 11. Multiple small touch targets at mobile
**Severity:** Minor
**Viewport(s):** 375px (most affected), 1280px (nav links)
**Evidence:** At 1280px: Nav links 24px height (Dashboard: 79x24, Work Graph: 84x24). Theme toggle: 36x36. At 375px additionally: "Close" button 51x22, "Resume" button 68x26. WCAG 2.5.5 minimum is 44x44.
**Impact:** Mobile users may have difficulty tapping small targets, especially the Close/Resume buttons on completion review cards.
**Recommendation:** Increase nav link tap targets to min 44px height. Increase Close/Resume button padding to min 44px height.

### Data Presentation: 1 finding

#### 12. "runtime unknown" and "tokens unknown" displayed as raw text
**Severity:** Minor
**Viewport(s):** All
**Evidence:** Ready to Complete section and in-progress agent rows show "runtime unknown" and "tokens unknown" — internal system values exposed to the user.
**Impact:** Looks unfinished/raw. A user unfamiliar with the system wouldn't know what "tokens unknown" means.
**Recommendation:** Hide unknown values (show only when known) or use a dash "—" placeholder.

### Navigation: 2 findings

#### 13. No visual active state on current nav link
**Severity:** Major
**Viewport(s):** All
**Screenshot:** `baseline-1280.png`
**Evidence:** All three nav links (Dashboard, Work Graph, Knowledge Tree) use identical `text-muted-foreground` styling with `color: rgb(110,110,110)` and `fontWeight: 500`. No underline, no accent color, no bold weight, no indicator on the active "Work Graph" link.
**Impact:** User has no visual cue indicating which page they're viewing. Must rely on page content for orientation.
**Recommendation:** Apply `text-foreground` or accent color to the active link. Add underline or bottom border indicator.

#### 14. Page title is generic "Swarm Dashboard" on all pages
**Severity:** Minor
**Viewport(s):** All
**Evidence:** `document.title === "Swarm Dashboard"` on the Work Graph page.
**Impact:** Users with multiple tabs can't distinguish pages. Bookmarks all have the same name.
**Recommendation:** Set page title to "Work Graph — Swarm Dashboard".

### Interactive States: 2 findings

#### 15. Focus ring not visible after Tab press
**Severity:** Minor
**Viewport(s):** All
**Screenshot:** `interactive-keyboard-focus.png`
**Evidence:** After pressing Tab, no visible focus indicator appeared on any element. The screenshot before and after Tab press are visually identical.
**Impact:** Keyboard users cannot tell which element has focus. WCAG 2.4.7 requires visible focus indicators.
**WCAG:** 2.4.7 Focus Visible
**Recommendation:** Ensure `:focus-visible` styles produce visible outline/ring on all interactive elements. Verify `outline: none` isn't suppressing default browser focus indicators.

#### 16. j/k keyboard navigation feedback unclear
**Severity:** Cosmetic
**Viewport(s):** All
**Evidence:** After pressing `j` key (advertised as "navigate" in shortcuts bar), the `[selected]` attribute on the first group header did not visibly change in the a11y tree. The visual appearance of the tree also appeared unchanged.
**Impact:** Users may not realize keyboard navigation is working. The shortcut bar advertises j/k but feedback is subtle or non-functional.
**Recommendation:** Add visible highlight/background change when j/k moves selection. Verify j/k handler is bound correctly.

---

## What Works Well

- **Proper ARIA tree roles:** `tree` and `treeitem` roles correctly applied with `[selected]` state management
- **Side panel is well-built:** Opens on click, has h2 heading, tabs (Overview/Activity/Screenshots), message input, close button. Escape key closes it properly.
- **Real-time updates:** Event ticker shows agent activity in real-time. "Ready to Complete" section appears dynamically when agents finish. Daemon status updates live.
- **Landmarks present:** `<header>`, `<nav>`, `<main>` — proper semantic structure
- **No unlabeled interactive elements:** All buttons have text content
- **Zero box shadows:** Borders only, consistent with Toolshed design direction
- **Typography correct:** Inter for UI text, JetBrains Mono for data values (issue IDs, event ticker)
- **Progressive disclosure:** Dependency chains are collapsible, Independent Issues section is collapsible
- **Daemon control banner:** "Daemon paused" banner appears with actionable buttons (Close All, Resume) when completions await review
- **Grouping dropdown functional:** By Priority, By Area, By Effort, By Dependency Chain modes all available
- **Console errors eliminated:** Prior audit found 3 ERR_CONNECTION_CLOSED errors; this audit found 0. Improvement.
- **Primary text contrast excellent:** 16.28:1 ratio for primary text and issue IDs

---

## Comparison with Prior Audit

**Prior:** 2026-02-28-audit-ux-work-graph.md (quick scan)
**Prior findings:** 14 | **This audit:** 16

| Prior Finding | Status | Notes |
|--------------|--------|-------|
| #3 Horizontal overflow 375px | **Confirmed** | Same: body 455px vs viewport 375px |
| #4 Status bar truncated | **Confirmed** | Same behavior at narrow viewports |
| #5 Badges clipped at mobile | **Confirmed** | Same issue |
| #6 Keyboard shortcut bar overlaps | **Confirmed** | Same issue |
| #7 No heading elements | **Confirmed** | Still zero headings |
| #8 No aria-current on nav | **Confirmed** | Still missing |
| #9 Small touch targets | **Confirmed** | Same measurements |
| #10 Tree items redundant button | **Not reproduced** | Tree items work correctly now |
| #11 "runtime unknown" raw text | **Confirmed** | Still present |
| #12 No visual active state | **Confirmed** | Same identical styling on all nav links |
| #13 Generic page title | **Confirmed** | Still "Swarm Dashboard" |
| #14 Console errors | **Fixed** | Was 3, now 0. SSE connection stable. |

**New findings (not in prior audit):**
- **#8 Contrast failure** (3.76:1 on muted text) — NEW Major finding
- **#10 Navigation no aria-label** — NEW Minor finding
- **#15 Focus ring not visible** — NEW Minor finding
- **#16 j/k navigation unclear** — NEW Cosmetic finding

---

## Screenshot Index

| Filename | Viewport | State | Description |
|----------|----------|-------|-------------|
| baseline-1280.png | 1280px | default | Desktop, dark theme, dependency chain grouping |
| baseline-1280-full.png | 1280px | full page | Full page scroll at desktop |
| baseline-1024.png | 1024px | default | lg breakpoint, some truncation starts |
| baseline-768.png | 768px | default | md breakpoint, titles begin truncating |
| baseline-640.png | 640px | default | sm breakpoint, significant truncation |
| baseline-375.png | 375px | default | Mobile, horizontal overflow, badges clipped |
| interactive-hover-treeitem.png | 1280px | hover | Hovered over first tree item |
| interactive-jk-navigate.png | 1280px | j pressed | After pressing j key for navigation |
| interactive-keyboard-focus.png | 1280px | tab pressed | After Tab key — no visible focus ring |
| interactive-sidepanel.png | 1280px | panel open | Side panel open showing issue detail |

---

## Reproducibility

**Auth:** None required (local dev server)
**Commands:** browser_navigate(localhost:5188/work-graph), browser_resize(1280/1024/768/640/375), browser_snapshot, browser_evaluate(computed styles, overflow check, contrast check), browser_hover, browser_press_key, browser_click
**axe-core:** CDN blocked by CSP — structural review and manual contrast check performed instead
**Re-audit schedule:** After responsive fixes and contrast fixes are implemented

---

## Recommended Next Steps

**Immediate actions (Major findings):**
- [ ] Fix muted text contrast: increase from `rgb(110,110,110)` to at least `rgb(137,137,137)` for 4.5:1 AA compliance (finding #8)
- [ ] Add heading structure h1-h3 to the page (finding #7)
- [ ] Add visual active state to current nav link (finding #13)
- [ ] Fix horizontal overflow at 375px mobile viewport (finding #3)
- [ ] Make status bar responsive — stack or summarize at narrow viewports (finding #4)

**Quick wins (Minor findings):**
- [ ] Add `aria-current="page"` to active nav link (finding #9)
- [ ] Add `aria-label="Main navigation"` to nav element (finding #10)
- [ ] Set page-specific `<title>` (finding #14)
- [ ] Hide keyboard shortcut bar on mobile (finding #6)
- [ ] Replace "runtime unknown"/"tokens unknown" with dash placeholder (finding #12)
- [ ] Ensure `:focus-visible` produces visible focus indicators (finding #15)

**Architect review recommended before implementation:**
- Responsive design requires structural decisions about what to show/hide at narrow widths
- This is a hotspot area — see spawn context warning
