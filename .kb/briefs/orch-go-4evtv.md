---
beads_id: orch-go-4evtv
category: knowledge
quality_signals:
  structural_completeness:
    score: "4/4"
    detected: true
    evidence: "TLDR, Delta, Evidence, Knowledge"
  evidence_specificity:
    score: "true"
    detected: true
    evidence: "pass"
  model_connection:
    score: "true"
    detected: true
    evidence: ".kb/models/"
  connective_reasoning:
    score: "false"
    detected: false
    evidence: ""
  tension_quality:
    score: "true"
    detected: true
    evidence: ""
  insight_vs_report:
    score: "4/4"
    detected: true
    evidence: ""
signal_count: 5
signal_total: 6
---

# Brief: orch-go-4evtv

## Frame

Probe NI-05 confirmed: across 7 orch-go features measured over 30-49 days, all 4 resolution-oriented features died or required reform, while all 3 gap-preserving features survived and are still growing. The advisory gate conversion provides the cleanest evidence — removing the resolution path from otherwise identical infrastructure produced zero productivity loss.

## Resolution

### New Artifacts
- `.kb/models/named-incompleteness/probes/2026-03-28-probe-resolution-side-effect-longevity-comparison.md`

### Constraints Discovered
- NI-05's negative prediction (resolution-oriented features die) is strongly confirmed, but the positive prediction (partial resolution is optimal) remains unmeasured. We know that full resolution kills features and zero resolution is also unproductive (unnamed incompleteness doesn't compose) — but the optimal resolution fraction is unknown.

## Tension

**Questions that emerged during this session:**
- What is the optimal resolution fraction? NI-05 says "partially resolved" but doesn't specify how partial. The probe system resolves claims while opening new ones — what's the ratio of resolved-to-opened that maximizes productivity?
- Does the pattern hold for non-knowledge features? All evidence is from knowledge/coordination infrastructure. Would a purely operational feature (deploy pipeline, CI runner) show the same pattern, or does NI-05 only apply to "compositional substrates" per Constraint 2?
- The advisory gate conversion is a quasi-experiment. Could this be repeated deliberately as a controlled experiment? Take a currently blocking gate (build gate, vet gate) and test advisory-only mode.

**What remains unclear:**
- Whether survivorship bias inflates the gap-preserving success rate — features that were never built (rejected code review gate) can't be measured
