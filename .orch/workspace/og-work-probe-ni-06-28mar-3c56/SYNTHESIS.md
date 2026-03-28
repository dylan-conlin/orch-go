# Session Synthesis

**Agent:** og-work-probe-ni-06-28mar-3c56
**Issue:** orch-go-wwxk6
**Duration:** 2026-03-28
**Outcome:** success

---

## TLDR

Probed NI-06 (optimal specificity of named incompleteness). The inverted-U shape appears consistently in every metric — medium specificity tensions outperform both low and high — but the effect is too weak for TF-IDF to confirm statistically (best p=0.117). The high-specificity decline is the clearest signal; the low-specificity side is flat. Verdict: directionally supported, not confirmed.

---

## Delta (What Changed)

### Files Created
- `.kb/models/named-incompleteness/probes/2026-03-28-probe-optimal-specificity-inverted-u.md` — Full probe document

### Files Modified
- None

### Commits
- (pending)

---

## Evidence (What Was Observed)

- 119 briefs parsed with non-trivial tensions
- Composite specificity score computed from entity density, specific questions, comparisons, lexical diversity
- Inverted-U (concave quadratic) appears in all three metrics: convergence, connection, combined product
- Peak at specificity ≈ 0.17-0.21 (D5 in decile analysis = 0.195)
- High-specificity decline: D9-D10 (spec > 0.30) consistently lowest at product ≈ 0.0031
- Permutation tests: best p=0.117 (product quadratic term), R²=0.04
- Word count confound: r=0.50 with convergence (stronger than specificity)
- Zero template-mandated tensions found in corpus — all tensions are organic

### Tests Run
```bash
# Python analysis with TF-IDF (scikit-learn), 119 briefs
# Quadratic regression, permutation tests (10,000), decile analysis
# All metrics directionally support inverted-U, none reach p<0.05
```

---

## Architectural Choices

No architectural choices — task was within existing patterns.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- TF-IDF lexical similarity has a word-count confound (r=0.50) that may dominate specificity effects
- The orch-go brief corpus has no template-mandated tensions — all are organic, varying only in scope/specificity
- The NI-06 effect shape is asymmetric: the high-specificity decline is robust, but the low-specificity ascent is flat

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Probe file written with full methodology, data, and interpretation
- [x] Verdict clearly stated: directionally supported, not confirmed
- [x] Ready for `orch complete orch-go-wwxk6`

---

## Unexplored Questions

- **Sentence-transformer replication**: Would semantic embeddings (not lexical TF-IDF) make the inverted-U significant? The NI-01 bibliometrics study saw TF-IDF miss what sentence-transformers caught.
- **Actual convergence measurement**: Can we trace which tensions generated follow-up work (subsequent briefs/probes/threads on the same concern)? This would measure real convergence rather than distributional similarity.
- **Cross-corpus template test**: Find a system with template-mandated open-question sections (compliance retroes, mandatory reflection prompts) and test whether those low-specificity gaps truly fail to converge.

---

## Friction

No friction — smooth session.

---

## Session Metadata

**Skill:** research
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-work-probe-ni-06-28mar-3c56/`
**Beads:** `bd show orch-go-wwxk6`
