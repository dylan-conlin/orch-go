# Session Synthesis

**Agent:** og-arch-flesh-out-named-27mar-58ca
**Issue:** orch-go-w9dgz
**Duration:** 2026-03-27
**Outcome:** success

---

## Plain-Language Summary

The named-incompleteness model was an empty scaffold promoted from a converged thread. I fleshed it out into a complete model with the core mechanism (an information-theoretic argument about why gaps compose and conclusions don't), six testable claims, four failure modes, five constraints, and a precise subsumption analysis showing how it relates to the three child models it claims to generalize. The key insight that required careful work: "subsumes" means NI provides the unifying WHY (gaps are specific coordinates in possibility space), while each child model adds independent HOW/WHAT/WHEN predictions that NI doesn't cover. NI doesn't replace the child models — it unifies them.

---

## TLDR

Fleshed out the named-incompleteness model from empty scaffold to complete model. Core mechanism: gaps compose because they're specific coordinates in possibility space; conclusions don't compose because they're generic. Six claims (NI-01 through NI-06), four failure modes, five constraints. Subsumption analysis confirms NI generalizes three child models at the "why" level while each retains independent predictions.

---

## Delta (What Changed)

### Files Created
- `.kb/models/generative-systems-are-organized-around/probes/2026-03-27-probe-subsumption-analysis-child-models.md` — Subsumption analysis probe testing NI-04
- `.orch/workspace/og-arch-flesh-out-named-27mar-58ca/VERIFICATION_SPEC.yaml` — Verification spec
- `.orch/workspace/og-arch-flesh-out-named-27mar-58ca/SYNTHESIS.md` — This file
- `.orch/workspace/og-arch-flesh-out-named-27mar-58ca/BRIEF.md` — Brief for comprehension queue

### Files Modified
- `.kb/models/generative-systems-are-organized-around/model.md` — Fleshed out from scaffold to complete model

### Commits
- TBD (will commit all artifacts together)

---

## Evidence (What Was Observed)

- Read all three child models in full: compositional-accretion (226 lines), attractor-gate (228 lines), knowledge-accretion (299+ lines + 310 line claims.yaml)
- Read the external validation probe (137 lines) confirming cross-domain pattern across Luhmann, Popper, Kuhn, Lakatos, PKM
- Read the promoted thread lineage (17 lines) with the raw convergence material
- Systematically checked subsumption for each child model: mapped core predictions, compared reasons, identified independent predictions

### Key Findings

1. **Subsumption is at the "why" level, not the "how" level.** NI makes the same core prediction as each child model (gaps compose, conclusions don't) for the same information-theoretic reason. But child models have 15+ independent predictions about mechanics, boundaries, and measurements.

2. **Attractor-gate subsumption is PARTIAL.** NI explains why attractors work (they're named gaps), but the timing dimension (design-time vs runtime embedding) is an independent insight from the AG model. This is the weakest subsumption of the three.

3. **The model is retrodictive, not predictive.** All evidence comes from explaining existing observations. No prediction has been generated and then tested. This is the biggest constraint.

---

## Architectural Choices

### Subsumption as "why" unification, not replacement
- **What I chose:** NI subsumes child models at the reasoning level — it provides the unifying explanation — while child models retain independent predictions
- **What I rejected:** Full replacement (child models are just NI in different domains) — this would lose domain-specific mechanics
- **Why:** Each child model has 5-10+ predictions that NI doesn't cover (CA-06 adoption mechanics, AG Claim 4 deterministic conflicts, KA-09 creation/removal asymmetry). Claiming replacement would be overclaiming.
- **Risk accepted:** The "partial subsumption" framing is less clean than full subsumption. It may feel unsatisfying — "it explains the why but not the how" could be read as "it's just a restatement at a higher level."

---

## Knowledge (What Was Learned)

### Decisions Made
- Subsumption relationship: NI provides WHY, child models provide HOW/WHAT/WHEN
- Six claims instead of the four originally suggested (added NI-05 resolution-as-side-effect and NI-06 optimal specificity)
- Four failure modes (premature closure, unnamed gaps, gap inflation, false gaps) — these are operational patterns from orch-go history

### Constraints Discovered
- The model is currently retrodictive only — no prediction has been tested
- Attractor-gate subsumption is weaker than the other two (timing dimension is independent)

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` — all automated checks pass. Manual verification needed for:
- Quality of the information-theoretic argument
- Falsifiability of the six claims
- Precision of the subsumption analysis

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (model, probe, VERIFICATION_SPEC, SYNTHESIS, BRIEF)
- [x] Probe merged into model (referenced in References section)
- [x] Model has all required sections
- [x] Ready for `orch complete orch-go-w9dgz`

---

## Unexplored Questions

- **Bibliometrics as testable prediction (from NI-03):** Citation networks organize by conclusions (co-citation) not by gaps (shared open questions). The model predicts gap-based clustering should outperform conclusion-based clustering. This is the most concrete novel prediction.
- **Gap inflation threshold (from NI-06):** At what point do too many named gaps become noise? Is there a measurable threshold where convergence breaks down?
- **Predictive test design:** What would a clean experiment look like for testing NI-01 (named gaps compose) against a null hypothesis?

---

## Friction

No friction — smooth session. Model reading and synthesis were straightforward.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-arch-flesh-out-named-27mar-58ca/`
**Beads:** `bd show orch-go-w9dgz`
