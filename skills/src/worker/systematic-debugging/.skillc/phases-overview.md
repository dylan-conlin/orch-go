## The Four Phases

Complete each phase before proceeding to next.

### Phase 1: Root Cause Investigation

**Goal:** Understand WHAT, WHY, and WHERE

1. **Read error messages carefully** — stack traces completely, note line numbers and error codes
2. **Reproduce consistently** — if not reproducible, gather more data, don't guess
3. **Check recent changes** — git diff, new dependencies, config changes, environmental differences
4. **Whack-a-mole detection** — search git history for similar fixes (`git log --all --grep="[issue-type]"`). If 2+ similar fixes found, stop fixing symptoms and investigate the systemic cause
5. **Layer bias check** — UI shows wrong state? Check backend returns correct data BEFORE touching UI. Fix at the lowest layer that addresses root cause
6. **Multi-component systems** — add diagnostic instrumentation at each component boundary before proposing fixes. Run once, gather evidence showing WHERE it breaks, then investigate that component
7. **Trace data flow** — where does the bad value originate? Keep tracing up until you find the source
8. **Security assessment** — could this bug be exploited? If yes: `bd comments add <beads-id> "SECURITY: [type] - [description]"`

**Success criteria:** You understand root cause (not just symptoms), origin location, and why it's broken.

**Reference:** [reference/phase1-root-cause-detailed.md](reference/phase1-root-cause-detailed.md) for detailed techniques, examples, and multi-component diagnostics patterns.

---

### Phase 2: Pattern Analysis

**Goal:** Identify differences between working and broken

1. **Find working examples** in same codebase — what works that's similar?
2. **Read reference implementations completely** — don't skim, read every line
3. **List every difference** — however small, don't assume "that can't matter"
4. **Understand dependencies** — what components, config, environment, assumptions?

**Success criteria:** You know what's different between working and broken, and why it matters.

---

### Phase 3: Hypothesis Testing

**Goal:** Scientific method — form and test specific hypothesis

1. **Form single hypothesis** — "I think X is the root cause because Y" (be specific, write it down)
2. **Test minimally** — smallest possible change, one variable at a time
3. **Verify before continuing** — didn't work? Form NEW hypothesis, don't add more fixes on top
4. **When you don't know** — say so. Don't pretend. Ask for help, research more.

**Success criteria:** Hypothesis confirmed (proceed to Phase 4) or new hypothesis formed from test results.

---

### Phase 4: Implementation

**Goal:** Fix root cause, not symptom

1. **Create failing test case** — simplest reproduction, automated if possible
2. **Implement single fix** — ONE change, no "while I'm here" improvements
3. **Verify** — test passes, no regressions, issue actually resolved
4. **If fix doesn't work** — if <3 attempts: return to Phase 1 with new information. If ≥3: STOP and question architecture (see below)
5. **Smoke-test** — run the actual failing scenario, verify expected behavior, document in completion comment

**When to iterate vs escalate:**
- Keep iterating if: related issue in same area, direction correct, you understand why it failed
- Escalate if: 3+ fixes failed, root cause misidentified, issue outside scope

**Reference:** [reference/phase4-implementation-detailed.md](reference/phase4-implementation-detailed.md) for architecture questioning, whack-a-mole escalation, and systemic solution patterns.

---

## 3+ Fixes Failed: Question Architecture

If 3+ fix attempts failed OR whack-a-mole pattern detected:

- Is this pattern fundamentally sound, or inertia?
- Do we need centralized config/validation/infrastructure instead of scattered fixes?
- Escalate to orchestrator (may spawn `architect`)

This is NOT a failed hypothesis — this is a wrong architecture or missing infrastructure.

---

## Common Rationalizations (All Wrong)

| Excuse | Reality |
|--------|---------|
| "Issue is simple, don't need process" | Simple issues have root causes too |
| "Emergency, no time for process" | Systematic is FASTER than guess-and-check |
| "I see the problem, let me fix it" | Seeing symptoms ≠ understanding root cause |
| "One more fix attempt" (after 2+) | 3+ failures = architectural problem |
