# Dimension: Accessibility (Full Depth)

**TLDR:** Comprehensive accessibility audit combining structural review (a11y tree via snapshot), automated WCAG testing (axe-core injection), and keyboard navigation testing. Produces violation counts, structural completeness assessment, and keyboard navigability rating.

**When to use:** Focused accessibility audit, full audit accessibility dimension, or follow-up from quick scan a11y findings.

**Duration:** 30-60 minutes

**Primary tools:** `playwright-cli snapshot`, `playwright-cli eval` (axe-core), `playwright-cli press`, `playwright-cli click`

---

## Accessibility Audit Workflow

Four stages, executed sequentially:

1. **Structural Review** — a11y tree analysis via snapshot
2. **axe-core WCAG Scan** — automated standards compliance
3. **Keyboard Navigation** — focus order, traps, skip links
4. **Synthesis** — combine results into accessibility score

---

## Stage 1: Structural Review (via playwright-cli snapshot)

**Take an accessibility snapshot at 1280px and analyze:**

### Heading Hierarchy

- [ ] Page has exactly ONE h1
- [ ] Headings don't skip levels (h1 → h2 → h3, not h1 → h3)
- [ ] Headings describe section content (not generic "Section 1")
- [ ] Heading hierarchy creates a logical document outline

**How to check:** In the snapshot output, look for heading elements. Map the hierarchy:
```
h1: "Page Title"
  h2: "Section A"
    h3: "Subsection A.1"
  h2: "Section B"
```

**Findings for heading issues:**
- Missing h1 → Major (page has no primary heading)
- Skipped levels → Minor (confusing for screen reader users)
- Multiple h1s → Minor (unclear page structure)

### Landmark Regions

- [ ] Page has `main` landmark (the primary content area)
- [ ] Page has `navigation` landmark (sidebar or nav bar)
- [ ] Page has `banner` landmark (header area)
- [ ] Landmarks don't nest improperly

**Expected landmarks for toolshed pages:**
```
banner (header with logo + user menu)
navigation (sidebar)
main (content area)
contentinfo (footer, if present)
```

**Findings for landmark issues:**
- Missing `main` → Major (screen readers can't find content)
- Missing `navigation` → Minor (screen readers can still navigate)
- Duplicate landmarks without labels → Minor (ambiguous navigation)

### Interactive Element Labels

- [ ] All buttons have accessible names (text content or aria-label)
- [ ] All links have descriptive text (not "click here" or bare URLs)
- [ ] All form inputs have associated labels
- [ ] Icon-only buttons have aria-label or title
- [ ] Images have alt text (or are decorative with alt="")

**How to check:** In the snapshot, look for interactive elements. Elements listed as `button ""` or `link ""` are missing accessible names.

**Findings for label issues:**
- Unlabeled button/link → Major (inaccessible to screen readers)
- Generic link text ("click here") → Minor (unhelpful for screen reader navigation)
- Missing form labels → Major (form unusable for screen reader users)

### ARIA State Management

- [ ] Expandable elements have `aria-expanded` attribute
- [ ] Selected items have `aria-selected` or `aria-current`
- [ ] Disabled elements have `aria-disabled` or `disabled` attribute
- [ ] Loading states have `aria-busy` or equivalent
- [ ] Dynamic regions have `aria-live` (polite or assertive)

---

## Stage 2: axe-core WCAG Scan

### Injection Script

```javascript
async () => {
  // Check if axe-core is already loaded
  if (typeof axe !== 'undefined') {
    const results = await axe.run({
      runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa'] }
    });
    return JSON.stringify(formatResults(results), null, 2);
  }

  // Inject axe-core from CDN
  const script = document.createElement('script');
  script.src = 'https://cdnjs.cloudflare.com/ajax/libs/axe-core/4.11.1/axe.min.js';
  document.head.appendChild(script);

  await new Promise((resolve, reject) => {
    script.onload = resolve;
    script.onerror = () => reject(new Error('Failed to load axe-core from CDN'));
    setTimeout(() => reject(new Error('axe-core load timeout')), 10000);
  });

  // Run WCAG AA scan
  const results = await axe.run({
    runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa'] }
  });

  function formatResults(r) {
    return {
      summary: {
        violationCount: r.violations.length,
        passCount: r.passes.length,
        incompleteCount: r.incomplete.length,
        inapplicableCount: r.inapplicable.length
      },
      violations: r.violations.map(v => ({
        id: v.id,
        impact: v.impact,
        description: v.description,
        helpUrl: v.helpUrl,
        nodeCount: v.nodes.length,
        tags: v.tags.filter(t => t.startsWith('wcag')),
        nodes: v.nodes.slice(0, 5).map(n => ({
          target: n.target,
          html: n.html.substring(0, 300),
          failureSummary: n.failureSummary
        }))
      })),
      incomplete: r.incomplete.map(i => ({
        id: i.id,
        impact: i.impact,
        description: i.description,
        nodeCount: i.nodes.length
      }))
    };
  }

  return JSON.stringify(formatResults(results), null, 2);
}
```

### Interpreting axe-core Results

**Impact levels map to audit severity:**

| axe-core Impact | Audit Severity | Meaning |
|----------------|---------------|---------|
| critical | **Blocker** | Users with disabilities cannot use the page |
| serious | **Major** | Significant barrier for assistive technology users |
| moderate | **Minor** | Noticeable issue, workaround usually exists |
| minor | **Cosmetic** | Best practice, minimal real-world impact |

### Common axe-core Violations and Fixes

| Violation ID | Description | Typical Fix |
|-------------|-------------|-------------|
| `color-contrast` | Text doesn't meet WCAG AA contrast ratio | Increase text color darkness or background lightness |
| `image-alt` | Image missing alt text | Add `alt="description"` or `alt=""` for decorative |
| `button-name` | Button has no accessible name | Add text content or `aria-label` |
| `link-name` | Link has no accessible name | Add descriptive text content |
| `label` | Form input missing label | Add `<label>` element or `aria-label` |
| `heading-order` | Heading levels skip (h1→h3) | Fix heading hierarchy |
| `landmark-one-main` | Page lacks `main` landmark | Add `<main>` element |
| `region` | Content not in landmark region | Wrap in appropriate landmark |
| `aria-allowed-attr` | Invalid ARIA attribute | Remove or fix ARIA attribute |
| `aria-valid-attr-value` | ARIA attribute has invalid value | Correct the attribute value |

### If axe-core Injection Fails

**CSP might block CDN scripts.** Fallback approach:

1. Check console for CSP errors: `playwright-cli console → level: "error"`
2. If CSP blocks:
   - Document: "axe-core CDN blocked by CSP — using structural review only"
   - Rely on Stage 1 (structural review) for accessibility findings
   - Note in investigation: "Automated WCAG scan not available; manual structural review only"
3. Report: `bd comment <beads-id> "CONSTRAINT: axe-core CDN blocked by CSP - structural review only"`

---

## Stage 3: Keyboard Navigation Testing

**Purpose:** Verify the page is fully usable without a mouse.

### Focus Order Test

1. Start at the top of the page
2. Press `Tab` repeatedly, tracking focus order
3. Verify focus moves in a logical order (top-to-bottom, left-to-right)
4. Verify focus is VISIBLE (focus ring or outline on focused element)

**How to test with playwright-cli:**
```
playwright-cli press → key: "Tab"     (advance focus)
playwright-cli snapshot                    (check which element has focus)
```

Repeat Tab + snapshot cycle through major interactive elements (don't need to Tab through every element — 10-15 Tabs is sufficient for a focused audit).

### Checklist

- [ ] **Focus visible:** Every focused element has a visible indicator (outline, ring, highlight)
- [ ] **Focus order logical:** Tab moves through elements in reading order
- [ ] **Skip-to-content link:** First Tab press reaches a "skip to main content" link (or similar)
- [ ] **No focus traps:** Tab doesn't get stuck in any element (can always Tab out)
- [ ] **Enter/Space activates:** Buttons respond to Enter/Space key
- [ ] **Escape closes overlays:** If modals/dropdowns exist, Escape dismisses them

### Modal/Overlay Testing (if applicable)

If the page has modals, overlays, or dropdowns:

1. Open the modal/overlay via keyboard (Enter/Space on trigger)
2. Verify focus moves INTO the modal
3. Verify Tab stays within the modal (focus trap)
4. Verify Escape closes the modal
5. Verify focus returns to the trigger element after close

**Finding template for keyboard issues:**
- No skip-to-content link → Minor (WCAG 2.4.1)
- Focus not visible → Major (WCAG 2.4.7)
- Focus order illogical → Minor (WCAG 2.4.3)
- Focus trapped in element → Blocker (keyboard user stuck)
- Modal doesn't trap focus → Minor (focus escapes to background)

---

## Stage 4: Contrast Spot Check

**axe-core covers most contrast issues, but verify key elements manually:**

```javascript
() => {
  function getContrastRatio(fg, bg) {
    function luminance(r, g, b) {
      const a = [r, g, b].map(v => {
        v /= 255;
        return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
      });
      return a[0] * 0.2126 + a[1] * 0.7152 + a[2] * 0.0722;
    }
    function parseColor(str) {
      const m = str.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
      return m ? [parseInt(m[1]), parseInt(m[2]), parseInt(m[3])] : [0, 0, 0];
    }
    const fgRGB = parseColor(fg);
    const bgRGB = parseColor(bg);
    const l1 = luminance(...fgRGB) + 0.05;
    const l2 = luminance(...bgRGB) + 0.05;
    return Math.round((Math.max(l1, l2) / Math.min(l1, l2)) * 100) / 100;
  }

  const samples = [];
  // Check body text
  const body = document.querySelector('main') || document.body;
  const bodyStyle = getComputedStyle(body);
  const bodyP = body.querySelector('p, span, td, div');
  if (bodyP) {
    const s = getComputedStyle(bodyP);
    samples.push({
      element: 'body text',
      color: s.color,
      background: s.backgroundColor || bodyStyle.backgroundColor,
      fontSize: s.fontSize,
      ratio: getContrastRatio(s.color, s.backgroundColor || bodyStyle.backgroundColor),
      required: parseFloat(s.fontSize) >= 18.66 ? 3.0 : 4.5
    });
  }
  // Check headings
  const h1 = document.querySelector('h1');
  if (h1) {
    const s = getComputedStyle(h1);
    samples.push({
      element: 'h1',
      color: s.color,
      background: s.backgroundColor,
      fontSize: s.fontSize,
      ratio: getContrastRatio(s.color, s.backgroundColor),
      required: parseFloat(s.fontSize) >= 18.66 ? 3.0 : 4.5
    });
  }
  // Check muted text (often fails contrast)
  const muted = document.querySelector('[class*="muted"], [class*="secondary"], [class*="text-gray"]');
  if (muted) {
    const s = getComputedStyle(muted);
    samples.push({
      element: 'muted text',
      color: s.color,
      background: s.backgroundColor,
      fontSize: s.fontSize,
      ratio: getContrastRatio(s.color, s.backgroundColor),
      required: parseFloat(s.fontSize) >= 18.66 ? 3.0 : 4.5
    });
  }

  return JSON.stringify({ contrastSamples: samples }, null, 2);
}
```

**WCAG AA contrast requirements:**
- Normal text (<18.66px or <14px bold): 4.5:1 minimum
- Large text (≥18.66px or ≥14px bold): 3.0:1 minimum
- UI components and graphical objects: 3.0:1 minimum

---

## Severity Guide (Accessibility)

| Finding | Severity | WCAG Reference |
|---------|----------|----------------|
| axe-core critical violation | **Blocker** | Various |
| No main landmark | **Major** | WCAG 1.3.1 |
| Unlabeled interactive elements | **Major** | WCAG 4.1.2 |
| Missing form labels | **Major** | WCAG 1.3.1 |
| Focus not visible | **Major** | WCAG 2.4.7 |
| Focus trapped (can't Tab out) | **Blocker** | WCAG 2.1.2 |
| Color contrast fails AA | **Major** | WCAG 1.4.3 |
| axe-core serious violation | **Major** | Various |
| Heading levels skip | **Minor** | WCAG 1.3.1 |
| Multiple h1 elements | **Minor** | WCAG 1.3.1 |
| Missing skip-to-content link | **Minor** | WCAG 2.4.1 |
| axe-core moderate violation | **Minor** | Various |
| Generic link text | **Minor** | WCAG 2.4.4 |
| axe-core minor violation | **Cosmetic** | Various |
| Missing aria-live on dynamic content | **Cosmetic** | WCAG 4.1.3 |

---

## Finding Template (Accessibility)

```markdown
#### {N}. {Finding Title}
**Severity:** {Blocker/Major/Minor/Cosmetic}
**WCAG:** {criterion reference, e.g., "1.4.3 Contrast (Minimum)"}
**Source:** {axe-core / structural review / keyboard test}
**Evidence:** {axe-core violation JSON, snapshot excerpt, or keyboard test result}
**Impact:** {how this affects users with disabilities}
**Recommendation:** {specific fix with code example if applicable}
```

---

## Accessibility Metrics (for Baseline)

Include these metrics in the investigation file:

```markdown
## Accessibility Metrics

| Metric | Value |
|--------|-------|
| axe-core violations (total) | N |
| axe-core critical | N |
| axe-core serious | N |
| axe-core moderate | N |
| axe-core minor | N |
| axe-core passes | N |
| axe-core incomplete (needs review) | N |
| Heading levels used | h1-hN |
| Landmark regions | N (list them) |
| Unlabeled interactive elements | N |
| Keyboard navigable | yes / partial / no |
| Skip-to-content link | present / absent |
| Focus visibility | visible / partial / invisible |
```

---

## Anti-Patterns

**Running only axe-core and calling it done**
axe-core detects ~30-40% of WCAG issues. Structural review and keyboard testing catch what automation misses. All three stages are required for a full accessibility audit.

**Reporting axe-core violations without interpretation**
Raw axe-core output is data, not findings. Each violation needs impact assessment and a specific recommendation. "color-contrast violation on 12 elements" should become "Muted text (#94A3B8) on white background fails AA contrast (3.2:1, needs 4.5:1)."

**Skipping keyboard navigation because "most users use a mouse"**
Keyboard accessibility is a legal and ethical requirement, not an optional feature. Every Major interactive element must be reachable and operable via keyboard.

**Treating all axe-core violations as equal**
Impact levels matter. A critical violation (page completely inaccessible) is far more important than a minor violation (best practice). Triage by impact.
