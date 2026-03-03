# Dimension: Interactive States (Full Depth)

**TLDR:** Comprehensive audit of UI element feedback across all interaction states — hover, active, disabled, loading, error, empty, success, and transitions. Ensures users receive appropriate visual and semantic feedback for every interaction.

**When to use:** Focused interactive-states audit, full audit interactive-states dimension, or follow-up from quick scan interaction findings.

**Duration:** 30-60 minutes

**Primary tools:** `browser_click`, `browser_hover`, `browser_snapshot`, `browser_take_screenshot`, `browser_evaluate`, `browser_console_messages`

---

## Interactive States Audit Workflow

Four stages, executed sequentially:

1. **Element Inventory** — catalog all interactive elements and their current states
2. **State Testing** — systematically trigger hover, click, disabled, loading, error states
3. **Feedback Verification** — confirm visual and semantic feedback for each state
4. **Console & Error Check** — verify no JS errors during interactions

---

## Stage 1: Element Inventory

**Identify all interactive elements on the page:**

### Interactive Element Catalog Script

```javascript
() => {
  const results = {
    buttons: [],
    links: [],
    formInputs: [],
    dropdowns: [],
    toggles: [],
    modals: [],
    tooltipTriggers: []
  };

  // Buttons
  document.querySelectorAll('button, [role="button"], input[type="submit"]').forEach(el => {
    const computed = getComputedStyle(el);
    results.buttons.push({
      text: (el.textContent.trim() || el.getAttribute('aria-label') || el.value || '').substring(0, 40),
      disabled: el.disabled || el.getAttribute('aria-disabled') === 'true',
      type: el.type || 'button',
      hasIcon: el.querySelector('svg, img, [class*="icon"]') !== null,
      hasText: el.textContent.trim().length > 0,
      cursor: computed.cursor,
      opacity: computed.opacity
    });
  });

  // Form inputs
  document.querySelectorAll('input:not([type="hidden"]):not([type="submit"]), textarea, select').forEach(el => {
    results.formInputs.push({
      type: el.type || el.tagName.toLowerCase(),
      name: el.name || el.id || '',
      placeholder: el.placeholder || '',
      required: el.required,
      disabled: el.disabled,
      hasLabel: !!document.querySelector(`label[for="${el.id}"]`) || !!el.closest('label'),
      ariaLabel: el.getAttribute('aria-label') || ''
    });
  });

  // Dropdowns / select menus
  document.querySelectorAll('select, [role="combobox"], [role="listbox"], [class*="dropdown"], [class*="select"]').forEach(el => {
    results.dropdowns.push({
      type: el.tagName,
      role: el.getAttribute('role'),
      expanded: el.getAttribute('aria-expanded'),
      text: el.textContent.trim().substring(0, 40)
    });
  });

  // Toggles / switches
  document.querySelectorAll('[role="switch"], [role="checkbox"], input[type="checkbox"]').forEach(el => {
    results.toggles.push({
      text: (el.textContent.trim() || el.getAttribute('aria-label') || '').substring(0, 40),
      checked: el.checked || el.getAttribute('aria-checked') === 'true'
    });
  });

  return JSON.stringify({
    summary: {
      buttons: results.buttons.length,
      formInputs: results.formInputs.length,
      dropdowns: results.dropdowns.length,
      toggles: results.toggles.length
    },
    buttons: results.buttons.slice(0, 15),
    formInputs: results.formInputs.slice(0, 10),
    dropdowns: results.dropdowns.slice(0, 5),
    toggles: results.toggles.slice(0, 5)
  }, null, 2);
}
```

---

## Stage 2: State Testing

### Hover States

For each button and interactive element:

1. **Use `browser_hover`** on the element
2. **Take snapshot** to check ARIA state changes
3. **Take screenshot** for visual evidence

**Check:**
- [ ] Buttons have visible hover state (background change, underline, or opacity shift)
- [ ] Hover uses design system surface-2 background (per Toolshed design direction)
- [ ] Cursor changes to `pointer` on clickable elements
- [ ] Icon-only buttons show tooltips on hover
- [ ] Links have distinct hover styling (underline or color change)

**Hover style detection script:**

```javascript
(element) => {
  const computed = getComputedStyle(element);
  return JSON.stringify({
    cursor: computed.cursor,
    backgroundColor: computed.backgroundColor,
    color: computed.color,
    textDecoration: computed.textDecoration,
    opacity: computed.opacity,
    outline: computed.outline,
    boxShadow: computed.boxShadow,
    transition: computed.transition
  }, null, 2);
}
```

### Click / Active States

- [ ] Buttons have visible active/pressed state (slight depression, color change)
- [ ] Click triggers the expected action (navigation, form submit, data operation)
- [ ] Double-click doesn't trigger duplicate actions (submit button debounced)

### Disabled States

- [ ] Disabled buttons are visually distinct (reduced opacity, grayed out)
- [ ] Disabled elements have `disabled` attribute or `aria-disabled="true"`
- [ ] Cursor shows `not-allowed` on disabled elements
- [ ] Disabled elements cannot be activated (click has no effect)

**Disabled state check script:**

```javascript
() => {
  const disabled = document.querySelectorAll('[disabled], [aria-disabled="true"]');
  return JSON.stringify({
    disabledCount: disabled.length,
    elements: Array.from(disabled).slice(0, 10).map(el => {
      const computed = getComputedStyle(el);
      return {
        tag: el.tagName,
        text: (el.textContent.trim() || el.getAttribute('aria-label') || '').substring(0, 40),
        opacity: computed.opacity,
        cursor: computed.cursor,
        pointerEvents: computed.pointerEvents
      };
    })
  }, null, 2);
}
```

---

## Stage 3: Feedback Verification

### Loading States

- [ ] Async operations show a loading indicator (spinner, skeleton, progress bar)
- [ ] Submit buttons show loading state during API calls (spinner or "Submitting...")
- [ ] Loading indicator is accessible (`aria-busy="true"` or `role="status"` with message)
- [ ] Page-level loading doesn't flash (minimum display time to avoid flicker)

**To test:** Click a button that triggers an API call and observe:
1. Does the button change state? (loading spinner, disabled)
2. Does the page show a loading indicator?
3. After completion, does the button return to normal?

### Error States

- [ ] Form validation errors appear immediately on invalid input (not only on submit)
- [ ] Error messages are specific ("Email is required" not "Validation failed")
- [ ] Error messages are visually prominent (red text, icon, border highlight)
- [ ] Error messages are associated with their field (`aria-describedby` or `aria-errormessage`)
- [ ] Network errors show user-friendly messages (not raw HTTP status or stack traces)
- [ ] Error state doesn't trap the user (can dismiss, retry, or navigate away)

### Success States

- [ ] Successful actions show confirmation (toast, flash message, inline confirmation)
- [ ] Success messages auto-dismiss after reasonable time (3-5 seconds)
- [ ] Success messages are accessible (`role="status"` or `aria-live="polite"`)

### Empty States

- [ ] Empty lists/tables show helpful content (not blank or "No results")
- [ ] Empty states suggest next actions ("Add your first item" with a CTA button)
- [ ] Empty state messaging is contextual (not generic across all empty states)

**Empty state detection script:**

```javascript
() => {
  const emptyIndicators = [];

  // Check for common empty state patterns
  const selectors = [
    '[class*="empty"]', '[class*="no-data"]', '[class*="no-results"]',
    '[class*="placeholder"]', '[class*="zero-state"]'
  ];

  selectors.forEach(sel => {
    document.querySelectorAll(sel).forEach(el => {
      if (el.offsetHeight > 0) {
        emptyIndicators.push({
          selector: sel,
          text: el.textContent.trim().substring(0, 100),
          hasButton: el.querySelector('button, a') !== null
        });
      }
    });
  });

  // Check for tables with no rows
  document.querySelectorAll('table').forEach(t => {
    const tbody = t.querySelector('tbody');
    if (tbody && tbody.querySelectorAll('tr').length === 0) {
      emptyIndicators.push({
        selector: 'empty table',
        text: 'Table has no data rows',
        hasButton: false
      });
    }
  });

  return JSON.stringify({
    count: emptyIndicators.length,
    indicators: emptyIndicators.slice(0, 10)
  }, null, 2);
}
```

### Transitions and Animations

- [ ] Expand/collapse animations are smooth (not janky or instant)
- [ ] Modal open/close has transition (fade, slide — not abrupt)
- [ ] Animations respect `prefers-reduced-motion` media query
- [ ] No animation causes layout shift in surrounding elements

---

## Stage 4: Console & Error Check

**After all interactions, check for JavaScript errors:**

Use `browser_console_messages` with level `error` to capture:

- [ ] No JavaScript errors during normal interactions
- [ ] No unhandled promise rejections
- [ ] No React/Svelte component errors

**Common console errors to flag:**
- `TypeError: Cannot read property of undefined/null` — broken data flow
- `404` network requests — missing assets or API endpoints
- `CORS` errors — misconfigured API calls
- Unhandled rejection — promise without catch

---

## Severity Guide (Interactive States)

| Finding | Severity | Rationale |
|---------|----------|-----------|
| Button click causes JS error / page crash | **Blocker** | User action breaks the application |
| No loading state for long async operation (>1s) | **Major** | User thinks app is frozen |
| Form submission succeeds with no confirmation | **Major** | User doesn't know if action worked |
| Error message is raw HTTP status or stack trace | **Major** | Unprofessional, confusing, possible data leak |
| No visible hover state on buttons | **Major** | User unsure if element is clickable |
| Disabled button has no visual distinction | **Major** | User cannot tell active from inactive |
| Form validation only on submit (not inline) | **Minor** | Delayed feedback, more friction |
| Empty state is blank (no helpful message) | **Minor** | Missed opportunity to guide user |
| Icon-only button missing tooltip | **Minor** | User must guess button purpose |
| Success toast doesn't auto-dismiss | **Minor** | Clutters the UI |
| Transition slightly janky on one animation | **Cosmetic** | Polish issue |
| Hover state color slightly off from design system | **Cosmetic** | Design system deviation |

---

## Finding Template (Interactive States)

```markdown
#### {N}. {Finding Title}
**Severity:** {Blocker/Major/Minor/Cosmetic}
**Viewport(s):** {affected viewport widths or "all"}
**Screenshot:** `{relative path to screenshot}`
**Evidence:** {interaction performed, state change observed, console output, script results}
**Impact:** {what feedback the user is missing or receiving incorrectly}
**Recommendation:** {specific fix — component, state management, ARIA attribute}
```

---

## Anti-Patterns

**Testing only default state**
Most interaction bugs hide behind user actions. Click every button, hover every interactive element, submit forms with valid and invalid data. The default state is the least interesting.

**Ignoring console errors**
JavaScript errors during interaction are often the root cause of missing feedback states. Always check `browser_console_messages` after testing interactions.

**Skipping loading state checks**
Fast local networks mask loading state issues. Consider that production users may be on slower connections. If an operation takes data from an API, there should be a loading state — even if it's fast locally.

**Testing only success paths**
Error states, empty states, and edge cases are where interactive feedback matters most. Deliberately trigger errors (invalid form data, disconnect from network) and empty states (filter to zero results).

**Reporting "has hover state" without specifying what**
"Button has hover state" is not evidence. Document what changes: "Background changes from transparent to #F1F5F9 (surface-2), cursor changes to pointer." Specificity enables verification.
