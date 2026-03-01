# UX Audit: Work Graph Page

**TLDR:** Work Graph page is functionally solid at desktop widths but has significant UX issues: mobile is unusable (truncated content, no responsive layout), header information density overwhelms, "Ready to Complete" section has fixed-width elements that break on narrow screens, and keyboard shortcuts footer lacks discoverability.

**Status:** Complete

## D.E.K.N. Summary

- **Delta:** Comprehensive UX audit of work graph at 5 viewport widths (390px, 640px, 768px, 860px, 1024px, 1280px)
- **Evidence:** Playwright CLI screenshots at each breakpoint, source code review of +page.svelte and work-graph-tree.svelte
- **Knowledge:** Page uses minimal responsive design (truncation over restructuring), no breakpoint-based layout changes, mobile experience is degraded
- **Next:** Architect review recommended before implementation — this is a hotspot area (view)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/2026-02-14-inv-revive-work-graph-ui-accessible.md | extends | yes | - |
| .kb/models/dashboard-architecture/probes/2026-02-25-probe-dashboard-web-ui-framework-and-responsive-patterns.md | extends | yes | - |

## Question

What are the UX issues with the Work Graph page across desktop and mobile viewports?

## Findings

### Finding 1: Desktop Experience (1280px) — Good

The full-width desktop experience is well-designed:
- Clear visual hierarchy with "Ready to Complete" section at top (green accent)
- Daemon paused banner is prominent with actionable buttons (Close All, Resume)
- Tree structure with dependency arrows (└→, ├→) communicates hierarchy effectively
- Priority badges (P2, P3) are scannable
- Type badges (feature, task, bug) provide quick categorization
- Keyboard shortcuts footer enables power-user workflows

**Issues at desktop:**
- "runtime unknown" / "tokens unknown" repeated across all Ready to Complete items — noise without signal
- Live event strip at 11px mono font is hard to scan, events truncated
- Daemon status line is very dense (slots, last poll, queued, review count, paused state all inline)

### Finding 2: Half-Screen MacBook (860px) — Acceptable

Works adequately at ~860px (half-screen development width):
- All content visible, tree structure intact
- Ready to Complete section remains functional
- Header stats wrap gracefully with `ml-auto` and truncation
- Issue titles truncate cleanly with ellipsis

**Issues:**
- "1 ready to complete" stat wraps to two lines, breaking header alignment
- Feature/task badges pushed to far right, disconnected from their issue titles
- No visual change at this width — just compressed, not restructured

### Finding 3: Tablet / Narrow Window (640px) — Functional but Cramped

At 640px the page still shows all data:
- Tree structure visible with dependency indicators
- Priority badges and issue IDs all present
- Type badges visible (some slightly clipped at right edge)

**Issues:**
- Ready to Complete row: fixed `min-w-[120px]` on ID + `ml-[132px]` on TLDR creates horizontal pressure
- Feature badges clip at viewport right edge
- No padding/margin adjustment for narrow screens
- Keyboard shortcuts footer text wraps to 2 lines

### Finding 4: Mobile (390px / iPhone 12) — Poor

Mobile experience is significantly degraded:
- Issue titles truncated to ~15 characters ("E...", "M...", "I.") — unreadable
- Issue IDs visible but titles are the primary scanning element
- Dependency arrows (└→, ├→) consume valuable horizontal space
- Type badges (feature, task) still rendered at full size, eating into title space
- Ready to Complete section: description, runtime, tokens all invisible — only ID and truncated title
- Keyboard shortcuts footer wraps to 3+ lines, consuming ~15% of viewport height
- No hamburger menu or mobile nav — full nav bar always shown
- Live event strip is completely unreadable at mobile width

### Finding 5: Visual Hierarchy Issues (All Widths)

- **"unassigned" label** appears under some in-progress issues (orange text) — confusing because they ARE assigned (they have running agents). Cross-project issues show "unassigned" because assignee data isn't hydrated cross-repo.
- **"x dead" badge** on issues — red badge without explanation of what "dead" means or what action to take
- **Status circles (empty circles)** — no legend or tooltip explaining what the empty circle means vs the diamond (◆) or play triangle (▶)
- **Group headers** (e.g., "Set review tier in manifest at spawn time (5)") — clickable to collapse, but no visual affordance indicating they're interactive
- **"INDEPENDENT ISSUES"** section — unclear what "independent" means without context (issues with no parent epic)

### Finding 6: Accessibility Concerns

- Color-only status indicators: green (completed), amber (paused), red (dead) — no text/icon fallback for colorblind users
- Priority badges rely on color tinting (P2 has different border color than P3) — subtle differentiation
- Keyboard shortcuts advertised in footer but not through screen reader accessible mechanisms
- Live event strip has no ARIA role or live region announcement
- Dark theme assumed — no explicit light/dark toggle visible (gear icon exists but not obvious)
- Focus indicators on tree rows not visible in screenshots
- "Close" / "Resume" buttons lack descriptive aria-labels

### Finding 7: Information Architecture

- **Daemon status overload:** "Daemon: paused · 2/3 slots · last poll 1 min ago · 188 queued · 4 to review(paused)" — this is 7 data points in one line. Most are irrelevant during normal operation.
- **Dual completion surfaces:** Both the "Ready to Complete" section AND the header "X ready to complete" badge show completion count. The banner also shows "4 completions awaiting review." Three different places showing overlapping information.
- **No filtering/search:** With 16+ issues, no way to search by ID, title, or filter by status/type beyond the GroupBy dropdown
- **No empty state guidance:** "No open issues found" provides no help — no "create an issue" link or explanation

## Test Performed

Playwright CLI screenshots at 6 viewport widths:
- 390x844 (iPhone 12 equivalent)
- 640x720 (sm breakpoint)
- 768x720 (md breakpoint)
- 860x720 (half-screen MacBook)
- 1024x720 (lg breakpoint)
- 1280x720 (standard desktop)

Source code review of `web/src/routes/work-graph/+page.svelte` (1035 lines) and subcomponent analysis.

## Conclusion

The Work Graph page is a functional power-user interface optimized for desktop use. Its primary strengths are the tree visualization with dependency arrows, clear completion workflow with "Ready to Complete" section, and keyboard-driven navigation.

**Critical issues (architect review recommended):**
1. Mobile experience is essentially unusable — titles truncated to illegibility
2. No responsive layout restructuring at any breakpoint — just truncation/compression
3. Information density in header is overwhelming
4. Accessibility gaps: color-only indicators, no ARIA live regions, poor focus visibility

**Severity Summary:**

| Severity | Count | Examples |
|----------|-------|---------|
| Critical | 1 | Mobile titles truncated to 1-2 chars |
| High | 3 | No responsive restructuring, header info overload, accessibility gaps |
| Medium | 4 | "unassigned" label confusion, "dead" badge unexplained, no search/filter, footer wrapping |
| Low | 3 | "runtime unknown" noise, dual completion surfaces, group header affordance |

**Recommendation:** Route through architect before implementation. The responsive design issues require structural decisions about what to show/hide at narrow widths, not just CSS tweaks.
