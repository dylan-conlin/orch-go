# Probe: Orchestrator Skill Investigation Cluster — Contradiction Analysis

**Model:** orchestrator-session-lifecycle
**Date:** 2026-03-12
**Status:** Complete

---

## Question

The model's skill evolution section synthesizes findings from 6+ investigations and 3+ probes into a coherent narrative (82% token reduction, ≤4 behavioral constraint budget, 6/7 hooks, knowledge-framing principle). Does this narrative hold under cross-referential scrutiny, or do the source investigations contain contradictions, temporal invalidations, and recommendation conflicts that the synthesis smooths over?

---

## What I Tested

Read all 7 investigations and 3 probes in the orchestrator skill cluster, cross-referencing for:

1. **Direct contradictions** — one source claims X, another claims not-X
2. **Temporal invalidations** — findings true at time of investigation but invalidated by later work
3. **Unverified categorizations** — the tension mapping (inv #7) classifies 9 tensions; verified each against source evidence
4. **Recommendation conflicts** — incompatible recommendations across investigations

**Sources examined:**
- 7 investigations: behavioral testing baseline (Mar 1), testing infrastructure design (Mar 1), behavioral compliance (Feb 24), simplify skill (Mar 4), grammar-first architecture (Mar 4), skill update (Mar 5), tension mapping (Mar 11)
- 3 probes: behavioral compliance (Feb 24), constraint dilution (Mar 1), emphasis language (Mar 2)

---

## What I Observed

### 4 Direct Contradictions

**DC-1: Constraint dilution thresholds cited as established despite failed replication (SEVERITY: blocks model synthesis)**

- Probe Mar 1 (constraint dilution): Claims behavioral budget ~2-4, bare parity at 10C, knowledge budget ~50
- Probe Mar 1 CAVEAT (added Mar 4): "The dilution curve did not replicate under clean isolation. All quantitative claims should be treated as directional hypotheses, not established findings."
- Yet: Investigation #4 (Mar 4 simplification) uses "≤4 behavioral constraints before dilution" as a design constraint. Investigation #5 (Mar 4 grammar-first) uses "~4 behavioral constraints" as the budget. Investigation #7 (Mar 11 tension mapping) treats these as evidence for fundamental tension categorization.
- The model itself (orchestrator-session-lifecycle) states: "budget for reliable behavioral compliance is ~2-4 co-resident constraints"
- **Mechanism:** The replication failure caveat was appended to the probe but NOT propagated to the 4 downstream artifacts that depend on it. The quantitative thresholds percolated into the design space as hard constraints despite being unvalidated.

**DC-2: 10C behavioral performance contradicts across sessions (SEVERITY: complicates but manageable)**

- Probe Mar 1 (constraint dilution): 10C-emphasis proposes-delegation = 0/3 (bare parity)
- Probe Mar 2 (emphasis language): 10C-emphasis proposes-delegation = 2/3 (above bare)
- Same test file, same model, same experimental framework. Combined: 2/6 (33%).
- Both probes acknowledge N=3 variance but neither resolves the contradiction. The model presents the Mar 1 finding (bare parity at 10) as canonical.

**DC-3: Testing infrastructure claimed fixed but still broken (SEVERITY: blocks model synthesis)**

- Investigation #1 (Mar 1 behavioral testing): "The `skillc test` isolation fix (strip `CLAUDECODE` env var, run from clean CWD) enables behavioral testing from within Claude Code sessions and spawned agents."
- Investigation #4 (Mar 4 simplification): "`skillc test` requires nested `claude --print` calls. From spawned agent: CLAUDECODE env var blocks nested sessions. `skillc test` env stripping doesn't work in current version"
- The Mar 1 claim that the fix was implemented was invalidated 3 days later. This means the behavioral testing baseline (Investigation #1) may have been run under different conditions than claimed.

**DC-4: Emphasis language as anti-pattern vs compliance tool (SEVERITY: complicates but manageable)**

- Investigation #2 (Mar 1 testing infrastructure): Proposes MUST-density linter that warns when MUST/NEVER/CRITICAL exceeds 3 per 100 words. Treats emphasis language as an anti-pattern ("MUST fatigue").
- Probe Mar 2 (emphasis language): Shows emphasis language provides measurable compliance lift over neutral language (proposes-delegation 2/3 vs 0/3 at 10C). Treats emphasis language as beneficial.
- These are operationally contradictory: the linter would flag the very language that the experiment shows helps.

### 5 Tension Points

**T-1: Simplify the skill vs add knowledge items (SEVERITY: complicates but manageable)**

- Investigation #4 (Mar 4): Strip to ~450 lines, remove dead weight
- Investigation #5 (Mar 4): Each behavioral slot needs ~5-8 calibration knowledge items (~27 total minimum)
- Investigation #6 (Mar 5): 13 specific edits adding +5 main skill lines, +4 reference lines
- The model notes post-simplification regrowth at +24% tokens in 7 days. Each investigation is individually correct but they collectively recreate the accretion cycle the simplification was meant to break.

**T-2: 4 behavioral slots — which 4? (SEVERITY: complicates but manageable)**

- Investigation #4 (Mar 4 simplification): delegation, filter, act-by-default, answer-the-question
- Investigation #5 (Mar 4 grammar-first): delegation, undefined behavior handler, filter+act-by-default (combined), pressure over compensation
- Same day, different agents, incompatible slot selections. Investigation #5 argues "answer the question asked" is knowledge wearing behavioral clothing. Neither was validated by behavioral testing.

**T-3: "Don't iterate content" vs detailed content iteration plans (SEVERITY: minor)**

- Investigation #1 (Mar 1 behavioral testing): "Don't revert or iterate the skill based on content analysis alone."
- Investigation #5 (Mar 4): Detailed content iteration plan for 4 behavioral slots
- Investigation #6 (Mar 5): 13 specific content edits
- Investigation #1's recommendation was effectively ignored by subsequent work.

**T-4: Infrastructure sufficient vs prompt still needed for soft preferences (SEVERITY: complicates but manageable)**

- Investigation #3 (Feb 24): Two-layer fix needed
- Investigation #7 (Mar 11): "Some action constraints can't be hooked — e.g., choosing `orch spawn` over `bd create -l triage:ready`"
- The two-layer design is correct but the boundary between hookable and unhookable constraints is undefined. The model doesn't specify which constraints remain prompt-dependent after infrastructure enforcement.

**T-5: Behavioral testing necessary vs impossible from agent sessions (SEVERITY: blocks model synthesis)**

- Investigation #1 (Mar 1): Created testing framework, claimed it works from agents
- Investigation #4 (Mar 4): Deployed v4 without testing because framework doesn't work from agents
- Investigation #7 (Mar 11): Lists testing as "live tension"
- Result: Every design decision in the cluster (constraint budget, slot selection, simplification target) lacks behavioral validation. The entire design was built on a testing capability that doesn't reliably work.

### Tension Mapping Verification (Investigation #7's 9 categorizations)

**Fundamental (3):**

1. **Knowledge-transfer vs behavioral-constraint:** PARTIALLY SUPPORTED. The qualitative finding (knowledge sticks, constraints don't) is well-evidenced by Investigation #1. The quantitative thresholds (≤4 behavioral, ≤50 knowledge) rest on unreplicated dilution data. Should be labeled "fundamental with unvalidated quantitative bounds."

2. **Skill-as-grammar vs skill-as-probability-shaper:** WEAKLY SUPPORTED. Investigation #7 cites a "formal grammar theory investigation" that is NOT in the 7 investigations analyzed here. The tension is conceptually sound but the primary evidence source wasn't available for verification.

3. **Simplicity vs completeness:** WELL SUPPORTED. The line count trajectory (640→2,368→448→512) is verifiable from multiple investigations. The oscillation pattern is real.

**Resolved (4):**

1. **Prompt vs infrastructure enforcement:** SUPPORTED. 6/7 hooks verified working per Investigation #4.

2. **Accretion vs simplification:** OVERSTATED. "Resolved" implies stability. Investigation #6 (just 1 day after Investigation #4) already adds +5 lines. The model itself notes +24% regrowth in 7 days. Should be "managed by budget, not resolved."

3. **Identity vs action compliance (structural):** SUPPORTED for structural component. But Investigation #7 itself lists residual action compliance as a live tension — it splits the same tension into "resolved (structural)" and "live (residual)," which is internally coherent but could confuse future readers.

4. **Orchestrator-centric vs Dylan-centric organization:** SUPPORTED.

**Live (2):**

1. **Testing feasibility vs measurement need:** SUPPORTED. The testing infrastructure genuinely doesn't work from agent sessions despite Investigation #1's claim it does.

2. **Identity vs action compliance (residual):** SUPPORTED. Soft preferences remain probabilistic.

### 2 Temporal Contradictions

**TC-1: skillc test isolation fix (Mar 1 → Mar 4)**

- Mar 1: "implemented as part of this investigation" — two changes to `skillc/pkg/scenario/runner.go`
- Mar 4: "env stripping doesn't work in current version (returns 0/0 scores)"
- Either the fix was incomplete, regressed, or the Mar 1 investigation ran tests in a different environment than claimed. This undermines the behavioral baseline data's reliability.

**TC-2: v4 norms → grammar-first norms (both Mar 4)**

- Investigation #4 proposes 4 norms (v4 deployed)
- Investigation #5 proposes different 4 norms (not deployed)
- Neither subsequent investigation (#6 or #7) clarifies which set was adopted. The current deployed skill (512 lines per Investigation #7) presumably contains v4's norms, but Investigation #5's recommendations were never explicitly rejected or accepted.

### 3 Recommendation Conflicts

**RC-1: Content iteration approach**

- Investigation #1: "Don't revert or iterate the skill based on content analysis alone." "Invest in Layer 2 (infrastructure enforcement)."
- Investigation #5: Detailed content iteration proposal (swap 2 of 4 behavioral slots, add calibration knowledge, reformulate norms)
- Investigation #6: 13 specific content edits
- Conflict: Investigation #1 deprioritizes content changes; Investigations #5 and #6 are entirely content changes.

**RC-2: Emphasis language treatment**

- Investigation #2 (testing infrastructure): Proposes linter with MUST-density check (>3 per 100 words = warning). Sourced from "DSL investigation" on MUST fatigue.
- Probe Mar 2 (emphasis): Shows emphasis language provides compliance lift, especially at high constraint counts. Recommends preserving emphasis for the 2-5 constraint sweet spot.
- These are directly incompatible: the linter would flag effective emphasis language.

**RC-3: Investment in testing infrastructure**

- Investigation #2: "This infrastructure is worth building only if Dylan is entering a phase of systematic skill iteration"
- Investigation #1: "Run tests before deploying skill changes" (gating requirement)
- Investigation #4: Deployed v4 without testing because infrastructure doesn't work
- Collectively: testing is recommended as a gate, the gate doesn't work, and the investment is conditional on iteration frequency — but iterations happened anyway (3 in 4 days) without the gate.

---

## Model Impact

- [x] **Contradicts** invariant: The model states "budget for reliable behavioral compliance is ~2-4 co-resident constraints" as established. The source probe carries a replication failure caveat that was never propagated to the model. The quantitative thresholds should be downgraded to "directional hypothesis" until replicated.

- [x] **Extends** model with: The investigation cluster contains a propagation failure pattern — findings are treated as established by downstream investigations even after the source is marked unreliable (dilution thresholds, testing fix). The model should document this as a methodological risk: **probe-to-downstream propagation delay** — caveats added to probes do not automatically propagate to investigations and designs that cited the original uncaveated version.

- [x] **Extends** model with: The recommendation corpus contains 3 irreconcilable conflicts (content iteration, emphasis language, testing investment). The model's narrative smooths these into a coherent progression. A more accurate representation would note that the investigation cluster produced a design SPACE (multiple valid configurations) rather than a single design.

- [x] **Confirms** invariant: The fundamental tension categorizations (knowledge vs behavioral, simplicity vs completeness) are well-supported. The qualitative findings from Investigation #1 (knowledge sticks, constraints don't) and Investigation #3 (identity compliance ≠ action compliance) are robust and consistent across all sources.

---

## Notes

- The most consequential finding is DC-1 (dilution thresholds used as hard constraints despite failed replication). The entire "≤4 behavioral norms" budget that shapes Investigation #4, #5, and the current deployed skill rests on unvalidated data. The qualitative direction is likely correct (fewer constraints = better compliance), but the specific number 4 has no more empirical support than 3 or 6.
- DC-3 (testing fix claimed but broken) means the behavioral baseline data from Investigation #1 should carry an environment caveat — we can't be certain it ran under the isolation conditions it describes.
- The recommendation conflicts (RC-1, RC-2, RC-3) aren't necessarily errors — they reflect genuine disagreement between investigations that were never reconciled because no single investigation reads all others. The tension mapping (Investigation #7) was the first attempt at reconciliation but smoothed over the conflicts rather than surfacing them.
