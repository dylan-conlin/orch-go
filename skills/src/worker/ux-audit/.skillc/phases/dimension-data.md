# Dimension: Data Presentation (Full Depth)

**TLDR:** Comprehensive audit of how data values, tables, numbers, labels, and visualizations are displayed. Checks formatting consistency, alignment, empty states, null handling, and data loading behavior.

**When to use:** Focused data audit, full audit data-presentation dimension, or follow-up from quick scan data findings.

**Duration:** 30-60 minutes

**Primary tools:** `browser_snapshot`, `browser_evaluate`, `browser_take_screenshot`

---

## Data Presentation Audit Workflow

Three stages, executed sequentially:

1. **Content Scan** — identify all data elements on the page via snapshot and evaluation
2. **Formatting Checks** — verify numbers, currencies, dates, labels are properly formatted
3. **Table & Layout Audit** — verify alignment, headers, empty/null states, loading behavior

---

## Stage 1: Content Scan

**Take a snapshot and identify all data-bearing elements:**

### Data Element Inventory

Run this to catalog data elements on the page:

```javascript
() => {
  const results = {
    tables: [],
    numbers: [],
    currencies: [],
    percentages: [],
    dates: [],
    emptyStates: [],
    rawValues: []
  };

  // Find tables
  document.querySelectorAll('table').forEach(t => {
    const headers = Array.from(t.querySelectorAll('th')).map(th => th.textContent.trim());
    results.tables.push({
      headers: headers.slice(0, 10),
      rows: t.querySelectorAll('tbody tr').length,
      hasHeader: t.querySelector('thead') !== null
    });
  });

  // Scan visible text for formatting patterns
  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  let node;
  while (node = walker.nextNode()) {
    const text = node.textContent.trim();
    if (!text || text.length > 200) continue;

    // Currency patterns
    if (/\$[\d,.]+/.test(text)) {
      results.currencies.push(text.substring(0, 50));
    }
    // Percentage patterns
    if (/\d+\.?\d*%/.test(text)) {
      results.percentages.push(text.substring(0, 50));
    }
    // Raw snake_case or ALL_CAPS that look like database values
    if (/^[a-z_]{3,}$/.test(text) || /^[A-Z_]{3,}$/.test(text)) {
      results.rawValues.push(text.substring(0, 50));
    }
  }

  // Deduplicate
  results.currencies = [...new Set(results.currencies)].slice(0, 15);
  results.percentages = [...new Set(results.percentages)].slice(0, 15);
  results.rawValues = [...new Set(results.rawValues)].slice(0, 15);

  // Find empty state indicators
  document.querySelectorAll('[class*="empty"], [class*="no-data"], [class*="placeholder"]').forEach(el => {
    results.emptyStates.push({
      text: el.textContent.trim().substring(0, 80),
      class: el.className.toString().substring(0, 60)
    });
  });

  return JSON.stringify(results, null, 2);
}
```

**Use inventory to guide deeper checks in Stage 2.**

---

## Stage 2: Formatting Checks

### Number Formatting

- [ ] Large numbers use comma separators (1,234 not 1234)
- [ ] Decimal precision is consistent within context (all prices $XX.XX, not mix of $XX and $XX.XX)
- [ ] No floating point artifacts (12.300000000001)
- [ ] Negative numbers are visually distinct (red, parentheses, or minus sign — consistently)

### Currency Formatting

- [ ] Currency symbol present ($ for USD)
- [ ] Consistent precision: prices use 2 decimal places ($1,234.56)
- [ ] No bare numbers where currency is expected
- [ ] Unit prices vs total prices are clearly labeled

**Currency precision check script:**

```javascript
() => {
  const currencyPattern = /\$[\d,.]+/g;
  const textContent = document.body.innerText;
  const matches = textContent.match(currencyPattern) || [];
  const precisions = matches.map(m => {
    const parts = m.replace('$', '').split('.');
    return {
      value: m,
      decimals: parts[1] ? parts[1].replace(/,/g, '').length : 0
    };
  });
  const uniquePrecisions = [...new Set(precisions.map(p => p.decimals))];
  return JSON.stringify({
    currencyCount: matches.length,
    samples: precisions.slice(0, 10),
    uniqueDecimalPrecisions: uniquePrecisions,
    isConsistent: uniquePrecisions.length <= 1
  }, null, 2);
}
```

### Percentage Formatting

- [ ] Consistent decimal precision (all 12.4% or all 12% — not mixed)
- [ ] No excessive precision (12.3567% should be 12.4% or 12%)
- [ ] Percentage sign present (not bare decimal like 0.124)

### Date Formatting

- [ ] Consistent date format across the page
- [ ] Dates use human-readable format (not Unix timestamps or ISO 8601 raw)
- [ ] Relative dates are clear ("2 hours ago" not just a timestamp)

### Labels and Headers

- [ ] Table headers are descriptive (not abbreviations or codes without explanation)
- [ ] No raw database column names (snake_case, camelCase exposed to user)
- [ ] No ALL_CAPS database enums displayed raw
- [ ] Units are included where applicable (e.g., "Weight (lbs)" not just "Weight")

---

## Stage 3: Table & Layout Audit

### Table Structure

- [ ] Tables have `<thead>` with clear column headers
- [ ] Header text uses appropriate weight (500+ for distinction)
- [ ] Data cells use monospace font for numbers (per design system: JetBrains Mono)
- [ ] Row height is adequate (≥40px per design system)
- [ ] No zebra-striping (per Toolshed design direction)

### Data Alignment

- [ ] Numeric columns are right-aligned
- [ ] Text columns are left-aligned
- [ ] Currency columns are right-aligned with decimal alignment
- [ ] Header alignment matches data alignment

**Alignment check script:**

```javascript
() => {
  const tables = document.querySelectorAll('table');
  return JSON.stringify({
    tables: Array.from(tables).slice(0, 3).map(t => {
      const headers = Array.from(t.querySelectorAll('th'));
      const firstRow = t.querySelector('tbody tr');
      const cells = firstRow ? Array.from(firstRow.querySelectorAll('td')) : [];
      return {
        headerAlignments: headers.map(h => ({
          text: h.textContent.trim().substring(0, 30),
          textAlign: getComputedStyle(h).textAlign
        })),
        cellAlignments: cells.map(c => ({
          text: c.textContent.trim().substring(0, 30),
          textAlign: getComputedStyle(c).textAlign,
          fontFamily: getComputedStyle(c).fontFamily.substring(0, 30)
        }))
      };
    })
  }, null, 2);
}
```

### Empty & Null States

- [ ] Empty tables show a helpful message (not blank or "No results")
- [ ] Null/undefined values display gracefully (dash, "N/A", or "—" — not "null", "undefined", "NaN")
- [ ] Zero values are distinguishable from empty/null (0 vs —)
- [ ] Loading states exist before data arrives (skeleton, spinner, or message)

**Null value detection script:**

```javascript
() => {
  const suspicious = [];
  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  let node;
  while (node = walker.nextNode()) {
    const text = node.textContent.trim();
    if (['null', 'undefined', 'NaN', 'None', '[object Object]'].includes(text)) {
      const parent = node.parentElement;
      suspicious.push({
        value: text,
        tag: parent.tagName,
        class: parent.className.toString().substring(0, 60),
        context: parent.parentElement ? parent.parentElement.textContent.trim().substring(0, 80) : ''
      });
    }
  }
  return JSON.stringify({
    suspiciousValueCount: suspicious.length,
    values: suspicious.slice(0, 10)
  }, null, 2);
}
```

### Data Loading Behavior

- [ ] Data loads without layout shift (content area doesn't jump when data arrives)
- [ ] Large datasets have pagination or virtual scrolling (not rendering 1000+ rows)
- [ ] Sort interactions update data (if table is sortable)

---

## Severity Guide (Data Presentation)

| Finding | Severity | Rationale |
|---------|----------|-----------|
| Raw "null", "undefined", or "NaN" displayed to user | **Blocker** | Broken data pipeline — user sees application internals |
| Database column names exposed (snake_case labels) | **Major** | Unprofessional, confusing for non-technical users |
| Currency values missing currency symbol | **Major** | Ambiguous — user cannot tell if value is dollars, euros, etc. |
| Inconsistent decimal precision within same context | **Major** | Undermines data credibility — $12.50 next to $12.5 |
| Numbers lack comma separators (1234567 vs 1,234,567) | **Minor** | Readability issue for large numbers |
| Mixed date formats on same page | **Minor** | Inconsistency, not blocking |
| Numeric columns left-aligned instead of right-aligned | **Minor** | Makes number comparison harder |
| Data table missing thead | **Minor** | Accessibility and semantic issue |
| Slightly excessive decimal precision (12.34% vs 12.3%) | **Cosmetic** | Minor polish |
| Row height slightly under 40px | **Cosmetic** | Design system deviation, not impactful |

---

## Finding Template (Data Presentation)

```markdown
#### {N}. {Finding Title}
**Severity:** {Blocker/Major/Minor/Cosmetic}
**Viewport(s):** {affected viewport widths or "all"}
**Screenshot:** `{relative path to screenshot}`
**Evidence:** {script output, computed style values, raw text observed}
**Impact:** {what the user sees or misunderstands}
**Recommendation:** {specific formatting fix — code-level if possible}
```

---

## Anti-Patterns

**Checking only visible data**
Use the evaluation scripts to find raw values buried in the DOM, not just what's immediately visible. Scroll through tables and check cells beyond the first few rows.

**Ignoring empty states**
The most common data presentation bug is a blank area where data should be. Always navigate to states where data might be empty (filtered to zero results, new user with no data, error response).

**Accepting "N/A" everywhere**
"N/A" is appropriate for truly not-applicable values, but it's often used as a lazy catch-all for null. Distinguish between "data doesn't exist" (—), "not applicable" (N/A), and "zero" (0).

**Skipping font family checks**
The design system specifies JetBrains Mono for data values. Numbers in Inter (the UI font) are harder to read in dense tables. Always check with the alignment script.
