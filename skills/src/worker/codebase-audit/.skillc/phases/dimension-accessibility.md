# Codebase Audit: Accessibility

**TLDR:** Accessibility-focused audit identifying WCAG violations, missing semantic HTML, keyboard traps, screen reader gaps, and color contrast failures across web UI code.

**When to use:** Accessibility review needed, WCAG compliance check, user reports of assistive technology issues, pre-launch a11y audit, or after significant UI changes.

**Output:** Investigation file with accessibility findings rated by severity (Critical/High/Medium/Low) mapped to WCAG success criteria, with remediation steps.

---

## Focus Areas

1. **Semantic HTML** - Missing landmarks, incorrect heading hierarchy, divs-as-buttons, non-semantic elements for interactive controls
2. **Keyboard Navigation** - Focus traps, missing focus indicators, non-focusable interactive elements, tab order issues
3. **Screen Reader Support** - Missing ARIA labels, live regions for dynamic content, form input associations, image alt text
4. **Color & Contrast** - Insufficient text contrast (4.5:1), UI component contrast (3:1), color-only information conveyance
5. **Forms & Inputs** - Missing labels, error announcements, required field indicators, autocomplete attributes
6. **Dynamic Content** - Modals without focus management, toast/notification announcements, loading state communication, route change announcements (SPA)
7. **Media & Images** - Missing alt text, decorative image handling, video captions, audio descriptions

---

## Pattern Search Commands

```bash
# Buttons implemented as divs/spans (should be <button> or <a>)
rg "onClick|on:click" --type svelte --type jsx --type tsx -C 1 | rg "<(div|span|li)"

# Images missing alt text
rg "<img " --glob "*.{svelte,jsx,tsx,html}" | rg -v "alt="

# Interactive elements without ARIA labels
rg "role=\"button\"|role=\"link\"" --glob "*.{svelte,jsx,tsx}" | rg -v "aria-label"

# Form inputs without associated labels
rg "<input|<select|<textarea" --glob "*.{svelte,jsx,tsx,html}" -C 2 | rg -v "label|aria-label|aria-labelledby"

# Hardcoded color values (potential contrast issues)
rg "color:\s*#[0-9a-fA-F]{3,6}" --glob "*.{css,svelte,tsx}" | rg -v "\/\/"

# Mouse-only event handlers (missing keyboard equivalents)
rg "onMouseDown|onMouseUp|on:mousedown|on:mouseup" --glob "*.{svelte,jsx,tsx}" -l

# Focus management in modals/dialogs
rg "dialog|modal|drawer" --glob "*.{svelte,jsx,tsx}" -l

# tabIndex misuse (positive values break natural tab order)
rg "tabIndex=\"[1-9]|tabindex=\"[1-9]" --glob "*.{svelte,jsx,tsx,html}"

# Heading hierarchy (find all heading levels to check order)
rg "<h[1-6]|<Heading" --glob "*.{svelte,jsx,tsx,html}" -n

# ARIA live regions for dynamic content
rg "aria-live|role=\"alert\"|role=\"status\"" --glob "*.{svelte,jsx,tsx}"

# Route change announcements (SPA-specific)
rg "announce|aria-live.*route|visually-hidden.*navigat" --glob "*.{svelte,jsx,tsx}" -i

# Autofocus usage (can disorient screen reader users)
rg "autofocus|autoFocus" --glob "*.{svelte,jsx,tsx,html}"
```

---

## WCAG Mapping

Map findings to WCAG 2.1 Level AA success criteria for actionable reporting:

| Focus Area | Key Success Criteria |
|------------|---------------------|
| Semantic HTML | 1.3.1 Info and Relationships, 4.1.2 Name/Role/Value |
| Keyboard | 2.1.1 Keyboard, 2.1.2 No Keyboard Trap, 2.4.7 Focus Visible |
| Screen Reader | 1.1.1 Non-text Content, 4.1.2 Name/Role/Value, 4.1.3 Status Messages |
| Color & Contrast | 1.4.3 Contrast (Minimum), 1.4.11 Non-text Contrast, 1.4.1 Use of Color |
| Forms | 1.3.5 Identify Input Purpose, 3.3.1 Error Identification, 3.3.2 Labels or Instructions |
| Dynamic Content | 4.1.3 Status Messages, 2.4.3 Focus Order, 3.2.1 On Focus |
| Media | 1.1.1 Non-text Content, 1.2.1 Audio-only/Video-only, 1.2.2 Captions |

---

## Severity Classification

| Severity | Definition | Examples |
|----------|------------|---------|
| **Critical** | Blocks assistive technology users entirely | Keyboard trap, missing form labels on login, no skip navigation |
| **High** | Major functionality inaccessible | Buttons as divs (no keyboard/SR), missing alt on functional images, no focus management in modals |
| **Medium** | Degraded experience for AT users | Incorrect heading hierarchy, missing live regions, low contrast on secondary text |
| **Low** | Minor improvement opportunity | Decorative images with redundant alt, tab order could be improved, missing autocomplete attributes |

---

## Workflow

### Phase 1: Automated Scan (20-30 minutes)

Run pattern search commands above. Capture counts and specific instances.

For projects with a web UI, also check:
```bash
# Count interactive elements to scope the audit
rg "<button|<a |<input|<select|role=\"button\"" --glob "*.{svelte,jsx,tsx}" | wc -l

# Check if any a11y testing exists
rg "axe|a11y|accessibility|getByRole|getByLabelText" --glob "*.{test,spec}.*" | wc -l

# Check for existing ARIA usage (baseline)
rg "aria-" --glob "*.{svelte,jsx,tsx}" | wc -l
```

### Phase 2: Manual Review (20-30 minutes)

Focus on patterns automated search can't catch:
- **Keyboard walkthrough** of key user flows (if running server available)
- **Heading hierarchy** analysis (are levels sequential? h1 → h2 → h3)
- **Focus order** review (does tab order match visual order?)
- **Error state review** (are errors announced to screen readers?)
- **Dynamic content** review (are updates communicated?)

### Phase 3: Evidence Collection (15-20 minutes)

For each finding, document:
- **File:line** reference
- **WCAG criterion** violated
- **Severity** (Critical/High/Medium/Low)
- **Remediation** (specific fix, not "add accessibility")

### Phase 4: Documentation (15 minutes)

Write investigation file with:
- Findings grouped by focus area
- WCAG criteria mapped to each finding
- Severity-prioritized remediation plan
- Baseline metrics for re-audit comparison

---

## Baseline Metrics

Capture these for re-audit comparison:

```markdown
## Baseline
- Total interactive elements: [count]
- Elements with ARIA labels: [count] ([%])
- Images with alt text: [count]/[total] ([%])
- Form inputs with labels: [count]/[total] ([%])
- Existing a11y tests: [count]
- Heading hierarchy violations: [count]
- Keyboard-inaccessible controls: [count]
```

---

## Anti-Patterns

**Reporting "add ARIA" without context**
- "Add aria-label to buttons" (which buttons? why?)
- Fix: Specific file:line, what the label should convey, which WCAG criterion

**Over-ARIA (ARIA as a band-aid)**
- Adding `role="button"` to a `<div>` instead of using `<button>`
- Fix: Recommend semantic HTML first, ARIA only when native elements can't express the semantics

**Ignoring keyboard navigation**
- Checking only screen reader attributes, not actual keyboard operability
- Fix: Verify both keyboard access AND programmatic semantics

**Color contrast without tooling**
- Guessing contrast ratios instead of measuring
- Fix: Reference specific hex values, recommend checking with contrast tools

---

*This dimension enables systematic, WCAG-mapped accessibility audits that produce actionable findings with clear remediation paths and baseline metrics for tracking improvement.*
