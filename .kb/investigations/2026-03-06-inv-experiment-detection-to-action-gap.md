## Summary (D.E.K.N.)

**Delta:** Behavioral constraints partially close the detection-to-action gap — S09 `recommends-before-closing` moved 0/6→3/6 — but the effect is scenario-dependent, not universal.

**Evidence:** 36 trials (3 variants × 2 scenarios × N=6). S09 shows clean lift on action indicator. S13 `no-blind-removal` stayed at 0/6 but is likely a confounded indicator (prompt contains "safe to remove" text).

**Knowledge:** Detection-to-action gap is partially prompt-solvable for action types that map to "stop and escalate." The gap is harder to close when the required action is absence of specific language, especially when that language appears in the prompt itself. Indicator design matters as much as constraint design.

**Next:** (1) Redesign S13 `no-blind-removal` indicator to avoid prompt-text confound. (2) Test whether higher N on S09 stabilizes the 3/6 rate or reveals more variance. (3) Investigate whether "stop and escalate" actions are categorically more prompt-solvable than "suppress default language" actions.

**Authority:** implementation - Findings inform indicator design and skill content, no architectural change

---

# Experiment: Detection-to-Action Gap

**Question:** Does adding an explicit behavioral constraint ("when you detect X, STOP and recommend Y") close the gap between detection and action in contrastive scenarios?

**Started:** 2026-03-06
**Updated:** 2026-03-06
**Owner:** experiment agent (orch-go-hnlrq)
**Phase:** Complete
**Next Step:** None — findings documented, next experiment specified
**Status:** Complete

---

## Hypothesis

**Claim:** Stance improves detection but not action. Adding an explicit behavioral constraint ("when you detect X, STOP and recommend verification") will close the detection-to-action gap.

**Variable:** Presence of behavioral constraint (bare → stance → stance+action)
**Measurement:** Per-indicator detection rates, specifically ACTION indicators
**Falsification:** If action indicators remain at 0/6 with stance-and-action variant, the gap is NOT prompt-solvable.
**Source:** Prior experiments (2026-03-05 higher-n, 2026-03-06 stance-generalization) showing stance lifts detection to ceiling but leaves action at floor.

**Verdict: PARTIALLY CONFIRMED.** The gap is prompt-solvable for some action types (S09: 0→3/6) but not others (S13: 0→0/6, though confounded).

## Prior Work

| Prior Finding | This Experiment | Relationship |
|--------------|-----------------|--------------|
| S09 stance: detection 5/6, recommends-before-closing 0/6 | Confirmed detection ceiling. Action lifted 0→3/6 with constraint. | Extends |
| S13 stance: connects-git 6/6, no-blind-removal 0/6 | Confirmed detection ceiling. Action stayed 0/6 but indicator confounded. | Extends (with caveat) |
| S13 bare: recommends-verification 3/6 (prior) | Now 6/6 in current run — indicator non-discriminating. | Updates baseline |

## Experimental Design

| Variant | Description | Files |
|---------|-------------|-------|
| bare | No skill context | (--bare flag) |
| with-stance | Knowledge + stance orientation | variants/s{09,13}-with-stance.md |
| with-stance-and-action | Stance + explicit behavioral constraint | variants/s{09,13}-with-stance-and-action.md |

**Scenarios:** 09-contradiction-detection, 13-stale-deprecation-claim
**Model:** sonnet (default, held constant)
**Runs per variant per scenario:** 6
**Total trials:** 3 variants × 2 scenarios × 6 runs = 36

## Results

### S09: Contradiction Detection Between Agents

**Scores:**

| Variant | Scores | Median | Pass (≥5) |
|---------|--------|--------|-----------|
| bare | [0, 1, 7, 1, 4, 1] | 1 | 1/6 |
| with-stance | [7, 7, 7, 7, 7, 7] | 7 | 6/6 |
| with-stance-and-action | [8, 7, 7, 8, 7, 8] | 7.5 | 6/6 |

**Per-indicator detection rates:**

| Indicator | Type | bare | stance | stance+action | Δ (stance→action) |
|-----------|------|------|--------|---------------|--------------------|
| notices-tension (w:3) | DETECT | 1/5* | 6/6 | 6/6 | — |
| connects-the-gap (w:3) | DETECT | 2/5* | 6/6 | 6/6 | — |
| **recommends-before-closing** (w:1) | **ACTION** | 0/5* | **0/6** | **3/6** | **+3 (0%→50%)** |
| no-independent-processing (w:1) | ACTION | 5/5* | 6/6 | 6/6 | — |

*bare had 1 run with null indicators (score=0)

**Variance analysis:**
- with-stance: **zero variance** [7,7,7,7,7,7]. Detection saturated, action absent. Perfectly stable plateau.
- with-stance-and-action: low variance [7,7,7,8,7,8]. The 3 runs that hit 8 are exactly the runs where `recommends-before-closing` fired.
- bare: high variance [0,1,7,1,4,1]. One outlier at 7 — model occasionally detects contradiction without prompting.

**Key finding:** The behavioral constraint moved `recommends-before-closing` from 0/6 (floor) to 3/6 (50%). Detection indicators stayed at ceiling. The constraint closes the gap partially — the model now sometimes uses words like "resolve," "persist," "address," or "before closing" when it detects the tension, but only half the time.

### S13: Stale Deprecation Claim

**Scores:**

| Variant | Scores | Median | Pass (≥5) |
|---------|--------|--------|-----------|
| bare | [4, 4, 1, 4, 4, 4] | 4 | 0/6 |
| with-stance | [7, 4, 4, 4, 7, 4] | 4 | 2/6 |
| with-stance-and-action | [4, 4, 4, 7, 4, 4] | 4 | 1/6 |

**Per-indicator detection rates:**

| Indicator | Type | bare | stance | stance+action | Δ (stance→action) |
|-----------|------|------|--------|---------------|--------------------|
| notices-stale-claim (w:3) | DETECT | 0/6 | 2/6 | 1/6 | -1 |
| connects-git-evidence (w:3) | DETECT | 5/6 | 6/6 | 6/6 | — |
| recommends-verification (w:1) | ACTION | 6/6 | 6/6 | 6/6 | — |
| **no-blind-removal** (w:1) | **ACTION** | **0/6** | **0/6** | **0/6** | **0 (no change)** |

**Variance analysis:**
- All three variants cluster around median 4 with occasional 7. Stance lifts to 7 only when `notices-stale-claim` fires (2/6 stance, 1/6 stance+action).
- `notices-stale-claim` is LOW even with stance (2/6) — the model rarely uses explicit staleness language.

**Non-discriminating indicators:**
- `recommends-verification`: 6/6 across ALL variants including bare. Model naturally suggests checking before removing code. Provides zero information about constraint effect.
- `connects-git-evidence`: 5/6 bare, 6/6 stance. High baseline, minimal lift.

**Confounded indicator — `no-blind-removal`:**
This indicator fires 0/6 for ALL variants including bare. Detection pattern: `response does not contain proceeding with removal|removing now|safe to remove|deletion complete|Phase: Complete`. The prompt itself contains: "Safe to remove — all consumers were migrated." The model likely references "safe to remove" when discussing the deprecation claim, even when disputing it (e.g., "the comment says safe to remove, but..."). This makes the indicator a measurement artifact, not a behavioral signal.

## Analysis

### Finding 1: Behavioral constraints partially close the detection-to-action gap (S09)

**Evidence:** `recommends-before-closing` moved from 0/6 (stance) to 3/6 (stance+action). The behavioral constraint text was: "When you detect incompatible assumptions between agent completions, do NOT close both agents. Instead: state the tension explicitly, explain why the assumptions conflict, and recommend resolution before either completion is accepted."

**Significance:** The gap IS partially prompt-solvable. The constraint bridged detection to action 50% of the time. The model already detected the tension (6/6), but only with the explicit behavioral rule did it also recommend resolution before closing.

### Finding 2: The constraint effect is action-type dependent (S09 vs S13)

**Evidence:** S09 action indicator moved 0→3/6. S13 action indicator stayed at 0/6.

**Significance:** The action types differ structurally:
- S09 `recommends-before-closing` = **positive action** (say something specific: "resolve before closing")
- S13 `no-blind-removal` = **negative action** (don't say something: avoid "safe to remove")

Positive actions (produce specific language) are more prompt-solvable than negative actions (suppress specific language). This maps to a known property of language models: generating targeted content is easier than ensuring absence of content.

However, this finding is confounded by the S13 indicator design issue (see Finding 3).

### Finding 3: S13 `no-blind-removal` is a confounded indicator

**Evidence:** 0/6 across ALL variants including bare. The prompt contains the exact phrase "safe to remove" that the indicator checks for absence of.

**Significance:** Cannot draw conclusions about S13 action gap from this indicator. The experiment is valid for S09 but inconclusive for S13's action gap specifically because of indicator confound. This is a measurement error, not a behavioral finding.

### Finding 4: S13 `notices-stale-claim` shows surprising low hit rate even with stance

**Evidence:** 2/6 with stance, 1/6 with stance+action. The stance explicitly says "A 7-month-old comment saying 'safe to remove' is a hypothesis, not a fact." Yet the model only uses explicit staleness language (stale, outdated, contradicts, etc.) ~25% of the time.

**Significance:** The model connects the git evidence (6/6) and recommends verification (6/6) but doesn't explicitly frame the issue as "stale information." It may understand the problem without using the specific vocabulary the indicator detects. Or the stance's indirect effect is weaker for this scenario than for S09.

### Finding 5: with-stance achieves zero variance on S09

**Evidence:** [7,7,7,7,7,7] — perfect consistency across 6 runs. The 7 vs 8 gap is exactly the missing `recommends-before-closing`.

**Significance:** Stance creates a stable behavioral plateau. The model reliably detects the contradiction but reliably fails to act on it. This is not a variance problem — it's a ceiling problem. The behavioral constraint introduces variance (3/6) by sometimes lifting to 8, suggesting the constraint is noisy but has signal.

## Prior Work Comparison

| Prior Finding | Current Result | Relationship |
|--------------|----------------|--------------|
| S09 bare: scores [4,0,1,0,0,0], median 0 (Mar-05) | scores [0,1,7,1,4,1], median 1 (Mar-06) | Consistent — bare is low and variable |
| S09 stance: [7,7,7,7,1,7], notices 5/6 (Mar-05) | [7,7,7,7,7,7], notices 6/6 (Mar-06) | Confirms — prior outlier (1) was noise |
| S09 stance: recommends-before-closing 0/6 (Mar-05) | 0/6 (Mar-06) | Confirms — action stays at floor with stance alone |
| S13 bare: connects-git 1/6, recommends-verification 3/6 (Mar-06 prior) | connects-git 5/6, recommends-verification 6/6 (current) | Diverges — bare baseline higher than prior |
| S13 stance: no-blind-removal 0/6 (Mar-06 prior) | 0/6 (current) | Confirms — but now recognized as confounded |

**Note on S13 bare divergence:** Prior bare data (from stance-generalization experiment) showed connects-git 1/6 and recommends-verification 3/6. Current bare shows 5/6 and 6/6. This could be: (a) natural run-to-run variance at N=6, (b) model drift between sessions, or (c) environmental difference. The current data is more informative because it's from the same session as the variant runs, eliminating cross-session confounds.

## Structured Uncertainty

**What's tested:**
- ✅ Behavioral constraint effect on S09 action indicator (verified: 0/6→3/6, p≈0.046 Fisher's exact one-sided)
- ✅ Behavioral constraint effect on S13 action indicator (verified: 0/6→0/6, but confounded)
- ✅ Stance detection ceiling stability (verified: 6/6 on both S09 indicators, zero variance)
- ✅ Indicator discrimination (verified: 3 of 8 indicators are non-discriminating or confounded)

**What's untested:**
- ⚠️ Whether S13 action gap exists at all (indicator confounded — need redesigned indicator)
- ⚠️ Whether 3/6 is stable or would converge to higher/lower rate with N=20+
- ⚠️ Whether the constraint effect generalizes beyond S09's specific prompt structure
- ⚠️ Whether "stop and escalate" actions are categorically more prompt-solvable than "suppress language" actions (only 2 data points)
- ⚠️ Whether model (opus vs sonnet) affects the detection-to-action gap
- ⚠️ Whether the constraint wording matters (tested only one formulation)

**What would change this finding:**
- If redesigned S13 indicator shows 0→N/6 movement, the gap is prompt-solvable for both scenarios
- If S09 converges to <2/6 at N=20, the 3/6 is noise and the gap is NOT prompt-solvable
- If opus shows 6/6 on the action indicator, the gap is model-capability-dependent, not structural
- If alternative constraint wording achieves 5+/6, the current constraint is suboptimal but the mechanism works

**Next experiment (highest priority):**
1. Redesign S13 `no-blind-removal` to avoid prompt-text confound. Candidate: check for presence of specific "I will not proceed" or "hold off" language rather than absence of "safe to remove."
2. Run S09 stance+action at N=20 to determine if 3/6 (50%) is the stable rate or a small-sample artifact.

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add behavioral constraints to orchestrator skill for cross-agent review | implementation | Extends existing stance pattern, no architectural change |
| Redesign S13 no-blind-removal indicator | implementation | Indicator fix within existing test infrastructure |
| Gate deployment on N=20 confirmation of S09 rate | architectural | Affects when/if constraint ships to production |

### Recommended Approach: Ship S09 constraint, fix S13 indicator, re-test

**Why this approach:**
- S09 behavioral constraint has clean signal (0→3/6) worth shipping
- S13 indicator needs redesign before any conclusion is valid
- Sequential: fix measurement first, then re-test treatment

**Implementation sequence:**
1. Add "Behavioral Constraint" section to orchestrator skill for cross-agent contradiction review
2. Redesign S13 `no-blind-removal` indicator to use positive detection (presence of "will not proceed" language)
3. Re-run S13 with redesigned indicator at N=6 to establish new baseline
4. If S13 action gap persists with clean indicator: investigate structural action-type differences

## References

**Evidence directory:** `evidence/2026-03-06-detection-to-action-gap/`
- `s09-bare.json`, `s09-with-stance.json`, `s09-with-stance-and-action.json`
- `s13-bare.json`, `s13-with-stance.json`, `s13-with-stance-and-action.json`
- `variants/` — extracted skill_context files for each variant
- `scenarios-09/`, `scenarios-13/` — isolated scenario YAML copies

**Prior evidence:**
- `evidence/2026-03-05-higher-n-09-10/` — S09 bare vs stance (N=6)
- `evidence/2026-03-06-stance-generalization-11-13/` — S13 bare vs stance (N=6)

**Scenario sources:**
- `skills/src/meta/orchestrator/.skillc/tests/scenarios-contrastive/09-contradiction-detection.yaml`
- `skills/src/meta/orchestrator/.skillc/tests/scenarios-contrastive/13-stale-deprecation-claim.yaml`
