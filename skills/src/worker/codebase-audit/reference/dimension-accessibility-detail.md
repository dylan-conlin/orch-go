# Accessibility Audit (Detailed Reference)

## Pattern Search Commands

```bash
# Buttons implemented as divs/spans
rg "onClick|on:click" --type svelte --type jsx --type tsx -C 1 | rg "<(div|span|li)"

# Images missing alt text
rg "<img " --glob "*.{svelte,jsx,tsx,html}" | rg -v "alt="

# Interactive elements without ARIA labels
rg "role=\"button\"|role=\"link\"" --glob "*.{svelte,jsx,tsx}" | rg -v "aria-label"

# Form inputs without labels
rg "<input|<select|<textarea" --glob "*.{svelte,jsx,tsx,html}" -C 2 | rg -v "label|aria-label|aria-labelledby"

# Hardcoded color values
rg "color:\s*#[0-9a-fA-F]{3,6}" --glob "*.{css,svelte,tsx}" | rg -v "\/\/"

# Mouse-only event handlers
rg "onMouseDown|onMouseUp|on:mousedown|on:mouseup" --glob "*.{svelte,jsx,tsx}" -l

# Focus management in modals
rg "dialog|modal|drawer" --glob "*.{svelte,jsx,tsx}" -l

# tabIndex misuse
rg "tabIndex=\"[1-9]|tabindex=\"[1-9]" --glob "*.{svelte,jsx,tsx,html}"

# Heading hierarchy
rg "<h[1-6]|<Heading" --glob "*.{svelte,jsx,tsx,html}" -n

# ARIA live regions
rg "aria-live|role=\"alert\"|role=\"status\"" --glob "*.{svelte,jsx,tsx}"

# Route change announcements (SPA)
rg "announce|aria-live.*route|visually-hidden.*navigat" --glob "*.{svelte,jsx,tsx}" -i

# Autofocus usage
rg "autofocus|autoFocus" --glob "*.{svelte,jsx,tsx,html}"
```

## WCAG 2.1 Level AA Mapping

| Focus Area | Key Success Criteria |
|------------|---------------------|
| Semantic HTML | 1.3.1 Info and Relationships, 4.1.2 Name/Role/Value |
| Keyboard | 2.1.1 Keyboard, 2.1.2 No Keyboard Trap, 2.4.7 Focus Visible |
| Screen Reader | 1.1.1 Non-text Content, 4.1.2 Name/Role/Value, 4.1.3 Status Messages |
| Color & Contrast | 1.4.3 Contrast (Minimum), 1.4.11 Non-text Contrast, 1.4.1 Use of Color |
| Forms | 1.3.5 Identify Input Purpose, 3.3.1 Error Identification, 3.3.2 Labels or Instructions |
| Dynamic Content | 4.1.3 Status Messages, 2.4.3 Focus Order, 3.2.1 On Focus |
| Media | 1.1.1 Non-text Content, 1.2.1 Audio-only/Video-only, 1.2.2 Captions |

## Severity Classification

| Severity | Definition | Examples |
|----------|------------|---------|
| **Critical** | Blocks AT users entirely | Keyboard trap, missing form labels on login, no skip navigation |
| **High** | Major functionality inaccessible | Buttons as divs, missing alt on functional images, no focus in modals |
| **Medium** | Degraded experience for AT users | Incorrect heading hierarchy, missing live regions, low contrast |
| **Low** | Minor improvement | Decorative images with redundant alt, tab order improvements |

## Workflow

### Phase 1: Automated Scan (20-30 min)
Run pattern search commands. Also check:
```bash
rg "<button|<a |<input|<select|role=\"button\"" --glob "*.{svelte,jsx,tsx}" | wc -l
rg "axe|a11y|accessibility|getByRole|getByLabelText" --glob "*.{test,spec}.*" | wc -l
rg "aria-" --glob "*.{svelte,jsx,tsx}" | wc -l
```

### Phase 2: Manual Review (20-30 min)
- Keyboard walkthrough of key user flows
- Heading hierarchy analysis
- Focus order review
- Error state review
- Dynamic content review

### Phase 3: Evidence Collection (15-20 min)
Document file:line, WCAG criterion, severity, remediation per finding.

### Phase 4: Documentation (15 min)
Write investigation with findings grouped by focus area, WCAG mapped, severity-prioritized.

## Baseline Metrics

```markdown
- Total interactive elements: [count]
- Elements with ARIA labels: [count] ([%])
- Images with alt text: [count]/[total] ([%])
- Form inputs with labels: [count]/[total] ([%])
- Existing a11y tests: [count]
- Heading hierarchy violations: [count]
- Keyboard-inaccessible controls: [count]
```

## Anti-Patterns

- Reporting "add ARIA" without context → specify file:line, what label should convey, which WCAG criterion
- Over-ARIA (`role="button"` on div instead of using `<button>`) → recommend semantic HTML first
- Ignoring keyboard navigation → verify both keyboard access AND programmatic semantics
- Color contrast without tooling → reference specific hex values, recommend contrast tools
