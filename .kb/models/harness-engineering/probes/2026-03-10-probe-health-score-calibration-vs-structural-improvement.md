# Probe: Health Score — Calibration vs Structural Improvement

**Model:** harness-engineering
**Date:** 2026-03-10
**Status:** Complete

---

## Question

Does the health score improvement from 37 (F) → 73 (C) reflect genuine structural improvement, or is it primarily a measurement artifact from calibration changes (threshold scaling, test file separation, bloat percentage formula)?

The model claims gates and extractions are bending the accretion curve. This probe tests whether the score corroborates that claim.

---

## What I Tested

### 1. Score Decomposition: Calibration vs Extraction

Computed the health score using both the old formula (fixed thresholds, exponential decay) and the new formula (scaled thresholds, linear ratio) at both baseline (26 bloated, 52 hotspots) and current (20 bloated, 42 hotspots) values.

```bash
# Old formula with baseline values (26 bloated, 52 hotspots, gate=1.0, ff=0.67)
gate=20.0, accretion=max(0,1-26/20)*20=0.0, fixfeat=15.5, hotspot=max(0,1-52/15)*20=0.0, bloatpct=exp(-26/10)*20=1.5
OLD BASELINE TOTAL: 37.0

# New formula with current values (20 bloated, 42 hotspots, 928 total files)
# Scaled thresholds: accretion=max(20, 928*0.10)=92.8, hotspot=max(15, 928*0.05)=46.4
gate=20.0, accretion=max(0,1-20/92.8)*20=15.7, fixfeat=15.6, hotspot=max(0,1-42/46.4)*20=1.9, bloatpct=(1-20/928)*20=19.6
NEW CURRENT TOTAL: 72.8
```

### 2. Accretion Velocity (Weekly Net Growth in cmd/orch/)

```bash
git log --since="<week_start>" --until="<week_end>" --numstat -- "cmd/orch/*.go"
```

| Week        | Net Lines | Commits |
|-------------|-----------|---------|
| Feb 10-17   | +370      | 60      |
| Feb 17-24   | +1,473    | 59      |
| Feb 24-Mar 3| +6,264    | 125     |
| Mar 3-10    | +6,131    | 99      |

Pre-commit gate wired: March 10 (today). No post-gate data exists.

Cmd/orch/ total: 48,280 (pre-extraction) → 48,845 (current) = +565 lines net despite extractions.

### 3. Extraction Net Line Impact

Checked 6 extraction commits:

| Commit | Description | Net Lines |
|--------|------------|-----------|
| 11ac150 | serve_system.go (1084→538) | **+31** |
| 9dcb647 | client.go (1115→306) | **+12** |
| d25f9cc | kb.go (1138→280) | **+43** |
| d7f3bf1 | session.go (1055→121) | **+64** |
| 8c86639 | spawn_cmd.go (1171→505) | **+65** |
| 1acc48c | daemon.go (1559→715) | **-214** |

5 of 6 extractions ADDED net lines. Only daemon extraction was net-negative.

### 4. Bloated File Stability (2-week delta)

| File | 2 wks ago | Now | Delta |
|------|-----------|-----|-------|
| +page.svelte | 778 | 1,201 | **+423** |
| hotspot.go | 858 | 1,050 | **+192** |
| userconfig.go | 611 | 975 | **+364** |
| daemon.go | 856 | 896 | +40 |
| learning.go | 975 | 979 | +4 |
| handoff.go | 898 | 898 | 0 |
| client.go | 1,449 | 1,040 | -409 |
| stats_cmd.go | 1,207 | 912 | -295 |
| context.go | 1,504 | 895 | -609 |

Three files growing rapidly (>100 lines/2 weeks). Three newly emerged bloated files.

### 5. Snapshot History Analysis

56 snapshots reveal three measurement regime changes:
- Snapshots 1-21: gate_coverage=0.00 (detection bug), bloated=57 (no test separation)
- Snapshot 22: bloated dropped 57→29 (test file separation shipped)
- Snapshot 38: gate_coverage jumped 0.00→1.00 (detection fixed), hotspot tracking added
- Snapshot 43+: TotalSourceFiles tracked (enabling threshold scaling)

---

## What I Observed

### Finding 1: 89% of score improvement is calibration, not structural

**Decomposition of the 37 → 73 (+36 point) improvement:**

| Source | Points | % of Total |
|--------|--------|------------|
| Calibration (formula change) | +32.2 | **89%** |
| Extraction (actual structural) | +3.5 | 10% |
| Cross-term | +0.3 | 1% |

The two biggest calibration effects:
1. **Threshold scaling** (accretion: 20→92.8, hotspot: 15→46.4): Turned 0/20 → 15.7/20 on accretion dimension. 42 hotspots that would score 0/20 under old formula score 1.9/20 under new.
2. **Bloat percentage formula** (exp decay → linear ratio): 20*exp(-20/10)=2.7 → 20*(1-20/928)=19.6. A **+16.9 point** swing from formula change alone.

### Finding 2: Accretion velocity has NOT decreased

Weekly net growth in cmd/orch/: 370 → 1,473 → 6,264 → 6,131 lines. Velocity is increasing, not decreasing. The pre-commit gate was only wired today (Mar 10) — zero post-gate data exists.

Total cmd/orch/ lines: 48,280 → 48,845 (+565) despite extraction rounds. Extractions are net-positive (5/6 added lines through boilerplate/header duplication).

### Finding 3: Remaining bloated files are NOT stable

Of 9 tracked source files >800 lines:
- 3 growing rapidly (>100 lines/2 weeks): +page.svelte, hotspot.go, userconfig.go
- 3 extracted (shrank): client.go, stats_cmd.go, context.go
- 3 stable: handoff.go, learning.go, daemon.go

New bloated files are emerging as fast as old ones are extracted. userconfig.go crossed 800 from 611 in 2 weeks (+364 lines). This confirms the model's thermodynamic analogy: extraction cools one spot while heat moves elsewhere.

### Finding 4: Test file separation was a counting change, not structural

Bloated count dropped 57 → 29 when test threshold was raised from 800 → 2000. Currently 41 files >800 lines exist (without test separation). The "improvement" from 57 → 20 bloated is:
- Test separation: -28 files (counting change)
- Actual extractions: -9 files (structural)

---

## Model Impact

- [x] **Contradicts** the implied claim that score ≥65 reflects genuine structural health. The score is 89% calibration artifact. At baseline values (26 bloated, 52 hotspots) the new formula would already score 69.2 — above the 65 gate threshold — with zero extractions performed.

- [x] **Confirms** invariant #4: "Extraction without routing is a pump." Three new bloated files emerged while three old ones were extracted. Net bloated count change is modest (-6), masking churn.

- [x] **Confirms** invariant #2: "Every convention without a gate will eventually be violated." Pre-commit gate was wired today; prior to that, accretion velocity was increasing weekly (370 → 6,131 lines/week).

- [x] **Extends** model with: **Score calibration as soft harness in disguise.** The health score gate (blocks feature-impl when score < 65) uses a formula whose thresholds determine what "healthy" means. The calibration commit changed what's measured, not what's measured against. A gate whose trigger is recalibrated to pass existing conditions is functionally equivalent to removing the gate. The health score gate is currently a soft harness component masquerading as hard harness — it enforces a formula that was calibrated to produce passing scores for the current codebase state.

- [x] **Extends** with: **Extraction adds lines.** 5 of 6 extraction commits were net-positive (added 12-65 lines each through file headers, package declarations, import blocks). Only the daemon extraction (-214) was net-negative because it also removed dead code. Extraction distributes entropy but doesn't reduce it without concurrent dead code removal.

---

## Verdict

**Score ≥65 does NOT reflect genuine structural health.** The 37→73 improvement is 89% calibration artifact. The same codebase at baseline values would score 69.2 under the new formula — already passing the ≥65 gate. Accretion velocity is increasing, not decreasing. New bloated files are emerging as fast as old ones are extracted.

**Recommendation:** Either (1) revert to fixed thresholds that actually pressure improvement, or (2) acknowledge the health score as an orientation metric (soft harness) rather than a gate condition (hard harness), or (3) set the gate threshold to a score only achievable through genuine structural improvement (e.g., ≥80 under new formula requires hotspot count < ~9 with current codebase size).

---

## Notes

- The TotalSourceFiles field was only added in the calibration commit. All pre-calibration snapshots have total=0, meaning historical scores are computed with the old formula and current scores with the new. Trend analysis across the regime change is meaningless.
- Snapshot 51-52 appear anomalous (bloat=5, hot=4, total=75, gate=0.83) — likely ran from a subdirectory or with different working directory.
- The health score floor gate (`pkg/spawn/gates/health.go`) blocks feature-impl at score < 65. With the calibrated formula, this gate will essentially never fire for the current codebase.
