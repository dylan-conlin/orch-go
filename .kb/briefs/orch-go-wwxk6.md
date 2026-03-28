---
beads_id: orch-go-wwxk6
category: knowledge
quality_signals:
  structural_completeness:
    score: "4/4"
    detected: true
    evidence: "TLDR, Delta, Evidence, Knowledge"
  evidence_specificity:
    score: "true"
    detected: true
    evidence: ".md"
  model_connection:
    score: "false"
    detected: false
    evidence: ""
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
signal_count: 4
signal_total: 6
---

# Brief: orch-go-wwxk6

## Frame

Probed NI-06 (optimal specificity of named incompleteness). The inverted-U shape appears consistently in every metric — medium specificity tensions outperform both low and high — but the effect is too weak for TF-IDF to confirm statistically (best p=0.117). The high-specificity decline is the clearest signal; the low-specificity side is flat. Verdict: directionally supported, not confirmed.

## Resolution

### Constraints Discovered
- TF-IDF lexical similarity has a word-count confound (r=0.50) that may dominate specificity effects
- The orch-go brief corpus has no template-mandated tensions — all are organic, varying only in scope/specificity
- The NI-06 effect shape is asymmetric: the high-specificity decline is robust, but the low-specificity ascent is flat

## Tension

- **Sentence-transformer replication**: Would semantic embeddings (not lexical TF-IDF) make the inverted-U significant? The NI-01 bibliometrics study saw TF-IDF miss what sentence-transformers caught.
- **Actual convergence measurement**: Can we trace which tensions generated follow-up work (subsequent briefs/probes/threads on the same concern)? This would measure real convergence rather than distributional similarity.
- **Cross-corpus template test**: Find a system with template-mandated open-question sections (compliance retroes, mandatory reflection prompts) and test whether those low-specificity gaps truly fail to converge.
