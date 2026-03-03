# Dimension: Visual Consistency (Full Depth)

**TLDR:** Comprehensive design system adherence audit checking color tokens, typography, spacing, depth strategy, component patterns, and semantic color usage against the toolshed design direction.

**When to use:** Focused visual consistency audit, full audit visual dimension, or follow-up from quick scan visual findings.

**Duration:** 30-60 minutes

**Primary tools:** `playwright-cli eval` (computed styles), `playwright-cli screenshot`, `playwright-cli snapshot`

**Design reference:** `.kb/decisions/2026-02-21-toolshed-design-direction.md`

---

## Design System Reference (Toolshed)

### Color Tokens

| Token | Expected Value | Role |
|-------|---------------|------|
| `--background` | `#FAFBFC` / `rgb(250, 251, 252)` | Page background |
| `--surface` | `#FFFFFF` / `rgb(255, 255, 255)` | Card/panel surfaces |
| `--surface-raised` | `#F8FAFC` / `rgb(248, 250, 252)` | Hover states, sidebar |
| `--border` | `rgba(15, 23, 42, 0.08)` | Primary borders |
| `--border-strong` | `rgba(15, 23, 42, 0.15)` | Section dividers |
| `--foreground` | `#0F172A` / `rgb(15, 23, 42)` | Primary text (Slate 900) |
| `--foreground-secondary` | `#475569` / `rgb(71, 85, 105)` | Secondary text (Slate 600) |
| `--foreground-muted` | `#94A3B8` / `rgb(148, 163, 184)` | Muted text (Slate 400) |
| `--accent` | `#509be0` / `rgb(80, 155, 224)` | SCS blue — actions, links |
| `--accent-hover` | `#3d85c6` | Hover state for accent |
| `--success` | `#10B981` | Positive trends, completed |
| `--warning` | `#F59E0B` | Anomalies, attention |
| `--destructive` | `#cb1217` | Errors, negative trends |

### Typography

| Role | Font | Notes |
|------|------|-------|
| UI text | Inter | Geometric sans, all body/label text |
| Data/numbers | JetBrains Mono | Monospace for tabular data, KPIs, code |

### Spacing System

| Step | Value | Usage |
|------|-------|-------|
| xs | 4px | Tight internal spacing |
| sm | 8px | Between related elements |
| md | 12px | Standard gap |
| base | 16px | Card padding, section spacing |
| lg | 24px | Between sections |
| xl | 32px | Major section breaks |

### Depth Strategy

| Rule | Expected |
|------|----------|
| Primary depth mechanism | **Borders** (0.5px solid rgba(15,23,42,0.08)) |
| Shadow usage | Dropdowns and modals ONLY |
| Card depth | Border, NOT shadow |
| Hover depth | `--surface-raised` background change |

### Component Patterns

| Component | Specification |
|-----------|--------------|
| KPI cards | Uppercase label, large mono number, trend indicator |
| Tables | 500 weight header, 14px mono data, 40px row height, no zebra-striping |
| Border radius | 4/6/8px system |
| Card padding | 16px |

---

## Visual Consistency Audit Workflow

### Step 1: Foundation Checks (Color & Background)

**Run computed style audit script:**

```javascript
() => {
  const body = getComputedStyle(document.body);
  const main = document.querySelector('main');
  const mainStyle = main ? getComputedStyle(main) : null;

  // Find all card-like elements
  const cardSelectors = '[class*="card"], [class*="rounded"], [class*="panel"], [class*="container"]';
  const cards = document.querySelectorAll(cardSelectors);

  // Find sidebar
  const sidebar = document.querySelector('nav, aside, [class*="sidebar"], [class*="side-nav"]');
  const sidebarStyle = sidebar ? getComputedStyle(sidebar) : null;

  return JSON.stringify({
    foundation: {
      bodyBackground: body.backgroundColor,
      bodyFont: body.fontFamily,
      bodyColor: body.color
    },
    main: mainStyle ? {
      background: mainStyle.backgroundColor,
      padding: mainStyle.padding
    } : null,
    sidebar: sidebarStyle ? {
      background: sidebarStyle.backgroundColor,
      borderRight: sidebarStyle.borderRight,
      width: sidebarStyle.width
    } : null,
    cards: {
      count: cards.length,
      samples: Array.from(cards).slice(0, 5).map(c => {
        const s = getComputedStyle(c);
        return {
          class: c.className.toString().substring(0, 60),
          background: s.backgroundColor,
          border: s.border,
          borderRadius: s.borderRadius,
          boxShadow: s.boxShadow,
          padding: s.padding
        };
      })
    }
  }, null, 2);
}
```

**Check against design tokens:**
- [ ] Body background matches `#FAFBFC` (or close — allow rgb(250,251,252))
- [ ] Card/panel surfaces are white (`#FFFFFF`)
- [ ] Sidebar uses raised surface (`#F8FAFC`)
- [ ] No unexpected background colors

### Step 2: Typography Audit

```javascript
() => {
  const elements = {
    h1: document.querySelector('h1'),
    h2: document.querySelector('h2'),
    h3: document.querySelector('h3'),
    bodyText: document.querySelector('main p, main span, main td'),
    label: document.querySelector('label, [class*="label"]'),
    dataValue: document.querySelector('[class*="mono"], [class*="tabular"], code, .font-mono'),
    link: document.querySelector('a:not([class*="nav"])')
  };

  const result = {};
  for (const [name, el] of Object.entries(elements)) {
    if (el) {
      const s = getComputedStyle(el);
      result[name] = {
        fontFamily: s.fontFamily,
        fontSize: s.fontSize,
        fontWeight: s.fontWeight,
        lineHeight: s.lineHeight,
        color: s.color,
        letterSpacing: s.letterSpacing
      };
    }
  }
  return JSON.stringify(result, null, 2);
}
```

**Check against design spec:**
- [ ] UI text uses Inter (or system sans-serif fallback that includes Inter)
- [ ] Data values use JetBrains Mono (or monospace fallback)
- [ ] Primary text color is Slate 900 (`#0F172A` / `rgb(15,23,42)`)
- [ ] Secondary text is Slate 600 (`#475569`)
- [ ] Muted text is Slate 400 (`#94A3B8`)
- [ ] Text hierarchy is consistent (h1 > h2 > h3 in size/weight)

### Step 3: Depth & Border Strategy

```javascript
() => {
  // Check all elements for shadows (should be rare — only dropdowns/modals)
  const allElements = document.querySelectorAll('*');
  const shadowElements = [];
  const borderElements = [];

  allElements.forEach(el => {
    const s = getComputedStyle(el);
    if (s.boxShadow && s.boxShadow !== 'none') {
      shadowElements.push({
        tag: el.tagName,
        class: el.className.toString().substring(0, 60),
        boxShadow: s.boxShadow.substring(0, 100),
        role: el.getAttribute('role') || '',
        isDropdown: el.matches('[class*="dropdown"], [class*="popover"], [class*="menu"], [role="dialog"], [role="listbox"]')
      });
    }
  });

  // Sample borders on card-like elements
  const cards = document.querySelectorAll('[class*="card"], [class*="panel"], [class*="rounded"]');
  cards.forEach(c => {
    const s = getComputedStyle(c);
    if (s.border && s.border !== '0px none rgb(0, 0, 0)') {
      borderElements.push({
        class: c.className.toString().substring(0, 60),
        border: s.border,
        borderRadius: s.borderRadius
      });
    }
  });

  return JSON.stringify({
    shadowCount: shadowElements.length,
    shadows: shadowElements.slice(0, 10),
    borderSamples: borderElements.slice(0, 10)
  }, null, 2);
}
```

**Check against depth strategy:**
- [ ] Cards use borders, NOT shadows
- [ ] Shadows only on dropdowns, modals, popovers
- [ ] Border style is `0.5px solid rgba(15,23,42,0.08)` or equivalent
- [ ] Border radius follows 4/6/8px system
- [ ] No arbitrary shadows on non-overlay elements

### Step 4: Accent & Semantic Color Usage

```javascript
() => {
  const results = {
    accentUsage: [],
    semanticUsage: [],
    unexpectedColors: []
  };

  // Check links and buttons for accent color
  document.querySelectorAll('a, button, [role="button"]').forEach(el => {
    const s = getComputedStyle(el);
    results.accentUsage.push({
      tag: el.tagName,
      text: (el.textContent || '').substring(0, 30),
      color: s.color,
      background: s.backgroundColor
    });
  });
  results.accentUsage = results.accentUsage.slice(0, 10);

  // Check for semantic colors (success, warning, destructive)
  const semanticClasses = '[class*="success"], [class*="warning"], [class*="error"], [class*="danger"], [class*="destructive"], [class*="green"], [class*="amber"], [class*="red"]';
  document.querySelectorAll(semanticClasses).forEach(el => {
    const s = getComputedStyle(el);
    results.semanticUsage.push({
      class: el.className.toString().substring(0, 60),
      color: s.color,
      background: s.backgroundColor
    });
  });
  results.semanticUsage = results.semanticUsage.slice(0, 10);

  return JSON.stringify(results, null, 2);
}
```

**Check against design spec:**
- [ ] Actions/links use SCS blue (`#509be0`)
- [ ] SCS red (`#cb1217`) only for logo/brand + destructive actions
- [ ] Green (`#10B981`) for success/positive trends only
- [ ] Amber (`#F59E0B`) for warnings/attention only
- [ ] Red for errors/negative trends only — not decorative
- [ ] No colors used decoratively that have semantic meaning

### Step 5: Spacing Consistency

```javascript
() => {
  // Sample spacing from common elements
  const cards = document.querySelectorAll('[class*="card"], [class*="panel"]');
  const sections = document.querySelectorAll('section, [class*="section"]');

  const spacingSamples = [];

  cards.forEach(c => {
    const s = getComputedStyle(c);
    spacingSamples.push({
      type: 'card',
      class: c.className.toString().substring(0, 40),
      padding: s.padding,
      margin: s.margin,
      gap: s.gap
    });
  });

  sections.forEach(sec => {
    const s = getComputedStyle(sec);
    spacingSamples.push({
      type: 'section',
      class: sec.className.toString().substring(0, 40),
      padding: s.padding,
      margin: s.margin,
      gap: s.gap
    });
  });

  return JSON.stringify({
    samples: spacingSamples.slice(0, 15),
    expectedSystem: '4px grid: 4, 8, 12, 16, 24, 32'
  }, null, 2);
}
```

**Check against spacing system:**
- [ ] Card padding is 16px (or follows 4px grid)
- [ ] Spacing between elements follows 4px grid (4, 8, 12, 16, 24, 32)
- [ ] No arbitrary spacing values (7px, 13px, 22px)
- [ ] Consistent gaps within similar component groups

### Step 6: Component Pattern Compliance

**KPI Cards (if present):**
- [ ] Label is uppercase (or small caps)
- [ ] Number uses monospace font (JetBrains Mono)
- [ ] Trend indicator present (arrow, color, percentage)
- [ ] Card padding is 16px

**Tables (if present):**
- [ ] Header row uses 500 font weight
- [ ] Data cells use 14px monospace
- [ ] Row height is ~40px
- [ ] No zebra-striping (per design spec)
- [ ] Hover state uses surface-2 background

**Buttons:**
- [ ] Primary buttons use accent color
- [ ] Hover state changes are visible
- [ ] Disabled state is visually distinct

---

## Severity Guide (Visual Consistency)

| Finding | Severity | Rationale |
|---------|----------|-----------|
| Wrong font family (sans when should be mono, or vice versa) | **Major** | Data/UI distinction is a core design principle |
| Shadows on cards (should be borders per design spec) | **Major** | Violates depth strategy — creates visual inconsistency |
| Wrong accent color (not SCS blue) | **Major** | Brand inconsistency |
| Semantic color misuse (red for non-error content) | **Major** | Confuses meaning |
| Missing hover states on interactive elements | **Minor** | Interaction feedback incomplete |
| Spacing not on 4px grid | **Minor** | Visual rhythm broken |
| Border radius inconsistency (mix of values) | **Minor** | Visual inconsistency |
| Typography weight inconsistency within same component type | **Minor** | Visual rhythm broken |
| Slightly wrong shade of gray (Slate 500 vs 600) | **Cosmetic** | Close enough, barely noticeable |
| 1px border radius difference between similar cards | **Cosmetic** | Visual polish |
| Padding differs by 2-4px between similar components | **Cosmetic** | Barely noticeable |

---

## Finding Template (Visual Consistency)

```markdown
#### {N}. {Finding Title}
**Severity:** {Blocker/Major/Minor/Cosmetic}
**Design Token:** {which token is violated}
**Expected:** {design spec value}
**Actual:** {computed style value}
**Screenshot:** `{relative path}`
**Elements Affected:** {count and description}
**Impact:** {how this affects visual coherence}
**Recommendation:** {specific CSS fix}
```

---

## Anti-Patterns

**Checking colors by eye instead of computed styles**
`rgb(15, 23, 42)` and `rgb(30, 41, 59)` look identical on screen but are different tokens. Always use `playwright-cli eval` to get computed values.

**Reporting every color as a violation**
Some color variance is acceptable (e.g., Tailwind utility classes applying slightly different opacities). Focus on meaningful deviations from the design system, not pixel-perfect matching.

**Auditing design tokens that don't exist yet**
If the design direction document doesn't specify a token (e.g., no specified chart color palette), that's a design gap to surface — not a violation to report. Use "design gap" findings, not "violation" findings.

**Ignoring dark mode considerations**
If CSS variables are used, the design system is dark-mode ready even if dark mode isn't implemented. Note whether CSS variables are used consistently (positive finding) vs hardcoded colors (finding).
