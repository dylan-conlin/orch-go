# Investigation: Worker Skill Industry Practice Gap Audit

**Question:** What standard software engineering best practices are missing from our worker skills, and which gaps would cause real damage if unaddressed?
**Started:** 2026-03-06
**Updated:** 2026-03-06
**Status:** Complete
**Phase:** Complete
**Beads:** orch-go-9b6do

---

## Context

Feature gating — a standard industry practice — was discovered missing from feature-impl and added (orch-go-t250c). This raised the question: what other standard practices are missing across all worker skills?

## Methodology

Audited all 10 worker skills against 9 industry best practice categories: accessibility, security, performance, error handling, observability, testing, deployment, code quality, browser compatibility. Compared each skill's checklists and phase guidance against what a senior engineer at a mature company would expect.

---

## Findings

### Finding 1: Accessibility (a11y) is Systemically Absent from the Implementation Path

**Severity:** HIGH — would prevent real damage

**Evidence:**
- **feature-impl:** Zero mentions of WCAG, screen reader, keyboard navigation, semantic HTML, ARIA, color contrast, or alt text across all 10 phases and 5,105 tokens of guidance.
- **codebase-audit:** 5 audit dimensions (security, performance, tests, architecture, organizational) — no a11y dimension.
- **design-session:** No prompting for a11y requirements when scoping features.

**Only ux-audit has a11y coverage** — but it runs *after* features are built. Retrofitting a11y is 5-10x costlier than building it in.

**What this means:** Every UI feature implemented by a feature-impl agent ships without:
- Keyboard navigability
- Screen reader compatibility
- Color contrast compliance (WCAG AA 4.5:1)
- Semantic HTML (proper heading hierarchy, landmarks, labels)
- ARIA attributes for dynamic content

**Recommendation:** Add a concise a11y checklist to feature-impl's self-review (knowledge framing, ~100 tokens). Add a11y as a dimension to codebase-audit.

---

### Finding 2: No Performance Regression Checking

**Severity:** HIGH — silent degradation

**Evidence:**
- **feature-impl:** No mention of bundle size impact, Web Vitals (LCP, FID, CLS), lighthouse budgets, lazy loading for new routes, or N+1 query detection.
- **ux-audit:** 6 dimensions but no performance dimension (no lighthouse, no load time, no Web Vitals).
- **design-session:** No performance budget prompting during scoping.

**What this means:** Features can silently degrade performance. No agent checks whether their new component adds 500KB to the bundle or their new API call introduces an N+1 query.

**Recommendation:** Add performance check to feature-impl validation phase. Add performance dimension to ux-audit.

---

### Finding 3: No Error Boundary / Graceful Degradation Requirement

**Severity:** HIGH — user-facing damage

**Evidence:**
- **feature-impl:** Security checklist catches injection attacks but doesn't check:
  - Error boundaries wrapping new Svelte components
  - API failure handling (what happens when fetch fails?)
  - User-facing error messages (helpful vs raw stack trace)
  - Loading states for async operations
  - Empty states for data-dependent components

**What this means:** New components crash the entire app when they error instead of failing gracefully. API failures show raw error text to users.

**Recommendation:** Add error handling checklist to feature-impl's self-review. This overlaps with ux-audit's interactive-states dimension — extract the implementation-time requirements into feature-impl.

---

### Finding 4: No Observability / Monitoring Guidance

**Severity:** MEDIUM — operational blindness

**Evidence:**
- **feature-impl:** No mention of error tracking integration, structured logging, or metrics for new features.
- **systematic-debugging:** No mention of adding monitoring to prevent bug recurrence.

**What this means:** Features ship "dark" — if they fail in production, nobody knows until a user reports it. Debugging agents fix bugs but don't add monitoring to catch recurrence.

**Recommendation:** Add observability checklist to feature-impl (where applicable). Note: for a single-developer tool, this is medium priority — monitoring infrastructure may not exist yet.

---

### Finding 5: No Dependency Audit for New Packages

**Severity:** MEDIUM — security and maintenance risk

**Evidence:**
- **feature-impl:** No mention of `npm audit`, checking new packages for CVEs, license compatibility, or maintenance status.
- **research:** When recommending libraries, no check for security posture.
- **codebase-audit:** No dependency health dimension.

**What this means:** Agents add npm packages without checking for known vulnerabilities, incompatible licenses, or abandoned maintenance.

**Recommendation:** Add one-line check to feature-impl when adding new dependencies: "Run `npm audit` / `go vet` after adding new dependencies." Add dependency health as a codebase-audit dimension.

---

### Finding 6: Debugging Agents Don't Assess Security Impact of Bugs

**Severity:** MEDIUM — missed security escalation

**Evidence:**
- **systematic-debugging:** Self-review checks the *fix* for injection vulnerabilities but doesn't check whether the *bug itself* represents a security issue requiring urgent escalation.

**What this means:** A debugging agent finding "user input passed directly to SQL query" would fix it as a regular bug without flagging it as a security vulnerability needing urgency.

**Recommendation:** Add security impact question to systematic-debugging Phase 1: "Could this bug be exploited? If yes, flag as security issue."

---

### Finding 7: Design Phase Doesn't Prompt for Non-Functional Requirements

**Severity:** MEDIUM — requirements discovered too late

**Evidence:**
- **design-session:** Context gathering (Phase 1) checks principles, models, decisions, and issues — but doesn't prompt for a11y requirements, performance budgets, security constraints, or error handling expectations.
- **architect:** Fork navigation doesn't explicitly list a11y/perf/security as standard dimensions to evaluate.

**What this means:** Non-functional requirements are discovered during implementation instead of design. By that point, the architecture may not support them well.

**Recommendation:** Add "non-functional requirements" to design-session's Phase 2 scoping forks. Add as standard fork dimensions in architect.

---

### Finding 8: No Cross-Browser Basic Sanity

**Severity:** LOW for this project

**Evidence:**
- **feature-impl:** No mention of testing in Safari, Firefox, or any browser besides the default.
- **ux-audit:** Tests viewports (responsive) but not different browser engines.

**What this means:** Features might work in Chrome but break in Safari (WebKit differences are common with CSS Grid, date handling, etc.).

**Recommendation:** Note but skip for now. Single-developer internal tool. If web UI becomes user-facing, add cross-browser to ux-audit.

---

## Synthesis

### The Gap Pattern

The gaps follow a clear pattern: **feature-impl is strong on "don't break things" (security, testing, integration wiring) but weak on "build things right" (a11y, performance, error UX, observability).** The skill was designed to prevent damage from bad code but not to ensure quality of the user experience.

This makes sense historically — security and testing prevent immediate breakage; a11y and performance degrade silently. But silent degradation compounds.

### Priority Ranking (by damage prevention potential)

| # | Gap | Skill(s) | Priority | Recommendation |
|---|-----|----------|----------|---------------|
| 1 | **Accessibility absent** | feature-impl, codebase-audit, design-session | P1 | Add to skill text (checklist items) |
| 2 | **Performance regression check absent** | feature-impl, ux-audit | P1 | Add to skill text (validation phase) |
| 3 | **Error boundary / graceful degradation absent** | feature-impl | P1 | Add to skill text (self-review) |
| 4 | **Observability absent** | feature-impl, systematic-debugging | P2 | Add to skill text (advisory) |
| 5 | **Dependency audit absent** | feature-impl, codebase-audit | P2 | Add to skill text (1-line check) |
| 6 | **Security impact assessment absent** | systematic-debugging | P2 | Add to skill text (Phase 1) |
| 7 | **Non-functional requirements absent** | design-session, architect | P2 | Add to skill text (scoping phase) |
| 8 | **Cross-browser testing absent** | feature-impl, ux-audit | P4 | Skip for now |

### Token Budget Consideration

feature-impl is already at 5,105 tokens (budget: 5,000). The skill-content-transfer model's Invariant 1 (≤500 lines/5,000 tokens) constrains what can be added inline.

**Recommended approach (per model guidance):**
- Add concise knowledge-framed checklist items to feature-impl self-review (~150 additional tokens)
- Use progressive disclosure for detailed methodology (reference docs, like investigation skill does)
- DO NOT add as MUST/NEVER behavioral constraints (per Invariant 3: "Knowledge framing, not prohibition")

### What Each Skill Gets

**feature-impl self-review additions (~150 tokens):**
```markdown
#### 11. Accessibility Check (UI Features)
- [ ] Semantic HTML (proper headings, landmarks, form labels)
- [ ] Keyboard navigable (Tab/Enter/Escape work, no focus traps)
- [ ] Color contrast sufficient (4.5:1 for text)
- [ ] ARIA attributes for dynamic content (aria-live, aria-expanded)

#### 12. Performance Check
- [ ] No unnecessary re-renders or large bundle additions
- [ ] Async data has loading states
- [ ] New dependencies audited (`npm audit` / `go vet`)

#### 13. Error Resilience
- [ ] API failures handled gracefully (not raw error display)
- [ ] Error boundaries wrap new components
- [ ] Empty/error/loading states defined
```

**systematic-debugging Phase 1 addition (~30 tokens):**
```markdown
**Security check:** Could this bug be exploited by a malicious actor? If yes, escalate as security issue.
```

**codebase-audit new dimension (~description in skill.yaml):**
- `accessibility` dimension for WCAG compliance scanning

**design-session Phase 2 addition (~50 tokens):**
```markdown
**Non-functional requirements fork:** Does this feature need:
- Accessibility requirements? (WCAG level, screen reader support)
- Performance budget? (load time, bundle size ceiling)
- Security constraints? (auth, input validation, data sensitivity)
```

**ux-audit addition (~dimension outline):**
- `dimension-performance` for lighthouse/Web Vitals/load time

---

## Recommendations

⭐ **RECOMMENDED:** Phased implementation

**Phase 1 (immediate, low-effort):** Add concise checklist items to feature-impl self-review — a11y, performance, error resilience. ~150 tokens, fits within budget with minor rewrite of existing sections.

**Phase 2 (next sprint):** Add non-functional requirements prompting to design-session. Add security impact assessment to systematic-debugging. Add accessibility dimension to codebase-audit.

**Phase 3 (when needed):** Add performance dimension to ux-audit. Consider cross-browser testing if web UI goes user-facing.

**Alternative: Create a "quality-gates" pre-flight hook**
- Pros: Enforcement without token cost, catches gaps at spawn time
- Cons: Hook infrastructure overhead, another moving part
- When to choose: If agents consistently ignore checklist items despite skill text

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
This audit identifies systemic gaps that will recur until addressed.

**Suggested blocks keywords:**
- "accessibility", "a11y", "WCAG"
- "performance budget", "lighthouse"
- "error boundary", "graceful degradation"
- "feature-impl skill", "worker skill"
