## When to Use This Skill

**Use when the user says:**
- "Audit the UX of [page]"
- "Run a UX health check on [page]"
- "Check accessibility of [page]"
- "Is [page] responsive?"
- "Quick UX scan of [page]"

**Auto-detect mode from context:**
- "Quick check" / "health check" / "pre-demo verify" → quick mode
- "Check responsive behavior" / "test at mobile" → focused:responsive
- "Accessibility audit" / "a11y check" → focused:accessibility
- "Full UX audit" / "comprehensive audit" → full mode

---

## Skill Overview

This skill performs systematic UI/UX audits using playwright-cli. It evaluates web pages across six dimensions, producing an investigation file with severity-classified findings, screenshot evidence, and baseline metrics comparable across audits.

**Six dimensions:**

| Dimension | What It Audits | Primary Tools |
|-----------|---------------|---------------|
| **visual-consistency** | Design tokens, color, typography, spacing, depth | screenshot + evaluate (computed styles) |
| **responsive** | Breakpoint behavior at sm/md/lg, reflow, touch targets | resize + snapshot + screenshot |
| **accessibility** | a11y tree, WCAG via axe-core, keyboard nav, contrast | snapshot + evaluate (axe-core) + press_key |
| **data-presentation** | Tables, numbers, labels, formatting, units | snapshot + screenshot |
| **navigation** | Page structure, sidebar, breadcrumbs, active states | snapshot + click + navigate |
| **interactive-states** | Loading, empty, error, hover, disabled, animations | click + hover + snapshot + screenshot |

**Three modes:**

| Mode | Duration | Coverage | When to Use |
|------|----------|----------|-------------|
| **Quick** | ~30 min | All 6 dimensions, surface checks only | Regular health check, pre-demo verify, after deploy |
| **Focused** | 30-60 min | 1-2 dimensions, full depth | Known problem area, follow-up from quick scan |
| **Full** | 2-4 hours | All 6 dimensions, full depth | Initial audit, major redesign, quarterly review |

---

## Severity Classification

| Severity | Definition | Example | Action |
|----------|-----------|---------|--------|
| **Blocker** | User cannot complete their task | Page crashes, data not visible, auth broken | Fix before any demo |
| **Major** | Significant friction — user struggles or loses data context | Content truncated, controls hidden at breakpoint, no loading state | Fix before rollout |
| **Minor** | Noticeable issue — user can work around it | Inconsistent spacing, raw data labels, minor alignment | Fix in next polish sprint |
| **Cosmetic** | Visual polish — noticed by designers, not users | 1px misalignment, slightly wrong shade, font weight inconsistency | Fix when convenient |

---

## Evidence Hierarchy

**Screenshots and snapshots are evidence. Artifact claims are not.**

| Source Type | Examples | Treatment |
|-------------|----------|-----------|
| **Primary** (authoritative) | Screenshots, browser snapshots, axe-core output, computed styles | This IS the evidence |
| **Secondary** (claims to verify) | Prior audit findings, design docs, CLAUDE.md rules | Hypotheses — verify in browser |

When a prior audit says "spacing is wrong," verify it in the current page before reporting. Prior findings may be fixed.

---

## Investigation File Setup

**CRITICAL:** Create investigation file BEFORE starting the audit. Document findings progressively.

```bash
kb create investigation "audit/ux-{page-slug}" --model <model-name>  # or --orphan
```

**After creating the template:**
1. Fill Question field with target URL and audit mode
2. Update metadata (date, mode, viewports)
3. Document findings progressively during audit (don't wait until end)
4. Update metrics and comparison sections when completing

---

## Deliverables

**Investigation file:** `.kb/investigations/YYYY-MM-DD-audit-ux-{slug}.md`

**Required sections:**
- Baseline Metrics table (finding counts, axe-core results, delta from prior)
- Findings by Dimension (each with severity, viewport, evidence, impact, recommendation)
- What Works Well (positive findings)
- Comparison with Prior Audit (if exists)
- Screenshot Index
- Reproducibility (auth method, commands, re-audit schedule)

**Screenshots:** `.kb/investigations/screenshots/{audit-date}-{page-slug}/`

**Naming convention:**
- `baseline-{viewport}.png` — default page state at viewport width
- `interactive-{action}.png` — after an interaction
- `error-{state}.png` — error states
- `a11y-{check}.png` — accessibility-specific screenshots

---

## Standard Viewports

All audits use these five viewports for consistency:

| Width | Breakpoint | Purpose |
|-------|-----------|---------|
| 1280px | Desktop | Default desktop layout |
| 1024px | lg | Full desktop expansion |
| 768px | md | Minor tweaks only (NOT structural per CLAUDE.md) |
| 640px | sm | First structural layout shift |
| 375px | Mobile | iPhone SE / small mobile |

**CLAUDE.md breakpoint rules apply:**
- 640px (sm) = first structural shift (sidebar, grid, nav collapse)
- 768px (md) = minor tweaks only (spacing, font sizes) — NEVER layout structure
- 1024px (lg) = full desktop expansion

---

## Token Budget Strategy

Browser snapshots cost 5-15K tokens each. Be strategic:

- **Quick mode:** 1 snapshot at 1280px + 5 screenshots (one per viewport) + 1 axe-core scan
- **Focused mode:** 1-2 snapshots per viewport for target dimensions + screenshots
- **Full mode:** Snapshot at each viewport + dimension-specific snapshots as needed

**Rule:** Take screenshots (cheap) liberally. Take snapshots (expensive) strategically — only when you need the accessibility tree or DOM structure.
