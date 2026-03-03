# Dimension: Responsive (Full Depth)

**TLDR:** Comprehensive responsive audit evaluating breakpoint behavior, layout reflow, content accessibility, and touch targets across all 5 standard viewports. Enforces CLAUDE.md breakpoint rules.

**When to use:** Focused responsive audit, full audit responsive dimension, or follow-up from quick scan responsive findings.

**Duration:** 30-60 minutes

**Primary tools:** `playwright-cli resize`, `playwright-cli snapshot`, `playwright-cli screenshot`, `playwright-cli eval`

---

## CLAUDE.md Breakpoint Rules (Non-Negotiable)

These rules are project constraints, not suggestions:

| Width | Breakpoint | Rule |
|-------|-----------|------|
| 640px | sm | **First structural shift** — sidebar collapse, grid column changes, nav collapse |
| 768px | md | **Minor tweaks ONLY** — spacing, font sizes. NEVER structural layout changes |
| 1024px | lg | **Full desktop expansion** — wider gutters, multi-column grids |

**Violation of these rules is a Major finding.** If structural layout changes occur at 768px (md), that is a breakpoint rule violation.

---

## Responsive Audit Workflow

### Step 1: Viewport Sweep (Systematic)

Test each viewport sequentially. At each viewport:

1. **Resize** the browser
2. **Take screenshot** for visual evidence
3. **Take snapshot** (a11y tree) to detect element visibility changes
4. **Run overflow check** via `playwright-cli eval`
5. **Document findings** before moving to next viewport

**Order:** 1280px → 1024px → 768px → 640px → 375px

---

### Step 2: At Each Viewport — Check These

#### Layout Structure

- [ ] Content is readable (text not cut off, not overlapping)
- [ ] No horizontal overflow (no unwanted horizontal scrollbar)
- [ ] Layout uses available width appropriately (no excessive whitespace, no cramping)
- [ ] Cards/panels stack or reflow sensibly
- [ ] Sidebar behavior is appropriate (visible at desktop, collapsed/hidden at mobile)

#### Content Visibility

- [ ] No content hidden without an escape hatch (e.g., truncated text with no "show more")
- [ ] Data tables are scrollable or adapted (not cut off)
- [ ] Charts/visualizations resize proportionally
- [ ] Images don't overflow their containers

#### Interactive Elements

- [ ] All buttons and controls are reachable
- [ ] Touch targets ≥44px at mobile viewports (375px, 640px) per WCAG 2.5.5
- [ ] Form inputs are usable (not too small, labels visible)
- [ ] Dropdowns/modals fit within viewport

#### Typography

- [ ] Text doesn't break awkwardly (orphan words on narrow lines)
- [ ] Line length is comfortable (45-75 characters for body text)
- [ ] Font sizes remain readable (≥14px for body text on mobile)

---

### Step 3: Breakpoint Transition Analysis

**Compare snapshots between adjacent viewports to detect structural changes:**

#### 1280px → 1024px (lg transition)

- [ ] Full desktop layout maintained or gracefully simplified
- [ ] Gutters may narrow, but structure should hold
- **Expected:** Minor adjustments, grid still multi-column

#### 1024px → 768px (lg → md transition)

- [ ] **CRITICAL:** Only MINOR changes allowed (spacing, font sizes)
- [ ] **VIOLATION if:** sidebar collapses, grid columns change, nav restructures
- [ ] Compare snapshot element trees — same structural elements should be present
- **Expected:** Same layout, slightly tighter spacing

#### 768px → 640px (md → sm transition)

- [ ] **This is where structural shift SHOULD occur**
- [ ] Sidebar collapses or becomes off-canvas
- [ ] Multi-column grids become single-column or two-column
- [ ] Navigation collapses to hamburger/mobile pattern
- **Expected:** First major layout restructure

#### 640px → 375px (sm → mobile)

- [ ] Mobile layout is fully committed (single column, stacked elements)
- [ ] Touch targets are appropriately sized
- [ ] Content priority is correct (most important content visible first)
- **Expected:** Refined mobile layout, no further structural shifts

---

### Step 4: Overflow Detection Script

Run at each viewport to detect elements that overflow their containers:

```javascript
() => {
  const body = document.body;
  const results = {
    viewport: {
      width: window.innerWidth,
      height: window.innerHeight
    },
    pageOverflow: {
      hasHorizontalOverflow: body.scrollWidth > window.innerWidth,
      bodyScrollWidth: body.scrollWidth,
      documentScrollWidth: document.documentElement.scrollWidth
    },
    overflowElements: [],
    smallTouchTargets: []
  };

  // Find elements with horizontal overflow
  document.querySelectorAll('*').forEach(el => {
    if (el.scrollWidth > el.clientWidth + 1 && el.clientWidth > 0) {
      const rect = el.getBoundingClientRect();
      if (rect.width > 0 && rect.height > 0) {
        results.overflowElements.push({
          tag: el.tagName,
          class: el.className.toString().substring(0, 60),
          scrollWidth: el.scrollWidth,
          clientWidth: el.clientWidth,
          overflow: getComputedStyle(el).overflow
        });
      }
    }
  });
  results.overflowElements = results.overflowElements.slice(0, 10);

  // Find interactive elements with small touch targets (< 44px)
  const interactive = document.querySelectorAll('a, button, input, select, textarea, [role="button"], [tabindex]');
  interactive.forEach(el => {
    const rect = el.getBoundingClientRect();
    if (rect.width > 0 && rect.height > 0 && (rect.width < 44 || rect.height < 44)) {
      results.smallTouchTargets.push({
        tag: el.tagName,
        text: (el.textContent || el.getAttribute('aria-label') || '').substring(0, 40),
        width: Math.round(rect.width),
        height: Math.round(rect.height)
      });
    }
  });
  results.smallTouchTargets = results.smallTouchTargets.slice(0, 10);

  return JSON.stringify(results, null, 2);
}
```

---

### Step 5: Table Responsiveness

If the page has data tables, verify their behavior:

```javascript
() => {
  const tables = document.querySelectorAll('table');
  return JSON.stringify({
    tableCount: tables.length,
    tables: Array.from(tables).slice(0, 5).map(t => {
      const rect = t.getBoundingClientRect();
      const parent = t.parentElement;
      const parentRect = parent.getBoundingClientRect();
      return {
        columns: t.rows[0] ? t.rows[0].cells.length : 0,
        rows: t.rows.length,
        tableWidth: Math.round(rect.width),
        parentWidth: Math.round(parentRect.width),
        overflows: rect.width > parentRect.width,
        parentOverflow: getComputedStyle(parent).overflowX,
        parentClass: parent.className.toString().substring(0, 60)
      };
    })
  }, null, 2);
}
```

**Expected behavior for tables on narrow viewports:**
- Wrapped in a scrollable container (`overflow-x: auto`)
- OR adapted layout (card-based, stacked rows)
- NOT truncated or clipped without scroll

---

## Severity Guide (Responsive)

| Finding | Severity | Rationale |
|---------|----------|-----------|
| Page is unusable at a viewport (content hidden, controls unreachable) | **Blocker** | User cannot complete task |
| Structural layout change at 768px (md) instead of 640px (sm) | **Major** | Violates CLAUDE.md breakpoint rules |
| Horizontal overflow creates scrollbar at mobile viewports | **Major** | Significant friction — content partially hidden |
| Touch targets <44px at mobile | **Major** | Accessibility barrier (WCAG 2.5.5) |
| Content truncation hiding data without escape hatch | **Major** | User loses data context |
| Table overflows without horizontal scroll container | **Minor** | User can still access data by scrolling page |
| Awkward text wrapping or orphan words | **Minor** | Readability issue, not functional |
| Excessive whitespace at a viewport (wasted space) | **Minor** | Suboptimal use of space |
| Slightly tight spacing at one viewport | **Cosmetic** | Noticeable but not impactful |
| 1-2px alignment difference between viewports | **Cosmetic** | Visual polish |

---

## Finding Template (Responsive)

```markdown
#### {N}. {Finding Title}
**Severity:** {Blocker/Major/Minor/Cosmetic}
**Viewport(s):** {affected viewport widths}
**Screenshot:** `{relative path to screenshot}`
**Evidence:** {overflow dimensions, snapshot diff, computed style values}
**Impact:** {what happens to the user at this viewport}
**Recommendation:** {specific CSS/layout fix}
```

---

## Anti-Patterns

**Testing only extreme viewports (1280 and 375)**
The interesting bugs happen at transition points (1024→768, 768→640). Test all 5 viewports.

**Reporting "looks fine" without evidence**
Capture overflow check script output and compare snapshots. "Looks fine" is not evidence.

**Ignoring the 768px rule**
768px (md) is the most common violation point. Half-screen MacBook Pro lands here. Structural changes at 768px are always Major severity.

**Assuming tables handle themselves**
Tables are the #1 responsive problem. Always run the table check script on data-heavy pages.
