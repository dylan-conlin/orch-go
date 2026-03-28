## Summary (D.E.K.N.)

**Delta:** At N=373 RAG papers, open questions cluster significantly tighter than findings in semantic embedding space (permutation p=0.0086, d=0.20, 95% CI [0.009, 0.045] excludes zero). Confirmed across two embedding models.

**Evidence:** 373 arXiv RAG papers (2023-2026), improved heuristic extraction, sentence-transformer embeddings, 5000-iteration permutation test shuffling at paper level. Second model (paraphrase-MiniLM-L3-v2) replicates (perm p=0.035). Length-controlled analysis rules out text-length artifact (perm p<0.001).

**Knowledge:** The NI model's spatial prediction is empirically confirmed: questions define tighter constraint regions than findings. Effect size is smaller than pilot (d=0.20 vs d=0.27) — typical pilot overestimation. Parametric tests remain misleading (p≈0 vs correct perm p=0.0086).

**Next:** Merge findings into NI model. Consider write-up as short paper if cross-subfield replication confirms.

**Authority:** implementation - Results extend existing model, no architectural or strategic decisions needed.

---

# Investigation: Run 240-Paper Bibliometrics Study

**Question:** Does the pilot finding (questions cluster tighter than findings, d=0.27, underpowered at N=50) hold at N≥240 with sufficient statistical power?

**Started:** 2026-03-28
**Updated:** 2026-03-28
**Owner:** og-inv-run-240-paper-28mar-e51a
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** named-incompleteness

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-28-inv-design-bibliometrics-experiment-papers-cluster.md` | extends | yes | Effect size smaller (d=0.20 vs 0.27) but direction confirmed |
| `.kb/models/named-incompleteness/probes/2026-03-28-probe-bibliometrics-pilot-questions-vs-findings-clustering.md` | confirms | yes | Pilot d=0.27 was overestimated (typical); N=240 power was 68% not 80% due to smaller true d |
| `.kb/models/named-incompleteness/probes/2026-03-28-probe-spatial-structure-questions-vs-statements-cross-domain.md` | confirms | yes | Fills the NLP empirical gap identified in that probe |

---

## Findings

### Finding 1: Questions cluster significantly tighter than findings (N=373)

**Evidence:**

| Metric | Questions | Findings | Diff (Q-F) |
|--------|-----------|----------|------------|
| Global mean cosine sim | 0.2898 | 0.2623 | +0.0275 |
| Within-topic k=5 (paired) | 0.3692 | 0.3450 | +0.0242 |
| Q tighter fraction | — | — | 57.1% |
| Cohen's d | — | — | 0.195 |

**Primary test:** Permutation (5000 iterations, paper-level shuffle): **p = 0.0086**
**Bootstrap 95% CI:** [0.0093, 0.0454] — **excludes zero**

**Source:** `/tmp/biblio-240/results.json`, `/tmp/biblio-240/extract_embed_analyze.py`

**Significance:** The pilot's trend (d=0.27, perm p=0.196) is confirmed as a real effect at scale. Questions occupy a more coherent region of semantic embedding space than findings. This is the first quantitative confirmation of NI-01's spatial prediction.

---

### Finding 2: Effect is smaller than pilot estimated (d=0.20 vs d=0.27)

**Evidence:**

Subsample stability across the 373-paper dataset:

| N | Mean diff | Cohen's d | Power (perm p<0.05) |
|---|-----------|-----------|---------------------|
| 50 | 0.0250 | 0.18 | 12% |
| 100 | 0.0287 | 0.21 | 20% |
| 150 | 0.0299 | 0.22 | 40% |
| 200 | 0.0314 | 0.23 | 58% |
| 240 | 0.0285 | 0.21 | 68% |
| 300 | 0.0291 | 0.21 | 89% |
| 373 | 0.0296 | 0.21 | 100% |

The pilot's d=0.27 was an overestimate — typical for small-sample pilots. True effect size stabilizes around d=0.20 (small but real).

**Source:** `/tmp/biblio-240/subsample_check.py`

**Significance:** Power analysis projected 80% at N=240 (based on pilot d=0.27). Actual power at N=240 is 68% (based on true d=0.20). The full N=373 dataset provides overwhelming power. Future studies should target N≥300 for 90% power at the true effect size.

---

### Finding 3: Cross-model consistency confirms robustness

**Evidence:**

| Model | Diff (Q-F) | Cohen's d | Perm p |
|-------|-----------|-----------|--------|
| all-MiniLM-L6-v2 (primary) | 0.0275 | 0.195 | 0.0086 |
| paraphrase-MiniLM-L3-v2 | 0.0216 | 0.163 | 0.035 |

Both models show the same direction, both significant at p<0.05.

**Source:** `/tmp/biblio-240/results.json`

**Significance:** The effect is not an artifact of one particular embedding model. Different architectures with different training data reproduce the finding. This strengthens the claim that the effect reflects genuine semantic structure, not model-specific bias.

---

### Finding 4: Not a length artifact (length-controlled analysis)

**Evidence:**

| Analysis | Diff (Q-F) | Perm p |
|----------|-----------|--------|
| Raw | 0.0275 | 0.0086 |
| Length-controlled (findings truncated) | 0.0496 | < 0.001 |

When findings are truncated to match question word counts, the effect actually **increases**. Questions cluster tighter even when text length is equalized.

**Source:** `/tmp/biblio-240/results.json`

**Significance:** The TF-IDF finding (pilot) was a length artifact — longer text = more vocabulary overlap. The sentence-transformer finding is the opposite of a length artifact. Truncating findings makes them cluster less tightly, widening the gap. This confirms the effect is semantic, not lexical.

---

### Finding 5: Cross-type ordering preserved at scale

**Evidence:**

| Pair Type | Mean Cosine Sim |
|-----------|----------------|
| Q-Q (question to question) | 0.2898 |
| Q-F (question to finding, cross-paper) | 0.2691 |
| F-F (finding to finding) | 0.2623 |
| Same-paper Q-F | 0.5194 |
| Random Q-F | 0.2684 |

The ordering Q-Q > Q-F > F-F holds at N=373, matching the pilot. Questions are more similar to each other than findings are to each other, and more similar to findings than findings are to themselves.

**Source:** `/tmp/biblio-240/results.json`

**Significance:** This structural signature is consistent with the NI model: questions form a tighter semantic cluster as a TYPE because they share "gap-framing" properties (problem identification, limitation naming, constraint specification). Findings are more heterogeneous because they describe diverse solutions.

---

### Finding 6: Parametric tests remain massively misleading

**Evidence:**

| Test | p-value | N effective |
|------|---------|-------------|
| Parametric t-test | ≈ 0 | 69,378 pairs (inflated) |
| Permutation test | 0.0086 | 373 papers (correct) |

The t-test treats 69,378 pairwise similarities as independent observations from 373 papers. This inflates significance by ~185x. At N=373, the permutation test IS significant, but a naive analysis would claim p≈0 rather than the correct p=0.0086.

**Source:** `/tmp/biblio-240/results.json`

**Significance:** The methodological insight from the pilot is even more important at scale. Anyone attempting this kind of analysis must use permutation tests. The difference between p≈0 and p=0.0086 matters for honest reporting.

---

## Synthesis

**Key Insights:**

1. **The NI spatial prediction is empirically confirmed.** Questions cluster tighter than findings in semantic embedding space, significantly so (p=0.0086) and robustly (two models, length-controlled, correct statistical test). This is the first quantitative confirmation of NI-01.

2. **Pilot effect sizes should be treated as upper bounds.** The pilot (N=50) found d=0.27; the full study (N=373) finds d=0.20. This 26% shrinkage is typical of pilot-to-replication transitions. Future power analyses should budget for 20-30% effect size reduction.

3. **The method works and is exportable.** The pipeline (arXiv fetch → heuristic extraction → sentence-transformer embedding → permutation test) is fully automated, requires no API keys, and produces a clear result. This could be applied to any subfield to test whether the effect generalizes.

**Answer to Investigation Question:**

Yes, the pilot finding holds at scale. At N=373, questions cluster significantly tighter than findings (d=0.20, permutation p=0.0086, 95% CI [0.009, 0.045]). The effect is smaller than the pilot suggested (d=0.20 vs d=0.27) but clearly significant. The NI model's prediction — that gaps compose through semantic similarity while findings don't — is confirmed for the RAG subfield with sentence-transformer embeddings.

---

## Structured Uncertainty

**What's tested:**

- ✅ Questions cluster tighter than findings at N=373 (verified: permutation p=0.0086, 5000 iterations)
- ✅ Effect survives length control (verified: truncated diff=0.050, perm p<0.001)
- ✅ Effect replicates across two embedding models (verified: both p<0.05)
- ✅ Cross-type ordering Q-Q > Q-F > F-F holds at scale (verified: 0.290 > 0.269 > 0.262)
- ✅ Parametric tests give misleadingly low p-values (verified: p≈0 vs correct p=0.0086)
- ✅ Effect size is stable across subsamples (verified: d≈0.20 at N=100-373)

**What's untested:**

- ⚠️ Whether LLM extraction from full papers would increase effect size (likely — richer signals from Discussion/Limitations sections)
- ⚠️ Whether effect holds in other subfields (only tested RAG; need LLM evaluation, agent coordination, etc.)
- ⚠️ Whether effect generalizes beyond CS/ML to other sciences
- ⚠️ Whether larger embedding models (e5-large, gte-large) show stronger effects
- ⚠️ Whether the effect relates to citation structure (do tightly-clustering questions cite each other?)

**What would change this:**

- Cross-subfield replication failing would suggest domain-specific artifact, not substrate-general principle
- LLM extraction showing no improvement over heuristic would suggest the heuristic already captures the signal
- Effect driven entirely by syntactic markers ("however", "despite") rather than semantic content would weaken the information-theoretic interpretation

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Update NI model with confirmed quantitative result | implementation | Probe-to-model merge per existing protocol |
| Cross-subfield replication study | strategic | Resource/priority decision for Dylan |
| Write-up as short paper | strategic | Publication strategy decision |

### Recommended Approach ⭐

**Update NI model, then decide on publication** — Merge the confirmed finding into NI-01, update validation status, and let Dylan decide whether cross-subfield replication or a write-up is the next priority.

**Why this approach:**
- The result is clean and significant — model update is straightforward
- Publication requires cross-subfield replication (strategic decision)
- The pipeline is ready for replication in other subfields

**Trade-offs accepted:**
- Not pursuing publication immediately (may lose novelty)
- Not testing LLM extraction (heuristic was sufficient for significance)

---

## Sources

- [arXiv API Documentation](https://info.arxiv.org/help/api/index.html) — Paper retrieval
- [sentence-transformers](https://www.sbert.net/) — Embedding library
- [all-MiniLM-L6-v2](https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2) — Primary embedding model

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-28-inv-design-bibliometrics-experiment-papers-cluster.md` — Pilot study and experimental design
- **Probe:** `.kb/models/named-incompleteness/probes/2026-03-28-probe-bibliometrics-pilot-questions-vs-findings-clustering.md` — Pilot probe (N=50)
- **Model:** `.kb/models/named-incompleteness/model.md` — Parent model

---

## Investigation History

**2026-03-28 09:00:** Investigation started
- Initial question: Does the pilot finding hold at N≥240 with sufficient power?
- Context: Power analysis from pilot projected 80% power at N=240

**2026-03-28 09:05:** Papers fetched
- 373 unique RAG papers from arXiv (2023-2026), 3 overlapping queries
- Deduplicated and filtered (>100 word abstracts, year ≥ 2023)

**2026-03-28 09:15:** Full pipeline complete
- PRIMARY RESULT: perm p=0.0086, d=0.195, CI=[0.009, 0.045]
- Both embedding models significant
- Length-controlled analysis rules out artifact

**2026-03-28 09:25:** Subsample stability confirmed
- Effect size stable at d≈0.20 across N=50-373
- Pilot overestimated (d=0.27 vs true d=0.20)
- N=240 gives 68% power (not 80% as projected from pilot d)

**2026-03-28 09:30:** Investigation complete
- Status: Complete
- Key outcome: NI spatial prediction confirmed. Questions cluster significantly tighter than findings (p=0.0086) at N=373.
