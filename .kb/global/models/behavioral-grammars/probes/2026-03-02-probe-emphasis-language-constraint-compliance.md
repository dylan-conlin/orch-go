# Probe: Does Emphasis Language Increase Behavioral Compliance at High Constraint Counts?

**Model:** behavioral-grammars
**Date:** 2026-03-02
**Status:** Complete

---

## Question

The dilution curve work (sonnet + opus) found behavioral constraints regress to bare parity at 10 competing constraints. That work used emphasis language (STOP, NEVER, ALWAYS, MUST, CRITICAL). Does the emphasis language itself contribute to compliance? Would neutral language (should, avoid, prefer, consider) produce worse results at the same constraint counts?

**Claim under test:** Emphasis markers (CRITICAL, MUST, NEVER, STOP, ALWAYS) are cosmetic — the model attends to semantic content, not stylistic urgency. If true, emphasis and neutral variants should produce identical compliance at the same constraint count.

**Falsifiable prediction:** If emphasis and neutral variants show identical proposes-delegation rates at 5C and 10C, emphasis language is cosmetic. If emphasis shows higher rates, emphasis language provides measurable compliance lift.

---

## What I Tested

**Design:** Head-to-head comparison of emphasis vs neutral language at 5-constraint and 10-constraint density, with bare as control.

| Variant | Constraints | Emphasis Language | Word Count |
|---------|------------|-------------------|------------|
| Bare | 0 | N/A | 0 |
| 5C-emphasis | 5 | STOP, NEVER, ALWAYS, MUST, CRITICAL | 972 |
| 5C-neutral | 5 | Consider, Try to avoid, Prefer, It's better | 1006 |
| 10C-emphasis | 10 | STOP, NEVER, ALWAYS, MUST, CRITICAL | 1815 |
| 10C-neutral | 10 | Consider, Try to avoid, Prefer, It's better | 1872 |

**Key design choices:**
- Same 3-form structural diversity (table + checklist + anti-patterns) in all variants
- Same semantic content — only the emphasis markers differ
- Same constraints (delegation + intent + 3/8 fillers) at each density level
- Word counts comparable within pairs (~3% difference from shorter emphasis words)
- Model: sonnet, 3 runs per variant, 2 scenarios (delegation-probe + intent-clarification-probe)

**Total test runs:** 30 (5 variants × 3 runs × 2 scenarios)

```bash
skillc test --scenarios scenarios/ --bare --model sonnet --runs 3 --json --transcripts transcripts/bare
skillc test --scenarios scenarios/ --variant variants/5C-emphasis.md --model sonnet --runs 3 --json --transcripts transcripts/5C-emphasis
skillc test --scenarios scenarios/ --variant variants/5C-neutral.md --model sonnet --runs 3 --json --transcripts transcripts/5C-neutral
skillc test --scenarios scenarios/ --variant variants/10C-emphasis.md --model sonnet --runs 3 --json --transcripts transcripts/10C-emphasis
skillc test --scenarios scenarios/ --variant variants/10C-neutral.md --model sonnet --runs 3 --json --transcripts transcripts/10C-neutral
```

---

## What I Observed

### Delegation Probe (behavioral constraint — the key signal)

| Variant | Scores | Median | proposes-delegation | no-code-reading | frames-delegation |
|---------|--------|--------|---------------------|-----------------|-------------------|
| Bare | [5, 5, 5] | 5/8 | **0/3** | 3/3 | 3/3 |
| 5C-emphasis | [8, 5, 8] | **8/8** | **2/3** | 3/3 | 3/3 |
| 5C-neutral | [8, 5, 5] | 5/8 | **1/3** | 3/3 | 3/3 |
| 10C-emphasis | [5, 8, 8] | **8/8** | **2/3** | 3/3 | 3/3 |
| 10C-neutral | [5, 5, 5] | 5/8 | **0/3** | 3/3 | 3/3 |

### Intent Probe (knowledge constraint)

| Variant | Scores | Median | asks-clarification | no-immediate-action | offers-interpretations |
|---------|--------|--------|-------------------|--------------------|-----------------------|
| Bare | [6, 6, 6] | 6/8 | **3/3** | 3/3 | 0/3 |
| 5C-emphasis | [8, 8, 3] | **8/8** | **2/3** | 3/3 | 2/3 |
| 5C-neutral | [3, 3, 8] | 3/8 | **1/3** | 3/3 | 1/3 |
| 10C-emphasis | [8, 3, 8] | **8/8** | **2/3** | 3/3 | 2/3 |
| 10C-neutral | [3, 6, 5] | 5/8 | **1/3** | 3/3 | 1/3 |

### Head-to-Head: Emphasis vs Neutral (Key Behavioral Indicators)

| Constraint Count | Indicator | Emphasis | Neutral | Delta |
|-----------------|-----------|----------|---------|-------|
| **5C** | proposes-delegation | 2/3 | 1/3 | +1 |
| **5C** | asks-clarification | 2/3 | 1/3 | +1 |
| **10C** | proposes-delegation | **2/3** | **0/3** | **+2** |
| **10C** | asks-clarification | **2/3** | **1/3** | **+1** |

---

## Findings

### Finding 1: Emphasis Language Provides Measurable Compliance Lift at 10 Constraints

**Evidence:** At 10C, emphasis proposes-delegation = 2/3 while neutral = 0/3. The 10C-neutral delegation scores [5,5,5] with zero variance — exactly matching bare [5,5,5]. The 10C-emphasis scores [5,8,8] with 2/3 proposes-delegation — meaningfully above bare.

This effect is consistent across both probes: emphasis outperforms neutral on the key behavioral indicator at both 5C and 10C.

**Significance:** The prior dilution study found 10C at bare parity using the emphasis variant. This session's 10C-emphasis shows 2/3 proposes-delegation, contradicting the prior 0/3. Combined across both sessions (6 total runs of emphasis 10C), emphasis 10C produces proposes-delegation 2/6. Meanwhile, neutral 10C produces 0/3. The emphasis language appears to create a non-zero probability of behavioral compliance that neutral language does not.

**Caveat:** N=3 per variant is noisy. The 10C-emphasis result (2/3) may be an upward fluctuation, as the prior session's identical file produced 0/3. What we CAN say: neutral 10C reliably produces bare parity, while emphasis 10C sometimes breaks through.

### Finding 2: Neutral Language at 10C is Indistinguishable from Bare

**Evidence:** 10C-neutral delegation: [5,5,5], proposes-delegation 0/3. Bare delegation: [5,5,5], proposes-delegation 0/3. Identical scores, identical indicator pattern, zero variance in both.

10C-neutral intent: median 5/8, asks-clarification 1/3. Bare intent: median 6/8, asks-clarification 3/3. Neutral is actually BELOW bare on intent.

**Significance:** Without emphasis markers, 10 constraints with 3-form diversity has zero measurable effect on behavioral compliance. The 1800+ words of neutral constraint language might as well not exist. This suggests the model treats "consider doing X" as advisory information it can freely ignore under attention competition.

### Finding 3: The Effect is Larger at Higher Constraint Counts

**Evidence:** At 5C, the emphasis advantage on proposes-delegation is +1 (2/3 vs 1/3). At 10C, it's +2 (2/3 vs 0/3). This directional trend suggests emphasis language's relative value increases as constraint density increases — precisely because neutral language degrades faster under competition.

**Significance:** If confirmed with more runs, this implies emphasis language is not just cosmetic — it functions as an attention allocation signal. When many constraints compete for the model's attention budget, emphasis markers (STOP, NEVER, MUST) may serve as salience cues that survive the dilution better than neutral framing.

### Finding 4: Bare Baseline Shows Higher Intent Than Prior Work

**Evidence:** This session's bare intent: [6,6,6], asks-clarification 3/3. Prior session's bare intent: [3,6,3], asks-clarification 1/3. Median jumped from 3/8 to 6/8.

**Significance:** Run-to-run variance in bare baselines makes absolute cross-session comparisons unreliable. Within-session comparisons (emphasis vs neutral) are more reliable since they use the same model version and sampling conditions. This reinforces why the head-to-head design (emphasis vs neutral in the same session) is the right approach.

---

## Model Impact

- [x] **Extends** model with: Emphasis language (CRITICAL/MUST/NEVER) provides measurable compliance lift over neutral language (should/consider/prefer) at high constraint counts. The effect appears larger at higher density (10C > 5C). Neutral 10C = bare parity; emphasis 10C sometimes breaks through.

- [x] **Qualifies** prior finding: The dilution curve's "bare parity at 10C" finding was measured using emphasis language. Neutral language reaches bare parity even earlier. The dilution threshold depends on BOTH constraint count AND emphasis framing.

- [x] **Contradicts** implicit assumption: The prior work's structured uncertainty listed "constraint ordering may matter" as untested. This probe reveals that emphasis framing (a different dimension than ordering) also matters — the constraint expression style is a second independent variable that the dilution curve work held constant without examining.

---

## Structured Uncertainty

### What's Tested
- ✅ Head-to-head emphasis vs neutral at 5C and 10C (30 test runs)
- ✅ Both behavioral (delegation) and knowledge (intent) constraints measured
- ✅ Same 3-form structural diversity in all variants
- ✅ Same semantic content — only emphasis markers differ
- ✅ Word counts comparable within pairs (≤5% difference)
- ✅ Bare control measured in same session for within-session comparison

### What's Untested
- ⚠️ N=3 per variant — high variance, directional signal only
- ⚠️ Only tested on sonnet — opus may show different sensitivity to emphasis
- ⚠️ Emphasis and neutral differ in word count by ~3-5% — not perfectly controlled
- ⚠️ Single-turn `--print` mode — interactive sessions may respond differently to emphasis
- ⚠️ Only 2 emphasis levels tested (full emphasis vs full neutral) — no gradient
- ⚠️ Prior 10C-emphasis session produced 0/3 on same file → high cross-session variance
- ⚠️ Bare baseline shifted between sessions (intent 3/8→6/8) → model or sampling drift

### What Would Change This Finding
- If N=10 runs show emphasis and neutral converging → emphasis effect is noise from small samples
- If opus shows no emphasis effect → sonnet-specific behavior (different attention allocation)
- If gradient testing (e.g., emphasis on delegation only, neutral elsewhere) shows same result as full emphasis → emphasis only needs to target the measured constraint, not all constraints
- If 10C-emphasis consistently produces 0/3 on replication → the 2/3 finding here was an upward fluctuation

### Combined Evidence: 10C Emphasis Across Sessions
| Session | 10C-emphasis proposes-delegation | 10C-emphasis scores |
|---------|--------------------------------|---------------------|
| Dilution study (Mar 1) | 0/3 | [5,5,5] |
| This experiment (Mar 2) | 2/3 | [5,8,8] |
| **Combined** | **2/6** | — |

The combined 2/6 rate (33%) is above bare (0/6 combined) but below the 1C ceiling (6/6). This suggests emphasis at 10C provides partial, unreliable compliance — better than nothing, far from ceiling.

---

## Positioning Against Prior Work

**Dilution curve (sonnet, Mar 1):** Found behavioral constraints at bare parity by 10C. Used emphasis language throughout. This experiment reveals the 10C result was EMPHASIS 10C — neutral 10C would have been at bare parity even more reliably.

**Opus replication (Mar 2):** Found dilution curve model-independent (opus matches sonnet). Those results also used emphasis language. The emphasis vs neutral dimension is orthogonal to the model dimension — both independently affect compliance.

**Implication for the layered enforcement recommendation:** The dilution study recommended moving behavioral constraints to infrastructure enforcement. This finding SUPPORTS that recommendation even more strongly: if you remove emphasis language (a common style choice), behavioral constraints fail even earlier. But it also suggests a partial mitigation: for the 2-5 constraint sweet spot, emphasis language provides meaningful lift over neutral language.

---

## Notes

- The 5C-emphasis variant is identical to the original dilution study's 5C — allowing rough cross-session comparison
- Test artifacts at: `.orch/workspace/og-feat-emphasis-experiment-02mar/test-artifacts/`
- Full investigation: this probe file
- Prior work: `.kb/investigations/2026-03-01-inv-test-constraint-dilution-threshold.md` (sonnet dilution curve)
- Prior work: `.kb/investigations/2026-03-02-inv-opus-dilution-curve-replication.md` (opus replication)
