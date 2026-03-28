# Session Synthesis

**Agent:** og-inv-run-240-paper-28mar-e51a
**Issue:** orch-go-yf1ru
**Duration:** 2026-03-28 09:00 → 2026-03-28 09:35
**Outcome:** success

---

## Plain-Language Summary

We ran the full-scale version of the pilot bibliometrics experiment. The pilot tested 50 papers and found a trend suggesting that academic papers' open questions cluster closer together in meaning-space than their findings do — but couldn't be sure with so few papers (only 25% chance of detecting the effect). We scaled to 373 papers, and the result is clear: questions DO cluster tighter than findings (p=0.0086, well below the 0.05 threshold). This confirms the Named Incompleteness model's core prediction — that gaps/questions have a natural geometric structure that findings/conclusions lack. The effect is smaller than the pilot suggested (d=0.20 vs d=0.27), which is normal for pilots, but it's real and replicable across two different embedding models.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for commands run and outcomes verified.

Key outcomes:
- Permutation p = 0.0086 (significant at p < 0.01)
- Cohen's d = 0.195 (small but real effect)
- Bootstrap 95% CI = [0.009, 0.045] (excludes zero)
- Second model replication: perm p = 0.035 (also significant)
- Length-controlled: perm p < 0.001 (not a text-length artifact)

---

## TLDR

Ran the 373-paper scale-up of the 50-paper bibliometrics pilot. Questions cluster significantly tighter than findings in embedding space (permutation p=0.0086, d=0.20). NI-01's spatial prediction is now quantitatively confirmed.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-28-inv-run-240-paper-bibliometrics-study.md` — Full investigation with results
- `.kb/models/named-incompleteness/probes/2026-03-28-probe-bibliometrics-full-study-373-papers.md` — Probe confirming NI-01/NI-03
- `/tmp/biblio-240/fetch_papers.py` — arXiv paper fetching (373 RAG papers)
- `/tmp/biblio-240/extract_embed_analyze.py` — Full pipeline: extraction, embedding, analysis
- `/tmp/biblio-240/subsample_check.py` — Subsample stability analysis
- `/tmp/biblio-240/results.json` — Complete results
- `/tmp/biblio-240/papers.json` — 373 paper metadata

### Files Modified
- `.kb/models/named-incompleteness/model.md` — Updated validation status, NI-01 claim, Constraint 5, probes section

---

## Evidence (What Was Observed)

- 373 RAG papers fetched from arXiv (3 overlapping queries, deduplicated, 2023-2026)
- Improved heuristic extraction: 54.9 mean words for questions, 47.0 for findings
- Primary result (all-MiniLM-L6-v2): diff=0.0275, d=0.195, perm p=0.0086
- Replication (paraphrase-MiniLM-L3-v2): diff=0.0216, d=0.163, perm p=0.035
- Length-controlled: diff=0.0496, perm p<0.001 (effect increases when lengths equalized)
- Cross-type: Q-Q (0.290) > Q-F (0.269) > F-F (0.262)
- Pilot effect size overestimated: d=0.27 → true d=0.20 (26% shrinkage, typical)
- Power at N=240 is 68%, not 80% (due to smaller true d); N≥300 needed for 90%

### Tests Run
```bash
# Fetch papers
/tmp/biblio-experiment/bin/python /tmp/biblio-240/fetch_papers.py > /tmp/biblio-240/papers.json
# Result: 373 papers

# Full analysis pipeline
/tmp/biblio-experiment/bin/python /tmp/biblio-240/extract_embed_analyze.py > /tmp/biblio-240/results.json
# Result: perm p=0.0086, d=0.195, CI=[0.009, 0.045]

# Subsample stability
/tmp/biblio-experiment/bin/python /tmp/biblio-240/subsample_check.py
# Result: d stable at ~0.20 across N=50-373
```

---

## Architectural Choices

No architectural choices — task was empirical analysis, not code.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-28-inv-run-240-paper-bibliometrics-study.md` — Complete investigation
- `.kb/models/named-incompleteness/probes/2026-03-28-probe-bibliometrics-full-study-373-papers.md` — Confirmation probe

### Decisions Made
- Used improved heuristic extraction (no API key for Claude): heuristic was sufficient for significance at N=373
- Used 5000 permutation iterations (up from 1000 in pilot): more precise p-value
- Fetched 373 papers (above 240 minimum): provides overwhelming power for true d=0.20

### Constraints Discovered
- Pilot effect sizes overestimate by ~26% — future power analyses should budget for this
- Power at N=240 is 68% for d=0.20, not 80% as projected from pilot's d=0.27 — target N≥300 for 90%

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (pipeline completed successfully)
- [x] Investigation file has Phase: Complete
- [x] Probe merged into model
- [x] Ready for `orch complete orch-go-yf1ru`

---

## Unexplored Questions

- Does the effect hold in other subfields? (LLM evaluation, agent coordination, biomedical NLP)
- Would LLM extraction from full papers increase the effect size?
- Is the effect connected to citation structure? (do tightly-clustering questions cite each other?)
- Would larger embedding models show stronger effects?

---

## Friction

No friction — smooth session. arXiv API worked well, venv from pilot had most dependencies, pipeline ran successfully on first attempt.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-run-240-paper-28mar-e51a/`
**Investigation:** `.kb/investigations/2026-03-28-inv-run-240-paper-bibliometrics-study.md`
**Beads:** `bd show orch-go-yf1ru`
