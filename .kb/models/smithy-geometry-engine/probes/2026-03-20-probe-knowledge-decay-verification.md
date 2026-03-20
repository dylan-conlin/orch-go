# Probe: Knowledge Decay Verification — Smithy Geometry Engine

**Model:** smithy-geometry-engine
**Date:** 2026-03-20
**Status:** Complete
**Triggered by:** daemon knowledge-decay detector (999d sentinel = no prior probes)

---

## Question

The smithy-geometry-engine model has 8 claims (SG-01 through SG-08) based on a single evaluation session (4 test parts through SCS AI Part Builder, 2026-03-20). This probe verifies claim currency and identifies any claims that require additional evidence or have staleness risk.

---

## Verification Method

Checked all claims against their source evidence, cross-referenced with the harness-engineering probe and fabrication thread created the same day, and assessed the evidence base for single-source risk.

---

## Claim-by-Claim Verification

### SG-01: Parametric 3D from text (confidence: 0.9)
**Verdict: CONFIRMED — current, well-evidenced.**
Source investigation documents 4 test parts with named parameters (base width, flange height, Pem hole), STEP export observed. Three.js rendering with interactive rotation confirmed. This is the strongest claim — observed across all 4 tests.

### SG-02: One-way handoff, no DFM feedback (confidence: 0.85)
**Verdict: CONFIRMED — current, well-evidenced.**
Iframe embedding architecture directly observed. One-way gate ("Model cannot be edited after continuing") documented in Test 1 full flow walkthrough. No DFM API calls observed during generation. Cross-confirmed by harness-engineering probe (2026-03-20-probe-scs-ai-part-builder-compositional-correctness.md).

### SG-03: 0% DFM recall on hardware+bend (confidence: 0.8)
**Verdict: CONFIRMED — current, but single-test evidence.**
Test 3 (PEM near bend line) is the sole evidence. The 0% figure is accurate for the tested scenario but based on N=1 DFM conflict test. Confidence 0.8 is appropriate given the sample size. A follow-up probe should test additional DFM conflict types (minimum flange length, bend radius vs material thickness, feature-to-edge clearance).

### SG-04: Refinement loop may not work (confidence: 0.5)
**Verdict: CONFIRMED — current, appropriately uncertain.**
Test 1 refinement ("make holes larger, 8mm") showed no parameter panel update. Confidence 0.5 correctly reflects single-observation uncertainty. The `falsifies_if` condition (>80% success rate in systematic testing) is well-defined. This claim has the highest staleness risk — Smithy is in public beta and refinement bugs may be fixed rapidly.

### SG-05: Sheet metal vocabulary understanding (confidence: 0.85)
**Verdict: CONFIRMED — current, multi-test evidence.**
PEM nuts, flanges, bends, mounting holes all mapped correctly across Tests 1-4. Semantic parameter naming observed consistently. Well-evidenced claim.

### SG-06: Three options to close DFM gap, none exist (confidence: 0.7)
**Verdict: CONFIRMED — current, but highest external-change risk.**
Architecture analysis from iframe observation is sound. However, this is the claim most likely to become stale: Smithy is in active development (public beta since Jun 2025), and any of the three options could ship without notice. The `falsifies_if` condition is correct. Recommend re-checking quarterly or when SCS announces AI Part Builder updates.

### SG-07: Silent default filling (confidence: 0.75)
**Verdict: CONFIRMED — current, single-test evidence.**
Test 4 ("electronics enclosure") generated defaults without clarification. N=1 for ambiguous prompts specifically. The claim wording ("silently fills... rather than asking") is accurate for observed behavior.

### SG-08: No exposed bend parameters (confidence: 0.8)
**Verdict: CONFIRMED — current, multi-test evidence.**
Tests 2-4 all generated bent parts without exposing bend radius, K-factor, or bend deduction in the parameter panel. Consistent across 3 tests.

---

## Overall Assessment

**Model verdict: CURRENT. All 8 claims confirmed against source evidence.**

The model is brand new (created 2026-03-20) — no actual knowledge decay exists. The 999d trigger was a sentinel value indicating no prior probes, not genuine staleness.

### Evidence Base Risks

1. **Single-source dependency.** All 8 claims derive from one evaluation session (4 test parts, one user, one date). No independent replication. The model correctly states "WORKING HYPOTHESIS" validation status.

2. **Rapid external change.** Smithy is in public beta. Claims SG-04 (refinement), SG-06 (no DFM integration), and SG-08 (no bend params) could be invalidated by product updates at any time. These are the highest-priority claims for future re-validation.

3. **No API-level testing.** The model acknowledges this: "No API-level testing performed." Open questions 1 (API beyond iframe), 2 (geometry kernel), and 4 (parametric representation) remain untested.

### Recommendations

- **No model update needed** — model is accurate and current
- **Next probe priority:** Test additional DFM conflict types (strengthen SG-03 beyond N=1) or investigate Smithy API directly (address open questions 1-2)
- **Re-validation schedule:** Quarterly, or triggered by SCS/Smithy product announcements

---

## Notes

- Cross-references: harness-engineering probe (2026-03-20-probe-scs-ai-part-builder-compositional-correctness.md) confirms SG-02 and SG-03 independently
- Thread: `.kb/threads/2026-03-20-ai-assisted-physical-fabrication-smithy.md` links Smithy gap to LED letters project (same compositional correctness pattern)
- The investigation source (`.kb/investigations/2026-03-20-eval-sendcutsend-ai-part-builder.md`) is well-structured with clear per-test observations
