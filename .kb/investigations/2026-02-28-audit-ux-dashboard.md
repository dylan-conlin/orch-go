# Investigation: UX Audit — Dashboard Page

**TLDR:** Full 6-dimension UX audit of the Swarm Dashboard (`/`). Found 14 total findings (0 Blocker, 7 Major, 5 Minor, 2 Cosmetic). Critical issues: page has horizontal overflow at all viewports below 1024px (page min-width ~772px), 42 color contrast failures (muted text 3.59:1 ratio), no semantic headings, nested interactive elements in agent cards, and agent card text overflow where completion summaries spill outside card boundaries.

**Status:** Complete
**Date:** 2026-02-28
**Beads:** orch-go-mzqk
**Mode:** full
**Target:** http://localhost:5188/
**Viewports:** 1280, 1024, 768, 640, 375

---

## Baseline Metrics

| Metric | Value | Prior Audit | Delta |
|--------|-------|-------------|-------|
| Total findings | 14 | first audit | — |
| Blocker | 0 | — | — |
| Major | 7 | — | — |
| Minor | 5 | — | — |
| Cosmetic | 2 | — | — |
| axe-core violations | 3 types (51 nodes) | — | — |
| axe-core passes | 18 | — | — |
| axe-core incomplete | 1 | — | — |
| Console errors | 22 | — | — |

---

## Findings by Dimension

### Visual Consistency: 2 findings

#### 1. Dark Theme — Design System Token Divergence
**Severity:** Minor
**Design Token:** All Toolshed light-mode tokens
**Expected:** Toolshed design direction specifies light mode tokens (#FAFBFC background, #FFFFFF surfaces)
**Actual:** Dark theme active — body background rgb(15,15,15), card surfaces rgb(21,20,26), borders rgb(46,46,46)
**Elements Affected:** Entire page
**Impact:** Not a bug per se — the dark theme exists intentionally. However, the Toolshed design direction document (`.kb/decisions/2026-02-21-toolshed-design-direction.md`) specifies only light-mode tokens. The dark theme appears to be a custom implementation without documented design tokens.
**Recommendation:** Document dark mode token values in the design direction, or ensure both themes follow the same token structure. Currently the dark theme works but isn't spec'd.

#### 2. Agent Card Semantic Shadow Usage
**Severity:** Cosmetic
**Design Token:** Depth strategy (borders for cards, shadows for dropdowns/modals only)
**Expected:** Cards use borders only, shadows reserved for overlays
**Actual:** Dead agent cards have red shadow glow (`rgba(239,68,68,0.2) 0px 4px 6px`), awaiting-cleanup cards have amber glow
**Elements Affected:** 4 agent cards in Needs Attention section
**Impact:** This is semantic shadow usage (red = error, amber = warning) — arguably a reasonable deviation from the "borders only" rule since it communicates state. Not blocking.
**Recommendation:** Consider if this is an intentional pattern. If so, document it as an exception to the depth strategy.

### Responsive: 4 findings

#### 3. Horizontal Overflow Below 1024px
**Severity:** Major
**Viewport(s):** 768px, 640px, 375px
**Screenshot:** `baseline-768.png`, `baseline-640.png`, `baseline-375.png`
**Evidence:** At 375px: body scrollWidth 772px vs viewport 375px (2x overflow). At 640px: same 772px content width. At 768px: 772px vs 768px (slight overflow). Content min-width appears to be ~772px.
**Impact:** On any viewport below 1024px, users get a horizontal scrollbar and content extends beyond the visible area. The metrics toolbar and agent cards in the Needs Attention section are the primary overflow sources. The page is essentially unusable on mobile and half-screen desktop.
**Recommendation:** The metrics toolbar needs to collapse/stack at narrow viewports. Agent cards in the grid need to be constrained to their container width. Consider `overflow-x: hidden` on the main container with responsive wrapping inside.

#### 4. Header/Nav Overlaps Content at Narrow Viewports
**Severity:** Major
**Viewport(s):** 768px, 640px, 375px
**Screenshot:** `baseline-768.png`, `baseline-640.png`, `baseline-375.png`
**Evidence:** At 640px and below, the sticky header nav ("Swarm Dashboard Work Graph Knowledge Tree") visually overlaps with the metrics toolbar below it. The nav items and connection status indicator collide with the stat metrics row.
**Impact:** Navigation links overlap with the metrics toolbar making both areas difficult to read and interact with.
**Recommendation:** The header should collapse to a hamburger menu at sm (640px) breakpoint per CLAUDE.md rules. Currently the header never collapses — it just overflows.

#### 5. Touch Targets Below Minimum Size on Mobile
**Severity:** Major
**Viewport(s):** 375px, 640px
**Evidence:** At 375px, most interactive elements are 24-28px tall (WCAG 2.5.5 minimum is 44px). Examples: Ops button (55x24), History button (73x24), Following button (28x24), Select theme (36x36), Settings (36x36).
**Impact:** Touch targets too small for reliable mobile interaction. Users will misclick frequently.
**Recommendation:** Since this is a desktop-first dashboard, adding `min-h-[44px]` at mobile breakpoints would fix this. However, given the fundamental overflow issues (#3), responsive mobile support requires broader architecture work.

#### 6. Metrics Toolbar Does Not Wrap Gracefully
**Severity:** Minor
**Viewport(s):** 768px, 640px, 375px
**Screenshot:** `baseline-768.png`
**Evidence:** The toolbar with Ops/History/Since/Following/errors/active/ready/review/slots items uses `flex-wrap` but the total width exceeds container bounds. The items wrap to a second line but still overflow horizontally.
**Impact:** Stats partially hidden or overlapping at narrow viewports.
**Recommendation:** At narrow viewports, collapse the toolbar into an expandable summary (e.g., show only error count + active count, with a "more" toggle). This is a design decision that needs architect review.

### Accessibility: 5 findings

#### 7. No Semantic Heading Hierarchy
**Severity:** Major
**WCAG:** 1.3.1 Info and Relationships
**Source:** Structural review (browser snapshot)
**Evidence:** `document.querySelector('h1')` = null, `h2` = null, `h3` = null. The page uses `<div>` and `<span>` elements with visual styling but zero semantic headings. Section headers like "Active Agents", "Needs Attention", "Up Next", "Ready Queue" are all divs.
**Impact:** Screen reader users have no way to navigate the page by headings. The page structure is invisible to assistive technology. This is a fundamental accessibility gap.
**Recommendation:** Add heading elements: `<h1>Dashboard</h1>`, `<h2>` for each section (Active Agents, Needs Attention, Up Next, etc.). This is a straightforward fix.

#### 8. Color Contrast Failures — Muted Text (42 nodes)
**Severity:** Major
**WCAG:** 1.4.3 Contrast (Minimum)
**Source:** axe-core scan
**Evidence:** Muted foreground color #6e6e6e on background #15141a has 3.59:1 contrast ratio. WCAG AA requires 4.5:1 for normal text. 42 elements affected including: "History" tab, "Since:" label, "errors"/"active"/"ready" labels, agent card metadata text, timestamps.
**Impact:** Low-vision users cannot read muted text. Even users with normal vision may struggle in bright environments.
**Recommendation:** Change `--muted-foreground` from #6e6e6e to at least #949494 (4.5:1 ratio on #15141a) or #8b8b8b (4.0:1 — still fails but closer). A value of #949494 passes AA for normal text.

#### 9. Nested Interactive Elements in Agent Cards (8 nodes)
**Severity:** Major
**WCAG:** 4.1.2 Name, Role, Value
**Source:** axe-core scan
**Evidence:** Agent cards in the Needs Attention section are `<button>` elements that contain nested `<button>` elements (action buttons for 🔥, 💀, 🧹, -, and beads ID links). Example: `<button class="group relative w-full cursor-pointer rounded border bg-card...">` contains `<button>🔥</button>`, `<button>orch-go-d5g6</button>`, etc.
**Impact:** Screen readers may not announce nested interactive elements correctly. Focus management is unpredictable — which button receives focus? Keyboard users may struggle to activate the correct nested action.
**Recommendation:** Restructure agent cards: make the card a `<div>` with `role="group"` or use a `<details>`/`<summary>` pattern. Move clickable behavior to a specific "expand" button rather than making the entire card a button.

#### 10. Time Filter Select Missing Accessible Name
**Severity:** Major
**WCAG:** 4.1.2 Name, Role, Value
**Source:** axe-core scan (critical impact)
**Evidence:** `<select class="h-6 rounded border..." data-testid="time-filter">` has no label, no `aria-label`, no `aria-labelledby`, and no `title`. The adjacent "Since:" text is not programmatically associated.
**Impact:** Screen reader users cannot determine the purpose of this dropdown. axe-core rated this as critical impact.
**Recommendation:** Add `aria-label="Time range filter"` to the select element, or wrap the "Since:" text in a `<label for="time-filter-id">` and add an `id` to the select.

#### 11. No `aria-current` on Active Nav Link
**Severity:** Minor
**WCAG:** ARIA best practice
**Source:** Structural review
**Evidence:** The "Dashboard" link has `href="/"` matching the current path, but no `aria-current="page"` attribute. All three nav links have identical styling (color: rgb(110,110,110), fontWeight: 500).
**Impact:** Screen reader users cannot tell which page they're currently on from the nav alone.
**Recommendation:** Add `aria-current="page"` to the active nav link and visually distinguish it (e.g., brighter text color or underline).

### Data Presentation: 0 findings

No null/undefined values, no raw snake_case database values, no formatting issues detected. Data values (beads IDs, timestamps, counts) are displayed appropriately. The "orch spawn" code reference in the empty state uses `<code>` semantically — good.

### Navigation: 2 findings

#### 12. No Visual Active State in Navigation
**Severity:** Major
**Viewport(s):** All
**Evidence:** All three nav links (Dashboard, Work Graph, Knowledge Tree) have identical computed styles: `color: rgb(110,110,110)`, `fontWeight: 500`. The active "Dashboard" link is visually indistinguishable from inactive links.
**Impact:** User cannot tell which page they're on. Combined with the missing `aria-current` (finding #11), both visual and programmatic indicators are absent.
**Recommendation:** Add active state styling — e.g., brighter text color (`rgb(237,236,238)` matching body text), an underline indicator, or a background highlight.

#### 13. Page Title Is Generic
**Severity:** Minor
**Evidence:** `document.title` = "Swarm Dashboard" on the Dashboard page. This is the page title regardless of which page you're on (all pages likely share the same title).
**Impact:** When multiple tabs are open, users can't distinguish between pages. Bookmarks would all have the same name.
**Recommendation:** Set page title to "Dashboard — Swarm" on `/`, "Work Graph — Swarm" on `/work-graph`, etc.

### Interactive States: 1 finding

#### 14. Agent Card Completion Text Overflows Card Boundaries
**Severity:** Major
**Viewport(s):** All (most severe at 1024px and below)
**Screenshot:** `needs-attention-overflow-1280.png`
**Evidence:** Awaiting-cleanup agent cards display the full completion summary text (e.g., "Complete - Deep investigation of Context Mode (mksglu/claude-context-mode). Key finding: NOT LLM compression — it's subprocess isolation...") in a 16px-height container. scrollHeight: 188px, clientHeight: 16px — text overflows by 12x. The overflow is `visible`, causing text to spill into adjacent cards and sections.
**Impact:** The "Needs Attention" section is extremely difficult to read. Overlapping text from multiple agent cards creates visual chaos. At narrower viewports this gets even worse as cards are closer together.
**Recommendation:** Truncate the status/completion text to 1-2 lines with `overflow: hidden; text-overflow: ellipsis; -webkit-line-clamp: 2`. Show full text on hover/click expand. The card tooltip/popover pattern already exists in the codebase (the 🔥 and 💀 buttons have tooltips) — apply the same pattern to the status text.

---

## What Works Well

- **Font system is correct** — Inter for UI text, JetBrains Mono for data values (beads IDs, code blocks). This matches the Toolshed design direction.
- **Empty states are helpful** — "No active agents / Spawn with `orch spawn`" is contextual and actionable. Better than a blank area.
- **Semantic color usage for agent state** — Red borders/glow for dead agents, amber for awaiting cleanup is immediately readable.
- **SSE real-time updates** — The dashboard connects to SSE endpoints for live data. The "connected"/"disconnected" indicator in the header provides status visibility.
- **Keyboard focus is visible** — Tab navigation has a clear blue outline ring. Focus order is logical (header → toolbar → main content).
- **No null/undefined values** — Data presentation is clean with no raw database values exposed.
- **Collapsible sections** — "Up Next" and "Ready Queue" are collapsible, reducing information overload.
- **Dark theme execution** — While not documented in the design system, the dark theme is consistent within itself (dark background → darker cards → border separation).

---

## Comparison with Prior Audit

First comprehensive audit of this page. A prior investigation (`2026-02-27-audit-ux-dashboard-views.md`) exists but covered dashboard views (tabs/modes), not the UX health of the base page.

---

## Screenshot Index

| Filename | Viewport | State | Description |
|----------|----------|-------|-------------|
| baseline-1280.png | 1280px | default | Full page, desktop layout |
| baseline-1024.png | 1024px | default | Full page, last non-overflowing viewport |
| baseline-768.png | 768px | default | Full page, slight horizontal overflow begins |
| baseline-640.png | 640px | default | Full page, header/content overlap visible |
| baseline-375.png | 375px | default | Full page, severe overflow (2x viewport width) |
| needs-attention-overflow-1280.png | 1280px | scrolled | Needs Attention section showing text overflow |

---

## Reproducibility

**Auth:** None required (local dev server)
**Commands:** `browser_navigate(localhost:5188)`, `browser_resize({widths})`, `browser_snapshot`, `browser_evaluate(axe-core from unpkg CDN — cdnjs blocked)`
**Re-audit schedule:** After responsive fixes are implemented (recommend re-audit after each Major finding fix)

---

## Recommended Next Steps

**Immediate actions (Major findings):**
- [ ] Fix agent card text overflow (#14) — truncate completion text, add expand on click
- [ ] Add semantic headings (#7) — h1 for page, h2 for sections
- [ ] Fix muted text contrast (#8) — change --muted-foreground to ≥#949494
- [ ] Add `aria-label` to time filter select (#10)
- [ ] Restructure nested buttons in agent cards (#9)
- [ ] Add navigation active state (#12) + aria-current (#11)
- [ ] Address horizontal overflow below 1024px (#3) — this requires architect review

**Focused audits needed:**
- [ ] Run focused responsive audit after overflow fixes to verify breakpoint behavior
- [ ] Run focused interactive-states audit with active agents present (current audit only saw empty state)

**Re-scan:** After Major findings are addressed
