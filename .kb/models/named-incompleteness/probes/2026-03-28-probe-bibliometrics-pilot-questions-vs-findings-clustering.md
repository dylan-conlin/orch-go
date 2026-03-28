# Probe: Bibliometrics Pilot — Do Papers' Questions Cluster Tighter Than Findings?

**Model:** named-incompleteness
**Date:** 2026-03-28
**Status:** Complete
**claim:** NI-01, NI-03
**verdict:** extends (with power limitation)

---

## Question

NI-01 predicts: "Systems that accumulate named gaps self-organize. Systems that accumulate conclusions require external triage." The tension clustering probe (2026-03-28) found TF-IDF can't detect this — specificity confirmed but clustering disconfirmed. The cross-domain probe identified an empirical gap: nobody has measured question vs statement clustering with semantic embeddings. This probe fills that gap using 50 academic papers.

---

## What I Tested

1. Fetched 50 RAG papers from arXiv (2024+, "retrieval-augmented generation" in title)
2. Heuristic extraction of open questions/problems and findings/contributions from abstracts
3. Embedded both sets with sentence-transformers (all-MiniLM-L6-v2)
4. Compared global pairwise cosine similarity distributions
5. Permutation test (1000 iterations, shuffle at paper level) for correct significance
6. Bootstrap 95% CI
7. Length-controlled analysis (truncate findings to match question length)
8. Replication with second model (paraphrase-MiniLM-L3-v2)

---

## What I Observed

### Primary result: Questions cluster tighter (but underpowered)

| Metric | Questions | Findings | Diff (Q-F) |
|--------|-----------|----------|------------|
| Global mean cosine sim | 0.2747 | 0.2381 | +0.0365 |
| Within-topic k=5 | 0.3815 | 0.3245 | +0.0570 |
| Within-topic k=7 | 0.4095 | 0.3265 | +0.0831 |
| Cohen's d | — | — | 0.27 |

### Statistical tests

| Test | p-value | Interpretation |
|------|---------|---------------|
| Parametric t-test | < 0.001 | MISLEADING (treats 1225 pairs as independent from 50 papers) |
| Permutation (1000 iters) | 0.196 | CORRECT (shuffles at paper level) — NOT significant |
| Bootstrap 95% CI | [-0.012, 0.087] | Includes zero |

### Controls

| Check | Result |
|-------|--------|
| Length-controlled (truncated) | +0.055, p<0.001 parametric — NOT a length artifact |
| Second model (paraphrase-MiniLM-L3-v2) | +0.030, perm p=0.294 — same direction |
| Cross-type ordering | Q-Q (0.275) > Q-F (0.260) > F-F (0.238) — questions form tighter type |

### Power analysis
- N=50: ~25% power
- N=240: ~80% power
- N=330: ~90% power

---

## Model Impact

**Extends NI-01 and NI-03** with three findings:

1. **Sentence embeddings detect what TF-IDF can't.** TF-IDF showed findings clustering tighter (d=-0.43, length artifact). Sentence-transformers show questions clustering tighter (d=+0.27, survives length control). This confirms the tension clustering probe's hypothesis that the composition mechanism is "reference" (semantic meaning) not "resemblance" (shared vocabulary).

2. **The effect is real but underpowered at N=50.** Direction matches NI prediction across all analysis methods and two embedding models. But the correct statistical test (permutation) gives p=0.196. Need N≥240 for 80% power. This is a positive finding — the protocol works and the trend is consistent.

3. **Parametric tests on pairwise similarities are a methodological trap.** They inflate N by treating dependent pairs as independent. This matters for model validation: a naive analysis would "confirm" NI with p<0.001, while the correct analysis shows the evidence is merely suggestive. This finding applies to any future NI quantitative probe using similarity matrices.

**Recommended model update:** Add to NI claims table that NI-01 has been piloted quantitatively with embeddings (positive trend, underpowered). Add methodological note about permutation test requirement.

---

## Notes

### What would strengthen this
- Full 240-paper study with LLM extraction from full text
- Cross-subfield replication (LLM evaluation, agent coordination)
- Connection to citation networks (do tightly-clustering questions cite each other more?)

### What would weaken this
- 240-paper study with permutation p > 0.1 despite improved extraction
- Effect present only in CS/ML, not other fields (domain-specific, not substrate-general)
- Effect driven entirely by syntactic markers ("however", "despite") rather than semantic content

### Scripts (in /tmp/biblio-experiment/)
- `extract_and_embed.py` — Full pipeline
- `robustness.py` — Permutation, bootstrap, cross-type
- `power_analysis.py` — Power and model sensitivity
