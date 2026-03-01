# Investigation: UX Audit — Work Graph (CLI Path)

**TLDR:** 15 findings across 6 dimensions. 0 blockers, 4 major, 7 minor, 3 cosmetic, 1 positive. Confirms all prior audit findings still present. New: tree role missing aria-label, data loading race condition exposes empty state. CLI path friction documented: axe-core CDN blocked, no interactive browser, data race requires explicit waits.

**Status:** Complete
**Date:** 2026-02-28
**Beads:** orch-go-t8ab
**Mode:** quick (CLI path — Playwright scripts via Bash, no MCP tools)
**Target:** http://localhost:5188/work-graph
**Viewports:** 1280, 1024, 768, 640, 375

---

## Baseline Metrics

| Metric | Value | Prior Audit | Delta |
|--------|-------|-------------|-------|
| Total findings | 15 | 14 | +1 |
| Blocker | 0 | 0 | 0 |
| Major | 4 | 4 | 0 |
| Minor | 7 | 7 | 0 |
| Cosmetic | 3 | 3 | 0 |
| Positive | 1 | 0 | +1 |
| axe-core violations | N/A (CDN blocked) | N/A | — |
| axe-core passes | N/A | N/A | — |
| Console errors | 0 | 3 | -3 |

---

## Findings by Dimension

### Visual Consistency: 2 findings

#### 1. Light theme captured — design tokens partially match Toolshed spec
**Severity:** Cosmetic
**Viewport(s):** All
**Evidence:** Body background `rgb(255, 255, 255)` (white), font-family `Inter`, body color `rgb(26, 26, 26)` (near-black). Zero box shadows detected. Borders used for depth (border-b on header, status bar, event ticker). Main padding: `12px 32px`.
**Impact:** Prior audit captured dark theme. This audit confirms light theme exists and mostly follows Toolshed direction: Inter font, borders-over-shadows depth strategy, no arbitrary shadow use. Body background is pure white rather than spec `#FAFBFC` — minor deviation.
**Recommendation:** Consider using `#FAFBFC` for body background per Toolshed design tokens.

#### 2. Zero box shadows — depth strategy correct (Positive)
**Severity:** Positive
**Viewport(s):** All
**Evidence:** `shadowCount: 0` across entire page. All depth conveyed through `border-b` and `border-border` classes.
**Impact:** Consistent with Toolshed depth strategy (borders, not shadows).
**Recommendation:** None — correct implementation.

### Responsive: 4 findings

#### 3. Horizontal overflow at 375px mobile viewport
**Severity:** Major
**Viewport(s):** 375px
**Screenshot:** `baseline-375.png`, `baseline-375-full.png`
**Evidence:** `body.scrollWidth: 459px` vs `viewport: 375px` (84px overflow). The overflow cascades from HEADER (container `flex h-10 items-center`), status bar (`flex items-center gap-6`), and event ticker (`whitespace-nowrap`). The `work-graph-container` has `scrollWidth: 1008px` vs `clientWidth: 311px`.
**Impact:** Horizontal scrollbar at mobile. Header, status bar, and content all extend beyond viewport.
**Recommendation:** Add `overflow-x: hidden` to body/root at mobile. Header container needs `flex-wrap` or responsive collapse. Status bar needs vertical stacking at narrow widths. Event ticker needs `overflow: hidden` with CSS animation instead of `whitespace-nowrap`.

#### 4. Status bar content truncated beyond readability at narrow viewports
**Severity:** Major
**Viewport(s):** 375px, 640px
**Screenshot:** `baseline-375.png`
**Evidence:** At 640px, the `truncate max-w-[40rem]` span has scrollWidth 539 vs clientWidth 475 (moderate truncation). At 375px, the status bar shows "Daemon: pause..." and key metrics (issue counts, edges, project name) are fully truncated. The `flex items-center gap-6` with `whitespace-nowrap` forces single-line layout that cannot fit.
**Impact:** User cannot see daemon status, issue counts, or project context at mobile — operational information is lost.
**Recommendation:** Stack status bar vertically at narrow viewports (below 640px). Show only critical info (daemon state, issue count) and collapse rest into expandable detail.

#### 5. Feature type badges clipped at mobile
**Severity:** Minor
**Viewport(s):** 375px
**Screenshot:** `baseline-375-full.png`
**Evidence:** At 375px, badges show "f" or "featur" — clipped at viewport edge. Badges like "feature" and "task" are positioned at the right edge with no room to render fully.
**Impact:** User can't identify issue type from the badge. Must infer from context or click to expand.
**Recommendation:** At narrow viewports, either hide badges (rely on color coding) or allow them to wrap below the title.

#### 6. Keyboard shortcut bar wastes space and overlaps content at mobile
**Severity:** Minor
**Viewport(s):** 375px, 640px
**Screenshot:** `baseline-375-full.png`
**Evidence:** Fixed bottom bar wraps to 3 lines at 375px showing keyboard shortcuts (j/k navigate, h/l collapse/expand, etc.). Covers approximately 60-80px of vertical space.
**Impact:** Keyboard shortcuts are irrelevant on mobile (no physical keyboard). The bar wastes critical vertical space and overlaps issue content.
**Recommendation:** Hide keyboard shortcut bar below 640px: `hidden sm:block`. On mobile, show shortcuts only in a help modal if needed.

### Accessibility: 4 findings

#### 7. No heading elements on the page (h1-h6)
**Severity:** Major
**Viewport(s):** All
**Evidence:** `document.querySelectorAll('h1,h2,h3,h4,h5,h6').length === 0`. Page has zero headings.
**Impact:** Screen reader users cannot navigate by headings — the primary navigation method. The page appears as an undifferentiated block. Section titles ("Set review tier in manifest at spawn time", "INDEPENDENT ISSUES") are styled text but not semantic headings.
**Recommendation:** Add `<h1>Work Graph</h1>` as page title. Use `<h2>` for section headers: "Ready to Complete", "Set review tier in manifest at spawn time", "Independent Issues".

#### 8. No `aria-current="page"` on active nav link
**Severity:** Minor
**Viewport(s):** All
**Evidence:** All 4 nav links (Swarm, Dashboard, Work Graph, Knowledge Tree) have `ariaCurrent: null`. "Work Graph" link at `/work-graph` has no `aria-current` attribute despite being the current page.
**Impact:** Screen reader users cannot tell which page they're on from the navigation.
**Recommendation:** Add `aria-current="page"` to the active nav link dynamically based on current route.

#### 9. Tree role missing `aria-label`
**Severity:** Minor
**Viewport(s):** All
**Evidence:** The `[role="tree"]` element has `ariaLabel: null` and `ariaLabelledBy: null`. Tree contains 16 treeitems.
**Impact:** Screen readers announce "tree" but cannot describe what the tree contains. Users must explore to understand context.
**Recommendation:** Add `aria-label="Work graph issues"` or similar to the tree element.

#### 10. Nested redundant buttons inside treeitems
**Severity:** Minor
**Viewport(s):** All
**Evidence:** Each `[role="treeitem"]` div contains a `[role="button"]` div with identical text. Additionally, each has a `span[role="button"]` for the beads ID (with `title="Click to copy"`). This creates 3 interactive layers per item: treeitem → button → copy span.
**Impact:** Screen readers announce items redundantly. Focus management is complex with nested interactive elements.
**Recommendation:** Review whether the inner `role="button"` is needed. If the treeitem itself handles click, the inner button div is redundant. Keep the copy span as it has a distinct purpose.

### Data Presentation: 2 findings

#### 11. "runtime unknown" and "tokens unknown" displayed as raw text
**Severity:** Minor
**Viewport(s):** All
**Screenshot:** `baseline-1280.png` (Ready to Complete section)
**Evidence:** The "Ready to Complete" card shows: `orch-go-mri5  Implement daemon agreement check periodic task  runtime unknown  tokens unknown  completed 3m ago  [Close]`
**Impact:** "runtime unknown" and "tokens unknown" are internal system values. A user unfamiliar with the system wouldn't understand what these mean.
**Recommendation:** Either hide unknown values (show only when known) or replace with "—" dash placeholder. Consider a tooltip "Runtime data not available".

#### 12. Data loading race condition exposes empty state before data arrives
**Severity:** Minor
**Viewport(s):** All
**Evidence:** First screenshots at 1280px showed "No open issues found" because the API data hadn't loaded within the initial wait period. The page uses `domcontentloaded` + polling with 30-second interval. There is no loading skeleton or spinner visible during the initial data fetch.
**Impact:** Users see "No open issues found" briefly before data appears, which could be confusing — especially if they know issues exist.
**Recommendation:** Add a loading skeleton or spinner that displays during the initial data fetch. Only show "No open issues found" after the first successful API response returns zero results.

### Navigation: 2 findings

#### 13. No visual active state on current nav link
**Severity:** Major
**Viewport(s):** All
**Screenshot:** `baseline-1280.png`
**Evidence:** All 4 nav links use identical styling: `color: rgb(138, 138, 138)` (muted gray), `fontWeight: 500`. No differentiation for the "Work Graph" link even though `window.location.pathname === "/work-graph"`.
**Impact:** User has no visual cue indicating which page they're viewing. Must rely on page content to orient themselves.
**Recommendation:** Apply accent color (SCS blue #509be0) or darker foreground color to the active link. Add an underline or bottom border indicator. Match the route via `window.location.pathname` comparison.

#### 14. Page title is generic "Swarm Dashboard" on all pages
**Severity:** Minor
**Viewport(s):** All
**Evidence:** `document.title === "Swarm Dashboard"` on the Work Graph page.
**Impact:** Users with multiple tabs cannot distinguish pages. Bookmarks all have the same name.
**Recommendation:** Set page title to "Work Graph — Swarm Dashboard" using SvelteKit's `<svelte:head>` component.

### Interactive States: 1 finding

#### 15. Multiple small touch targets across all viewports
**Severity:** Minor
**Viewport(s):** All (most impactful at 375px)
**Evidence:** 48 of 53 interactive elements have at least one dimension below 44px:
- Nav links: 73-109px wide × 24px tall
- Theme toggle button: 36×36px
- "Close All" button: 91×26px
- "Resume" button: 68×26px
- "Close" button: 51×22px (smallest actionable button)
- Select dropdown: 181×29px
**Impact:** On mobile, small targets are difficult to tap. The 22px-tall Close button is particularly problematic for touch interaction. On desktop with a mouse, this is less impactful but still violates WCAG 2.5.5.
**Recommendation:** Increase minimum height of all buttons to 36px (acceptable) or 44px (ideal). Nav links should have 44px tall tap targets on mobile via padding. The Close button especially needs a height increase.

---

## What Works Well

- **Landmarks present:** header, nav, main — proper semantic HTML structure
- **Tree ARIA roles:** `role="tree"` with `role="treeitem"` correctly implemented with `tabindex` management
- **Zero box shadows:** Consistent depth strategy using borders only (matches Toolshed design direction)
- **Inter font family:** Correctly applied as primary UI font
- **Hover states functional:** Theme toggle gets orange bg on hover. "Close All" gets amber-tinted bg. "Resume" gets green-tinted bg. All have `cursor: pointer`.
- **Copy-to-clipboard on issue IDs:** Beads IDs have `title="Click to copy"` and `role="button"` — clear affordance
- **Grouping dropdown:** 4 useful options (Priority, Area, Effort, Dependency Chain) with native `<select>` element
- **Progressive disclosure:** Dependency chains are collapsible tree nodes. Section headers are toggleable.
- **Event ticker:** Real-time event strip showing recent agent lifecycle events (auto-closed, spawned, abandoned)
- **Status bar information density:** At desktop, shows daemon state, slot count, last poll time, queue count, review count, issue count, edge count, project name — comprehensive operational awareness
- **No console errors:** 0 JS errors during this audit session (improved from prior audit's 3 SSE connection errors)

---

## Comparison with Prior Audit

**Prior audit:** 2026-02-28 (MCP path) — 14 findings (0B, 4M, 7m, 3C)
**This audit:** 2026-02-28 (CLI path) — 15 findings (0B, 4M, 7m, 3C, 1 positive)

| Prior Finding | Status in This Audit | Notes |
|---------------|---------------------|-------|
| #1 Dark theme only | N/A | This audit captured light theme instead |
| #2 No box shadows (positive) | Confirmed | Same — zero shadows |
| #3 Horizontal overflow at 375px | **Confirmed** | 459px vs 375px (slightly different numbers, same root cause) |
| #4 Status bar truncated | **Confirmed** | Same behavior |
| #5 Feature badges clipped | **Confirmed** | Same behavior |
| #6 Keyboard shortcut bar overlaps | **Confirmed** | Same behavior |
| #7 No heading elements | **Confirmed** | Still zero headings |
| #8 No aria-current on nav link | **Confirmed** | Still missing |
| #9 Small touch targets | **Confirmed** | Same elements, same sizes |
| #10 Nested buttons in treeitems | **Confirmed** | Same pattern |
| #11 "runtime unknown" raw text | **Confirmed** | Same raw values |
| #12 No visual active state on nav | **Confirmed** | Same styling |
| #13 Generic page title | **Confirmed** | Still "Swarm Dashboard" |
| #14 Console errors from SSE | **Not reproduced** | 0 console errors this session |

**New findings in this audit:**
- #9 Tree role missing aria-label (accessibility)
- #12 Data loading race condition exposing empty state (data presentation)

**Delta analysis:** All prior findings remain open. No fixes applied between audits. The console error improvement (#14 → 0 errors) may be due to orch serve being available during this session or different SSE reconnection timing.

---

## CLI Path Friction (Tool Limitation Documentation)

Per the task requirement, documenting friction from using Playwright CLI commands instead of MCP tools:

### 1. No Interactive Browser Control
**Impact:** High
**Description:** MCP tools allow real-time hover/click/inspect cycles. CLI scripts are fire-and-forget — each interaction requires writing a new script, running it, reading output. Testing hover states required writing explicit hover scripts rather than interactively hovering and observing.
**Workaround:** Batch multiple evaluations into single scripts with JSON output.

### 2. axe-core CDN Injection Blocked
**Impact:** High
**Description:** `page.addScriptTag({ url: 'cdnjs.cloudflare.com/...' })` fails in headless Playwright. The CDN script cannot be loaded, blocking automated WCAG compliance scanning.
**Workaround:** Could bundle axe-core locally and inject from file. Not attempted in this audit.

### 3. Data Loading Race Condition
**Impact:** Medium
**Description:** SSE streams prevent `waitUntil: 'networkidle'`, requiring `domcontentloaded` + explicit waits. First screenshots captured empty state because API data hadn't loaded. MCP tools can wait interactively; CLI scripts must guess wait times.
**Workaround:** Added `waitForSelector('[role="treeitem"]', { timeout: 20000 })` to wait for actual data. Second pass succeeded.

### 4. No Accessibility Snapshot
**Impact:** Medium
**Description:** MCP's `browser_snapshot` provides a rich accessibility tree view. Playwright CLI has `page.accessibility.snapshot()` (deprecated) but no convenient equivalent. Had to use `page.evaluate()` with manual DOM traversal to assess a11y tree.
**Workaround:** Used evaluation scripts to query roles, ARIA attributes, and semantic structure.

### 5. Script Boilerplate Overhead
**Impact:** Low
**Description:** Each evaluation requires a full script with browser launch, context creation, navigation, and cleanup. MCP tools maintain persistent browser state across operations. CLI required writing ~4 separate scripts.
**Workaround:** Batched evaluations into comprehensive scripts to minimize overhead.

### Summary: CLI Path Viability
The CLI path is **viable but slower** (~2x time vs MCP). The biggest gap is axe-core (blocks automated WCAG testing) and interactive exploration (cannot hover-and-inspect in real-time). For structured audits with known checks, CLI scripts work well. For exploratory audits, MCP tools are significantly more productive.

---

## Screenshot Index

| Filename | Viewport | State | Description |
|----------|----------|-------|-------------|
| baseline-1280.png | 1280px | default (data loaded) | Desktop view, light theme, dependency chain grouping, all sections visible |
| baseline-1280-full.png | 1280px | full page | Same, full page scroll |
| baseline-1024.png | 1024px | default | lg breakpoint, event ticker starts truncating |
| baseline-1024-full.png | 1024px | full page | Full page at lg |
| baseline-768.png | 768px | default | md breakpoint, daemon banner visible, tree items with badges |
| baseline-768-full.png | 768px | full page | Full page at md |
| baseline-640.png | 640px | default | sm breakpoint, status bar heavily truncated, badges clipping |
| baseline-640-full.png | 640px | full page | Full page at sm |
| baseline-375.png | 375px | default | Mobile, nav wrapping, heavy truncation |
| baseline-375-full.png | 375px | full page | Full mobile scroll, keyboard bar wrapping to 3 lines |
| interactive-hover-treeitem.png | 1280px | hover | Tree item with hover state at desktop |
| interactive-grouping-dropdown.png | 1280px | element | Grouping select dropdown close-up |

---

## Reproducibility

**Auth:** None required (local dev server, no login)
**Tool:** Playwright 1.58.2 via `npx playwright` / Node.js scripts with `NODE_PATH`
**Server:** Dashboard at localhost:5188 (web UI), orch-go serve at localhost:3348 (API backend)
**Commands:**
```bash
# Screenshot capture
NODE_PATH=/Users/dylanconlin/claude-npm-global/lib/node_modules node audit-screenshots.js

# Evaluation scripts
NODE_PATH=/Users/dylanconlin/claude-npm-global/lib/node_modules node audit-evaluate.js
NODE_PATH=/Users/dylanconlin/claude-npm-global/lib/node_modules node audit-responsive.js
```
**Re-audit schedule:** After responsive fixes are implemented. Recommend running MCP-path audit in parallel for axe-core coverage.

---

## Recommended Next Steps

**Immediate actions (Major findings):**
- [ ] Fix horizontal overflow at 375px mobile viewport (#3)
- [ ] Add heading structure h1-h3 to the page (#7)
- [ ] Add visual active state to current nav link (#13)
- [ ] Make status bar responsive — stack or summarize at narrow viewports (#4)

**Quick wins (Minor findings):**
- [ ] Add `aria-current="page"` to active nav link (#8)
- [ ] Add `aria-label` to tree element (#9)
- [ ] Set page-specific `<title>` (#14)
- [ ] Hide keyboard shortcut bar on mobile (#6)
- [ ] Replace "runtime unknown"/"tokens unknown" with dash placeholder (#11)
- [ ] Add loading skeleton for initial data fetch (#12)

**Investigation needed:**
- [ ] Run axe-core with locally bundled script (CDN blocked in both MCP and CLI paths)
- [ ] Evaluate whether nested button-inside-treeitem pattern is necessary (#10)

**Architecture recommendation:**
These are all UI fixes in `web/src/` (Svelte components). No hotspot files affected. Can be handled by `feature-impl` skill without architect review.
