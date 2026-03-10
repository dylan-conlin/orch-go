# Session Synthesis

**Agent:** og-inv-phase-probe-validate-10mar-5a34
**Issue:** orch-go-y642j
**Outcome:** success

---

## Plain-Language Summary

The health score improvement from 37 to 73 is almost entirely a measurement artifact. 89% of the +36 point gain came from recalibrating the formula (widening thresholds so the same number of problems scores better), not from reducing problems. The baseline codebase — before any extractions — would already score 69 under the new formula, above the 65 gate threshold. Meanwhile, accretion velocity is still increasing (370→6,131 lines/week in cmd/orch/), new bloated files are emerging as fast as old ones shrink, and the pre-commit gate was only wired today so has zero track record. The score ≥65 gate is functionally soft harness: it can't fire for the current codebase regardless of structural health.

## Verification Contract

See: `VERIFICATION_SPEC.yaml` in this workspace.

Key outcomes:
- Score decomposition: +32.2 calibration, +3.5 extraction, +0.3 cross-term
- Accretion velocity: weekly growth in cmd/orch/ increasing, not decreasing
- Bloated file stability: 3 rapidly growing, 3 extracted, 3 stable

---

## Delta (What Changed)

### Files Created
- `.kb/models/harness-engineering/probes/2026-03-10-probe-health-score-calibration-vs-structural-improvement.md` — Probe with full decomposition evidence

### Files Modified
- `.kb/models/harness-engineering/model.md` — Merged three findings: score-calibration failure mode, extraction-adds-lines evidence, Layer 0 status update with velocity data

---

## Evidence (What Was Observed)

- Health score at baseline (26 bloated, 52 hotspots) under new formula: 69.2 — already above 65 gate
- Accretion threshold scaled from 20→92.8 (10% of 928 files), hotspot from 15→46.4
- Bloat% dimension swing: 2.7→19.6 points from formula change alone (exp decay → linear ratio)
- 5/6 extraction commits added net lines (+12 to +65 each)
- cmd/orch/ grew 48,280→48,845 (+565 lines) despite extraction rounds
- 3 files crossed 800-line threshold in 2 weeks: userconfig.go (+364), +page.svelte (+423), hotspot.go (+192)

### Tests Run
```bash
# Score decomposition verified by manual formula computation
# against observed orch health output (73/100 C)

# Snapshot history verified: 56 snapshots in ~/.orch/health-snapshots.jsonl
# Three measurement regime changes identified

# git log --numstat verified extraction net-line impact
```

---

## Architectural Choices

No architectural choices — this was an investigation/probe session.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Score calibration that adjusts thresholds to pass existing state is functionally equivalent to removing the gate
- Extraction adds lines (5/6 net-positive) without concurrent dead code removal
- Pre-commit gate wired Mar 10 — no velocity data to measure effectiveness

---

## Next (What Should Happen)

**Recommendation:** escalate

### If Escalate
**Question:** The health score ≥65 gate is non-functional — what should replace it?

**Options:**
1. **Revert to fixed thresholds** (bloat=20, hotspot=15) — gate actually fires, forces real improvement, but may produce false pressure via "gate calibration death spiral" (model §2.2)
2. **Reclassify as orientation metric** — remove gate, keep score as dashboard signal. Honest about what it is.
3. **Raise gate threshold to ≥80** — under new formula, requires hotspot count <~9 with 928 files. Forces genuine improvement but may be unreachable short-term.
4. **Use absolute bloated file count as gate** instead of composite score — simpler, harder to game via formula recalibration

**Recommendation:** Option 4. A gate on `bloated_files <= N` is deterministic (hard harness). The composite score is inherently soft because the formula is a modifiable variable. This aligns with model invariant #6: "mutable hard harness is soft harness with extra steps."

---

## Unexplored Questions

- What accretion velocity looks like 2 weeks after pre-commit gate is wired (first real data point for gate effectiveness)
- Whether the test file threshold (2000 lines) is correct — several test files are 1,500-1,800 lines and growing
- The snapshot anomalies (51-52) suggest `orch health` is sensitive to working directory — may need cwd validation

---

## Friction

- Friction: none

---

## Session Metadata

**Skill:** investigation (probe mode)
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-phase-probe-validate-10mar-5a34/`
**Probe:** `.kb/models/harness-engineering/probes/2026-03-10-probe-health-score-calibration-vs-structural-improvement.md`
**Beads:** `bd show orch-go-y642j`
