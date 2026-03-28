---
beads_id: orch-go-0ro8w
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
    score: "9/9"
    detected: true
    evidence: ""
signal_count: 5
signal_total: 6
---

# Brief: orch-go-0ro8w

## Frame

Tested NI-01's spatial prediction: do tensions cluster tighter than resolutions in 103 briefs? **Disconfirmed** — resolutions cluster tighter (mostly a text-length artifact; after controlling for length, no difference). But tensions ARE more specific (lower entropy, higher concentration, p<0.001 on all measures). The paradox: specificity ≠ proximity. Named gaps are specific coordinates but each points at a DIFFERENT coordinate. The model's "self-organization" mechanism is likely referential (gaps link to each other) not distributional (gaps look alike).

## Resolution

### New Artifacts
- `.kb/models/named-incompleteness/probes/2026-03-28-probe-tension-clustering-spatial-prediction.md` — First quantitative test of NI-01

### Decisions Made
- Used TF-IDF rather than API-dependent embeddings: reduces external dependency, establishes baseline (semantic embeddings would be a follow-up)
- Used Frame text for topic clustering rather than combined text: topic assignment should be independent of the measured variables (tensions and resolutions)
- Ran three independent length controls: truncation, windowing, binary features — ensures the length-controlled finding is robust to control method

### Constraints Discovered
- TF-IDF cosine similarity measures distributional proximity (shared vocabulary), not referential structure (pointing at the same thing). The NI model's "coordinate" metaphor may describe referential structure, which this method can't measure.
- Brief tensions are short (73 words median) — low signal for bag-of-words methods. Semantic embeddings might capture more of the meaning structure.

## Tension

- **Same-gap pair test**: Manually identify briefs that share a specific named gap. Do their tensions have higher similarity than random pairs? This directly tests the "convergence on shared coordinates" claim at the gap-pair level rather than the aggregate level.
- **Semantic embedding replication**: Does the finding hold with sentence-transformers? If tensions cluster tighter under semantic embeddings but not TF-IDF, the model's insight is about meaning-level structure.
- **Reference chain structure**: Can we measure whether tensions form longer reference chains (A→B→C) than resolutions? This would test the "compositional structure" hypothesis directly.
- **Cross-project replication**: Do scs-special-projects briefs (if they exist) show the same pattern?
