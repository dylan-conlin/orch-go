# Probe: Resolution-Oriented Features Die; Gap-Preserving Features Grow — 30-49 Day Comparison

**Model:** named-incompleteness
**Date:** 2026-03-28
**Status:** Complete
**claim:** NI-05
**verdict:** confirms

---

## Question

NI-05 claims: "Resolution is a side effect, not the goal. Systems designed to 'resolve gaps' lose their generative engine. The productive state is 'partially resolved with named remaining questions.'"

The verification method specifies: compare longevity/productivity of features that preserve open questions (probes that extend models) vs features that aim for complete resolution (advisory gates that enforce a single correct state). Measure over 30+ days.

---

## What I Tested

Identified 7 features in orch-go's history, classified by whether they aim to *resolve gaps* (enforce a correct state) or *preserve open questions* (maintain named incompleteness). Measured each feature's longevity (days active), current status (alive/dead/reformed), and productivity (behavioral change achieved).

**Resolution-oriented features** (designed to close gaps):
1. Blocking accretion gates (Feb 13 - Mar 17): enforce "files must be under N lines"
2. Self-review completion gate (Mar 6 - Mar 13): enforce "no debug statements in code"
3. Health score spawn gate (Mar 10 - Mar 11): enforce "codebase health > 65"
4. SYNTHESIS.md v1 (Jan - Feb 2026): produce a conclusion document per session

**Gap-preserving features** (designed to preserve open questions):
5. Probe system (Feb 8 - present): test model claims, verdicts open new questions
6. Thread-to-model-to-claims lifecycle (Feb 8 - present): promote questions into testable claims
7. SYNTHESIS.md v2 with Unexplored Questions (Dec 21 reform, adopted Feb 26+): same artifact, reformed to preserve gaps

```bash
# Evidence sources (all in orch-go repo):
# Gates lifecycle:
#   .kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md
#   .kb/decisions/2026-03-13-remove-self-review-completion-gate.md
#   .kb/decisions/2026-03-11-remove-health-score-spawn-gate.md
#   .kb/models/harness-engineering/probes/2026-03-17-probe-pre-commit-accretion-gate-2-week-effectiveness.md
# Probes:
#   .kb/models/*/probes/ (328 files across 48 models)
#   git log --oneline -- '.kb/models/*/probes/*' (60+ commits)
# SYNTHESIS.md:
#   .orch/templates/SYNTHESIS.md (current template with Unexplored Questions)
#   Named-incompleteness model.md Failure Mode 1 (31 files, zero clustering)
```

---

## What I Observed

### Resolution-oriented features: consistent pattern of death or reform

| Feature | Days Active | Status | Productivity |
|---------|-------------|--------|-------------|
| Blocking accretion gates | 32 → converted to advisory | **Dead as blocking** | 100% bypass rate. Zero behavioral change from blocking itself. Hotspot reduction (75%) came from daemon extraction cascades triggered by gate *events*, not by gate *blocks*. |
| Self-review completion gate | 7 → removed | **Dead** | 79% false positive rate, 0 true positives. Removed because "79% FP rate normalizes override behavior, undermines gate system credibility." |
| Health score spawn gate | ~1 → removed | **Dead** | Calibration artifact. Formula tuned to pass existing codebase (score 69 under new formula with zero extractions). Gate never fired meaningfully. |
| SYNTHESIS.md v1 | ~60 → reformed | **Dead** (v1 form) | 31 undifferentiated conclusion files. Zero auto-clustering. Every file demanded individual triage. No compositional signal because no outward pointers. |

**Key finding on gates:** The Mar 17 decision documents the mechanism precisely: "Gates work through signaling, not blocking. The blocking path adds friction agents route around instantly, producing zero behavioral change. The event emission path triggers daemon responses that produce the actual structural improvement." The part of the gate that *resolved* (blocked) died. The part that *preserved the question* (emitted an event for daemon consideration) survived.

### Gap-preserving features: consistent pattern of growth

| Feature | Days Active | Status | Productivity |
|---------|-------------|--------|-------------|
| Probe system | 49+ days | **Growing** | 328 probes across 48 models (6.7/day avg). Multiple probes generated new models (coordination, probe-system). 83% of claims tested. Orphan rate dropped from 94.7% to 52.0%. Still accelerating (92 probes in peak week). |
| Thread→model→claims lifecycle | 49+ days | **Growing** | 51 models. Named-incompleteness itself was born from this lifecycle (3 threads converged → model promoted → 6 claims → 6+ probes). Each model promotion creates untested claims that attract new probes. |
| SYNTHESIS.md v2 (with Unexplored Questions) | 30+ days | **Active** | 87% adoption of Unexplored Questions section — the highest opt-in signal adoption rate in the system (normal max: 15-25%). Files are now distinguishable by their gaps. |

### The SYNTHESIS.md natural experiment

SYNTHESIS.md provides the cleanest comparison because the same artifact was tested in both modes:

- **v1 (resolution-oriented):** 31 files, each a pure conclusion. Zero clustering. Every file demanded individual attention. The system was "complete" after each session — and therefore inert.
- **v2 (gap-preserving):** Same format plus Unexplored Questions. 87% adoption. Files now carry outward pointers that enable clustering. Classification improved from "piling up" to "mixed" (intermediate; not yet fully composing).

The reform didn't change the artifact's purpose (session documentation). It changed whether the artifact closed or preserved gaps. The result: the gap-preserving version survived and became productive; the resolution-oriented version was explicitly identified as a failure instance in the named-incompleteness model.

### The advisory gate conversion as mechanism confirmation

The March 17 gate conversion is the strongest single data point:

- Same infrastructure (accretion checking, event emission, thresholds)
- Same codebase context (hotspot files, agent coordination)
- Only change: removed the "resolve" path (blocking), kept the "preserve question" path (signal emission)
- Result after conversion: hotspot files continued to shrink via daemon extraction cascades. The resolution mechanism was removed and *nothing changed* — because the productive mechanism was the question ("is this file too big?") being visible to the daemon, not the answer ("you may not commit") being enforced.

---

## Model Impact

- [x] **Confirms** invariant: NI-05 — Resolution is a side effect, not the goal.

Across 7 features measured over 30-49 days:
- **4/4 resolution-oriented features** either died (3) or required reform to survive (1)
- **3/3 gap-preserving features** survived and are still growing
- The advisory gate conversion demonstrates the mechanism in isolation: removing the resolution path from an otherwise identical infrastructure produced zero productivity loss
- SYNTHESIS.md provides a within-artifact comparison: same format, different orientation, opposite outcomes

The evidence pattern is: features die when they close gaps without opening new ones. Features thrive when they preserve the question. Resolution happens as a side effect of the question being visible (daemon extraction cascades, probe verdicts, thread convergence) — it is never the goal.

---

## Notes

**Strength:** Multiple independent feature comparisons (gates, SYNTHESIS, probes) all converge on the same pattern. The advisory gate conversion provides a quasi-experimental design where only the resolution mechanism was removed. The SYNTHESIS comparison controls for artifact type (same format, different orientation).

**Limitation:** The comparison is not perfectly controlled. Resolution-oriented features and gap-preserving features serve different system purposes (enforcement vs knowledge production). A stronger test would be: take two features serving the same purpose, make one resolution-oriented and one gap-preserving, and compare outcomes. The SYNTHESIS comparison approximates this but wasn't designed as an experiment.

**Limitation:** Survivorship bias is possible. We only observe features that existed long enough to measure. Resolution-oriented features that were never built (e.g., the rejected code review gate) can't be measured but would strengthen the pattern further.

**Untested extension:** NI-05 predicts that "the productive state is partially resolved with named remaining questions." The probe confirms the negative prediction (resolution-oriented features die) but doesn't directly measure whether *partial* resolution is more productive than *no* resolution. The probe system does resolve individual claims while opening new ones — this matches "partially resolved with named remaining questions" — but the optimal resolution fraction is unmeasured.
