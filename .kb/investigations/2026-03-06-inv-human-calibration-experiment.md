# Investigation: Human Calibration Experiment for Comprehension Measurement

**Date:** 2026-03-06
**Status:** Complete (awaiting Dylan's ratings)
**Issue:** orch-go-54y23
**Phase:** Phase 2 of comprehension measurement program

---

## D.E.K.N.

**Delta:** Generated 24 blind-rated responses across 4 contrastive scenarios, 3 variants each, N=2 runs. Produced a randomized rating sheet (R01-R24) for Dylan to rate 1-5 without knowing which responses had skill context.

**Evidence:** 24 transcripts captured via `skillc test --transcripts`. Automated indicator scores range from 0/8 to 7/8. Score summary below.

**Knowledge:** The automated scores already show clear patterns: S09 (contradiction detection) shows strong stance effect (bare [4,1] → with-stance [7,7]). S13 (stale deprecation) shows knowledge lifting scores but stance providing no additional lift. S11 (absence) shows a surprising inversion (with-stance [3,3] ≤ bare [3,3]). S12 (consumer contract) shows knowledge ceiling but stance reducing consistency.

**Next:** Dylan rates the 24 responses blind. Compute Pearson correlation between human ratings and automated scores. If r > 0.6, automated proxies are validated. If r < 0.4, scorer extensions needed (Phase 3).

---

## Hypothesis

**Claim:** Human comprehension ratings will correlate with automated indicator scores at r > 0.6, validating keyword-based detection as a proxy for comprehension quality in contrastive scenarios.

**Variable:** Automated indicator score (0-8) vs human comprehension rating (1-5)
**Measurement:** Pearson correlation coefficient
**Falsification:** If r < 0.3, keyword detection does not measure what humans recognize as comprehension.
**Source:** Phase 2 of comprehension measurement program plan.

---

## Experimental Design

| Scenario | Variant 1 | Variant 2 | Variant 3 |
|----------|-----------|-----------|-----------|
| S09 contradiction-detection | bare | with-stance | with-stance-and-action |
| S11 absence-as-evidence | bare | without-stance | with-stance |
| S12 downstream-consumer-contract | bare | without-stance | with-stance |
| S13 stale-deprecation-claim | bare | without-stance | with-stance |

**Model:** sonnet (held constant across all trials)
**Runs per variant:** 2
**Total trials:** 4 scenarios × 3 variants × 2 runs = 24

---

## Results (Automated Scores Only)

### Per-Variant Scores

| Scenario | Variant | Run 1 | Run 2 | Median |
|----------|---------|-------|-------|--------|
| S09 | bare | 4 | 1 | 2.5 |
| S09 | with-stance | 7 | 7 | 7 |
| S09 | with-stance-and-action | 7 | 7 | 7 |
| S11 | bare | 3 | 3 | 3 |
| S11 | without-stance | 4 | 6 | 5 |
| S11 | with-stance | 3 | 3 | 3 |
| S12 | bare | 0 | 6 | 3 |
| S12 | without-stance | 7 | 7 | 7 |
| S12 | with-stance | 3 | 6 | 4.5 |
| S13 | bare | 1 | 1 | 1 |
| S13 | without-stance | 4 | 4 | 4 |
| S13 | with-stance | 4 | 4 | 4 |

### Per-Indicator Detection Rates (Key Findings)

**S09 — notices-tension (weight 3):**
- bare: 0/2, with-stance: 2/2, with-stance-and-action: 2/2
- Clear stance signal. Knowledge alone insufficient for implicit contradiction detection.

**S11 — identifies-mechanism (weight 3):**
- bare: 0/2, without-stance: 1/2, with-stance: 0/2
- Surprising: without-stance outperforms with-stance. Small N but counterintuitive.

**S12 — connects-volume-change (weight 3):**
- bare: ?/2 (mixed), without-stance: 2/2, with-stance: 1/2
- Knowledge about consumer counts sufficient; stance adds noise.

**S13 — notices-stale-claim (weight 3):**
- 0/2 across ALL variants including with-stance
- Stale claim detection not achieved by any variant at N=2. Core discriminating indicator is floor.

### Preliminary Observations

1. **S09 confirms prior finding:** Stance drives contradiction detection. With-stance-and-action shows no additional lift over with-stance (both [7,7]).
2. **S13 is floor-bound:** Neither knowledge nor stance enables stale claim detection. The `notices-stale-claim` indicator may need keyword expansion, or this scenario is genuinely hard.
3. **S11/S12 inversions:** with-stance sometimes scores lower than without-stance. Possible explanations: (a) stance text is too generic, (b) stance adds cognitive load without helping for these scenarios, (c) N=2 variance.
4. **High variance in bare:** S12 bare [0, 6] is extreme range, suggesting bare performance is unstable for explicit-consumer scenarios.

---

## Deliverables

1. `evidence/2026-03-06-human-calibration/transcripts/` — 24 raw responses organized by scenario+variant
2. `evidence/2026-03-06-human-calibration/blind-rating-sheet.md` — Randomized R01-R24 for Dylan to rate 1-5
3. `evidence/2026-03-06-human-calibration/answer-key.json` — Maps blind ID → scenario, variant, run, auto score
4. `evidence/2026-03-06-human-calibration/build-rating-sheet.py` — Reproducible script for generating rating sheet

---

## Structured Uncertainty

**What's tested:**
- Whether 4 contrastive scenarios produce scoreable responses via skillc test
- Whether automated indicator scores discriminate across variants at N=2
- Whether blind rating sheet creation is feasible for human calibration

**What's untested (awaiting Dylan's ratings):**
- Human-automated score correlation
- Which specific indicators correlate with human judgment
- Whether high auto-scores (7/8) correspond to high human ratings (4-5)
- Whether low auto-scores (0-1/8) correspond to low human ratings (1-2)

**What would change this finding:**
- If human ratings show flat distribution (all 3s), the scenarios don't discriminate for humans either
- If human ratings correlate with response length rather than auto-score, length is the dominant proxy
- If human-auto correlation is high for S09 but low for S13, the measurement is scenario-dependent, not universal

**Next experiment (after ratings):**
1. Compute correlation. If r > 0.6: automated proxies validated, proceed to scorer extensions.
2. If r < 0.4: keyword detection is noise. Consider structural analysis (cross-reference density, relational language).
3. If mixed: some indicators correlate, others don't. Per-indicator correlation analysis to identify which proxies to keep.
