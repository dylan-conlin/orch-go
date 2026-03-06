# Probe: Worker Skill Industry Practice Gap Audit

**Model:** skill-content-transfer
**Date:** 2026-03-06
**Status:** Complete

---

## Question

The skill-content-transfer model claims skills contain three content types: **knowledge** (routing tables, templates), **behavioral constraints** (NEVER/MUST prohibitions), and **stance** (epistemic orientation). Critical Invariant 1 says skills should be ≤500 lines/5,000 tokens.

**This probe tests:** Do the current worker skills contain the right *knowledge* for their domain? Specifically, are there standard industry software engineering practices that agents implementing features need to know about, but that aren't represented in any skill content? The feature-gate gap discovered in feature-impl (orch-go-t250c) suggests knowledge gaps exist. If so, this extends the model: knowledge gaps aren't just about missing routing tables — they're about missing domain practices that agents can't infer from bare capabilities.

---

## What I Tested

Full audit of all 10 worker skills against industry best practices across 9 categories:
- Accessibility (a11y), Security, Performance, Error handling, Observability, Testing, Deployment, Code quality, Browser compatibility

**Skills audited:**
1. feature-impl (5,105 tokens, 10 phases)
2. systematic-debugging (6,265 tokens, 4 phases + self-review)
3. ux-audit (35,000 tokens, 6 dimensions)
4. investigation (5,000 tokens)
5. research (standard budget)
6. codebase-audit (15,000 tokens, 5 dimensions)
7. design-session (standard budget)
8. architect (loaded in this session)
9. experiment (3,000 tokens)
10. experiential-eval (1,500 tokens)

**Method:** Read complete skill source (.skillc/ directories), compared checklists and phase guidance against what a senior engineer at a mature company would expect for each skill's domain.

---

## What I Observed

### feature-impl — The Primary Gap Carrier

Feature-impl is where most production code gets written. It has strong coverage of:
- ✅ Security (SQL injection, XSS, path traversal, command injection, hardcoded secrets)
- ✅ TDD/testing methodology
- ✅ Self-review (anti-patterns, commit hygiene, integration wiring)
- ✅ Feature gates (newly added, orch-go-t250c)
- ✅ Visual verification for UI changes
- ✅ Harm assessment (pre-implementation ethics)
- ✅ Demo/placeholder data ban

**Missing (HIGH impact):**

1. **Accessibility (a11y) — ABSENT.** Zero mentions of WCAG, screen reader, keyboard navigation, semantic HTML, ARIA attributes, color contrast, or alt text in any feature-impl phase. Agents building UI features will produce inaccessible interfaces. This is the single largest gap. Any mature company requires WCAG AA compliance for new features.

2. **Performance regression check — ABSENT.** No mention of bundle size impact, Web Vitals (LCP, FID, CLS), lighthouse budgets, lazy loading for new routes/components, or checking for N+1 queries. New features can silently degrade performance.

3. **Error boundary/graceful degradation — ABSENT.** No requirement to wrap new Svelte components in error boundaries, handle API failures gracefully, or ensure user-facing error messages are helpful (not raw stack traces). The security checklist catches injection but not user-facing error quality.

4. **Observability — ABSENT.** No mention of adding error tracking integration, structured logging, or metrics for new features. Features ship "dark" — if they fail in production, nobody knows.

5. **Dependency audit — ABSENT.** No mention of `npm audit`, checking new packages for CVEs, license compatibility, or maintenance status before adding.

**Missing (MEDIUM impact):**

6. **Loading/skeleton states for async operations — NOT in feature-impl.** (ux-audit catches this, but feature-impl agents don't see ux-audit guidance.)

7. **Cross-browser basic sanity — ABSENT.** No mention of testing in Safari/Firefox even once.

### systematic-debugging — Mostly Complete, One Gap

- ✅ Root cause methodology (strong)
- ✅ Smoke-test requirement
- ✅ Pattern scope verification
- ✅ Self-review

**Missing:**

8. **Security impact assessment of discovered bugs — ABSENT.** When a debugging agent finds a bug (e.g., "input not validated"), there's no check for whether it's also a security vulnerability that needs urgency escalation, a CVE filing, or a security advisory. The security review in self-review only checks the *fix* for injection, not whether the *bug itself* was a security issue.

### ux-audit — Most Comprehensive, Performance Gap

- ✅ Accessibility (excellent — axe-core, WCAG AA, keyboard nav, ARIA, heading hierarchy, landmarks, contrast)
- ✅ Responsive (5 viewports with CLAUDE.md breakpoint rules)
- ✅ Visual consistency (design token verification)
- ✅ Data presentation (formatting, empty states)
- ✅ Navigation (active states, error recovery, deep links)
- ✅ Interactive states (hover, loading, error, success, transitions)

**Missing:**

9. **Performance dimension — ABSENT.** No lighthouse audit, no Web Vitals measurement, no load time checks, no bundle size analysis. The 6 dimensions cover visual/structural/interactive quality but not speed. A senior FE engineer would always include performance in a UX audit.

### codebase-audit — Missing Dimension

- ✅ Security dimension
- ✅ Performance dimension
- ✅ Test dimension
- ✅ Architecture dimension
- ✅ Organizational dimension

**Missing:**

10. **Accessibility audit dimension — ABSENT.** Has 5 audit dimensions but no a11y dimension. Periodic codebase audits never check for WCAG violations, missing alt text, unlabeled form fields, or inaccessible component patterns.

11. **Dependency health dimension — ABSENT.** No dimension for checking outdated dependencies, known CVEs, license compliance, or abandoned packages.

### design-session — Non-Functional Requirements Gap

**Missing:**

12. **Non-functional requirements prompting — ABSENT.** When scoping a feature, no mention of asking about a11y requirements, performance budgets, security constraints, or error handling expectations. These get discovered during implementation instead of design.

### research — Recommendation Safety Gap

**Missing:**

13. **Security/health vetting of recommended tools — ABSENT.** When recommending a library, no check for CVE history, npm audit status, maintenance activity, or license compatibility.

### investigation, experiment, experiential-eval — No Significant Gaps

These are research/meta skills. Industry practice categories (a11y, security, performance) are implementation concerns, not investigation concerns. No gaps relevant to their purpose.

### architect (this session's skill) — Minor Gap

14. **Non-functional requirements in design exploration — WEAK.** The fork navigation doesn't explicitly prompt for a11y, performance, or security as standard fork dimensions. It relies on the architect knowing to check these.

---

## Model Impact

- [x] **Extends** model with: Knowledge gaps in skills aren't limited to missing routing tables or vocabulary — they include **missing domain practices** that agents can't infer from bare capabilities. The skill-content-transfer model correctly identifies knowledge as "routing tables, templates, vocabulary" but should extend to include **domain checklists** — standard practices that a skilled human would apply but that an agent won't without explicit prompting.

  **Specifically:** The model's Invariant 1 (≤500 lines/5,000 tokens) creates a tension: feature-impl is already at 5,105 tokens. Adding a11y, performance, observability checklists would push it over. The model's recommendation to "strip behavioral weight to hooks" applies here: domain checklists should be **knowledge** (framed as "here's what to check") not **behavioral constraints** (framed as "NEVER ship without a11y"). This means they can fit within the token budget if written as concise knowledge references rather than verbose prescriptions.

  **Proposed resolution:** Add domain practice checklists to feature-impl as compact knowledge sections (not MUST/NEVER constraints), reference ux-audit's existing a11y methodology where possible, and consider a pre-implementation "non-functional requirements" step in design-session that prompts for a11y/perf/security requirements upfront.

- [x] **Confirms** invariant: "Knowledge framing, not prohibition" (Invariant 3). The feature-gate addition (orch-go-t250c) was added as a knowledge-framed checklist item ("Feature gate status declaration: gated|ungated|N/A"), not a NEVER/MUST prohibition. This confirms the model's recommendation works for domain practice gaps.

---

## Notes

### The a11y Gap is Systemic

The accessibility gap is the most significant finding. It exists in 3 of the 4 implementation-relevant skills:
- **feature-impl:** Where a11y code should be written — no guidance
- **codebase-audit:** Where a11y issues should be caught periodically — no dimension
- **design-session:** Where a11y requirements should be specified — no prompting

Only **ux-audit** has comprehensive a11y coverage, but ux-audit runs after features are built. The cost of retrofitting a11y is 5-10x higher than building it in.

### Token Budget Tension

feature-impl is at 5,105 tokens (vs 5,000 budget). Adding comprehensive a11y + performance + observability guidance would push it significantly over. Options:
1. Add concise checklist items (like feature-gate) — fits within ~200 additional tokens
2. Extract to reference docs (progressive disclosure pattern already used by investigation skill)
3. Add as a hook/gate (enforcement without token cost in skill)
4. Create a "non-functional requirements" pre-flight skill that runs before feature-impl

Option 1 (concise checklist items) is recommended for a11y and performance. Option 2 for detailed methodology.
