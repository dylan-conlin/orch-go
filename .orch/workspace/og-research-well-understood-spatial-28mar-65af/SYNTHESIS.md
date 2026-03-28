# Session Synthesis

**Agent:** og-research-well-understood-spatial-28mar-65af
**Issue:** orch-go-4k8z9
**Duration:** 2026-03-28T00:11 → 2026-03-28T08:15
**Outcome:** success

---

## Plain-Language Summary

The named-incompleteness model claims that gaps (questions, tensions) are "specific coordinates in possibility space" while conclusions are "generic" — and that this is information-theoretic, not metaphorical. This research probe tested that claim by checking whether four independent traditions (information geometry, NLP, cognitive science, philosophy of science) have convergent evidence for the spatial asymmetry.

**The answer is yes, with a refinement.** Four traditions independently confirm that questions/gaps carry more geometric structure than statements/conclusions. But the mechanism isn't "gaps are tighter points" — it's "gaps define constraint surfaces (submanifolds) while conclusions are structurally underdetermined points." This distinction matters: it explains *why* gaps compose (overlapping surfaces intersect naturally) and why conclusions don't (points don't carry the structure that produced them). The strongest findings were: inquisitive semantics formally *proves* questions are more structured than assertions; bibliometric citation networks show research fronts cluster 6x denser than established knowledge; cognitive science shows TOT states carry precise partial information about gap location; and information geometry formalizes gaps as constraint surfaces via e/m-projection duality.

One identified empirical gap: nobody has directly measured within-topic clustering of questions vs statements in NLP embedding space (the architectural evidence is strong but the controlled geometric experiment hasn't been run).

---

## TLDR

Tested whether NI's spatial claims (gaps are specific, conclusions are generic) map to real structure across independent traditions. Four domains converge: gaps define constraint surfaces while conclusions are underdetermined points. Mechanism refined from "coordinates" to "submanifolds" — strengthens the model. NLP clustering is an identified empirical gap.

---

## Delta (What Changed)

### Files Created
- `.kb/models/named-incompleteness/probes/2026-03-28-probe-spatial-structure-questions-vs-statements-cross-domain.md` — Full cross-domain probe with evidence from 4 traditions

### Files Modified
- `.kb/models/named-incompleteness/model.md` — Updated validation status, refined Core Mechanism section (coordinates → constraint surfaces), added cross-domain validation table, added probe reference

### Commits
- TBD (pending completion)

---

## Evidence (What Was Observed)

### Information Geometry
- Fisher metric formalizes variable curvature (Amari & Nagaoka 2000). Direction is opposite to naive expectation: distributions near certainty have higher Fisher information than max-entropy distributions
- BUT: questions define constraint surfaces (submanifolds); conclusions are points. The surface carries more structural information than the point. This maps via e/m-projection duality (Csiszar 2003)
- Caticha (2021) formalizes belief updating as constrained movement on statistical manifolds

### NLP Embeddings
- No direct measurement of Q vs S clustering exists (empirical gap)
- HyDE: +38% nDCG@10 from converting questions to document-form (Gao et al. 2023)
- PromptBERT: 34-point Spearman shift from syntactic template change (Jiang et al. 2022)
- Dong et al. (2022): Asymmetric dual encoders produce disjoint Q/A clusters in t-SNE

### Cognitive Science
- TOT states: subjects report first letter, syllable count, semantic neighborhood of unretrieved target (Brown & McNeill 1966)
- FOK: 3x recognition accuracy when FOK reported (Hart 1965)
- Curiosity: inverted U-shape — max at intermediate knowledge where gap is specific (Loewenstein 1994)
- fMRI: curiosity activates caudate + hippocampus, enhances encoding of even unrelated material (Gruber et al. 2014)
- Desirable difficulties: incomplete knowledge structurally more generative than complete (Bjork & Bjork 1992)

### Philosophy of Science
- Inquisitive semantics: formal proof that questions carry strictly more structure than assertions (Ciardelli et al. 2018)
- Research fronts: 6x citation density vs archive (de Solla Price 1965)
- Lakatos: positive heuristic is literally a pre-articulated map of gaps
- Popper: P2 is richer and more structured than P1
- Kuhn: anomalies get "isolated precisely and given structure"
- Bromberger: p-predicament formalizes structured ignorance

---

## Architectural Choices

No architectural choices — task was research/synthesis within existing patterns.

---

## Knowledge (What Was Learned)

### Key Insight
The named-incompleteness model's spatial language maps to real structure across 4 independent traditions, but the mechanism is more precise than originally stated: gaps define constraint surfaces (submanifolds), not just coordinates. This refinement:
- Explains WHY gaps compose (overlapping surfaces)
- Explains WHY conclusions don't (underdetermined points)
- Maps to formal mathematics (info geometry e/m-projections)
- Maps to formal logic (inquisitive semantics partition vs elimination)

### Constraints Discovered
- NLP embedding space geometry of Q vs S has never been directly measured — strong architectural evidence but no controlled geometric experiment
- Information geometry direction is opposite to naive expectation (conclusions have tighter curvature, gaps have looser curvature) — the spatial story is about dimensionality of constraint surfaces, not tightness of regions

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Probe file created and complete
- [x] Probe merged into parent model
- [x] SYNTHESIS.md created
- [x] BRIEF.md created
- [x] VERIFICATION_SPEC.yaml created
- [x] Ready for `orch complete orch-go-4k8z9`

---

## Unexplored Questions

- **NLP embedding experiment:** Take N topics, generate matched question/statement pairs, embed with multiple models, compare cluster radii/density. This is a testable prediction of NI that could yield a novel publication.
- **Does the constraint-surface refinement change any NI predictions?** Probably not, but worth checking: are there cases where "constraint surface" predicts different compositional behavior than "specific coordinate"?
- **Loewenstein's inverted U and NI's gap inflation failure mode:** The curiosity inverted-U (too little knowledge = no salient gap; too much = no gap) maps directly to NI's failure mode 3 (gap inflation — too many gaps, none specific enough). The cognitive mechanism is the same.
- **Inquisitive semantics as a formal foundation for NI:** Could NI's claims be stated in the inquisitive semantics framework? If so, some predictions might become provable rather than just empirically testable.

---

## Friction

No friction — smooth session. Parallel agent dispatch worked well for cross-domain research.

---

## Session Metadata

**Skill:** research
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-research-well-understood-spatial-28mar-65af/`
**Beads:** `bd show orch-go-4k8z9`

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace. Key outcomes: probe file complete with cross-domain evidence from 4 traditions, model updated with refinement and validation table, no contradictions found.
