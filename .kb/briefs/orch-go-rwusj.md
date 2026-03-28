---
beads_id: orch-go-rwusj
category: knowledge
quality_signals:
  structural_completeness:
    score: "4/4"
    detected: true
    evidence: "TLDR, Delta, Evidence, Knowledge"
  evidence_specificity:
    score: "true"
    detected: true
    evidence: "fail"
  model_connection:
    score: "true"
    detected: true
    evidence: ".kb/models/"
  connective_reasoning:
    score: "true"
    detected: true
    evidence: ""
  tension_quality:
    score: "true"
    detected: true
    evidence: ""
  insight_vs_report:
    score: "6/6"
    detected: true
    evidence: ""
signal_count: 6
signal_total: 6
---

# Brief: orch-go-rwusj

## Frame

Classified 17 orch-go features by whether they preserve or close named incompleteness. The NI classification correctly predicted outcomes (success/failure/mixed) for all 17, matching measured adoption rates, compose rates, and orphan rates. Verdict: confirms NI-02 with circularity qualification — most features are in-sample, but the gradient prediction (degree of NI preservation predicts degree of success) and the SYNTHESIS.md intervention trajectory provide genuine out-of-sample evidence.

## Resolution

### New Artifacts
- `.kb/models/named-incompleteness/probes/2026-03-28-probe-feature-level-ni-preservation-classification.md` — NI-02 probe

### Constraints Discovered
- Circularity constraint: NI-02 was built from observing these features, so retrospective classification has limited evidential value. The genuine out-of-sample evidence is: (a) SYNTHESIS.md intervention trajectory, (b) gradient prediction on mixed features, (c) pre-spawn kb context, (d) two very new features.

### Key Finding: The Gradient
The most precise and hardest-to-dismiss evidence is the gradient: mixed features show partial success proportional to their NI preservation rate. This is a stronger prediction than binary success/failure because it's quantitative and harder to produce by post-hoc rationalization.

## Tension

**Questions that emerged during this session that weren't directly in scope:**

- Can NI-02 be tested prospectively? Specifically: design a new feature to add NI to a currently-piling-up surface (e.g., add "remaining questions" to session debriefs) and measure before/after. This would be the first genuinely predictive test of NI-02.
- Does the gradient hold quantitatively? Is adoption rate of gap-naming signals a continuous predictor of composition quality, or are there thresholds (the model's Constraint 1 about specificity suggests there might be a minimum specificity threshold)?
- False gaps (Failure Mode 4) are not measured in this probe. How prevalent are placeholder tensions in briefs? The 100% adoption rate for Tension sections could include compliance noise that looks like NI but isn't.

**What remains unclear:**
- Whether the 7 in-sample success features would have been correctly predicted BEFORE the model was built, or whether the model was shaped to fit them
- Whether NI-02 generalizes to other systems (this probe is one system only)
