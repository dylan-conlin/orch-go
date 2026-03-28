# Session Synthesis

**Agent:** og-work-probe-ni-02-28mar-a2ae
**Issue:** orch-go-rwusj
**Duration:** 2026-03-28
**Outcome:** success

---

## TLDR

Classified 17 orch-go features by whether they preserve or close named incompleteness. The NI classification correctly predicted outcomes (success/failure/mixed) for all 17, matching measured adoption rates, compose rates, and orphan rates. Verdict: confirms NI-02 with circularity qualification — most features are in-sample, but the gradient prediction (degree of NI preservation predicts degree of success) and the SYNTHESIS.md intervention trajectory provide genuine out-of-sample evidence.

---

## Delta (What Changed)

### Files Created
- `.kb/models/named-incompleteness/probes/2026-03-28-probe-feature-level-ni-preservation-classification.md` — Full probe document with 17-feature classification table, failure mode mapping, circularity assessment, and gradient finding

### Files Modified
- None

### Commits
- Pending

---

## Evidence (What Was Observed)

- 7 features that preserve NI all show success outcomes (100% adoption, low orphan rates, compose effectively)
- 5 features that close NI all show failure outcomes (100% bypass rate, 81.9% orphan rate, pile-up)
- 3 features with mixed NI show mixed outcomes (18-57% adoption), and the degree of preservation correlates with degree of success
- The SYNTHESIS.md intervention (adding UnexploredQuestions = named gap) improved the feature from pure pile-up to 87% gap-naming adoption
- All 4 model failure modes (premature closure, unnamed gaps, gap inflation, false gaps) map to observed orch-go failures

### Data Sources
- Artifact type audit probe (2026-03-27): 13 artifact types with measured adoption rates
- Automated adoption rate probe (2026-03-27): 7 signals with live measurements
- Accretion gate effectiveness probe (2026-03-17): 55 firings, 100% bypass rate over 2 weeks
- Named-incompleteness model.md: failure mode descriptions and thread lineage

---

## Architectural Choices

No architectural choices — task was within existing patterns.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/named-incompleteness/probes/2026-03-28-probe-feature-level-ni-preservation-classification.md` — NI-02 probe

### Constraints Discovered
- Circularity constraint: NI-02 was built from observing these features, so retrospective classification has limited evidential value. The genuine out-of-sample evidence is: (a) SYNTHESIS.md intervention trajectory, (b) gradient prediction on mixed features, (c) pre-spawn kb context, (d) two very new features.

### Key Finding: The Gradient
The most precise and hardest-to-dismiss evidence is the gradient: mixed features show partial success proportional to their NI preservation rate. This is a stronger prediction than binary success/failure because it's quantitative and harder to produce by post-hoc rationalization.

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete (probe document with verdict)
- [x] Evidence sources cited with measurements
- [x] Probe has claim/verdict frontmatter and Model Impact section
- [x] Ready for `orch complete orch-go-rwusj`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

- Can NI-02 be tested prospectively? Specifically: design a new feature to add NI to a currently-piling-up surface (e.g., add "remaining questions" to session debriefs) and measure before/after. This would be the first genuinely predictive test of NI-02.
- Does the gradient hold quantitatively? Is adoption rate of gap-naming signals a continuous predictor of composition quality, or are there thresholds (the model's Constraint 1 about specificity suggests there might be a minimum specificity threshold)?
- False gaps (Failure Mode 4) are not measured in this probe. How prevalent are placeholder tensions in briefs? The 100% adoption rate for Tension sections could include compliance noise that looks like NI but isn't.

**What remains unclear:**
- Whether the 7 in-sample success features would have been correctly predicted BEFORE the model was built, or whether the model was shaped to fit them
- Whether NI-02 generalizes to other systems (this probe is one system only)

---

## Friction

No friction — smooth session. Metrics from prior probes were well-organized and readily accessible.

---

## Session Metadata

**Skill:** research
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-work-probe-ni-02-28mar-a2ae/`
**Beads:** `bd show orch-go-rwusj`
