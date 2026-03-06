# Session Synthesis

**Agent:** og-work-human-calibration-experiment-06mar-1a91
**Issue:** orch-go-54y23
**Outcome:** success

---

## Plain-Language Summary

Generated 24 model responses across 4 code review scenarios (contradiction detection, absence-as-evidence, downstream consumer contract, stale deprecation claim), each tested in 3 conditions (no skill context, knowledge-only, knowledge+stance) with 2 runs each. All responses were randomized into a blind rating sheet (R01-R24) where Dylan can rate comprehension quality 1-5 without knowing which had skill context. The purpose is to validate whether automated keyword-based indicator scores actually correlate with human judgment of comprehension quality — if they do (r > 0.6), the measurement program can proceed with confidence; if not, we need better proxies.

## Verification Contract

See `VERIFICATION_SPEC.yaml`.

Key outcomes:
- 24 transcript files captured in `evidence/2026-03-06-human-calibration/transcripts/`
- Blind rating sheet at `evidence/2026-03-06-human-calibration/blind-rating-sheet.md`
- Answer key at `evidence/2026-03-06-human-calibration/answer-key.json`
- Investigation file at `.kb/investigations/2026-03-06-inv-human-calibration-experiment.md`

---

## Delta (What Changed)

### Files Created
- `evidence/2026-03-06-human-calibration/blind-rating-sheet.md` — 24 randomized responses for blind rating
- `evidence/2026-03-06-human-calibration/answer-key.json` — Maps blind IDs to scenario/variant/auto-scores
- `evidence/2026-03-06-human-calibration/transcripts/` — 24 raw model response transcripts
- `evidence/2026-03-06-human-calibration/variants/` — 8 extracted skill context .md files
- `evidence/2026-03-06-human-calibration/prompts/` — 4 scenario prompt text files
- `evidence/2026-03-06-human-calibration/scenarios/` — Isolated scenario YAML directories
- `evidence/2026-03-06-human-calibration/build-rating-sheet.py` — Reproducible rating sheet generator
- `evidence/2026-03-06-human-calibration/run-trials.sh` — Trial runner script (unused — used skillc test directly)
- `.kb/investigations/2026-03-06-inv-human-calibration-experiment.md` — Experiment investigation

---

## Evidence (What Was Observed)

### Automated Score Summary

| Scenario | bare | knowledge-only | knowledge+stance |
|----------|------|----------------|------------------|
| S09 contradiction | [4, 1] | — | [7, 7] |
| S09 contradiction+action | — | — | [7, 7] |
| S11 absence | [3, 3] | [4, 6] | [3, 3] |
| S12 consumer | [0, 6] | [7, 7] | [3, 6] |
| S13 deprecation | [1, 1] | [4, 4] | [4, 4] |

### Notable Observations
- S09: Strong stance signal (0→2 tension detection, bare→with-stance). Action variant adds no lift.
- S11: With-stance scores EQUAL to bare. Stance doesn't help absence detection here.
- S12: Knowledge-only outperforms knowledge+stance ([7,7] vs [3,6]). Variance is high.
- S13: `notices-stale-claim` indicator fires 0/2 for ALL variants including with-stance.

### Tests Run
```bash
# 12 skillc test runs (2 runs each)
CLAUDECODE= skillc test --scenarios <dir> [--bare|--variant <file>] --runs 2 --transcripts <dir> --json
# All 24 produced valid transcripts with response text
```

---

## Architectural Choices

No architectural choices — task was experiment execution within existing patterns.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-06-inv-human-calibration-experiment.md` — Full experiment record

### Constraints Discovered
- `skillc test` requires `CLAUDECODE=` unset when run from within a Claude Code session (nested session detection)
- `skillc test --transcripts` flag exists but isn't shown in `--help` output
- `skillc test --json` output does not include raw response text — must use `--transcripts` for that

---

## Next (What Should Happen)

**Recommendation:** close (experiment data collection complete)

### Next Steps (for orchestrator)
1. Dylan rates the 24 responses in `blind-rating-sheet.md` (1-5 scale)
2. Agent computes Pearson correlation between human ratings and auto_scores from answer-key.json
3. Decision: if r > 0.6, automated proxies validated → proceed to Phase 3 scorer extensions
4. If r < 0.4, keyword-level detection is insufficient → escalate to structural analysis

---

## Unexplored Questions

- S11 with-stance scoring equal to bare: is the "absence is evidence" stance too abstract? Needs larger N or revised stance wording.
- S13 stale-claim detection at floor for all variants: is the indicator too narrow, or is temporal reasoning genuinely hard for single-turn --print mode?
- Do response length or tool-call-intent patterns correlate with human ratings independently of indicator scores?

---

## Friction

- `skillc test --help` doesn't list `--transcripts` flag, causing initial detour to build manual runner script
- `CLAUDECODE` nesting guard required workaround (`CLAUDECODE=`) for all skillc test invocations

---

## Session Metadata

**Skill:** experiment
**Model:** opus (agent), sonnet (trial model)
**Workspace:** `.orch/workspace/og-work-human-calibration-experiment-06mar-1a91/`
**Investigation:** `.kb/investigations/2026-03-06-inv-human-calibration-experiment.md`
**Beads:** `bd show orch-go-54y23`
