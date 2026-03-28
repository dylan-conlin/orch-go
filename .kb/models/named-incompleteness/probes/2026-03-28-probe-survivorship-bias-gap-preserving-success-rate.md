# Probe: Survivorship Bias in Gap-Preserving Success Rate

**Model:** named-incompleteness
**Date:** 2026-03-28
**Status:** Complete
**claim:** NI-02, NI-05
**verdict:** qualifies (bias is real but bounded; does not invalidate the finding)

---

## Question

NI-05 reports 3/3 gap-preserving features survived, 4/4 resolution-oriented features died. NI-02 reports 17/17 correct predictions. Do these rates suffer from survivorship bias? Specifically: features that were never built (e.g., rejected code review gate) or died too quickly to observe can't appear in the measurement. Does this inflate the gap-preserving success rate or deflate the resolution-oriented failure rate?

---

## What I Tested

Enumerated the selection filters that determine which features enter the sample, and analyzed how each filter biases the observed success/failure rates.

### Selection Filters

A feature enters the NI-05/NI-02 sample only if it:
1. **Was built** — proposed features that were rejected at review can't be measured
2. **Existed long enough to observe** — features killed in hours can't accumulate outcome data
3. **Was salient enough to remember** — features that quietly degraded might not be recalled during retrospective classification

Each filter could differentially exclude features from one category, biasing the ratio.

---

## What I Observed

### Filter 1: "Was built" — affects both categories

**Resolution-oriented features never built:**
- Code review gate (mentioned in NI-05 probe as example) — rejected before implementation
- Unknown others — we can't enumerate what was never proposed or was rejected in conversation

**Gap-preserving features never built:**
- Also unknown — but the asymmetry is instructive. Resolution-oriented features are more likely to be rejected at proposal stage because they're enforcement mechanisms that face immediate pushback ("100% bypass rate" is predictable from the design). Gap-preserving features face less resistance at proposal because they're additive (adding an Unexplored Questions section doesn't block anyone).

**Direction of bias:** This filter likely removes more resolution-oriented proposals than gap-preserving ones. But the removed proposals are ones that *would have been predicted to fail* — a code review gate is resolution-oriented, and the model predicts it would fail. So the filter removes cases that *confirm* the model, not cases that disconfirm it. This makes the evidence *weaker* (smaller sample) but doesn't *inflate* the success rate.

**Magnitude:** Even if 2-3 resolution-oriented features were rejected before building, they'd move the tally from 4/4 died to 4/7 died (the 3 unbuilt are outcome-unknown, not successes). The pattern would still hold directionally.

### Filter 2: "Existed long enough to observe" — asymmetric but in the expected direction

The health score spawn gate lasted ~1 day before removal. It's in the sample as a failure. So the floor for "long enough" is very low — approximately 1 day.

**Could gap-preserving features have died in <1 day?** Theoretically yes, but gap-preserving features are less likely to die quickly because they don't trigger immediate friction. An enforcement gate can be killed in hours because it blocks the first agent that encounters it. A gap-naming section on a template just... sits there. The failure mode for gap-preserving features is slow degradation (gap inflation, false gaps), not rapid rejection.

**Direction of bias:** This filter preferentially removes gap-preserving features that failed fast — but the failure mode of gap-preserving features is slow, so this filter probably removes nothing. Resolution-oriented features that failed fast ARE captured (health score gate, ~1 day).

**Magnitude:** Near zero. No evidence of gap-preserving features dying in <1 day in orch-go's history.

### Filter 3: "Salient enough to remember" — the real bias risk

This is the most dangerous filter. The NI-02 probe was a retrospective classification. The 17 features were recalled from memory and project artifacts. Features that were quietly abandoned — gap-preserving OR resolution-oriented — might not be recalled.

**Gap-preserving features that might be missing:**
- Early knowledge artifacts that were tried and abandoned before the model era
- Template sections that were added, never adopted, and quietly removed
- Signal types that were proposed in SYNTHESIS.md or briefs but never caught on

**Resolution-oriented features that might be missing:**
- Informal enforcement rules that were stated once and never followed up on
- Dashboard thresholds that were set and never checked

**Direction of bias:** Unclear. Both categories could have forgotten members. But resolution-oriented features leave louder traces (decisions to remove gates, bypass rate measurements) than gap-preserving features that quietly failed (a template section nobody filled out just... stays empty). The quiet failure mode of gap-preserving features is exactly the kind of thing that doesn't get recalled in retrospective enumeration.

**This is the genuine bias risk:** Gap-preserving features that failed by being ignored are invisible in exactly the way survivorship bias describes. You don't see the dead gap-preserving features because they died quietly.

### Assessing Magnitude

**How many quiet gap-preserving failures could exist?**

Looking at the model's own Failure Mode 3 (Gap Inflation) and Failure Mode 4 (False Gaps):
- Template-mandated gaps with 15-25% adoption → these are gap-preserving features that partially failed. Some ARE captured (beads enrichment labels at 18%). But the 15-25% figure suggests multiple surfaces with mandated-but-ignored gap-naming.
- The NI-02 probe counts 3 "mixed" features with partial NI preservation. These are borderline cases where gap-preservation partially failed.

**Estimate:** 1-3 additional gap-preserving features that failed quietly might be missing from the sample. If we add them:
- Current: 7/7 gap-preserving succeeded (NI-02) or 3/3 (NI-05)
- Adjusted: 7/10 or 3/6 gap-preserving succeeded

Even at the pessimistic bound, gap-preserving features still succeed at a higher rate than resolution-oriented features (0/4 or 0/5). The gap narrows but doesn't close.

---

## The Deeper Issue: Binary Classification Hides a Gradient

The survivorship bias concern matters most for the binary claim ("gap-preserving features succeed, resolution-oriented features fail"). But the NI-02 probe already found something more robust: a **gradient**. Features with partial NI preservation show partial success:

- Threads: 45% linked → mixed outcome
- Beads: 18% enriched → mostly failing
- Decisions: 57% linked → mixed outcome
- SYNTHESIS v2: 87% adoption of gap-naming → mostly succeeding

This gradient is resistant to survivorship bias because:
1. It doesn't depend on the tails (all-success, all-failure) which are most vulnerable to bias
2. Partial failures are observable — you can measure 18% enrichment without knowing about features that were never built
3. The gradient predicts within-feature variation, not just between-feature

**The gradient finding survives even if the binary tally is biased.**

---

## Model Impact

- [x] **Qualifies** NI-02 and NI-05: Survivorship bias is real and acts through a specific mechanism — gap-preserving features that fail quietly (Failure Modes 3 and 4) are less likely to be recalled in retrospective enumeration than resolution-oriented features that fail loudly (gates with measured bypass rates, features with explicit removal decisions).

- [x] **Bounds the bias:** Even at the pessimistic estimate (1-3 missing gap-preserving failures), the directional finding holds. The bias narrows the gap between categories but does not close it.

- [x] **Confirms the gradient is the stronger evidence:** The gradient finding (degree of NI preservation predicts degree of success) is robust to survivorship bias because it measures within-feature variation, not between-feature binary outcomes. Future claims should emphasize the gradient over the binary tally.

- [x] **Identifies a prospective test that eliminates the bias:** Design a new gap-preserving feature for a currently-failing surface (e.g., add remaining-questions section to session debriefs). Measure before/after. Prospective interventions can't suffer from retrospective selection bias.

---

## Notes

**Strength:** This probe identifies the specific mechanism by which survivorship bias enters (quiet failure of gap-preserving features vs loud failure of resolution-oriented features) rather than just noting the bias exists.

**Limitation:** The estimate of 1-3 missing features is itself uncertain. A systematic audit of all template changes, removed sections, and abandoned artifact types would provide a firmer count.

**Connection to measurement-honesty model:** The retrospective classification method in NI-02 has properties of the "absent-signal trap" — gap-preserving features that failed quietly produce no signal of their failure, which gets interpreted as "no failures" rather than "failure signal is missing."
