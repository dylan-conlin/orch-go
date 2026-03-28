# Probe: Retroduction Audit — Were NI-02's 7 Success Features Predicted or Fitted?

**Model:** named-incompleteness
**Date:** 2026-03-28
**Status:** Complete
**claim:** NI-02
**verdict:** qualifies

---

## Question

NI-02's feature-level classification probe (2026-03-28) found 7 features classified as "preserves NI → success" with 7/7 correct predictions. But 100% accuracy on in-sample features is suspicious. Were these 7 successes genuinely predicted by the NI model, or did the model inherit them from observations that were already explained by a simpler predecessor (compositional-accretion)?

---

## What I Tested

For each of the 7 success features, I traced:
1. **When was the feature first classified as a success?** (Before or after NI was formulated on 2026-03-27?)
2. **Did the compositional-accretion model already predict this outcome?** (CA's artifact audit was committed 2026-03-27 10:24 AM, hours before NI crystallized.)
3. **Does NI add any predictive power over CA for this feature?**

---

## What I Observed

### Timeline

- **2026-03-27 10:24 AM** — CA artifact audit committed. Classifies 13 artifact types as COMPOSING / PILING UP / MIXED using the "outward-pointing" criterion.
- **2026-03-27 ~afternoon** — NI model formulated from three converging threads. The thread text explicitly lists the same successes and failures as examples.
- **2026-03-28** — NI-02 probe classifies 17 features using the "preserves NI" criterion. Reports 7/7 success, 5/5 failure, etc.

### Feature-by-Feature Retroduction

| # | Feature | CA classification (pre-NI) | NI classification | Already predicted by CA? | NI adds predictive power? |
|---|---------|---------------------------|-------------------|--------------------------|---------------------------|
| 1 | Brief Tension section | COMPOSING — "100% tension coverage, no changes needed" | Preserves NI → Success | **YES** — same prediction, same outcome | No. CA's "outward-pointing" already predicts this. |
| 2 | Probe Model Impact section | COMPOSING — "84% structural signal, probes relate to models" | Preserves NI → Success | **YES** — same prediction, same outcome | No. CA's "natural composition signal" already predicts this. |
| 3 | Model Claims table | COMPOSING — "Claims table invites probes, no changes needed" | Preserves NI → Success | **YES** — same prediction, same outcome | No. CA's "outward-pointing atoms" already predicts this. |
| 4 | Comprehension queue (briefs → compose) | Not directly audited in CA, but briefs (the input) were classified COMPOSING | Preserves NI → Success | **PARTIAL** — CA predicts briefs compose; the pipeline wasn't separately assessed | Marginally. NI explains WHY the pipeline works (gaps accumulate until processed), but CA's "outward-pointing inputs compose" covers it. |
| 5 | Pre-spawn kb context | Not in CA audit | Preserves NI → Success | **NO** — genuinely novel feature not covered by CA | **Yes** — NI predicts this works because it injects prior named gaps. CA had no mechanism to predict context injection effectiveness. |
| 6 | orch orient (claim status surfacing) | Not in CA audit (built after model) | Preserves NI → Success | **NO** — built after both models | **Self-fulfilling** — designed specifically to preserve NI. The prediction can't fail because the feature was built to satisfy it. |
| 7 | orch research (claims parser, spawn from gaps) | Not in CA audit (built after model) | Preserves NI → Success | **NO** — built after both models | **Self-fulfilling** — same as #6. Feature was designed around NI claims. |

### Summary

| Category | Count | Features |
|----------|-------|----------|
| Already predicted by CA (no NI needed) | 3 | Briefs, Probes, Models |
| Partially predicted by CA | 1 | Comprehension queue |
| Genuinely novel NI prediction | 1 | Pre-spawn kb context |
| Self-fulfilling (designed around NI) | 2 | orch orient, orch research |

### What This Means

**3 of 7 "NI success predictions" were already predicted by the simpler CA model.** CA uses "outward-pointing" as its criterion; NI uses "preserves named incompleteness." For features #1-3, these criteria make identical predictions for the same reason — an outward-pointing artifact is one that preserves incompleteness. NI provides a deeper *explanation* (information-theoretic mechanism) but not a different *prediction*.

**1 of 7 is a genuinely novel prediction** (#5, pre-spawn kb context). This feature wasn't in CA's audit and its success isn't obvious from "outward-pointing" alone — it works because it injects *specific named gaps* into agent context, which is a distinctly NI-flavored mechanism.

**2 of 7 are self-fulfilling** (#6, #7). Features designed to preserve NI aren't evidence that NI predicts success — they're evidence that the builder *believed* NI predicts success. This is design validation, not empirical confirmation. The features could succeed for reasons unrelated to NI (e.g., surfacing any context at session start improves outcomes, regardless of whether it's "named gaps" specifically).

**1 of 7 is borderline** (#4). The comprehension queue's success is partially predicted by CA (briefs compose) but NI adds a mechanism (gaps accumulate and compound) that CA lacks.

---

## The Deeper Problem: Relabeling vs Predicting

NI's claim to predict success is weaker than it appears because:

1. **NI inherited CA's successes.** The three converging threads that produced NI on 2026-03-27 explicitly named the same features CA had already classified as COMPOSING that morning. The NI model's thread lineage says: "Every orch-go success (briefs, probes, threads, comprehension queue, attractor-gates) preserves named incompleteness." These were already *known* successes — NI provided a new label, not a new prediction.

2. **"Preserves NI" and "outward-pointing" are near-synonyms for orch-go features.** For surfaces in a knowledge system, "outward-pointing" (CA's criterion) and "preserves named incompleteness" (NI's criterion) make the same classification. The criteria might diverge in other domains (NI claims substrate-independence), but within the orch-go feature set used for NI-02, they are interchangeable.

3. **The causal arrow is backwards.** NI didn't predict these features would succeed, then check. NI *observed* that the successes shared a property (preserving incompleteness), then classified that property as the cause. This is abduction (inference to best explanation), not prediction. Abduction is legitimate theory-building, but it doesn't count as predictive confirmation.

---

## What Would Count as Genuine Prediction

The NI-02 probe already identified this: "Design a new orch-go feature specifically to preserve NI (e.g., add a 'remaining questions' section to a currently-piling-up surface like session debriefs). Measure outcome before and after."

More specifically, NI adds predictive power over CA in cases where:
- **NI and CA disagree.** Find a feature that is outward-pointing (CA predicts success) but closes named incompleteness (NI predicts failure), or vice versa. If NI is right where CA is wrong, that's evidence NI adds something.
- **The intervention trajectory is tested prospectively.** Take a failing feature, add NI-preservation *without* making it outward-pointing, and measure. If it improves, NI explains something CA doesn't.
- **Cross-domain prediction.** NI claims substrate-independence; CA is explicitly orch-go-specific. Testing NI outside orch-go (the 373-paper bibliometrics study did this) is where NI genuinely exceeds CA.

---

## Model Impact

- [x] **Qualifies** NI-02: The 7/7 success prediction accuracy is real but mostly inherited from the simpler CA model. Only 1 of 7 is a genuinely novel NI prediction (pre-spawn kb context). 2 are self-fulfilling (designed around NI). 3 were already predicted by CA.

- [x] **Does NOT contradict** NI-02: NI's prediction is correct — features that preserve NI do succeed. The qualification is about whether NI is doing the predictive work, or whether the simpler "outward-pointing" criterion already covers it.

- [x] **Strengthens** NI's cross-domain claims: The weakest evidence for NI within orch-go (where CA already explains the same outcomes) is offset by the strongest evidence outside orch-go (bibliometrics study, cross-domain spatial validation). NI's value-add over CA is primarily in cross-domain generalization, not in-sample orch-go feature prediction.

- [x] **Identifies the real NI-vs-CA discriminator:** NI and CA diverge when (a) a feature is outward-pointing but closes incompleteness, or (b) a feature preserves incompleteness but isn't outward-pointing. Finding such cases would distinguish the models. Within the current 17-feature set, no such divergence exists — the criteria are empirically identical.

**Evidence quality:** Analytical (comparison of two models' predictions on the same feature set; no new measurements, but identifies where the models' predictions overlap vs diverge).
