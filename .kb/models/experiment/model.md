# Model: Experiment

**Created:** 2026-03-09
**Status:** Active
**Source:** Synthesized from 4 investigation(s)

## What This Is

[What phenomenon or pattern does this model describe? What makes it a coherent concept worth naming?]

---

## Core Claims (Testable)

### Claim 1: [Concise claim statement]

[Explanation of the claim. What would you observe if it's true? What would falsify it?]

**Test:** [How to test this claim]

**Status:** Hypothesis

### Claim 2: [Concise claim statement]

[Explanation of the claim.]

**Test:** [How to test this claim]

**Status:** Hypothesis

---

## Implications

[What follows from these claims? How should this model change behavior, design, or decision-making?]

---

## Boundaries

**What this model covers:**
- [Scope item 1]

**What this model does NOT cover:**
- [Exclusion 1]

---

## Evidence

| Date | Source | Finding |
|------|--------|---------|
| 2026-03-09 | Model creation | Initial synthesis from source investigations |

---

## Open Questions

- [Question that further investigation could answer]
- [Question about model boundaries or edge cases]

## Source Investigations

### 2026-03-03-inv-experiment-post-hooks-behavioral-baseline.md

**Delta:** [What was discovered/answered - the key finding in one sentence]
**Evidence:** [Primary evidence that supports the conclusion - test results, observations]
**Knowledge:** [What was learned - insights, constraints, or decisions made]
**Next:** [Recommended action - close, implement, investigate further, or escalate]

---

### 2026-03-05-inv-experiment-comprehension-calibration-contrastive-scenarios.md

**Delta:** Stance does NOT generalize to all attention types. Of 3 new scenarios testing absence, relationship-tracing, and information-freshness, NONE showed three-way separation (bare < without-stance < with-stance). Scenario 12 (downstream consumer) showed strong knowledge lift (bare 17% → skill 83%) but no stance advantage. Scenario 11 (auth gap) hit ceiling on all variants at 3/8. Scenario 13 (stale deprecation) showed knowledge lift but with-stance scored WORSE than without-stance.
**Evidence:** 144 total trials. Initial round: 54 (N=3 x 3 scenarios x 3 variants x 2 rounds). Higher-N: 36 (N=6 x 2 scenarios x 3 variants for 09+10). Generalization: 54 (N=6 x 3 scenarios x 3 variants for 11-13). Scenario 09 stance gap confirmed: 83% vs 17% vs 0%. Scenarios 11-13: 0 of 3 show stance advantage.
**Knowledge:** Stance as attention priming is scenario-09-specific, not a general mechanism. It works for implicit contradiction (two agents with incompatible assumptions) but not for: (1) absence detection — Sonnet already notices missing auth without stance, (2) relationship tracing — knowledge provides the lift, not stance orientation, (3) information freshness — temporal reasoning is hard and stance doesn't help. The distinction may be that scenario 09 requires *holding two things in mind simultaneously*, while the other attention types require different cognitive operations.
**Next:** Revise skill-content-transfer model to narrow stance claims. Scenario 09 remains the primary stance discriminator. Investigate what makes implicit contradiction uniquely responsive to stance. Consider whether scenarios 11/13 have indicator design issues vs genuine stance-irrelevance.

---

### 2026-03-05-inv-experiment-stance-generalization-11-13.md

**Delta:** Stance generalizes selectively — strong lift on relationship tracing (+4.5 median) and information freshness (+4.0 median), no lift on absence detection (bare already moderate at median 5). The prerequisite for stance lift is low bare detection; stance primes cross-source reasoning, not pattern visibility.
**Evidence:** 36 trials (N=6 x 3 scenarios x 2 variants). Scenario 12: bare median 1.5 → stance median 6 (notices-consumer-impact: 3/6→6/6). Scenario 13: bare median 0 → stance median 4 (connects-git-evidence: 1/6→6/6). Scenario 11: bare median 5 → stance median 3 (no lift — auth gap is structurally visible). Action indicators (recommends-fix, no-premature-completion) remain at floor across all variants.
**Knowledge:** Stance is cross-source reasoning primer, not generic attention amplifier. It helps when the defect lives in the GAP between information sources (query change vs dashboard assumptions, deprecation claim vs git log). It doesn't help when the defect is structurally visible within a single source (auth middleware pattern). Action indicators need redesign — they're non-discriminating due to vocabulary limitations.
**Next:** (1) Redesign action indicators (no-premature-completion, recommends-fix) to capture hedged approval language. (2) Test whether adding behavioral constraints (in addition to stance) closes the detection-to-action gap. (3) Consider an intermediate-difficulty absence scenario where bare doesn't already detect.

---

### 2026-03-06-inv-experiment-detection-to-action-gap.md

**Delta:** Behavioral constraints partially close the detection-to-action gap — S09 `recommends-before-closing` moved 0/6→3/6 — but the effect is scenario-dependent, not universal.
**Evidence:** 36 trials (3 variants × 2 scenarios × N=6). S09 shows clean lift on action indicator. S13 `no-blind-removal` stayed at 0/6 but is likely a confounded indicator (prompt contains "safe to remove" text).
**Knowledge:** Detection-to-action gap is partially prompt-solvable for action types that map to "stop and escalate." The gap is harder to close when the required action is absence of specific language, especially when that language appears in the prompt itself. Indicator design matters as much as constraint design.
**Next:** (1) Redesign S13 `no-blind-removal` indicator to avoid prompt-text confound. (2) Test whether higher N on S09 stabilizes the 3/6 rate or reveals more variance. (3) Investigate whether "stop and escalate" actions are categorically more prompt-solvable than "suppress default language" actions.
