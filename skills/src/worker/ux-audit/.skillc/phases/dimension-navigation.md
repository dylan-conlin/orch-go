# Dimension: Navigation (Full Depth)

**TLDR:** Comprehensive navigation audit evaluating page structure, sidebar/nav behavior, active states, link integrity, URL state management, and error page handling. Ensures users can orient themselves and move through the application confidently.

**When to use:** Focused navigation audit, full audit navigation dimension, or follow-up from quick scan navigation findings.

**Duration:** 30-60 minutes

**Primary tools:** `browser_snapshot`, `browser_click`, `browser_navigate`, `browser_navigate_back`, `browser_take_screenshot`, `browser_evaluate`

---

## Navigation Audit Workflow

Four stages, executed sequentially:

1. **Structure Scan** — map nav elements, sidebar, breadcrumbs, page title
2. **Active State Verification** — confirm current page is correctly indicated
3. **Link & Route Testing** — test links, deep links, back button, URL state
4. **Error & Edge Cases** — 404 page, auth redirects, error recovery

---

## Stage 1: Structure Scan

**Take a snapshot and identify all navigation elements:**

### Navigation Inventory Script

```javascript
() => {
  const results = {
    pageTitle: document.title,
    h1: [],
    navElements: [],
    sidebarLinks: [],
    breadcrumbs: [],
    activeIndicators: []
  };

  // H1 elements
  document.querySelectorAll('h1').forEach(h => {
    results.h1.push(h.textContent.trim().substring(0, 80));
  });

  // Nav elements
  document.querySelectorAll('nav, [role="navigation"]').forEach(nav => {
    const links = Array.from(nav.querySelectorAll('a'));
    results.navElements.push({
      ariaLabel: nav.getAttribute('aria-label') || nav.getAttribute('aria-labelledby') || 'unlabeled',
      linkCount: links.length,
      links: links.slice(0, 10).map(a => ({
        text: (a.textContent.trim() || a.getAttribute('aria-label') || '').substring(0, 40),
        href: a.getAttribute('href'),
        ariaCurrent: a.getAttribute('aria-current'),
        classes: a.className.toString().substring(0, 60)
      }))
    });
  });

  // Look for sidebar patterns
  document.querySelectorAll('[class*="sidebar"], [class*="side-nav"], aside').forEach(el => {
    const links = Array.from(el.querySelectorAll('a'));
    results.sidebarLinks = links.slice(0, 15).map(a => ({
      text: (a.textContent.trim() || a.getAttribute('aria-label') || '').substring(0, 40),
      href: a.getAttribute('href'),
      ariaCurrent: a.getAttribute('aria-current'),
      hasActiveClass: /active|current|selected/.test(a.className)
    }));
  });

  // Breadcrumbs
  document.querySelectorAll('[class*="breadcrumb"], [aria-label*="breadcrumb"], nav ol').forEach(el => {
    results.breadcrumbs = Array.from(el.querySelectorAll('a, span, li')).map(item =>
      item.textContent.trim().substring(0, 40)
    ).slice(0, 10);
  });

  return JSON.stringify(results, null, 2);
}
```

### Navigation Structure Checklist

- [ ] Page has a clear primary navigation (sidebar or top nav)
- [ ] Navigation has an accessible label (`aria-label` or `aria-labelledby`)
- [ ] Navigation links have descriptive text (not just icons without labels)
- [ ] Page `<title>` reflects current location (not generic "Toolshed" on every page)
- [ ] Exactly one `<h1>` that describes the page content

---

## Stage 2: Active State Verification

### Current Page Indication

- [ ] Current page is visually indicated in sidebar/nav (active state styling)
- [ ] Active state uses the design system accent (SCS blue #509be0 indicator)
- [ ] `aria-current="page"` is set on the current nav link
- [ ] Active state is distinguishable from hover state

**Active state detection script:**

```javascript
() => {
  const navLinks = document.querySelectorAll('nav a, [role="navigation"] a, aside a');
  const currentUrl = window.location.pathname;

  return JSON.stringify({
    currentPath: currentUrl,
    navLinks: Array.from(navLinks).slice(0, 20).map(a => {
      const href = a.getAttribute('href');
      const computed = getComputedStyle(a);
      const indicator = a.querySelector('[class*="indicator"], [class*="active"]');
      return {
        text: (a.textContent.trim() || a.getAttribute('aria-label') || '').substring(0, 40),
        href: href,
        matchesCurrentPath: href === currentUrl || currentUrl.startsWith(href + '/'),
        ariaCurrent: a.getAttribute('aria-current'),
        hasActiveClass: /active|current|selected/.test(a.className),
        color: computed.color,
        fontWeight: computed.fontWeight,
        hasIndicator: indicator !== null,
        indicatorColor: indicator ? getComputedStyle(indicator).backgroundColor : null
      };
    })
  }, null, 2);
}
```

### Tab/Sub-Navigation (if present)

- [ ] Active tab is visually distinct from inactive tabs
- [ ] Tab content updates when tab is clicked
- [ ] URL updates to reflect active tab (for deep linking)
- [ ] Tab state persists on page reload

---

## Stage 3: Link & Route Testing

### Link Integrity

**Click each nav link and verify:**

- [ ] All sidebar/nav links navigate to a valid page (no 404s, no blank pages)
- [ ] Links open in the current tab (no unexpected `target="_blank"` for internal links)
- [ ] After navigation, active state updates to reflect new page

**Workflow:**
1. Note current page's nav links from Stage 1 inventory
2. Click each link sequentially
3. At each destination: verify page loads, active state updates, title changes
4. Navigate back and verify return works

### Browser Back Button

- [ ] Back button returns to previous page (not a broken state)
- [ ] Page state is preserved on back navigation (scroll position, filter state)
- [ ] No infinite redirect loops when using back button

**Test:** Navigate forward 2-3 pages, then use `browser_navigate_back` twice. Verify each page loads correctly.

### Deep Links / URL State

- [ ] Current URL can be bookmarked and revisited (produces same view)
- [ ] Filter/sort state is reflected in URL params (if applicable)
- [ ] Tab selection is reflected in URL (if applicable)
- [ ] Sharing the URL gives another user the same view (minus auth)

**URL state check script:**

```javascript
() => {
  return JSON.stringify({
    fullUrl: window.location.href,
    pathname: window.location.pathname,
    search: window.location.search,
    hash: window.location.hash,
    hasQueryParams: window.location.search.length > 1,
    queryParams: Object.fromEntries(new URLSearchParams(window.location.search))
  }, null, 2);
}
```

---

## Stage 4: Error & Edge Cases

### 404 Page

- [ ] Navigate to a non-existent route (e.g., `/this-page-does-not-exist`)
- [ ] 404 page exists (not a blank screen or raw error)
- [ ] 404 page has navigation back to working pages (link to home, sidebar still visible)
- [ ] 404 page is styled consistently with the rest of the application

### Auth Flow Navigation

- [ ] Unauthenticated user is redirected to login (not shown a blank page)
- [ ] After login, user is redirected to their original destination (not always to home)
- [ ] Logout redirects to an appropriate page (login page or home)
- [ ] Session expiration is handled gracefully (not a raw 401 error)

### Error Recovery

- [ ] If a page errors, user has a path back to working pages
- [ ] Error messages include navigation hints ("Go back" or "Return to dashboard")
- [ ] Network errors don't trap the user on a broken page

---

## Severity Guide (Navigation)

| Finding | Severity | Rationale |
|---------|----------|-----------|
| Nav link leads to blank page or crash | **Blocker** | User is stuck with no way forward |
| No way to navigate back from error page | **Blocker** | User is trapped |
| No active state indication in navigation | **Major** | User cannot orient themselves — "where am I?" |
| Back button breaks (infinite loop, wrong destination) | **Major** | Core browser navigation fails |
| Missing 404 page (raw error or blank screen) | **Major** | Unprofessional, user has no recovery path |
| Login redirect doesn't return to original page | **Minor** | Friction — user must re-navigate after login |
| `aria-current` missing on active nav link | **Minor** | Accessibility issue for screen readers |
| Page title doesn't reflect current page | **Minor** | Confusing for tab management and bookmarks |
| URL doesn't reflect filter/tab state | **Minor** | Deep linking and sharing compromised |
| Active state uses wrong accent color | **Cosmetic** | Design system deviation |
| Breadcrumbs slightly misaligned | **Cosmetic** | Visual polish |

---

## Finding Template (Navigation)

```markdown
#### {N}. {Finding Title}
**Severity:** {Blocker/Major/Minor/Cosmetic}
**Viewport(s):** {affected viewport widths or "all"}
**Screenshot:** `{relative path to screenshot}`
**Evidence:** {snapshot excerpt, URL state, click result, script output}
**Impact:** {user orientation or navigation problem}
**Recommendation:** {specific fix — route, component, or ARIA attribute}
```

---

## Anti-Patterns

**Testing only the happy path**
Navigation bugs surface at edges: 404 pages, expired sessions, deep links, back button. Always test error states.

**Skipping `aria-current` checks**
Visual active states are necessary but not sufficient. Screen reader users rely on `aria-current="page"` to know where they are. Always verify both visual and semantic indicators.

**Assuming links work because they're in the nav**
Dead links are common after route restructuring. Click each link and verify the destination page loads — don't trust the `href` alone.

**Testing only at one viewport**
Sidebar navigation often collapses or transforms at narrow viewports. Verify that navigation remains accessible at mobile widths. (This overlaps with responsive dimension — focus here on functional availability, not layout.)
