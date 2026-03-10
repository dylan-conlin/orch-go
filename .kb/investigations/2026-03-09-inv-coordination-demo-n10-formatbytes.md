## Summary (D.E.K.N.)

**Delta:** [Pending — running N=10 experiment]

**Evidence:** [Pending]

**Knowledge:** [Pending]

**Next:** [Pending]

**Authority:** implementation — Extends prior N=1 pilot with statistical data, no architectural changes

---

# Investigation: Coordination Demo N=10 FormatBytes — Statistical Significance

**Question:** Does the N=1 pilot finding (100% coordination failure rate independent of model capability) hold at N=10 for both Haiku and Opus?

**Started:** 2026-03-09
**Updated:** 2026-03-09
**Owner:** investigation agent (orch-go-ei2q2)
**Phase:** Investigating
**Next Step:** Run N=10 trials and analyze results
**Status:** In Progress

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-09-inv-coordination-failure-controlled-demo-same.md | extends | yes — N=1 pilot data confirmed | - |

**Key claims from prior work:**
- Both Haiku and Opus score 6/6 individually on FormatBytes (N=1)
- 100% merge conflict rate (N=1, 1/1 trials)
- Haiku 22% faster than Opus (49s vs 63s, N=1)
- Coordination failure is structural (git merge at same insertion points), not capability-based

**This investigation:** Validates these claims at N=10 for statistical significance.

---

## Experiment Design

### Methodology
- **Baseline:** Both agents start from same git commit each trial
- **Isolation:** Independent git worktrees per model per trial
- **Task:** Identical — FormatBytes(bytes int64) string with tests
- **Models:** claude-haiku-4-5-20251001 vs claude-opus-4-5-20251101
- **N:** 10 trials per model (20 total agent runs)
- **Scoring:** Same 6-dimension rubric as pilot (F0-F5)
- **Merge test:** Each trial pair tested for git merge conflicts
- **Parallelism:** Haiku and Opus run in parallel within each trial

### Statistical Goals
- With N=10 per model, a Fisher's exact test can detect differences with:
  - 10/10 vs 10/10 success: p=1.0 (no difference)
  - 10/10 vs 8/10 success: p=0.47 (not significant)
  - 10/10 vs 6/10 success: p=0.04 (significant at α=0.05)
- Main hypothesis: coordination failure rate is independent of model capability

---

## Findings

[Pending — experiment in progress]

---

## References

**Commands Run:**
```bash
# Run N=10 experiment (pending)
```
