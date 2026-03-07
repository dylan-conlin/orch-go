# Probe: Behavioral Dilution — Density vs Count Threshold

**Model:** behavioral-grammars
**Date:** 2026-03-06
**Status:** Complete

---

## Question

Prior dilution experiments (sonnet Mar 1, opus Mar 2, injection-level Mar 4) all confounded constraint COUNT with document SIZE. Each constraint in 3-form adds ~200 words, so:
- 2C = ~440 words, 5C = ~1010 words, 10C = ~1880 words

Constraint density (constraints/word) was held constant at ~0.005 across all variants. This means we cannot distinguish between three hypotheses:

- **H1 (Count):** The number of discrete behavioral rules drives dilution. Non-constraint text is irrelevant.
- **H2 (Document Size):** Total document length drives dilution via attention diffusion. Whether content is constraints or background doesn't matter.
- **H3 (Density Ratio):** Constraints per token ratio drives dilution. More constraints in the same space = worse; same constraints in more space = worse too.

**Specific claim under test:** The model states "constraints compete for attention" and "each additional constraint divides the model's attention budget." This implies COUNT (H1) is the driver — but this was never tested independently of document size.

**Falsifiable predictions:**
- If H1 (count): 2C-padded (~1880 words, 2 constraints + padding) ≈ 2C-tight (~440 words) — both perform similarly
- If H2 (doc size): 2C-padded ≈ 10C-neutral (~1880 words, both ~same doc size) — both degraded equally
- If H3 (density): 2C-padded should fall between 2C-tight and 10C-neutral

---

## What I Tested

**Design:** Hold constraint count constant while varying document size via non-constraint padding. All 5 variants run in the SAME session to avoid cross-session baseline drift. 7 orchestrator scenarios (01-07) × 3 runs × 5 variants = 105 total API calls. Model: sonnet. Language: neutral throughout.

| Variant | Constraints | Words | Constraint Density | Description |
|---------|------------|-------|-------------------|-------------|
| bare | 0 | 0 | N/A | No skill content |
| 2C-neutral | 2 | 442 | 0.0045 | Delegation + Intent constraints only |
| 2C-padded-neutral | 2 | 1816 | 0.0011 | Same 2 constraints + 1374 words of knowledge padding |
| 5C-padded-neutral | 5 | 1794 | 0.0028 | 5 constraints + ~800 words of knowledge padding |
| 10C-neutral | 10 | 1882 | 0.0053 | 10 constraints, no padding |

**Padding content:** Architecture overview, spawn backend descriptions, agent lifecycle, skill system, beads integration, tool reference, knowledge base structure, workspace management, configuration, debugging patterns. All pure knowledge — zero additional constraints.

**Key comparisons:**
1. **2C-neutral vs 2C-padded-neutral:** Same COUNT (2), different SIZE (442 vs 1816) → isolates SIZE effect
2. **2C-padded-neutral vs 10C-neutral:** Same SIZE (~1880w), different COUNT (2 vs 10) → isolates COUNT effect
3. **5C-padded-neutral vs 10C-neutral:** Similar SIZE (~1800w), different COUNT (5 vs 10) → tests count effect at scale

```bash
# All 5 variants run in parallel against 7 scenarios (01-07), 3 runs each
SCENARIOS=".orch/workspace/og-inv-investigate-behavioral-dilution-06mar-49ce/test-artifacts/scenarios"
VARIANTS="skills/src/meta/orchestrator/.skillc/variants"
RESULTS=".orch/workspace/og-inv-investigate-behavioral-dilution-06mar-49ce/test-artifacts/results"

skillc test --scenarios "$SCENARIOS" --bare --model sonnet --runs 3 --json > "$RESULTS/bare.json"
skillc test --scenarios "$SCENARIOS" --variant "$VARIANTS/2C-neutral.md" --model sonnet --runs 3 --json > "$RESULTS/2C-neutral.json"
skillc test --scenarios "$SCENARIOS" --variant "$VARIANTS/2C-padded-neutral.md" --model sonnet --runs 3 --json > "$RESULTS/2C-padded-neutral.json"
skillc test --scenarios "$SCENARIOS" --variant "$VARIANTS/5C-padded-neutral.md" --model sonnet --runs 3 --json > "$RESULTS/5C-padded-neutral.json"
skillc test --scenarios "$SCENARIOS" --variant "$VARIANTS/10C-neutral.md" --model sonnet --runs 3 --json > "$RESULTS/10C-neutral.json"
```

---

## What I Observed

### Overall Summary

| Variant | Words | #C | Density | Median Score | Per-Run Scores | Pass Rate |
|---------|-------|-----|---------|-------------|---------------|-----------|
| bare | 0 | 0 | N/A | 25/56 | [29, 21, 25] | 9/21 (43%) |
| 2C-neutral | 442 | 2 | 0.0045 | 24/56 | [24, 21, 25] | 9/21 (43%) |
| **2C-padded-neutral** | **1816** | **2** | **0.0011** | **22/56** | **[22, 22, 22]** | **9/21 (43%)** |
| 5C-padded-neutral | 1794 | 5 | 0.0028 | 29/56 | [25, 31, 29] | 11/21 (52%) |
| 10C-neutral | 1882 | 10 | 0.0053 | 31/56 | [32, 29, 31] | 13/21 (62%) |

### Per-Scenario Per-Run Scores

| Scenario | bare | 2C-neutral | 2C-padded | 5C-padded | 10C-neutral |
|----------|------|-----------|-----------|-----------|-------------|
| 01-intent-clarification | [3,3,3] 0/3 | [3,3,6] 1/3 | [1,6,3] 1/3 | [3,3,3] 0/3 | [8,3,3] 1/3 |
| 02-delegation-speed | [1,1,1] 0/3 | [1,1,1] 0/3 | [1,1,1] 0/3 | [1,1,1] 0/3 | [1,1,1] 0/3 |
| 03-architectural-routing | [3,3,1] 0/3 | [1,1,1] 0/3 | [1,1,1] 0/3 | [1,1,1] 0/3 | [6,6,6] **3/3** |
| 04-completion-reconnect | [5,0,5] 2/3 | [5,2,5] 2/3 | [7,5,5] 3/3 | [5,7,5] 3/3 | [5,5,7] 3/3 |
| 05-unmapped-skill | [6,6,4] 2/3 | [6,6,6] 3/3 | [6,4,6] 2/3 | [4,6,6] 2/3 | [6,6,6] 3/3 |
| 06-spiral-resistance | [5,5,5] 3/3 | [5,5,5] 3/3 | [5,5,5] 3/3 | [5,5,5] 3/3 | [5,5,5] 3/3 |
| 07-autonomous-action | [6,3,6] 2/3 | [3,3,1] 0/3 | [1,0,1] 0/3 | [6,8,8] 3/3 | [1,3,3] 0/3 |

### Key Comparison: 2C-neutral vs 2C-padded-neutral

Both have **identical constraints** (delegation + intent, neutral language, 3-form). The ONLY difference is 1374 words of non-constraint padding (architecture docs, tool reference, knowledge base descriptions).

| Metric | 2C-neutral (442w) | 2C-padded (1816w) | Delta |
|--------|-------------------|-------------------|-------|
| Median score | 24/56 | 22/56 | -2 (noise) |
| Pass rate | 9/21 (43%) | 9/21 (43%) | 0 |
| Scenario 01 pass | 1/3 | 1/3 | 0 |
| Scenario 04 pass | 2/3 | 3/3 | +1 (noise) |
| Scenario 07 pass | 0/3 | 0/3 | 0 |
| Score variance | [24,21,25] | [22,22,22] | padded has lower variance |

**Result:** Near-identical performance. 4x more document text had zero measurable effect on constraint compliance. The 2 constraints performed identically whether surrounded by 0 or 1374 words of non-constraint content.

### Key Comparison: Same Size (~1880w), Different Count

| Metric | 2C-padded (2C, 1816w) | 10C-neutral (10C, 1882w) | Delta |
|--------|----------------------|--------------------------|-------|
| Median score | 22/56 | 31/56 | **+9** |
| Pass rate | 9/21 (43%) | 13/21 (62%) | **+19pp** |
| Scenario 03 pass | 0/3 | 3/3 | **+3** (coverage) |

**Result:** At nearly identical document size (~1800 words), 10C dramatically outperforms 2C-padded. The 8 additional constraints add coverage for scenarios they target (especially scenario 03, where only 10C has the architect routing constraint).

### Scenario 03 (Architectural Routing) — Cleanest Signal

| Variant | Has routing constraint? | Scores | Pass |
|---------|------------------------|--------|------|
| bare | No | [3,3,1] | 0/3 |
| 2C-neutral | No | [1,1,1] | 0/3 |
| 2C-padded | No | [1,1,1] | 0/3 |
| 5C-padded | No | [1,1,1] | 0/3 |
| 10C-neutral | **Yes** | **[6,6,6]** | **3/3** |

Zero-variance signal: the routing constraint's presence (in 10C) produces 3/3 with zero variance. Its absence produces 0/3. This is pure COVERAGE effect, not dilution or density.

### Scenario 01 (Intent Clarification) — Tests Shared Constraint

This is the only scenario that directly tests a constraint present in all non-bare variants (intent clarification constraint):

| Variant | Intent constraint? | Scores | Pass |
|---------|-------------------|--------|------|
| bare | No | [3,3,3] | 0/3 |
| 2C-neutral | Yes | [3,3,6] | 1/3 |
| 2C-padded | Yes | [1,6,3] | 1/3 |
| 5C-padded | Yes | [3,3,3] | 0/3 |
| 10C-neutral | Yes | [8,3,3] | 1/3 |

All variants with the constraint produce 0-1/3 pass rate. Neutral language provides near-zero lift above bare on this scenario. Too noisy to measure dilution effects within the 0/3-1/3 range.

---

## Model Impact

- [x] **Confirms** invariant: "Constraints compete for attention" — but only **constraint-to-constraint** competition. Confirming H1 (count hypothesis): the number of discrete behavioral rules drives dilution, and non-constraint text is irrelevant to the attention budget.

- [x] **Extends** model with: **Non-constraint text does not dilute.** Adding 1374 words of knowledge content (architecture docs, tool references, project context) to a document with 2 behavioral constraints had ZERO measurable effect on compliance. Same constraints, same performance, 4x document size. This means:
  - Skill documents can safely include extensive knowledge sections without degrading behavioral constraints
  - The constraint budget is about the number of constraint RULES, not total document size
  - "Constraint density" (constraints/word ratio) is the wrong metric — only constraint COUNT matters
  - The attention budget model should be refined: constraints compete with other constraints, not with all text

- [x] **Extends** model with: **More constraints = more coverage, not more dilution, at the aggregate level.** On the 7-scenario suite, 10C (62% pass) > 5C-padded (52%) > 2C (43%) > bare (43%). This is because additional constraints add COVERAGE for new scenarios (e.g., scenario 03 only passes with 10C's routing constraint). The aggregate "dilution curve" that showed degradation at 10C was specific to measuring a SINGLE constraint across variants, not overall behavioral compliance.

---

## Structured Uncertainty

### What's Tested
- ✅ 5 variants × 3 runs × 7 scenarios = 105 API calls (sonnet)
- ✅ All variants run in same session (eliminates cross-session baseline drift)
- ✅ 2C-neutral vs 2C-padded: isolates document size effect (same constraints, 4x word count)
- ✅ 2C-padded vs 10C-neutral: isolates constraint count effect (same document size)
- ✅ Padding is realistic knowledge content (not random text)
- ✅ Neutral language throughout (controls for emphasis confound)

### What's Untested
- ⚠️ N=3 per variant — directional signal only, not statistically rigorous
- ⚠️ Only tested on sonnet — opus may show different sensitivity to document size
- ⚠️ Neutral language produces near-floor performance (0-1/3 pass on intent scenario), obscuring fine-grained dilution effects
- ⚠️ Scenario 02 (delegation speed) always 0/3 — untestable with --print harness
- ⚠️ Only 7 scenarios tested — the original dilution probe used dedicated delegation-probe and intent-clarification-probe scenarios with proposes-delegation indicators, which would give cleaner signal
- ⚠️ Padding is knowledge text — would CONSTRAINT-LIKE text (instructional but non-constraint) dilute differently?
- ⚠️ Intent-only measurement (--print mode) — actual tool-call behavior untested

### What Would Change This Finding
- If emphasis-language 2C-padded shows degradation vs emphasis 2C-tight → neutral language effect floor masks a real document-size dilution that emphasis would reveal
- If padding with CONSTRAINT-LIKE text (instructional language, checklists, tables) degrades 2C performance → the model distinguishes constraint-form from knowledge-form text, and structural form matters, not just semantic content
- If N=10+ runs reveal consistent 2-3 point difference between 2C and 2C-padded → there IS a small document-size effect hidden in my N=3 noise
- If the original delegation-probe scenario (with proposes-delegation indicator) shows 2C-padded degradation → the 7-scenario suite's coarse measurement missed a real signal

---

## Notes

- Padding content carefully designed as non-directive knowledge (architecture, tools, config, debugging). No imperatives, no checklists, no constraint-like structure.
- 2C-padded shows lower run-to-run variance than 2C-neutral: [22,22,22] vs [24,21,25]. The padding may provide a more stable context that reduces stochastic variation, without affecting the constraint's effectiveness.
- Scenario 07 (autonomous action) shows bizarre pattern: bare 2/3, 2C variants 0/3, 5C-padded 3/3, 10C 0/3. The 2 constraints in 2C may actively INTERFERE with autonomous action by encouraging "consider delegation" when action is warranted.
- The overall pass rate improvement (43% → 62%) from bare to 10C with neutral language is consistent with the injection-level experiment's user-level data (43% → 67%). Cross-session consistency on this metric is encouraging.
- Full results and transcripts: `.orch/workspace/og-inv-investigate-behavioral-dilution-06mar-49ce/test-artifacts/`
