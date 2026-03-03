# UX Audit: Quick Scan

**TLDR:** ~30-minute surface-level UX health check across all 6 dimensions. Returns top findings with severity classifications and recommended follow-up dimensions.

**When to use:** Regular health check, pre-demo verify, after deploy, before deciding which focused audit to run.

**Output:** Investigation file with top findings across all dimensions, baseline metrics, and recommended next steps.

---

## Quick Reference

### Scan Areas (All 6 Dimensions)

1. **Visual Consistency** — Design tokens, color, typography, spacing
2. **Responsive** — Breakpoint behavior, overflow, touch targets
3. **Accessibility** — a11y tree structure, axe-core WCAG AA scan
4. **Data Presentation** — Number formatting, labels, empty states
5. **Navigation** — Active states, link health, page titles
6. **Interactive States** — Hover feedback, loading states, error handling

### Process (~30 minutes)

1. **Setup & Baseline** (done in Phase 1) — Auth, navigate, 5 viewport screenshots
2. **Surface Scan** (15-20 min) — Quick checks per dimension
3. **Document** (10 min) — Write investigation with findings

---

## Dimension 1: Visual Consistency (Surface Checks)

**Using baseline screenshots and playwright-cli eval:**

- [ ] Background color matches design system (check body background)
- [ ] Card/panel surfaces are distinct from page background
- [ ] Border style is consistent (borders vs shadows — which pattern is used?)
- [ ] Typography is consistent (one font family for UI, one for data/code)
- [ ] Text color hierarchy exists (primary, secondary, muted levels)
- [ ] Accent color used consistently for actions/links
- [ ] Semantic colors used correctly (green=success, amber=warning, red=error)

**Quick evaluation script:**
```javascript
() => {
  const body = getComputedStyle(document.body);
  const cards = document.querySelectorAll('[class*="card"], [class*="rounded"], [class*="panel"]');
  const links = document.querySelectorAll('a');
  return JSON.stringify({
    bodyBackground: body.backgroundColor,
    bodyFont: body.fontFamily,
    cardCount: cards.length,
    cardStyles: Array.from(cards).slice(0, 3).map(c => ({
      borderRadius: getComputedStyle(c).borderRadius,
      boxShadow: getComputedStyle(c).boxShadow,
      border: getComputedStyle(c).border,
      background: getComputedStyle(c).backgroundColor
    })),
    linkColor: links.length > 0 ? getComputedStyle(links[0]).color : 'none',
    linkCount: links.length
  }, null, 2);
}
```

**Record:** Top 3 findings with severity.

---

## Dimension 2: Responsive (Surface Checks)

**Using baseline screenshots (already captured in Phase 1):**

- [ ] Content readable at all 5 viewports (review baseline screenshots)
- [ ] No horizontal overflow at any viewport (check for scrollbars in screenshots)
- [ ] No content truncation that hides critical data
- [ ] Touch targets appear adequate at mobile viewports (375px, 640px)

**Quick overflow check:**
```javascript
() => {
  const body = document.body;
  const html = document.documentElement;
  return JSON.stringify({
    bodyScrollWidth: body.scrollWidth,
    viewportWidth: window.innerWidth,
    hasHorizontalOverflow: body.scrollWidth > window.innerWidth,
    overflowElements: Array.from(document.querySelectorAll('*')).filter(el =>
      el.scrollWidth > el.clientWidth && el.clientWidth > 0
    ).slice(0, 5).map(el => ({
      tag: el.tagName,
      class: el.className.toString().substring(0, 50),
      scrollWidth: el.scrollWidth,
      clientWidth: el.clientWidth
    }))
  }, null, 2);
}
```

**Workflow:** Review the 5 baseline screenshots side by side (mentally). Note any obvious issues.

**Record:** Top 3 findings with severity and affected viewports.

---

## Dimension 3: Accessibility (Surface Checks)

**Using the accessibility snapshot from Phase 1 + axe-core scan:**

- [ ] Page has exactly one h1
- [ ] Heading levels don't skip (no h1 to h3 without h2)
- [ ] All interactive elements have accessible names (from snapshot)
- [ ] Page has landmark regions (main, nav, banner)
- [ ] axe-core WCAG AA: zero critical/serious violations

**axe-core injection and scan:**
```javascript
async () => {
  const script = document.createElement('script');
  script.src = 'https://cdnjs.cloudflare.com/ajax/libs/axe-core/4.11.1/axe.min.js';
  document.head.appendChild(script);
  await new Promise(resolve => {
    script.onload = resolve;
    setTimeout(resolve, 5000);
  });
  const results = await axe.run({
    runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa'] }
  });
  return JSON.stringify({
    violations: results.violations.map(v => ({
      id: v.id,
      impact: v.impact,
      description: v.description,
      nodeCount: v.nodes.length,
      nodes: v.nodes.slice(0, 3).map(n => ({
        target: n.target,
        html: n.html.substring(0, 200),
        failureSummary: n.failureSummary
      }))
    })),
    violationCount: results.violations.length,
    passCount: results.passes.length,
    incompleteCount: results.incomplete.length
  }, null, 2);
}
```

**Record:** axe-core violation count by impact level. Top 3 findings with severity.

---

## Dimension 4: Data Presentation (Surface Checks)

**Using accessibility snapshot (text content) and visual review:**

- [ ] Numbers use appropriate formatting (commas, currency symbols, percentages)
- [ ] No raw database values displayed (snake_case, ALL CAPS, encoded strings)
- [ ] Table headers are clear and descriptive
- [ ] Data alignment: numbers right-aligned, text left-aligned
- [ ] Empty states have helpful messaging (not blank or "null")

**Quick check:** Review the accessibility snapshot for any raw values, missing formatting, or "null"/"undefined" text.

**Record:** Top 3 findings with severity.

---

## Dimension 5: Navigation (Surface Checks)

**Using accessibility snapshot and playwright-cli click:**

- [ ] Current page is indicated in sidebar/nav (active state)
- [ ] Page title reflects current location
- [ ] Browser back button works correctly (navigate back, check page loads)
- [ ] All visible nav links are functional (check for dead hrefs)

**Quick check:** Look at the nav/sidebar in the snapshot. Is the current page highlighted?

**Record:** Top 3 findings with severity.

---

## Dimension 6: Interactive States (Surface Checks)

**Using playwright-cli hover and playwright-cli click:**

- [ ] Buttons have visible hover states (hover over primary buttons)
- [ ] Loading states exist for async operations (if observable)
- [ ] Error states are handled (check console for JS errors)
- [ ] Disabled states are visually distinct (if any disabled elements exist)

**Console error check:**
```
playwright-cli console → level: "error"
```

**Record:** Top 3 findings with severity. Note any JS console errors.

---

## Synthesis (Quick Mode)

After scanning all 6 dimensions:

1. **Compile findings** — Gather all recorded findings (up to ~18 max, 3 per dimension)
2. **Prioritize** — Sort by severity (Blocker > Major > Minor > Cosmetic)
3. **Calculate baseline metrics:**

```markdown
## Baseline Metrics

| Metric | Value |
|--------|-------|
| Total findings | N |
| Blocker | N |
| Major | N |
| Minor | N |
| Cosmetic | N |
| axe-core violations | N |
| axe-core passes | N |
| Console errors | N |
```

4. **Recommend follow-up** — Which dimensions need focused/full audit?

---

## Investigation File Structure (Quick Mode)

```markdown
# Investigation: UX Audit — {Page Name}

**TLDR:** {1-2 sentence summary with finding counts by severity}

**Status:** Complete
**Date:** YYYY-MM-DD
**Beads:** {beads-id}
**Mode:** quick
**Target:** {URL}
**Viewports:** 1280, 1024, 768, 640, 375

---

## Baseline Metrics

| Metric | Value | Prior Audit | Delta |
|--------|-------|-------------|-------|
| Total findings | N | N (or "first audit") | +/-N |
| Blocker | N | N | +/-N |
| Major | N | N | +/-N |
| Minor | N | N | +/-N |
| Cosmetic | N | N | +/-N |
| axe-core violations | N | N | +/-N |
| axe-core passes | N | N | +/-N |
| Console errors | N | N | +/-N |

---

## Findings by Dimension

### Visual Consistency: N findings

#### 1. {Finding Title}
**Severity:** {Blocker/Major/Minor/Cosmetic}
**Viewport(s):** {affected viewports}
**Evidence:** {computed style value, screenshot reference, or snapshot excerpt}
**Impact:** {why this matters to the user}
**Recommendation:** {specific fix}

### Responsive: N findings
...

### Accessibility: N findings
...

### Data Presentation: N findings
...

### Navigation: N findings
...

### Interactive States: N findings
...

---

## What Works Well

- {Positive finding 1 — be specific}
- {Positive finding 2}

---

## Comparison with Prior Audit

{If prior audit exists, compare metrics. If not: "First audit of this page."}

---

## Screenshot Index

| Filename | Viewport | State | Description |
|----------|----------|-------|-------------|
| baseline-1280.png | 1280px | default | {description} |
| baseline-1024.png | 1024px | default | {description} |
| baseline-768.png | 768px | default | {description} |
| baseline-640.png | 640px | default | {description} |
| baseline-375.png | 375px | default | {description} |

---

## Reproducibility

**Auth:** {storageState/dev-login/cdp-tab}
**Commands:** playwright-cli goto, playwright-cli resize({widths}), playwright-cli snapshot, playwright-cli eval(axe-core)
**Re-audit schedule:** {recommended — weekly for active development, monthly for stable pages}

---

## Recommended Next Steps

**Immediate actions:**
- [ ] {Blocker/Major finding fix}

**Focused audits needed:**
- [ ] Run ux-audit focused:{dimension} for {reason}

**Re-scan:** {recommended timing}
```

---

## Anti-Patterns

**Treating quick scan as comprehensive**
Quick scan is triage, not deep analysis. Use focused/full audits for thorough investigation.

**No follow-up action**
Running scan without addressing findings. Always identify at least one fix to make immediately.

**No baseline tracking**
Can't measure improvement without baseline. Re-run periodically and compare metrics.

**Reporting axe-core output verbatim**
axe-core output is raw data. Translate violations into human-readable findings with impact and recommendations.
